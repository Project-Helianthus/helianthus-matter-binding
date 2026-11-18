// Package publication decides which live resources the Matter bridge may
// publish and keeps the endpoint number of each published resource.
//
// A live resource is published only when its catalog element has the
// disposition "transformed" on the derived Matter roster and that element is
// not named in the single exclusion list, exclusions.json. Every other
// disposition, an unmapped element and a malformed resource id are withheld
// with a named reason. Several published elements of one resource are one
// endpoint. A relation allocates no endpoint.
//
// The store keys an endpoint number to a resource id, assigns the lowest free
// number from one upward, never moves or frees a number and holds at most
// StoreCapacity ids. A new number is saved as a complete document before it is
// returned. The package creates no Matter node, endpoint, fabric or
// commissioning state and is not an admission decision.
package publication

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Dispositions of a derived roster row.
const (
	Transformed = "transformed"
	Unknown     = "unknown"
	Unsupported = "unsupported"
	Withheld    = "withheld"
)

// KindRelation is the roster kind of a relation row.
const KindRelation = "relation"

// Named reasons for a withheld element or a refused id.
const (
	ReasonResourceID    = "resource id is empty or malformed"
	ReasonUnmapped      = "element has no row on the derived roster"
	ReasonExcluded      = "element is on the exclusion list"
	ReasonRelation      = "a relation allocates no endpoint"
	ReasonNoDisposition = "element has no disposition"
	ReasonCapacity      = "endpoint store holds its full capacity of resource ids"
	ReasonUnsaved       = "endpoint store could not be saved"
)

// StoreCapacity is the number of resource ids the store holds for the life of
// a node. A further new id is refused.
const StoreCapacity = 64

const (
	firstEndpoint = 1
	lastEndpoint  = 0xFFFE
)

//go:embed exclusions.json
var exclusionsDocument []byte

var errMalformed = errors.New("malformed document")

// Row is one row of the derived Matter roster.
type Row struct {
	Element     string
	Kind        string
	Disposition string
}

// Live is one catalog element observed on one live resource.
type Live struct {
	ResourceID string
	Element    string
}

// Resource is one published resource and the elements it publishes.
type Resource struct {
	ID       string
	Elements []string
}

// Withholding names one live element that is not published.
type Withholding struct {
	ResourceID string
	Element    string
	Reason     string
}

// Decision lists published resources by id and withheld elements by resource
// id and element.
type Decision struct {
	Published []Resource
	Withheld  []Withholding
}

// ResourceIDs returns the published resource ids in order.
func (d Decision) ResourceIDs() []string {
	ids := make([]string, 0, len(d.Published))
	for _, resource := range d.Published {
		ids = append(ids, resource.ID)
	}
	return ids
}

// Rule is the publication rule over one derived roster and the committed
// exclusion list.
type Rule struct {
	rows     map[string]Row
	excluded map[string]struct{}
}

// NewRule builds the rule from roster rows and the embedded exclusion list.
func NewRule(rows []Row) (*Rule, error) {
	return newRule(rows, exclusionsDocument)
}

// Exclusions returns the committed exclusion list.
func Exclusions() ([]string, error) {
	return parseExclusions(exclusionsDocument)
}

func newRule(rows []Row, exclusions []byte) (*Rule, error) {
	elements, err := parseExclusions(exclusions)
	if err != nil {
		return nil, err
	}
	rule := &Rule{rows: make(map[string]Row, len(rows)), excluded: make(map[string]struct{}, len(elements))}
	for _, row := range rows {
		if !token(row.Element) {
			return nil, fmt.Errorf("roster row has an empty or malformed element %q", row.Element)
		}
		if _, ok := rule.rows[row.Element]; ok {
			return nil, fmt.Errorf("roster has more than one row for %q", row.Element)
		}
		rule.rows[row.Element] = row
	}
	for _, element := range elements {
		rule.excluded[element] = struct{}{}
	}
	return rule, nil
}

// Decide applies the rule to live elements. The input order does not matter.
func (r *Rule) Decide(live []Live) Decision {
	published := map[string]map[string]struct{}{}
	withheld := map[Withholding]struct{}{}
	for _, item := range live {
		if reason := r.reason(item); reason != "" {
			withheld[Withholding{ResourceID: item.ResourceID, Element: item.Element, Reason: reason}] = struct{}{}
			continue
		}
		if published[item.ResourceID] == nil {
			published[item.ResourceID] = map[string]struct{}{}
		}
		published[item.ResourceID][item.Element] = struct{}{}
	}
	var decision Decision
	for id, elements := range published {
		decision.Published = append(decision.Published, Resource{ID: id, Elements: sortedKeys(elements)})
	}
	sort.Slice(decision.Published, func(i, j int) bool { return decision.Published[i].ID < decision.Published[j].ID })
	for item := range withheld {
		decision.Withheld = append(decision.Withheld, item)
	}
	sort.Slice(decision.Withheld, func(i, j int) bool {
		a, b := decision.Withheld[i], decision.Withheld[j]
		if a.ResourceID != b.ResourceID {
			return a.ResourceID < b.ResourceID
		}
		return a.Element < b.Element
	})
	return decision
}

func (r *Rule) reason(item Live) string {
	if !token(item.ResourceID) {
		return ReasonResourceID
	}
	row, ok := r.rows[item.Element]
	if !ok {
		return ReasonUnmapped
	}
	if _, ok := r.excluded[item.Element]; ok {
		return ReasonExcluded
	}
	switch row.Disposition {
	case Transformed:
		if row.Kind == KindRelation {
			return ReasonRelation
		}
		return ""
	case "":
		return ReasonNoDisposition
	case Unknown, Unsupported, Withheld:
		return "element disposition is " + row.Disposition
	default:
		return "element disposition is not recognised"
	}
}

// SaveFunc durably writes one complete store document. It returns only after
// the document is written.
type SaveFunc func(document []byte) error

// Assignment holds the endpoint numbers the caller may expose and the ids
// refused with a named reason.
type Assignment struct {
	Endpoints map[string]uint16
	Refused   map[string]string
}

// Store keeps one endpoint number per resource id.
type Store struct {
	byID     map[string]uint16
	byNumber map[uint16]string
}

// NewStore returns an empty store for a node that has stored no document.
func NewStore() *Store {
	return &Store{byID: map[string]uint16{}, byNumber: map[uint16]string{}}
}

type storedRecord struct {
	ResourceID *string `json:"resource_id"`
	Endpoint   *int64  `json:"endpoint"`
}

type storedDocument struct {
	Endpoints *[]*storedRecord `json:"endpoints"`
}

// Load restores a saved store document. A malformed document is refused and
// no store is returned.
func Load(document []byte) (*Store, error) {
	var stored storedDocument
	if err := decodeStrict(document, &stored); err != nil {
		return nil, fmt.Errorf("endpoint store: %w", err)
	}
	if stored.Endpoints == nil || len(*stored.Endpoints) > StoreCapacity {
		return nil, fmt.Errorf("endpoint store: %w: missing endpoints or more than its capacity", errMalformed)
	}
	store := NewStore()
	for _, record := range *stored.Endpoints {
		if record == nil || record.ResourceID == nil || record.Endpoint == nil {
			return nil, fmt.Errorf("endpoint store: %w: incomplete record", errMalformed)
		}
		id, number := *record.ResourceID, *record.Endpoint
		if !token(id) || number < firstEndpoint || number > lastEndpoint {
			return nil, fmt.Errorf("endpoint store: %w: invalid record", errMalformed)
		}
		if _, ok := store.byID[id]; ok {
			return nil, fmt.Errorf("endpoint store: %w: duplicate resource id", errMalformed)
		}
		if _, ok := store.byNumber[uint16(number)]; ok {
			return nil, fmt.Errorf("endpoint store: %w: duplicate endpoint number", errMalformed)
		}
		store.byID[id] = uint16(number)
		store.byNumber[uint16(number)] = id
	}
	return store, nil
}

// Len returns the number of stored resource ids.
func (s *Store) Len() int { return len(s.byID) }

// Endpoint returns the stored number of a resource id.
func (s *Store) Endpoint(id string) (uint16, bool) {
	number, ok := s.byID[id]
	return number, ok
}

// Document returns the complete store document.
func (s *Store) Document() []byte {
	return encode(s.byID)
}

// Assign returns the stored number of every live id and assigns the lowest
// free number to each new id, in id order, up to StoreCapacity. New numbers
// are saved before they are returned; when the save fails no new number is
// kept or returned.
func (s *Store) Assign(ids []string, save SaveFunc) Assignment {
	result := Assignment{Endpoints: map[string]uint16{}, Refused: map[string]string{}}
	var fresh []string
	for _, id := range uniqueSorted(ids) {
		if number, ok := s.byID[id]; ok {
			result.Endpoints[id] = number
		} else if !token(id) {
			result.Refused[id] = ReasonResourceID
		} else if len(s.byID)+len(fresh) >= StoreCapacity {
			result.Refused[id] = ReasonCapacity
		} else {
			fresh = append(fresh, id)
		}
	}
	if len(fresh) == 0 {
		return result
	}
	next := make(map[string]uint16, len(s.byID)+len(fresh))
	used := make(map[uint16]struct{}, len(s.byID)+len(fresh))
	for id, number := range s.byID {
		next[id] = number
		used[number] = struct{}{}
	}
	candidate := uint16(firstEndpoint)
	for _, id := range fresh {
		for ; candidate <= lastEndpoint; candidate++ {
			if _, taken := used[candidate]; !taken {
				break
			}
		}
		next[id] = candidate
		used[candidate] = struct{}{}
	}
	if save == nil || save(encode(next)) != nil {
		for _, id := range fresh {
			result.Refused[id] = ReasonUnsaved
		}
		return result
	}
	for _, id := range fresh {
		s.byID[id] = next[id]
		s.byNumber[next[id]] = id
		result.Endpoints[id] = next[id]
	}
	return result
}

type outputRecord struct {
	ResourceID string `json:"resource_id"`
	Endpoint   uint16 `json:"endpoint"`
}

func encode(byID map[string]uint16) []byte {
	records := make([]outputRecord, 0, len(byID))
	for id, number := range byID {
		records = append(records, outputRecord{ResourceID: id, Endpoint: number})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Endpoint < records[j].Endpoint })
	data, err := json.Marshal(struct {
		Endpoints []outputRecord `json:"endpoints"`
	}{records})
	if err != nil {
		panic(err)
	}
	return data
}

type exclusionFile struct {
	Elements *[]string `json:"elements"`
}

func parseExclusions(document []byte) ([]string, error) {
	var parsed exclusionFile
	if err := decodeStrict(document, &parsed); err != nil {
		return nil, fmt.Errorf("exclusion list: %w", err)
	}
	if parsed.Elements == nil {
		return nil, fmt.Errorf("exclusion list: %w: missing elements", errMalformed)
	}
	seen := map[string]struct{}{}
	for _, element := range *parsed.Elements {
		if !token(element) {
			return nil, fmt.Errorf("exclusion list: %w: empty or malformed element", errMalformed)
		}
		if _, ok := seen[element]; ok {
			return nil, fmt.Errorf("exclusion list: %w: duplicate element", errMalformed)
		}
		seen[element] = struct{}{}
	}
	return append([]string{}, *parsed.Elements...), nil
}

// decodeStrict accepts exactly one JSON object with known fields, no
// duplicate keys at any depth and no trailing data.
func decodeStrict(document []byte, target any) error {
	if err := uniqueKeys(document); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %v", errMalformed, err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("%w: trailing data", errMalformed)
	}
	return nil
}

func uniqueKeys(document []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("%w: %v", errMalformed, err)
	}
	if first != json.Delim('{') {
		return fmt.Errorf("%w: not an object", errMalformed)
	}
	return walkContainer(decoder, '{')
}

func walkContainer(decoder *json.Decoder, open json.Delim) error {
	seen := map[string]struct{}{}
	for decoder.More() {
		if open == '{' {
			key, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("%w: %v", errMalformed, err)
			}
			name, _ := key.(string)
			if _, ok := seen[name]; ok {
				return fmt.Errorf("%w: duplicate key %q", errMalformed, name)
			}
			seen[name] = struct{}{}
		}
		value, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("%w: %v", errMalformed, err)
		}
		if delim, ok := value.(json.Delim); ok {
			if err := walkContainer(decoder, delim); err != nil {
				return err
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("%w: %v", errMalformed, err)
	}
	return nil
}

func token(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n\t")
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	return sortedKeys(seen)
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
