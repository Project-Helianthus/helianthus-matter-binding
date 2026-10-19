package bridge

import (
	"reflect"
	"testing"
)

func fixture() Association {
	return Association{Node: "synthetic-node", Owner: "synthetic-native", Profile: "electrical-family", Capability: "service-a.power", Composition: "review-composition-a", Binding: "opaque-a", Slot: 1}
}

func has(r Result, code string) bool {
	for _, i := range r.Issues {
		if i.Code == code {
			return true
		}
	}
	return false
}
func withheld(t *testing.T, r Result, code string) {
	t.Helper()
	if !has(r, code) || r.Candidate != nil || r.Graph != nil {
		t.Fatalf("want withheld %s, got %+v", code, r)
	}
}

func TestValidDetachedAndPermutation(t *testing.T) {
	a := fixture()
	b := a
	b.Capability = "service-b.power"
	b.Binding = "opaque-b"
	b.Slot = 2
	old := []Association{a}
	proposed := []Association{b, a}
	edges := []Edge{{a.Node, a.Binding, b.Binding}}
	r := Review(old, proposed, edges)
	want := Result{Retained: []Association{a}, Candidate: []Association{a, b}, Graph: []Edge{{"synthetic-node", "opaque-a", "opaque-b"}}}
	if !reflect.DeepEqual(r, want) {
		t.Fatalf("got %+v want %+v", r, want)
	}
	if other := Review(old, []Association{a, b}, edges); !reflect.DeepEqual(other, want) {
		t.Fatalf("permuted: %+v", other)
	}
	old[0].Owner = "changed"
	proposed[0].Binding = "changed"
	edges[0].Child = "changed"
	if !reflect.DeepEqual(r, want) {
		t.Fatal("input mutation leaked into result")
	}
	r.Candidate[0].Owner = "output-change"
	r.Retained[0].Owner = "retained-change"
	if old[0].Owner != "changed" || proposed[1].Owner != "synthetic-native" {
		t.Fatal("output mutation leaked to caller")
	}
	if r := Review(nil, nil, nil); len(r.Issues) != 0 || len(r.Candidate) != 0 {
		t.Fatalf("empty review: %+v", r)
	}
}

func TestEveryIdentityAndValueAxis(t *testing.T) {
	fields := []struct {
		name string
		set  func(*Association, string)
	}{
		{"node", func(a *Association, s string) { a.Node = s }},
		{"owner", func(a *Association, s string) { a.Owner = s }},
		{"profile", func(a *Association, s string) { a.Profile = s }},
		{"capability", func(a *Association, s string) { a.Capability = s }},
		{"binding", func(a *Association, s string) { a.Binding = s }},
		{"composition", func(a *Association, s string) { a.Composition = s }},
	}
	for _, f := range fields {
		for _, value := range []string{"", " ", " padded", "padded ", "bad\nvalue", "bad\rvalue", "bad\tvalue", "bad\x00value"} {
			t.Run(f.name+"/"+value, func(t *testing.T) {
				a := fixture()
				f.set(&a, value)
				code := "identity_invalid"
				if f.name == "composition" {
					code = "composition_unknown"
				}
				withheld(t, Review(nil, []Association{a}, nil), code)
			})
		}
	}
	for _, slot := range []int64{0, -1} {
		a := fixture()
		a.Slot = slot
		withheld(t, Review(nil, []Association{a}, nil), "slot_invalid")
	}
}

// These variants kill dropping each individual source-key axis. All other
// source-key fields stay equal; distinct bindings/slots avoid incidental aliases.
func TestSourceKeyAxes(t *testing.T) {
	for _, axis := range []string{"node", "owner", "profile", "capability"} {
		t.Run(axis, func(t *testing.T) {
			a := fixture()
			b := a
			b.Binding = "opaque-b"
			b.Slot = 2
			switch axis {
			case "node":
				b.Node = "synthetic-node-b"
			case "owner":
				b.Owner = "synthetic-native-b"
			case "profile":
				b.Profile = "electrical-family-b"
			case "capability":
				b.Capability = "service-b.power"
			}
			want := []Association{a, b}
			r := Review(nil, []Association{b, a}, nil)
			if len(r.Issues) != 0 || !reflect.DeepEqual(r.Candidate, want) {
				t.Fatalf("lost %s axis: %+v", axis, r)
			}
			// Replacing the old source rather than adding it must lose history.
			withheld(t, Review([]Association{a}, []Association{b}, nil), "history_lost")
		})
	}
}

func TestAliasesAndNodeScopedKeys(t *testing.T) {
	a := fixture()
	b := a
	b.Node = "synthetic-node-b"
	if r := Review(nil, []Association{a, b}, nil); len(r.Issues) != 0 || len(r.Candidate) != 2 {
		t.Fatalf("node scoping: %+v", r)
	}
	b = a
	b.Capability = "service-b.power"
	b.Slot = 2
	withheld(t, Review(nil, []Association{a, b}, nil), "binding_alias")
	b.Binding = "opaque-b"
	b.Slot = 1
	withheld(t, Review(nil, []Association{a, b}, nil), "slot_alias")
	b = a
	b.Binding = "opaque-b"
	b.Slot = 2
	withheld(t, Review(nil, []Association{a, b}, nil), "source_alias")
	b.Composition = "review-composition-b"
	r := Review([]Association{a}, []Association{a, b}, nil)
	other := Review([]Association{a}, []Association{b, a}, nil)
	if !reflect.DeepEqual(r, other) {
		t.Fatalf("ambiguous history lookup: %+v / %+v", r, other)
	}
}

func TestHistoryLossChangeRetirementAndUnknown(t *testing.T) {
	a := fixture()
	for _, tc := range []struct {
		name, code string
		change     func(*Association)
	}{
		{"composition", "composition_changed", func(a *Association) { a.Composition = "review-composition-b" }},
		{"binding", "association_changed", func(a *Association) { a.Binding = "opaque-b" }},
		{"slot", "association_changed", func(a *Association) { a.Slot = 2 }},
		{"unknown", "composition_unknown", func(a *Association) { a.Composition = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := a
			tc.change(&b)
			r := Review([]Association{a}, []Association{b}, nil)
			withheld(t, r, tc.code)
			if !reflect.DeepEqual(r.Retained, []Association{a}) {
				t.Fatal("lost retained evidence")
			}
		})
	}
	withheld(t, Review([]Association{a}, nil, nil), "history_lost")
	retired := a
	retired.Retired = true
	want := []Association{retired}
	if r := Review([]Association{a}, want, nil); len(r.Issues) != 0 || !reflect.DeepEqual(r.Candidate, want) {
		t.Fatalf("retirement: %+v", r)
	}
	if r := Review(want, want, nil); len(r.Issues) != 0 {
		t.Fatalf("retired preservation: %+v", r)
	}
	withheld(t, Review(want, []Association{a}, nil), "retirement_reversed")
	b := a
	b.Capability = "service-b.power"
	b.Binding = "opaque-b"
	withheld(t, Review(want, []Association{retired, b}, nil), "slot_alias")
	withheld(t, Review(want, []Association{b}, nil), "history_lost")
	// Invalid history is retained as supplied, never called accepted history.
	for _, tc := range []struct {
		code    string
		records []Association
	}{
		{"history_source_alias", []Association{a, a}},
		{"history_identity_invalid", []Association{{}}},
		{"history_composition_unknown", []Association{{}}},
		{"history_slot_invalid", []Association{{}}},
		{"history_binding_alias", []Association{a, a}},
		{"history_slot_alias", []Association{a, a}},
	} {
		withheld(t, Review(tc.records, []Association{a}, nil), tc.code)
	}
}

func TestGraphGuards(t *testing.T) {
	a := fixture()
	b := a
	b.Binding = "opaque-b"
	b.Capability = "service-b.power"
	b.Slot = 2
	c := b
	c.Binding = "opaque-c"
	c.Capability = "service-c.power"
	c.Slot = 3
	records := []Association{a, b, c}
	for _, tc := range []struct {
		name, code string
		edges      []Edge
	}{
		{"node", "edge_unknown", []Edge{{"other-node", a.Binding, b.Binding}}},
		{"parent", "edge_unknown", []Edge{{a.Node, "absent", b.Binding}}},
		{"child", "edge_unknown", []Edge{{a.Node, a.Binding, "absent"}}},
		{"blank-node", "edge_identity_invalid", []Edge{{"", a.Binding, b.Binding}}},
		{"blank-parent", "edge_identity_invalid", []Edge{{a.Node, "", b.Binding}}},
		{"blank-child", "edge_identity_invalid", []Edge{{a.Node, a.Binding, ""}}},
		{"duplicate", "edge_duplicate", []Edge{{a.Node, a.Binding, b.Binding}, {a.Node, a.Binding, b.Binding}}},
		{"parents", "multiple_parents", []Edge{{a.Node, a.Binding, c.Binding}, {a.Node, b.Binding, c.Binding}}},
		{"self", "graph_cycle", []Edge{{a.Node, a.Binding, a.Binding}}},
		{"cycle", "graph_cycle", []Edge{{a.Node, a.Binding, b.Binding}, {a.Node, b.Binding, c.Binding}, {a.Node, c.Binding, a.Binding}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Review(nil, records, tc.edges)
			withheld(t, r, tc.code)
			reversed := append([]Edge(nil), tc.edges...)
			for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
				reversed[i], reversed[j] = reversed[j], reversed[i]
			}
			if other := Review(nil, []Association{c, b, a}, reversed); !reflect.DeepEqual(r, other) {
				t.Fatalf("nondeterministic: %+v / %+v", r, other)
			}
		})
	}
	for _, index := range []int{0, 1} {
		copyRecords := append([]Association(nil), records...)
		copyRecords[index].Retired = true
		withheld(t, Review(nil, copyRecords, []Edge{{a.Node, a.Binding, b.Binding}}), "edge_retired")
	}
	edges := []Edge{{a.Node, b.Binding, c.Binding}, {a.Node, a.Binding, b.Binding}}
	r := Review(nil, records, edges)
	want := []Edge{{"synthetic-node", "opaque-a", "opaque-b"}, {"synthetic-node", "opaque-b", "opaque-c"}}
	if len(r.Issues) != 0 || !reflect.DeepEqual(r.Graph, want) {
		t.Fatalf("valid chain: %+v", r)
	}
	if other := Review(nil, []Association{c, b, a}, []Edge{edges[1], edges[0]}); !reflect.DeepEqual(other, r) {
		t.Fatal("valid graph permutation")
	}
}

// Edge-key mutations dropping Node, Parent or Child must fail these independent
// expected forests/diagnostics. Endpoint lookup mutations dropping any of those
// fields are additionally caught by TestGraphGuards' unknown-reference cases.
func TestEdgeKeyAxes(t *testing.T) {
	a := fixture()
	b := a
	b.Binding = "opaque-b"
	b.Capability = "service-b.power"
	b.Slot = 2
	c := a
	c.Binding = "opaque-c"
	c.Capability = "service-c.power"
	c.Slot = 3
	star := []Edge{{"synthetic-node", "opaque-a", "opaque-b"}, {"synthetic-node", "opaque-a", "opaque-c"}}
	r := Review(nil, []Association{a, b, c}, star)
	if len(r.Issues) != 0 || !reflect.DeepEqual(r.Graph, star) {
		t.Fatalf("child edge axis: %+v", r)
	}
	otherA, otherB := a, b
	otherA.Node = "synthetic-node-b"
	otherB.Node = "synthetic-node-b"
	pair := []Edge{{"synthetic-node", "opaque-a", "opaque-b"}, {"synthetic-node-b", "opaque-a", "opaque-b"}}
	r = Review(nil, []Association{a, b, otherA, otherB}, pair)
	if len(r.Issues) != 0 || !reflect.DeepEqual(r.Graph, pair) {
		t.Fatalf("node edge axis: %+v", r)
	}
	r = Review(nil, []Association{a, b, c}, []Edge{{a.Node, a.Binding, c.Binding}, {a.Node, b.Binding, c.Binding}})
	want := []Issue{{"multiple_parents", "synthetic-node", "opaque-c"}}
	if !reflect.DeepEqual(r.Issues, want) || r.Candidate != nil || r.Graph != nil {
		t.Fatalf("parent edge axis: %+v", r)
	}
}
