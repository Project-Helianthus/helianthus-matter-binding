package storeharness

import (
	"fmt"
	"reflect"
	"testing"
)

// aa11Port is a synthetic namespace store with atomic per-record operations.
// Fabric values are opaque public fixture strings, never commissioned material.
type aa11Port struct {
	endpoints                       map[StoreScope]map[Key]Request
	fabrics                         map[FabricScope]string
	sentinel                        map[string]string
	outcome                         Outcome
	failRead                        bool
	reads, writes, removals, resets int
	lastStore                       StoreScope
	lastFabric                      FabricScope
}

func aa11Fixture() (*aa11Port, StoreScope) {
	scope := StoreScope{"store-a", "node-a"}
	return &aa11Port{endpoints: map[StoreScope]map[Key]Request{scope: {}}, fabrics: map[FabricScope]string{}, sentinel: map[string]string{"unrelated": "retained"}, outcome: Confirmed}, scope
}
func (p *aa11Port) ReadEndpoints(s StoreScope) (map[Key]Request, error) {
	p.reads++
	p.lastStore = s
	if p.failRead {
		return nil, fmt.Errorf("injected read failure")
	}
	records, ok := p.endpoints[s]
	if !ok || records == nil {
		return nil, nil
	}
	result := map[Key]Request{}
	for k, r := range records {
		result[k] = r
	}
	return result, nil
}
func (p *aa11Port) CommitEndpoint(s StoreScope, r Request) Outcome {
	p.writes++
	p.lastStore = s
	if p.outcome != PrecommitFailure {
		if p.endpoints[s] == nil {
			p.endpoints[s] = map[Key]Request{}
		}
		p.endpoints[s][r.Key] = r
	}
	return p.outcome
}
func (p *aa11Port) RemoveFabric(s FabricScope) Outcome {
	p.removals++
	p.lastFabric = s
	if p.outcome != PrecommitFailure {
		delete(p.fabrics, s)
	}
	return p.outcome
}
func (p *aa11Port) ResetNode(s StoreScope) Outcome {
	p.resets++
	p.lastStore = s
	if p.outcome != PrecommitFailure {
		delete(p.endpoints, s)
		for k := range p.fabrics {
			if k.StoreScope == s {
				delete(p.fabrics, k)
			}
		}
	}
	return p.outcome
}
func aa11Request(scope StoreScope, suffix string) Request {
	return Request{Identity{Key{scope.Node, "cap-" + suffix, "composition-a"}, 1, 1}, Association{"binding-" + suffix, "slot-" + suffix}}
}
func aa11Harness(t *testing.T, p *aa11Port, scope StoreScope) (*SDKStore, *Harness) {
	t.Helper()
	s, err := NewSDKStore(p, scope)
	if err != nil {
		t.Fatal(err)
	}
	h, err := New(s, 200)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Stop)
	return s, h
}
func aa11Begin(t *testing.T, h *Harness, r Request) Completion {
	t.Helper()
	c, err := h.Begin(r.Key, r.Association)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func aa11Sentinel(t *testing.T, p *aa11Port) {
	t.Helper()
	if !reflect.DeepEqual(p.sentinel, map[string]string{"unrelated": "retained"}) {
		t.Fatal("sentinel changed")
	}
}

func TestAA11PortOutcomes(t *testing.T) {
	for _, o := range []Outcome{Confirmed, PrecommitFailure, Ambiguous} {
		t.Run(fmt.Sprint(o), func(t *testing.T) {
			p, scope := aa11Fixture()
			sibling := aa11Request(scope, "b")
			p.endpoints[scope][sibling.Key] = sibling
			s, h := aa11Harness(t, p, scope)
			p.outcome = o
			r := aa11Request(scope, "a")
			c := aa11Begin(t, h, r)
			equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
			if again := aa11Begin(t, h, r); again != c || p.writes != 1 {
				t.Fatal("pending retry")
			}
			if !h.Complete(c) || h.Complete(c) {
				t.Fatal("completion ownership")
			}
			want := Snapshot{sibling.Key: sibling.Association}
			if o == Confirmed {
				want[r.Key] = r.Association
			}
			equal(t, h.Exposed(), want)
			if p.endpoints[scope][sibling.Key] != sibling {
				t.Fatal("sibling replaced by write")
			}
			if _, ok := p.endpoints[scope][r.Key]; ok != (o != PrecommitFailure) {
				t.Fatal("commit outcome conflated")
			}
			if o == Ambiguous {
				if _, err := h.Begin(r.Key, r.Association); err == nil || p.writes != 1 {
					t.Fatal("blind retry")
				}
				h.Stop()
				if _, err := h.Start(); err != nil {
					t.Fatal(err)
				}
				equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
				delete(p.endpoints[scope], r.Key)
				reads := p.reads
				if err := s.Reconcile(h); err != nil || p.reads != reads+1 {
					t.Fatal("bounded absent readback", err)
				}
				if _, err := h.Begin(r.Key, r.Association); err == nil {
					t.Fatal("absence released ambiguous reservation")
				}
				p.endpoints[scope][r.Key] = c.Request
				if err := s.Reconcile(h); err != nil {
					t.Fatal(err)
				}
				equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association, r.Key: r.Association})
			}
			aa11Sentinel(t, p)
		})
	}
}

// Each mutation changes exactly one completion field. The valid barrier must
// remain usable; dropping that field from Harness.Complete equality is caught.
func TestAA11CompletionFields(t *testing.T) {
	variants := map[string]func(*Completion){
		"node":        func(c *Completion) { c.Key.Node = "node-other" },
		"capability":  func(c *Completion) { c.Key.Capability = "cap-other" },
		"composition": func(c *Completion) { c.Key.Composition = "composition-other" },
		"binding":     func(c *Completion) { c.Association.Binding = "binding-other" },
		"slot":        func(c *Completion) { c.Association.Slot = "slot-other" },
		"generation":  func(c *Completion) { c.Generation++ },
		"token":       func(c *Completion) { c.Token++ },
		"outcome":     func(c *Completion) { c.Outcome = Confirmed },
	}
	for name, change := range variants {
		t.Run(name, func(t *testing.T) {
			p, scope := aa11Fixture()
			_, h := aa11Harness(t, p, scope)
			p.outcome = Ambiguous
			good := aa11Begin(t, h, aa11Request(scope, "a"))
			bad := good
			change(&bad)
			if h.Complete(bad) || h.Pending() != 1 {
				t.Fatal("changed identity consumed barrier")
			}
			equal(t, h.Exposed(), Snapshot{})
			if !h.Complete(good) {
				t.Fatal("valid completion lost")
			}
			equal(t, h.Exposed(), Snapshot{})
			aa11Sentinel(t, p)
		})
	}
}

// These independently changed stored fields catch omitting each field from
// exact readback equality (or key/namespace validation). Others remain valid.
func TestAA11StoredIdentityFields(t *testing.T) {
	variants := map[string]func(*Request){
		"node":        func(r *Request) { r.Key.Node = "node-other" },
		"capability":  func(r *Request) { r.Key.Capability = "cap-other" },
		"composition": func(r *Request) { r.Key.Composition = "composition-other" },
		"binding":     func(r *Request) { r.Association.Binding = "binding-other" },
		"slot":        func(r *Request) { r.Association.Slot = "slot-other" },
		"generation":  func(r *Request) { r.Generation++ },
		"token":       func(r *Request) { r.Token++ },
	}
	for name, change := range variants {
		t.Run(name, func(t *testing.T) {
			p, scope := aa11Fixture()
			sibling := aa11Request(scope, "b")
			p.endpoints[scope][sibling.Key] = sibling
			s, h := aa11Harness(t, p, scope)
			p.outcome = Ambiguous
			c := aa11Begin(t, h, aa11Request(scope, "a"))
			h.Complete(c)
			bad := c.Request
			change(&bad)
			p.endpoints[scope][c.Key] = bad
			if err := s.Reconcile(h); err == nil {
				t.Fatal("changed stored identity accepted")
			}
			equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
			if p.endpoints[scope][sibling.Key] != sibling {
				t.Fatal("sibling changed")
			}
			p.endpoints[scope][c.Key] = c.Request
			if err := s.Reconcile(h); err != nil {
				t.Fatal(err)
			}
			equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association, c.Key: c.Association})
			aa11Sentinel(t, p)
		})
	}
}

func TestAA11SnapshotsAndReadFailures(t *testing.T) {
	for _, variant := range []string{"nil", "corrupt", "conflict", "read-failure", "absent"} {
		t.Run(variant, func(t *testing.T) {
			p, scope := aa11Fixture()
			sibling := aa11Request(scope, "b")
			p.endpoints[scope][sibling.Key] = sibling
			s, h := aa11Harness(t, p, scope)
			saved := p.endpoints[scope]
			bad := aa11Request(scope, "a")
			switch variant {
			case "nil":
				p.endpoints[scope] = nil
			case "corrupt":
				bad.Association.Binding = ""
				p.endpoints[scope][bad.Key] = bad
			case "conflict":
				bad.Association.Slot = sibling.Association.Slot
				p.endpoints[scope][bad.Key] = bad
			case "read-failure":
				p.failRead = true
			case "absent":
				p.endpoints[scope] = map[Key]Request{}
			}
			reads := p.reads
			err := s.Reconcile(h)
			if (err != nil) != (variant != "absent") || p.reads != reads+1 {
				t.Fatal("snapshot result", err)
			}
			equal(t, h.Accepted(), Snapshot{sibling.Key: sibling.Association})
			want := Snapshot{sibling.Key: sibling.Association}
			if variant == "absent" {
				want = Snapshot{}
			}
			equal(t, h.Exposed(), want)
			h.Stop()
			_, err = h.Start()
			if (err != nil) != (variant != "absent") {
				t.Fatal("restart validation", err)
			}
			equal(t, h.Exposed(), Snapshot{})
			p.failRead = false
			delete(saved, bad.Key)
			p.endpoints[scope] = saved
			if !h.running {
				if _, err = h.Start(); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.Reconcile(h); err != nil {
				t.Fatal(err)
			}
			equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
			aa11Sentinel(t, p)
		})
	}
}

func TestAA11StaleRestartAndBoundedRead(t *testing.T) {
	p, scope := aa11Fixture()
	s, h := aa11Harness(t, p, scope)
	p.outcome = PrecommitFailure
	old := aa11Begin(t, h, aa11Request(scope, "a"))
	h.Complete(old)
	h.Stop()
	if _, err := h.Start(); err != nil {
		t.Fatal(err)
	}
	current := aa11Begin(t, h, aa11Request(scope, "a"))
	if old.Generation == current.Generation || old.Token == current.Token || h.Complete(old) {
		t.Fatal("stale restart completion")
	}
	reads := p.reads
	if err := s.Reconcile(h); err == nil || p.reads != reads {
		t.Fatal("read before callback drain")
	}
	if !h.Complete(current) {
		t.Fatal("valid current callback lost")
	}
	if err := s.Reconcile(h); err != nil || p.reads != reads+1 {
		t.Fatal("explicit readback", err)
	}
	aa11Sentinel(t, p)
}

// Every namespace field varies independently. Exact target deletion and the
// three axis-neighbors detect dropping Store, Node or Fabric from port routing.
func TestAA11NamespaceScopes(t *testing.T) {
	for _, axis := range []string{"store", "node", "fabric"} {
		t.Run(axis, func(t *testing.T) {
			p, scope := aa11Fixture()
			target := FabricScope{scope, "fabric-a"}
			neighbor := target
			switch axis {
			case "store":
				neighbor.Store = "store-b"
			case "node":
				neighbor.Node = "node-b"
			case "fabric":
				neighbor.Fabric = "fabric-b"
			}
			p.fabrics[target] = "target"
			p.fabrics[neighbor] = "neighbor"
			sibling := aa11Request(scope, "b")
			p.endpoints[scope][sibling.Key] = sibling
			s, h := aa11Harness(t, p, scope)
			if p.lastStore != scope {
				t.Fatal("read namespace")
			}
			c := aa11Begin(t, h, aa11Request(scope, "a"))
			if p.lastStore != scope || !h.Complete(c) {
				t.Fatal("write namespace")
			}
			if s.RemoveFabric(target.Fabric) != Confirmed || p.lastFabric != target {
				t.Fatal("fabric namespace")
			}
			if _, ok := p.fabrics[target]; ok || p.fabrics[neighbor] != "neighbor" || len(p.endpoints[scope]) != 2 {
				t.Fatal("fabric removal exceeded scope")
			}
			// A reset owns every fabric of this exact node, but no other node/store.
			p.fabrics[target] = "target"
			other := StoreScope{"store-other", "node-other"}
			p.endpoints[other] = map[Key]Request{}
			p.fabrics[FabricScope{other, "fabric-a"}] = "other"
			if s.ResetNode(h) != PrecommitFailure || p.resets != 0 {
				t.Fatal("reset while running")
			}
			h.Stop()
			if s.ResetNode(h) != Confirmed || p.resets != 1 || p.lastStore != scope {
				t.Fatal("explicit reset")
			}
			if _, ok := p.endpoints[scope]; ok {
				t.Fatal("node endpoints retained after reset")
			}
			if _, ok := p.endpoints[other]; !ok {
				t.Fatal("other node reset")
			}
			for k, v := range p.fabrics {
				if k.StoreScope == scope || v == "" {
					t.Fatal("node fabric reset incomplete")
				}
			}
			if axis != "fabric" && p.fabrics[neighbor] != "neighbor" {
				t.Fatal("neighbor reset")
			}
			if p.fabrics[FabricScope{other, "fabric-a"}] != "other" {
				t.Fatal("other fabric lost")
			}
			if _, err := h.Start(); err == nil {
				t.Fatal("old lifecycle restarted")
			}
			aa11Sentinel(t, p)
		})
	}
}

func TestAA11LifecycleFaults(t *testing.T) {
	for _, o := range []Outcome{PrecommitFailure, Ambiguous} {
		t.Run(fmt.Sprint(o), func(t *testing.T) {
			p, scope := aa11Fixture()
			s, h := aa11Harness(t, p, scope)
			p.outcome = o
			p.fabrics[FabricScope{scope, "fabric-a"}] = "opaque"
			if got := s.RemoveFabric("fabric-a"); got != o {
				t.Fatal(got)
			}
			if o == Ambiguous {
				if s.RemoveFabric("fabric-a") != PrecommitFailure || p.removals != 1 {
					t.Fatal("blind lifecycle retry")
				}
				if _, err := s.Read(); err == nil {
					t.Fatal("ambiguous lifecycle unfenced")
				}
			}
			h.Stop()
			p2, scope2 := aa11Fixture()
			s2, h2 := aa11Harness(t, p2, scope2)
			h2.Stop()
			p2.outcome = o
			if s2.ResetNode(h2) != o {
				t.Fatal("reset outcome")
			}
			if o == Ambiguous && (s2.ResetNode(h2) != PrecommitFailure || p2.resets != 1) {
				t.Fatal("ambiguous reset retried")
			}
			aa11Sentinel(t, p)
			aa11Sentinel(t, p2)
		})
	}
}

func TestAA11WithdrawalRetainsEndpointAndFabric(t *testing.T) {
	p, scope := aa11Fixture()
	sibling := aa11Request(scope, "b")
	p.endpoints[scope][sibling.Key] = sibling
	fabric := FabricScope{scope, "fabric-a"}
	p.fabrics[fabric] = "opaque"
	s, h := aa11Harness(t, p, scope)
	c := aa11Begin(t, h, aa11Request(scope, "a"))
	if s.Withdraw(h, c.Key) == nil {
		t.Fatal("pending withdrawal permitted late exposure")
	}
	if !h.Complete(c) {
		t.Fatal("completion lost")
	}
	if err := s.Withdraw(h, c.Key); err != nil {
		t.Fatal(err)
	}
	equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
	equal(t, h.Accepted(), Snapshot{sibling.Key: sibling.Association, c.Key: c.Association})
	if len(p.endpoints[scope]) != 2 || p.fabrics[fabric] != "opaque" || p.removals != 0 || p.resets != 0 {
		t.Fatal("withdrawal erased storage")
	}
	h.Stop()
	if _, err := h.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.Reconcile(h); err != nil {
		t.Fatal(err)
	}
	equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
	aa11Sentinel(t, p)
}

func TestAA11WithdrawalReservationsAcrossHarnesses(t *testing.T) {
	for _, outcome := range []Outcome{Confirmed, Ambiguous} {
		for _, association := range []string{"binding", "slot", "distinct"} {
			t.Run(fmt.Sprint(outcome)+"/"+association, func(t *testing.T) {
				p, scope := aa11Fixture()
				s, h := aa11Harness(t, p, scope)
				p.outcome = outcome
				retained := aa11Begin(t, h, aa11Request(scope, "a"))
				if !h.Complete(retained) {
					t.Fatal("retained completion lost")
				}
				if err := s.Withdraw(h, retained.Key); err != nil {
					t.Fatal(err)
				}
				h.Stop()
				next, err := New(s, 200)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(next.Stop)
				if _, err := next.Start(); err != nil {
					t.Fatal(err)
				}
				equal(t, next.Accepted(), Snapshot{})
				equal(t, next.Exposed(), Snapshot{})
				r := aa11Request(scope, "b")
				switch association {
				case "binding":
					r.Association.Binding = retained.Association.Binding
				case "slot":
					r.Association.Slot = retained.Association.Slot
				}
				p.outcome = Confirmed
				c := aa11Begin(t, next, r)
				wantOutcome, wantWrites := PrecommitFailure, 1
				wantRecords := map[Key]Request{retained.Key: retained.Request}
				wantExposure := Snapshot{}
				if association == "distinct" {
					wantOutcome, wantWrites = Confirmed, 2
					wantRecords[c.Key] = c.Request
					wantExposure[c.Key] = c.Association
				}
				if c.Outcome != wantOutcome || p.writes != wantWrites {
					t.Fatalf("write outcome=%v calls=%d; want outcome=%v calls=%d", c.Outcome, p.writes, wantOutcome, wantWrites)
				}
				if !next.Complete(c) {
					t.Fatal("new completion lost")
				}
				equal(t, next.Exposed(), wantExposure)
				if !reflect.DeepEqual(p.endpoints[scope], wantRecords) || !reflect.DeepEqual(s.expected, wantRecords) {
					t.Fatal("retained reservations or endpoint records changed")
				}
				aa11Sentinel(t, p)
			})
		}
	}
}

func TestAA11AmbiguousCompletionWithdrawal(t *testing.T) {
	p, scope := aa11Fixture()
	sibling := aa11Request(scope, "b")
	p.endpoints[scope][sibling.Key] = sibling
	fabric := FabricScope{scope, "fabric-a"}
	p.fabrics[fabric] = "opaque"
	s, h := aa11Harness(t, p, scope)
	p.outcome = Ambiguous
	c := aa11Begin(t, h, aa11Request(scope, "a"))
	if err := s.Withdraw(h, c.Key); err == nil {
		t.Fatal("pending withdrawal permitted")
	}
	if !h.Complete(c) {
		t.Fatal("completion lost")
	}
	equal(t, h.Accepted(), Snapshot{sibling.Key: sibling.Association})
	if err := s.Withdraw(h, c.Key); err != nil {
		t.Fatal(err)
	}
	for _, reconcile := range []bool{false, true} {
		if reconcile {
			if err := s.Reconcile(h); err != nil {
				t.Fatal(err)
			}
			equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
		}
		h.Stop()
		if _, err := h.Start(); err != nil {
			t.Fatal(err)
		}
		equal(t, h.Exposed(), Snapshot{sibling.Key: sibling.Association})
		if _, err := h.Begin(c.Key, c.Association); err == nil {
			t.Fatal("withdrawn identity allowed retry")
		}
		other := aa11Request(scope, "c")
		if _, err := h.Begin(other.Key, other.Association); err == nil {
			t.Fatal("withdrawal released reservation")
		}
	}
	if p.writes != 1 || !reflect.DeepEqual(p.endpoints[scope], map[Key]Request{c.Key: c.Request, sibling.Key: sibling}) || p.fabrics[fabric] != "opaque" || p.removals != 0 || p.resets != 0 {
		t.Fatal("withdrawal changed retained storage")
	}
	aa11Sentinel(t, p)
}

func TestAA11RestoredRecordIdentityFence(t *testing.T) {
	for _, field := range []string{"generation", "token"} {
		t.Run(field, func(t *testing.T) {
			p, scope := aa11Fixture()
			r := aa11Request(scope, "a")
			p.endpoints[scope][r.Key] = r
			s, h := aa11Harness(t, p, scope)
			bad := r
			if field == "generation" {
				bad.Generation++
			} else {
				bad.Token++
			}
			p.endpoints[scope][r.Key] = bad
			if err := s.Reconcile(h); err == nil {
				t.Fatal("restored identity changed")
			}
			equal(t, h.Accepted(), Snapshot{r.Key: r.Association})
			h.Stop()
			if _, err := h.Start(); err == nil {
				t.Fatal("restart exposed changed restored identity")
			}
			equal(t, h.Exposed(), Snapshot{})
			p.endpoints[scope][r.Key] = r
			if _, err := h.Start(); err != nil {
				t.Fatal(err)
			}
			equal(t, h.Exposed(), Snapshot{r.Key: r.Association})
		})
	}
}

func TestAA11NamespaceRoutingFields(t *testing.T) {
	for _, axis := range []string{"store", "node", "fabric"} {
		t.Run(axis, func(t *testing.T) {
			p, base := aa11Fixture()
			target := base
			fabric := "fabric-a"
			switch axis {
			case "store":
				target.Store = "store-b"
			case "node":
				target.Node = "node-b"
			case "fabric":
				fabric = "fabric-b"
			}
			p.endpoints[target] = map[Key]Request{}
			s, h := aa11Harness(t, p, target)
			if p.lastStore != target {
				t.Fatal("read dropped namespace field")
			}
			r := aa11Request(target, "a")
			c := aa11Begin(t, h, r)
			if p.lastStore != target || p.endpoints[target][r.Key] != c.Request {
				t.Fatal("commit dropped namespace field")
			}
			h.Complete(c)
			key := FabricScope{target, fabric}
			p.fabrics[key] = "target"
			if s.RemoveFabric(fabric) != Confirmed || p.lastFabric != key {
				t.Fatal("fabric routing dropped namespace field")
			}
			if _, ok := p.fabrics[key]; ok {
				t.Fatal("target fabric not removed")
			}
			h.Stop()
			if s.ResetNode(h) != Confirmed || p.lastStore != target {
				t.Fatal("reset dropped namespace field")
			}
			aa11Sentinel(t, p)
		})
	}
}

func TestAA11DistinctEndpointKeyAxes(t *testing.T) {
	for _, axis := range []string{"capability", "composition"} {
		t.Run(axis, func(t *testing.T) {
			p, scope := aa11Fixture()
			a := aa11Request(scope, "a")
			p.endpoints[scope][a.Key] = a
			_, h := aa11Harness(t, p, scope)
			b := a
			switch axis {
			case "capability":
				b.Key.Capability = "cap-b"
			case "composition":
				b.Key.Composition = "composition-b"
			}
			b.Association = Association{"binding-b", "slot-b"}
			c := aa11Begin(t, h, b)
			if !h.Complete(c) {
				t.Fatal("valid distinct key completion")
			}
			equal(t, h.Exposed(), Snapshot{a.Key: a.Association, b.Key: b.Association})
			aa11Sentinel(t, p)
		})
	}
}
