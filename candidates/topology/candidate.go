// Package topology validates supplied graphs for an internal contribution
// candidate. Its identifiers and association rules are not Matter definitions.
package topology

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Source is an explicit native owner, profile and service/measurement scope.
// These opaque labels carry no qualification or protocol interpretation.
type Source struct {
	Owner   string
	Profile string
	Scope   string
}

type Identity struct {
	Source Source
	ID     string
}

// Phase and Endpoint are opaque source-local association labels, not phase
// enums, endpoint numbers or endpoint allocations. Empty means unknown.
type Node struct {
	Identity Identity
	Phase    string
	Endpoint string
}

// Edge is a supplied directed relation. Both ends must belong to one source.
type Edge struct {
	From Identity
	To   Identity
}

type Graph struct {
	Sources []Source
	Nodes   []Node
	Edges   []Edge
}

type Status string

const (
	Valid    Status = "valid"
	Invalid  Status = "invalid"
	Withheld Status = "withheld"
)

// Result exposes a detached, sorted graph only on structural success. Problems
// are sorted, unique reason codes, not evidence of native or target support.
type Result struct {
	Status    Status
	Candidate *Graph
	Problems  []string
}

// Validate checks the entire supplied graph atomically. Rejection never edits
// input or a caller's retained state. No discovery, mapping or admission occurs.
func Validate(g Graph) Result {
	problems := map[string]bool{}
	invalid, unknown := false, false
	bad := func(code string) { invalid = true; problems[code] = true }
	missing := func(code string) { unknown = true; problems[code] = true }
	if len(g.Sources) == 0 {
		missing("unknown_sources")
	}
	if len(g.Nodes) == 0 {
		missing("unknown_nodes")
	}
	sources := map[Source]bool{}
	for _, s := range g.Sources {
		if !validSource(s) {
			bad("invalid_source")
		}
		if sources[s] {
			bad("duplicate_source")
		}
		sources[s] = true
	}
	type association struct {
		Source Source
		Label  string
	}
	phases, endpoints := map[association]string{}, map[association]string{}
	nodes := map[Identity]bool{}
	for _, n := range g.Nodes {
		if !validIdentity(n.Identity) {
			bad("invalid_node_identity")
		}
		if !sources[n.Identity.Source] {
			bad("undeclared_source")
		}
		if nodes[n.Identity] {
			bad("duplicate_node")
		}
		nodes[n.Identity] = true
		if n.Phase == "" {
			missing("unknown_phase")
		} else if !validLabel(n.Phase) {
			bad("invalid_phase")
		}
		if n.Endpoint == "" {
			missing("unknown_endpoint")
		} else if !validLabel(n.Endpoint) {
			bad("invalid_endpoint")
		}
		if n.Phase != "" && n.Endpoint != "" {
			p := association{n.Identity.Source, n.Phase}
			e := association{n.Identity.Source, n.Endpoint}
			if prior, ok := phases[p]; ok && prior != n.Endpoint {
				bad("phase_endpoint_conflict")
			}
			if prior, ok := endpoints[e]; ok && prior != n.Phase {
				bad("endpoint_phase_conflict")
			}
			phases[p], endpoints[e] = n.Endpoint, n.Phase
		}
	}
	seen := map[Edge]bool{}
	adj := map[Identity][]Identity{}
	degree := map[Identity]int{}
	for id := range nodes {
		degree[id] = 0
	}
	for _, e := range g.Edges {
		if !validIdentity(e.From) || !validIdentity(e.To) {
			bad("invalid_edge_identity")
		}
		if !nodes[e.From] || !nodes[e.To] {
			bad("dangling_reference")
			continue
		}
		if e.From.Source != e.To.Source {
			bad("cross_source_edge")
		}
		if seen[e] {
			bad("duplicate_edge")
			continue
		}
		seen[e] = true
		adj[e.From] = append(adj[e.From], e.To)
		degree[e.To]++
	}
	// Iterative cycle detection also covers disconnected components and self-loops.
	queue := make([]Identity, 0, len(nodes))
	for id, d := range degree {
		if d == 0 {
			queue = append(queue, id)
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, to := range adj[queue[i]] {
			degree[to]--
			if degree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}
	if len(queue) != len(nodes) {
		bad("cycle")
	}
	if invalid || unknown {
		r := Result{Status: Withheld}
		if invalid {
			r.Status = Invalid
		}
		for p := range problems {
			r.Problems = append(r.Problems, p)
		}
		sort.Strings(r.Problems)
		return r
	}
	out := Graph{
		Sources: append([]Source(nil), g.Sources...),
		Nodes:   append([]Node(nil), g.Nodes...),
		Edges:   append([]Edge(nil), g.Edges...),
	}
	sort.Slice(out.Sources, func(i, j int) bool { return lessSource(out.Sources[i], out.Sources[j]) })
	sort.Slice(out.Nodes, func(i, j int) bool { return lessIdentity(out.Nodes[i].Identity, out.Nodes[j].Identity) })
	sort.Slice(out.Edges, func(i, j int) bool {
		a, b := out.Edges[i], out.Edges[j]
		if a.From != b.From {
			return lessIdentity(a.From, b.From)
		}
		return lessIdentity(a.To, b.To)
	})
	return Result{Status: Valid, Candidate: &out}
}

func validLabel(s string) bool {
	return s != "" && utf8.ValidString(s) && strings.TrimSpace(s) == s &&
		!strings.ContainsFunc(s, unicode.IsControl)
}

func validSource(s Source) bool {
	return validLabel(s.Owner) && validLabel(s.Profile) && validLabel(s.Scope)
}

func validIdentity(id Identity) bool { return validSource(id.Source) && validLabel(id.ID) }

func lessSource(a, b Source) bool {
	if a.Owner != b.Owner {
		return a.Owner < b.Owner
	}
	if a.Profile != b.Profile {
		return a.Profile < b.Profile
	}
	return a.Scope < b.Scope
}

func lessIdentity(a, b Identity) bool {
	if a.Source != b.Source {
		return lessSource(a.Source, b.Source)
	}
	return a.ID < b.ID
}
