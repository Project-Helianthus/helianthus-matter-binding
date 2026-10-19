package storeharness

import (
	"reflect"
	"strings"
	"testing"
)

var ka = Key{"n-a", "c-a", "p-a"}
var kb = Key{"n-a", "c-b", "p-a"}
var aa = Association{"b-a", "slot-a"}
var ab = Association{"b-b", "slot-b"}

func setup(t *testing.T, records Snapshot) (*Harness, *MemoryStore) {
	t.Helper()
	s := &MemoryStore{Records: clone(records), Unrelated: map[string]string{"opaque-sibling": "retained"}, Faults: map[Key]Outcome{}}
	h, err := New(s, 100)
	if err != nil {
		t.Fatal(err)
	}
	if g, err := h.Start(); err != nil || g != 1 {
		t.Fatalf("start %d %v", g, err)
	}
	return h, s
}
func begin(t *testing.T, h *Harness, k Key, a Association) Completion {
	t.Helper()
	c, err := h.Begin(k, a)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func equal(t *testing.T, got, want Snapshot) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot %+v, want %+v", got, want)
	}
}
func sentinel(t *testing.T, s *MemoryStore) {
	t.Helper()
	if !reflect.DeepEqual(s.Unrelated, map[string]string{"opaque-sibling": "retained"}) {
		t.Fatal("unrelated store key lost")
	}
}
func TestStoreFaults(t *testing.T) {
	for _, o := range []Outcome{PrecommitFailure, Confirmed, Ambiguous} {
		t.Run(map[Outcome]string{PrecommitFailure: "precommit", Confirmed: "confirmed", Ambiguous: "ambiguous"}[o], func(t *testing.T) {
			h, s := setup(t, Snapshot{kb: ab})
			s.Faults[ka] = o
			c := begin(t, h, ka, aa)
			equal(t, h.Exposed(), Snapshot{kb: ab})
			if again := begin(t, h, ka, aa); again != c || s.Writes != 1 || h.Pending() != 1 {
				t.Fatal("duplicate accumulated work")
			}
			if !h.Complete(c) || h.Pending() != 0 || h.Complete(c) {
				t.Fatal("completion ownership")
			}
			want := Snapshot{kb: ab}
			if o == Confirmed {
				want[ka] = aa
			}
			equal(t, h.Exposed(), want)
			equal(t, h.Accepted(), want)
			if o == PrecommitFailure {
				equal(t, s.Records, Snapshot{kb: ab})
			} else {
				equal(t, s.Records, Snapshot{ka: aa, kb: ab})
			}
			if o == Ambiguous {
				if _, err := h.Begin(ka, aa); err == nil || s.Writes != 1 {
					t.Fatal("blind retry")
				}
				if err := h.Reconcile(Snapshot{kb: ab}); err != nil {
					t.Fatal(err)
				}
				equal(t, h.Exposed(), Snapshot{kb: ab})
				if err := h.Reconcile(Snapshot{ka: aa, kb: ab}); err != nil {
					t.Fatal(err)
				}
				equal(t, h.Exposed(), Snapshot{ka: aa, kb: ab})
			}
			sentinel(t, s)
		})
	}
}
func TestReadFailureAndRestart(t *testing.T) {
	h, s := setup(t, Snapshot{kb: ab})
	h.Stop()
	s.ReadFailure = true
	if _, err := h.Start(); err == nil || s.Reads != 2 {
		t.Fatal("read failure admitted")
	}
	equal(t, h.Exposed(), Snapshot{})
	equal(t, h.Accepted(), Snapshot{kb: ab})
	s.ReadFailure = false
	if g, err := h.Start(); err != nil || g != 2 {
		t.Fatalf("restart %d %v", g, err)
	}
	equal(t, h.Exposed(), Snapshot{kb: ab})
	sentinel(t, s)
}
func TestRestartAfterEveryOutcome(t *testing.T) {
	for _, o := range []Outcome{PrecommitFailure, Confirmed, Ambiguous} {
		t.Run(map[Outcome]string{PrecommitFailure: "precommit", Confirmed: "clean", Ambiguous: "ambiguous"}[o], func(t *testing.T) {
			h, s := setup(t, Snapshot{kb: ab})
			s.Faults[ka] = o
			old := begin(t, h, ka, aa)
			h.Complete(old)
			h.Stop()
			if _, err := h.Start(); err != nil {
				t.Fatal(err)
			}
			want := Snapshot{kb: ab}
			if o == Confirmed {
				want[ka] = aa
			}
			equal(t, h.Exposed(), want)
			if h.Complete(old) {
				t.Fatal("old callback accepted")
			}
			if o == Ambiguous {
				snap, _ := s.Read()
				if err := h.Reconcile(snap); err != nil {
					t.Fatal(err)
				}
				snap[ka] = Association{"tamper", "tamper"}
				equal(t, h.Exposed(), Snapshot{ka: aa, kb: ab})
			}
			sentinel(t, s)
		})
	}
}
func TestCompletionEveryField(t *testing.T) {
	variants := map[string]func(*Completion){
		"node":        func(c *Completion) { c.Key.Node = "n-other" },
		"capability":  func(c *Completion) { c.Key.Capability = "c-other" },
		"composition": func(c *Completion) { c.Key.Composition = "p-other" },
		"generation":  func(c *Completion) { c.Generation++ },
		"token":       func(c *Completion) { c.Token++ },
		"binding":     func(c *Completion) { c.Association.Binding = "b-other" },
		"slot":        func(c *Completion) { c.Association.Slot = "slot-other" },
		"outcome":     func(c *Completion) { c.Outcome = Confirmed },
	}
	for name, change := range variants {
		t.Run(name, func(t *testing.T) {
			h, s := setup(t, Snapshot{kb: ab})
			s.Faults[ka] = Ambiguous
			good := begin(t, h, ka, aa)
			bad := good
			change(&bad)
			if h.Complete(bad) || h.Pending() != 1 {
				t.Fatal("mismatched completion consumed barrier")
			}
			equal(t, h.Exposed(), Snapshot{kb: ab})
			equal(t, h.Accepted(), Snapshot{kb: ab})
			if !h.Complete(good) {
				t.Fatal("good completion lost")
			}
			equal(t, h.Exposed(), Snapshot{kb: ab})
			sentinel(t, s)
		})
	}
}
func TestStaleRetryAndGeneration(t *testing.T) {
	h, s := setup(t, Snapshot{})
	s.Faults[ka] = PrecommitFailure
	first := begin(t, h, ka, aa)
	h.Complete(first)
	second := begin(t, h, ka, aa)
	if first.Token == second.Token || h.Complete(first) || h.Pending() != 1 {
		t.Fatal("stale retry")
	}
	h.Stop()
	if _, err := h.Start(); err == nil {
		t.Fatal("restart before drain")
	}
	if !h.Complete(second) {
		t.Fatal("drain failed")
	}
	if g, err := h.Start(); err != nil || g != 2 {
		t.Fatal("restart")
	}
	current := begin(t, h, ka, aa)
	old := current
	old.Generation = 1
	if h.Complete(old) || h.Pending() != 1 {
		t.Fatal("old generation")
	}
	h.Complete(current)
	equal(t, h.Exposed(), Snapshot{})
}
func TestSnapshotVariants(t *testing.T) {
	variants := map[string]Snapshot{
		"missing":           nil,
		"empty-node":        {Key{"", "c-a", "p-a"}: aa},
		"empty-capability":  {Key{"n-a", "", "p-a"}: aa},
		"empty-composition": {Key{"n-a", "c-a", ""}: aa},
		"empty-binding":     {ka: Association{"", "slot-a"}},
		"empty-slot":        {ka: Association{"b-a", ""}},
		"changed-binding":   {kb: Association{"replacement", "slot-b"}},
		"changed-slot":      {kb: Association{"b-b", "replacement"}},
		"binding-conflict":  {ka: Association{"b-b", "slot-a"}},
		"slot-conflict":     {ka: Association{"b-a", "slot-b"}},
	}
	for name, snap := range variants {
		t.Run(name, func(t *testing.T) {
			h, s := setup(t, Snapshot{kb: ab})
			if err := h.Reconcile(snap); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
			equal(t, h.Accepted(), Snapshot{kb: ab})
			equal(t, h.Exposed(), Snapshot{kb: ab})
			h.Stop()
			s.Records = snap
			if _, err := h.Start(); err == nil {
				t.Fatal("invalid restart admitted")
			}
			equal(t, h.Accepted(), Snapshot{kb: ab})
			equal(t, h.Exposed(), Snapshot{})
			sentinel(t, s)
		})
	}
	h, _ := setup(t, Snapshot{kb: ab})
	if err := h.Reconcile(Snapshot{}); err != nil {
		t.Fatal(err)
	}
	equal(t, h.Accepted(), Snapshot{kb: ab})
	equal(t, h.Exposed(), Snapshot{})
}
func TestKeyAxesAreDistinct(t *testing.T) {
	for name, k := range map[string]Key{"node": {"n-b", "c-a", "p-a"}, "capability": {"n-a", "c-b", "p-a"}, "composition": {"n-a", "c-a", "p-b"}} {
		t.Run(name, func(t *testing.T) {
			h, _ := setup(t, Snapshot{ka: aa})
			c := begin(t, h, k, ab)
			h.Complete(c)
			equal(t, h.Accepted(), Snapshot{ka: aa, k: ab})
			equal(t, h.Exposed(), Snapshot{ka: aa, k: ab})
		})
	}
}
func TestBoundsAndDetachedState(t *testing.T) {
	h, s := setup(t, Snapshot{})
	a := begin(t, h, ka, aa)
	b := begin(t, h, kb, ab)
	if h.Pending() != 2 || s.Writes != 2 {
		t.Fatal("callback counts")
	}
	if _, err := h.Begin(Key{"n-c", "c-c", "p-c"}, Association{"b-c", "slot-c"}); err == nil || s.Writes != 2 {
		t.Fatal("budget admitted")
	}
	if err := h.Reconcile(Snapshot{}); err == nil {
		t.Fatal("reconcile with pending callbacks")
	}
	if _, err := h.Start(); err == nil || s.Reads != 1 {
		t.Fatal("duplicate start")
	}
	h.Complete(b)
	h.Complete(a)
	equal(t, h.Exposed(), Snapshot{ka: aa, kb: ab})
	copy := h.Accepted()
	delete(copy, ka)
	equal(t, h.Accepted(), Snapshot{ka: aa, kb: ab})
	if _, err := h.Begin(ka, aa); err == nil {
		t.Fatal("accepted identity rewritten")
	}
	h2, s2 := setup(t, Snapshot{})
	if _, err := h2.Begin(ka, Association{strings.Repeat("b", 101), "slot-a"}); err == nil || s2.Writes != 0 {
		t.Fatal("byte cap")
	}
	if _, err := New(nil, 10); err == nil {
		t.Fatal("nil store")
	}
	if _, err := New(s, 0); err == nil {
		t.Fatal("zero cap")
	}
}
func TestStopDrainAndTimeout(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "drained", true: "timeout"}[timeout], func(t *testing.T) {
			h, s := setup(t, Snapshot{kb: ab})
			c := begin(t, h, ka, aa)
			h.Stop()
			h.Stop()
			if _, err := h.Begin(ka, aa); err == nil {
				t.Fatal("stopped write")
			}
			for i := 1; i < 8; i++ {
				if err := h.Step(); err != nil {
					t.Fatal(err)
				}
			}
			if timeout {
				if err := h.Step(); err == nil {
					t.Fatal("missing deadline")
				}
			}
			h.Complete(c)
			equal(t, h.Exposed(), Snapshot{})
			equal(t, h.Accepted(), Snapshot{kb: ab})
			_, err := h.Start()
			if timeout {
				if err == nil || h.Step() == nil {
					t.Fatal("timeout reopened")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				equal(t, h.Exposed(), Snapshot{kb: ab})
				if err := h.Reconcile(Snapshot{ka: aa, kb: ab}); err != nil {
					t.Fatal(err)
				}
			}
			sentinel(t, s)
		})
	}
}
func TestUnrelatedFaultAndConflictingRecovery(t *testing.T) {
	h, s := setup(t, Snapshot{})
	s.Faults[kb] = PrecommitFailure
	a := begin(t, h, ka, aa)
	b := begin(t, h, kb, ab)
	h.Complete(b)
	h.Complete(a)
	equal(t, h.Exposed(), Snapshot{ka: aa})
	equal(t, s.Records, Snapshot{ka: aa})
	sentinel(t, s)
	h2, s2 := setup(t, Snapshot{kb: ab})
	s2.Faults[ka] = Ambiguous
	h2.Complete(begin(t, h2, ka, aa))
	for _, a := range []Association{{"replacement", "slot-a"}, {"b-a", "replacement"}} {
		if h2.Reconcile(Snapshot{ka: a, kb: ab}) == nil {
			t.Fatal("ambiguous identity replaced")
		}
		equal(t, h2.Accepted(), Snapshot{kb: ab})
		equal(t, h2.Exposed(), Snapshot{kb: ab})
	}
}

func TestRestartSnapshotMatrix(t *testing.T) {
	for _, o := range []Outcome{PrecommitFailure, Confirmed, Ambiguous} {
		for _, variant := range []string{"valid", "absent", "missing", "corrupt", "conflicting"} {
			name := map[Outcome]string{PrecommitFailure: "precommit", Confirmed: "confirmed", Ambiguous: "ambiguous"}[o] + "/" + variant
			t.Run(name, func(t *testing.T) {
				h, s := setup(t, Snapshot{kb: ab})
				s.Faults[ka] = o
				old := begin(t, h, ka, aa)
				h.Complete(old)
				h.Stop()
				accepted := Snapshot{kb: ab}
				if o == Confirmed {
					accepted[ka] = aa
				}
				snap := Snapshot{kb: ab}
				if o != PrecommitFailure {
					snap[ka] = aa
				}
				switch variant {
				case "absent":
					snap = Snapshot{}
				case "missing":
					snap = nil
				case "corrupt":
					snap[Key{"n-a", "c-a", ""}] = aa
				case "conflicting":
					snap[ka] = Association{"b-other", "slot-b"}
				}
				s.Records = snap
				_, err := h.Start()
				invalid := variant == "missing" || variant == "corrupt" || variant == "conflicting"
				if (err != nil) != invalid {
					t.Fatalf("restart error %v", err)
				}
				want := Snapshot{}
				if !invalid && variant != "absent" {
					want[kb] = ab
					if o == Confirmed {
						want[ka] = aa
					}
				}
				equal(t, h.Exposed(), want)
				equal(t, h.Accepted(), accepted)
				if h.Complete(old) {
					t.Fatal("old callback survived recovery")
				}
				equal(t, h.Exposed(), want)
				equal(t, h.Accepted(), accepted)
				sentinel(t, s)
			})
		}
	}
}
func TestPendingMismatchAndDetachedRead(t *testing.T) {
	h, s := setup(t, Snapshot{})
	c := begin(t, h, ka, aa)
	for _, a := range []Association{{"b-other", "slot-a"}, {"b-a", "slot-other"}} {
		if _, err := h.Begin(ka, a); err == nil || s.Writes != 1 || h.Pending() != 1 {
			t.Fatal("pending identity replaced")
		}
	}
	snap, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	delete(snap, ka)
	equal(t, s.Records, Snapshot{ka: aa})
	h.Complete(c)
	detached := h.Exposed()
	detached[ka] = ab
	equal(t, h.Exposed(), Snapshot{ka: aa})
	h.Stop()
	if _, err := h.Begin(kb, ab); err == nil || s.Writes != 1 {
		t.Fatal("stopped side effect")
	}
}
