package topology

import (
	"reflect"
	"testing"
)

func source() Source              { return Source{"synthetic-owner", "meter-family", "service-a"} }
func identity(id string) Identity { return Identity{source(), id} }
func fixture() Graph {
	return Graph{
		Sources: []Source{source()},
		Nodes: []Node{
			{identity("a"), "phase-a", "endpoint-a"},
			{identity("b"), "phase-b", "endpoint-b"},
			{identity("c"), "phase-c", "endpoint-c"},
		},
		Edges: []Edge{{identity("a"), identity("b")}, {identity("a"), identity("c")}},
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

func TestValidVectors(t *testing.T) {
	// Expected values are supplied independently, not produced by Validate/sorts.
	want := Graph{
		Sources: []Source{{"synthetic-owner", "meter-family", "service-a"}},
		Nodes: []Node{
			{Identity{Source{"synthetic-owner", "meter-family", "service-a"}, "a"}, "phase-a", "endpoint-a"},
			{Identity{Source{"synthetic-owner", "meter-family", "service-a"}, "b"}, "phase-b", "endpoint-b"},
			{Identity{Source{"synthetic-owner", "meter-family", "service-a"}, "c"}, "phase-c", "endpoint-c"},
		},
		Edges: []Edge{
			{Identity{Source{"synthetic-owner", "meter-family", "service-a"}, "a"}, Identity{Source{"synthetic-owner", "meter-family", "service-a"}, "b"}},
			{Identity{Source{"synthetic-owner", "meter-family", "service-a"}, "a"}, Identity{Source{"synthetic-owner", "meter-family", "service-a"}, "c"}},
		},
	}
	expect(t, fixture(), Result{Status: Valid, Candidate: &want})
	g := fixture()
	g.Edges = append(g.Edges, Edge{identity("b"), identity("c")})
	want.Edges = append(want.Edges, Edge{identity("b"), identity("c")})
	expect(t, g, Result{Status: Valid, Candidate: &want}) // DAG, not a forced tree.
	g = Graph{Sources: []Source{source()}, Nodes: []Node{{identity("a"), "0", "0"}}}
	want = Graph{Sources: []Source{source()}, Nodes: []Node{{identity("a"), "0", "0"}}}
	expect(t, g, Result{Status: Valid, Candidate: &want}) // Opaque zero label is known.
	g.Nodes = append(g.Nodes, Node{identity("b"), "0", "0"})
	want.Nodes = append(want.Nodes, Node{identity("b"), "0", "0"})
	expect(t, g, Result{Status: Valid, Candidate: &want}) // Consistent repeated association.
}

var identityAxes = []struct {
	name string
	set  func(*Identity, string)
}{
	{"owner", func(i *Identity, v string) { i.Source.Owner = v }},
	{"profile", func(i *Identity, v string) { i.Source.Profile = v }},
	{"scope", func(i *Identity, v string) { i.Source.Scope = v }},
	{"id", func(i *Identity, v string) { i.ID = v }},
}

func TestIdentityAxesRemainDistinct(t *testing.T) {
	// Each case kills a node-key mutation that drops the named tuple field.
	for _, axis := range identityAxes {
		t.Run(axis.name, func(t *testing.T) {
			a, b := identity("a"), identity("a")
			axis.set(&b, "z")
			g := Graph{Sources: []Source{a.Source}, Nodes: []Node{{b, "p", "e"}, {a, "p", "e"}}}
			want := Graph{Sources: []Source{a.Source}, Nodes: []Node{{a, "p", "e"}, {b, "p", "e"}}}
			if a.Source != b.Source {
				g.Sources = []Source{b.Source, a.Source}
				want.Sources = []Source{a.Source, b.Source}
				// Same labels across sources may have different associations.
				g.Nodes[0].Endpoint, want.Nodes[1].Endpoint = "other-e", "other-e"
				g.Nodes = append(g.Nodes, Node{Identity{b.Source, "b"}, "other-p", "e"})
				want.Nodes = append(want.Nodes, Node{Identity{b.Source, "b"}, "other-p", "e"})
			}
			expect(t, g, Result{Status: Valid, Candidate: &want})
		})
	}
}

func TestScopedEdgeKeys(t *testing.T) {
	// Each case kills dropping the named source field from both edge ends.
	// From/To ID loss is separately caught by the fan-in/fan-out valid vectors.
	for _, axis := range identityAxes[:3] {
		t.Run(axis.name, func(t *testing.T) {
			a, b := identity("a"), identity("a")
			axis.set(&b, "z")
			aa, bb := Identity{a.Source, "b"}, Identity{b.Source, "b"}
			g := Graph{
				Sources: []Source{b.Source, a.Source},
				Nodes:   []Node{{bb, "p-b", "e-b"}, {aa, "p-b", "e-b"}, {b, "p-a", "e-a"}, {a, "p-a", "e-a"}},
				Edges:   []Edge{{b, bb}, {a, aa}},
			}
			want := Graph{
				Sources: []Source{a.Source, b.Source},
				Nodes:   []Node{{a, "p-a", "e-a"}, {aa, "p-b", "e-b"}, {b, "p-a", "e-a"}, {bb, "p-b", "e-b"}},
				Edges:   []Edge{{a, aa}, {b, bb}},
			}
			expect(t, g, Result{Status: Valid, Candidate: &want})
		})
	}
}

func TestMalformedIdentityAxes(t *testing.T) {
	for _, axis := range identityAxes {
		for name, value := range map[string]string{"empty": "", "space": " ", "padding": " x", "control": "x\x00", "utf8": "\xff"} {
			t.Run(axis.name+"/"+name, func(t *testing.T) {
				id := identity("a")
				axis.set(&id, value)
				g := Graph{Sources: []Source{id.Source}, Nodes: []Node{{id, "p", "e"}}}
				codes := []string{"invalid_node_identity"}
				if axis.name != "id" {
					codes = append(codes, "invalid_source")
				}
				expect(t, g, rejected(Invalid, codes...))
			})
		}
	}
}

func TestReferenceAxes(t *testing.T) {
	// Separate endpoint/axis cases kill lookup mutations dropping any From/To field.
	for _, end := range []string{"from", "to"} {
		for _, axis := range identityAxes {
			t.Run(end+"/"+axis.name, func(t *testing.T) {
				g := fixture()
				ref := &g.Edges[0].From
				if end == "to" {
					ref = &g.Edges[0].To
				}
				axis.set(ref, "missing")
				expect(t, g, rejected(Invalid, "dangling_reference"))
			})
			for name, value := range map[string]string{"empty": "", "padding": "x ", "control": "x\n", "utf8": "\xff"} {
				t.Run(end+"/"+axis.name+"/"+name, func(t *testing.T) {
					g := fixture()
					ref := &g.Edges[0].From
					if end == "to" {
						ref = &g.Edges[0].To
					}
					axis.set(ref, value)
					expect(t, g, rejected(Invalid, "dangling_reference", "invalid_edge_identity"))
				})
			}
		}
	}
}

func TestGraphGuards(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Graph)
		want Result
	}{
		{"duplicate-source", func(g *Graph) { g.Sources = append(g.Sources, source()) }, rejected(Invalid, "duplicate_source")},
		{"undeclared-source", func(g *Graph) { g.Sources[0].Scope = "other" }, rejected(Invalid, "undeclared_source")},
		{"duplicate-node", func(g *Graph) { g.Nodes = append(g.Nodes, g.Nodes[0]) }, rejected(Invalid, "duplicate_node")},
		{"duplicate-edge", func(g *Graph) { g.Edges = append(g.Edges, g.Edges[0]) }, rejected(Invalid, "duplicate_edge")},
		{"self-loop", func(g *Graph) { g.Edges = []Edge{{identity("b"), identity("b")}} }, rejected(Invalid, "cycle")},
		{"cycle", func(g *Graph) {
			g.Edges = append(g.Edges, Edge{identity("b"), identity("c")}, Edge{identity("c"), identity("a")})
		}, rejected(Invalid, "cycle")},
		{"disconnected-cycle", func(g *Graph) { g.Edges = []Edge{{identity("b"), identity("c")}, {identity("c"), identity("b")}} }, rejected(Invalid, "cycle")},
		{"phase-conflict", func(g *Graph) { g.Nodes[1].Phase = "phase-a" }, rejected(Invalid, "phase_endpoint_conflict")},
		{"endpoint-conflict", func(g *Graph) { g.Nodes[1].Endpoint = "endpoint-a" }, rejected(Invalid, "endpoint_phase_conflict")},
		{"unknown-phase", func(g *Graph) { g.Nodes[1].Phase = "" }, rejected(Withheld, "unknown_phase")},
		{"unknown-endpoint", func(g *Graph) { g.Nodes[1].Endpoint = "" }, rejected(Withheld, "unknown_endpoint")},
		{"invalid-dominates-unknown", func(g *Graph) { g.Nodes[0].Phase = ""; g.Nodes = append(g.Nodes, g.Nodes[0]) }, rejected(Invalid, "duplicate_node", "unknown_phase")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { g := fixture(); c.edit(&g); expect(t, g, c.want) })
	}
	for _, axis := range identityAxes[:3] {
		t.Run("cross-source/"+axis.name, func(t *testing.T) {
			g := fixture()
			id := identity("b")
			axis.set(&id, "other")
			g.Sources = append(g.Sources, id.Source)
			g.Nodes = append(g.Nodes, Node{id, "p", "e"})
			g.Edges = []Edge{{identity("a"), id}}
			expect(t, g, rejected(Invalid, "cross_source_edge"))
		})
	}
}

func TestAssociationValueAxes(t *testing.T) {
	for _, field := range []string{"phase", "endpoint"} {
		for name, value := range map[string]string{"space": " ", "padding": "x ", "control": "x\t", "utf8": "\xff"} {
			t.Run(field+"/"+name, func(t *testing.T) {
				g := fixture()
				if field == "phase" {
					g.Nodes[0].Phase = value
				} else {
					g.Nodes[0].Endpoint = value
				}
				expect(t, g, rejected(Invalid, "invalid_"+field))
			})
		}
	}
}

func TestWithheldEmpty(t *testing.T) {
	expect(t, Graph{}, rejected(Withheld, "unknown_nodes", "unknown_sources"))
	expect(t, Graph{Sources: []Source{source()}}, rejected(Withheld, "unknown_nodes"))
	g := fixture()
	g.Sources = nil
	expect(t, g, rejected(Invalid, "undeclared_source", "unknown_sources"))
}

func TestPermutationAndDetachment(t *testing.T) {
	want := fixture()
	// All node permutations and both edge permutations, with independent expected order.
	for _, p := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		for _, reverse := range []bool{false, true} {
			g := fixture()
			g.Nodes = []Node{want.Nodes[p[0]], want.Nodes[p[1]], want.Nodes[p[2]]}
			if reverse {
				g.Edges[0], g.Edges[1] = g.Edges[1], g.Edges[0]
			}
			expect(t, g, Result{Status: Valid, Candidate: &want})
		}
	}
	g := fixture()
	r := Validate(g)
	if r.Candidate == nil {
		t.Fatal(r)
	}
	g.Sources[0].Owner = "changed"
	g.Nodes[0].Phase = "changed"
	g.Edges[0].To.ID = "changed"
	if !reflect.DeepEqual(*r.Candidate, want) {
		t.Fatal("input mutation changed candidate")
	}
	g = fixture()
	r = Validate(g)
	r.Candidate.Sources[0].Scope = "changed"
	r.Candidate.Nodes[0].Endpoint = "changed"
	r.Candidate.Edges[0].From.ID = "changed"
	if !reflect.DeepEqual(g, want) {
		t.Fatal("output mutation changed input")
	}
	// Withholding does not erase a retained candidate or edit partial native input.
	g = fixture()
	g.Nodes[1].Phase = ""
	before := fixture()
	before.Nodes[1].Phase = ""
	expect(t, g, rejected(Withheld, "unknown_phase"))
	if !reflect.DeepEqual(g, before) || !reflect.DeepEqual(want, fixture()) {
		t.Fatal("withholding changed retained facts")
	}
}

func TestProblemPermutation(t *testing.T) {
	g := fixture()
	g.Nodes[0].Phase, g.Nodes[1].Endpoint = "", ""
	g.Edges = append(g.Edges, g.Edges[0])
	want := rejected(Invalid, "duplicate_edge", "unknown_endpoint", "unknown_phase")
	expect(t, g, want)
	g.Nodes[0], g.Nodes[2] = g.Nodes[2], g.Nodes[0]
	g.Edges[0], g.Edges[2] = g.Edges[2], g.Edges[0]
	expect(t, g, want)
}
