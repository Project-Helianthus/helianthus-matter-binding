package epm

import (
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func aa15Int(v int64) *int64 { return &v }

func aa15Row(fact, unit, decimal string) Input {
	return Input{
		Key:          Key{Owner: "synthetic-native", Profile: "meter-family", Source: "source-a", Scope: "phase", Phase: "a", Fact: fact},
		Availability: "available", Decimal: decimal, Unit: unit, OutputUnit: unit, OutputScale: -3,
		Evidence: []byte("synthetic " + fact), Accuracy: []byte("opaque " + fact),
	}
}

func aa15Power() Input   { return aa15Row("active-power", "W", "-250") }
func aa15Current() Input { return aa15Row("active-current", "A", "1.5") }
func aa15Voltage() Input { return aa15Row("voltage", "V", "230") }

func aa15Accuracy(kind string) Accuracy161 {
	return Accuracy161{
		MeasurementType: kind, MinMeasured: aa15Int(-1000), MaxMeasured: aa15Int(1000), Fixed: aa15Int(50),
		Reference: "synthetic:accuracy-" + kind,
	}
}

// aa15Facts declares only ActivePower and ActiveCurrent; each test adds the rest.
func aa15Facts() Facts161 {
	return Facts161{
		PowerMode:        PowerMode161{"AC", "synthetic:power-mode"},
		MeasurementTypes: Count161{2, "synthetic:count"},
		Accuracy:         []Accuracy161{aa15Accuracy("ActivePower"), aa15Accuracy("ActiveCurrent")},
		Bindings: []Binding161{
			{ActivePower161, aa15Power().Key, "synthetic:power-binding"},
			{ActiveCurrent161, aa15Current().Key, "synthetic:current-binding"},
		},
	}
}

func aa15FactsWithVoltage() Facts161 {
	f := aa15Facts()
	f.MeasurementTypes.Value = 3
	f.Accuracy = append(f.Accuracy, aa15Accuracy("Voltage"))
	f.Bindings = append(f.Bindings, Binding161{Voltage161, aa15Voltage().Key, "synthetic:voltage-binding"})
	return f
}

func aa15PowerOnlyFacts() Facts161 {
	f := aa15Facts()
	f.MeasurementTypes.Value = 1
	f.Accuracy, f.Bindings = f.Accuracy[:1], f.Bindings[:1]
	return f
}

func aa15Slot(r Result161, slot Slot161) *Attribute161 {
	switch slot {
	case ActivePower161:
		return r.ActivePower
	case ActiveCurrent161:
		return r.ActiveCurrent
	case Voltage161:
		return r.Voltage
	}
	return nil
}

func aa15OnlyWithheld(t *testing.T, r Result161, slot Slot161, key Key, reason string) {
	t.Helper()
	if a := aa15Slot(r, slot); a != nil {
		t.Fatalf("%s kept a value: %+v", slot, a)
	}
	want := []Withheld161{{slot, key, reason}}
	if !reflect.DeepEqual(r.Withheld, want) {
		t.Fatalf("withheld %+v; want %+v", r.Withheld, want)
	}
}

// Catches: changed identity, unit or evidence constants, a dropped DTO field and
// any upgraded outcome. Expected values are literal.
func TestAA15ExactDTO(t *testing.T) {
	rows := []Input{aa15Power(), aa15Current(), aa15Voltage()}
	got := EncodePower161(rows, aa15FactsWithVoltage())
	target := Target161{"1.6.1", "ElectricalPowerMeasurement", 0x0090, 3, 0x0005, "mW", "mA", "mV", -4611686018427387904, 4611686018427387904, "Proven", "Hypothesis"}
	if got.Target != target {
		t.Fatalf("target: %+v", got.Target)
	}
	power := &Attribute161{"ActivePower", "mW", aa15Int(-250000), false, aa15Power(), "synthetic:power-binding"}
	current := &Attribute161{"ActiveCurrent", "mA", aa15Int(1500), false, aa15Current(), "synthetic:current-binding"}
	voltage := &Attribute161{"Voltage", "mV", aa15Int(230000), false, aa15Voltage(), "synthetic:voltage-binding"}
	for _, c := range []struct{ got, want *Attribute161 }{{got.ActivePower, power}, {got.ActiveCurrent, current}, {got.Voltage, voltage}} {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Fatalf("attribute %+v; want %+v", c.got, c.want)
		}
	}
	metadata := &Metadata161{
		PowerMode161{"AC", "synthetic:power-mode"}, Count161{3, "synthetic:count"},
		[]Accuracy161{aa15Accuracy("ActivePower"), aa15Accuracy("ActiveCurrent"), aa15Accuracy("Voltage")},
	}
	if !reflect.DeepEqual(got.Metadata, metadata) {
		t.Fatalf("metadata %+v", got.Metadata)
	}
	if len(got.Withheld) != 0 || len(got.Reasons) != 0 {
		t.Fatalf("control withheld: %+v %+v", got.Withheld, got.Reasons)
	}
	if !reflect.DeepEqual(got.Validation, Validate(rows)) {
		t.Fatalf("validation changed: %+v", got.Validation)
	}
	if got.SemanticMapping != "Unknown" || got.TargetClosure != "Unknown" || got.LibrarySupport != "Unknown" || got.ProductionAdmitted {
		t.Fatalf("outcome upgraded: %+v", got)
	}
	names := []string{"apparent-power", "events", "feature-map", "frequency", "harmonics", "native-control", "polyphase", "power-factor", "power-quality", "reactive-power"}
	if len(got.Surfaces) != len(names) {
		t.Fatalf("%+v", got.Surfaces)
	}
	for i, name := range names {
		if got.Surfaces[i] != (Surface161{name, "withheld", "Unknown"}) {
			t.Fatalf("%d: %+v", i, got.Surfaces[i])
		}
	}
}

// Catches: a wrong multiplier on any slot, a swapped slot unit, truncation and a
// dropped Scale. Expected integers are literal and independent of the code path.
func TestAA15ExactConversion(t *testing.T) {
	for _, c := range []struct {
		name    string
		slot    Slot161
		unit    string
		decimal string
		scale   int
		want    int64
		target  string
	}{
		{"watt-zero", ActivePower161, "W", "0", 0, 0, "mW"},
		{"watt-negative-zero", ActivePower161, "W", "-0.000", 0, 0, "mW"},
		{"watt-signed", ActivePower161, "W", "-250", 0, -250000, "mW"},
		{"watt-fraction", ActivePower161, "W", "12.345", 0, 12345, "mW"},
		{"watt-smallest", ActivePower161, "W", "0.001", 0, 1, "mW"},
		{"watt-negative-smallest", ActivePower161, "W", "-0.001", 0, -1, "mW"},
		{"watt-trailing-zeroes", ActivePower161, "W", "1.2300", 0, 1230, "mW"},
		{"watt-negative-input-scale", ActivePower161, "W", "2500", -1, 250000, "mW"},
		{"watt-milli-input-scale", ActivePower161, "W", "7", -3, 7, "mW"},
		{"watt-positive-input-scale", ActivePower161, "W", "2", 3, 2000000, "mW"},
		{"watt-upper-bound", ActivePower161, "W", "4611686018427387.904", 0, 4611686018427387904, "mW"},
		{"watt-lower-bound", ActivePower161, "W", "-4611686018427387.904", 0, -4611686018427387904, "mW"},
		{"ampere-zero", ActiveCurrent161, "A", "0", 0, 0, "mA"},
		{"ampere-fraction", ActiveCurrent161, "A", "1.5", 0, 1500, "mA"},
		{"ampere-signed", ActiveCurrent161, "A", "-16", 0, -16000, "mA"},
		{"ampere-smallest", ActiveCurrent161, "A", "0.001", 0, 1, "mA"},
		{"ampere-upper-bound", ActiveCurrent161, "A", "4611686018427387.904", 0, 4611686018427387904, "mA"},
		{"ampere-lower-bound", ActiveCurrent161, "A", "-4611686018427387.904", 0, -4611686018427387904, "mA"},
		{"volt-zero", Voltage161, "V", "0", 0, 0, "mV"},
		{"volt-fraction", Voltage161, "V", "230.125", 0, 230125, "mV"},
		{"volt-input-scale", Voltage161, "V", "2301", -1, 230100, "mV"},
		{"volt-upper-bound", Voltage161, "V", "4611686018427387.904", 0, 4611686018427387904, "mV"},
		{"volt-lower-bound", Voltage161, "V", "-4611686018427387.904", 0, -4611686018427387904, "mV"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fact := map[Slot161]func() Input{ActivePower161: aa15Power, ActiveCurrent161: aa15Current, Voltage161: aa15Voltage}[c.slot]
			row := fact()
			row.Decimal, row.Scale = c.decimal, c.scale
			got := EncodePower161([]Input{row}, aa15FactsWithVoltage())
			a := aa15Slot(got, c.slot)
			if a == nil || a.Null || a.Value == nil || *a.Value != c.want || a.Unit != c.target {
				t.Fatalf("got %+v withheld %+v; want %d %s", a, got.Withheld, c.want, c.target)
			}
			if !reflect.DeepEqual(a.Source, row) {
				t.Fatalf("source %+v; want %+v", a.Source, row)
			}
		})
	}
}

// Catches: truncation, rounding, a removed range guard, wrapped arithmetic and a
// null or zero standing in for a rejected value. The current sibling must survive.
func TestAA15RejectsLossAndOverflow(t *testing.T) {
	for _, c := range []struct {
		name, decimal string
		scale         int
		reason        string
	}{
		{"finer-than-milli", "0.0001", 0, "precision-loss"},
		{"half-milli", "0.0005", 0, "precision-loss"},
		{"just-below-next-milli", "1.2349", 0, "precision-loss"},
		{"negative-finer", "-0.0005", 0, "precision-loss"},
		{"scale-finer", "1", -4, "precision-loss"},
		{"above-upper-bound", "4611686018427387.905", 0, TargetRange161},
		{"below-lower-bound", "-4611686018427387.905", 0, TargetRange161},
		{"past-int64", "9223372036854775.808", 0, "integer-overflow"},
		{"before-int64", "-9223372036854775.809", 0, "integer-overflow"},
		{"scale-overflow", "9223372036854775807", 1, "integer-overflow"},
		{"malformed", "1e3", 0, "invalid-decimal"},
		{"available-without-value", "", 0, "invalid-decimal"},
	} {
		t.Run(c.name, func(t *testing.T) {
			row := aa15Power()
			row.Decimal, row.Scale = c.decimal, c.scale
			got := EncodePower161([]Input{row, aa15Current()}, aa15Facts())
			aa15OnlyWithheld(t, got, ActivePower161, row.Key, c.reason)
			if got.ActiveCurrent == nil || *got.ActiveCurrent.Value != 1500 || got.Metadata == nil {
				t.Fatalf("sibling or metadata lost: %+v", got)
			}
		})
	}
}

// Each of Unit, OutputUnit, Scale and OutputScale varied alone. Catches: a removed
// unit or grid guard, a slot table swap and treating another grid as milli-units.
func TestAA15UnitAndScaleAxes(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(*Input)
		reason string
	}{
		{"unit-current-on-power", func(i *Input) { i.Unit, i.OutputUnit = "A", "A" }, TargetUnit161},
		{"unit-voltage-on-power", func(i *Input) { i.Unit, i.OutputUnit = "V", "V" }, TargetUnit161},
		{"unit-apparent-on-power", func(i *Input) { i.Unit, i.OutputUnit = "VA", "VA" }, TargetUnit161},
		{"unit-reactive-on-power", func(i *Input) { i.Unit, i.OutputUnit = "var", "var" }, TargetUnit161},
		{"unit-alone", func(i *Input) { i.Unit = "VA" }, "unit-unknown-or-mismatch"},
		{"unit-unknown", func(i *Input) { i.Unit, i.OutputUnit = "mW", "mW" }, "unit-unknown-or-mismatch"},
		{"unit-missing", func(i *Input) { i.Unit = "" }, "unit-unknown-or-mismatch"},
		{"output-unit-alone", func(i *Input) { i.OutputUnit = "VA" }, "unit-unknown-or-mismatch"},
		{"output-unit-target-label", func(i *Input) { i.OutputUnit = "mW" }, "unit-unknown-or-mismatch"},
		{"output-unit-missing", func(i *Input) { i.OutputUnit = "" }, "unit-unknown-or-mismatch"},
		{"output-scale-whole", func(i *Input) { i.OutputScale = 0 }, TargetScale161},
		{"output-scale-centi", func(i *Input) { i.OutputScale = -2 }, TargetScale161},
		{"output-scale-micro", func(i *Input) { i.OutputScale = -6 }, TargetScale161},
		{"output-scale-out-of-bounds", func(i *Input) { i.OutputScale = -19 }, "scale-out-of-bounds"},
		{"scale-out-of-bounds", func(i *Input) { i.Scale = 19 }, "scale-out-of-bounds"},
	} {
		t.Run(c.name, func(t *testing.T) {
			row := aa15Power()
			c.change(&row)
			aa15OnlyWithheld(t, EncodePower161([]Input{row}, aa15PowerOnlyFacts()), ActivePower161, row.Key, c.reason)
			// A null must pass the same unit and grid checks as a value.
			if row.Unit != "" && c.reason != "scale-out-of-bounds" {
				row.Availability, row.Decimal = "unavailable", ""
				aa15OnlyWithheld(t, EncodePower161([]Input{row}, aa15PowerOnlyFacts()), ActivePower161, row.Key, c.reason)
			}
		})
	}
	// The slot table on the other two slots, varied alone.
	for _, c := range []struct {
		slot Slot161
		row  func() Input
		unit string
	}{{ActiveCurrent161, aa15Current, "W"}, {Voltage161, aa15Voltage, "A"}} {
		row := c.row()
		row.Unit, row.OutputUnit = c.unit, c.unit
		rows := []Input{row, aa15Power()}
		rows = append(rows, map[Slot161]Input{ActiveCurrent161: aa15Voltage(), Voltage161: aa15Current()}[c.slot])
		aa15OnlyWithheld(t, EncodePower161(rows, aa15FactsWithVoltage()), c.slot, row.Key, TargetUnit161)
	}
}

// Catches: null turned into zero, zero turned into null, unknown availability
// turned into either, and a value silently dropped from an unavailable row.
func TestAA15AvailabilityAndNull(t *testing.T) {
	for _, c := range []struct {
		slot Slot161
		row  func() Input
	}{{ActivePower161, aa15Power}, {ActiveCurrent161, aa15Current}, {Voltage161, aa15Voltage}} {
		t.Run(string(c.slot), func(t *testing.T) {
			null := c.row()
			null.Availability, null.Decimal = "unavailable", ""
			got := EncodePower161([]Input{null}, aa15FactsWithVoltage())
			a := aa15Slot(got, c.slot)
			if a == nil || !a.Null || a.Value != nil || !reflect.DeepEqual(a.Source, null) {
				t.Fatalf("null not preserved: %+v", a)
			}
			zero := c.row()
			zero.Decimal = "0"
			a = aa15Slot(EncodePower161([]Input{zero}, aa15FactsWithVoltage()), c.slot)
			if a == nil || a.Null || a.Value == nil || *a.Value != 0 {
				t.Fatalf("known zero not preserved: %+v", a)
			}
			for _, bad := range []struct{ name, availability, decimal string }{
				{"unavailable-with-value", "unavailable", "7"},
				{"unavailable-with-zero", "unavailable", "0"},
				{"unknown", "unknown", ""},
				{"unknown-with-value", "unknown", "7"},
				{"missing", "", ""},
				{"case-changed", "Available", "7"},
			} {
				row := c.row()
				row.Availability, row.Decimal = bad.availability, bad.decimal
				got := EncodePower161([]Input{row}, aa15FactsWithVoltage())
				if aa15Slot(got, c.slot) != nil {
					t.Fatalf("%s kept a slot", bad.name)
				}
				found := false
				for _, w := range got.Withheld {
					found = found || w == Withheld161{c.slot, row.Key, "availability-unknown-or-conflicting"}
				}
				if !found {
					t.Fatalf("%s: %+v", bad.name, got.Withheld)
				}
			}
		})
	}
	t.Run("native-evidence-required", func(t *testing.T) {
		for _, evidence := range [][]byte{nil, {}} {
			row := aa15Power()
			row.Evidence = evidence
			aa15OnlyWithheld(t, EncodePower161([]Input{row}, aa15PowerOnlyFacts()), ActivePower161, row.Key, NativeMissing161)
			row.Availability, row.Decimal = "unavailable", ""
			aa15OnlyWithheld(t, EncodePower161([]Input{row}, aa15PowerOnlyFacts()), ActivePower161, row.Key, NativeMissing161)
		}
	})
}

// Each field of the join key varied alone. A sibling that differs in one field
// keeps its own row and value; dropping that field from the join selects the wrong
// sibling in one direction or fills the slot from a row that was never bound.
func TestAA15KeyAxes(t *testing.T) {
	for _, axis := range axes {
		t.Run(axis.name, func(t *testing.T) {
			a, b := aa15Power(), aa15Power()
			axis.set(&b.Key, "other")
			b.Decimal = "7"
			for _, c := range []struct {
				key  Key
				want int64
			}{{a.Key, -250000}, {b.Key, 7000}} {
				f := aa15PowerOnlyFacts()
				f.Bindings[0].Key = c.key
				for _, rows := range [][]Input{{a, b}, {b, a}} {
					got := EncodePower161(rows, f)
					if got.ActivePower == nil || *got.ActivePower.Value != c.want || got.ActivePower.Source.Key != c.key || len(got.Validation) != 2 {
						t.Fatalf("join on %+v: %+v", c.key, got.ActivePower)
					}
				}
			}
			// The row alone, and then the binding alone, no longer match.
			f := aa15PowerOnlyFacts()
			aa15OnlyWithheld(t, EncodePower161([]Input{b}, f), ActivePower161, f.Bindings[0].Key, RowMissing161)
			f.Bindings[0].Key = b.Key
			aa15OnlyWithheld(t, EncodePower161([]Input{a}, f), ActivePower161, b.Key, RowMissing161)
		})
	}
}

// Catches: validating rows one at a time, or picking one of a conflicting pair.
// Both identical and differing duplicates of the full key withhold the slot.
func TestAA15DuplicateFullKeys(t *testing.T) {
	other := aa15Power()
	other.Decimal = "9"
	for name, rows := range map[string][]Input{
		"identical": {aa15Power(), aa15Power()},
		"differing": {aa15Power(), other},
		"reversed":  {other, aa15Power()},
	} {
		t.Run(name, func(t *testing.T) {
			got := EncodePower161(append(rows, aa15Current()), aa15Facts())
			aa15OnlyWithheld(t, got, ActivePower161, aa15Power().Key, "duplicate-identity")
			if got.ActiveCurrent == nil || *got.ActiveCurrent.Value != 1500 {
				t.Fatalf("sibling lost: %+v", got.ActiveCurrent)
			}
			if len(got.Validation) != 3 {
				t.Fatalf("rows lost: %+v", got.Validation)
			}
		})
	}
}

// Catches: a value encoded while a binding is missing, unevidenced, unknown or in
// conflict, and any slot filled from the text of a Fact label.
func TestAA15BindingsAreSuppliedFactsOnly(t *testing.T) {
	rows := []Input{aa15Power(), aa15Current()}
	check := func(t *testing.T, f Facts161, reasons ...string) Result161 {
		t.Helper()
		got := EncodePower161(rows, f)
		if !reflect.DeepEqual(got.Reasons, reasons) || got.Metadata != nil || got.ActivePower != nil || got.ActiveCurrent != nil || got.Voltage != nil {
			t.Fatalf("got %+v; want reasons %v and nothing encoded", got, reasons)
		}
		if len(got.Validation) != 2 {
			t.Fatalf("rows lost: %+v", got.Validation)
		}
		return got
	}
	t.Run("active-power-undeclared", func(t *testing.T) {
		f := aa15Facts()
		f.Bindings = f.Bindings[1:]
		got := check(t, f, ActivePowerUndeclared161)
		if !reflect.DeepEqual(got.Withheld, []Withheld161{{ActiveCurrent161, aa15Current().Key, FactsIncomplete161}}) {
			t.Fatalf("%+v", got.Withheld)
		}
	})
	t.Run("no-bindings", func(t *testing.T) {
		f := aa15Facts()
		f.Bindings = nil
		if got := check(t, f, ActivePowerUndeclared161); len(got.Withheld) != 0 {
			t.Fatalf("%+v", got.Withheld)
		}
	})
	for _, reference := range []string{"", " ", "synthetic:", "synthetic: ", "native:power", "SYNTHETIC:power"} {
		t.Run("binding-reference/"+reference, func(t *testing.T) {
			f := aa15Facts()
			f.Bindings[0].Reference = reference
			check(t, f, BindingInvalid161)
		})
	}
	t.Run("unknown-slot", func(t *testing.T) {
		f := aa15Facts()
		f.Bindings = append(f.Bindings, Binding161{"ReactivePower", Key{Owner: "o", Profile: "p", Source: "s", Scope: "c", Phase: "a", Fact: "reactive"}, "synthetic:x"})
		check(t, f, BindingInvalid161, AccuracySlotMissing161)
	})
	t.Run("slot-bound-twice", func(t *testing.T) {
		f := aa15Facts()
		f.Bindings[1].Slot = ActivePower161
		f.Accuracy = []Accuracy161{aa15Accuracy("ActivePower")}
		f.MeasurementTypes.Value = 1
		got := check(t, f, BindingConflict161)
		if !reflect.DeepEqual(got.Withheld, []Withheld161{{Slot: ActivePower161, Reason: BindingConflict161}}) {
			t.Fatalf("%+v", got.Withheld)
		}
	})
	t.Run("key-bound-twice", func(t *testing.T) {
		f := aa15Facts()
		f.Bindings[1].Key = f.Bindings[0].Key
		check(t, f, BindingConflict161)
	})
	t.Run("no-mapping-from-labels", func(t *testing.T) {
		f := aa15Facts()
		f.Bindings = f.Bindings[:1]
		got := EncodePower161(rows, f)
		if got.ActiveCurrent != nil || len(got.Withheld) != 0 || got.ActivePower == nil {
			t.Fatalf("current mapped from its label: %+v", got)
		}
		if got.Validation[0].Native.Key != aa15Current().Key || got.Validation[0].Status != "exact" {
			t.Fatalf("unbound row lost: %+v", got.Validation)
		}
		// A label naming the accepted fact grades nothing: the mapping stays Unknown.
		named := aa15Current()
		named.Key.Fact = "evse.ac.current"
		f = aa15Facts()
		f.Bindings[1].Key = named.Key
		got = EncodePower161([]Input{aa15Power(), named}, f)
		if got.ActiveCurrent == nil || got.SemanticMapping != "Unknown" || got.ProductionAdmitted {
			t.Fatalf("%+v", got)
		}
		f.Bindings[1].Key = aa15Current().Key
		if got = EncodePower161([]Input{aa15Power(), named}, f); got.ActiveCurrent != nil {
			t.Fatalf("accepted-fact label filled a slot unbound: %+v", got.ActiveCurrent)
		}
	})
	t.Run("optional-current-absent", func(t *testing.T) {
		f := aa15Facts()
		f.Bindings = f.Bindings[:1]
		f.Accuracy, f.MeasurementTypes.Value = f.Accuracy[:1], 1
		got := EncodePower161(rows[:1], f)
		if got.ActivePower == nil || got.ActiveCurrent != nil || got.Metadata == nil || len(got.Withheld) != 0 || len(got.Reasons) != 0 {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("binding-order-does-not-matter", func(t *testing.T) {
		f, g := aa15Facts(), aa15Facts()
		g.Bindings[0], g.Bindings[1] = g.Bindings[1], g.Bindings[0]
		if !reflect.DeepEqual(EncodePower161(rows, f), EncodePower161(rows, g)) {
			t.Fatal("binding order changed the result")
		}
	})
}

// Each mandatory fact varied alone from a valid control. Catches: a defaulted
// PowerMode, count or Accuracy, a count guessed from the entries, a nil bound
// defaulted to zero, a measured claim accepted and a fixture reference waived.
func TestAA15MandatoryFacts(t *testing.T) {
	rows := []Input{aa15Power(), aa15Current()}
	if got := EncodePower161(rows, aa15Facts()); got.Metadata == nil || got.ActivePower == nil || len(got.Reasons) != 0 {
		t.Fatalf("control: %+v", got)
	}
	if got := EncodePower161(rows, func() Facts161 { f := aa15Facts(); f.PowerMode.Value = "DC"; return f }()); got.Metadata == nil || got.Metadata.PowerMode.Value != "DC" {
		t.Fatalf("DC control: %+v", got)
	}
	if got := EncodePower161(rows, func() Facts161 { f := aa15Facts(); f.Accuracy[0].Fixed = aa15Int(0); return f }()); got.Metadata == nil || got.Metadata.Accuracy[0].Fixed == nil || *got.Metadata.Accuracy[0].Fixed != 0 {
		t.Fatalf("evidenced zero bound lost: %+v", got)
	}
	for _, c := range []struct {
		name   string
		change func(*Facts161)
		want   []string
	}{
		{"mode-empty", func(f *Facts161) { f.PowerMode.Value = "" }, []string{PowerModeInvalid161}},
		{"mode-unknown", func(f *Facts161) { f.PowerMode.Value = "Unknown" }, []string{PowerModeInvalid161}},
		{"mode-unknown-upper", func(f *Facts161) { f.PowerMode.Value = "UNKNOWN" }, []string{PowerModeInvalid161}},
		{"mode-unknown-lower", func(f *Facts161) { f.PowerMode.Value = "unknown" }, []string{PowerModeInvalid161}},
		{"mode-case-changed", func(f *Facts161) { f.PowerMode.Value = "ac" }, []string{PowerModeInvalid161}},
		{"mode-other", func(f *Facts161) { f.PowerMode.Value = "three-phase" }, []string{PowerModeInvalid161}},
		{"mode-unevidenced", func(f *Facts161) { f.PowerMode.Reference = "" }, []string{PowerModeInvalid161}},
		{"mode-not-fixture", func(f *Facts161) { f.PowerMode.Reference = "native:mode" }, []string{PowerModeInvalid161}},
		{"count-zero", func(f *Facts161) { f.MeasurementTypes.Value = 0 }, []string{CountInvalid161}},
		{"count-negative", func(f *Facts161) { f.MeasurementTypes.Value = -2 }, []string{CountInvalid161}},
		{"count-unevidenced", func(f *Facts161) { f.MeasurementTypes.Reference = "" }, []string{CountInvalid161}},
		{"count-not-fixture", func(f *Facts161) { f.MeasurementTypes.Reference = "native:count" }, []string{CountInvalid161}},
		{"count-too-low", func(f *Facts161) { f.MeasurementTypes.Value = 1 }, []string{AccuracyCountMismatch161}},
		{"count-too-high", func(f *Facts161) { f.MeasurementTypes.Value = 3 }, []string{AccuracyCountMismatch161}},
		{"accuracy-nil", func(f *Facts161) { f.Accuracy = nil }, []string{AccuracyAbsent161, AccuracySlotMissing161}},
		{"accuracy-empty", func(f *Facts161) { f.Accuracy = []Accuracy161{} }, []string{AccuracyAbsent161, AccuracySlotMissing161}},
		{"accuracy-type-missing", func(f *Facts161) { f.Accuracy[1].MeasurementType = "" }, []string{AccuracyMalformed161, AccuracySlotMissing161}},
		{"accuracy-type-duplicate", func(f *Facts161) { f.Accuracy[1].MeasurementType = "ActivePower" }, []string{AccuracyMalformed161, AccuracySlotMissing161}},
		{"accuracy-slot-not-covered", func(f *Facts161) { f.Accuracy[1].MeasurementType = "Voltage" }, []string{AccuracySlotMissing161}},
		{"accuracy-unevidenced", func(f *Facts161) { f.Accuracy[0].Reference = "" }, []string{AccuracyMalformed161}},
		{"accuracy-not-fixture", func(f *Facts161) { f.Accuracy[0].Reference = "native:accuracy" }, []string{AccuracyMalformed161}},
		{"accuracy-min-nil", func(f *Facts161) { f.Accuracy[0].MinMeasured = nil }, []string{AccuracyMalformed161}},
		{"accuracy-max-nil", func(f *Facts161) { f.Accuracy[0].MaxMeasured = nil }, []string{AccuracyMalformed161}},
		{"accuracy-fixed-nil", func(f *Facts161) { f.Accuracy[0].Fixed = nil }, []string{AccuracyMalformed161}},
		{"accuracy-fixed-negative", func(f *Facts161) { f.Accuracy[0].Fixed = aa15Int(-1) }, []string{AccuracyMalformed161}},
		{"accuracy-range-reversed", func(f *Facts161) { f.Accuracy[0].MinMeasured = aa15Int(1001) }, []string{AccuracyMalformed161}},
		{"accuracy-measured-claim", func(f *Facts161) { f.Accuracy[0].Measured = true }, []string{MeasuredClaim161}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := aa15Facts()
			c.change(&f)
			got := EncodePower161(rows, f)
			if !reflect.DeepEqual(got.Reasons, c.want) {
				t.Fatalf("reasons %v; want %v", got.Reasons, c.want)
			}
			if got.Metadata != nil || got.ActivePower != nil || got.ActiveCurrent != nil {
				t.Fatalf("value encoded without its facts: %+v", got)
			}
			want := []Withheld161{{ActivePower161, aa15Power().Key, FactsIncomplete161}, {ActiveCurrent161, aa15Current().Key, FactsIncomplete161}}
			if !reflect.DeepEqual(got.Withheld, want) {
				t.Fatalf("withheld %+v", got.Withheld)
			}
			if !reflect.DeepEqual(got.Validation, Validate(rows)) {
				t.Fatal("candidate rows changed by a fact defect")
			}
		})
	}
	t.Run("every-defect-in-fixed-order", func(t *testing.T) {
		got := EncodePower161(rows, Facts161{})
		want := []string{PowerModeInvalid161, CountInvalid161, AccuracyAbsent161, ActivePowerUndeclared161}
		if !reflect.DeepEqual(got.Reasons, want) {
			t.Fatalf("%v", got.Reasons)
		}
	})
}

// Catches: reading Accuracy from the opaque row bytes, from the value or from the
// scale. The bytes neither satisfy a fact nor change a result, and stay retained.
func TestAA15OpaqueAccuracyBytesEstablishNothing(t *testing.T) {
	row := aa15Power()
	row.Accuracy = []byte(`{"measurement_type":"ActivePower","fixed":50}`)
	f := aa15Facts()
	f.Accuracy = nil
	f.MeasurementTypes = Count161{}
	f.PowerMode = PowerMode161{}
	got := EncodePower161([]Input{row}, f)
	if got.Metadata != nil || got.ActivePower != nil {
		t.Fatalf("opaque bytes supplied a fact: %+v", got)
	}
	if !reflect.DeepEqual(got.Validation[0].Native.Accuracy, row.Accuracy) {
		t.Fatalf("opaque bytes lost: %+v", got.Validation)
	}
	base := EncodePower161([]Input{aa15Power()}, aa15Facts())
	for _, bytes := range [][]byte{nil, {}, []byte("other")} {
		other := aa15Power()
		other.Accuracy = bytes
		a := EncodePower161([]Input{other}, aa15Facts())
		if !reflect.DeepEqual(a.Metadata, base.Metadata) || *a.ActivePower.Value != *base.ActivePower.Value {
			t.Fatalf("opaque bytes changed the encoding: %+v", a)
		}
		if !reflect.DeepEqual(a.ActivePower.Source.Accuracy, bytes) {
			t.Fatalf("opaque bytes changed: %+v", a.ActivePower.Source.Accuracy)
		}
	}
}

// Catches: an aliased row buffer, value or Accuracy bound, on any path, and state
// kept between calls.
func TestAA15Detachment(t *testing.T) {
	row, f := aa15Power(), aa15Facts()
	first, second := EncodePower161([]Input{row}, f), EncodePower161([]Input{row}, f)
	row.Evidence[0], row.Accuracy[0], *f.Accuracy[0].Fixed, *f.Accuracy[0].MinMeasured, *f.Accuracy[0].MaxMeasured = 'X', 'X', 99, 99, 99
	for _, got := range []Result161{first, second} {
		a := got.ActivePower
		if string(a.Source.Evidence) != "synthetic active-power" || string(a.Source.Accuracy) != "opaque active-power" {
			t.Fatalf("attribute aliases input: %+v", a)
		}
		if string(got.Validation[0].Native.Evidence) != "synthetic active-power" {
			t.Fatalf("validation aliases input: %+v", got.Validation)
		}
		e := got.Metadata.Accuracy[0]
		if *e.Fixed != 50 || *e.MinMeasured != -1000 || *e.MaxMeasured != 1000 {
			t.Fatalf("metadata aliases input: %+v", e)
		}
	}
	first.ActivePower.Source.Evidence[0], first.ActivePower.Source.Accuracy[0], *first.ActivePower.Value = 'Y', 'Y', 99
	if string(first.Validation[0].Native.Evidence) != "synthetic active-power" || string(first.Validation[0].Native.Accuracy) != "opaque active-power" || *first.Validation[0].Value != -250000 {
		t.Fatalf("attribute aliases its validation row: %+v", first.Validation)
	}
	first.Metadata.Accuracy[0].Fixed, first.Surfaces[0].Name = aa15Int(7), "changed"
	if *second.Metadata.Accuracy[0].Fixed != 50 || second.Surfaces[0].Name != "apparent-power" || *second.ActivePower.Value != -250000 {
		t.Fatalf("results alias each other: %+v", second)
	}
	null := aa15Power()
	null.Availability, null.Decimal = "unavailable", ""
	got := EncodePower161([]Input{null}, aa15Facts())
	null.Evidence[0] = 'X'
	if string(got.ActivePower.Source.Evidence) != "synthetic active-power" {
		t.Fatalf("null aliases input: %+v", got.ActivePower)
	}
}

// All 24 row orders of a mix of accepted, withheld, null and unbound rows. Catches
// a result that depends on row order or a row dropped with its withheld neighbour.
func TestAA15PartialResultsAreOrderIndependent(t *testing.T) {
	power, current, voltage, sibling := aa15Power(), aa15Current(), aa15Voltage(), aa15Power()
	power.Decimal = "0.0001"
	current.Availability, current.Decimal = "unavailable", ""
	sibling.Key.Phase, sibling.Decimal = "b", "1"
	rows := []Input{power, current, voltage, sibling}
	want := EncodePower161(rows, aa15FactsWithVoltage())
	if want.ActivePower != nil || want.ActiveCurrent == nil || !want.ActiveCurrent.Null || *want.Voltage.Value != 230000 || len(want.Validation) != 4 {
		t.Fatalf("control: %+v", want)
	}
	if !reflect.DeepEqual(want.Withheld, []Withheld161{{ActivePower161, power.Key, "precision-loss"}}) {
		t.Fatalf("control: %+v", want.Withheld)
	}
	var permute func(int)
	permute = func(start int) {
		if start == len(rows) {
			if got := EncodePower161(rows, aa15FactsWithVoltage()); !reflect.DeepEqual(got, want) {
				t.Fatalf("permutation changed results: %+v", got)
			}
			return
		}
		for j := start; j < len(rows); j++ {
			rows[start], rows[j] = rows[j], rows[start]
			permute(start + 1)
			rows[start], rows[j] = rows[j], rows[start]
		}
	}
	permute(0)
}

// A static check of its declared document and of the adapter's imports: it catches
// the contribution order or the boundary being edited, and no runtime defect. This
// package has no runtime candidate list.
func TestAA15OrderAndBoundaryStayDeclared(t *testing.T) {
	data, err := os.ReadFile("../../docs/adoption/reference-bridge.md")
	if err != nil {
		t.Fatal(err)
	}
	last := -1
	for _, stage := range []string{"| Reference + [bridge]", "| 1. [Thermostat/HVAC]", "| 2. [EEM]", "| 3. [EPM]"} {
		at := strings.Index(string(data), stage)
		if at <= last {
			t.Fatalf("%q at %d after %d", stage, at, last)
		}
		last = at
	}
	file, err := parser.ParseFile(token.NewFileSet(), "matter161.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	imports := []string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		imports = append(imports, path)
	}
	if !reflect.DeepEqual(imports, []string{"strings"}) {
		t.Fatalf("adapter imports %v", imports)
	}
}
