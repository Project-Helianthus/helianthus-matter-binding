// Package bridge implements an internal, offline contribution candidate.
// It neither imports an SDK nor makes production admission decisions.
package bridge

import (
	"sort"
	"strings"
)

// Association contains caller-owned opaque review identifiers. Slot is a
// positive bookkeeping number, not a validated Matter endpoint number.
// Capability must include the source fact's service/channel/phase scope.
type Association struct {
	Node, Owner, Profile, Capability, Composition, Binding string
	Slot                                                   int64
	Retired                                                bool
}

type Edge struct{ Node, Parent, Child string }

type Issue struct{ Code, Node, Binding string }

// Result owns all its slices. Retained is an untrusted copy of the supplied
// history, even when that history is malformed; it is never upgraded to valid.
// Candidate and Graph are withheld together whenever Issues is nonempty.
type Result struct {
	Retained, Candidate []Association
	Graph               []Edge
	Issues              []Issue
}

type sourceKey struct{ node, owner, profile, capability string }
type bindingKey struct{ node, binding string }
type slotKey struct {
	node string
	slot int64
}

func source(a Association) sourceKey   { return sourceKey{a.Node, a.Owner, a.Profile, a.Capability} }
func binding(a Association) bindingKey { return bindingKey{a.Node, a.Binding} }
func token(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n\t")
}

// canonical orders even ambiguous records for deterministic diagnostics.
func canonical(records []Association) []Association {
	ordered := append([]Association(nil), records...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		left := []string{a.Node, a.Owner, a.Profile, a.Capability, a.Composition, a.Binding}
		right := []string{b.Node, b.Owner, b.Profile, b.Capability, b.Composition, b.Binding}
		for k := range left {
			if left[k] != right[k] {
				return left[k] < right[k]
			}
		}
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return !a.Retired && b.Retired
	})
	return ordered
}

// Review checks complete snapshots, never deltas. All old records must remain;
// retirement is one-way and cannot release a slot within a node lifecycle.
// A different node lifecycle must be reviewed separately, not substituted here.
func Review(previous, proposed []Association, edges []Edge) Result {
	r := Result{Retained: append([]Association(nil), previous...)}
	add := func(code, node, id string) { r.Issues = append(r.Issues, Issue{code, node, id}) }
	check := func(records []Association, prefix string) {
		sources := map[sourceKey]bool{}
		bindings := map[bindingKey]bool{}
		slots := map[slotKey]bool{}
		for _, a := range records {
			if !token(a.Node) || !token(a.Owner) || !token(a.Profile) || !token(a.Capability) || !token(a.Binding) {
				add(prefix+"identity_invalid", a.Node, a.Binding)
			}
			if !token(a.Composition) {
				add(prefix+"composition_unknown", a.Node, a.Binding)
			}
			if a.Slot <= 0 {
				add(prefix+"slot_invalid", a.Node, a.Binding)
			}
			if sources[source(a)] {
				add(prefix+"source_alias", a.Node, a.Binding)
			}
			if bindings[binding(a)] {
				add(prefix+"binding_alias", a.Node, a.Binding)
			}
			sk := slotKey{a.Node, a.Slot}
			if slots[sk] {
				add(prefix+"slot_alias", a.Node, a.Binding)
			}
			sources[source(a)], bindings[binding(a)], slots[sk] = true, true, true
		}
	}
	ordered := canonical(proposed)
	check(canonical(previous), "history_")
	check(ordered, "")
	bySource := map[sourceKey]Association{}
	byBinding := map[bindingKey]Association{}
	for _, a := range ordered {
		bySource[source(a)], byBinding[binding(a)] = a, a
	}
	for _, old := range previous {
		next, ok := bySource[source(old)]
		if !ok {
			add("history_lost", old.Node, old.Binding)
			continue
		}
		if old.Composition != next.Composition {
			add("composition_changed", old.Node, old.Binding)
		}
		if old.Binding != next.Binding || old.Slot != next.Slot {
			add("association_changed", old.Node, old.Binding)
		}
		if old.Retired && !next.Retired {
			add("retirement_reversed", old.Node, old.Binding)
		}
	}
	parents := map[bindingKey]bindingKey{}
	seen := map[Edge]bool{}
	for _, e := range edges {
		p, c := bindingKey{e.Node, e.Parent}, bindingKey{e.Node, e.Child}
		pa, pok := byBinding[p]
		ca, cok := byBinding[c]
		if !token(e.Node) || !token(e.Parent) || !token(e.Child) {
			add("edge_identity_invalid", e.Node, e.Child)
		}
		if !pok || !cok {
			add("edge_unknown", e.Node, e.Child)
		}
		if (pok && pa.Retired) || (cok && ca.Retired) {
			add("edge_retired", e.Node, e.Child)
		}
		if seen[e] {
			add("edge_duplicate", e.Node, e.Child)
		}
		seen[e] = true
		if prior, ok := parents[c]; ok && prior != p {
			add("multiple_parents", e.Node, e.Child)
		}
		parents[c] = p
	}
	// Topological elimination uses every distinct edge, including ambiguous
	// parent proposals, so diagnostic selection cannot depend on input order.
	degrees := map[bindingKey]int{}
	children := map[bindingKey][]bindingKey{}
	for e := range seen {
		p, c := bindingKey{e.Node, e.Parent}, bindingKey{e.Node, e.Child}
		if _, ok := degrees[p]; !ok {
			degrees[p] = 0
		}
		degrees[c]++
		children[p] = append(children[p], c)
	}
	queue := []bindingKey{}
	for k, d := range degrees {
		if d == 0 {
			queue = append(queue, k)
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, child := range children[queue[i]] {
			degrees[child]--
			if degrees[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if len(queue) != len(degrees) {
		add("graph_cycle", "", "")
	}
	sort.Slice(r.Issues, func(i, j int) bool {
		a, b := r.Issues[i], r.Issues[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Node != b.Node {
			return a.Node < b.Node
		}
		return a.Binding < b.Binding
	})
	unique := r.Issues[:0]
	for _, issue := range r.Issues {
		if len(unique) == 0 || unique[len(unique)-1] != issue {
			unique = append(unique, issue)
		}
	}
	r.Issues = unique
	if len(r.Issues) != 0 {
		return r
	}
	r.Candidate = append([]Association(nil), proposed...)
	sort.Slice(r.Candidate, func(i, j int) bool {
		a, b := r.Candidate[i], r.Candidate[j]
		if a.Node != b.Node {
			return a.Node < b.Node
		}
		return a.Binding < b.Binding
	})
	r.Graph = append([]Edge(nil), edges...)
	sort.Slice(r.Graph, func(i, j int) bool {
		a, b := r.Graph[i], r.Graph[j]
		if a.Node != b.Node {
			return a.Node < b.Node
		}
		if a.Parent != b.Parent {
			return a.Parent < b.Parent
		}
		return a.Child < b.Child
	})
	return r
}
