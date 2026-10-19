// Package storeharness models synthetic store barriers, never production admission.
package storeharness

import "fmt"

const (
	RecordLimit   = 2
	CallbackLimit = 2
	DrainSteps    = 8
)

type Key struct{ Node, Capability, Composition string }
type Association struct{ Binding, Slot string }
type Snapshot map[Key]Association
type Identity struct {
	Key               Key
	Generation, Token uint64
}
type Request struct {
	Identity
	Association Association
}
type Outcome uint8

const (
	PrecommitFailure Outcome = iota
	Confirmed
	Ambiguous
)

type Completion struct {
	Request
	Outcome Outcome
}

// Store methods return at logical barriers. They must not reenter the harness.
// Read returns a supplied association snapshot, not commissioned fabric state.
type Store interface {
	Read() (Snapshot, error)
	Write(Request) Outcome
}

// MemoryStore injects per-key faults without touching unrelated opaque records.
// Ambiguous writes commit internally but do not establish confirmed success.
type MemoryStore struct {
	Records       Snapshot
	Unrelated     map[string]string
	Faults        map[Key]Outcome
	ReadFailure   bool
	Reads, Writes int
}

func (s *MemoryStore) Read() (Snapshot, error) {
	s.Reads++
	if s.ReadFailure {
		return nil, fmt.Errorf("injected read failure")
	}
	if s.Records == nil {
		return nil, fmt.Errorf("missing snapshot")
	}
	return clone(s.Records), nil
}
func (s *MemoryStore) Write(r Request) Outcome {
	s.Writes++
	o := Confirmed
	if fault, ok := s.Faults[r.Key]; ok {
		o = fault
	}
	if o == Confirmed || o == Ambiguous {
		if s.Records == nil {
			s.Records = Snapshot{}
		}
		s.Records[r.Key] = r.Association
	}
	return o
}

type Harness struct {
	store                        Store
	byteCap                      int
	accepted, exposed, uncertain Snapshot
	pending                      map[Key]Completion
	running, stopping, failed    bool
	generation, token            uint64
	steps                        int
}

func New(store Store, byteCap int) (*Harness, error) {
	if store == nil || byteCap <= 0 {
		return nil, fmt.Errorf("store and positive record byte cap required")
	}
	return &Harness{store: store, byteCap: byteCap, accepted: Snapshot{}, exposed: Snapshot{}, uncertain: Snapshot{}, pending: map[Key]Completion{}}, nil
}
func clone(s Snapshot) Snapshot {
	d := Snapshot{}
	for k, a := range s {
		d[k] = a
	}
	return d
}
func (h *Harness) validate(s Snapshot) error {
	if s == nil {
		return fmt.Errorf("missing snapshot")
	}
	all := clone(h.accepted)
	for k, a := range h.uncertain {
		all[k] = a
	}
	for k, c := range h.pending {
		all[k] = c.Association
	}
	for k, a := range s {
		if old, ok := all[k]; ok && old != a {
			return fmt.Errorf("association replacement withheld")
		}
		all[k] = a
	}
	if len(all) > RecordLimit {
		return fmt.Errorf("record budget exhausted")
	}
	for k, a := range all {
		if k.Node == "" || k.Capability == "" || k.Composition == "" || a.Binding == "" || a.Slot == "" || len(k.Node)+len(k.Capability)+len(k.Composition)+len(a.Binding)+len(a.Slot) > h.byteCap {
			return fmt.Errorf("invalid or oversized record")
		}
		for other, b := range all {
			if other != k && (a.Binding == b.Binding || a.Slot == b.Slot) {
				return fmt.Errorf("conflicting association")
			}
		}
	}
	return nil
}

// Start reads once. A failed read or invalid snapshot retains accepted state but
// exposes nothing. Unresolved commits require Reconcile, even across restarts.
func (h *Harness) Start() (uint64, error) {
	if h.running || h.stopping || h.failed || len(h.pending) != 0 || h.generation == ^uint64(0) {
		return h.generation, fmt.Errorf("start fenced")
	}
	s, err := h.store.Read()
	if err != nil {
		return h.generation, err
	}
	if err = h.validate(s); err != nil {
		return h.generation, err
	}
	h.generation++
	h.running = true
	h.exposed = Snapshot{}
	for k, a := range s {
		if _, unresolved := h.uncertain[k]; !unresolved {
			h.accepted[k], h.exposed[k] = a, a
		}
	}
	return h.generation, nil
}

// Begin records at most one completion per key; repeats return the same barrier.
// The caller supplies the association. No allocation or implicit retry occurs.
func (h *Harness) Begin(k Key, a Association) (Completion, error) {
	if !h.running {
		return Completion{}, fmt.Errorf("not running")
	}
	if c, ok := h.pending[k]; ok {
		if c.Association == a {
			return c, nil
		}
		return Completion{}, fmt.Errorf("pending association mismatch")
	}
	if _, ok := h.accepted[k]; ok {
		return Completion{}, fmt.Errorf("association already retained")
	}
	if _, ok := h.uncertain[k]; ok {
		return Completion{}, fmt.Errorf("reconciliation required")
	}
	if len(h.pending) >= CallbackLimit || h.token == ^uint64(0) {
		return Completion{}, fmt.Errorf("callback budget exhausted")
	}
	if err := h.validate(Snapshot{k: a}); err != nil {
		return Completion{}, err
	}
	h.token++
	r := Request{Identity{k, h.generation, h.token}, a}
	c := Completion{r, h.store.Write(r)}
	h.pending[k] = c
	return c, nil
}

// Complete accepts only the exact stored barrier, including every identity and
// association field and outcome. Duplicates cannot release another callback.
func (h *Harness) Complete(c Completion) bool {
	p, ok := h.pending[c.Key]
	if !ok || p != c {
		return false
	}
	delete(h.pending, c.Key)
	if c.Outcome == Ambiguous || (!h.running && c.Outcome == Confirmed) {
		h.uncertain[c.Key] = c.Association
	} else if h.running && c.Outcome == Confirmed {
		h.accepted[c.Key], h.exposed[c.Key] = c.Association, c.Association
	}
	if h.stopping && len(h.pending) == 0 && !h.failed {
		h.stopping = false
	}
	return true
}

// Reconcile consumes an explicit detached snapshot after all callbacks drain.
// Absent records remain withheld; changed or corrupt records change no state.
func (h *Harness) Reconcile(s Snapshot) error {
	if !h.running || len(h.pending) != 0 {
		return fmt.Errorf("reconciliation fenced")
	}
	if err := h.validate(s); err != nil {
		return err
	}
	h.exposed = Snapshot{}
	for k, a := range s {
		h.accepted[k], h.exposed[k] = a, a
		delete(h.uncertain, k)
	}
	return nil
}
func (h *Harness) Stop() {
	if !h.running {
		return
	}
	h.running = false
	h.exposed = Snapshot{}
	h.stopping = len(h.pending) != 0
	h.steps = 0
}
func (h *Harness) Step() error {
	if h.failed {
		return fmt.Errorf("drain qualification failed")
	}
	if h.stopping {
		h.steps++
		if h.steps >= DrainSteps {
			h.failed = true
			return fmt.Errorf("logical drain deadline exceeded")
		}
	}
	return nil
}
func (h *Harness) Accepted() Snapshot { return clone(h.accepted) }
func (h *Harness) Exposed() Snapshot  { return clone(h.exposed) }
func (h *Harness) Pending() int       { return len(h.pending) }
