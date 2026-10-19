// Package hvac validates internal, read-only contribution-candidate vectors.
// It neither implements a Matter target nor qualifies supplied native evidence.
package hvac

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"
)

const Observation = "native-to-Matter observation"

// Key is a source-fact/target-candidate join, never an endpoint or fabric key.
// Target is an opaque review slot, not a normative Matter component identifier.
type Key struct {
	Domain, Owner, Profile, Scope, Fact, Direction, Target string
}

type Evidence struct {
	NativeContract, SourceReference string
	Confidence                      string // Proven, Hypothesis or Unknown; caller assertions only.
	Qualification                   string // Retained separately; never inferred from confidence.
}

// Support separates unknown, unsupported and explicitly supported capability.
// Its evidence does not establish selected mode, action or command support.
type Support struct {
	State, Reference string // Unknown, No or Yes.
}

// Constraints are supplied source bounds for this role, never target limits.
type Constraints struct {
	MinC, MaxC float64
	Reference  string
}

// Mode supplies an explicit equivalence assertion for one native code.
// Equivalent is an opaque candidate label, not a Matter enum assignment.
type Mode struct {
	Code, NativeMeaning, Equivalent, Reference string
	Requirement                                string // none, heating, cooling or heating-and-cooling.
	Loss                                       []string
}

// Vector retains raw source context even when equivalence is withheld.
// Temperature roles are measured-room, heating/cooling-configured-target,
// heating/cooling-manual-target and heating/cooling-effective-target.
// A nil Celsius means unavailable, never zero or a replacement for retained data.
type Vector struct {
	Key              Key
	Evidence         Evidence
	Raw              []byte
	Kind, Role, Unit string // Kind is temperature or mode; temperature unit is C.
	Celsius          *float64
	Bounds           *Constraints
	Heating, Cooling Support
	NativeCode       string
	Codebook         []Mode
	Loss             []string // Non-nil means loss was explicitly reviewed; empty is valid.
}

type Projection struct {
	Role, Equivalent string
	Celsius          *float64
}

type Evaluation struct {
	Source    Vector
	Candidate *Projection
	Withheld  []string
}

type Result struct {
	Rows                                           []Evaluation
	SemanticMapping, TargetClosure, LibrarySupport string
	ProductionAdmitted                             bool
}

// Validate is pure and detached. A candidate means structural equivalence only.
// No supplied metadata can close the target or change runtime admission.
func Validate(input []Vector) Result {
	r := Result{Rows: make([]Evaluation, 0, len(input)), SemanticMapping: "Unknown", TargetClosure: "Unknown", LibrarySupport: "Unknown"}
	counts := make(map[Key]int)
	for _, v := range input {
		counts[v.Key]++
	}
	for _, original := range input {
		v := clone(original)
		e := Evaluation{Source: v, Withheld: []string{}}
		deny := func(reason string) { e.Withheld = append(e.Withheld, reason) }
		k := v.Key
		if !text(k.Domain) || !text(k.Owner) || !text(k.Profile) || !text(k.Scope) || !text(k.Fact) || !text(k.Direction) || !text(k.Target) {
			deny("incomplete identity")
		}
		if k.Domain != "Thermal" || k.Direction != Observation {
			deny("only Thermal observations are candidates")
		}
		if counts[k] != 1 {
			deny("duplicate full key")
		}
		if !text(v.Evidence.NativeContract) || !text(v.Evidence.SourceReference) || !text(v.Evidence.Qualification) || len(v.Raw) == 0 || !grade(v.Evidence.Confidence) {
			deny("incomplete native evidence")
		}
		if v.Evidence.Confidence != "Proven" {
			deny("source confidence is not Proven")
		}
		if !loss(v.Loss) {
			deny("unreviewed source loss")
		}
		if !support(v.Heating) || !support(v.Cooling) {
			deny("invalid capability evidence")
		}
		switch v.Kind {
		case "temperature":
			heat, cool, valid := role(v.Role)
			if !valid || v.Unit != "C" || v.NativeCode != "" || len(v.Codebook) != 0 {
				deny("invalid temperature role or shape")
			}
			if (heat && v.Heating.State != "Yes") || (cool && v.Cooling.State != "Yes") {
				deny("target role lacks explicit capability")
			}
			if v.Celsius == nil || !finite(*v.Celsius) {
				deny("temperature unavailable or nonfinite")
			}
			if b := v.Bounds; b != nil {
				if !text(b.Reference) || !finite(b.MinC) || !finite(b.MaxC) || b.MinC > b.MaxC {
					deny("invalid optional source bounds")
				} else if v.Celsius != nil && (*v.Celsius < b.MinC || *v.Celsius > b.MaxC) {
					deny("outside supplied source bounds")
				}
			}
			if len(e.Withheld) == 0 {
				e.Candidate = &Projection{Role: v.Role, Celsius: number(*v.Celsius)}
			}
		case "mode":
			if v.Role != "selected-mode" || v.Unit != "" || v.Celsius != nil || v.Bounds != nil || !code(v.NativeCode) || len(v.Codebook) == 0 {
				deny("invalid mode shape")
			}
			seen := make(map[string]bool)
			selected := ""
			for _, m := range v.Codebook {
				if !code(m.Code) || seen[m.Code] || !text(m.NativeMeaning) || !code(m.Equivalent) || !text(m.Reference) || !loss(m.Loss) {
					deny("codebook lacks unique explicit equivalence or loss")
				}
				seen[m.Code] = true
				if m.Requirement != "none" && m.Requirement != "heating" && m.Requirement != "cooling" && m.Requirement != "heating-and-cooling" {
					deny("mode capability requirement is not explicit")
				}
				if ((m.Requirement == "heating" || m.Requirement == "heating-and-cooling") && v.Heating.State != "Yes") || ((m.Requirement == "cooling" || m.Requirement == "heating-and-cooling") && v.Cooling.State != "Yes") {
					deny("codebook requires unsupported or unknown capability")
				}
				if m.Code == v.NativeCode {
					selected = m.Equivalent
				}
			}
			if selected == "" {
				deny("native mode has no explicit equivalence")
			}
			if len(e.Withheld) == 0 {
				e.Candidate = &Projection{Role: v.Role, Equivalent: selected}
			}
		default:
			deny("unknown observation kind")
		}
		r.Rows = append(r.Rows, e)
	}
	// Canonical ordering covers retained values, including unavailable/nonfinite data.
	// Sorting the full retained vector also makes malformed duplicate rows stable.
	sort.Slice(r.Rows, func(i, j int) bool { return bytes.Compare(order(r.Rows[i].Source), order(r.Rows[j].Source)) < 0 })
	return r
}

func text(s string) bool { return s != "" && strings.TrimSpace(s) == s }
func code(s string) bool {
	return text(s) && s != "*" && !strings.EqualFold(s, "default")
}
func grade(s string) bool { return s == "Proven" || s == "Hypothesis" || s == "Unknown" }
func support(s Support) bool {
	return s.State == "Unknown" || ((s.State == "Yes" || s.State == "No") && text(s.Reference))
}
func finite(n float64) bool     { return !math.IsNaN(n) && !math.IsInf(n, 0) }
func number(n float64) *float64 { return &n }
func loss(s []string) bool {
	if s == nil {
		return false
	}
	for _, item := range s {
		if !text(item) {
			return false
		}
	}
	return true
}
func role(s string) (heat, cool, valid bool) {
	switch s {
	case "measured-room":
		return false, false, true
	case "heating-configured-target", "heating-manual-target", "heating-effective-target":
		return true, false, true
	case "cooling-configured-target", "cooling-manual-target", "cooling-effective-target":
		return false, true, true
	}
	return false, false, false
}
func clone(v Vector) Vector {
	v.Raw = append([]byte(nil), v.Raw...)
	if v.Celsius != nil {
		v.Celsius = number(*v.Celsius)
	}
	if v.Bounds != nil {
		b := *v.Bounds
		v.Bounds = &b
	}
	v.Loss = stringsCopy(v.Loss)
	if v.Codebook != nil {
		v.Codebook = append([]Mode{}, v.Codebook...)
		for i := range v.Codebook {
			v.Codebook[i].Loss = stringsCopy(v.Codebook[i].Loss)
		}
	}
	// Codebook order is not semantic; normalize only the detached copy.
	sort.Slice(v.Codebook, func(i, j int) bool { return bytes.Compare(order(v.Codebook[i]), order(v.Codebook[j])) < 0 })
	return v
}
func stringsCopy(s []string) []string {
	if s == nil {
		return nil
	}
	return append([]string{}, s...)
}

// Explicit tuples avoid pointer addresses and distinguish nil from reviewed empty
// slices. This is only an ordering key; it is not a persistence or wire format.
func order(value interface{}) []byte {
	switch v := value.(type) {
	case Mode:
		return []byte(fmt.Sprintf("%#v", []interface{}{v.Code, v.NativeMeaning, v.Equivalent, v.Reference, v.Requirement, v.Loss}))
	case Vector:
		var c, b interface{}
		if v.Celsius != nil {
			c = math.Float64bits(*v.Celsius)
		}
		if v.Bounds != nil {
			b = []interface{}{math.Float64bits(v.Bounds.MinC), math.Float64bits(v.Bounds.MaxC), v.Bounds.Reference}
		}
		return []byte(fmt.Sprintf("%#v", []interface{}{v.Key, v.Evidence, v.Raw, v.Kind, v.Role, v.Unit, c, b, v.Heating, v.Cooling, v.NativeCode, v.Codebook, v.Loss}))
	default:
		panic("unsupported internal ordering value")
	}
}
