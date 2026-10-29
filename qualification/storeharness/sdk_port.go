package storeharness

import "fmt"

// StoreScope separates synthetic stores and node lifecycles. No field contains
// fabric material or a device identifier.
type StoreScope struct{ Store, Node string }
type FabricScope struct {
	StoreScope
	Fabric string
}

// EndpointFabricPort is an injected logical-barrier contract, not a go-fabric
// API. Commit changes only the exact endpoint key. RemoveFabric changes only
// the exact fabric namespace; ResetNode changes only the exact node namespace.
// Implementations must preserve all other records, return promptly, and never
// reenter their caller. Ambiguous means committed but unacknowledged, not absent.
type EndpointFabricPort interface {
	ReadEndpoints(StoreScope) (map[Key]Request, error)
	CommitEndpoint(StoreScope, Request) Outcome
	RemoveFabric(FabricScope) Outcome
	ResetNode(StoreScope) Outcome
}

// SDKStore adapts the injected port to Store. It is serial and instance-local;
// cross-process recovery and real SDK glue are deliberately outside this seam.
// Callers drive Harness.Complete before explicit Reconcile. Lifecycle writes
// below are synthetic operations, not authorization for live fabric actions.
type SDKStore struct {
	port            EndpointFabricPort
	scope           StoreScope
	unresolved      map[Key]Request
	expected        map[Key]Request
	withdrawn       map[Key]bool
	lifecycleFenced bool
}

func NewSDKStore(port EndpointFabricPort, scope StoreScope) (*SDKStore, error) {
	if port == nil || scope.Store == "" || scope.Node == "" {
		return nil, fmt.Errorf("port and complete store namespace required")
	}
	return &SDKStore{port: port, scope: scope, unresolved: map[Key]Request{}, expected: map[Key]Request{}, withdrawn: map[Key]bool{}}, nil
}

func (s *SDKStore) read() (map[Key]Request, error) {
	if s.lifecycleFenced {
		return nil, fmt.Errorf("node lifecycle fenced")
	}
	records, err := s.port.ReadEndpoints(s.scope)
	if err != nil {
		return nil, err
	}
	if records == nil || len(records) > RecordLimit {
		return nil, fmt.Errorf("missing or oversized endpoint snapshot")
	}
	copy := make(map[Key]Request, len(records))
	bindings, slots := map[string]bool{}, map[string]bool{}
	for k, r := range records {
		if k != r.Key || k.Node != s.scope.Node || k.Capability == "" || k.Composition == "" || r.Association.Binding == "" || r.Association.Slot == "" || r.Generation == 0 || r.Token == 0 || bindings[r.Association.Binding] || slots[r.Association.Slot] {
			return nil, fmt.Errorf("invalid endpoint snapshot")
		}
		if expected, ok := s.expected[k]; ok && expected != r {
			return nil, fmt.Errorf("stored identity differs")
		}
		bindings[r.Association.Binding], slots[r.Association.Slot] = true, true
		copy[k] = r
	}
	all := make(map[Key]Request, len(s.expected)+len(copy))
	for k, r := range s.expected {
		all[k] = r
	}
	for k, r := range copy {
		all[k] = r
	}
	if len(all) > RecordLimit {
		return nil, fmt.Errorf("retained record budget exhausted")
	}
	bindings, slots = map[string]bool{}, map[string]bool{}
	for _, r := range all {
		if bindings[r.Association.Binding] || slots[r.Association.Slot] {
			return nil, fmt.Errorf("retained association conflict")
		}
		bindings[r.Association.Binding], slots[r.Association.Slot] = true, true
	}
	return copy, nil
}

func (s *SDKStore) Read() (Snapshot, error) {
	records, err := s.read()
	if err != nil {
		return nil, err
	}
	snapshot := Snapshot{}
	for k, r := range records {
		// An explicit exact readback, not restart alone, releases ambiguity.
		if _, unresolved := s.unresolved[k]; !unresolved && !s.withdrawn[k] {
			snapshot[k] = r.Association
		}
		s.expected[k] = r
	}
	return snapshot, nil
}

func (s *SDKStore) Write(r Request) Outcome {
	if s.lifecycleFenced || r.Key.Node != s.scope.Node || r.Key.Capability == "" || r.Key.Composition == "" || r.Association.Binding == "" || r.Association.Slot == "" || r.Generation == 0 || r.Token == 0 {
		return PrecommitFailure
	}
	if _, retained := s.expected[r.Key]; retained {
		return PrecommitFailure
	}
	for _, retained := range s.expected {
		if retained.Association.Binding == r.Association.Binding || retained.Association.Slot == r.Association.Slot {
			return PrecommitFailure
		}
	}
	if len(s.expected) >= RecordLimit {
		return PrecommitFailure
	}
	if _, unresolved := s.unresolved[r.Key]; unresolved {
		return PrecommitFailure
	}
	o := s.port.CommitEndpoint(s.scope, r)
	if o != Confirmed && o != PrecommitFailure {
		o = Ambiguous
	}
	if o != PrecommitFailure {
		s.expected[r.Key] = r
	}
	if o == Ambiguous {
		s.unresolved[r.Key] = r
	}
	return o
}

// Reconcile performs exactly one read after callbacks drain. Absent unresolved
// keys remain fenced; mismatched identities change no harness or adapter state.
func (s *SDKStore) Reconcile(h *Harness) error {
	if h == nil || h.store != s || !h.running || h.Pending() != 0 {
		return fmt.Errorf("reconciliation fenced")
	}
	records, err := s.read()
	if err != nil {
		return err
	}
	snapshot := Snapshot{}
	for k, r := range records {
		if !s.withdrawn[k] {
			snapshot[k] = r.Association
		}
	}
	if err = h.Reconcile(snapshot); err != nil {
		return err
	}
	for k := range records {
		s.expected[k] = records[k]
		delete(s.unresolved, k)
	}
	return nil
}

// Withdraw stops mock exposure only. It retains endpoint identity, reservations
// and all fabrics, including unresolved identities after ambiguous completion.
// Pending writes must complete first, so a late completion cannot undo
// withdrawal. Future selection intent remains a separate owner.
func (s *SDKStore) Withdraw(h *Harness, k Key) error {
	if h == nil || h.store != s || k.Node != s.scope.Node {
		return fmt.Errorf("withdrawal namespace mismatch")
	}
	if _, pending := h.pending[k]; pending {
		return fmt.Errorf("withdrawal awaits completion")
	}
	if _, accepted := h.accepted[k]; !accepted {
		if _, uncertain := h.uncertain[k]; !uncertain {
			return fmt.Errorf("unknown retained endpoint")
		}
	}
	s.withdrawn[k] = true
	delete(h.exposed, k)
	return nil
}

// RemoveFabric never removes endpoint reservations. An unacknowledged lifecycle
// operation fences subsequent operations rather than blindly retrying it.
func (s *SDKStore) RemoveFabric(fabric string) Outcome {
	if s.lifecycleFenced || fabric == "" {
		return PrecommitFailure
	}
	o := s.port.RemoveFabric(FabricScope{s.scope, fabric})
	if o != Confirmed && o != PrecommitFailure {
		s.lifecycleFenced = true
		return Ambiguous
	}
	return o
}

// ResetNode is an explicit synthetic node-lifecycle boundary. A successful or
// ambiguous reset permanently fences this adapter; a new lifecycle needs a new
// store scope. Existing harness observers must be stopped before this operation.
func (s *SDKStore) ResetNode(h *Harness) Outcome {
	if s.lifecycleFenced || h == nil || h.store != s || h.running || h.Pending() != 0 {
		return PrecommitFailure
	}
	o := s.port.ResetNode(s.scope)
	if o == PrecommitFailure {
		return o
	}
	s.lifecycleFenced = true
	if o != Confirmed {
		return Ambiguous
	}
	return Confirmed
}
