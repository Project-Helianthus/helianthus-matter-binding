// Package topology validates supplied graphs for an internal contribution
// candidate. A node is keyed by its resource identifier. Native owner and
// profile are provenance. An edge cites a relation key of one published
// instance. These rules are not Matter definitions.
package topology

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// NativeTuple is the refused native owner, profile, scope and local id.
// It is not a node key.
type NativeTuple struct {
	Owner   string
	Profile string
	Scope   string
	ID      string
}

// Node is one resource. Owner and Profile are provenance and do not select
// the node. Native, when set, is a native-tuple identity and is refused.
// Phase and Endpoint are opaque association labels, not enumerations or
// endpoint allocations. Empty means unknown.
type Node struct {
	ResourceID string
	Owner      string
	Profile    string
	Phase      string
	Endpoint   string
	Native     *NativeTuple
}

// RelationKey cites one published relation instance. Every field is part of
// the key. From and To are resource identifiers.
type RelationKey struct {
	PackID      string
	PackVersion string
	Definition  string
	Version     string
	From        string
	To          string
}

// Instance is one published relation. Coherence is the cut coherence of that
// instance. Publisher is "core" when the core catalog published it.
type Instance struct {
	Key       RelationKey
	Coherence string
	Publisher string
}

// Edge cites one relation key. A nil citation is uncited.
type Edge struct {
	Cited *RelationKey
}

type Graph struct {
	Nodes     []Node
	Instances []Instance
	Edges     []Edge
}

type Status string

const (
	Valid    Status = "valid"
	Invalid  Status = "invalid"
	Withheld Status = "withheld"
)

// Result exposes a detached, sorted graph only on structural success. Problems
// are sorted, unique reason codes.
type Result struct {
	Status    Status
	Candidate *Graph
	Problems  []string
}

type provenance struct{ owner, profile string }

// Validate checks the entire supplied graph atomically. Rejection never edits
// input or a caller's retained state. No discovery, mapping or admission occurs.
func Validate(g Graph) Result {
	problems := map[string]bool{}
	invalid, unknown := false, false
	bad := func(code string) { invalid = true; problems[code] = true }
	missing := func(code string) { unknown = true; problems[code] = true }
	if len(g.Nodes) == 0 {
		missing("unknown_nodes")
	}
	phases := map[association]string{}
	endpoints := map[association]string{}
	nodeByID := map[string]Node{}
	for _, n := range g.Nodes {
		if n.Native != nil {
			bad("native_tuple_node")
			continue
		}
		if !validLabel(n.ResourceID) {
			bad("invalid_node_identity")
			continue
		}
		if !validLabel(n.Owner) || !validLabel(n.Profile) {
			bad("invalid_provenance")
			continue
		}
		if _, ok := nodeByID[n.ResourceID]; ok {
			bad("duplicate_node")
			continue
		}
		nodeByID[n.ResourceID] = n
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
		if n.Phase != "" && n.Endpoint != "" && validLabel(n.Phase) && validLabel(n.Endpoint) {
			p := association{provenance{n.Owner, n.Profile}, n.Phase}
			e := association{provenance{n.Owner, n.Profile}, n.Endpoint}
			if prior, ok := phases[p]; ok && prior != n.Endpoint {
				bad("phase_endpoint_conflict")
			}
			if prior, ok := endpoints[e]; ok && prior != n.Phase {
				bad("endpoint_phase_conflict")
			}
			phases[p], endpoints[e] = n.Endpoint, n.Phase
		}
	}
	byKey := map[RelationKey]Instance{}
	for _, inst := range g.Instances {
		if !validKey(inst.Key) || !validLabel(inst.Coherence) || !validLabel(inst.Publisher) {
			bad("invalid_relation")
			continue
		}
		if _, ok := byKey[inst.Key]; ok {
			bad("duplicate_relation")
		}
		byKey[inst.Key] = inst
	}
	seen := map[RelationKey]bool{}
	adj := map[string][]string{}
	degree := map[string]int{}
	for id := range nodeByID {
		degree[id] = 0
	}
	for _, e := range g.Edges {
		if e.Cited == nil || !validKey(*e.Cited) {
			bad("uncited_edge")
			continue
		}
		key := *e.Cited
		if seen[key] {
			bad("duplicate_edge")
			continue
		}
		seen[key] = true
		inst, ok := byKey[key]
		if !ok {
			bad("uncited_edge")
			continue
		}
		from, fok := nodeByID[key.From]
		to, tok := nodeByID[key.To]
		if !fok || !tok {
			bad("dangling_reference")
			continue
		}
		if inst.Coherence != "coherent" {
			bad("incoherent_relation")
		}
		if (from.Owner != to.Owner || from.Profile != to.Profile) && inst.Publisher != "core" {
			bad("cross_source_edge")
		}
		adj[key.From] = append(adj[key.From], key.To)
		degree[key.To]++
	}
	queue := make([]string, 0, len(nodeByID))
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
	if len(queue) != len(nodeByID) {
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
		Nodes:     append([]Node(nil), g.Nodes...),
		Instances: append([]Instance(nil), g.Instances...),
	}
	if g.Edges != nil {
		out.Edges = make([]Edge, len(g.Edges))
		for i, e := range g.Edges {
			out.Edges[i] = cloneEdge(e)
		}
		sort.Slice(out.Edges, func(i, j int) bool { return lessKey(*out.Edges[i].Cited, *out.Edges[j].Cited) })
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].ResourceID < out.Nodes[j].ResourceID })
	sort.Slice(out.Instances, func(i, j int) bool { return lessKey(out.Instances[i].Key, out.Instances[j].Key) })
	return Result{Status: Valid, Candidate: &out}
}

type association struct {
	provenance provenance
	label      string
}

func cloneEdge(e Edge) Edge {
	if e.Cited != nil {
		k := *e.Cited
		e.Cited = &k
	}
	return e
}

func validLabel(s string) bool {
	return s != "" && utf8.ValidString(s) && strings.TrimSpace(s) == s &&
		!strings.ContainsFunc(s, unicode.IsControl)
}

func validKey(k RelationKey) bool {
	return validLabel(k.PackID) && validLabel(k.PackVersion) && validLabel(k.Definition) &&
		validLabel(k.Version) && validLabel(k.From) && validLabel(k.To)
}

func lessKey(a, b RelationKey) bool {
	left := []string{a.PackID, a.PackVersion, a.Definition, a.Version, a.From, a.To}
	right := []string{b.PackID, b.PackVersion, b.Definition, b.Version, b.From, b.To}
	for i := range left {
		if left[i] != right[i] {
			return left[i] < right[i]
		}
	}
	return false
}
