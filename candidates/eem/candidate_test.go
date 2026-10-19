package eem

import (
	"math"
	"reflect"
	"testing"
)

func synthetic() Reading {
	return Reading{
		Identity:  Identity{Owner: "synthetic-native", Profile: "meter-family", Fact: "energy", Scope: "service-a", Direction: Import, Kind: Periodic, ResetEpoch: "epoch-a", HasInterval: true, Start: 0, End: 10},
		Magnitude: 7, WhPerUnit: 1000, Known: true, ContinuityKnown: true, Complete: true, Native: []byte{7, 0},
	}
}

func TestValidVectors(t *testing.T) {
	for _, direction := range []Direction{Import, Export} {
		for _, kind := range []Kind{Cumulative, Periodic} {
			r := synthetic()
			r.Identity.Direction, r.Identity.Kind = direction, kind
			if kind == Cumulative {
				r.Identity.HasInterval, r.Identity.Start, r.Identity.End = false, 0, 0
			}
			t.Run(string(direction)+"/"+string(kind), func(t *testing.T) {
				got := Validate(r.Identity, r)
				want := &Vector{Identity: r.Identity, Wh: 7000}
				if got.Reason != Valid || !reflect.DeepEqual(got.Vector, want) || !reflect.DeepEqual(got.Retained, r) {
					t.Fatalf("got %+v; want vector %+v and retained source", got, want)
				}
			})
		}
	}
	for _, tc := range []struct {
		magnitude int64
		scale     uint64
		want      int64
	}{
		{0, 1000, 0}, {math.MaxInt64, 1, math.MaxInt64}, {3, 2, 6},
	} {
		r := synthetic()
		r.Magnitude, r.WhPerUnit = tc.magnitude, tc.scale
		got := Validate(r.Identity, r)
		if got.Reason != Valid || got.Vector == nil || got.Vector.Wh != tc.want {
			t.Fatalf("boundary %+v: %+v", tc, got)
		}
	}
}

func TestIdentityMismatchAxes(t *testing.T) {
	// Each variant changes one independently valid key axis. Dropping that
	// axis from the observed/expected comparison must lose this withholding.
	axes := []struct {
		name   string
		change func(*Identity)
	}{
		{"owner", func(i *Identity) { i.Owner = "other-native" }},
		{"profile", func(i *Identity) { i.Profile = "other-family" }},
		{"fact", func(i *Identity) { i.Fact = "other-energy" }},
		{"scope", func(i *Identity) { i.Scope = "service-b" }},
		{"direction", func(i *Identity) { i.Direction = Export }},
		{"reset_epoch", func(i *Identity) { i.ResetEpoch = "epoch-b" }},
		{"start", func(i *Identity) { i.Start = 1 }},
		{"end", func(i *Identity) { i.End = 11 }},
	}
	for _, axis := range axes {
		t.Run(axis.name, func(t *testing.T) {
			r := synthetic()
			expected := r.Identity
			axis.change(&r.Identity)
			assertWithheld(t, expected, r, IdentityMismatch)
		})
	}
	// Kind and interval presence have coupled structural constraints. This
	// valid variant catches collapsing the entire cumulative/periodic tuple.
	t.Run("kind_and_interval_tuple", func(t *testing.T) {
		r := synthetic()
		expected := r.Identity
		r.Identity.Kind, r.Identity.HasInterval, r.Identity.End = Cumulative, false, 0
		assertWithheld(t, expected, r, IdentityMismatch)
	})
}

func TestMalformedIdentityAxes(t *testing.T) {
	axes := []struct {
		name   string
		change func(*Identity)
	}{
		{"owner_missing", func(i *Identity) { i.Owner = "" }},
		{"profile_missing", func(i *Identity) { i.Profile = "" }},
		{"fact_missing", func(i *Identity) { i.Fact = "" }},
		{"scope_missing", func(i *Identity) { i.Scope = "" }},
		{"direction_missing", func(i *Identity) { i.Direction = "" }},
		{"direction_generation_not_export", func(i *Identity) { i.Direction = "generated" }},
		{"kind_missing", func(i *Identity) { i.Kind = "" }},
		{"kind_daily_not_cumulative", func(i *Identity) { i.Kind = "daily" }},
		{"reset_missing", func(i *Identity) { i.ResetEpoch = "" }},
		{"interval_presence_missing", func(i *Identity) { i.HasInterval = false }},
		{"start_negative", func(i *Identity) { i.Start = -1 }},
		{"end_equal", func(i *Identity) { i.End = 0 }},
		{"end_reversed", func(i *Identity) { i.End = -1 }},
		{"cumulative_with_interval", func(i *Identity) { i.Kind = Cumulative }},
		{"cumulative_presence_only", func(i *Identity) { i.Kind = Cumulative; i.End = 0 }},
		{"cumulative_with_start", func(i *Identity) { i.Kind = Cumulative; i.HasInterval = false; i.Start = 1; i.End = 0 }},
		{"cumulative_with_end", func(i *Identity) { i.Kind = Cumulative; i.HasInterval = false }},
	}
	for _, axis := range axes {
		t.Run(axis.name, func(t *testing.T) {
			r := synthetic()
			axis.change(&r.Identity)
			// Both sides malformed must not pass by equality. Also exercise
			// each input independently to catch dropping either validation.
			assertWithheld(t, r.Identity, r, InvalidIdentity)
			assertWithheld(t, synthetic().Identity, r, InvalidIdentity)
			assertWithheld(t, r.Identity, synthetic(), InvalidIdentity)
		})
	}
}

func TestWithholdingGuards(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Reading)
		want   Reason
	}{
		{"unknown_not_zero", func(r *Reading) { r.Known = false; r.Magnitude = 0 }, ValueUnknown},
		{"continuity_loss", func(r *Reading) { r.ContinuityKnown = false }, ContinuityUnavailable},
		{"partial_interval", func(r *Reading) { r.Complete = false }, PartialPeriod},
		{"negative_not_direction", func(r *Reading) { r.Magnitude = -7 }, InvalidValue},
		{"scale_missing", func(r *Reading) { r.WhPerUnit = 0 }, InvalidValue},
		{"multiplication_overflow", func(r *Reading) { r.Magnitude = math.MaxInt64; r.WhPerUnit = 2 }, Overflow},
		{"scale_overflow", func(r *Reading) { r.Magnitude = 0; r.WhPerUnit = math.MaxUint64 }, Overflow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := synthetic()
			tc.change(&r)
			assertWithheld(t, r.Identity, r, tc.want)
		})
	}
	t.Run("partial_cannot_be_total", func(t *testing.T) {
		r := synthetic()
		r.Complete = false
		expected := r.Identity
		expected.Kind, expected.HasInterval, expected.End = Cumulative, false, 0
		assertWithheld(t, expected, r, IdentityMismatch)
	})
	t.Run("positive_sign_cannot_supply_direction", func(t *testing.T) {
		r := synthetic()
		r.Identity.Direction = ""
		assertWithheld(t, r.Identity, r, InvalidIdentity)
	})
}

func assertWithheld(t *testing.T, expected Identity, r Reading, reason Reason) {
	t.Helper()
	got := Validate(expected, r)
	if got.Reason != reason || got.Vector != nil || !reflect.DeepEqual(got.Retained, r) {
		t.Fatalf("got %+v; want %s, no vector, and retained source %+v", got, reason, r)
	}
}

func TestPermutationAndDetachment(t *testing.T) {
	a := synthetic()
	b := synthetic()
	b.Identity.Direction, b.Magnitude = Export, 9
	c := synthetic()
	c.Known = false
	inputs := []Reading{a, b, c}
	want := []Result{
		{Retained: a, Vector: &Vector{Identity: a.Identity, Wh: 7000}, Reason: Valid},
		{Retained: b, Vector: &Vector{Identity: b.Identity, Wh: 9000}, Reason: Valid},
		{Retained: c, Reason: ValueUnknown},
	}
	for _, order := range [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		for _, index := range order {
			got := Validate(inputs[index].Identity, inputs[index])
			if !reflect.DeepEqual(got, want[index]) {
				t.Fatalf("order %v index %d: %+v", order, index, got)
			}
		}
	}
	for _, known := range []bool{true, false} {
		r := synthetic()
		r.Known = known
		first, second := Validate(r.Identity, r), Validate(r.Identity, r)
		r.Native[0] = 99
		r.Identity.ResetEpoch = "changed"
		if first.Retained.Native[0] != 7 || first.Retained.Identity.ResetEpoch != "epoch-a" {
			t.Fatal("input aliases retained evidence")
		}
		first.Retained.Native[1] = 88
		if second.Retained.Native[1] != 0 || r.Native[1] != 0 {
			t.Fatal("results alias source or each other")
		}
		if known {
			first.Vector.Wh = 99
			if second.Vector.Wh != 7000 {
				t.Fatal("vectors alias")
			}
		}
	}
}
