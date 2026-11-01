package eem

import (
	"math"
	"reflect"
	"testing"
)

func aa14Bound(v int64) *int64 { return &v }

func aa14Periodic() ExactReading161 {
	return ExactReading161{
		Identity: Identity{Owner: "synthetic-native", Profile: "meter-family", Fact: "energy", Scope: "service-a", Direction: Import, Kind: Periodic, ResetEpoch: "epoch-a", HasInterval: true, Start: 2, End: 10},
		Value:    Decimal161{Coefficient: 15, Exponent: -1, Unit: Wh161},
		Known:    true, ContinuityKnown: true, Complete: true,
		Native: []byte{15, 0},
	}
}

func aa14Cumulative() ExactReading161 {
	r := aa14Periodic()
	r.Identity.Kind, r.Identity.HasInterval, r.Identity.Start, r.Identity.End = Cumulative, false, 0, 0
	return r
}

func aa14Encoded(t *testing.T, r ExactReading161) *Energy161 {
	t.Helper()
	got := EncodeEnergy161(r.Identity, r)
	if got.Reason != Valid || got.Energy == nil {
		t.Fatalf("valid control withheld: %+v", got)
	}
	return got.Energy
}

func aa14Withheld(t *testing.T, expected Identity, r ExactReading161, reason Reason) {
	t.Helper()
	got := EncodeEnergy161(expected, r)
	if got.Reason != reason || got.Energy != nil {
		t.Fatalf("got %s and %+v; want %s and no value", got.Reason, got.Energy, reason)
	}
	if !reflect.DeepEqual(got.Retained, r) {
		t.Fatalf("retained %+v; want %+v", got.Retained, r)
	}
}

// Catches: changed identity or evidence constants, a dropped Energy161 field, and
// any value derived from the whole-Wh vector. Expected values are literal.
func TestAA14ExactDTO(t *testing.T) {
	r := aa14Periodic()
	r.Time = &Time161{Frame: "synthetic-frame", Start: aa14Bound(100), End: aa14Bound(200)}
	got := EncodeEnergy161(r.Identity, r)
	target := Target161{"1.6.1", "ElectricalEnergyMeasurement", 0x0091, 2, "mWh", 0, math.MaxInt64, "Proven", "Hypothesis"}
	if got.Target != target {
		t.Fatalf("target: %+v", got.Target)
	}
	want := &Energy161{
		Identity: Identity{"synthetic-native", "meter-family", "energy", "service-a", Import, Periodic, "epoch-a", true, 2, 10},
		Slot:     "imported-periodic", MilliWh: 1500,
		Time:   &Time161{"synthetic-frame", aa14Bound(100), aa14Bound(200)},
		Source: Decimal161{15, -1, "Wh"}, Native: []byte{15, 0},
	}
	if got.Reason != "candidate_valid" || !reflect.DeepEqual(got.Energy, want) || !reflect.DeepEqual(got.Retained, r) {
		t.Fatalf("%+v", got)
	}
	if got.SemanticMapping != "Unknown" || got.TargetClosure != "Unknown" || got.LibrarySupport != "Unknown" || got.ProductionAdmitted {
		t.Fatalf("outcome upgraded: %+v", got)
	}
	names := []string{"aggregation", "differencing", "events", "feature-map", "measurement-accuracy", "reset-operation", "reset-structure"}
	if len(got.Surfaces) != len(names) {
		t.Fatalf("%+v", got.Surfaces)
	}
	for i, name := range names {
		if got.Surfaces[i] != (Surface161{name, "withheld", "Unknown"}) {
			t.Fatalf("%d: %+v", i, got.Surfaces[i])
		}
	}
}

// Catches: Wh treated as mWh, any changed unit scale, truncation, rounding and an
// exponent ignored or applied with the wrong sign. Expected values are literal.
func TestAA14ExactConversion(t *testing.T) {
	for _, c := range []struct {
		name    string
		value   Decimal161
		milliWh int64
	}{
		{"mwh-unscaled", Decimal161{7, 0, MilliWh161}, 7},
		{"wh-is-thousand-mwh", Decimal161{7, 0, Wh161}, 7000},
		{"kwh-is-million-mwh", Decimal161{7, 0, KiloWh161}, 7000000},
		{"upper-mwh-is-billion-mwh", Decimal161{7, 0, MegaWh161}, 7000000000},
		{"fractional-wh-not-truncated", Decimal161{15, -1, Wh161}, 1500},
		{"fractional-wh-near-whole", Decimal161{1999, -3, Wh161}, 1999},
		{"sub-wh-half", Decimal161{5, -1, Wh161}, 500},
		{"sub-wh-smallest", Decimal161{1, -3, Wh161}, 1},
		{"fractional-kwh", Decimal161{1234567, -6, KiloWh161}, 1234567},
		{"positive-exponent", Decimal161{3, 2, MilliWh161}, 300},
		{"positive-exponent-with-unit", Decimal161{3, 2, Wh161}, 300000},
		{"trailing-zero-fraction", Decimal161{10, -4, Wh161}, 1},
		{"exponent-cancels-unit", Decimal161{42, -3, Wh161}, 42},
		{"limit-mwh", Decimal161{math.MaxInt64, 0, MilliWh161}, math.MaxInt64},
		{"limit-whole-wh", Decimal161{9223372036854775, 0, Wh161}, 9223372036854775000},
		{"limit-fractional-wh", Decimal161{9223372036854775807, -3, Wh161}, 9223372036854775807},
		{"zero-mwh", Decimal161{0, 0, MilliWh161}, 0},
		{"zero-wh", Decimal161{0, 0, Wh161}, 0},
		{"zero-fractional-scale", Decimal161{0, -9, Wh161}, 0},
		{"zero-low-exponent-limit", Decimal161{0, math.MinInt32, MegaWh161}, 0},
		{"zero-high-exponent-limit", Decimal161{0, math.MaxInt32, MegaWh161}, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := aa14Periodic()
			r.Value = c.value
			e := aa14Encoded(t, r)
			if e.MilliWh != c.milliWh || e.Source != c.value {
				t.Fatalf("%+v; want %d mWh", e, c.milliWh)
			}
		})
	}
}

// Catches: truncation, rounding, wrapped multiplication, sign folding, a missing
// or case-insensitive unit match and a power table that cannot hold the exponent.
func TestAA14RejectsAmbiguousValues(t *testing.T) {
	for _, c := range []struct {
		name  string
		value Decimal161
		want  Reason
	}{
		{"sub-mwh", Decimal161{5, -1, MilliWh161}, PrecisionLoss161},
		{"sub-mwh-from-wh", Decimal161{5, -4, Wh161}, PrecisionLoss161},
		{"one-digit-past-mwh", Decimal161{15005, -4, Wh161}, PrecisionLoss161},
		{"sub-mwh-from-upper-unit", Decimal161{1, -10, MegaWh161}, PrecisionLoss161},
		{"low-exponent-limit", Decimal161{1, math.MinInt32, MilliWh161}, PrecisionLoss161},
		{"limit-coefficient-past-mwh", Decimal161{math.MaxInt64, -4, Wh161}, PrecisionLoss161},
		{"over-limit-whole-wh", Decimal161{9223372036854776, 0, Wh161}, Overflow},
		{"limit-coefficient-as-wh", Decimal161{math.MaxInt64, 0, Wh161}, Overflow},
		{"limit-coefficient-times-ten", Decimal161{math.MaxInt64, 1, MilliWh161}, Overflow},
		{"exponent-overflow", Decimal161{1, 19, MilliWh161}, Overflow},
		{"high-exponent-limit", Decimal161{1, math.MaxInt32, MegaWh161}, Overflow},
		{"negative-not-direction", Decimal161{-7, 0, Wh161}, InvalidValue},
		{"negative-fraction", Decimal161{-15, -1, Wh161}, InvalidValue},
		{"negative-limit", Decimal161{math.MinInt64, 0, MilliWh161}, InvalidValue},
		{"unit-missing", Decimal161{7, 0, ""}, UnitUnknown161},
		{"unit-lowercase-wh", Decimal161{7, 0, "wh"}, UnitUnknown161},
		{"unit-uppercase-mwh", Decimal161{7, 0, "MWH"}, UnitUnknown161},
		{"unit-other-quantity", Decimal161{7, 0, "J"}, UnitUnknown161},
		{"unit-missing-zero-not-fabricated", Decimal161{0, 0, ""}, UnitUnknown161},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := aa14Periodic()
			r.Value = c.value
			aa14Withheld(t, r.Identity, r, c.want)
		})
	}
	// mWh and MWh differ only by case and a factor of a billion.
	lower, upper := aa14Periodic(), aa14Periodic()
	lower.Value, upper.Value = Decimal161{1, 0, "mWh"}, Decimal161{1, 0, "MWh"}
	if a, b := aa14Encoded(t, lower).MilliWh, aa14Encoded(t, upper).MilliWh; a != 1 || b != 1000000000 {
		t.Fatalf("%d %d", a, b)
	}
}

// Each field varied alone from a valid control on either side, so dropping it
// from the expected/observed comparison loses the withholding.
func TestAA14IdentityMismatchAxes(t *testing.T) {
	if e := aa14Encoded(t, aa14Periodic()); e.MilliWh != 1500 {
		t.Fatalf("control: %+v", e)
	}
	for _, c := range []struct {
		name   string
		change func(*Identity)
	}{
		{"owner", func(i *Identity) { i.Owner = "other-native" }},
		{"profile", func(i *Identity) { i.Profile = "other-family" }},
		{"fact", func(i *Identity) { i.Fact = "other-energy" }},
		{"scope", func(i *Identity) { i.Scope = "service-b" }},
		{"direction", func(i *Identity) { i.Direction = Export }},
		{"reset-epoch", func(i *Identity) { i.ResetEpoch = "epoch-b" }},
		{"start", func(i *Identity) { i.Start = 3 }},
		{"start-evidenced-zero", func(i *Identity) { i.Start = 0 }},
		{"end", func(i *Identity) { i.End = 11 }},
		// Kind and HasInterval are structurally coupled; this is their valid tuple.
		{"kind-and-interval-tuple", func(i *Identity) { i.Kind, i.HasInterval, i.Start, i.End = Cumulative, false, 0, 0 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := aa14Periodic()
			expected := r.Identity
			c.change(&r.Identity)
			aa14Withheld(t, expected, r, IdentityMismatch)
			flipped := aa14Periodic()
			c.change(&expected)
			aa14Withheld(t, expected, flipped, IdentityMismatch)
		})
	}
}

// A malformed identity fails on either side and on both, so equality cannot pass
// it. Kind and HasInterval alone leave the coupled structure and land here.
func TestAA14MalformedIdentityAxes(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(*Identity)
	}{
		{"owner", func(i *Identity) { i.Owner = "" }},
		{"profile", func(i *Identity) { i.Profile = "" }},
		{"fact", func(i *Identity) { i.Fact = "" }},
		{"scope", func(i *Identity) { i.Scope = "" }},
		{"direction-missing", func(i *Identity) { i.Direction = "" }},
		{"direction-generated-not-export", func(i *Identity) { i.Direction = "generated" }},
		{"kind-missing", func(i *Identity) { i.Kind = "" }},
		{"kind-alone", func(i *Identity) { i.Kind = Cumulative }},
		{"reset-epoch", func(i *Identity) { i.ResetEpoch = "" }},
		{"has-interval-alone", func(i *Identity) { i.HasInterval = false }},
		{"unknown-time-bounds", func(i *Identity) { i.HasInterval, i.Start, i.End = false, 0, 0 }},
		{"start-negative", func(i *Identity) { i.Start = -1 }},
		{"end-equal", func(i *Identity) { i.End = i.Start }},
		{"end-reversed", func(i *Identity) { i.End = 1 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := aa14Periodic()
			c.change(&r.Identity)
			aa14Withheld(t, r.Identity, r, InvalidIdentity)
			aa14Withheld(t, aa14Periodic().Identity, r, InvalidIdentity)
			aa14Withheld(t, r.Identity, aa14Periodic(), InvalidIdentity)
		})
	}
	for _, c := range []struct {
		name   string
		change func(*Identity)
	}{
		{"has-interval", func(i *Identity) { i.HasInterval = true }},
		{"start", func(i *Identity) { i.Start = 1 }},
		{"end", func(i *Identity) { i.End = 1 }},
	} {
		t.Run("cumulative-"+c.name, func(t *testing.T) {
			r := aa14Cumulative()
			c.change(&r.Identity)
			aa14Withheld(t, r.Identity, r, InvalidIdentity)
		})
	}
}

// Catches: a slot or identity that conflates import/export or cumulative/periodic,
// or a sign choosing direction. Each slot is independently literal.
func TestAA14DirectionAndKindStaySeparate(t *testing.T) {
	cases := []struct {
		direction Direction
		kind      Kind
		slot      Slot161
	}{
		{Import, Cumulative, "imported-cumulative"},
		{Export, Cumulative, "exported-cumulative"},
		{Import, Periodic, "imported-periodic"},
		{Export, Periodic, "exported-periodic"},
	}
	for _, c := range cases {
		t.Run(string(c.slot), func(t *testing.T) {
			r := aa14Periodic()
			if c.kind == Cumulative {
				r = aa14Cumulative()
			}
			r.Identity.Direction = c.direction
			e := aa14Encoded(t, r)
			if e.Slot != c.slot || e.Identity.Direction != c.direction || e.Identity.Kind != c.kind || e.MilliWh != 1500 {
				t.Fatalf("%+v", e)
			}
			for _, other := range cases {
				if other.direction == c.direction && other.kind == c.kind {
					continue
				}
				o := aa14Periodic()
				if other.kind == Cumulative {
					o = aa14Cumulative()
				}
				o.Identity.Direction = other.direction
				aa14Withheld(t, o.Identity, r, IdentityMismatch)
			}
		})
	}
}

// The whole identity reaches the DTO, including a changed reset epoch.
func TestAA14EncodedIdentityIsRetained(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(*Identity)
	}{
		{"owner", func(i *Identity) { i.Owner = "other-native" }},
		{"profile", func(i *Identity) { i.Profile = "other-family" }},
		{"fact", func(i *Identity) { i.Fact = "other-energy" }},
		{"scope", func(i *Identity) { i.Scope = "service-b" }},
		{"direction", func(i *Identity) { i.Direction = Export }},
		{"reset-epoch", func(i *Identity) { i.ResetEpoch = "epoch-b" }},
		{"start", func(i *Identity) { i.Start = 0 }},
		{"end", func(i *Identity) { i.End = 11 }},
		{"kind-and-interval-tuple", func(i *Identity) { i.Kind, i.HasInterval, i.Start, i.End = Cumulative, false, 0, 0 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := aa14Periodic()
			c.change(&r.Identity)
			if e := aa14Encoded(t, r); e.Identity != r.Identity {
				t.Fatalf("%+v; want %+v", e.Identity, r.Identity)
			}
		})
	}
}

// Each availability fact varied alone on each kind. Catches: a removed guard, a
// period-only guard, or an unknown value becoming zero. A zero value still needs
// every fact, so the guards cannot be bypassed through the zero path.
func TestAA14AvailabilityGuards(t *testing.T) {
	for _, base := range []struct {
		name    string
		reading func() ExactReading161
	}{{"periodic", aa14Periodic}, {"cumulative", aa14Cumulative}} {
		for _, c := range []struct {
			name   string
			change func(*ExactReading161)
			want   Reason
		}{
			{"unknown", func(r *ExactReading161) { r.Known = false }, ValueUnknown},
			{"continuity", func(r *ExactReading161) { r.ContinuityKnown = false }, ContinuityUnavailable},
			{"incomplete", func(r *ExactReading161) { r.Complete = false }, PartialPeriod},
			{"native-missing", func(r *ExactReading161) { r.Native = nil }, NativeMissing161},
			{"native-empty", func(r *ExactReading161) { r.Native = []byte{} }, NativeMissing161},
		} {
			t.Run(base.name+"/"+c.name, func(t *testing.T) {
				aa14Encoded(t, base.reading())
				r := base.reading()
				c.change(&r)
				aa14Withheld(t, r.Identity, r, c.want)
				r.Value = Decimal161{0, 0, Wh161}
				aa14Withheld(t, r.Identity, r, c.want)
			})
		}
		t.Run(base.name+"/known-zero-encodes", func(t *testing.T) {
			r := base.reading()
			r.Value = Decimal161{0, 0, Wh161}
			if e := aa14Encoded(t, r); e.MilliWh != 0 {
				t.Fatalf("%+v", e)
			}
		})
	}
	// First failing guard wins, whatever else is also wrong.
	r := aa14Periodic()
	r.Known, r.ContinuityKnown, r.Complete, r.Native, r.Value = false, false, false, nil, Decimal161{-1, 0, "J"}
	aa14Withheld(t, r.Identity, r, ValueUnknown)
	r.Known = true
	aa14Withheld(t, r.Identity, r, ContinuityUnavailable)
	r.ContinuityKnown = true
	aa14Withheld(t, r.Identity, r, PartialPeriod)
	r.Complete = true
	aa14Withheld(t, r.Identity, r, NativeMissing161)
}

// Catches: unknown bounds fabricated as zero, ticks copied into time metadata, an
// evidenced zero lost, half-known periods accepted, and unordered bounds accepted.
func TestAA14TimeMetadata(t *testing.T) {
	t.Run("absent-stays-unknown", func(t *testing.T) {
		if e := aa14Encoded(t, aa14Periodic()); e.Time != nil || e.Identity.Start != 2 || e.Identity.End != 10 {
			t.Fatalf("%+v", e)
		}
		if e := aa14Encoded(t, aa14Cumulative()); e.Time != nil || e.Identity.HasInterval {
			t.Fatalf("%+v", e)
		}
	})
	zero := func() *int64 { return aa14Bound(0) }
	for _, c := range []struct {
		name string
		base func() ExactReading161
		time Time161
	}{
		{"cumulative-start-only", aa14Cumulative, Time161{"frame", aa14Bound(5), nil}},
		{"cumulative-end-only", aa14Cumulative, Time161{"frame", nil, aa14Bound(5)}},
		{"cumulative-evidenced-zero-start", aa14Cumulative, Time161{"frame", zero(), nil}},
		{"cumulative-evidenced-zero-end", aa14Cumulative, Time161{"frame", nil, zero()}},
		{"cumulative-both", aa14Cumulative, Time161{"frame", aa14Bound(5), aa14Bound(6)}},
		{"periodic-both", aa14Periodic, Time161{"frame", aa14Bound(5), aa14Bound(6)}},
		{"periodic-evidenced-zero-start", aa14Periodic, Time161{"other-frame", zero(), aa14Bound(6)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := c.base()
			r.Time = &c.time
			if e := aa14Encoded(t, r); !reflect.DeepEqual(e.Time, &c.time) {
				t.Fatalf("%+v; want %+v", e.Time, c.time)
			}
		})
	}
	for _, c := range []struct {
		name string
		base func() ExactReading161
		time Time161
	}{
		{"frame-missing", aa14Cumulative, Time161{"", aa14Bound(5), nil}},
		{"no-bounds", aa14Cumulative, Time161{"frame", nil, nil}},
		{"periodic-no-bounds", aa14Periodic, Time161{"frame", nil, nil}},
		{"periodic-start-only", aa14Periodic, Time161{"frame", aa14Bound(5), nil}},
		{"periodic-end-only", aa14Periodic, Time161{"frame", nil, aa14Bound(5)}},
		{"negative-start", aa14Cumulative, Time161{"frame", aa14Bound(-1), nil}},
		{"negative-end", aa14Cumulative, Time161{"frame", nil, aa14Bound(-1)}},
		{"negative-start-with-end", aa14Periodic, Time161{"frame", aa14Bound(-1), aa14Bound(5)}},
		{"equal-bounds", aa14Periodic, Time161{"frame", aa14Bound(5), aa14Bound(5)}},
		{"reversed-bounds", aa14Periodic, Time161{"frame", aa14Bound(6), aa14Bound(5)}},
		{"cumulative-reversed-bounds", aa14Cumulative, Time161{"frame", aa14Bound(6), aa14Bound(5)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := c.base()
			r.Time = &c.time
			aa14Withheld(t, r.Identity, r, TimeInvalid161)
		})
	}
}

// Catches: an aliased native buffer or time bound on any path, and cross-call
// state. A withheld result keeps its evidence exactly, nil and empty included.
func TestAA14RetainedEvidenceIsDetached(t *testing.T) {
	r := aa14Cumulative()
	r.Time = &Time161{"frame", aa14Bound(5), aa14Bound(6)}
	first, second := EncodeEnergy161(r.Identity, r), EncodeEnergy161(r.Identity, r)
	r.Native[0], *r.Time.Start, *r.Time.End = 99, 99, 99
	for _, got := range []Result161{first, second} {
		if got.Retained.Native[0] != 15 || *got.Retained.Time.Start != 5 || *got.Retained.Time.End != 6 {
			t.Fatalf("retained aliases input: %+v", got.Retained)
		}
		if got.Energy.Native[0] != 15 || *got.Energy.Time.Start != 5 || *got.Energy.Time.End != 6 {
			t.Fatalf("value aliases input: %+v", got.Energy)
		}
	}
	first.Retained.Native[0], *first.Retained.Time.Start, *first.Retained.Time.End = 77, 77, 77
	if first.Energy.Native[0] != 15 || *first.Energy.Time.Start != 5 || *first.Energy.Time.End != 6 {
		t.Fatal("value aliases retained evidence")
	}
	first.Energy.Native[0], first.Surfaces[0].Name = 88, "changed"
	if second.Energy.Native[0] != 15 || second.Retained.Native[0] != 15 || second.Surfaces[0].Name != "aggregation" {
		t.Fatal("results alias each other")
	}
	withheld := aa14Periodic()
	withheld.Complete = false
	withheld.Time = &Time161{"frame", aa14Bound(5), nil}
	got := EncodeEnergy161(withheld.Identity, withheld)
	withheld.Native[0], *withheld.Time.Start = 99, 99
	if got.Energy != nil || got.Retained.Native[0] != 15 || *got.Retained.Time.Start != 5 || got.Retained.Time.End != nil {
		t.Fatalf("withheld evidence: %+v", got)
	}
	missing := aa14Periodic()
	missing.Native = nil
	if got := EncodeEnergy161(missing.Identity, missing); got.Retained.Native != nil {
		t.Fatalf("absent evidence fabricated: %+v", got.Retained)
	}
	missing.Native = []byte{}
	if got := EncodeEnergy161(missing.Identity, missing); got.Retained.Native == nil || len(got.Retained.Native) != 0 {
		t.Fatalf("empty evidence changed: %+v", got.Retained)
	}
}

// Readings in any order give the same results, so no pair is differenced or summed.
func TestAA14NoCrossReadingState(t *testing.T) {
	a := aa14Periodic()
	b := aa14Periodic()
	b.Identity.Start, b.Identity.End, b.Value = 10, 20, Decimal161{25, -1, Wh161}
	c := aa14Cumulative()
	c.Identity.Direction, c.Value = Export, Decimal161{7, 0, KiloWh161}
	inputs := []ExactReading161{a, b, c}
	want := []int64{1500, 2500, 7000000}
	for _, order := range [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		for _, index := range order {
			if e := aa14Encoded(t, inputs[index]); e.MilliWh != want[index] {
				t.Fatalf("order %v index %d: %+v", order, index, e)
			}
		}
	}
}

// The whole-Wh convention keeps its own input and result: 7 units of 1000 Wh stay
// 7000 Wh there, and the decimal path is what carries a fraction without loss.
func TestAA14WholeWhConventionUnchanged(t *testing.T) {
	whole := Reading{Identity: aa14Periodic().Identity, Magnitude: 7, WhPerUnit: 1000, Known: true, ContinuityKnown: true, Complete: true, Native: []byte{7, 0}}
	got := Validate(whole.Identity, whole)
	if got.Reason != Valid || got.Vector == nil || got.Vector.Wh != 7000 {
		t.Fatalf("%+v", got)
	}
	exact := aa14Periodic()
	exact.Value = Decimal161{7, 0, KiloWh161}
	if e := aa14Encoded(t, exact); e.MilliWh != 7000000 {
		t.Fatalf("%+v", e)
	}
	exact.Value = Decimal161{70005, -1, Wh161}
	if e := aa14Encoded(t, exact); e.MilliWh != 7000500 {
		t.Fatalf("%+v", e)
	}
}
