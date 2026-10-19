package commissioning

import (
	"reflect"
	"testing"
)

type fakePorts struct {
	calls    []string
	requests []Request
}

func (p *fakePorts) call(op string, q Request) {
	p.calls = append(p.calls, op)
	p.requests = append(p.requests, q)
}
func (p *fakePorts) PASE(q Request) { p.call("PASE", q) }
func (p *fakePorts) OpenWindow(q Request) {
	op := "CloseWindow"
	if q.Open {
		op = "OpenWindow"
	}
	p.call(op, q)
}
func (p *fakePorts) Discover(q Request) { p.call("Discover", q) }
func (p *fakePorts) Restart(q Request)  { p.call("Restart", q) }
func fixture(t *testing.T) (*Runner, *fakePorts) {
	t.Helper()
	p := &fakePorts{}
	r, err := New(p, "n-a", Store{})
	if err != nil {
		t.Fatal(err)
	}
	return r, p
}
func want(t *testing.T, e Event, outcome, state string, pending int) {
	t.Helper()
	if e.Outcome != outcome || e.State != state || e.Pending != pending {
		t.Fatalf("got %+v; want %s/%s/%d", e, outcome, state, pending)
	}
}
func pairedWindow(t *testing.T, r *Runner) {
	t.Helper()
	want(t, r.Start(), "accepted", "ready", 0)
	q := r.PASE()
	want(t, q, "pending", "ready", 1)
	want(t, r.Complete(q.Identity, true, nil), "succeeded", "paired", 0)
	q = r.OpenWindow(true)
	want(t, q, "pending", "paired", 1)
	want(t, r.Complete(q.Identity, true, nil), "succeeded", "window", 0)
}
func TestOrderedCommissionAndRestart(t *testing.T) {
	r, p := fixture(t)
	pairedWindow(t, r)
	q := r.Discover()
	want(t, r.Complete(q.Identity, true, []Record{{"n-a"}}), "succeeded", "discovered", 0)
	if r.Snapshot() != (Store{}) {
		t.Fatal("discovery commissioned a node")
	}
	want(t, r.Stop(), "rejected", "discovered", 0)
	close := r.OpenWindow(false)
	if p.requests[len(p.requests)-1].Open {
		t.Fatal("closure dispatched as open")
	}
	want(t, r.Complete(close.Identity, true, nil), "succeeded", "commissioned", 0)
	if r.Snapshot() != (Store{Record{"n-a"}}) {
		t.Fatal("missing committed synthetic association")
	}
	want(t, r.Stop(), "accepted", "stopped", 0)
	q = r.Restart()
	want(t, q, "pending", "ready", 1)
	want(t, r.Complete(q.Identity, true, nil), "succeeded", "commissioned", 0)
	if !reflect.DeepEqual(p.calls, []string{"PASE", "OpenWindow", "Discover", "CloseWindow", "Restart"}) {
		t.Fatal(p.calls)
	}
	if q.Generation != 2 || q.Token != 5 || q.Node != "n-a" || r.ProductionAllocations() != 0 {
		t.Fatal(q)
	}
}
func TestOrderedRejectionsAndPendingBound(t *testing.T) {
	r, p := fixture(t)
	want(t, r.Discover(), "rejected", "stopped", 0)
	want(t, r.Start(), "accepted", "ready", 0)
	want(t, r.Start(), "rejected", "ready", 0)
	want(t, r.OpenWindow(true), "rejected", "ready", 0)
	want(t, r.Discover(), "rejected", "ready", 0)
	q := r.PASE()
	want(t, r.PASE(), "rejected", "ready", 1)
	want(t, r.Restart(), "rejected", "ready", 1)
	want(t, r.Complete(q.Identity, false, nil), "failed", "ready", 0)
	if !reflect.DeepEqual(p.calls, []string{"PASE"}) || r.Snapshot() != (Store{}) {
		t.Fatal(p.calls)
	}
}
func TestCallbackIdentityAxes(t *testing.T) {
	// Each variant changes exactly one field; expected state is literal, not
	// computed using the runner's identity comparison.
	for _, axis := range []string{"generation", "token", "node"} {
		t.Run(axis, func(t *testing.T) {
			r, _ := fixture(t)
			r.Start()
			q := r.PASE()
			wrong := q.Identity
			switch axis {
			case "generation":
				wrong.Generation++
			case "token":
				wrong.Token++
			case "node":
				wrong.Node = "n-b"
			}
			want(t, r.Complete(wrong, true, nil), "ignored", "ready", 1)
			want(t, r.Complete(q.Identity, true, nil), "succeeded", "paired", 0)
			// Replaying the old successful PASE while another operation owns the slot
			// must not release that registration or alter the state.
			open := r.OpenWindow(true)
			want(t, r.Complete(q.Identity, true, nil), "ignored", "paired", 1)
			want(t, r.Complete(open.Identity, true, nil), "succeeded", "window", 0)
			want(t, r.Complete(open.Identity, false, nil), "ignored", "window", 0)
		})
	}
}
func TestDiscoveryAndExplicitClosure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		records []Record
	}{
		{"absent", nil}, {"ambiguous", []Record{{"n-a"}, {"n-b"}}}, {"duplicate", []Record{{"n-a"}, {"n-a"}}},
		{"wrong-node", []Record{{"n-b"}}}, {"oversized-symbol", []Record{{string(make([]byte, SymbolBytes+1))}}},
		{"record-bound", []Record{{"n-a"}, {"n-b"}, {"n-c"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := fixture(t)
			pairedWindow(t, r)
			q := r.Discover()
			want(t, r.Complete(q.Identity, true, tc.records), "ambiguous-or-missing", "window", 0)
			want(t, r.Stop(), "rejected", "window", 0)
			close := r.OpenWindow(false)
			want(t, r.Stop(), "rejected", "window", 1)
			want(t, r.Complete(close.Identity, false, nil), "failed", "window", 0)
			close = r.OpenWindow(false)
			want(t, r.Complete(close.Identity, true, nil), "succeeded", "ready", 0)
			if r.Snapshot() != (Store{}) || r.ProductionAllocations() != 0 {
				t.Fatal("unexpected allocation")
			}
		})
	}
}
func TestDrainAndStaleGeneration(t *testing.T) {
	r, _ := fixture(t)
	r.Start()
	old := r.PASE()
	want(t, r.Stop(), "accepted", "stopped", 1)
	want(t, r.Stop(), "unchanged", "stopped", 1)
	want(t, r.Restart(), "rejected", "stopped", 1)
	for i := 0; i < DrainSteps-1; i++ {
		want(t, r.Step(), "accepted", "stopped", 1)
	}
	want(t, r.Complete(old.Identity, true, nil), "drained", "stopped", 0)
	q := r.Restart()
	want(t, q, "pending", "ready", 1)
	// Give the old generation the exact current token/node: generation alone
	// must fence it, rather than relying on monotonically increasing tokens.
	stale := q.Identity
	stale.Generation = old.Generation
	want(t, r.Complete(stale, true, nil), "ignored", "ready", 1)
	want(t, r.Complete(q.Identity, true, nil), "succeeded", "ready", 0)
	want(t, r.Complete(old.Identity, true, nil), "ignored", "ready", 0)
	if r.Snapshot() != (Store{}) {
		t.Fatal("stopped callback persisted state")
	}
}
func TestDrainDeadlinePermanentlyFencesRestart(t *testing.T) {
	r, _ := fixture(t)
	r.Start()
	q := r.PASE()
	r.Stop()
	for i := 0; i < DrainSteps-1; i++ {
		want(t, r.Step(), "accepted", "stopped", 1)
	}
	want(t, r.Step(), "drain-failed", "stopped", 1)
	want(t, r.Complete(q.Identity, true, nil), "drained", "stopped", 0)
	want(t, r.Restart(), "rejected", "stopped", 0)
	want(t, r.Start(), "rejected", "stopped", 0)
}
func TestStoreDetachedAndRetainedOnFailure(t *testing.T) {
	p := &fakePorts{}
	snapshot := Store{Record{"n-a"}}
	r, err := New(p, "n-a", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Commissioned.Node = "n-b"
	q := r.Restart()
	want(t, r.Complete(q.Identity, false, nil), "failed", "ready", 0)
	out := r.Snapshot()
	out.Commissioned.Node = "n-c"
	if r.Snapshot() != (Store{Record{"n-a"}}) {
		t.Fatal("lost or aliased retained state")
	}
	r.Stop()
	q = r.Restart()
	want(t, r.Complete(q.Identity, true, nil), "succeeded", "commissioned", 0)
	if _, err = New(p, "n-a", Store{Record{"n-b"}}); err == nil {
		t.Fatal("conflict accepted")
	}
	if _, err = New(nil, "n-a", Store{}); err == nil {
		t.Fatal("missing ports accepted")
	}
	if _, err = New(p, "", Store{}); err == nil {
		t.Fatal("missing identity accepted")
	}
}
func TestCountersAndLogicalTrace(t *testing.T) {
	r, _ := fixture(t)
	a := r.Start()
	b := r.PASE()
	c := r.Complete(b.Identity, true, nil)
	if a.Step != 1 || b.Step != 2 || c.Step != 3 {
		t.Fatal(a, b, c)
	}
	r.token = ^uint64(0)
	want(t, r.OpenWindow(true), "rejected", "paired", 0)
	r.Stop()
	want(t, r.Restart(), "rejected", "stopped", 0)
	r.generation = ^uint64(0)
	want(t, r.Start(), "rejected", "stopped", 0)
}
