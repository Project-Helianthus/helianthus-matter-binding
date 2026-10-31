// Package adoption implements the isolated, offline reference bridge host. It
// imports no SDK and is reachable only through a synthetic fixture entry point;
// it makes no production admission decision and qualifies no target.
package adoption

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	// DataModelVersion is the data model of the adopted SDK direction.
	DataModelVersion = "1.6.1"
	// CurrentFactID is the only semantic fact this host accepts as a value.
	CurrentFactID = "evse.ac.current"
	// CurrentUnit is the projected unit. Unit conversion belongs to the
	// projection port; the host never converts.
	CurrentUnit = "mA"

	syntheticSource  = "synthetic-fixture"
	syntheticRefHead = "synthetic:"

	maxTokenBytes       = 128
	maxMeasurementTypes = 8
	maxCurrentMilliamps = int64(1) << 62
)

// Evidence grades one claim about a definition identity. Proven here means
// reported source presence only, not qualification, conformance or closure.
type Evidence string

const (
	EvidenceProven     Evidence = "Proven"
	EvidenceHypothesis Evidence = "Hypothesis"
	EvidenceUnknown    Evidence = "Unknown"
)

// Definition names a device type or cluster server. An Unknown ID or Revision
// stays zero; it is resolved by inspecting the real SDK, never by a default.
type Definition struct {
	Name             string
	ID               uint32
	IDEvidence       Evidence
	Revision         uint16
	RevisionEvidence Evidence
}

func known(name string, id uint32, revision uint16) Definition {
	return Definition{name, id, EvidenceProven, revision, EvidenceProven}
}

func rootNodeType() Definition {
	return Definition{Name: "Root Node", IDEvidence: EvidenceUnknown, RevisionEvidence: EvidenceUnknown}
}
func aggregatorType() Definition { return known("Aggregator", 0x000E, 2) }
func bridgedNodeType() Definition {
	return Definition{Name: "Bridged Node", IDEvidence: EvidenceUnknown, Revision: 3, RevisionEvidence: EvidenceProven}
}
func electricalSensorType() Definition { return known("Electrical Sensor", 0x0510, 1) }
func descriptorServer() Definition {
	return Definition{Name: "Descriptor", ID: 0x001D, IDEvidence: EvidenceProven, RevisionEvidence: EvidenceUnknown}
}
func bridgedBasicServer() Definition { return known("Bridged Device Basic Information", 0x0039, 6) }
func powerTopologyServer() Definition {
	return known("Power Topology", 0x009C, 1)
}
func powerMeasurementServer() Definition {
	return known("Electrical Power Measurement", 0x0090, 3)
}

// Errors are separate so that callers can tell each withheld case apart. Fixture
// context failures, down to ErrTopologyFact, happen before any port call.
var (
	ErrPortMissing         = errors.New("adoption: required port is missing")
	ErrContextNotSynthetic = errors.New("adoption: context is not an explicitly synthetic fixture")
	ErrManifest            = errors.New("adoption: fixture manifest identity is invalid")
	ErrTarget              = errors.New("adoption: declared target differs from the reference model")
	ErrIdentity            = errors.New("adoption: sensor identity is invalid")
	ErrProvenance          = errors.New("adoption: mandatory fact is not explicitly synthetic")
	ErrPowerMode           = errors.New("adoption: PowerMode is absent, unknown or unsupported")
	ErrMeasurementCount    = errors.New("adoption: NumberOfMeasurementTypes is absent or not positive")
	ErrAccuracy            = errors.New("adoption: Accuracy is absent, malformed or mismatched")
	ErrActivePower         = errors.New("adoption: nullable ActivePower is undeclared or contradictory")
	ErrFeatures            = errors.New("adoption: Electrical Power Measurement features are unsupported")
	ErrTopologyFact        = errors.New("adoption: Power Topology metadata is absent or unsupported")

	ErrAlreadyStarted = errors.New("adoption: host is already running")
	ErrNotRunning     = errors.New("adoption: host is not running")
	ErrNoNode         = errors.New("adoption: SDK returned neither node nor error")
	ErrQuiesce        = errors.New("adoption: SDK node did not stop; restart is prohibited")

	ErrSecondSensor       = errors.New("adoption: only the one reference sensor is allowed")
	ErrTopologyOp         = errors.New("adoption: unsupported topology operation")
	ErrTopologyUnresolved = errors.New("adoption: a failed topology change is unresolved until stop")
	ErrSensorDetached     = errors.New("adoption: sensor endpoint is detached")

	ErrProjectionFact     = errors.New("adoption: projection returned a different fact")
	ErrProjectionIdentity = errors.New("adoption: projection returned a different sensor identity")
	ErrProjectionState    = errors.New("adoption: projection availability, freshness or confidence is invalid")
	ErrProjectionUnit     = errors.New("adoption: projection unit is not milliamperes")
	ErrProjectionValue    = errors.New("adoption: projection value is malformed or contradicts availability")
	ErrProjectionRange    = errors.New("adoption: projection value is out of range")
	ErrProjectionEvidence = errors.New("adoption: projection value has no native evidence reference")
	ErrProjectionRead     = errors.New("adoption: projection read failed")
)

// ContextOrigin labels who built a context. Only OriginSyntheticFixture starts.
type ContextOrigin string

const (
	OriginSyntheticFixture ContextOrigin = "synthetic-fixture"
	OriginProduction       ContextOrigin = "production"
)

// Provenance marks one mandatory fact as fixture input. These facts are never
// native evidence and never reusable as such.
type Provenance struct {
	Source    string
	Reference string
}

// SyntheticProvenance returns the only accepted source with its reference.
func SyntheticProvenance(name string) Provenance {
	return Provenance{Source: syntheticSource, Reference: syntheticRefHead + name}
}

// Identity is the full supplied association of the one sensor. Slot is a
// positive bookkeeping number, not a Matter endpoint number.
type Identity struct {
	Node, Owner, Profile, Capability, Composition, Binding string
	Slot                                                   int64
}

// TargetDeclaration is the model the fixture manifest claims to target. It must
// equal the reference model exactly; older revisions are rejected, not adapted.
type TargetDeclaration struct {
	DataModel                string
	AggregatorRevision       uint16
	BridgedNodeRevision      uint16
	BridgedBasicRevision     uint16
	ElectricalSensorRevision uint16
	PowerTopologyRevision    uint16
	PowerMeasurementRevision uint16
}

// PowerModeFact, CountFact, AccuracyFact, ActivePowerFact, FeatureFact and
// TopologyFact are the separately supplied mandatory metadata. None of them is
// derived from the projected current or from any other fact.
type PowerModeFact struct {
	Value      string
	Provenance Provenance
}

type CountFact struct {
	Value      int
	Provenance Provenance
}

type AccuracyFact struct {
	MeasurementType          string
	Measured                 bool
	MinMeasured, MaxMeasured int64
	FixedMax                 int64
	Provenance               Provenance
}

// ActivePowerFact separates undeclared (Declared false) from a declared null
// and from a declared known zero.
type ActivePowerFact struct {
	Declared   bool
	Null       bool
	Milliwatts int64
	Provenance Provenance
}

type FeatureFact struct {
	Values     []string
	Provenance Provenance
}

type TopologyFact struct {
	Mode       string
	Provenance Provenance
}

// SyntheticMetadata groups the mandatory facts handed to the SDK port.
type SyntheticMetadata struct {
	PowerMode        PowerModeFact
	MeasurementTypes CountFact
	Accuracy         []AccuracyFact
	ActivePower      ActivePowerFact
	Features         FeatureFact
	Topology         TopologyFact
}

func (m SyntheticMetadata) clone() SyntheticMetadata {
	m.Accuracy = append([]AccuracyFact(nil), m.Accuracy...)
	m.Features.Values = append([]string(nil), m.Features.Values...)
	return m
}

// FixtureContext is the fixture entry point's only input. A zero context is not
// synthetic, so nothing is allocated until a caller supplies this explicitly.
type FixtureContext struct {
	Origin   ContextOrigin
	Manifest string
	Target   TargetDeclaration
	Sensor   Identity
	Metadata SyntheticMetadata
}

// EndpointRole identifies a planned endpoint without assigning an endpoint number.
type EndpointRole string

const (
	RoleRoot       EndpointRole = "root"
	RoleAggregator EndpointRole = "aggregator"
	RoleSensor     EndpointRole = "bridged-electrical-sensor"
)

// EndpointPlan is a frozen descriptor view. Parts holds full-family keys, and
// SDKOwnedServersUnknown marks the root, whose SDK-owned servers are not listed.
type EndpointPlan struct {
	Key                    string
	Role                   EndpointRole
	DeviceTypes            []Definition
	Parts                  []string
	Servers                []Definition
	SDKOwnedServersUnknown bool
}

func (e EndpointPlan) clone() EndpointPlan {
	e.DeviceTypes = append([]Definition(nil), e.DeviceTypes...)
	e.Parts = append([]string(nil), e.Parts...)
	e.Servers = append([]Definition(nil), e.Servers...)
	return e
}

// NodePlan is the detached construction request. Its ActiveCurrent starts null;
// values arrive only through CurrentUpdate after the node has started.
type NodePlan struct {
	DataModel  string
	Generation uint64
	Manifest   string
	SensorKey  string
	Endpoints  []EndpointPlan
	Metadata   SyntheticMetadata
}

func (p NodePlan) clone() NodePlan {
	endpoints := make([]EndpointPlan, len(p.Endpoints))
	for i, endpoint := range p.Endpoints {
		endpoints[i] = endpoint.clone()
	}
	p.Endpoints = endpoints
	p.Metadata = p.Metadata.clone()
	return p
}

// TopologyOp is one serialized change to the sensor endpoint.
type TopologyOp string

const (
	TopologyAttach TopologyOp = "attach"
	TopologyDetach TopologyOp = "detach"
)

// TopologyRequest must name the full identity of the started sensor.
type TopologyRequest struct {
	Op     TopologyOp
	Sensor Identity
}

// TopologyChange is the detached SDK request. Detach keeps the association
// reserved for this sensor; there is no operation that releases it.
type TopologyChange struct {
	Generation uint64
	Sequence   uint64
	Op         TopologyOp
	SensorKey  string
	Endpoint   EndpointPlan
}

// CurrentUpdate is ActiveCurrent only: no other attribute can travel through
// it. Null with a WithheldReason differs from a known zero value.
type CurrentUpdate struct {
	Generation     uint64
	Sequence       uint64
	SensorKey      string
	Null           bool
	Milliamperes   int64
	WithheldReason string
	NativeEvidence string
}

// SDKNode is a returned node. Calls must return promptly and must not call back
// into the host. Stop is called exactly once for every node ever returned.
type SDKNode interface {
	Start() error
	Stop() error
	ApplyTopology(TopologyChange) error
	PublishCurrent(CurrentUpdate) error
}

// SDKPort constructs one node. It is a local port: no upstream API is implied.
// A node returned together with an error is still owned and is stopped.
type SDKPort interface {
	New(NodePlan) (SDKNode, error)
}

// Availability, Freshness and the confidence and qualification fields of
// CurrentObservation are independent axes; none implies another.
type Availability string

const (
	AvailabilityAvailable   Availability = "available"
	AvailabilityUnavailable Availability = "unavailable"
	AvailabilityUnknown     Availability = "unknown"
)

type Freshness string

const (
	FreshnessFresh   Freshness = "fresh"
	FreshnessStale   Freshness = "stale"
	FreshnessUnknown Freshness = "unknown"
)

// CurrentRequest asks the accepted projection for one sensor's current.
type CurrentRequest struct {
	Fact   string
	Sensor Identity
}

// CurrentObservation is the projection output. Value is a canonical decimal
// integer of CurrentUnit, empty when there is no value.
type CurrentObservation struct {
	Fact           string
	Sensor         Identity
	Availability   Availability
	Freshness      Freshness
	Qualified      bool
	Confidence     Evidence
	Unit           string
	Value          string
	NativeEvidence string
}

// ProjectionPort supplies the accepted projection result. The host keeps no
// second truth store and does not interpret native protocol values.
type ProjectionPort interface {
	ReadCurrent(CurrentRequest) (CurrentObservation, error)
}

// LifecycleState names one observable transition of a generation.
type LifecycleState string

const (
	LifecycleStarting      LifecycleState = "starting"
	LifecycleRunning       LifecycleState = "running"
	LifecycleStartFailed   LifecycleState = "start-failed"
	LifecycleStopping      LifecycleState = "stopping"
	LifecycleStopped       LifecycleState = "stopped"
	LifecycleQuiesceFailed LifecycleState = "quiesce-failed"
)

// LifecycleEvent reports a transition of exactly one generation.
type LifecycleEvent struct {
	Generation uint64
	State      LifecycleState
}

// LifecyclePort observes transitions. It must return promptly and must not call
// back into the host; it cannot admit, retry or restart anything.
type LifecyclePort interface {
	Transition(LifecycleEvent)
}

// Ports are the three injected boundaries. No production launcher is reachable
// from this package and no port is called before a valid fixture context.
type Ports struct {
	SDK        SDKPort
	Projection ProjectionPort
	Lifecycle  LifecyclePort
}

type hostState int

const (
	stateIdle hostState = iota
	stateRunning
	stateFailed
)

// Snapshot is a detached view. Published is what the SDK last accepted;
// LastKnownGood is retained evidence and is never republished as fresh.
type Snapshot struct {
	Running       bool
	Failed        bool
	Generation    uint64
	Attached      bool
	Published     CurrentUpdate
	LastKnownGood *CurrentUpdate
}

// ReferenceHost owns at most one node with exactly one sensor. mu is held across
// port calls: that is what serializes topology changes against each other, and
// against start, stop and publication, since concurrent reassemblies are unordered.
type ReferenceHost struct {
	mu         sync.Mutex
	ports      Ports
	state      hostState
	failure    error
	generation uint64
	sequence   uint64
	pinned     *Identity
	sensorKey  string
	plan       *NodePlan
	node       SDKNode
	attached   bool
	unresolved bool
	published  CurrentUpdate
	lastGood   *CurrentUpdate
}

// NewReferenceHost stores the ports and calls none of them.
func NewReferenceHost(ports Ports) (*ReferenceHost, error) {
	if ports.SDK == nil || ports.Projection == nil || ports.Lifecycle == nil {
		return nil, ErrPortMissing
	}
	return &ReferenceHost{ports: ports}, nil
}

// StableKey encodes every identity axis, length-prefixed so that no two
// identities can share a key. A display label is not sufficient.
func StableKey(id Identity) string {
	var b strings.Builder
	b.WriteString("helianthus.reference/")
	for _, axis := range []string{id.Node, id.Owner, id.Profile, id.Capability, id.Composition, id.Binding} {
		b.WriteString(strconv.Itoa(len(axis)))
		b.WriteByte(':')
		b.WriteString(axis)
	}
	b.WriteString(strconv.FormatInt(id.Slot, 10))
	return b.String()
}

func token(s string) bool {
	return s != "" && len(s) <= maxTokenBytes && utf8.ValidString(s) && strings.TrimSpace(s) == s &&
		!strings.ContainsFunc(s, unicode.IsControl)
}

func validProvenance(p Provenance) bool {
	return p.Source == syntheticSource && token(p.Reference) && strings.HasPrefix(p.Reference, syntheticRefHead) &&
		len(p.Reference) > len(syntheticRefHead)
}

func referenceTarget() TargetDeclaration {
	return TargetDeclaration{
		DataModel:                DataModelVersion,
		AggregatorRevision:       aggregatorType().Revision,
		BridgedNodeRevision:      bridgedNodeType().Revision,
		BridgedBasicRevision:     bridgedBasicServer().Revision,
		ElectricalSensorRevision: electricalSensorType().Revision,
		PowerTopologyRevision:    powerTopologyServer().Revision,
		PowerMeasurementRevision: powerMeasurementServer().Revision,
	}
}

func validIdentity(id Identity) bool {
	for _, axis := range []string{id.Node, id.Owner, id.Profile, id.Capability, id.Composition, id.Binding} {
		if !token(axis) {
			return false
		}
	}
	return id.Slot > 0
}

// validate is the whole admission of the fixture boundary and runs before any
// port call. A missing fact is rejected here; nothing fills it in.
func (c FixtureContext) validate() error {
	if c.Origin != OriginSyntheticFixture {
		return fmt.Errorf("%w: origin %q", ErrContextNotSynthetic, c.Origin)
	}
	if !token(c.Manifest) || !strings.HasPrefix(c.Manifest, syntheticRefHead) || len(c.Manifest) == len(syntheticRefHead) {
		return ErrManifest
	}
	if c.Target != referenceTarget() {
		return ErrTarget
	}
	if !validIdentity(c.Sensor) {
		return ErrIdentity
	}
	return c.Metadata.validate()
}

// Each fact's own value check runs first, so that a missing fact reports itself.
func (m SyntheticMetadata) validate() error {
	if m.PowerMode.Value != "AC" {
		return fmt.Errorf("%w: %q", ErrPowerMode, m.PowerMode.Value)
	}
	if !validProvenance(m.PowerMode.Provenance) {
		return fmt.Errorf("%w: PowerMode", ErrProvenance)
	}
	if m.MeasurementTypes.Value < 1 || m.MeasurementTypes.Value > maxMeasurementTypes {
		return fmt.Errorf("%w: %d", ErrMeasurementCount, m.MeasurementTypes.Value)
	}
	if !validProvenance(m.MeasurementTypes.Provenance) {
		return fmt.Errorf("%w: NumberOfMeasurementTypes", ErrProvenance)
	}
	if len(m.Accuracy) != m.MeasurementTypes.Value {
		return fmt.Errorf("%w: %d entries", ErrAccuracy, len(m.Accuracy))
	}
	seen := make(map[string]bool, len(m.Accuracy))
	for _, entry := range m.Accuracy {
		if !token(entry.MeasurementType) || seen[entry.MeasurementType] || entry.MinMeasured > entry.MaxMeasured || entry.FixedMax < 0 {
			return fmt.Errorf("%w: entry %q", ErrAccuracy, entry.MeasurementType)
		}
		if !validProvenance(entry.Provenance) {
			return fmt.Errorf("%w: Accuracy", ErrProvenance)
		}
		seen[entry.MeasurementType] = true
	}
	if !m.ActivePower.Declared || (m.ActivePower.Null && m.ActivePower.Milliwatts != 0) {
		return ErrActivePower
	}
	if !validProvenance(m.ActivePower.Provenance) {
		return fmt.Errorf("%w: ActivePower", ErrProvenance)
	}
	if len(m.Features.Values) != 1 || m.Features.Values[0] != "AC" {
		return fmt.Errorf("%w: %q", ErrFeatures, m.Features.Values)
	}
	if !validProvenance(m.Features.Provenance) {
		return fmt.Errorf("%w: features", ErrProvenance)
	}
	if m.Topology.Mode != "NODE" {
		return fmt.Errorf("%w: %q", ErrTopologyFact, m.Topology.Mode)
	}
	if !validProvenance(m.Topology.Provenance) {
		return fmt.Errorf("%w: topology", ErrProvenance)
	}
	return nil
}

// buildPlan takes no observation: mandatory facts cannot depend on current.
// The single-endpoint bridged shape and the full-family Parts are a Hypothesis
// that the real SDK binding must inspect.
func buildPlan(c FixtureContext, generation uint64) NodePlan {
	sensorKey := StableKey(c.Sensor)
	return NodePlan{
		DataModel:  DataModelVersion,
		Generation: generation,
		Manifest:   c.Manifest,
		SensorKey:  sensorKey,
		Metadata:   c.Metadata.clone(),
		Endpoints: []EndpointPlan{
			{
				Key:                    string(RoleRoot),
				Role:                   RoleRoot,
				DeviceTypes:            []Definition{rootNodeType()},
				Parts:                  []string{string(RoleAggregator), sensorKey},
				Servers:                []Definition{descriptorServer()},
				SDKOwnedServersUnknown: true,
			},
			{
				Key:         string(RoleAggregator),
				Role:        RoleAggregator,
				DeviceTypes: []Definition{aggregatorType()},
				Parts:       []string{sensorKey},
				Servers:     []Definition{descriptorServer()},
			},
			{
				Key:         sensorKey,
				Role:        RoleSensor,
				DeviceTypes: []Definition{bridgedNodeType(), electricalSensorType()},
				Servers:     []Definition{descriptorServer(), bridgedBasicServer(), powerTopologyServer(), powerMeasurementServer()},
			},
		},
	}
}

func nullCurrent(generation, sequence uint64, key, reason string) CurrentUpdate {
	return CurrentUpdate{Generation: generation, Sequence: sequence, SensorKey: key, Null: true, WithheldReason: reason}
}

func (h *ReferenceHost) emit(generation uint64, state LifecycleState) {
	h.ports.Lifecycle.Transition(LifecycleEvent{Generation: generation, State: state})
}

// StartFixture validates the synthetic context and only then constructs a node.
// Every node returned, including one whose Start fails, is stopped exactly once.
func (h *ReferenceHost) StartFixture(c FixtureContext) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch h.state {
	case stateRunning:
		return ErrAlreadyStarted
	case stateFailed:
		return h.failure
	}
	if err := c.validate(); err != nil {
		return err
	}
	if h.pinned != nil && *h.pinned != c.Sensor {
		return ErrSecondSensor
	}
	sensor := c.Sensor
	h.pinned = &sensor
	h.generation++
	generation := h.generation
	plan := buildPlan(c, generation)
	h.plan = &plan
	h.emit(generation, LifecycleStarting)
	node, err := h.ports.SDK.New(plan.clone())
	if err == nil && node == nil {
		err = ErrNoNode
	}
	if err != nil {
		return h.abort(generation, node, err)
	}
	if err := node.Start(); err != nil {
		return h.abort(generation, node, err)
	}
	h.node, h.state, h.attached, h.unresolved = node, stateRunning, true, false
	h.sensorKey = plan.SensorKey
	h.published = nullCurrent(generation, 0, plan.SensorKey, "not-yet-read")
	h.emit(generation, LifecycleRunning)
	return nil
}

func (h *ReferenceHost) abort(generation uint64, node SDKNode, cause error) error {
	if node != nil {
		if err := h.stopNode(generation, node); err != nil {
			return errors.Join(cause, err)
		}
	}
	h.emit(generation, LifecycleStartFailed)
	return cause
}

// stopNode latches a failed stop: late cleanup cannot make a restart safe.
func (h *ReferenceHost) stopNode(generation uint64, node SDKNode) error {
	h.emit(generation, LifecycleStopping)
	if err := node.Stop(); err != nil {
		h.state = stateFailed
		h.failure = fmt.Errorf("%w: %w", ErrQuiesce, err)
		h.emit(generation, LifecycleQuiesceFailed)
		return h.failure
	}
	h.state = stateIdle
	h.emit(generation, LifecycleStopped)
	return nil
}

// Stop is idempotent. The retained last-known-good value survives it.
func (h *ReferenceHost) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch h.state {
	case stateIdle:
		return nil
	case stateFailed:
		return h.failure
	}
	node := h.node
	h.node, h.attached = nil, false
	h.published = nullCurrent(h.generation, h.sequence, h.sensorKey, "stopped")
	return h.stopNode(h.generation, node)
}

// ChangeTopology is the only topology mutator. Any identity other than the
// started sensor is a second sensor and reaches no port. An exact duplicate is
// a no-op. A failed change has an unknown outcome and fences the node until stop.
func (h *ReferenceHost) ChangeTopology(request TopologyRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.state != stateRunning {
		return ErrNotRunning
	}
	if h.unresolved {
		return ErrTopologyUnresolved
	}
	if request.Op != TopologyAttach && request.Op != TopologyDetach {
		return fmt.Errorf("%w: %q", ErrTopologyOp, request.Op)
	}
	if request.Sensor != *h.pinned {
		return ErrSecondSensor
	}
	if (request.Op == TopologyAttach) == h.attached {
		return nil
	}
	h.sequence++
	err := h.node.ApplyTopology(TopologyChange{
		Generation: h.generation,
		Sequence:   h.sequence,
		Op:         request.Op,
		SensorKey:  h.sensorKey,
		Endpoint:   h.plan.Endpoints[2].clone(),
	})
	if err != nil {
		h.unresolved = true
		return err
	}
	h.attached = request.Op == TopologyAttach
	if !h.attached {
		h.published = nullCurrent(h.generation, h.sequence, h.sensorKey, "detached")
	}
	return nil
}

// evaluate returns the value and, when it must be null, why. Absent, stale,
// unqualified or unproven input is withheld; malformed input is also an error.
func evaluate(o CurrentObservation, sensor Identity) (value int64, withheld string, err error) {
	if o.Fact != CurrentFactID {
		return 0, "invalid-fact", fmt.Errorf("%w: %q", ErrProjectionFact, o.Fact)
	}
	if o.Sensor != sensor {
		return 0, "invalid-identity", ErrProjectionIdentity
	}
	switch o.Availability {
	case AvailabilityAvailable:
	case AvailabilityUnavailable, AvailabilityUnknown:
		if o.Value != "" {
			return 0, "invalid-value", fmt.Errorf("%w: value with availability %q", ErrProjectionValue, o.Availability)
		}
		return 0, string(o.Availability), nil
	default:
		return 0, "invalid-state", fmt.Errorf("%w: availability %q", ErrProjectionState, o.Availability)
	}
	switch o.Freshness {
	case FreshnessFresh:
	case FreshnessStale, FreshnessUnknown:
		return 0, "freshness-" + string(o.Freshness), nil
	default:
		return 0, "invalid-state", fmt.Errorf("%w: freshness %q", ErrProjectionState, o.Freshness)
	}
	switch o.Confidence {
	case EvidenceProven:
	case EvidenceHypothesis, EvidenceUnknown:
		return 0, "confidence-" + strings.ToLower(string(o.Confidence)), nil
	default:
		return 0, "invalid-state", fmt.Errorf("%w: confidence %q", ErrProjectionState, o.Confidence)
	}
	if !o.Qualified {
		return 0, "unqualified", nil
	}
	if o.Unit != CurrentUnit {
		return 0, "invalid-unit", fmt.Errorf("%w: %q", ErrProjectionUnit, o.Unit)
	}
	parsed, parseErr := strconv.ParseInt(o.Value, 10, 64)
	if errors.Is(parseErr, strconv.ErrRange) {
		return 0, "out-of-range", ErrProjectionRange
	}
	if parseErr != nil || strconv.FormatInt(parsed, 10) != o.Value {
		return 0, "invalid-value", fmt.Errorf("%w: %q", ErrProjectionValue, o.Value)
	}
	if parsed < -maxCurrentMilliamps || parsed > maxCurrentMilliamps {
		return 0, "out-of-range", ErrProjectionRange
	}
	if !token(o.NativeEvidence) {
		return 0, "no-evidence", ErrProjectionEvidence
	}
	return parsed, "", nil
}

// RefreshCurrent reads the projection and publishes ActiveCurrent only. Anything
// not publishable becomes null: a retained value is kept aside, never reported as
// current, and the valid node and metadata stay untouched.
func (h *ReferenceHost) RefreshCurrent() (CurrentUpdate, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.state != stateRunning {
		return CurrentUpdate{}, ErrNotRunning
	}
	if h.unresolved {
		return CurrentUpdate{}, ErrTopologyUnresolved
	}
	if !h.attached {
		return CurrentUpdate{}, ErrSensorDetached
	}
	observation, err := h.ports.Projection.ReadCurrent(CurrentRequest{Fact: CurrentFactID, Sensor: *h.pinned})
	var value int64
	var withheld string
	if err != nil {
		withheld, err = "read-failed", fmt.Errorf("%w: %w", ErrProjectionRead, err)
	} else {
		value, withheld, err = evaluate(observation, *h.pinned)
	}
	h.sequence++
	update := nullCurrent(h.generation, h.sequence, h.sensorKey, withheld)
	if withheld == "" {
		update.Null, update.Milliamperes, update.NativeEvidence = false, value, observation.NativeEvidence
	}
	if publishErr := h.node.PublishCurrent(update); publishErr != nil {
		return update, errors.Join(err, publishErr)
	}
	h.published = update
	if !update.Null {
		retained := update
		h.lastGood = &retained
	}
	return update, err
}

// Plan returns a detached copy of the most recent plan, if one was built.
func (h *ReferenceHost) Plan() (NodePlan, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.plan == nil {
		return NodePlan{}, false
	}
	return h.plan.clone(), true
}

// Snapshot returns a detached state view.
func (h *ReferenceHost) Snapshot() Snapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := Snapshot{
		Running:    h.state == stateRunning,
		Failed:     h.state == stateFailed,
		Generation: h.generation,
		Attached:   h.attached,
		Published:  h.published,
	}
	if h.lastGood != nil {
		retained := *h.lastGood
		s.LastKnownGood = &retained
	}
	return s
}
