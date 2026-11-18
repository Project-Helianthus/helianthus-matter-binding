package topology

import (
	"reflect"
	"testing"
)

func node(id, owner, profile, phase, endpoint string) Node {
	return Node{ResourceID: id, Owner: owner, Profile: profile, Phase: phase, Endpoint: endpoint}
}

func key(from, to string) RelationKey {
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

func graphOf(nodes []Node, instances []Instance, keys ...RelationKey) Graph {
	edges := make([]Edge, 0, len(keys))
	for _, k := range keys {
		edges = append(edges, Edge{Cited: cite(k)})
	}
	return Graph{
		Nodes:     append([]Node(nil), nodes...),
		Instances: append([]Instance(nil), instances...),
		Edges:     edges,
	}
}

func expect(t *testing.T, g Graph, want Result) {
	t.Helper()
	if got := Validate(g); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v candidate=%+v; want %#v candidate=%+v", got, got.Candidate, want, want.Candidate)
	}
}

func rejected(status Status, codes ...string) Result {
	return Result{Status: status, Problems: codes}
}

func meterNodes() []Node {
	return []Node{
		node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a"),
		node("resource:meter:b", "synthetic-owner", "meter-family", "phase-b", "endpoint-b"),
		node("resource:meter:c", "synthetic-owner", "meter-family", "phase-c", "endpoint-c"),
	}
}

func TestResourceIDIsTheNodeKey(t *testing.T) {
	// ResourceID is the only node-key field. Owner, profile and phase are not.
	// Including owner, profile or phase in the key would accept the duplicate
	// cases. Dropping ResourceID would refuse the distinct-resource case.
	a := node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
	for _, tc := range []struct {
		name string
		edit func(*Node)
	}{
		{"owner", func(n *Node) { n.Owner = "other-owner" }},
		{"profile", func(n *Node) { n.Profile = "other-family" }},
		{"phase", func(n *Node) { n.Phase = "phase-z" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := a
			tc.edit(&other)
			g := Graph{Nodes: []Node{a, other}}
			expect(t, g, rejected(Invalid, "duplicate_node"))
		})
	}
	b := node("resource:meter:b", "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
	want := Graph{Nodes: []Node{a, b}}
	expect(t, Graph{Nodes: []Node{b, a}}, Result{Status: Valid, Candidate: &want})
}

func TestNativeTupleRefused(t *testing.T) {
	// A native (owner, profile, scope, id) tuple is not a node key.
	// Each case empties one tuple field, or supplies the tuple beside a
	// resource id. Skipping the refusal when that field is empty, or when a
	// resource id is also present, would admit the node.
	full := NativeTuple{Owner: "synthetic-owner", Profile: "meter-family", Scope: "service-a", ID: "local-a"}
	cases := []struct {
		name     string
		native   NativeTuple
		resource string
	}{
		{"owner", NativeTuple{Profile: "meter-family", Scope: "service-a", ID: "local-a"}, "resource:meter:a"},
		{"profile", NativeTuple{Owner: "synthetic-owner", Scope: "service-a", ID: "local-a"}, "resource:meter:a"},
		{"scope", NativeTuple{Owner: "synthetic-owner", Profile: "meter-family", ID: "local-a"}, "resource:meter:a"},
		{"id", NativeTuple{Owner: "synthetic-owner", Profile: "meter-family", Scope: "service-a"}, "resource:meter:a"},
		{"complete", full, ""},
		{"beside-resource", full, "resource:meter:a"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := node(tc.resource, "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
			tuple := tc.native
			n.Native = &tuple
			expect(t, Graph{Nodes: []Node{n}}, rejected(Invalid, "native_tuple_node"))
		})
	}
}

func TestRelationKeyFields(t *testing.T) {
	// The citation matches an instance only when every relation-key field
	// agrees. One coherent instance is published. Changing the named field
	// of the citation must be uncited_edge: a lookup that ignored the field
	// would still find that instance and accept the edge.
	// A second coherent instance that differs only in the named field must
	// stay distinct: duplicate detection that ignored the field would report
	// duplicate_relation or duplicate_edge.
	base := key("resource:meter:a", "resource:meter:b")
	nodes := []Node{
		node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a"),
		node("resource:meter:b", "synthetic-owner", "meter-family", "phase-b", "endpoint-b"),
		node("resource:meter:z", "synthetic-owner", "meter-family", "phase-z", "endpoint-z"),
	}
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
			g := graphOf(nodes, []Instance{published(base, "coherent", "pack")}, changed)
			expect(t, g, rejected(Invalid, "uncited_edge"))
			both := graphOf(nodes,
				[]Instance{published(base, "coherent", "pack"), published(changed, "coherent", "pack")},
				changed, base)
			got := Validate(both)
			if got.Status != Valid || got.Candidate == nil || len(got.Candidate.Edges) != 2 || len(got.Candidate.Instances) != 2 {
				t.Fatalf("distinct %s keys merged: %#v", field.name, got)
			}
		})
	}
}

func TestIncoherentRelationRefused(t *testing.T) {
	// Only the coherent value admits the cited instance. Treating any listed
	// value as coherent would accept the edge.
	k := key("resource:meter:a", "resource:meter:b")
	nodes := meterNodes()[:2]
	for _, coherence := range []string{"stale", "dangling", "outside_cut", "excluded", "conflict"} {
		t.Run(coherence, func(t *testing.T) {
			g := graphOf(nodes, []Instance{published(k, coherence, "pack")}, k)
			expect(t, g, rejected(Invalid, "incoherent_relation"))
		})
	}
}

func TestCoreCrossSourceEdge(t *testing.T) {
	// Owner and profile are provenance. A coherent edge whose ends differ in
	// the named field is cross-source. The core publication is accepted; a
	// pack publication is cross_source_edge. Rejecting every cross-source
	// edge, accepting every cross-source edge, or comparing only the other
	// field fails the matching case.
	base := node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
	for _, tc := range []struct {
		name string
		edit func(*Node)
	}{
		{"owner", func(n *Node) { n.Owner = "other-owner" }},
		{"profile", func(n *Node) { n.Profile = "circuit-family" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := node("resource:circuit:b", "synthetic-owner", "meter-family", "phase-b", "endpoint-b")
			tc.edit(&other)
			k := key(base.ResourceID, other.ResourceID)
			core := graphOf([]Node{other, base}, []Instance{published(k, "coherent", "core")}, k)
			got := Validate(core)
			if got.Status != Valid || got.Candidate == nil || len(got.Candidate.Edges) != 1 {
				t.Fatalf("core cross-source %s: %#v", tc.name, got)
			}
			pack := graphOf([]Node{base, other}, []Instance{published(k, "coherent", "pack")}, k)
			expect(t, pack, rejected(Invalid, "cross_source_edge"))
		})
	}
}

func TestSameProvenancePackEdge(t *testing.T) {
	// A pack publication is admitted when provenance agrees. Requiring the
	// core publisher for every edge would refuse this graph.
	nodes := meterNodes()
	ab := key("resource:meter:a", "resource:meter:b")
	ac := key("resource:meter:a", "resource:meter:c")
	bc := key("resource:meter:b", "resource:meter:c")
	instances := []Instance{published(ac, "coherent", "pack"), published(ab, "coherent", "pack")}
	want := Graph{
		Nodes: []Node{
			node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a"),
			node("resource:meter:b", "synthetic-owner", "meter-family", "phase-b", "endpoint-b"),
			node("resource:meter:c", "synthetic-owner", "meter-family", "phase-c", "endpoint-c"),
		},
		Instances: []Instance{published(ab, "coherent", "pack"), published(ac, "coherent", "pack")},
		Edges:     []Edge{{Cited: cite(ab)}, {Cited: cite(ac)}},
	}
	expect(t, graphOf(nodes, instances, ac, ab), Result{Status: Valid, Candidate: &want})
	instances = append(instances, published(bc, "coherent", "pack"))
	want.Instances = append(want.Instances, published(bc, "coherent", "pack"))
	want.Edges = append(want.Edges, Edge{Cited: cite(bc)})
	expect(t, graphOf(nodes, instances, ac, ab, bc), Result{Status: Valid, Candidate: &want})
}

func TestPhaseAssociationFollowsProvenance(t *testing.T) {
	// Phase and endpoint labels associate within one provenance. Scoping that
	// association by resource id would accept the conflict. Dropping owner or
	// profile from the scope would refuse the other two graphs.
	a := node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
	conflict := node("resource:meter:b", "synthetic-owner", "meter-family", "phase-a", "endpoint-b")
	expect(t, Graph{Nodes: []Node{a, conflict}}, rejected(Invalid, "phase_endpoint_conflict"))
	for _, tc := range []struct {
		name string
		edit func(*Node)
	}{
		{"owner", func(n *Node) { n.Owner = "other-owner" }},
		{"profile", func(n *Node) { n.Profile = "circuit-family" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := node("resource:circuit:b", "synthetic-owner", "meter-family", "phase-a", "endpoint-b")
			tc.edit(&other)
			got := Validate(Graph{Nodes: []Node{other, a}})
			if got.Status != Valid || got.Candidate == nil || len(got.Candidate.Nodes) != 2 {
				t.Fatalf("provenance %s: %#v", tc.name, got)
			}
		})
	}
}

func TestGraphGuards(t *testing.T) {
	nodes := meterNodes()
	ab := key("resource:meter:a", "resource:meter:b")
	ac := key("resource:meter:a", "resource:meter:c")
	bc := key("resource:meter:b", "resource:meter:c")
	cb := key("resource:meter:c", "resource:meter:b")
	ca := key("resource:meter:c", "resource:meter:a")
	loop := key("resource:meter:b", "resource:meter:b")
	instances := []Instance{
		published(ab, "coherent", "pack"), published(ac, "coherent", "pack"),
		published(bc, "coherent", "pack"), published(cb, "coherent", "pack"),
		published(ca, "coherent", "pack"), published(loop, "coherent", "pack"),
	}
	cases := []struct {
		name string
		g    Graph
		want Result
	}{
		{"duplicate-edge", graphOf(nodes, instances, ab, ab), rejected(Invalid, "duplicate_edge")},
		{"duplicate-relation", Graph{
			Nodes:     nodes,
			Instances: []Instance{published(ab, "coherent", "pack"), published(ab, "coherent", "pack")},
			Edges:     []Edge{{Cited: cite(ab)}},
		}, rejected(Invalid, "duplicate_relation")},
		{"self-loop", graphOf(nodes, instances, loop), rejected(Invalid, "cycle")},
		{"cycle", graphOf(nodes, instances, ab, bc, ca), rejected(Invalid, "cycle")},
		{"disconnected-cycle", graphOf(nodes, instances, bc, cb), rejected(Invalid, "cycle")},
		{"dangling", graphOf(nodes, []Instance{published(key("resource:meter:a", "resource:meter:missing"), "coherent", "pack")}, key("resource:meter:a", "resource:meter:missing")), rejected(Invalid, "dangling_reference")},
		{"unknown-phase", func() Graph {
			g := graphOf(nodes, nil)
			g.Nodes[1].Phase = ""
			return g
		}(), rejected(Withheld, "unknown_phase")},
		{"unknown-endpoint", func() Graph {
			g := graphOf(nodes, nil)
			g.Nodes[1].Endpoint = ""
			return g
		}(), rejected(Withheld, "unknown_endpoint")},
		{"endpoint-conflict", func() Graph {
			g := graphOf(nodes, nil)
			g.Nodes[1].Endpoint = "endpoint-a"
			return g
		}(), rejected(Invalid, "endpoint_phase_conflict")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { expect(t, tc.g, tc.want) })
	}
	// The uncited guard above uses no edges. An edge value with no citation
	// is the same refusal. A missing citation check would accept it.
	g := graphOf(nodes, []Instance{published(ab, "coherent", "pack")})
	g.Edges = []Edge{{}}
	expect(t, g, rejected(Invalid, "uncited_edge"))
}

func TestMalformedLabels(t *testing.T) {
	// Each value must stay invalid for the named field. Dropping that field's
	// label guard would admit it.
	bad := map[string]string{"empty": "", "space": " ", "padding": " x", "control": "x\x00", "utf8": "\xff"}
	for name, value := range bad {
		t.Run("resource/"+name, func(t *testing.T) {
			n := node(value, "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
			expect(t, Graph{Nodes: []Node{n}}, rejected(Invalid, "invalid_node_identity"))
		})
		for _, field := range []string{"owner", "profile"} {
			t.Run(field+"/"+name, func(t *testing.T) {
				n := node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
				if field == "owner" {
					n.Owner = value
				} else {
					n.Profile = value
				}
				expect(t, Graph{Nodes: []Node{n}}, rejected(Invalid, "invalid_provenance"))
			})
		}
		for _, field := range []string{"phase", "endpoint"} {
			if value == "" {
				continue
			}
			t.Run(field+"/"+name, func(t *testing.T) {
				n := node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a")
				if field == "phase" {
					n.Phase = value
				} else {
					n.Endpoint = value
				}
				expect(t, Graph{Nodes: []Node{n}}, rejected(Invalid, "invalid_"+field))
			})
		}
	}
}

func TestWithheldEmpty(t *testing.T) {
	expect(t, Graph{}, rejected(Withheld, "unknown_nodes"))
}

func TestPermutationAndDetachment(t *testing.T) {
	nodes := meterNodes()
	ab := key("resource:meter:a", "resource:meter:b")
	ac := key("resource:meter:a", "resource:meter:c")
	instances := []Instance{published(ab, "coherent", "pack"), published(ac, "coherent", "pack")}
	want := Graph{
		Nodes: []Node{
			node("resource:meter:a", "synthetic-owner", "meter-family", "phase-a", "endpoint-a"),
			node("resource:meter:b", "synthetic-owner", "meter-family", "phase-b", "endpoint-b"),
			node("resource:meter:c", "synthetic-owner", "meter-family", "phase-c", "endpoint-c"),
		},
		Instances: instances,
		Edges:     []Edge{{Cited: cite(ab)}, {Cited: cite(ac)}},
	}
	for _, p := range [][3]int{{0, 1, 2}, {2, 0, 1}, {1, 2, 0}} {
		g := graphOf([]Node{nodes[p[0]], nodes[p[1]], nodes[p[2]]}, []Instance{instances[1], instances[0]}, ac, ab)
		expect(t, g, Result{Status: Valid, Candidate: &want})
	}
	g := graphOf(nodes, instances, ab, ac)
	r := Validate(g)
	if r.Candidate == nil {
		t.Fatal(r)
	}
	g.Nodes[0].Owner = "changed"
	g.Edges[0].Cited.From = "changed"
	g.Instances[0].Publisher = "changed"
	if !reflect.DeepEqual(*r.Candidate, want) {
		t.Fatal("input mutation changed candidate")
	}
	g = graphOf(nodes, instances, ab, ac)
	r = Validate(g)
	r.Candidate.Nodes[0].Owner = "changed"
	r.Candidate.Edges[0].Cited.To = "changed"
	r.Candidate.Instances[0].Coherence = "changed"
	fresh := graphOf(nodes, instances, ab, ac)
	if !reflect.DeepEqual(g, fresh) {
		t.Fatal("output mutation changed input")
	}
	held := nodes
	held[1].Phase = ""
	before := append([]Node(nil), held...)
	expect(t, Graph{Nodes: held}, rejected(Withheld, "unknown_phase"))
	if !reflect.DeepEqual(held, before) {
		t.Fatal("withholding changed retained facts")
	}
}

func TestProblemOrder(t *testing.T) {
	nodes := meterNodes()
	nodes[0].Phase = ""
	nodes[1].Endpoint = ""
	ab := key("resource:meter:a", "resource:meter:b")
	g := graphOf(nodes, []Instance{published(ab, "coherent", "pack")}, ab, ab)
	expect(t, g, rejected(Invalid, "duplicate_edge", "unknown_endpoint", "unknown_phase"))
}
