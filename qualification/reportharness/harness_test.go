package reportharness

import (
	"reflect"
	"strings"
	"testing"
)

type fakeSink struct{ calls []Report }

func (s *fakeSink) Dispatch(r Report) { s.calls = append(s.calls, r) }

var fixtureKey = Key{"n-a", "c-a", "p-a"}
var fixtureAssociation = Association{"b-a", "slot-a"}

func fixture(t *testing.T, cap int) (*Harness, *fakeSink) {
	t.Helper()
	s := &fakeSink{}
	h, err := New(s, map[Key]Association{fixtureKey: fixtureAssociation}, cap)
	if err != nil {
		t.Fatal(err)
	}
	if g, err := h.Start(); err != nil || g != 1 {
		t.Fatalf("start: %d %v", g, err)
	}
	return h, s
}
func report(n uint64) Report {
	return Report{Identity: Identity{fixtureKey, 1, n, n, "native-a"}, State: State{"retained", "partial", "Hypothesis", "unqualified", "stale"}, Payload: "abc"}
}
func submit(t *testing.T, h *Harness, r Report) {
	t.Helper()
	if err := h.Submit(r); err != nil {
		t.Fatal(err)
	}
}
func outcome(t *testing.T, got []Result, r Report, want string) {
	t.Helper()
	if !reflect.DeepEqual(got, []Result{{r, want}}) {
		t.Fatalf("result: %+v, want %s for %+v", got, want, r)
	}
}
func idle(t *testing.T, h *Harness) {
	t.Helper()
	c := h.Counts()
	if c.Queue != 0 || c.Flight != 0 || c.Bytes != 0 {
		t.Fatalf("owned work: %+v", c)
	}
}

func TestReportHarnessFIFOAndBudgets(t *testing.T) {
	h, s := fixture(t, 1000)
	for n := uint64(1); n <= 3; n++ {
		submit(t, h, report(n))
		if n == 1 && !h.Dispatch() {
			t.Fatal("dispatch barrier")
		}
	}
	if len(s.calls) != 1 || h.Dispatch() {
		t.Fatal("flight bound")
	}
	before := h.Counts()
	if err := h.Submit(report(4)); err == nil {
		t.Fatal("full queue accepted")
	}
	if h.Counts() != before {
		t.Fatal("rejection changed counters")
	}
	if got, _ := h.Source(fixtureKey); got != report(3) {
		t.Fatal("rejection changed evidence")
	}
	// Independent count: key 9, evidence 8, payload 3, source axes 41 = 61.
	if got := h.HighWater(); got != (Counters{2, 1, 183, 61}) {
		t.Fatalf("high water %+v", got)
	}
	for n := uint64(1); n <= 3; n++ {
		outcome(t, h.Complete(report(n).Identity, true), report(n), "received")
		if got := h.Complete(report(n).Identity, false); len(got) != 0 {
			t.Fatal("duplicate terminal")
		}
		if n < 3 && !h.Dispatch() {
			t.Fatal("next dispatch")
		}
	}
	if !reflect.DeepEqual(s.calls, []Report{report(1), report(2), report(3)}) {
		t.Fatalf("FIFO: %+v", s.calls)
	}
	h.Stop()
	idle(t, h)
	if h.Dispatch() || len(s.calls) != 3 {
		t.Fatal("automatic replay")
	}
}

func TestReportHarnessByteRejection(t *testing.T) {
	h, s := fixture(t, 122)
	r := report(1)
	r.Payload = strings.Repeat("x", 127)
	if h.Submit(r) == nil || len(s.calls) != 0 {
		t.Fatal("oversized admission")
	}
	idle(t, h)
	if _, ok := h.Source(fixtureKey); ok {
		t.Fatal("rejected source retained")
	}
	submit(t, h, report(1))
	h.Dispatch()
	submit(t, h, report(2))
	before := h.Counts()
	if h.Submit(report(3)) == nil || h.Counts() != before {
		t.Fatal("aggregate byte cap")
	}
	if h.HighWater().Bytes != 122 {
		t.Fatal("byte high water")
	}
	outcome(t, h.Complete(report(1).Identity, false), report(1), "failed")
	h.Dispatch()
	outcome(t, h.Complete(report(2).Identity, true), report(2), "received")
	h.Stop()
	idle(t, h)
}

func TestReportHarnessCancellation(t *testing.T) {
	h, s := fixture(t, 1000)
	submit(t, h, report(1))
	h.Dispatch()
	submit(t, h, report(2))
	outcome(t, h.Cancel(report(2).Identity), report(2), "cancelled")
	if len(h.Cancel(report(2).Identity)) != 0 {
		t.Fatal("queued duplicate terminal")
	}
	outcome(t, h.Cancel(report(1).Identity), report(1), "delivery-unknown")
	if len(h.Cancel(report(1).Identity)) != 0 || h.Counts().Flight != 1 {
		t.Fatal("claimed retraction")
	}
	submit(t, h, report(3))
	if h.Dispatch() {
		t.Fatal("released cancelled flight before receipt")
	}
	if len(h.Complete(report(1).Identity, true)) != 0 {
		t.Fatal("late receipt rewrote ambiguity")
	}
	h.Dispatch()
	outcome(t, h.Complete(report(3).Identity, true), report(3), "received")
	if !reflect.DeepEqual(s.calls, []Report{report(1), report(3)}) {
		t.Fatal("cancelled dispatch or replay")
	}
	h.Stop()
	idle(t, h)
	if got, _ := h.Source(fixtureKey); got != report(3) {
		t.Fatal("retained source changed")
	}
}

func identityMutations() map[string]func(*Identity) {
	return map[string]func(*Identity){
		"node":        func(i *Identity) { i.Key.Node = "n-b" },
		"capability":  func(i *Identity) { i.Key.Capability = "c-b" },
		"composition": func(i *Identity) { i.Key.Composition = "p-b" },
		"generation":  func(i *Identity) { i.Generation++ },
		"token":       func(i *Identity) { i.Token++ },
		"sequence":    func(i *Identity) { i.Sequence++ },
		"evidence":    func(i *Identity) { i.Evidence = "native-b" },
	}
}

func TestReportHarnessIdentityFences(t *testing.T) {
	for name, mutate := range identityMutations() {
		t.Run(name, func(t *testing.T) {
			h, s := fixture(t, 1000)
			r := report(1)
			submit(t, h, r)
			bad := r.Identity
			mutate(&bad)
			if len(h.Cancel(bad)) != 0 || h.Counts().Queue != 1 {
				t.Fatal("queued cancel fence")
			}
			h.Dispatch()
			before := h.Counts()
			if len(h.Cancel(bad)) != 0 {
				t.Fatal("flight cancel fence")
			}
			if len(h.Complete(bad, true)) != 0 || h.Counts() != before {
				t.Fatal("receipt fence")
			}
			outcome(t, h.Complete(r.Identity, true), r, "received")
			if !reflect.DeepEqual(s.calls, []Report{r}) {
				t.Fatal("dispatch identity")
			}
		})
	}
}

func TestReportHarnessAdmissionFences(t *testing.T) {
	for name, mutate := range identityMutations() {
		if name == "evidence" {
			continue
		} // Nonempty evidence is opaque, not an allowlist.
		t.Run(name, func(t *testing.T) {
			h, _ := fixture(t, 1000)
			r := report(1)
			if name == "token" || name == "sequence" {
				submit(t, h, r)
				h.Dispatch()
				h.Complete(r.Identity, true)
				r = report(2)
				if name == "token" {
					r.Token = 1
				} else {
					r.Sequence = 1
				}
			} else {
				mutate(&r.Identity)
			}
			if h.Submit(r) == nil {
				t.Fatal("invalid admission")
			}
			idle(t, h)
		})
	}
	h, _ := fixture(t, 1000)
	r := report(1)
	r.Evidence = ""
	if h.Submit(r) == nil {
		t.Fatal("missing evidence accepted")
	}
}

func TestReportHarnessSourceAxes(t *testing.T) {
	variations := map[string]func(*Report){
		"value":         func(r *Report) { r.State.Value = "other" },
		"availability":  func(r *Report) { r.State.Availability = "absent" },
		"confidence":    func(r *Report) { r.State.Confidence = "Unknown" },
		"qualification": func(r *Report) { r.State.Qualification = "withheld" },
		"freshness":     func(r *Report) { r.State.Freshness = "unknown" },
		"evidence":      func(r *Report) { r.Evidence = "native-b" },
	}
	for name, mutate := range variations {
		t.Run(name, func(t *testing.T) {
			h, s := fixture(t, 1000)
			r := report(1)
			mutate(&r)
			expected := r
			submit(t, h, r)
			h.Dispatch()
			outcome(t, h.Cancel(r.Identity), expected, "delivery-unknown")
			h.Stop()
			h.Complete(r.Identity, true)
			if _, err := h.Start(); err != nil {
				t.Fatal(err)
			}
			if h.Dispatch() || !reflect.DeepEqual(s.calls, []Report{expected}) {
				t.Fatal("replay or projection")
			}
			if got, _ := h.Source(fixtureKey); got != expected {
				t.Fatal("source axis changed")
			}
		})
	}
}

func TestReportHarnessStopAndGeneration(t *testing.T) {
	h, s := fixture(t, 1000)
	if g, err := h.Start(); err == nil || g != 1 {
		t.Fatal("duplicate start")
	}
	submit(t, h, report(1))
	h.Dispatch()
	submit(t, h, report(2))
	if got := h.Stop(); !reflect.DeepEqual(got, []Result{{report(2), "cancelled"}, {report(1), "delivery-unknown"}}) {
		t.Fatalf("stop: %+v", got)
	}
	if len(h.Stop()) != 0 || h.Dispatch() || h.Submit(report(3)) == nil {
		t.Fatal("stop fence")
	}
	if _, err := h.Start(); err == nil {
		t.Fatal("unsafe restart")
	}
	for i := 1; i < 8; i++ {
		if err := h.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if len(h.Complete(report(1).Identity, true)) != 0 {
		t.Fatal("stop late terminal")
	}
	idle(t, h)
	for generation := uint64(2); generation <= 4; generation++ {
		if g, err := h.Start(); err != nil || g != generation {
			t.Fatalf("restart: %d %v", g, err)
		}
		r := report(generation + 1)
		r.Generation = generation
		r.Token = 1
		submit(t, h, r)
		h.Dispatch()
		if len(h.Complete(report(1).Identity, false)) != 0 || h.Counts().Flight != 1 {
			t.Fatal("old callback")
		}
		outcome(t, h.Complete(r.Identity, true), r, "received")
		h.Stop()
		idle(t, h)
		if a, ok := h.Association(fixtureKey); !ok || a != fixtureAssociation {
			t.Fatal("association changed")
		}
	}
	if len(s.calls) != 4 {
		t.Fatal("unexpected dispatch/replay")
	}
}

func TestReportHarnessDrainDeadline(t *testing.T) {
	h, _ := fixture(t, 1000)
	submit(t, h, report(1))
	h.Dispatch()
	h.Stop()
	for step := 1; step <= 8; step++ {
		err := h.Step()
		if (err != nil) != (step == 8) {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	if _, err := h.Start(); err == nil {
		t.Fatal("nonquiescent restart")
	}
	if len(h.Complete(report(1).Identity, true)) != 0 {
		t.Fatal("late success")
	}
	idle(t, h)
	if h.Step() == nil {
		t.Fatal("failure forgotten")
	}
	if _, err := h.Start(); err == nil {
		t.Fatal("failed qualification restarted")
	}
}

func TestReportHarnessSuppliedAssociations(t *testing.T) {
	for _, axis := range []string{"binding", "slot"} {
		t.Run(axis, func(t *testing.T) {
			other := Association{"b-b", "slot-b"}
			if axis == "binding" {
				other.Binding = "b-a"
			} else {
				other.Slot = "slot-a"
			}
			if _, err := New(&fakeSink{}, map[Key]Association{fixtureKey: fixtureAssociation, {"n-a", "c-b", "p-a"}: other}, 1000); err == nil {
				t.Fatal("association conflict")
			}
		})
	}
	snapshot := map[Key]Association{fixtureKey: fixtureAssociation}
	h, err := New(&fakeSink{}, snapshot, 1000)
	if err != nil {
		t.Fatal(err)
	}
	snapshot[fixtureKey] = Association{"b-changed", "slot-changed"}
	if a, _ := h.Association(fixtureKey); a != fixtureAssociation {
		t.Fatal("borrowed association map")
	}
	for _, cap := range []int{0, -1} {
		if _, err := New(&fakeSink{}, snapshot, cap); err == nil {
			t.Fatal("nonpositive cap")
		}
	}
}
