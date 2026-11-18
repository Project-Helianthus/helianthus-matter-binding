package bridge

import (
	"reflect"
	"testing"
)

func fixture() Association {
	return Association{
		Node: "synthetic-node", ResourceID: "resource:meter:a", Owner: "synthetic-native",
		Profile: "electrical-family", Capability: "service-a.power", Composition: "review-composition-a",
		Binding: "opaque-a", Slot: 1,
	}
}

func relate(from, to string) RelationKey {
	return RelationKey{
		PackID: "pack-infrastructure", PackVersion: "mark-a",
		Definition: "contains", Version: "mark-a", From: from, To: to,
	}
}

func cite(k RelationKey) *RelationKey {
	c := k
	return &c
}

func published(k RelationKey, coherence, publisher string) Instance {
	return Instance{Key: k, Coherence: coherence, Publisher: publisher}
}

func linked(parent, child Association, publisher string) (Edge, Instance) {
	k := relate(parent.ResourceID, child.ResourceID)
	return Edge{Node: parent.Node, Parent: parent.Binding, Child: child.Binding, Cited: cite(k)},
		published(k, "coherent", publisher)
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
	b.ResourceID = "resource:meter:b"
	b.Capability = "service-b.power"
	b.Binding = "opaque-b"
	b.Slot = 2
	edge, instance := linked(a, b, "pack")
	old := []Association{a}
	proposed := []Association{b, a}
	r := Review(old, proposed, []Edge{edge}, []Instance{instance})
	wantEdge, _ := linked(a, b, "pack")
	want := Result{Retained: []Association{a}, Candidate: []Association{a, b}, Graph: []Edge{wantEdge}}
	if !reflect.DeepEqual(r, want) {
		t.Fatalf("got %+v want %+v", r, want)
	}
	if other := Review(old, []Association{a, b}, []Edge{edge}, []Instance{instance}); !reflect.DeepEqual(other, want) {
		t.Fatalf("permuted: %+v", other)
	}
	old[0].Owner = "changed"
	proposed[0].Binding = "changed"
	edge.Cited.From = "changed"
	if !reflect.DeepEqual(r, want) {
		t.Fatal("input mutation leaked into result")
	}
	r.Candidate[0].Owner = "output-change"
	r.Retained[0].Owner = "retained-change"
	r.Graph[0].Cited.To = "output-change"
	if old[0].Owner != "changed" || proposed[1].Owner != "synthetic-native" || edge.Cited.To != b.ResourceID {
		t.Fatal("output mutation leaked to caller")
	}
	if r := Review(nil, nil, nil, nil); len(r.Issues) != 0 || len(r.Candidate) != 0 {
		t.Fatalf("empty review: %+v", r)
	}
}

func TestEveryIdentityAndValueAxis(t *testing.T) {
	fields := []struct {
		name string
		set  func(*Association, string)
	}{
		{"node", func(a *Association, s string) { a.Node = s }},
		{"resource", func(a *Association, s string) { a.ResourceID = s }},
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
				withheld(t, Review(nil, []Association{a}, nil, nil), code)
			})
		}
	}
	for _, slot := range []int64{0, -1} {
		a := fixture()
		a.Slot = slot
		withheld(t, Review(nil, []Association{a}, nil, nil), "slot_invalid")
	}
}

func TestNativeTupleRefused(t *testing.T) {
	// Owner, profile, scope and id are not an association key. Each case
	// empties one tuple field or sets the tuple beside a resource id. Skipping
	// the refusal when that field is empty, or when a resource id is present,
	// would admit the association.
	full := NativeTuple{Owner: "synthetic-native", Profile: "electrical-family", Scope: "service-a", ID: "local-a"}
	cases := []struct {
		name   string
		native NativeTuple
	}{
		{"owner", NativeTuple{Profile: "electrical-family", Scope: "service-a", ID: "local-a"}},
		{"profile", NativeTuple{Owner: "synthetic-native", Scope: "service-a", ID: "local-a"}},
		{"scope", NativeTuple{Owner: "synthetic-native", Profile: "electrical-family", ID: "local-a"}},
		{"id", NativeTuple{Owner: "synthetic-native", Profile: "electrical-family", Scope: "service-a"}},
		{"complete", full},
		{"beside-resource", full},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture()
			tuple := tc.native
			a.Native = &tuple
			withheld(t, Review(nil, []Association{a}, nil, nil), "native_tuple_node")
		})
	}
}

func TestResourceKeyNotProvenance(t *testing.T) {
	// ResourceID is the association key. Owner, profile and capability are not.
	// Including one of those fields would make the changed record a new source
	// and report history_lost. Dropping ResourceID would make the second record
	// a source_alias of the first.
	a := fixture()
	for _, tc := range []struct {
		name string
		edit func(*Association)
	}{
		{"owner", func(b *Association) { b.Owner = "synthetic-native-b" }},
		{"profile", func(b *Association) { b.Profile = "electrical-family-b" }},
		{"capability", func(b *Association) { b.Capability = "service-b.power" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := a
			tc.edit(&b)
			code := "provenance_changed"
			if tc.name == "capability" {
				code = "association_changed"
			}
			withheld(t, Review([]Association{a}, []Association{b}, nil, nil), code)
			if has(Review([]Association{a}, []Association{b}, nil, nil), "history_lost") {
				t.Fatal("provenance or capability split the resource key")
			}
		})
	}
	b := a
	b.ResourceID = "resource:meter:b"
	b.Binding = "opaque-b"
	b.Slot = 2
	r := Review(nil, []Association{b, a}, nil, nil)
	if len(r.Issues) != 0 || !reflect.DeepEqual(r.Candidate, []Association{a, b}) {
		t.Fatalf("resource key conflated: %+v", r)
	}
	withheld(t, Review([]Association{a}, []Association{b}, nil, nil), "history_lost")
	same := a
	same.Owner = "synthetic-native-b"
	same.Capability = "service-b.power"
	same.Binding = "opaque-b"
	same.Slot = 2
	withheld(t, Review(nil, []Association{a, same}, nil, nil), "source_alias")
}

func TestNodeScopedBindings(t *testing.T) {
	a := fixture()
	b := a
	b.Node = "synthetic-node-b"
	b.ResourceID = "resource:meter:b"
	if r := Review(nil, []Association{a, b}, nil, nil); len(r.Issues) != 0 || len(r.Candidate) != 2 {
		t.Fatalf("distinct resources: %+v", r)
	}
	b = a
	b.Node = "synthetic-node-b"
	withheld(t, Review(nil, []Association{a, b}, nil, nil), "source_alias")
	b = a
	b.ResourceID = "resource:meter:b"
	b.Slot = 2
	withheld(t, Review(nil, []Association{a, b}, nil, nil), "binding_alias")
	b.Binding = "opaque-b"
	b.Slot = 1
	withheld(t, Review(nil, []Association{a, b}, nil, nil), "slot_alias")
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
		{"node", "association_changed", func(a *Association) { a.Node = "synthetic-node-b" }},
		{"unknown", "composition_unknown", func(a *Association) { a.Composition = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := a
			tc.change(&b)
			r := Review([]Association{a}, []Association{b}, nil, nil)
			withheld(t, r, tc.code)
			if !reflect.DeepEqual(r.Retained, []Association{a}) {
				t.Fatal("lost retained evidence")
			}
		})
	}
	withheld(t, Review([]Association{a}, nil, nil, nil), "history_lost")
	retired := a
	retired.Retired = true
	want := []Association{retired}
	if r := Review([]Association{a}, want, nil, nil); len(r.Issues) != 0 || !reflect.DeepEqual(r.Candidate, want) {
		t.Fatalf("retirement: %+v", r)
	}
	if r := Review(want, want, nil, nil); len(r.Issues) != 0 {
		t.Fatalf("retired preservation: %+v", r)
	}
	withheld(t, Review(want, []Association{a}, nil, nil), "retirement_reversed")
	b := a
	b.ResourceID = "resource:meter:b"
	b.Capability = "service-b.power"
	b.Binding = "opaque-b"
	withheld(t, Review(want, []Association{retired, b}, nil, nil), "slot_alias")
	withheld(t, Review(want, []Association{b}, nil, nil), "history_lost")
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
		withheld(t, Review(tc.records, []Association{a}, nil, nil), tc.code)
	}
}

func TestGraphGuards(t *testing.T) {
	a := fixture()
	b := a
	b.ResourceID = "resource:meter:b"
	b.Binding = "opaque-b"
	b.Capability = "service-b.power"
	b.Slot = 2
	c := b
	c.ResourceID = "resource:meter:c"
	c.Binding = "opaque-c"
	c.Capability = "service-c.power"
	c.Slot = 3
	records := []Association{a, b, c}
	ab, iab := linked(a, b, "pack")
	bc, ibc := linked(b, c, "pack")
	ca, ica := linked(c, a, "pack")
	self, iself := linked(a, a, "pack")
	ac, iac := linked(a, c, "pack")
	for _, tc := range []struct {
		name, code string
		edges      []Edge
		instances  []Instance
	}{
		{"node", "edge_unknown", []Edge{{Node: "other-node", Parent: a.Binding, Child: b.Binding, Cited: cite(relate(a.ResourceID, b.ResourceID))}}, []Instance{iab}},
		{"parent", "edge_unknown", []Edge{{Node: a.Node, Parent: "absent", Child: b.Binding, Cited: cite(relate(a.ResourceID, b.ResourceID))}}, []Instance{iab}},
		{"child", "edge_unknown", []Edge{{Node: a.Node, Parent: a.Binding, Child: "absent", Cited: cite(relate(a.ResourceID, b.ResourceID))}}, []Instance{iab}},
		{"blank-node", "edge_identity_invalid", []Edge{{Parent: a.Binding, Child: b.Binding, Cited: cite(relate(a.ResourceID, b.ResourceID))}}, []Instance{iab}},
		{"blank-parent", "edge_identity_invalid", []Edge{{Node: a.Node, Child: b.Binding, Cited: cite(relate(a.ResourceID, b.ResourceID))}}, []Instance{iab}},
		{"blank-child", "edge_identity_invalid", []Edge{{Node: a.Node, Parent: a.Binding, Cited: cite(relate(a.ResourceID, b.ResourceID))}}, []Instance{iab}},
		{"duplicate", "edge_duplicate", []Edge{ab, ab}, []Instance{iab}},
		{"parents", "multiple_parents", []Edge{ac, bc}, []Instance{iac, ibc}},
		{"self", "graph_cycle", []Edge{self}, []Instance{iself}},
		{"cycle", "graph_cycle", []Edge{ab, bc, ca}, []Instance{iab, ibc, ica}},
		{"uncited", "uncited_edge", []Edge{{Node: a.Node, Parent: a.Binding, Child: b.Binding}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Review(nil, records, tc.edges, tc.instances)
			withheld(t, r, tc.code)
		})
	}
	for _, index := range []int{0, 1} {
		copyRecords := append([]Association(nil), records...)
		copyRecords[index].Retired = true
		withheld(t, Review(nil, copyRecords, []Edge{ab}, []Instance{iab}), "edge_retired")
	}
	r := Review(nil, records, []Edge{bc, ab}, []Instance{ibc, iab})
	wantAB, _ := linked(a, b, "pack")
	wantBC, _ := linked(b, c, "pack")
	if len(r.Issues) != 0 || !reflect.DeepEqual(r.Graph, []Edge{wantAB, wantBC}) {
		t.Fatalf("valid chain: %+v", r)
	}
}

func TestRelationKeyFields(t *testing.T) {
	// One coherent instance is published between the parent and child resources.
	// Changing the named citation field must be uncited_edge. A lookup that
	// ignored the field would still find the instance. From and To are also
	// compared with the parent and child resource ids: ignoring that comparison
	// accepts a citation of a different resource.
	a := fixture()
	b := a
	b.ResourceID = "resource:meter:b"
	b.Binding = "opaque-b"
	b.Slot = 2
	base := relate(a.ResourceID, b.ResourceID)
	fields := []struct {
		name string
		set  func(*RelationKey)
	}{
		{"pack_id", func(k *RelationKey) { k.PackID = "pack-other" }},
		{"pack_version", func(k *RelationKey) { k.PackVersion = "mark-b" }},
		{"definition", func(k *RelationKey) { k.Definition = "measures" }},
		{"version", func(k *RelationKey) { k.Version = "mark-b" }},
		{"from", func(k *RelationKey) { k.From = "resource:meter:z" }},
		{"to", func(k *RelationKey) { k.To = "resource:meter:z" }},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			changed := base
			field.set(&changed)
			edge := Edge{Node: a.Node, Parent: a.Binding, Child: b.Binding, Cited: cite(changed)}
			withheld(t, Review(nil, []Association{a, b}, []Edge{edge}, []Instance{published(base, "coherent", "pack")}), "uncited_edge")
			// The other instance differs only in the named field and is stored
			// after the coherent one. A last write that ignored the field would
			// keep the incoherent instance and refuse the exact citation.
			incoherent := changed
			r := Review(nil, []Association{a, b}, []Edge{{Node: a.Node, Parent: a.Binding, Child: b.Binding, Cited: cite(base)}},
				[]Instance{published(base, "coherent", "pack"), published(incoherent, "stale", "pack")})
			if len(r.Issues) != 0 || len(r.Graph) != 1 {
				t.Fatalf("distinct %s keys merged: %+v", field.name, r)
			}
		})
	}
}

func TestIncoherentRelationRefused(t *testing.T) {
	// Only the coherent value admits the cited instance. Treating any listed
	// value as coherent would accept the edge.
	a := fixture()
	b := a
	b.ResourceID = "resource:meter:b"
	b.Binding = "opaque-b"
	b.Slot = 2
	k := relate(a.ResourceID, b.ResourceID)
	edge := Edge{Node: a.Node, Parent: a.Binding, Child: b.Binding, Cited: cite(k)}
	for _, coherence := range []string{"stale", "dangling", "outside_cut", "excluded", "conflict"} {
		t.Run(coherence, func(t *testing.T) {
			withheld(t, Review(nil, []Association{a, b}, []Edge{edge}, []Instance{published(k, coherence, "pack")}), "incoherent_relation")
		})
	}
}

func TestCoreCrossSourceEdge(t *testing.T) {
	// Owner and profile are provenance. A coherent edge whose ends differ in
	// the named field is cross-source. The core publication is accepted; a
	// pack publication is cross_source_edge. Rejecting every cross-source
	// edge, accepting every cross-source edge, or comparing only the other
	// field fails the matching case.
	a := fixture()
	for _, tc := range []struct {
		name string
		edit func(*Association)
	}{
		{"owner", func(b *Association) { b.Owner = "other-native" }},
		{"profile", func(b *Association) { b.Profile = "circuit-family" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := a
			b.ResourceID = "resource:circuit:b"
			b.Binding = "opaque-b"
			b.Slot = 2
			tc.edit(&b)
			coreEdge, coreInst := linked(a, b, "core")
			r := Review(nil, []Association{a, b}, []Edge{coreEdge}, []Instance{coreInst})
			if len(r.Issues) != 0 || len(r.Graph) != 1 {
				t.Fatalf("core cross-source %s: %+v", tc.name, r)
			}
			packEdge, packInst := linked(a, b, "pack")
			withheld(t, Review(nil, []Association{b, a}, []Edge{packEdge}, []Instance{packInst}), "cross_source_edge")
		})
	}
}

func TestEdgeEndpointResource(t *testing.T) {
	// The cited instance names the parent and child resources. An instance
	// whose from or to resource is not that association is not this edge.
	// Comparing bindings alone would accept it.
	a := fixture()
	b := a
	b.ResourceID = "resource:meter:b"
	b.Binding = "opaque-b"
	b.Slot = 2
	z := relate("resource:meter:z", b.ResourceID)
	edge := Edge{Node: a.Node, Parent: a.Binding, Child: b.Binding, Cited: cite(z)}
	withheld(t, Review(nil, []Association{a, b}, []Edge{edge}, []Instance{published(z, "coherent", "core")}), "uncited_edge")
	z = relate(a.ResourceID, "resource:meter:z")
	edge.Cited = cite(z)
	withheld(t, Review(nil, []Association{a, b}, []Edge{edge}, []Instance{published(z, "coherent", "core")}), "uncited_edge")
}
