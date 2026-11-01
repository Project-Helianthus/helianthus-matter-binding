package hvac

import (
	"math"
	"reflect"
	"testing"
)

var aa13Subject = Subject161{"synthetic-native", "family-a/profile-a", "room-a"}

func aa13Yes(ref string) Support { return Support{"Yes", ref} }

func aa13Temp(role string, celsius float64) Vector {
	return Vector{
		Key:      Key{"Thermal", "synthetic-native", "family-a/profile-a", "room-a", "fact-" + role, Observation, "slot-" + role},
		Evidence: Evidence{"synthetic-contract-" + role, "synthetic-source-" + role, "Proven", "Unknown"},
		Raw:      []byte(role), Kind: "temperature", Role: role, Unit: "C", Celsius: number(celsius),
		Heating: aa13Yes("synthetic-heat"), Cooling: aa13Yes("synthetic-cool"), Loss: []string{"loss-" + role},
	}
}

func aa13Mode(code string) Vector {
	v := aa13Temp("measured-room", 0)
	v.Key.Fact, v.Key.Target = "fact-mode", "slot-mode"
	v.Evidence = Evidence{"synthetic-contract-mode", "synthetic-source-mode", "Proven", "Unknown"}
	v.Raw, v.Loss = []byte("mode"), []string{"loss-mode"}
	v.Kind, v.Role, v.Unit, v.Celsius, v.NativeCode = "mode", "selected-mode", "", nil, code
	v.Codebook = []Mode{
		{Code: "native-off", NativeMeaning: "off meaning", Equivalent: "label-off", Reference: "book-off", Requirement: "none", Loss: []string{}},
		{Code: "native-heat", NativeMeaning: "heat meaning", Equivalent: "label-heat", Reference: "book-heat", Requirement: "heating", Loss: []string{"book-heat-loss"}},
		{Code: "native-cool", NativeMeaning: "cool meaning", Equivalent: "label-cool", Reference: "book-cool", Requirement: "cooling", Loss: []string{}},
		{Code: "native-auto", NativeMeaning: "auto meaning", Equivalent: "label-auto", Reference: "book-auto", Requirement: "heating-and-cooling", Loss: []string{"book-auto-loss"}},
	}
	return v
}

func aa13Evidence() TargetEvidence161 {
	return TargetEvidence161{Subject: aa13Subject, Auto: aa13Yes("synthetic-auto"), Modes: []ModeBinding161{
		{"label-off", "OFF", "bind-off", []string{}},
		{"label-heat", "HEAT", "bind-heat", []string{"bind-heat-loss"}},
		{"label-cool", "COOL", "bind-cool", []string{}},
		{"label-auto", "AUTO", "bind-auto", []string{"bind-auto-loss"}},
	}}
}

func aa13One(t *testing.T, input []Vector, evidence ...TargetEvidence161) Thermostat161 {
	t.Helper()
	r := EncodeThermostat161(input, evidence)
	if len(r.Thermostats) != 1 {
		t.Fatalf("thermostats: %+v", r.Thermostats)
	}
	return r.Thermostats[0]
}

func aa13Reason(t *testing.T, th Thermostat161, k Key, reason string) {
	t.Helper()
	for _, w := range th.Withheld {
		if w.Key == k {
			for _, s := range w.Reasons {
				if s == reason {
					return
				}
			}
			t.Fatalf("missing reason %q: %v", reason, w.Reasons)
		}
	}
	t.Fatalf("row not withheld, wanted %q: %+v", reason, th)
}

func aa13Empty(t *testing.T, th Thermostat161) {
	t.Helper()
	if !reflect.DeepEqual(th.Temperatures, Temperatures161{}) {
		t.Fatalf("slot populated: %+v", th.Temperatures)
	}
}

func aa13Has(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// Catches: swapped role slots, a dropped Provenance field and changed identity,
// feature, enum or Unknown-outcome constants. Expected wires are literal.
func TestAA13FullThermostatExactDTO(t *testing.T) {
	roles := []struct {
		role  string
		value float64
		wire  int16
		slot  func(Temperatures161) *Temperature161
	}{
		{"measured-room", 21.5, 2150, func(s Temperatures161) *Temperature161 { return s.MeasuredRoom }},
		{"heating-configured-target", 20.25, 2025, func(s Temperatures161) *Temperature161 { return s.HeatingConfigured }},
		{"heating-manual-target", 19.75, 1975, func(s Temperatures161) *Temperature161 { return s.HeatingManual }},
		{"heating-effective-target", 18.5, 1850, func(s Temperatures161) *Temperature161 { return s.HeatingEffective }},
		{"cooling-configured-target", 26.25, 2625, func(s Temperatures161) *Temperature161 { return s.CoolingConfigured }},
		{"cooling-manual-target", 25.75, 2575, func(s Temperatures161) *Temperature161 { return s.CoolingManual }},
		{"cooling-effective-target", 24.5, 2450, func(s Temperatures161) *Temperature161 { return s.CoolingEffective }},
	}
	input := []Vector{aa13Mode("native-auto")}
	for _, c := range roles {
		input = append(input, aa13Temp(c.role, c.value))
	}
	r := EncodeThermostat161(input, []TargetEvidence161{aa13Evidence()})
	want := Target161{"1.6.1", "Thermostat", 0x0201, 11, 100, -27315, 32767, -32768, "Proven", "Hypothesis"}
	if r.Target != want {
		t.Fatalf("target: %+v", r.Target)
	}
	if len(r.Thermostats) != 1 || len(r.Thermostats[0].Withheld) != 0 || len(r.Thermostats[0].Reasons) != 0 {
		t.Fatalf("%+v", r.Thermostats)
	}
	th := r.Thermostats[0]
	if th.Subject != aa13Subject {
		t.Fatalf("subject: %+v", th.Subject)
	}
	for _, c := range roles {
		wantSlot := &Temperature161{Role: c.role, Wire: c.wire, Source: Provenance161{
			Key:      Key{"Thermal", "synthetic-native", "family-a/profile-a", "room-a", "fact-" + c.role, Observation, "slot-" + c.role},
			Evidence: Evidence{"synthetic-contract-" + c.role, "synthetic-source-" + c.role, "Proven", "Unknown"},
			Raw:      []byte(c.role), Loss: []string{"loss-" + c.role},
		}}
		if got := c.slot(th.Temperatures); !reflect.DeepEqual(got, wantSlot) {
			t.Fatalf("%s: %+v", c.role, got)
		}
	}
	wantMode := &SystemMode161{
		Target: "AUTO", Wire: 1, NativeCode: "native-auto", NativeMeaning: "auto meaning", Equivalent: "label-auto",
		CodebookReference: "book-auto", BindingReference: "bind-auto", CodebookLoss: []string{"book-auto-loss"}, BindingLoss: []string{"bind-auto-loss"},
		Source: Provenance161{
			Key:      Key{"Thermal", "synthetic-native", "family-a/profile-a", "room-a", "fact-mode", Observation, "slot-mode"},
			Evidence: Evidence{"synthetic-contract-mode", "synthetic-source-mode", "Proven", "Unknown"},
			Raw:      []byte("mode"), Loss: []string{"loss-mode"},
		},
	}
	if !reflect.DeepEqual(th.SystemMode, wantMode) {
		t.Fatalf("mode: %+v", th.SystemMode)
	}
	if th.Composition.FeatureMap == nil || *th.Composition.FeatureMap != 0x23 {
		t.Fatalf("feature map: %+v", th.Composition)
	}
	if r.Validation.SemanticMapping != "Unknown" || r.Validation.TargetClosure != "Unknown" || r.Validation.LibrarySupport != "Unknown" || r.Validation.ProductionAdmitted {
		t.Fatalf("outcome upgraded: %+v", r.Validation)
	}
}

// Each role alone fills its own slot and no other, so swapping any two slots fails.
func TestAA13EachRoleFillsOnlyItsOwnSlot(t *testing.T) {
	names := map[string]string{
		"measured-room": "MeasuredRoom", "heating-configured-target": "HeatingConfigured", "heating-manual-target": "HeatingManual",
		"heating-effective-target": "HeatingEffective", "cooling-configured-target": "CoolingConfigured", "cooling-manual-target": "CoolingManual",
		"cooling-effective-target": "CoolingEffective",
	}
	for role, field := range names {
		t.Run(role, func(t *testing.T) {
			th := aa13One(t, []Vector{aa13Temp(role, 21.5)})
			slots := reflect.ValueOf(th.Temperatures)
			for i := 0; i < slots.NumField(); i++ {
				if populated := !slots.Field(i).IsNil(); populated != (slots.Type().Field(i).Name == field) {
					t.Fatalf("slot %s populated=%v", slots.Type().Field(i).Name, populated)
				}
			}
		})
	}
}

// Catches: rounding, tolerance, a missing range check, the wrong scale, and null
// or zero standing in for a value.
func TestAA13TemperatureScaleRangeAndNull(t *testing.T) {
	for _, c := range []struct {
		celsius float64
		wire    int16
	}{{21.5, 2150}, {0, 0}, {21.55, 2155}, {-0.01, -1}, {0.07, 7}, {-273.15, -27315}, {327.67, 32767}} {
		th := aa13One(t, []Vector{aa13Temp("measured-room", c.celsius)})
		if s := th.Temperatures.MeasuredRoom; s == nil || s.Wire != c.wire || s.Null {
			t.Fatalf("%v: %+v", c.celsius, s)
		}
	}
	for _, c := range []struct {
		celsius float64
		reason  string
	}{
		{21.555, "temperature not representable at target scale"},
		{21.5000001, "temperature not representable at target scale"},
		{0.001, "temperature not representable at target scale"},
		{1e-300, "temperature not representable at target scale"},
		{-273.16, "temperature outside target range"},
		{327.68, "temperature outside target range"},
		{-327.68, "temperature outside target range"},
		{1e300, "temperature outside target range"},
	} {
		v := aa13Temp("measured-room", c.celsius)
		th := aa13One(t, []Vector{v})
		aa13Empty(t, th)
		aa13Reason(t, th, v.Key, c.reason)
	}
	for n := int(TemperatureMinWire); n <= int(TemperatureMaxWire); n++ {
		if wire, why := wire161(float64(n) / 100); why != "" || int(wire) != n {
			t.Fatalf("%d encoded as %d (%s)", n, wire, why)
		}
	}
	null := aa13Temp("measured-room", 0)
	null.Celsius = nil
	th := aa13One(t, []Vector{null})
	if s := th.Temperatures.MeasuredRoom; s == nil || !s.Null || s.Wire != -32768 || !reflect.DeepEqual(s.Source.Raw, []byte("measured-room")) {
		t.Fatalf("null: %+v", s)
	}
	if len(th.Withheld) != 0 {
		t.Fatalf("null row reported withheld: %+v", th.Withheld)
	}
	if s := aa13One(t, []Vector{aa13Temp("measured-room", 0)}).Temperatures.MeasuredRoom; s == nil || s.Null || s.Wire != 0 {
		t.Fatalf("zero became null: %+v", s)
	}
}

// Only an otherwise clean measured sample that is truly absent may be null.
func TestAA13NullNeverCoversAnotherDefect(t *testing.T) {
	cases := map[string]func(*Vector){
		"nan":          func(v *Vector) { v.Celsius = number(math.NaN()) },
		"infinity":     func(v *Vector) { v.Celsius = number(math.Inf(1)) },
		"hypothesis":   func(v *Vector) { v.Celsius, v.Evidence.Confidence = nil, "Hypothesis" },
		"unknown":      func(v *Vector) { v.Celsius, v.Evidence.Confidence = nil, "Unknown" },
		"no-raw":       func(v *Vector) { v.Celsius, v.Raw = nil, nil },
		"unreviewed":   func(v *Vector) { v.Celsius, v.Loss = nil, nil },
		"non-nullable": func(v *Vector) { v.Celsius, v.Role = nil, "heating-manual-target" },
		"effective":    func(v *Vector) { v.Celsius, v.Role = nil, "cooling-effective-target" },
		"configured":   func(v *Vector) { v.Celsius, v.Role = nil, "heating-configured-target" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			v := aa13Temp("measured-room", 21.5)
			change(&v)
			th := aa13One(t, []Vector{v})
			aa13Empty(t, th)
			if len(th.Withheld) != 1 || th.Withheld[0].Key != v.Key {
				t.Fatalf("%+v", th.Withheld)
			}
		})
	}
}

func TestAA13FeatureComposition(t *testing.T) {
	yes, no, unknown := func(r string) Support { return Support{"Yes", r} }, func(r string) Support { return Support{"No", r} }, Support{State: "Unknown"}
	mp := func(m uint32) *uint32 { return &m }
	cases := []struct {
		name             string
		heat, cool, auto Support
		wantH, wantC     string
		wantA            string
		wantRefs         []string
		wantMap          *uint32
		reason           string
	}{
		{"all", yes("h"), yes("c"), yes("a"), "Yes", "Yes", "Yes", []string{"a"}, mp(0x23), ""},
		{"both-auto-no", yes("h"), yes("c"), no("a"), "Yes", "Yes", "No", []string{"a"}, mp(0x03), ""},
		{"both-auto-unknown", yes("h"), yes("c"), unknown, "Yes", "Yes", "Unknown", []string{}, nil, ""},
		{"heating-only", yes("h"), no("c"), unknown, "Yes", "No", "No", []string{"c"}, mp(0x01), ""},
		{"cooling-only", no("h"), yes("c"), no("a"), "No", "Yes", "No", []string{"h"}, mp(0x02), ""},
		{"heating-only-claims-auto", yes("h"), no("c"), yes("a"), "Yes", "No", "Unknown", []string{}, nil, "AUTO lacks both heating and cooling"},
		{"cooling-only-claims-auto", no("h"), yes("c"), yes("a"), "No", "Yes", "Unknown", []string{}, nil, "AUTO lacks both heating and cooling"},
		{"cooling-unknown-claims-auto", yes("h"), unknown, yes("a"), "Yes", "Unknown", "Unknown", []string{}, nil, "AUTO lacks both heating and cooling"},
		{"heating-unknown-claims-auto", unknown, yes("c"), yes("a"), "Unknown", "Yes", "Unknown", []string{}, nil, "AUTO lacks both heating and cooling"},
		{"heating-cooling-unknown", yes("h"), unknown, unknown, "Yes", "Unknown", "Unknown", []string{}, nil, ""},
		{"none", unknown, unknown, unknown, "Unknown", "Unknown", "Unknown", []string{}, nil, "no evidenced heating or cooling feature"},
		{"both-no", no("h"), no("c"), no("a"), "No", "No", "No", []string{"c", "h"}, nil, "no evidenced heating or cooling feature"},
		{"auto-reference", yes("h"), yes("c"), Support{State: "Yes"}, "Yes", "Yes", "Unknown", []string{}, nil, "invalid AUTO evidence"},
		{"auto-state", yes("h"), yes("c"), Support{State: "maybe", Reference: "a"}, "Yes", "Yes", "Unknown", []string{}, nil, "invalid AUTO evidence"},
		{"auto-missing", yes("h"), yes("c"), Support{}, "Yes", "Yes", "Unknown", []string{}, nil, "invalid AUTO evidence"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := aa13Temp("measured-room", 21.5)
			v.Heating, v.Cooling = c.heat, c.cool
			ev := aa13Evidence()
			ev.Auto = c.auto
			got := aa13One(t, []Vector{v}, ev)
			comp := got.Composition
			if comp.Heat.State != c.wantH || comp.Cool.State != c.wantC || comp.Auto.State != c.wantA || !reflect.DeepEqual(comp.Auto.References, c.wantRefs) {
				t.Fatalf("%+v", comp)
			}
			if (comp.FeatureMap == nil) != (c.wantMap == nil) || (c.wantMap != nil && *comp.FeatureMap != *c.wantMap) {
				t.Fatalf("map: %+v", comp.FeatureMap)
			}
			if c.reason != "" && !aa13Has(got.Reasons, c.reason) {
				t.Fatalf("reasons: %v", got.Reasons)
			}
			if c.reason == "" && len(got.Reasons) != 0 {
				t.Fatalf("unexpected reasons: %v", got.Reasons)
			}
		})
	}
	// Both capabilities without any supplied evidence never derive AUTO.
	if th := aa13One(t, []Vector{aa13Temp("measured-room", 21.5)}); th.Composition.Auto.State != "Unknown" || th.Composition.FeatureMap != nil {
		t.Fatalf("AUTO inferred: %+v", th.Composition)
	}
}

// A row that failed Validate cannot enable or contradict a capability.
func TestAA13CapabilityOnlyFromValidatedRows(t *testing.T) {
	good := aa13Temp("measured-room", 21.5)
	good.Cooling = Support{State: "Unknown"}
	bad := aa13Temp("heating-manual-target", 20)
	bad.Evidence.Confidence = "Hypothesis"
	th := aa13One(t, []Vector{good, bad}, aa13Evidence())
	if th.Composition.Cool.State != "Unknown" || th.Composition.Auto.State != "Unknown" || th.Composition.FeatureMap != nil {
		t.Fatalf("withheld row supplied cooling: %+v", th.Composition)
	}
	denial := aa13Temp("heating-manual-target", 20)
	denial.Evidence.Confidence = "Hypothesis"
	denial.Heating, denial.Cooling = Support{"No", "denial"}, Support{"No", "denial"}
	if th := aa13One(t, []Vector{aa13Temp("measured-room", 21.5), denial}, aa13Evidence()); th.Composition.Heat.State != "Yes" || th.Composition.Cool.State != "Yes" || th.Temperatures.MeasuredRoom == nil {
		t.Fatalf("withheld row poisoned sibling: %+v", th.Composition)
	}
}

func TestAA13ContradictoryCapabilityWithholdsDependents(t *testing.T) {
	target := aa13Temp("cooling-manual-target", 25)
	denial := aa13Temp("measured-room", 21)
	denial.Cooling = Support{"No", "synthetic-no-cool"}
	auto := aa13Mode("native-auto")
	th := aa13One(t, []Vector{target, denial, auto}, aa13Evidence())
	if th.Composition.Cool.State != "Unknown" || !reflect.DeepEqual(th.Composition.Cool.References, []string{"synthetic-cool", "synthetic-no-cool"}) || !aa13Has(th.Reasons, "contradictory cooling evidence") {
		t.Fatalf("%+v", th)
	}
	aa13Reason(t, th, target.Key, "target role lacks evidenced feature")
	aa13Reason(t, th, auto.Key, "target enum lacks evidenced feature")
	if th.Temperatures.MeasuredRoom == nil || th.Temperatures.CoolingManual != nil || th.SystemMode != nil {
		t.Fatalf("%+v", th)
	}
	heatDenial := aa13Temp("measured-room", 21)
	heatDenial.Heating = Support{"No", "synthetic-no-heat"}
	heatTarget := aa13Temp("heating-manual-target", 20)
	th = aa13One(t, []Vector{heatTarget, heatDenial}, aa13Evidence())
	if !aa13Has(th.Reasons, "contradictory heating evidence") || th.Temperatures.HeatingManual != nil {
		t.Fatalf("%+v", th)
	}
	aa13Reason(t, th, heatTarget.Key, "target role lacks evidenced feature")
}

func TestAA13SystemModeClosedEnum(t *testing.T) {
	for _, c := range []struct {
		code, target string
		wire         uint8
	}{{"native-off", "OFF", 0}, {"native-auto", "AUTO", 1}, {"native-cool", "COOL", 3}, {"native-heat", "HEAT", 4}} {
		t.Run(c.target, func(t *testing.T) {
			th := aa13One(t, []Vector{aa13Mode(c.code)}, aa13Evidence())
			if m := th.SystemMode; m == nil || m.Target != c.target || m.Wire != c.wire || m.NativeCode != c.code {
				t.Fatalf("%+v", m)
			}
		})
	}
	for _, target := range []string{"EMERGENCY_HEAT", "PRECOOLING", "FAN_ONLY", "DRY", "SLEEP", "heat", "Auto", "2", "", " HEAT"} {
		t.Run("unsupported/"+target, func(t *testing.T) {
			v, ev := aa13Mode("native-heat"), aa13Evidence()
			ev.Modes[1].Target = target
			th := aa13One(t, []Vector{v}, ev)
			if th.SystemMode != nil {
				t.Fatalf("fabricated: %+v", th.SystemMode)
			}
			aa13Reason(t, th, v.Key, "unsupported target enum")
		})
	}
	cases := []struct {
		name, reason string
		change       func(*TargetEvidence161)
	}{
		{"no-binding", "no explicit enum evidence", func(e *TargetEvidence161) { e.Modes = nil }},
		{"other-label-only", "no explicit enum evidence", func(e *TargetEvidence161) { e.Modes = e.Modes[2:] }},
		{"duplicate", "duplicate enum evidence", func(e *TargetEvidence161) { e.Modes = append(e.Modes, e.Modes[1]) }},
		{"reference", "incomplete enum evidence", func(e *TargetEvidence161) { e.Modes[1].Reference = "" }},
		{"loss-unreviewed", "incomplete enum evidence", func(e *TargetEvidence161) { e.Modes[1].Loss = nil }},
		{"loss-blank", "incomplete enum evidence", func(e *TargetEvidence161) { e.Modes[1].Loss = []string{" "} }},
		{"requirement-swap", "target enum contradicts source capability requirement", func(e *TargetEvidence161) { e.Modes[1].Target = "COOL" }},
		{"requirement-off", "target enum contradicts source capability requirement", func(e *TargetEvidence161) { e.Modes[1].Target = "OFF" }},
		{"requirement-auto", "target enum contradicts source capability requirement", func(e *TargetEvidence161) { e.Modes[1].Target = "AUTO" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, ev := aa13Mode("native-heat"), aa13Evidence()
			c.change(&ev)
			th := aa13One(t, []Vector{v}, ev)
			if th.SystemMode != nil {
				t.Fatalf("%+v", th.SystemMode)
			}
			aa13Reason(t, th, v.Key, c.reason)
		})
	}
	// A label reading like an enum supplies nothing without its evidence.
	th := aa13One(t, []Vector{aa13Mode("native-heat")})
	aa13Reason(t, th, aa13Mode("native-heat").Key, "no explicit enum evidence")
}

// Heating-only evidence must not reach AUTO by either guard: the feature guard
// alone (source requirement agrees) and the requirement guard alone (feature Yes).
func TestAA13AutoModeNeedsEvidencedFeature(t *testing.T) {
	v, ev := aa13Mode("native-auto"), aa13Evidence()
	ev.Auto = Support{"No", "synthetic-no-auto"}
	th := aa13One(t, []Vector{v}, ev)
	if th.SystemMode != nil {
		t.Fatalf("%+v", th.SystemMode)
	}
	aa13Reason(t, th, v.Key, "target enum lacks evidenced feature")

	ev = aa13Evidence()
	ev.Auto = Support{State: "Unknown"}
	th = aa13One(t, []Vector{v}, ev)
	aa13Reason(t, th, v.Key, "target enum lacks evidenced feature")

	// Heating-only source: a heating entry bound to AUTO, with a supplied AUTO claim.
	heatOnly := aa13Mode("native-heat")
	heatOnly.Cooling = Support{"No", "synthetic-no-cool"}
	heatOnly.Codebook = heatOnly.Codebook[:2]
	ev = aa13Evidence()
	ev.Modes[1].Target = "AUTO"
	th = aa13One(t, []Vector{heatOnly}, ev)
	if th.SystemMode != nil || th.Composition.Auto.State != "Unknown" || th.Composition.FeatureMap != nil {
		t.Fatalf("%+v", th)
	}
	aa13Reason(t, th, heatOnly.Key, "target enum contradicts source capability requirement")

	// Fully evidenced AUTO feature cannot rescue an entry whose requirement disagrees.
	mismatch := aa13Mode("native-heat")
	ev = aa13Evidence()
	ev.Modes[1].Target = "AUTO"
	th = aa13One(t, []Vector{mismatch}, ev)
	if th.Composition.Auto.State != "Yes" || th.SystemMode != nil {
		t.Fatalf("%+v", th)
	}
	aa13Reason(t, th, mismatch.Key, "target enum contradicts source capability requirement")

	// COOL and HEAT need their own feature; each guard is separately reachable.
	coolDenial := aa13Temp("measured-room", 21)
	coolDenial.Cooling = Support{"No", "synthetic-no-cool"}
	cool := aa13Mode("native-cool")
	th = aa13One(t, []Vector{cool, coolDenial}, aa13Evidence())
	aa13Reason(t, th, cool.Key, "target enum lacks evidenced feature")
	heatDenial := aa13Temp("measured-room", 21)
	heatDenial.Heating = Support{"No", "synthetic-no-heat"}
	heat := aa13Mode("native-heat")
	th = aa13One(t, []Vector{heat, heatDenial}, aa13Evidence())
	aa13Reason(t, th, heat.Key, "target enum lacks evidenced feature")
}

// Fields: Domain, Owner, Profile, Scope, Fact, Direction, Target. Each case varies
// one field with the others valid. Catches: a field dropped from the subject join
// or from the retained key, and a removed domain, direction or completeness check.
func TestAA13EveryKeyAxisIndependently(t *testing.T) {
	joined := []struct {
		name  string
		field func(*Key) *string
		other string
		want  Subject161
	}{
		{"Owner", func(k *Key) *string { return &k.Owner }, "synthetic-other", Subject161{"synthetic-other", "family-a/profile-a", "room-a"}},
		{"Profile", func(k *Key) *string { return &k.Profile }, "family-b/profile-b", Subject161{"synthetic-native", "family-b/profile-b", "room-a"}},
		{"Scope", func(k *Key) *string { return &k.Scope }, "room-b", Subject161{"synthetic-native", "family-a/profile-a", "room-b"}},
	}
	for _, axis := range joined {
		t.Run(axis.name+"/changed", func(t *testing.T) {
			a, b := aa13Temp("measured-room", 21.5), aa13Temp("measured-room", 19.5)
			*axis.field(&b.Key) = axis.other
			r := EncodeThermostat161([]Vector{a, b}, nil)
			if len(r.Thermostats) != 2 {
				t.Fatalf("subjects joined: %+v", r.Thermostats)
			}
			seen := 0
			for _, th := range r.Thermostats {
				s := th.Temperatures.MeasuredRoom
				switch {
				case s == nil || len(th.Withheld) != 0:
					t.Fatalf("sibling withheld: %+v", th)
				case th.Subject == aa13Subject && s.Wire == 2150 && s.Source.Key == a.Key:
					seen++
				case th.Subject == axis.want && s.Wire == 1950 && s.Source.Key == b.Key:
					seen++
				}
			}
			if seen != 2 {
				t.Fatalf("wrong join: %+v", r.Thermostats)
			}
		})
	}
	for _, axis := range []struct {
		name  string
		field func(*Key) *string
		other string
	}{
		{"Fact", func(k *Key) *string { return &k.Fact }, "other-fact"},
		{"Target", func(k *Key) *string { return &k.Target }, "other-slot"},
	} {
		t.Run(axis.name+"/changed-role", func(t *testing.T) {
			a, b := aa13Temp("measured-room", 21.5), aa13Temp("measured-room", 19.5)
			b.Role = "heating-manual-target"
			b.Key = a.Key
			*axis.field(&b.Key) = axis.other
			th := aa13One(t, []Vector{a, b})
			if s := th.Temperatures.MeasuredRoom; s == nil || s.Source.Key != a.Key || s.Wire != 2150 {
				t.Fatalf("%+v", s)
			}
			if s := th.Temperatures.HeatingManual; s == nil || s.Source.Key != b.Key || s.Wire != 1950 {
				t.Fatalf("%+v", s)
			}
		})
		t.Run(axis.name+"/same-role-is-ambiguous", func(t *testing.T) {
			a, b := aa13Temp("measured-room", 21.5), aa13Temp("measured-room", 19.5)
			*axis.field(&b.Key) = axis.other
			th := aa13One(t, []Vector{a, b})
			aa13Empty(t, th)
			aa13Reason(t, th, a.Key, "ambiguous role within thermostat")
			aa13Reason(t, th, b.Key, "ambiguous role within thermostat")
		})
	}
	for _, axis := range []struct {
		name  string
		field func(*Key) *string
		other string
	}{
		{"Domain", func(k *Key) *string { return &k.Domain }, "PV"},
		{"Direction", func(k *Key) *string { return &k.Direction }, "Matter-to-native operation"},
	} {
		t.Run(axis.name+"/changed", func(t *testing.T) {
			a, b := aa13Temp("measured-room", 21.5), aa13Temp("heating-manual-target", 19.5)
			*axis.field(&b.Key) = axis.other
			r := EncodeThermostat161([]Vector{a, b}, nil)
			if len(r.Validation.Rows) != 2 || len(r.Thermostats) != 1 {
				t.Fatalf("%+v", r)
			}
			th := r.Thermostats[0]
			if th.Temperatures.MeasuredRoom == nil || th.Temperatures.HeatingManual != nil || len(th.Withheld) != 0 {
				t.Fatalf("rejected row joined thermostat: %+v", th)
			}
			if lone := EncodeThermostat161([]Vector{b}, nil); len(lone.Thermostats) != 0 || len(lone.Validation.Rows) != 1 {
				t.Fatalf("rejected row invented a thermostat: %+v", lone)
			}
		})
	}
	// Each field missing withholds that row while its valid sibling stays encoded.
	for _, axis := range []struct {
		name  string
		field func(*Key) *string
	}{
		{"Domain", func(k *Key) *string { return &k.Domain }},
		{"Owner", func(k *Key) *string { return &k.Owner }},
		{"Profile", func(k *Key) *string { return &k.Profile }},
		{"Scope", func(k *Key) *string { return &k.Scope }},
		{"Fact", func(k *Key) *string { return &k.Fact }},
		{"Direction", func(k *Key) *string { return &k.Direction }},
		{"Target", func(k *Key) *string { return &k.Target }},
	} {
		t.Run(axis.name+"/missing", func(t *testing.T) {
			good, bad := aa13Temp("measured-room", 21.5), aa13Temp("heating-manual-target", 19.5)
			*axis.field(&bad.Key) = ""
			r := EncodeThermostat161([]Vector{good, bad}, nil)
			if len(r.Validation.Rows) != 2 {
				t.Fatalf("native row dropped: %+v", r.Validation.Rows)
			}
			found := false
			for _, th := range r.Thermostats {
				if th.Temperatures.HeatingManual != nil {
					t.Fatalf("incomplete key encoded: %+v", th)
				}
				if th.Subject == aa13Subject && th.Temperatures.MeasuredRoom != nil && th.Temperatures.MeasuredRoom.Source.Key == good.Key {
					found = true
				}
			}
			if !found {
				t.Fatalf("valid sibling lost: %+v", r.Thermostats)
			}
		})
	}
	// A duplicate full key is withheld with both raw rows retained.
	d := aa13Temp("measured-room", 21.5)
	r := EncodeThermostat161([]Vector{d, d}, nil)
	if len(r.Validation.Rows) != 2 || r.Thermostats[0].Temperatures.MeasuredRoom != nil || len(r.Thermostats[0].Withheld) != 2 {
		t.Fatalf("%+v", r)
	}
}

// Each axis varied alone from a valid control. Catches: a confidence or reference
// check removed upstream or an axis not retained in the DTO.
func TestAA13ConfidenceQualificationAndReferenceAxes(t *testing.T) {
	control := aa13Temp("heating-manual-target", 20)
	if s := aa13One(t, []Vector{control}).Temperatures.HeatingManual; s == nil {
		t.Fatal("valid control withheld")
	}
	for _, c := range []struct {
		name   string
		change func(*Vector)
	}{
		{"confidence-hypothesis", func(v *Vector) { v.Evidence.Confidence = "Hypothesis" }},
		{"confidence-unknown", func(v *Vector) { v.Evidence.Confidence = "Unknown" }},
		{"confidence-invalid", func(v *Vector) { v.Evidence.Confidence = "certain" }},
		{"qualification-missing", func(v *Vector) { v.Evidence.Qualification = "" }},
		{"native-contract-missing", func(v *Vector) { v.Evidence.NativeContract = "" }},
		{"source-reference-missing", func(v *Vector) { v.Evidence.SourceReference = "" }},
		{"heating-reference-missing", func(v *Vector) { v.Heating.Reference = "" }},
		{"cooling-reference-missing", func(v *Vector) { v.Cooling.Reference = "" }},
		{"bounds-reference-missing", func(v *Vector) { v.Bounds = &Constraints{10, 30, ""} }},
		{"raw-missing", func(v *Vector) { v.Raw = nil }},
		{"loss-unreviewed", func(v *Vector) { v.Loss = nil }},
	} {
		t.Run(c.name, func(t *testing.T) {
			v := control
			c.change(&v)
			th := aa13One(t, []Vector{v})
			aa13Empty(t, th)
			if len(th.Withheld) != 1 || len(th.Withheld[0].Reasons) == 0 {
				t.Fatalf("%+v", th.Withheld)
			}
		})
	}
	mode := aa13Mode("native-heat")
	mode.Codebook[1].Reference = ""
	if th := aa13One(t, []Vector{mode}, aa13Evidence()); th.SystemMode != nil {
		t.Fatalf("codebook reference ignored: %+v", th.SystemMode)
	}
	for _, c := range []struct {
		name   string
		change func(*Vector)
		check  func(Provenance161) bool
	}{
		{"qualification", func(v *Vector) { v.Evidence.Qualification = "synthetic-qualified-assertion" }, func(p Provenance161) bool { return p.Evidence.Qualification == "synthetic-qualified-assertion" }},
		{"native-contract", func(v *Vector) { v.Evidence.NativeContract = "other-contract" }, func(p Provenance161) bool { return p.Evidence.NativeContract == "other-contract" }},
		{"source-reference", func(v *Vector) { v.Evidence.SourceReference = "other-source" }, func(p Provenance161) bool { return p.Evidence.SourceReference == "other-source" }},
		{"confidence", func(v *Vector) {}, func(p Provenance161) bool { return p.Evidence.Confidence == "Proven" }},
	} {
		t.Run("retained/"+c.name, func(t *testing.T) {
			v := control
			c.change(&v)
			r := EncodeThermostat161([]Vector{v}, nil)
			s := r.Thermostats[0].Temperatures.HeatingManual
			if s == nil || !c.check(s.Source) {
				t.Fatalf("%+v", s)
			}
			if r.Validation.TargetClosure != "Unknown" || r.Validation.SemanticMapping != "Unknown" || r.Validation.LibrarySupport != "Unknown" || r.Validation.ProductionAdmitted {
				t.Fatalf("qualification upgraded outcomes: %+v", r.Validation)
			}
		})
	}
	v := control
	v.Heating.Reference = "other-heat"
	if got := aa13One(t, []Vector{v}).Composition.Heat.References; !reflect.DeepEqual(got, []string{"other-heat"}) {
		t.Fatalf("heating reference: %v", got)
	}
	v = control
	v.Cooling.Reference = "other-cool"
	if got := aa13One(t, []Vector{v}).Composition.Cool.References; !reflect.DeepEqual(got, []string{"other-cool"}) {
		t.Fatalf("cooling reference: %v", got)
	}
}

// No input can reach a write, hold, schedule or optional surface.
func TestAA13UnsupportedSurfacesStayWithheld(t *testing.T) {
	th := aa13One(t, []Vector{aa13Temp("measured-room", 21.5), aa13Mode("native-auto")}, aa13Evidence())
	names := []string{
		"ControlSequenceOfOperation", "RunningState", "demand", "events", "hold", "local-temperature-not-exposed",
		"occupancy", "presets", "schedules", "setpoint-limits", "setpoint-writes", "system-mode-writes", "write-commands",
	}
	if len(th.Surfaces) != len(names) {
		t.Fatalf("%+v", th.Surfaces)
	}
	for i, name := range names {
		if th.Surfaces[i] != (Surface161{name, "withheld", "Unknown"}) {
			t.Fatalf("%d: %+v", i, th.Surfaces[i])
		}
	}
	// FeatureMap carries only HEAT, COOL and AUTO whatever else is supplied.
	if m := th.Composition.FeatureMap; m == nil || *m&^uint32(0x23) != 0 {
		t.Fatalf("%+v", th.Composition)
	}
	command := aa13Temp("heating-manual-target", 20)
	command.Kind = "command"
	got := aa13One(t, []Vector{command})
	aa13Empty(t, got)
	aa13Reason(t, got, command.Key, "unknown observation kind")
	write := aa13Temp("heating-manual-target", 20)
	write.Key.Direction = "Matter-to-native operation"
	if r := EncodeThermostat161([]Vector{write}, []TargetEvidence161{aa13Evidence()}); len(r.Thermostats) != 0 || len(r.Validation.Rows) != 1 {
		t.Fatalf("operation row encoded: %+v", r)
	}
	// Source bounds are retained but never become target limits.
	bounded := aa13Temp("measured-room", 21.5)
	bounded.Bounds = &Constraints{18, 25, "synthetic-bounds"}
	r := EncodeThermostat161([]Vector{bounded}, nil)
	if r.Target.MinWire != -27315 || r.Target.MaxWire != 32767 || r.Validation.Rows[0].Source.Bounds.MinC != 18 {
		t.Fatalf("%+v", r)
	}
	outside := aa13Temp("measured-room", 26)
	outside.Bounds = &Constraints{18, 25, "synthetic-bounds"}
	got = aa13One(t, []Vector{outside})
	aa13Empty(t, got)
	aa13Reason(t, got, outside.Key, "outside supplied source bounds")
}

func TestAA13EvidenceScopedToOneSubject(t *testing.T) {
	v := aa13Mode("native-heat")
	other := aa13Evidence()
	other.Subject.Scope = "room-b"
	r := EncodeThermostat161([]Vector{v}, []TargetEvidence161{other})
	if r.Thermostats[0].SystemMode != nil || r.Thermostats[0].Composition.Auto.State != "Unknown" {
		t.Fatalf("evidence crossed subjects: %+v", r.Thermostats[0])
	}
	if !reflect.DeepEqual(r.UnmatchedEvidence, []Subject161{other.Subject}) {
		t.Fatalf("%+v", r.UnmatchedEvidence)
	}
	for _, change := range []func(*Subject161){
		func(s *Subject161) { s.Owner = "other" },
		func(s *Subject161) { s.Profile = "other" },
		func(s *Subject161) { s.Scope = "other" },
	} {
		ev := aa13Evidence()
		change(&ev.Subject)
		if th := aa13One(t, []Vector{v}, ev); th.SystemMode != nil {
			t.Fatalf("evidence crossed subject axis: %+v", ev.Subject)
		}
	}
	th := aa13One(t, []Vector{v}, aa13Evidence(), aa13Evidence())
	if th.SystemMode != nil || th.Composition.Auto.State != "Unknown" || !aa13Has(th.Reasons, "duplicate target evidence") {
		t.Fatalf("duplicate evidence accepted: %+v", th)
	}
}

// Literal order over every subject field; a map-ordered result fails this.
func TestAA13SubjectOrdering(t *testing.T) {
	want := []Subject161{
		{"owner-a", "profile-a", "scope-a"}, {"owner-a", "profile-a", "scope-b"}, {"owner-a", "profile-b", "scope-a"},
		{"owner-b", "profile-a", "scope-a"}, {"owner-b", "profile-a", "scope-b"}, {"owner-b", "profile-b", "scope-a"},
		{"owner-c", "profile-a", "scope-a"}, {"owner-c", "profile-b", "scope-b"},
	}
	input, unmatched := []Vector{}, []TargetEvidence161{}
	for i := len(want) - 1; i >= 0; i-- {
		v := aa13Temp("measured-room", 21.5)
		v.Key.Owner, v.Key.Profile, v.Key.Scope = want[i].Owner, want[i].Profile, want[i].Scope
		input = append(input, v)
		unmatched = append(unmatched, TargetEvidence161{Subject: Subject161{"absent-" + want[i].Owner, want[i].Profile, want[i].Scope}})
	}
	r := EncodeThermostat161(input, unmatched)
	if len(r.Thermostats) != len(want) || len(r.UnmatchedEvidence) != len(want) {
		t.Fatalf("%+v", r)
	}
	for i, s := range want {
		if r.Thermostats[i].Subject != s {
			t.Fatalf("%d: %+v", i, r.Thermostats[i].Subject)
		}
		if r.UnmatchedEvidence[i] != (Subject161{"absent-" + s.Owner, s.Profile, s.Scope}) {
			t.Fatalf("%d: %+v", i, r.UnmatchedEvidence[i])
		}
	}
}

func TestAA13PartialDetachedAndPermutationStable(t *testing.T) {
	good, heat, unavailable := aa13Temp("measured-room", 21.5), aa13Temp("heating-manual-target", 20), aa13Temp("cooling-manual-target", 25)
	unavailable.Celsius = nil
	mode := aa13Mode("native-heat")
	other := aa13Temp("measured-room", 19)
	other.Key.Scope = "room-b"
	evidence := []TargetEvidence161{aa13Evidence()}
	want := EncodeThermostat161([]Vector{good, heat, unavailable, mode, other}, evidence)
	for _, p := range [][]Vector{{other, mode, unavailable, heat, good}, {unavailable, good, other, heat, mode}, {mode, other, good, unavailable, heat}} {
		if !reflect.DeepEqual(EncodeThermostat161(p, evidence), want) {
			t.Fatal("row permutation changed result")
		}
	}
	shuffled := aa13Evidence()
	shuffled.Modes[0], shuffled.Modes[3] = shuffled.Modes[3], shuffled.Modes[0]
	if !reflect.DeepEqual(EncodeThermostat161([]Vector{good, heat, unavailable, mode, other}, []TargetEvidence161{shuffled}), want) {
		t.Fatal("binding permutation changed result")
	}
	th := want.Thermostats[0]
	if th.Temperatures.MeasuredRoom == nil || th.Temperatures.HeatingManual == nil || th.SystemMode == nil || th.Temperatures.CoolingManual != nil {
		t.Fatalf("partial loss erased sibling: %+v", th)
	}
	aa13Reason(t, th, unavailable.Key, "temperature unavailable or nonfinite")
	if len(want.Validation.Rows) != 5 {
		t.Fatalf("native evidence dropped: %d", len(want.Validation.Rows))
	}
	for _, e := range want.Validation.Rows {
		if e.Source.Key == unavailable.Key && !reflect.DeepEqual(e.Source.Raw, []byte("cooling-manual-target")) {
			t.Fatal("withheld raw evidence lost")
		}
	}

	// Mutating an input or a returned DTO reaches nothing else.
	v, ev := aa13Mode("native-auto"), aa13Evidence()
	temp := aa13Temp("measured-room", 21.5)
	r := EncodeThermostat161([]Vector{v, temp}, []TargetEvidence161{ev})
	v.Raw[0], v.Loss[0], v.Codebook[3].Loss[0], ev.Modes[3].Loss[0] = 0, "changed", "changed", "changed"
	temp.Raw[0], temp.Loss[0] = 0, "changed"
	got := r.Thermostats[0]
	m, s := got.SystemMode, got.Temperatures.MeasuredRoom
	if m.Source.Raw[0] != 'm' || m.Source.Loss[0] != "loss-mode" || m.CodebookLoss[0] != "book-auto-loss" || m.BindingLoss[0] != "bind-auto-loss" || s.Source.Raw[0] != 'm' || s.Source.Loss[0] != "loss-measured-room" {
		t.Fatalf("result aliases input: %+v %+v", m, s)
	}
	m.Source.Raw[0], m.Source.Loss[0], m.CodebookLoss[0], s.Source.Raw[0], s.Source.Loss[0] = 0, "changed", "changed", 0, "changed"
	for _, e := range r.Validation.Rows {
		if e.Source.Raw[0] == 0 || e.Source.Loss[0] == "changed" {
			t.Fatalf("DTO aliases retained validation: %+v", e.Source)
		}
		for _, entry := range e.Source.Codebook {
			if len(entry.Loss) > 0 && entry.Loss[0] == "changed" {
				t.Fatal("DTO aliases retained codebook")
			}
		}
	}
	if !reflect.DeepEqual(EncodeThermostat161(nil, nil).Thermostats, []Thermostat161{}) {
		t.Fatal("empty input invented a thermostat")
	}
	empty := EncodeThermostat161(nil, nil).Validation
	if empty.ProductionAdmitted || empty.TargetClosure != "Unknown" {
		t.Fatalf("%+v", empty)
	}
}
