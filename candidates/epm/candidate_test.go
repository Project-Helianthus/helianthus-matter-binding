package epm

import (
	"reflect"
	"strings"
	"testing"
)

func sample() Input {
	return Input{
		Key:          Key{Owner: "synthetic-native", Profile: "meter-family", Source: "source-a", Scope: "phase", Phase: "a", Fact: "active-power"},
		Availability: "available", Decimal: "-250", Unit: "W", OutputUnit: "W",
		Evidence: []byte("synthetic evidence"), Accuracy: []byte("caller accuracy: unresolved schema"),
	}
}

func TestExactVectors(t *testing.T) {
	for _, tc := range []struct {
		name, decimal string
		scale, out    int
		want          int64
	}{
		{"signed-power", "-250", 0, 0, -250},
		{"zero", "0", 0, -3, 0},
		{"negative-zero", "-0.000", -18, 18, 0},
		{"decimal-grid", "12.345", 0, -3, 12345},
		{"native-integer-scale", "-2500", -1, -3, -250000},
		{"exact-division", "12000", -3, 0, 12},
		{"trailing-zeroes", "1.2300", 0, -2, 123},
		{"int64-max", "9223372036854775807", 0, 0, 9223372036854775807},
		{"int64-min", "-9223372036854775808", 0, 0, -9223372036854775808},
		{"large-coefficient-exact-division", "922337203685477580700", -2, 0, 9223372036854775807},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := sample()
			in.Decimal, in.Scale, in.OutputScale = tc.decimal, tc.scale, tc.out
			got := Validate([]Input{in})
			want := []Result{{Native: in, Status: "exact", Value: &tc.want}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v, want exact %d with unchanged envelope", got, tc.want)
			}
		})
	}
}

// Each setter also names the production key omission this independent variant
// catches: dropping Owner, Profile, Source, Scope, Phase, or Fact from counts.
var axes = []struct {
	name string
	set  func(*Key, string)
}{
	{"Owner", func(k *Key, s string) { k.Owner = s }},
	{"Profile", func(k *Key, s string) { k.Profile = s }},
	{"Source", func(k *Key, s string) { k.Source = s }},
	{"Scope", func(k *Key, s string) { k.Scope = s }},
	{"Phase", func(k *Key, s string) { k.Phase = s }},
	{"Fact", func(k *Key, s string) { k.Fact = s }},
}

func TestEveryKeyAxis(t *testing.T) {
	for _, axis := range axes {
		t.Run(axis.name, func(t *testing.T) {
			a, b := sample(), sample()
			axis.set(&b.Key, "other")
			b.Decimal = "7"
			got := Validate([]Input{a, b})
			if len(got) != 2 {
				t.Fatalf("lost row: %+v", got)
			}
			want := map[Key]int64{a.Key: -250, b.Key: 7}
			for _, r := range got {
				n, ok := want[r.Native.Key]
				if !ok || r.Status != "exact" || r.Reason != "" || r.Value == nil || *r.Value != n {
					t.Fatalf("key axis collapsed: %+v", r)
				}
				delete(want, r.Native.Key)
			}
			if len(want) != 0 {
				t.Fatal("missing independent source")
			}
		})
	}
}

func TestMalformedIdentityAxes(t *testing.T) {
	for _, axis := range axes {
		for _, bad := range []struct{ name, value string }{
			{"empty", ""}, {"space", "bad value"}, {"control", "bad\n"},
			{"unicode", "pháse"}, {"invalid-utf8", string([]byte{255})}, {"too-long", strings.Repeat("a", 129)},
		} {
			t.Run(axis.name+"/"+bad.name, func(t *testing.T) {
				in := sample()
				axis.set(&in.Key, bad.value)
				assertWithheld(t, in, "invalid-identity")
			})
		}
	}
}

func TestMalformedValuesAndLoss(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		change       func(*Input)
	}{
		{"unknown-unit", "unit-unknown-or-mismatch", func(i *Input) { i.Unit, i.OutputUnit = "unknown", "unknown" }},
		{"missing-unit", "unit-unknown-or-mismatch", func(i *Input) { i.Unit = "" }},
		{"missing-output-unit", "unit-unknown-or-mismatch", func(i *Input) { i.OutputUnit = "" }},
		{"unit-conflation", "unit-unknown-or-mismatch", func(i *Input) { i.OutputUnit = "VA" }},
		{"input-scale-low", "scale-out-of-bounds", func(i *Input) { i.Scale = -19 }},
		{"input-scale-high", "scale-out-of-bounds", func(i *Input) { i.Scale = 19 }},
		{"output-scale-low", "scale-out-of-bounds", func(i *Input) { i.OutputScale = -19 }},
		{"output-scale-high", "scale-out-of-bounds", func(i *Input) { i.OutputScale = 19 }},
		{"unknown-availability", "availability-unknown-or-conflicting", func(i *Input) { i.Availability = "unknown" }},
		{"missing-availability", "availability-unknown-or-conflicting", func(i *Input) { i.Availability = "" }},
		{"null-with-value", "availability-unknown-or-conflicting", func(i *Input) { i.Availability = "unavailable" }},
		{"positive-loss", "precision-loss", func(i *Input) { i.Decimal = "1.001" }},
		{"negative-loss", "precision-loss", func(i *Input) { i.Decimal = "-0.1" }},
		{"scale-loss", "precision-loss", func(i *Input) { i.Decimal, i.Scale = "1", -1 }},
		{"positive-overflow", "integer-overflow", func(i *Input) { i.Decimal = "9223372036854775808" }},
		{"negative-overflow", "integer-overflow", func(i *Input) { i.Decimal = "-9223372036854775809" }},
		{"scaled-overflow", "integer-overflow", func(i *Input) { i.Decimal, i.Scale = "9223372036854775807", 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := sample()
			tc.change(&in)
			assertWithheld(t, in, tc.reason)
		})
	}
	for _, bad := range []string{"", "-", "+1", " 1", "1 ", "1e2", "NaN", "Inf", "1/2", ".1", "1.", "1.2.3", "--1", "0x10", "１", string([]byte{255}), strings.Repeat("1", 129)} {
		t.Run("decimal/"+bad, func(t *testing.T) {
			in := sample()
			in.Decimal = bad
			assertWithheld(t, in, "invalid-decimal")
		})
	}
}

func assertWithheld(t *testing.T, in Input, reason string) {
	t.Helper()
	got := Validate([]Input{in})
	want := []Result{{Native: in, Status: "withheld", Reason: reason}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want withheld %s with retained evidence", got, reason)
	}
}

func TestNullableAndOpaqueMetadata(t *testing.T) {
	zero, unavailable := sample(), sample()
	zero.Decimal, zero.Accuracy = "0", nil
	unavailable.Key.Phase = "b"
	unavailable.Availability, unavailable.Decimal = "unavailable", ""
	got := Validate([]Input{zero, unavailable})
	var n int64
	want := []Result{{Native: zero, Status: "exact", Value: &n}, {Native: unavailable, Status: "unavailable"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("zero/null/accuracy changed: %+v", got)
	}
	for _, unit := range []string{"W", "var", "VA", "A", "V", "Hz", "ratio"} {
		in := sample()
		in.Unit, in.OutputUnit = unit, unit
		in.Accuracy = []byte{}
		r := Validate([]Input{in})[0]
		if r.Status != "exact" || !reflect.DeepEqual(r.Native, in) {
			t.Fatalf("local unit or empty metadata not preserved: %+v", r)
		}
	}
}

func TestPermutationDuplicatesAndPartialResults(t *testing.T) {
	a, b, c, d := sample(), sample(), sample(), sample()
	b.Key.Phase, b.Decimal = "b", "0"
	c.Key.Phase, c.Decimal = "c", "0.1"
	d.Decimal = "9" // conflicts with a: both must withhold
	var zero int64
	want := []Result{
		{Native: a, Status: "withheld", Reason: "duplicate-identity"},
		{Native: d, Status: "withheld", Reason: "duplicate-identity"},
		{Native: b, Status: "exact", Value: &zero},
		{Native: c, Status: "withheld", Reason: "precision-loss"},
	}
	// All 24 permutations, with independently authored expected envelopes.
	rows := []Input{a, b, c, d}
	var permute func(int)
	permute = func(start int) {
		if start == len(rows) {
			if got := Validate(rows); !reflect.DeepEqual(got, want) {
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
	if got := Validate(nil); got == nil || len(got) != 0 {
		t.Fatal("empty selection should return explicit empty results")
	}
}

func TestDetachment(t *testing.T) {
	in := sample()
	a := Validate([]Input{in})
	b := Validate([]Input{in})
	in.Evidence[0], in.Accuracy[0] = 'X', 'X'
	if string(a[0].Native.Evidence) != "synthetic evidence" || string(a[0].Native.Accuracy) != "caller accuracy: unresolved schema" {
		t.Fatal("caller mutation reached result")
	}
	a[0].Native.Evidence[0], a[0].Native.Accuracy[0], *a[0].Value = 'Y', 'Y', 99
	if string(b[0].Native.Evidence) != "synthetic evidence" || string(b[0].Native.Accuracy) != "caller accuracy: unresolved schema" || *b[0].Value != -250 {
		t.Fatal("result mutation reached independent result")
	}
	// Shared input backing storage must not create aliases between output rows.
	other := in
	other.Key.Phase = "b"
	results := Validate([]Input{in, other})
	results[0].Native.Evidence[0], results[0].Native.Accuracy[0] = 'Z', 'Z'
	if results[1].Native.Evidence[0] != 'X' || results[1].Native.Accuracy[0] != 'X' {
		t.Fatal("sibling result alias")
	}
	for _, availability := range []string{"unavailable", "unknown"} {
		row := sample()
		row.Availability, row.Decimal = availability, ""
		r := Validate([]Input{row})[0]
		row.Evidence[0], row.Accuracy[0] = 'X', 'X'
		if string(r.Native.Evidence) != "synthetic evidence" || string(r.Native.Accuracy) != "caller accuracy: unresolved schema" {
			t.Fatal("non-exact result evidence aliases input")
		}
	}
}

func TestRejectedOrderingAndTupleBoundaries(t *testing.T) {
	a, b := sample(), sample()
	a.Decimal, b.Decimal = string([]byte{255}), string([]byte{254})
	// Rejected bytes must not normalize to the same replacement character.
	want := []Result{
		{Native: b, Status: "withheld", Reason: "duplicate-identity"},
		{Native: a, Status: "withheld", Reason: "duplicate-identity"},
	}
	for _, rows := range [][]Input{{a, b}, {b, a}} {
		if got := Validate(rows); !reflect.DeepEqual(got, want) {
			t.Fatalf("rejected bytes lost deterministic order: %+v", got)
		}
	}
	a, b = sample(), sample()
	a.Key.Owner, a.Key.Profile = "a/b", "c"
	b.Key.Owner, b.Key.Profile = "a", "b/c"
	got := Validate([]Input{a, b})
	if len(got) != 2 || got[0].Status != "exact" || got[1].Status != "exact" || got[0].Native.Key != b.Key || got[1].Native.Key != a.Key {
		t.Fatalf("delimiter concatenation collapsed tuple: %+v", got)
	}
}
