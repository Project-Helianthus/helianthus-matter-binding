package reportharness

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

type aa10Port struct {
	calls       []SDKReport
	callbacks   []func(SDKCompletion)
	synchronous bool
}

func (p *aa10Port) Send(r SDKReport, done func(SDKCompletion)) {
	p.calls = append(p.calls, r)
	p.callbacks = append(p.callbacks, done)
	if p.synchronous {
		done(SDKCompletion{r, SDKControllerApplication})
	}
}
func aa10Bounds() SDKRouteBounds {
	return SDKRouteBounds{true, []SDKBuffer{{true, 1, 1024}}, []SDKBuffer{{false, 0, 0}}}
}
func aa10Fixture(t *testing.T, cap int) (*SDKAdapter, *aa10Port) {
	t.Helper()
	p := &aa10Port{}
	a, err := NewSDKAdapter(p, aa10Bounds(), map[Key]Association{fixtureKey: fixtureAssociation}, cap)
	if err != nil {
		t.Fatal(err)
	}
	if g, err := a.Start(); g != 1 || err != nil {
		t.Fatalf("start %d %v", g, err)
	}
	t.Cleanup(func() {
		a.Stop()
		for i, cb := range p.callbacks {
			cb(SDKCompletion{p.calls[i], SDKControllerApplication})
		}
		a.Poll()
		if c := a.Counts(); c.Queue != 0 || c.Flight != 0 || c.Bytes != 0 {
			t.Errorf("cleanup ownership: %+v", c)
		}
	})
	return a, p
}
func aa10Submit(t *testing.T, a *SDKAdapter, r Report) {
	t.Helper()
	if err := a.Submit(r); err != nil {
		t.Fatal(err)
	}
}
func aa10Dispatch(t *testing.T, a *SDKAdapter) {
	t.Helper()
	if !a.DispatchNext() {
		t.Fatal("dispatch barrier")
	}
}
func aa10Receipt(p *aa10Port, i int, receipt SDKReceipt) {
	p.callbacks[i](SDKCompletion{p.calls[i], receipt})
}

func TestAA10FIFOAndReceipt(t *testing.T) {
	a, p := aa10Fixture(t, 1000)
	aa10Submit(t, a, report(1))
	aa10Dispatch(t, a)
	aa10Submit(t, a, report(2))
	aa10Submit(t, a, report(3))
	before := a.Counts()
	if before != (Counters{2, 1, 183, 61}) {
		t.Fatalf("counts %+v", before)
	}
	if a.Submit(report(4)) == nil || a.Counts() != before {
		t.Fatal("full Q=2/F=1 admitted sequence 4")
	}
	if got, _ := a.Source(fixtureKey); got != report(3) {
		t.Fatal("rejected report changed source")
	}
	for _, receipt := range []SDKReceipt{SDKDirty, SDKEnqueued, SDKDatagram, SDKMRPAcknowledged, "unknown"} {
		aa10Receipt(p, 0, receipt)
		if len(a.Poll()) != 0 || a.Counts() != before || a.DispatchNext() {
			t.Fatalf("premature receipt %s", receipt)
		}
	}
	for i := 0; i < 3; i++ {
		if i > 0 {
			aa10Dispatch(t, a)
			// An earlier callback cannot release the current flight.
			aa10Receipt(p, i-1, SDKControllerApplication)
			if len(a.Poll()) != 0 || a.Counts().Flight != 1 {
				t.Fatal("stale callback released current budget")
			}
		}
		if len(p.calls) != i+1 || p.calls[i] != (SDKReport{report(uint64(i + 1)), fixtureAssociation}) {
			t.Fatalf("lost/coalesced observation: %+v", p.calls)
		}
		aa10Receipt(p, i, SDKControllerApplication)
		if a.Counts().Flight != 1 {
			t.Fatal("callback reentered harness")
		}
		outcome(t, a.Poll(), report(uint64(i+1)), "received")
		if len(a.Poll()) != 0 {
			t.Fatal("duplicate terminal")
		}
	}
	if a.HighWater() != (Counters{2, 1, 183, 61}) {
		t.Fatal("high water")
	}
}

func TestAA10InnerBounds(t *testing.T) {
	cases := map[string]func(*SDKRouteBounds){
		"inventory":         func(b *SDKRouteBounds) { b.InventoryComplete = false },
		"missing callbacks": func(b *SDKRouteBounds) { b.Callbacks = nil },
		"missing events":    func(b *SDKRouteBounds) { b.Events = nil },
	}
	for _, kind := range []string{"callbacks", "events"} {
		for _, axis := range []string{"items", "bytes", "negative items", "negative bytes", "disabled items", "disabled bytes"} {
			kind, axis := kind, axis
			cases[kind+"/"+axis] = func(b *SDKRouteBounds) {
				inner := SDKBuffer{true, 1, 100}
				switch axis {
				case "items":
					inner.Items = 0
				case "bytes":
					inner.Bytes = 0
				case "negative items":
					inner.Items = -1
				case "negative bytes":
					inner.Bytes = -1
				case "disabled items":
					inner = SDKBuffer{false, 1, 0}
				case "disabled bytes":
					inner = SDKBuffer{false, 0, 1}
				}
				// A bounded sibling cannot hide an uncapped enabled inner buffer.
				if kind == "callbacks" {
					b.Callbacks = append(b.Callbacks, inner)
				} else {
					b.Events = append(b.Events, inner)
				}
			}
		}
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := &aa10Port{}
			b := aa10Bounds()
			mutate(&b)
			if _, err := NewSDKAdapter(p, b, map[Key]Association{fixtureKey: fixtureAssociation}, 1000); err == nil {
				t.Fatal("unbounded route admitted with empty outer queue")
			}
			if len(p.calls) != 0 {
				t.Fatal("rejected route invoked port")
			}
			b = aa10Bounds()
			b.Events = []SDKBuffer{{true, 2, 256}}
			if _, err := NewSDKAdapter(p, b, map[Key]Association{fixtureKey: fixtureAssociation}, 1000); err != nil {
				t.Fatal("valid positive control", err)
			}
		})
	}
	if _, err := NewSDKAdapter(nil, aa10Bounds(), map[Key]Association{fixtureKey: fixtureAssociation}, 1000); err == nil {
		t.Fatal("nil port admitted")
	}
}

func aa10Fields() map[string]func(*SDKReport) {
	return map[string]func(*SDKReport){
		"node":           func(r *SDKReport) { r.Report.Key.Node = "n-other" },
		"capability":     func(r *SDKReport) { r.Report.Key.Capability = "c-other" },
		"composition":    func(r *SDKReport) { r.Report.Key.Composition = "p-other" },
		"binding":        func(r *SDKReport) { r.Association.Binding = "b-other" },
		"slot":           func(r *SDKReport) { r.Association.Slot = "slot-other" },
		"generation":     func(r *SDKReport) { r.Report.Generation++ },
		"token":          func(r *SDKReport) { r.Report.Token++ },
		"sequence":       func(r *SDKReport) { r.Report.Sequence++ },
		"evidence":       func(r *SDKReport) { r.Report.Evidence = "native-other" },
		"value":          func(r *SDKReport) { r.Report.State.Value = "other" },
		"availability":   func(r *SDKReport) { r.Report.State.Availability = "absent" },
		"confidence":     func(r *SDKReport) { r.Report.State.Confidence = "Unknown" },
		"qualification":  func(r *SDKReport) { r.Report.State.Qualification = "withheld" },
		"freshness":      func(r *SDKReport) { r.Report.State.Freshness = "unknown" },
		"payload length": func(r *SDKReport) { r.Report.Payload += "longer" },
	}
}
func TestAA10CompletionFences(t *testing.T) {
	for name, mutate := range aa10Fields() {
		t.Run(name, func(t *testing.T) {
			a, p := aa10Fixture(t, 1000)
			aa10Submit(t, a, report(1))
			aa10Dispatch(t, a)
			expected := SDKReport{report(1), fixtureAssociation}
			if p.calls[0] != expected {
				t.Fatal("request changed")
			}
			bad := expected
			mutate(&bad)
			before := a.Counts()
			p.callbacks[0](SDKCompletion{bad, SDKControllerApplication})
			if len(a.Poll()) != 0 || a.Counts() != before {
				t.Fatal("mismatched completion released budget")
			}
			aa10Receipt(p, 0, SDKControllerApplication)
			outcome(t, a.Poll(), expected.Report, "received")
			aa10Submit(t, a, report(2))
			aa10Dispatch(t, a)
			// A stale closure must reject even a forged current request.
			p.callbacks[0](SDKCompletion{p.calls[1], SDKControllerApplication})
			if len(a.Poll()) != 0 || a.Counts().Flight != 1 {
				t.Fatal("old closure released newer flight")
			}
			// Deliver an exact old duplicate immediately before the current receipt,
			// without a Poll between them: stale work must not occupy the mailbox.
			aa10Receipt(p, 0, SDKControllerApplication)
			aa10Receipt(p, 1, SDKControllerApplication)
			outcome(t, a.Poll(), report(2), "received")
		})
	}
}

func TestAA10PreservesEveryField(t *testing.T) {
	variations := aa10Fields()
	variations["unchanged"] = func(*SDKReport) {}
	for name, mutate := range variations {
		t.Run(name, func(t *testing.T) {
			expected := SDKReport{report(1), fixtureAssociation}
			mutate(&expected)
			// Each variant is valid in its own supplied association and generation.
			p := &aa10Port{}
			a, err := NewSDKAdapter(p, aa10Bounds(), map[Key]Association{expected.Report.Key: expected.Association}, 1000)
			if err != nil {
				t.Fatal(err)
			}
			for g := uint64(0); g < expected.Report.Generation; g++ {
				if _, err := a.Start(); err != nil {
					t.Fatal(err)
				}
				if g+1 < expected.Report.Generation {
					a.Stop()
				}
			}
			aa10Submit(t, a, expected.Report)
			aa10Dispatch(t, a)
			if !reflect.DeepEqual(p.calls, []SDKReport{expected}) {
				t.Fatalf("projection %+v want %+v", p.calls, expected)
			}
			outcome(t, a.Cancel(expected.Report.Identity), expected.Report, "delivery-unknown")
			aa10Receipt(p, 0, SDKControllerApplication)
			if len(a.Poll()) != 0 {
				t.Fatal("receipt revised cancellation")
			}
			if got, ok := a.Source(expected.Report.Key); !ok || got != expected.Report {
				t.Fatal("partial last-known-good changed")
			}
			a.Stop()
			if _, err := a.Start(); err != nil {
				t.Fatal(err)
			}
			if a.DispatchNext() || len(p.calls) != 1 {
				t.Fatal("replay")
			}
			a.Stop()
			if c := a.Counts(); c.Queue != 0 || c.Flight != 0 || c.Bytes != 0 {
				t.Fatal("retained work")
			}
		})
	}
}

func TestAA10Bytes(t *testing.T) {
	// Independent envelope budget: key 9, evidence 8, payload 3, state 41.
	a, p := aa10Fixture(t, 122)
	aa10Submit(t, a, report(1))
	aa10Dispatch(t, a)
	aa10Submit(t, a, report(2))
	if a.Counts().Bytes != 122 {
		t.Fatal("byte accounting")
	}
	if a.Submit(report(3)) == nil {
		t.Fatal("aggregate byte budget ignored")
	}
	aa10Receipt(p, 0, SDKControllerApplication)
	outcome(t, a.Poll(), report(1), "received")
	aa10Dispatch(t, a)
	aa10Receipt(p, 1, SDKControllerApplication)
	outcome(t, a.Poll(), report(2), "received")
	oversized := report(3)
	oversized.Payload = strings.Repeat("x", 123)
	if a.Submit(oversized) == nil {
		t.Fatal("payload length ignored")
	}
	if got, _ := a.Source(fixtureKey); got != report(2) {
		t.Fatal("rejected oversized source replaced")
	}
	aa10Submit(t, a, report(3))
	aa10Dispatch(t, a)
	aa10Receipt(p, 2, SDKFailed)
	outcome(t, a.Poll(), report(3), "failed")
}

func TestAA10SynchronousAndConcurrentCallbacks(t *testing.T) {
	a, p := aa10Fixture(t, 1000)
	p.synchronous = true
	for n := uint64(1); n <= 32; n++ {
		aa10Submit(t, a, report(n))
		aa10Dispatch(t, a)
		if len(p.calls) != int(n) || a.Counts().Flight != 1 {
			t.Fatal("recursive dispatch or synchronous completion")
		}
		var callbacks sync.WaitGroup
		for i := 0; i < 16; i++ {
			callbacks.Add(1)
			go func(i int) { defer callbacks.Done(); aa10Receipt(p, i, SDKControllerApplication) }(int(n - 1))
		}
		callbacks.Wait()
		outcome(t, a.Poll(), report(n), "received")
		if len(a.Poll()) != 0 {
			t.Fatal("duplicate completion")
		}
	}
}

func TestAA10DrainDeadline(t *testing.T) {
	for _, receiptStep := range []int{7, 8, 9} {
		t.Run(string(rune('0'+receiptStep)), func(t *testing.T) {
			a, p := aa10Fixture(t, 1000)
			aa10Submit(t, a, report(1))
			aa10Dispatch(t, a)
			aa10Submit(t, a, report(2))
			if got := a.Stop(); !reflect.DeepEqual(got, []Result{{report(2), "cancelled"}, {report(1), "delivery-unknown"}}) {
				t.Fatalf("stop %+v", got)
			}
			if _, err := a.Start(); err == nil {
				t.Fatal("restart with outstanding callback")
			}
			for step := 1; step <= receiptStep; step++ {
				err := a.Step()
				if (err != nil) != (step >= 8) {
					t.Fatalf("deadline step %d: %v", step, err)
				}
			}
			aa10Receipt(p, 0, SDKControllerApplication)
			if len(a.Poll()) != 0 {
				t.Fatal("late receipt rewrote unknown")
			}
			_, err := a.Start()
			if (err != nil) != (receiptStep >= 8) {
				t.Fatal("late callback treated as quiescent qualification")
			}
			if receiptStep >= 8 && a.Step() == nil {
				t.Fatal("failure not latched")
			}
			if a.DispatchNext() || len(p.calls) != 1 {
				t.Fatal("replay after stop")
			}
		})
	}
}

func TestAA10CancellationAndRestart(t *testing.T) {
	a, p := aa10Fixture(t, 1000)
	aa10Submit(t, a, report(1))
	aa10Dispatch(t, a)
	aa10Submit(t, a, report(2))
	outcome(t, a.Cancel(report(2).Identity), report(2), "cancelled")
	if len(a.Cancel(report(2).Identity)) != 0 {
		t.Fatal("duplicate queued cancellation")
	}
	outcome(t, a.Cancel(report(1).Identity), report(1), "delivery-unknown")
	if len(a.Cancel(report(1).Identity)) != 0 || a.Counts().Flight != 1 {
		t.Fatal("cancel released flight")
	}
	aa10Submit(t, a, report(3))
	if a.DispatchNext() {
		t.Fatal("cancel allowed next dispatch before cleanup")
	}
	aa10Receipt(p, 0, SDKFailed)
	if len(a.Poll()) != 0 {
		t.Fatal("failure revised unknown")
	}
	aa10Dispatch(t, a)
	if p.calls[1].Report != report(3) {
		t.Fatal("cancelled sample dispatched")
	}
	aa10Receipt(p, 1, SDKControllerApplication)
	outcome(t, a.Poll(), report(3), "received")
	if len(a.Stop()) != 0 {
		t.Fatal("clean stop returned terminal")
	}
	if g, err := a.Start(); err != nil || g != 2 {
		t.Fatalf("clean restart %d %v", g, err)
	}
	if a.DispatchNext() || len(p.calls) != 2 {
		t.Fatal("restart replayed retained evidence")
	}
	r := report(4)
	r.Generation = 2
	r.Token = 1
	aa10Submit(t, a, r)
	aa10Dispatch(t, a)
	aa10Receipt(p, 0, SDKControllerApplication)
	p.callbacks[1](SDKCompletion{p.calls[2], SDKControllerApplication})
	if len(a.Poll()) != 0 || a.Counts().Flight != 1 {
		t.Fatal("old generation callback released new flight")
	}
	aa10Receipt(p, 2, SDKControllerApplication)
	outcome(t, a.Poll(), r, "received")
	if got, _ := a.Source(fixtureKey); got != r {
		t.Fatal("source state changed")
	}
}
