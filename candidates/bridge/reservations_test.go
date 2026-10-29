package bridge

import (
	"math"
	"reflect"
	"testing"
)

func aa11Association() Association {
	return Association{Node: "node-a", Owner: "owner-a", Profile: "profile-a", Capability: "cap-a", Composition: "composition-a", Binding: "binding-a"}
}
func aa11Policy(t *testing.T, limit int64, retained []Association) *Reservations {
	t.Helper()
	p, err := NewReservations("node-a", limit, retained)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func aa11Reserve(t *testing.T, p *Reservations, a Association) Association {
	t.Helper()
	got, err := p.Reserve(a)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestAA11DeselectionRetainsAndRetirementIsOneWay(t *testing.T) {
	p := aa11Policy(t, 3, nil)
	a := aa11Reserve(t, p, aa11Association())
	if a.Slot != 1 {
		t.Fatal("first number")
	}
	b := aa11Association()
	b.Capability = "cap-b"
	b.Binding = "binding-b"
	b = aa11Reserve(t, p, b)
	if b.Slot != 2 {
		t.Fatal("second number")
	}
	want := []Association{a, b}
	if err := p.Deselect(a.Binding); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Retained(), want) {
		t.Fatal("deselection forgot identity or sibling")
	}
	if again := aa11Reserve(t, p, aa11Association()); again != a {
		t.Fatal("deselection changed identity")
	}
	restored := aa11Policy(t, 3, p.Retained())
	if again := aa11Reserve(t, restored, aa11Association()); again != a {
		t.Fatal("restart changed identity")
	}
	if err := p.Retire(a.Binding); err != nil {
		t.Fatal(err)
	}
	want[0].Retired = true
	if !reflect.DeepEqual(p.Retained(), want) {
		t.Fatal("retirement lost tombstone or sibling")
	}
	if _, err := p.Reserve(aa11Association()); err == nil {
		t.Fatal("retirement reversed")
	}
	c := aa11Association()
	c.Capability = "cap-c"
	c.Binding = "binding-c"
	c = aa11Reserve(t, p, c)
	if c.Slot != 3 {
		t.Fatal("retired slot recycled")
	}
	detached := p.Retained()
	detached[0].Binding = "tampered"
	if p.Retained()[0] != want[0] {
		t.Fatal("observer aliases ledger")
	}
	if p.Deselect("unknown") == nil || p.Retire("unknown") == nil {
		t.Fatal("unknown binding accepted")
	}
}

func TestAA11ExhaustionBeforeEffectsAndNoWrap(t *testing.T) {
	for _, limit := range []int64{2, math.MaxInt64} {
		a := aa11Association()
		a.Slot = limit
		a.Retired = true
		p := aa11Policy(t, limit, []Association{a})
		b := aa11Association()
		b.Capability = "cap-b"
		b.Binding = "binding-b"
		effects := 0
		if _, err := p.Reserve(b); err == nil {
			effects++
		}
		if effects != 0 || !reflect.DeepEqual(p.Retained(), []Association{a}) {
			t.Fatal("exhaustion admitted SDK effects or recycled slot")
		}
		// A genuine positive control immediately below the boundary must succeed.
		a.Slot = limit - 1
		q := aa11Policy(t, limit, []Association{a})
		got := aa11Reserve(t, q, b)
		if got.Slot != limit {
			t.Fatal("final number withheld")
		}
		if _, err := q.Reserve(Association{Node: "node-a", Owner: "owner-a", Profile: "profile-a", Capability: "cap-c", Composition: "composition-a", Binding: "binding-c"}); err == nil {
			t.Fatal("allocator wrapped")
		}
	}
}

// Independent field changes target a missing identity comparison, Review
// invocation, or node-scope check. Each altered proposal keeps all other fields
// valid; failure must leave both the original and valid sibling untouched.
func TestAA11ReservationIdentityFields(t *testing.T) {
	variants := map[string]func(*Association){
		"node":        func(a *Association) { a.Node = "node-b" },
		"owner":       func(a *Association) { a.Owner = "owner-b" },
		"profile":     func(a *Association) { a.Profile = "profile-b" },
		"capability":  func(a *Association) { a.Capability = "cap-b" },
		"composition": func(a *Association) { a.Composition = "composition-b" },
		"binding":     func(a *Association) { a.Binding = "binding-b" },
		"slot":        func(a *Association) { a.Slot = 3 },
		"retired":     func(a *Association) { a.Retired = true },
	}
	for name, change := range variants {
		t.Run(name, func(t *testing.T) {
			a := aa11Association()
			a.Slot = 1
			b := a
			b.Capability = "cap-sibling"
			b.Binding = "binding-sibling"
			b.Slot = 2
			p := aa11Policy(t, 4, []Association{a, b})
			bad := a
			change(&bad)
			if _, err := p.Reserve(bad); err == nil {
				t.Fatal("changed retained identity accepted")
			}
			if !reflect.DeepEqual(p.Retained(), []Association{a, b}) {
				t.Fatal("failed review replaced sibling or identity")
			}
			if good := aa11Reserve(t, p, a); good != a {
				t.Fatal("valid identity lost")
			}
		})
	}
}

// Source-key axes must also remain distinct when every binding/slot is unique.
// These positives catch a source key that drops owner, profile or capability.
func TestAA11DistinctSourceAxes(t *testing.T) {
	variants := map[string]func(*Association){"owner": func(a *Association) { a.Owner = "owner-b" }, "profile": func(a *Association) { a.Profile = "profile-b" }, "capability": func(a *Association) { a.Capability = "cap-b" }}
	for name, change := range variants {
		t.Run(name, func(t *testing.T) {
			p := aa11Policy(t, 2, nil)
			a := aa11Reserve(t, p, aa11Association())
			b := aa11Association()
			change(&b)
			b.Binding = "binding-b"
			got := aa11Reserve(t, p, b)
			b.Slot = 2
			if got != b || !reflect.DeepEqual(p.Retained(), []Association{a, b}) {
				t.Fatal("source key conflated")
			}
		})
	}
	// Node lifecycle is a separate ledger; identical identities there may start at one.
	a := aa11Association()
	a.Node = "node-b"
	p, err := NewReservations("node-b", 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := aa11Reserve(t, p, a); got.Slot != 1 {
		t.Fatal("node scope conflated")
	}
}

func TestAA11RetainedSnapshotValidation(t *testing.T) {
	a := aa11Association()
	a.Slot = 1
	for _, variant := range []string{"node", "owner", "profile", "capability", "composition", "binding", "slot", "duplicate", "conflict", "out-of-range"} {
		t.Run(variant, func(t *testing.T) {
			bad := a
			records := []Association{bad}
			switch variant {
			case "node":
				bad.Node = ""
			case "owner":
				bad.Owner = ""
			case "profile":
				bad.Profile = ""
			case "capability":
				bad.Capability = ""
			case "composition":
				bad.Composition = ""
			case "binding":
				bad.Binding = ""
			case "slot":
				bad.Slot = 0
			case "out-of-range":
				bad.Slot = 3
			case "duplicate":
				records = append(records, a)
			case "conflict":
				b := a
				b.Capability = "cap-b"
				b.Binding = "binding-b"
				records = append(records, b)
			}
			records[0] = bad
			if _, err := NewReservations("node-a", 2, records); err == nil {
				t.Fatal("corrupt snapshot accepted")
			}
			p := aa11Policy(t, 2, []Association{a})
			records[0].Binding = "changed"
			if got := aa11Reserve(t, p, a); got != a {
				t.Fatal("valid snapshot control")
			}
		})
	}
	for _, limit := range []int64{0, -1} {
		if _, err := NewReservations("node-a", limit, nil); err == nil {
			t.Fatal("invalid limit")
		}
	}
	if _, err := NewReservations("", 2, nil); err == nil {
		t.Fatal("missing node")
	}
	p := aa11Policy(t, 2, nil)
	if len(p.Retained()) != 0 {
		t.Fatal("absence fabricated reservations")
	}
}

func TestAA11ReviewFailurePreservesSiblings(t *testing.T) {
	a := aa11Association()
	a.Slot = 1
	b := a
	b.Capability = "cap-sibling"
	b.Binding = "binding-sibling"
	b.Slot = 2
	p := aa11Policy(t, 4, []Association{a, b})
	// Slot zero is valid allocation input, so the duplicate opaque binding must
	// reach Review rather than fail the next-number check. A failed review must
	// not replace the prior snapshot with the rejected proposal.
	proposal := aa11Association()
	proposal.Capability = "cap-new"
	proposal.Binding = b.Binding
	if _, err := p.Reserve(proposal); err == nil {
		t.Fatal("Review binding collision bypassed")
	}
	if !reflect.DeepEqual(p.Retained(), []Association{a, b}) {
		t.Fatal("failed review replaced valid sibling")
	}
	proposal.Binding = "binding-new"
	got := aa11Reserve(t, p, proposal)
	proposal.Slot = 3
	if got != proposal || !reflect.DeepEqual(p.Retained(), []Association{a, b, proposal}) {
		t.Fatal("valid control after failed review")
	}
}
