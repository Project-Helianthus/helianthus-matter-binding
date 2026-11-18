// Package bridge implements an internal, offline contribution candidate.
// It neither imports an SDK nor makes production admission decisions.
// An association is keyed by its resource identifier. Native owner and profile
// are provenance. A parent edge cites a relation key of one published instance.
package bridge

import (
	"sort"
	"strings"
)

// NativeTuple is the refused native owner, profile, scope and local id.
// It is not an association key.
type NativeTuple struct {
	Owner   string
	Profile string
	Scope   string
	ID      string
}

// Association contains caller-owned opaque review identifiers. ResourceID is
// the association key. Owner and Profile are provenance. Slot is a positive
// bookkeeping number, not a validated Matter endpoint number.
type Association struct {
	Node, ResourceID, Owner, Profile, Capability, Composition, Binding string
	Slot                                                               int64
	Retired                                                            bool
	Native                                                             *NativeTuple
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

// Edge is a proposed parent and child binding. Cited is the relation key.
// A nil citation is uncited.
type Edge struct {
	Node, Parent, Child string
	Cited               *RelationKey
}

type Issue struct{ Code, Node, Binding string }

// Result owns all its slices. Retained is an untrusted copy of the supplied
// history, even when that history is malformed; it is never upgraded to valid.
// Candidate and Graph are withheld together whenever Issues is nonempty.
type Result struct {
	Retained, Candidate []Association
	Graph               []Edge
	Issues              []Issue
}

type resourceKey struct{ resourceID string }
type bindingKey struct{ node, binding string }
type slotKey struct {
	node string
	slot int64
}
type edgeKey struct{ node, parent, child string }

func source(a Association) resourceKey { return resourceKey{a.ResourceID} }
func binding(a Association) bindingKey { return bindingKey{a.Node, a.Binding} }
func token(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n\t")
}

func validKey(k RelationKey) bool {
	return token(k.PackID) && token(k.PackVersion) && token(k.Definition) &&
		token(k.Version) && token(k.From) && token(k.To)
}

func cloneAssociation(a Association) Association {
	if a.Native != nil {
		n := *a.Native
		a.Native = &n
	}
	return a
}

func cloneEdge(e Edge) Edge {
	if e.Cited != nil {
		k := *e.Cited
		e.Cited = &k
	}
	return e
}

func cloneAssociations(records []Association) []Association {
	if records == nil {
		return nil
	}
	out := make([]Association, len(records))
	for i, a := range records {
		out[i] = cloneAssociation(a)
	}
	return out
}

// canonical orders even ambiguous records for deterministic diagnostics.
func canonical(records []Association) []Association {
	ordered := cloneAssociations(records)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		left := []string{a.Node, a.ResourceID, a.Owner, a.Profile, a.Capability, a.Composition, a.Binding}
		right := []string{b.Node, b.ResourceID, b.Owner, b.Profile, b.Capability, b.Composition, b.Binding}
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

// Review checks complete snapshots, never deltas. The association key is the
// resource identifier. All old resource keys must remain. Retirement is
// one-way and cannot release a slot within a node lifecycle. Instances are the
// published relations an edge may cite.
func Review(previous, proposed []Association, edges []Edge, instances []Instance) Result {
	r := Result{Retained: cloneAssociations(previous)}
	add := func(code, node, id string) { r.Issues = append(r.Issues, Issue{code, node, id}) }
	check := func(records []Association, prefix string) {
		sources := map[resourceKey]bool{}
		bindings := map[bindingKey]bool{}
		slots := map[slotKey]bool{}
		for _, a := range records {
			if a.Native != nil {
				add(prefix+"native_tuple_node", a.Node, a.Binding)
			}
			if !token(a.Node) || !token(a.ResourceID) || !token(a.Owner) || !token(a.Profile) || !token(a.Capability) || !token(a.Binding) {
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
	bySource := map[resourceKey]Association{}
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
		if old.Owner != next.Owner || old.Profile != next.Profile {
			add("provenance_changed", old.Node, old.Binding)
		}
		if old.Composition != next.Composition {
			add("composition_changed", old.Node, old.Binding)
		}
		if old.Node != next.Node || old.Capability != next.Capability || old.Binding != next.Binding || old.Slot != next.Slot {
			add("association_changed", old.Node, old.Binding)
		}
		if old.Retired && !next.Retired {
			add("retirement_reversed", old.Node, old.Binding)
		}
	}
	byRelation := map[RelationKey]Instance{}
	for _, inst := range instances {
		if !validKey(inst.Key) || !token(inst.Coherence) || !token(inst.Publisher) {
			add("invalid_relation", "", "")
			continue
		}
		if _, ok := byRelation[inst.Key]; ok {
			add("duplicate_relation", "", "")
		}
		byRelation[inst.Key] = inst
	}
	parents := map[bindingKey]bindingKey{}
	seen := map[edgeKey]bool{}
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
		id := edgeKey{e.Node, e.Parent, e.Child}
		if seen[id] {
			add("edge_duplicate", e.Node, e.Child)
		}
		seen[id] = true
		if e.Cited == nil || !validKey(*e.Cited) {
			add("uncited_edge", e.Node, e.Child)
		} else if inst, ok := byRelation[*e.Cited]; !ok {
			add("uncited_edge", e.Node, e.Child)
		} else if inst.Coherence != "coherent" {
			add("incoherent_relation", e.Node, e.Child)
		} else if !pok || !cok || inst.Key.From != pa.ResourceID || inst.Key.To != ca.ResourceID {
			add("uncited_edge", e.Node, e.Child)
		} else if (pa.Owner != ca.Owner || pa.Profile != ca.Profile) && inst.Publisher != "core" {
			add("cross_source_edge", e.Node, e.Child)
		}
		if prior, ok := parents[c]; ok && prior != p {
			add("multiple_parents", e.Node, e.Child)
		}
		parents[c] = p
	}
	degrees := map[bindingKey]int{}
	children := map[bindingKey][]bindingKey{}
	for e := range seen {
		p, c := bindingKey{e.node, e.parent}, bindingKey{e.node, e.child}
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
	r.Candidate = cloneAssociations(proposed)
	sort.Slice(r.Candidate, func(i, j int) bool {
		a, b := r.Candidate[i], r.Candidate[j]
		if a.Node != b.Node {
			return a.Node < b.Node
		}
		return a.Binding < b.Binding
	})
	r.Graph = make([]Edge, len(edges))
	for i, e := range edges {
		r.Graph[i] = cloneEdge(e)
	}
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
