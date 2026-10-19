// Package tagtriage validates supplied offline release comparisons. It does not
// fetch sources, authenticate evidence, qualify a library, or admit a runtime.
package tagtriage

import (
	"fmt"
	"strings"
)

type Relevance string

const (
	Relevant    Relevance = "relevant"
	NotRelevant Relevance = "not-relevant"
	Unknown     Relevance = "unknown"
)

type Verification string

const (
	Verified   Verification = "verified"
	Unverified Verification = "unverified"
)

type Axis string

const (
	Binding       Axis = "binding"
	APIBreaks     Axis = "api-breaks"
	Commissioning Axis = "commissioning"
	BLE           Axis = "ble"
	Target        Axis = "target"
	Toolchain     Axis = "toolchain"
)

type EvidenceKind string

const (
	ReleaseNotes EvidenceKind = "release-notes"
	Source       EvidenceKind = "source"
)

// Evidence retains supplied excerpts. Reference locates an offline artifact or
// public source; this package never resolves it. Empty excerpts are missing input.
type Evidence struct {
	ID        string
	Kind      EvidenceKind
	Reference string
	Excerpt   string
}

// Classification keeps relevance separate from a caller's verification claim.
// Verified means supplied evidence was reviewed, not that this validator proved it.
type Classification struct {
	Axis         Axis
	Relevance    Relevance
	Verification Verification
	EvidenceIDs  []string
}

type Record struct {
	PreviousRelease string
	NewRelease      string
	Evidence        []Evidence
	Classifications []Classification
}

func text(s string) bool { return s != "" && strings.TrimSpace(s) == s }

// Validate returns a detached record only when all structural rules hold. Missing
// input is representable as Unknown/Unverified; it never defaults to NotRelevant.
// Failure returns no usable record and leaves the caller's input unchanged.
func Validate(r Record) (Record, error) {
	fail := func(reason string) (Record, error) { return Record{}, fmt.Errorf("tag triage: %s", reason) }
	if !text(r.PreviousRelease) || !text(r.NewRelease) || r.PreviousRelease == r.NewRelease {
		return fail("missing, noncanonical or duplicate release identity")
	}
	evidence := make(map[string]Evidence, len(r.Evidence))
	for _, e := range r.Evidence {
		if !text(e.ID) || !text(e.Reference) || (e.Kind != ReleaseNotes && e.Kind != Source) {
			return fail("invalid evidence identity, reference or kind")
		}
		if _, exists := evidence[e.ID]; exists {
			return fail("duplicate evidence identity")
		}
		evidence[e.ID] = e
	}
	required := map[Axis]bool{Binding: false, APIBreaks: false, Commissioning: false, BLE: false, Target: false, Toolchain: false}
	relevance := map[Axis]Relevance{}
	for _, c := range r.Classifications {
		seen, exists := required[c.Axis]
		if !exists || seen {
			return fail("unknown or duplicate classification axis")
		}
		required[c.Axis] = true
		if c.Relevance != Relevant && c.Relevance != NotRelevant && c.Relevance != Unknown {
			return fail("invalid relevance")
		}
		if c.Verification != Verified && c.Verification != Unverified {
			return fail("invalid verification")
		}
		refs := map[string]bool{}
		hasExcerpt := false
		for _, id := range c.EvidenceIDs {
			e, exists := evidence[id]
			if !exists || refs[id] {
				return fail("missing or duplicate evidence reference")
			}
			refs[id] = true
			if strings.TrimSpace(e.Excerpt) != "" {
				hasExcerpt = true
			}
			if c.Verification == Verified && strings.TrimSpace(e.Excerpt) == "" {
				return fail("verified claim cites missing excerpt")
			}
		}
		if c.Relevance != Unknown && !hasExcerpt {
			return fail("determinate relevance requires an excerpt")
		}
		if c.Verification == Verified && (!hasExcerpt || c.Relevance == Unknown) {
			return fail("verified claim requires determinate supported relevance")
		}
		relevance[c.Axis] = c.Relevance
	}
	for _, seen := range required {
		if !seen {
			return fail("incomplete classification axes")
		}
	}
	if relevance[Binding] == NotRelevant {
		for axis, value := range relevance {
			if axis != Binding && value != NotRelevant {
				return fail("not-relevant binding contradicts relevant or unknown axis")
			}
		}
	}
	out := r
	out.Evidence = append([]Evidence(nil), r.Evidence...)
	out.Classifications = append([]Classification(nil), r.Classifications...)
	for i := range out.Classifications {
		out.Classifications[i].EvidenceIDs = append([]string(nil), r.Classifications[i].EvidenceIDs...)
	}
	return out, nil
}
