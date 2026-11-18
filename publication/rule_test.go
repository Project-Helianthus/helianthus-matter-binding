package publication

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
)

// Synthetic catalog elements. The names follow device families and are not
// rows of any real roster.
const (
	current     = "synthetic.heat-pump-family.electrical-current"
	power       = "synthetic.heat-pump-family.electrical-power"
	unknown     = "synthetic.heat-generator-family.operating-status"
	excluded    = "synthetic.ventilation-family.fan-speed"
	unsupported = "synthetic.heat-generator-family.service"
	withheld    = "synthetic.heat-generator-family.contains"
	blank       = "synthetic.heat-generator-family.no-disposition"
	relation    = "synthetic.heat-pump-family.hosts"
	unmapped    = "synthetic.heat-pump-family.unmapped"

	resA = "synthetic-resource-a"
	resB = "synthetic-resource-b"
	resC = "synthetic-resource-c"
	resD = "synthetic-resource-d"
)

func syntheticRows() []Row {
	return []Row{
		{Element: current, Kind: "fact", Disposition: Transformed},
		{Element: power, Kind: "fact", Disposition: Transformed},
		{Element: unknown, Kind: "fact", Disposition: Unknown},
		{Element: excluded, Kind: "capability", Disposition: Transformed},
		{Element: unsupported, Kind: "service", Disposition: Unsupported},
		{Element: withheld, Kind: "relation", Disposition: Withheld},
		{Element: blank, Kind: "fact", Disposition: ""},
		{Element: relation, Kind: KindRelation, Disposition: Transformed},
	}
}

func exclusionDocument(elements ...string) []byte {
	if elements == nil {
		elements = []string{}
	}
	data, err := json.Marshal(map[string][]string{"elements": elements})
	if err != nil {
		panic(err)
	}
	return data
}

func mustRule(t *testing.T, rows []Row, exclusions []byte) *Rule {
	t.Helper()
	rule, err := newRule(rows, exclusions)
	if err != nil {
		t.Fatalf("newRule: %v", err)
	}
	return rule
}

type recorder struct {
	saved [][]byte
	fail  error
}

func (r *recorder) save(document []byte) error {
	if r.fail != nil {
		return r.fail
	}
	r.saved = append(r.saved, append([]byte(nil), document...))
	return nil
}

func (r *recorder) last(t *testing.T) []byte {
	t.Helper()
	if len(r.saved) == 0 {
		t.Fatal("no document was saved")
	}
	return r.saved[len(r.saved)-1]
}

func refuseSave(t *testing.T) SaveFunc {
	return func([]byte) error {
		t.Helper()
		t.Fatal("a store that assigns nothing new must not save")
		return nil
	}
}

func syntheticID(index int) string { return fmt.Sprintf("synthetic-resource-%03d", index) }

// The rule publishes two resources that share one transformed element and
// withholds an unknown element, an excluded element and a resource with an
// empty id. The published resources survive a save and a second load.
func TestPublishedResourcesKeepTheirNumbersAcrossALoad(t *testing.T) {
	rule := mustRule(t, syntheticRows(), exclusionDocument(excluded))
	decision := rule.Decide([]Live{
		{ResourceID: resA, Element: current},
		{ResourceID: resB, Element: current},
		{ResourceID: resC, Element: unknown},
		{ResourceID: resD, Element: excluded},
		{ResourceID: "", Element: current},
	})
	wantPublished := []Resource{
		{ID: resA, Elements: []string{current}},
		{ID: resB, Elements: []string{current}},
	}
	if !reflect.DeepEqual(decision.Published, wantPublished) {
		t.Fatalf("published = %#v, want %#v", decision.Published, wantPublished)
	}
	wantWithheld := []Withholding{
		{ResourceID: "", Element: current, Reason: ReasonResourceID},
		{ResourceID: resC, Element: unknown, Reason: "element disposition is unknown"},
		{ResourceID: resD, Element: excluded, Reason: ReasonExcluded},
	}
	if !reflect.DeepEqual(decision.Withheld, wantWithheld) {
		t.Fatalf("withheld = %#v, want %#v", decision.Withheld, wantWithheld)
	}

	var disk recorder
	store := NewStore()
	first := store.Assign(decision.ResourceIDs(), disk.save)
	if want := map[string]uint16{resA: 1, resB: 2}; !reflect.DeepEqual(first.Endpoints, want) || len(first.Refused) != 0 {
		t.Fatalf("first assignment = %#v", first)
	}

	reloaded, err := Load(disk.last(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	second := reloaded.Assign(decision.ResourceIDs(), refuseSave(t))
	if !reflect.DeepEqual(second.Endpoints, first.Endpoints) || len(second.Refused) != 0 {
		t.Fatalf("second assignment = %#v, want %#v", second, first)
	}
}

// Sixty-four numbers are stored; the sixty-fifth live id is refused and the
// stored ids stay unchanged, also after a second load.
func TestStoreRefusesAnIDBeyondItsCapacity(t *testing.T) {
	var disk recorder
	store := NewStore()
	ids := make([]string, 0, StoreCapacity)
	for index := 1; index <= StoreCapacity; index++ {
		ids = append(ids, syntheticID(index))
	}
	full := store.Assign(ids, disk.save)
	if len(full.Endpoints) != StoreCapacity || len(full.Refused) != 0 {
		t.Fatalf("filling the store = %d endpoints, refused %#v", len(full.Endpoints), full.Refused)
	}
	for index, id := range ids {
		if full.Endpoints[id] != uint16(index+1) {
			t.Fatalf("%s = %d, want %d", id, full.Endpoints[id], index+1)
		}
	}
	saved := disk.last(t)

	reloaded, err := Load(saved)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	extra := syntheticID(StoreCapacity + 1)
	result := reloaded.Assign(append([]string{extra}, ids...), refuseSave(t))
	if want := map[string]string{extra: ReasonCapacity}; !reflect.DeepEqual(result.Refused, want) {
		t.Fatalf("refused = %#v, want %#v", result.Refused, want)
	}
	if !reflect.DeepEqual(result.Endpoints, full.Endpoints) {
		t.Fatal("stored ids changed after a refused assignment")
	}
	if !bytes.Equal(reloaded.Document(), saved) {
		t.Fatal("stored document changed after a refused assignment")
	}
	if _, ok := reloaded.Endpoint(extra); ok {
		t.Fatal("refused id holds a number")
	}
	if reloaded.Len() != StoreCapacity {
		t.Fatalf("stored ids = %d, want %d", reloaded.Len(), StoreCapacity)
	}
}

// Several transformed facts of one resource are one endpoint, and an
// unmapped, unsupported, withheld or undisposed element or a relation is
// withheld with its named reason while the resource keeps its endpoint.
func TestOneResourceIsOneEndpointAndEveryOtherDispositionIsWithheld(t *testing.T) {
	rule := mustRule(t, syntheticRows(), exclusionDocument())
	decision := rule.Decide([]Live{
		{ResourceID: resA, Element: power},
		{ResourceID: resA, Element: current},
		{ResourceID: resA, Element: current},
		{ResourceID: resA, Element: unsupported},
		{ResourceID: resB, Element: withheld},
		{ResourceID: resB, Element: blank},
		{ResourceID: resB, Element: relation},
		{ResourceID: resC, Element: unmapped},
		{ResourceID: resC, Element: ""},
		{ResourceID: " " + resD, Element: current},
	})
	wantPublished := []Resource{{ID: resA, Elements: []string{current, power}}}
	if !reflect.DeepEqual(decision.Published, wantPublished) {
		t.Fatalf("published = %#v, want %#v", decision.Published, wantPublished)
	}
	wantWithheld := []Withholding{
		{ResourceID: " " + resD, Element: current, Reason: ReasonResourceID},
		{ResourceID: resA, Element: unsupported, Reason: "element disposition is unsupported"},
		{ResourceID: resB, Element: withheld, Reason: "element disposition is withheld"},
		{ResourceID: resB, Element: blank, Reason: ReasonNoDisposition},
		{ResourceID: resB, Element: relation, Reason: ReasonRelation},
		{ResourceID: resC, Element: "", Reason: ReasonUnmapped},
		{ResourceID: resC, Element: unmapped, Reason: ReasonUnmapped},
	}
	if !reflect.DeepEqual(decision.Withheld, wantWithheld) {
		t.Fatalf("withheld = %#v, want %#v", decision.Withheld, wantWithheld)
	}
	if got := decision.ResourceIDs(); !reflect.DeepEqual(got, []string{resA}) {
		t.Fatalf("resource ids = %#v", got)
	}
}

// Each field of a roster row and of a live resource is varied on its own.
// Element: a rule keyed by anything other than the element would publish the
// renamed row. Kind: a rule that ignores the kind would publish the relation.
// Disposition: a rule that treats any row as mapped would publish it.
// ResourceID: a rule that keys the endpoint by element would merge resources.
func TestEveryRowAndLiveFieldDecides(t *testing.T) {
	base := Row{Element: current, Kind: "fact", Disposition: Transformed}
	rows := map[string]Row{
		"element":      {Element: current + ".other", Kind: base.Kind, Disposition: base.Disposition},
		"kind":         {Element: base.Element, Kind: KindRelation, Disposition: base.Disposition},
		"disposition":  {Element: base.Element, Kind: base.Kind, Disposition: Unknown},
		"unrecognised": {Element: base.Element, Kind: base.Kind, Disposition: "proposed"},
	}
	reasons := map[string]string{
		"element":      ReasonUnmapped,
		"kind":         ReasonRelation,
		"disposition":  "element disposition is unknown",
		"unrecognised": "element disposition is not recognised",
	}
	for field, row := range rows {
		t.Run(field, func(t *testing.T) {
			rule := mustRule(t, []Row{row}, exclusionDocument())
			decision := rule.Decide([]Live{{ResourceID: resA, Element: current}})
			if len(decision.Published) != 0 {
				t.Fatalf("published %#v", decision.Published)
			}
			want := []Withholding{{ResourceID: resA, Element: current, Reason: reasons[field]}}
			if !reflect.DeepEqual(decision.Withheld, want) {
				t.Fatalf("withheld = %#v, want %#v", decision.Withheld, want)
			}
		})
	}
	t.Run("resource id", func(t *testing.T) {
		rule := mustRule(t, []Row{base}, exclusionDocument())
		decision := rule.Decide([]Live{{ResourceID: resA, Element: current}, {ResourceID: resB, Element: current}})
		if got := decision.ResourceIDs(); !reflect.DeepEqual(got, []string{resA, resB}) {
			t.Fatalf("resource ids = %#v", got)
		}
	})
	t.Run("live element", func(t *testing.T) {
		rule := mustRule(t, []Row{base}, exclusionDocument())
		decision := rule.Decide([]Live{{ResourceID: resA, Element: power}})
		want := []Withholding{{ResourceID: resA, Element: power, Reason: ReasonUnmapped}}
		if len(decision.Published) != 0 || !reflect.DeepEqual(decision.Withheld, want) {
			t.Fatalf("decision = %#v", decision)
		}
	})
}

func TestRuleRefusesAnAmbiguousRoster(t *testing.T) {
	cases := map[string][]Row{
		"duplicate element": {{Element: current, Kind: "fact", Disposition: Transformed}, {Element: current, Kind: "fact", Disposition: Unknown}},
		"empty element":     {{Element: "", Kind: "fact", Disposition: Transformed}},
		"padded element":    {{Element: " " + current, Kind: "fact", Disposition: Transformed}},
	}
	for name, rows := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := newRule(rows, exclusionDocument()); err == nil {
				t.Fatal("ambiguous roster was accepted")
			}
		})
	}
}

func TestCommittedExclusionListIsEmptyAndEmbedded(t *testing.T) {
	onDisk, err := os.ReadFile("exclusions.json")
	if err != nil {
		t.Fatalf("read exclusions.json: %v", err)
	}
	if !bytes.Equal(onDisk, exclusionsDocument) {
		t.Fatal("embedded exclusion list differs from the committed file")
	}
	elements, err := Exclusions()
	if err != nil {
		t.Fatalf("Exclusions: %v", err)
	}
	if len(elements) != 0 {
		t.Fatalf("committed exclusion list = %#v, want empty", elements)
	}
	rule, err := NewRule(syntheticRows())
	if err != nil {
		t.Fatalf("NewRule: %v", err)
	}
	decision := rule.Decide([]Live{{ResourceID: resA, Element: excluded}})
	if got := decision.ResourceIDs(); !reflect.DeepEqual(got, []string{resA}) {
		t.Fatalf("committed rule withheld an element that no list names: %#v", decision)
	}
}

func TestMalformedExclusionListIsRefused(t *testing.T) {
	cases := map[string]string{
		"not an object":     `["x"]`,
		"no elements":       `{}`,
		"null elements":     `{"elements":null}`,
		"extra field":       `{"elements":[],"resources":[]}`,
		"duplicate key":     `{"elements":[],"elements":[]}`,
		"duplicate element": `{"elements":["x","x"]}`,
		"empty element":     `{"elements":[""]}`,
		"padded element":    `{"elements":[" x"]}`,
		"not text":          `{"elements":[1]}`,
		"trailing data":     `{"elements":[]} {}`,
		"empty document":    ``,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseExclusions([]byte(document)); err == nil {
				t.Fatal("malformed exclusion list was accepted")
			}
		})
	}
}

// Absence frees nothing: a resource that leaves and returns keeps its number,
// and a number given once is never given to another id.
func TestAbsentResourceKeepsItsNumberAndNoNumberIsReused(t *testing.T) {
	var disk recorder
	store := NewStore()
	store.Assign([]string{resA, resB}, disk.save)
	onlyB := store.Assign([]string{resB}, refuseSave(t))
	if want := map[string]uint16{resB: 2}; !reflect.DeepEqual(onlyB.Endpoints, want) {
		t.Fatalf("assignment without a = %#v", onlyB.Endpoints)
	}
	withC := store.Assign([]string{resC}, disk.save)
	if want := map[string]uint16{resC: 3}; !reflect.DeepEqual(withC.Endpoints, want) {
		t.Fatalf("a new id reused a number: %#v", withC.Endpoints)
	}
	reloaded, err := Load(disk.last(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	back := reloaded.Assign([]string{resA, resA}, refuseSave(t))
	if want := map[string]uint16{resA: 1}; !reflect.DeepEqual(back.Endpoints, want) {
		t.Fatalf("returning resource = %#v", back.Endpoints)
	}
}

func TestStoreAssignsTheLowestFreeNumber(t *testing.T) {
	store, err := Load([]byte(`{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-b","endpoint":3}]}`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var disk recorder
	result := store.Assign([]string{resD, resC}, disk.save)
	if want := map[string]uint16{resC: 2, resD: 4}; !reflect.DeepEqual(result.Endpoints, want) {
		t.Fatalf("assignment = %#v, want %#v", result.Endpoints, want)
	}
	want := `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-c","endpoint":2},{"resource_id":"synthetic-resource-b","endpoint":3},{"resource_id":"synthetic-resource-d","endpoint":4}]}`
	if got := string(disk.last(t)); got != want {
		t.Fatalf("saved = %s, want %s", got, want)
	}
}

func TestFailedSaveExposesNothingNew(t *testing.T) {
	disk := recorder{}
	store := NewStore()
	store.Assign([]string{resA}, disk.save)
	before := store.Document()
	disk.fail = errors.New("synthetic save failure")
	result := store.Assign([]string{resA, resB}, disk.save)
	if want := map[string]uint16{resA: 1}; !reflect.DeepEqual(result.Endpoints, want) {
		t.Fatalf("endpoints = %#v", result.Endpoints)
	}
	if want := map[string]string{resB: ReasonUnsaved}; !reflect.DeepEqual(result.Refused, want) {
		t.Fatalf("refused = %#v", result.Refused)
	}
	if !bytes.Equal(store.Document(), before) {
		t.Fatal("store changed after a failed save")
	}
	noSaver := store.Assign([]string{resB}, nil)
	if want := map[string]string{resB: ReasonUnsaved}; !reflect.DeepEqual(noSaver.Refused, want) {
		t.Fatalf("refused without a saver = %#v", noSaver.Refused)
	}
	disk.fail = nil
	retry := store.Assign([]string{resB}, disk.save)
	if want := map[string]uint16{resB: 2}; !reflect.DeepEqual(retry.Endpoints, want) {
		t.Fatalf("retry = %#v", retry.Endpoints)
	}
}

func TestStoreRefusesAMalformedID(t *testing.T) {
	store := NewStore()
	result := store.Assign([]string{"", " " + resA, resA + "\n"}, refuseSave(t))
	if len(result.Endpoints) != 0 || len(result.Refused) != 3 {
		t.Fatalf("assignment = %#v", result)
	}
	for id, reason := range result.Refused {
		if reason != ReasonResourceID {
			t.Fatalf("%q refused with %q", id, reason)
		}
	}
	if store.Len() != 0 {
		t.Fatal("a malformed id was stored")
	}
}

func TestMalformedStoreDocumentIsRefused(t *testing.T) {
	records := make([]string, 0, StoreCapacity+1)
	for index := 1; index <= StoreCapacity+1; index++ {
		records = append(records, fmt.Sprintf(`{"resource_id":%q,"endpoint":%d}`, syntheticID(index), index))
	}
	overfull := `{"endpoints":[` + joinComma(records) + `]}`
	cases := map[string]string{
		"empty document":    ``,
		"not json":          `{not json`,
		"not an object":     `[]`,
		"no endpoints":      `{}`,
		"null endpoints":    `{"endpoints":null}`,
		"extra field":       `{"endpoints":[],"node":"x"}`,
		"duplicate key":     `{"endpoints":[],"endpoints":[]}`,
		"null record":       `{"endpoints":[null]}`,
		"record extra":      `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1,"element":"x"}]}`,
		"record no id":      `{"endpoints":[{"endpoint":1}]}`,
		"record no number":  `{"endpoints":[{"resource_id":"synthetic-resource-a"}]}`,
		"record dup key":    `{"endpoints":[{"resource_id":"synthetic-resource-a","resource_id":"synthetic-resource-b","endpoint":1}]}`,
		"empty id":          `{"endpoints":[{"resource_id":"","endpoint":1}]}`,
		"padded id":         `{"endpoints":[{"resource_id":" synthetic-resource-a","endpoint":1}]}`,
		"number zero":       `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":0}]}`,
		"number too large":  `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":65535}]}`,
		"number fraction":   `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1.5}]}`,
		"number as text":    `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":"1"}]}`,
		"number as boolean": `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":true}]}`,
		"duplicate id":      `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-a","endpoint":2}]}`,
		"duplicate number":  `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-b","endpoint":1}]}`,
		"trailing data":     `{"endpoints":[]}{}`,
		"over capacity":     overfull,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			store, err := Load([]byte(document))
			if err == nil || store != nil {
				t.Fatalf("malformed document was accepted: %v", store)
			}
		})
	}
}

func joinComma(items []string) string {
	var buffer bytes.Buffer
	for index, item := range items {
		if index > 0 {
			buffer.WriteByte(',')
		}
		buffer.WriteString(item)
	}
	return buffer.String()
}
