// Package reportharness exercises isolated Z20 mechanics, without SDK integration.
package reportharness

import "fmt"

const (
	QueueLimit  = 2
	FlightLimit = 1
	DrainSteps  = 8
)

type Key struct{ Node, Capability, Composition string }
type Association struct{ Binding, Slot string }

// State is opaque source evidence; no transition upgrades any of its axes.
type State struct{ Value, Availability, Confidence, Qualification, Freshness string }
type Identity struct {
	Key                         Key
	Generation, Token, Sequence uint64
	Evidence                    string
}
type Report struct {
	Identity
	State   State
	Payload string
}
type Result struct {
	Report  Report
	Outcome string
}
type Counters struct{ Queue, Flight, Bytes, RetainedBytes int }
type Sink interface{ Dispatch(Report) }

// Harness is driven serially by its caller. Sink.Dispatch must return without
// calling back; receipt is a separate explicit Complete barrier. Returned results
// belong to the caller: the harness retains no terminal history or retry queue.
type Harness struct {
	sink                      Sink
	cap                       int
	associations              map[Key]Association
	source                    map[Key]Report
	queue                     []Report
	flight                    *Report
	terminal                  bool
	running, stopping, failed bool
	generation, token         uint64
	steps                     int
	peak                      Counters
}

func New(sink Sink, associations map[Key]Association, byteCap int) (*Harness, error) {
	if sink == nil || byteCap <= 0 || len(associations) == 0 {
		return nil, fmt.Errorf("sink, supplied associations and positive byte cap required")
	}
	h := &Harness{sink: sink, cap: byteCap, associations: make(map[Key]Association), source: make(map[Key]Report)}
	slots, bindings := map[string]bool{}, map[string]bool{}
	for k, a := range associations {
		if k.Node == "" || k.Capability == "" || k.Composition == "" || a.Binding == "" || a.Slot == "" || slots[a.Slot] || bindings[a.Binding] {
			return nil, fmt.Errorf("invalid or conflicting supplied association")
		}
		h.associations[k] = a
		slots[a.Slot], bindings[a.Binding] = true, true
	}
	return h, nil
}

func (h *Harness) Start() (uint64, error) {
	if h.running || h.stopping || h.failed || h.flight != nil || h.generation == ^uint64(0) {
		return h.generation, fmt.Errorf("start fenced")
	}
	h.generation++
	h.running = true
	h.token = 0
	return h.generation, nil
}

// Submit never invokes the sink. Token order is strictly increasing per run;
// sequence order is strictly increasing per supplied key, across clean restarts.
func (h *Harness) Submit(r Report) error {
	_, known := h.associations[r.Key]
	if !h.running || !known || r.Generation != h.generation || r.Token <= h.token || r.Sequence == 0 || r.Sequence <= h.source[r.Key].Sequence || r.Evidence == "" {
		return fmt.Errorf("invalid, stale or fenced report")
	}
	if len(h.queue) >= QueueLimit || size(r) > h.cap-h.Counts().Bytes {
		return fmt.Errorf("report queue or byte budget exhausted")
	}
	h.queue = append(h.queue, r)
	h.token = r.Token
	h.source[r.Key] = r
	h.measure()
	return nil
}

// Dispatch releases exactly one queued item when the single flight slot is free.
func (h *Harness) Dispatch() bool {
	if !h.running || h.flight != nil || len(h.queue) == 0 {
		return false
	}
	r := h.queue[0]
	h.queue[0] = Report{}
	h.queue = h.queue[1:]
	h.flight = &r
	h.terminal = false
	h.measure()
	h.sink.Dispatch(r)
	return true
}

func (h *Harness) Cancel(id Identity) []Result {
	for i, r := range h.queue {
		if r.Identity == id {
			copy(h.queue[i:], h.queue[i+1:])
			h.queue[len(h.queue)-1] = Report{}
			h.queue = h.queue[:len(h.queue)-1]
			return []Result{{r, "cancelled"}}
		}
	}
	if h.flight != nil && h.flight.Identity == id && !h.terminal {
		h.terminal = true
		return []Result{{*h.flight, "delivery-unknown"}}
	}
	return nil
}

// Complete requires every identity fence even for cleanup. A receipt after
// cancellation/stop releases ownership but cannot revise its terminal outcome.
func (h *Harness) Complete(id Identity, received bool) []Result {
	if h.flight == nil || h.flight.Identity != id {
		return nil
	}
	r, terminal := *h.flight, h.terminal
	h.flight = nil
	h.terminal = false
	if h.stopping && !h.failed {
		h.stopping = false
	}
	if terminal {
		return nil
	}
	outcome := "failed"
	if received {
		outcome = "received"
	}
	return []Result{{r, outcome}}
}

func (h *Harness) Stop() []Result {
	if !h.running {
		return nil
	}
	h.running = false
	h.steps = 0
	var results []Result
	for _, r := range h.queue {
		results = append(results, Result{r, "cancelled"})
	}
	h.queue = nil
	if h.flight != nil {
		h.stopping = true
		if !h.terminal {
			h.terminal = true
			results = append(results, Result{*h.flight, "delivery-unknown"})
		}
	}
	return results
}

// Step advances only the drain deadline. A receipt must arrive before step D.
func (h *Harness) Step() error {
	if h.failed {
		return fmt.Errorf("drain qualification failed")
	}
	if h.stopping {
		h.steps++
		if h.steps >= DrainSteps {
			h.failed = true
			return fmt.Errorf("drain did not quiesce within logical deadline")
		}
	}
	return nil
}

func (h *Harness) Counts() Counters {
	c := Counters{Queue: len(h.queue)}
	for _, r := range h.queue {
		c.Bytes += size(r)
	}
	if h.flight != nil {
		c.Flight = 1
		c.Bytes += size(*h.flight)
	}
	for _, r := range h.source {
		c.RetainedBytes += size(r)
	}
	return c
}
func (h *Harness) HighWater() Counters                   { return h.peak }
func (h *Harness) Source(k Key) (Report, bool)           { r, ok := h.source[k]; return r, ok }
func (h *Harness) Association(k Key) (Association, bool) { a, ok := h.associations[k]; return a, ok }
func (h *Harness) measure() {
	c := h.Counts()
	if c.Queue > h.peak.Queue {
		h.peak.Queue = c.Queue
	}
	if c.Flight > h.peak.Flight {
		h.peak.Flight = c.Flight
	}
	if c.Bytes > h.peak.Bytes {
		h.peak.Bytes = c.Bytes
	}
	if c.RetainedBytes > h.peak.RetainedBytes {
		h.peak.RetainedBytes = c.RetainedBytes
	}
}

// Bytes counts variable envelope data, including evidence and every source axis.
// Fixed scalar overhead is bounded separately by Q/F and supplied key count.
func size(r Report) int {
	return len(r.Key.Node) + len(r.Key.Capability) + len(r.Key.Composition) + len(r.Evidence) + len(r.Payload) + len(r.State.Value) + len(r.State.Availability) + len(r.State.Confidence) + len(r.State.Qualification) + len(r.State.Freshness)
}
