package hvac

import (
	"math"
	"reflect"
	"testing"
)

func temperature() Vector {
	return Vector{
		Key:      Key{"Thermal", "synthetic-native", "family-a/profile-a", "room-a", "air", Observation, "unresolved-temperature"},
		Evidence: Evidence{"synthetic-public-contract", "synthetic-source-fact", "Proven", "Unknown"},
		Raw:      []byte{21, 5}, Kind: "temperature", Role: "measured-room", Unit: "C", Celsius: number(21.5),
		Heating: Support{State: "Unknown"}, Cooling: Support{State: "Unknown"}, Loss: []string{"sensor-placement-not-target-closure"},
	}
}
func mode() Vector {
	v := temperature()
	v.Key.Fact, v.Key.Target = "selected-mode", "unresolved-mode"
	v.Kind, v.Role, v.Unit, v.Celsius = "mode", "selected-mode", "", nil
	v.Heating = Support{"Yes", "synthetic-heating-fact"}
	v.Cooling = Support{"No", "synthetic-no-cooling-fact"}
	v.NativeCode = "native-manual"
	v.Codebook = []Mode{
		{Code: "native-off", NativeMeaning: "source disabled selection", Equivalent: "candidate-disabled", Reference: "synthetic-off-equivalence", Requirement: "none", Loss: []string{}},
		{Code: "native-manual", NativeMeaning: "source manual heating selection", Equivalent: "candidate-manual-heating", Reference: "synthetic-manual-equivalence", Requirement: "heating", Loss: []string{"schedule-context-retained-only"}},
	}
	return v
}
func row(t *testing.T, r Result, k Key) Evaluation {
	t.Helper()
	for _, e := range r.Rows {
		if e.Source.Key == k {
			return e
		}
	}
	t.Fatalf("missing full key: %+v", k)
	return Evaluation{}
}
func withheld(t *testing.T, v Vector, reason string) {
	t.Helper()
	r := Validate([]Vector{v})
	if len(r.Rows) != 1 || r.Rows[0].Candidate != nil {
		t.Fatalf("expected one withheld row: %+v", r)
	}
	for _, s := range r.Rows[0].Withheld {
		if s == reason {
			return
		}
	}
	t.Fatalf("missing reason %q: %v", reason, r.Rows[0].Withheld)
}

func TestValidSyntheticIndependentExpectations(t *testing.T) {
	a, b := temperature(), mode()
	r := Validate([]Vector{b, a})
	if len(r.Rows) != 2 {
		t.Fatal(r)
	}
	if !reflect.DeepEqual(row(t, r, a.Key).Source, a) {
		t.Fatal("source evidence or loss changed")
	}
	wantTemp := &Projection{Role: "measured-room", Celsius: number(21.5)}
	wantMode := &Projection{Role: "selected-mode", Equivalent: "candidate-manual-heating"}
	if e := row(t, r, a.Key); !reflect.DeepEqual(e.Candidate, wantTemp) || len(e.Withheld) != 0 {
		t.Fatalf("temperature: %+v", e)
	}
	if e := row(t, r, b.Key); !reflect.DeepEqual(e.Candidate, wantMode) || len(e.Withheld) != 0 {
		t.Fatalf("mode: %+v", e)
	}
	if r.SemanticMapping != "Unknown" || r.TargetClosure != "Unknown" || r.LibrarySupport != "Unknown" || r.ProductionAdmitted {
		t.Fatal("candidate upgraded separate outcomes", r)
	}
	if row(t, r, a.Key).Source.Evidence.Qualification != "Unknown" {
		t.Fatal("qualification inferred")
	}
}

// Every Key field has both a missing-field case and an independent changed-axis
// case. Removing that field from completeness or duplicate counting must fail.
func TestEveryKeyAxis(t *testing.T) {
	axes := []struct {
		name  string
		field func(*Key) *string
		other string
		valid bool
	}{
		{"Domain", func(k *Key) *string { return &k.Domain }, "PV", false},
		{"Owner", func(k *Key) *string { return &k.Owner }, "synthetic-other-native", true},
		{"Profile", func(k *Key) *string { return &k.Profile }, "family-b/profile-b", true},
		{"Scope", func(k *Key) *string { return &k.Scope }, "room-b", true},
		{"Fact", func(k *Key) *string { return &k.Fact }, "other-air", true},
		{"Direction", func(k *Key) *string { return &k.Direction }, "Matter-to-native operation", false},
		{"Target", func(k *Key) *string { return &k.Target }, "other-unresolved-component", true},
	}
	for _, axis := range axes {
		t.Run(axis.name+"/missing", func(t *testing.T) {
			v := temperature()
			*axis.field(&v.Key) = ""
			withheld(t, v, "incomplete identity")
		})
		t.Run(axis.name+"/changed", func(t *testing.T) {
			a, b := temperature(), temperature()
			*axis.field(&b.Key) = axis.other
			r := Validate([]Vector{a, b})
			if len(r.Rows) != 2 || row(t, r, a.Key).Candidate == nil {
				t.Fatal("distinct sibling collapsed", r)
			}
			if (row(t, r, b.Key).Candidate != nil) != axis.valid {
				t.Fatal("wrong changed-axis disposition", r)
			}
		})
	}
	v := temperature()
	r := Validate([]Vector{v, v})
	for _, e := range r.Rows {
		if e.Candidate != nil || !reflect.DeepEqual(e.Withheld, []string{"duplicate full key"}) {
			t.Fatal("duplicate accepted", e)
		}
	}
}

func TestMalformedEvidenceAndValues(t *testing.T) {
	cases := []struct {
		name, reason string
		change       func(*Vector)
	}{
		{"native-contract", "incomplete native evidence", func(v *Vector) { v.Evidence.NativeContract = "" }},
		{"source-reference", "incomplete native evidence", func(v *Vector) { v.Evidence.SourceReference = "" }},
		{"qualification", "incomplete native evidence", func(v *Vector) { v.Evidence.Qualification = "" }},
		{"confidence-enum", "incomplete native evidence", func(v *Vector) { v.Evidence.Confidence = "certain" }},
		{"confidence-unknown", "source confidence is not Proven", func(v *Vector) { v.Evidence.Confidence = "Unknown" }},
		{"confidence-hypothesis", "source confidence is not Proven", func(v *Vector) { v.Evidence.Confidence = "Hypothesis" }},
		{"raw", "incomplete native evidence", func(v *Vector) { v.Raw = nil }},
		{"loss-unreviewed", "unreviewed source loss", func(v *Vector) { v.Loss = nil }},
		{"loss-blank", "unreviewed source loss", func(v *Vector) { v.Loss = []string{" "} }},
		{"kind", "unknown observation kind", func(v *Vector) { v.Kind = "command" }},
		{"role", "invalid temperature role or shape", func(v *Vector) { v.Role = "flow-or-dew-point" }},
		{"unit", "invalid temperature role or shape", func(v *Vector) { v.Unit = "F" }},
		{"mode-code-on-temperature", "invalid temperature role or shape", func(v *Vector) { v.NativeCode = "0" }},
		{"mode-book-on-temperature", "invalid temperature role or shape", func(v *Vector) { v.Codebook = mode().Codebook }},
		{"unavailable", "temperature unavailable or nonfinite", func(v *Vector) { v.Celsius = nil }},
		{"nan", "temperature unavailable or nonfinite", func(v *Vector) { v.Celsius = number(math.NaN()) }},
		{"positive-infinity", "temperature unavailable or nonfinite", func(v *Vector) { v.Celsius = number(math.Inf(1)) }},
		{"negative-infinity", "temperature unavailable or nonfinite", func(v *Vector) { v.Celsius = number(math.Inf(-1)) }},
		{"heating-enum", "invalid capability evidence", func(v *Vector) { v.Heating.State = "" }},
		{"cooling-enum", "invalid capability evidence", func(v *Vector) { v.Cooling.State = "maybe" }},
		{"heating-reference", "invalid capability evidence", func(v *Vector) { v.Heating = Support{State: "Yes"} }},
		{"cooling-reference", "invalid capability evidence", func(v *Vector) { v.Cooling = Support{State: "No"} }},
		{"identity-whitespace", "incomplete identity", func(v *Vector) { v.Key.Profile = " profile " }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { v := temperature(); c.change(&v); withheld(t, v, c.reason) })
	}
}

func TestTemperatureRolesAndCapabilitySeparation(t *testing.T) {
	for _, r := range []string{"heating-configured-target", "heating-manual-target", "heating-effective-target", "cooling-configured-target", "cooling-manual-target", "cooling-effective-target"} {
		t.Run(r, func(t *testing.T) {
			v := temperature()
			v.Role = r
			withheld(t, v, "target role lacks explicit capability")
			v.Heating = Support{"Yes", "synthetic-heat"}
			if r[:7] == "cooling" {
				withheld(t, v, "target role lacks explicit capability")
				v.Cooling = Support{"Yes", "synthetic-cool"}
			}
			e := Validate([]Vector{v}).Rows[0]
			if !reflect.DeepEqual(e.Candidate, &Projection{Role: r, Celsius: number(21.5)}) {
				t.Fatal(e)
			}
			v.Key.Direction = "Matter-to-native operation"
			withheld(t, v, "only Thermal observations are candidates")
		})
	}
	for _, state := range []string{"Unknown", "No"} {
		t.Run("measured-with-"+state, func(t *testing.T) {
			v := temperature()
			v.Heating = Support{state, "synthetic-heat"}
			v.Cooling = Support{state, "synthetic-cool"}
			if Validate([]Vector{v}).Rows[0].Candidate == nil {
				t.Fatal("measurement inferred capability requirement")
			}
		})
	}
	v := temperature()
	v.Celsius = number(0)
	if e := Validate([]Vector{v}).Rows[0]; e.Candidate == nil || *e.Candidate.Celsius != 0 {
		t.Fatal("zero treated as missing", e)
	}
}

func TestOptionalSourceConstraints(t *testing.T) {
	cases := []struct {
		name, reason string
		b            Constraints
		value        float64
	}{
		{"lower-inclusive", "", Constraints{18, 25, "synthetic-bounds"}, 18},
		{"upper-inclusive", "", Constraints{18, 25, "synthetic-bounds"}, 25},
		{"below", "outside supplied source bounds", Constraints{18, 25, "synthetic-bounds"}, 17},
		{"above", "outside supplied source bounds", Constraints{18, 25, "synthetic-bounds"}, 26},
		{"reference", "invalid optional source bounds", Constraints{18, 25, ""}, 21},
		{"reversed", "invalid optional source bounds", Constraints{25, 18, "synthetic-bounds"}, 21},
		{"min-nan", "invalid optional source bounds", Constraints{math.NaN(), 25, "synthetic-bounds"}, 21},
		{"max-nan", "invalid optional source bounds", Constraints{18, math.NaN(), "synthetic-bounds"}, 21},
		{"min-infinite", "invalid optional source bounds", Constraints{math.Inf(-1), 25, "synthetic-bounds"}, 21},
		{"max-infinite", "invalid optional source bounds", Constraints{18, math.Inf(1), "synthetic-bounds"}, 21},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := temperature()
			v.Bounds = &c.b
			v.Celsius = number(c.value)
			if c.reason != "" {
				withheld(t, v, c.reason)
			} else {
				e := Validate([]Vector{v}).Rows[0]
				if !reflect.DeepEqual(e.Candidate, &Projection{Role: "measured-room", Celsius: number(c.value)}) {
					t.Fatal(e)
				}
			}
		})
	}
}

func TestEveryModeRequiresSuppliedEquivalence(t *testing.T) {
	cases := []struct {
		name, reason string
		change       func(*Vector)
	}{
		{"requirement-missing", "mode capability requirement is not explicit", func(v *Vector) { v.Codebook[0].Requirement = "" }},
		{"requirement-unknown", "mode capability requirement is not explicit", func(v *Vector) { v.Codebook[0].Requirement = "automatic" }},
		{"combined-requirement", "codebook requires unsupported or unknown capability", func(v *Vector) { v.Codebook[0].Requirement = "heating-and-cooling" }},
		{"no-codebook", "invalid mode shape", func(v *Vector) { v.Codebook = nil }},
		{"no-current-code", "invalid mode shape", func(v *Vector) { v.NativeCode = "" }},
		{"unknown-current-code", "native mode has no explicit equivalence", func(v *Vector) { v.NativeCode = "native-unmapped" }},
		{"role", "invalid mode shape", func(v *Vector) { v.Role = "action" }},
		{"unit", "invalid mode shape", func(v *Vector) { v.Unit = "C" }},
		{"temperature", "invalid mode shape", func(v *Vector) { v.Celsius = number(20) }},
		{"bounds", "invalid mode shape", func(v *Vector) { v.Bounds = &Constraints{18, 25, "synthetic-bounds"} }},
		{"entry-code", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Code = "" }},
		{"entry-meaning", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].NativeMeaning = "" }},
		{"unselected-equivalence", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Equivalent = "" }},
		{"selected-equivalence", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[1].Equivalent = "" }},
		{"entry-reference", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Reference = "" }},
		{"entry-loss", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Loss = nil }},
		{"entry-loss-blank", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Loss = []string{""} }},
		{"duplicate-code", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Code = v.Codebook[1].Code }},
		{"default-code", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Code = "default" }},
		{"wildcard-code", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Code = "*" }},
		{"default-equivalence", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Equivalent = "DEFAULT" }},
		{"wildcard-equivalence", "codebook lacks unique explicit equivalence or loss", func(v *Vector) { v.Codebook[0].Equivalent = "*" }},
		{"heating-unknown", "codebook requires unsupported or unknown capability", func(v *Vector) { v.Heating = Support{State: "Unknown"} }},
		{"heating-no", "codebook requires unsupported or unknown capability", func(v *Vector) { v.Heating = Support{"No", "synthetic-no-heat"} }},
		{"cooling-unknown", "codebook requires unsupported or unknown capability", func(v *Vector) { v.Codebook[0].Requirement = "cooling"; v.Cooling = Support{State: "Unknown"} }},
		{"cooling-no", "codebook requires unsupported or unknown capability", func(v *Vector) { v.Codebook[0].Requirement = "cooling" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { v := mode(); c.change(&v); withheld(t, v, c.reason) })
	}
	v := mode()
	v.Cooling = Support{"Yes", "synthetic-cooling-fact"}
	v.Codebook[0].Requirement = "cooling"
	v.NativeCode = "native-off"
	if e := Validate([]Vector{v}).Rows[0]; !reflect.DeepEqual(e.Candidate, &Projection{Role: "selected-mode", Equivalent: "candidate-disabled"}) {
		t.Fatal(e)
	}
}

func TestPermutationDetachmentAndPartialWithholding(t *testing.T) {
	a, b, c := temperature(), mode(), temperature()
	c.Key.Scope = "room-b"
	c.Celsius = nil
	a.Bounds = &Constraints{18, 25, "synthetic-bounds"}
	input := []Vector{a, b, c}
	before := []Vector{temperature(), mode(), temperature()}
	before[0].Bounds = &Constraints{18, 25, "synthetic-bounds"}
	before[2].Key.Scope = "room-b"
	before[2].Celsius = nil
	r := Validate(input)
	if !reflect.DeepEqual(input, before) {
		t.Fatal("input mutated")
	}
	for _, p := range [][]Vector{{c, b, a}, {b, a, c}, {b, c, a}, {a, c, b}, {c, a, b}} {
		if !reflect.DeepEqual(Validate(p), r) {
			t.Fatal("row permutation changed envelope")
		}
	}
	b.Codebook[0], b.Codebook[1] = b.Codebook[1], b.Codebook[0]
	if !reflect.DeepEqual(Validate([]Vector{a, b, c}), r) {
		t.Fatal("codebook permutation changed envelope")
	}
	if row(t, r, a.Key).Candidate == nil || row(t, r, c.Key).Candidate != nil || !reflect.DeepEqual(row(t, r, c.Key).Source.Raw, []byte{21, 5}) {
		t.Fatal("partial loss erased sibling/native evidence")
	}
	*input[0].Celsius = 99
	input[0].Raw[0] = 99
	input[0].Loss[0] = "changed"
	input[0].Bounds.MinC = 99
	for i := range input[1].Codebook {
		if len(input[1].Codebook[i].Loss) > 0 {
			input[1].Codebook[i].Loss[0] = "changed"
		}
	}
	for _, m := range row(t, r, b.Key).Source.Codebook {
		if m.Code == "native-manual" && !reflect.DeepEqual(m.Loss, []string{"schedule-context-retained-only"}) {
			t.Fatal("input codebook loss aliases result")
		}
	}
	e := row(t, r, a.Key)
	if *e.Candidate.Celsius != 21.5 || *e.Source.Celsius != 21.5 || e.Source.Raw[0] != 21 || e.Source.Bounds.MinC != 18 || e.Source.Loss[0] != "sensor-placement-not-target-closure" {
		t.Fatal("input aliases result")
	}
	// Mutate every output reference and verify a separately supplied vector remains intact.
	v := mode()
	v.Codebook[1].Loss = []string{"schedule-context-retained-only"}
	out := Validate([]Vector{v})
	out.Rows[0].Source.Raw[0] = 0
	out.Rows[0].Source.Loss[0] = "changed"
	for i := range out.Rows[0].Source.Codebook {
		if len(out.Rows[0].Source.Codebook[i].Loss) > 0 {
			out.Rows[0].Source.Codebook[i].Loss[0] = "changed"
		}
	}
	if v.Raw[0] != 21 || v.Loss[0] != "sensor-placement-not-target-closure" || v.Codebook[1].Loss[0] != "schedule-context-retained-only" {
		t.Fatal("output aliases input")
	}
	tv := temperature()
	tv.Bounds = &Constraints{18, 25, "synthetic-bounds"}
	tr := Validate([]Vector{tv})
	*tr.Rows[0].Candidate.Celsius = 99
	if *tr.Rows[0].Source.Celsius != 21.5 {
		t.Fatal("projection aliases retained source")
	}
	*tr.Rows[0].Source.Celsius = 19
	tr.Rows[0].Source.Bounds.MinC = 0
	if *tr.Rows[0].Source.Celsius != 19 || *tv.Celsius != 21.5 || tv.Bounds.MinC != 18 {
		t.Fatal("projection aliases source")
	}
	// Duplicate nil and reviewed-empty loss must not share an ordering key.
	n, z := temperature(), temperature()
	n.Loss = nil
	z.Loss = []string{}
	nr, zr := Validate([]Vector{n, z}), Validate([]Vector{z, n})
	if !reflect.DeepEqual(nr, zr) {
		t.Fatal("nil/empty loss permutation changed result")
	}
	// Duplicate malformed rows also have a deterministic ordering.
	d := temperature()
	d.Celsius = number(math.NaN())
	f := temperature()
	f.Celsius = number(math.Inf(1))
	x, y := Validate([]Vector{d, f}), Validate([]Vector{f, d})
	for i := range x.Rows {
		xn, yn := *x.Rows[i].Source.Celsius, *y.Rows[i].Source.Celsius
		if !(math.IsNaN(xn) && math.IsNaN(yn)) && xn != yn {
			t.Fatal("malformed duplicates reorder")
		}
	}
}

func TestExplicitCombinedModeAndUnknownIsolation(t *testing.T) {
	v := mode()
	v.Codebook[1].Requirement = "heating-and-cooling"
	withheld(t, v, "codebook requires unsupported or unknown capability")
	v.Cooling = Support{"Yes", "synthetic-independent-cool"}
	r := Validate([]Vector{v})
	if !reflect.DeepEqual(r.Rows[0].Candidate, &Projection{Role: "selected-mode", Equivalent: "candidate-manual-heating"}) {
		t.Fatal(r)
	}
	// Qualification text is retained without changing semantic/target/library outcomes.
	v.Evidence.Qualification = "synthetic-qualified-assertion"
	r = Validate([]Vector{v})
	if r.Rows[0].Source.Evidence.Qualification != "synthetic-qualified-assertion" || r.SemanticMapping != "Unknown" || r.TargetClosure != "Unknown" || r.LibrarySupport != "Unknown" || r.ProductionAdmitted {
		t.Fatal("qualification upgraded independent outcomes", r)
	}
	// Both absent and explicitly denied source capabilities withhold target roles.
	for _, role := range []string{"heating-manual-target", "cooling-manual-target"} {
		tv := temperature()
		tv.Role = role
		tv.Heating = Support{"No", "synthetic-no-heat"}
		tv.Cooling = Support{"No", "synthetic-no-cool"}
		withheld(t, tv, "target role lacks explicit capability")
	}
}

func TestEmptyCannotAdmitOrClose(t *testing.T) {
	r := Validate(nil)
	if len(r.Rows) != 0 || r.ProductionAdmitted || r.SemanticMapping != "Unknown" || r.TargetClosure != "Unknown" || r.LibrarySupport != "Unknown" {
		t.Fatal(r)
	}
}
