//go:build gofabric

// Package sdkfabric binds the adoption reference host's SDK port to the
// go-fabric node, its Aggregator and the generated Electrical Sensor, Power
// Topology and Electrical Power Measurement facades. It is reachable only
// through an explicitly synthetic fixture configuration with an injected
// launcher, advertiser and in-memory transport; it has no production node
// factory, makes no admission decision and qualifies no target.
package sdkfabric

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Project-Helianthus/helianthus-matter-binding/adoption"
	bridgedinfodef "github.com/SukramJ/go-fabric/cluster/spec/bridgeddevicebasicinformation"
	descriptordef "github.com/SukramJ/go-fabric/cluster/spec/descriptor"
	epmdef "github.com/SukramJ/go-fabric/cluster/spec/electricalpowermeasurement"
	ptdef "github.com/SukramJ/go-fabric/cluster/spec/powertopology"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/mdns"
	"github.com/SukramJ/go-fabric/node"
)

const (
	// SensorLabel is the bridged NodeLabel of the one reference sensor. The
	// full identity travels in the StableKey; a label is not an identity.
	SensorLabel = "Reference sensor"

	syntheticManifestHead = "synthetic:"
)

var (
	ErrPortMissing     = errors.New("sdkfabric: launcher is missing")
	ErrNotSynthetic    = errors.New("sdkfabric: configuration is not an explicitly synthetic fixture")
	ErrAdvertiser      = errors.New("sdkfabric: advertiser is not explicitly injected")
	ErrTransport       = errors.New("sdkfabric: transport is not explicitly in-memory")
	ErrTimeout         = errors.New("sdkfabric: call bound is not positive")
	ErrPlan            = errors.New("sdkfabric: plan differs from the bound reference composition")
	ErrRevision        = errors.New("sdkfabric: plan definition differs from the SDK definition")
	ErrMetadata        = errors.New("sdkfabric: mandatory metadata cannot be encoded")
	ErrNodeState       = errors.New("sdkfabric: node is not in a state for this call")
	ErrChange          = errors.New("sdkfabric: change names another generation or sensor")
	ErrLaunch          = errors.New("sdkfabric: launcher returned neither node nor error")
	ErrNotCommissioned = errors.New("sdkfabric: no CommissioningComplete was observed")
	ErrWindowOpen      = errors.New("sdkfabric: commissioning window is still open")
	ErrWindowClosed    = errors.New("sdkfabric: commissioning window did not open")
)

// Transport names the datagram transport a launcher puts the node on. Only
// TransportInMemory is accepted; the zero value is refused.
type Transport string

const TransportInMemory Transport = "in-memory"

// Launched is a node a launcher built and started. Act serializes access to
// the running node; Stop releases it and is called once for every Launched.
type Launched interface {
	Node() *node.Node
	Act(func(*node.Node) error) error
	Stop(context.Context) error
}

// Launcher builds and starts one node from cfg. This package has no default:
// a nil launcher is refused, so no production node construction is reachable.
type Launcher interface {
	Launch(node.Config) (Launched, error)
}

// Config is the explicit fixture configuration. Advertiser must be non-nil:
// a nil node.Config.Advertiser makes the SDK build its own network advertiser.
type Config struct {
	Origin             adoption.ContextOrigin
	Launcher           Launcher
	Advertiser         mdns.Advertiser
	Transport          Transport
	Listen             string
	Timeout            time.Duration
	Commissioning      node.Commissioning
	ProductDescription node.ProductDescription
	BasicInformation   node.BasicInformation
}

func (c Config) validate() error {
	switch {
	case c.Origin != adoption.OriginSyntheticFixture:
		return fmt.Errorf("%w: origin %q", ErrNotSynthetic, c.Origin)
	case c.Launcher == nil:
		return ErrPortMissing
	case c.Advertiser == nil:
		return ErrAdvertiser
	case c.Transport != TransportInMemory || c.Listen == "":
		return fmt.Errorf("%w: %q", ErrTransport, c.Transport)
	case c.Timeout <= 0:
		return ErrTimeout
	}
	return nil
}

// Stats counts real dispatches to the launcher and their cleanup.
type Stats struct {
	Plans, Launches, LaunchErrors, Stops, StopErrors int
}

// Port implements adoption.SDKPort over go-fabric.
type Port struct {
	cfg   Config
	mu    sync.Mutex
	stats Stats
	nodes []*Node
}

// NewPort validates cfg and calls nothing.
func NewPort(cfg Config) (*Port, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Port{cfg: cfg}, nil
}

// Stats returns the dispatch counters.
func (p *Port) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stats
}

// Nodes returns every node the port constructed, oldest first.
func (p *Port) Nodes() []*Node {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*Node(nil), p.nodes...)
}

func (p *Port) count(fn func(*Stats)) {
	p.mu.Lock()
	fn(&p.stats)
	p.mu.Unlock()
}

// New binds the plan to the SDK's generated definitions and builds the sensor
// device. It launches nothing: the node starts in Start.
func (p *Port) New(plan adoption.NodePlan) (adoption.SDKNode, error) {
	if err := checkPlan(plan); err != nil {
		return nil, err
	}
	pt, epm, err := declarations(plan.Metadata)
	if err != nil {
		return nil, err
	}
	sensor, err := device.NewElectricalSensor(device.Options{
		StableKey: device.StringKey(plan.SensorKey), Name: SensorLabel,
		DeviceAddress: plan.SensorKey, ChannelAddress: plan.SensorKey + ":0",
	}, pt, epm)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMetadata, err)
	}
	n := &Node{port: p, plan: plan, sensor: sensor}
	n.cfg = p.nodeConfig(n)
	p.mu.Lock()
	p.stats.Plans++
	p.nodes = append(p.nodes, n)
	p.mu.Unlock()
	return n, nil
}

// nodeConfig states every transport and advertisement choice explicitly.
func (p *Port) nodeConfig(n *Node) node.Config {
	return node.Config{
		Network: node.Network{
			Listen:     p.cfg.Listen,
			Reannounce: node.Reannounce{Off: true},
			MDNS:       node.MDNS{Off: true},
		},
		Commissioning:      p.cfg.Commissioning,
		ProductDescription: p.cfg.ProductDescription,
		BasicInformation:   p.cfg.BasicInformation,
		Advertiser:         p.cfg.Advertiser,
		Observers: node.Observers{
			OnCommissioningComplete: n.commissioningComplete,
			OnWindowTransition:      n.windowTransition,
		},
	}
}

func definition(d adoption.Definition, id uint32, revision uint16) bool {
	return d.ID == id && d.IDEvidence == adoption.EvidenceProven &&
		d.Revision == revision && d.RevisionEvidence == adoption.EvidenceProven
}

// checkPlan accepts only the AA09 one-sensor shape, with every definition the
// plan declares equal to the tagged SDK definition. A differing revision is
// rejected, never adapted.
func checkPlan(plan adoption.NodePlan) error {
	if plan.DataModel != adoption.DataModelVersion || plan.Generation == 0 ||
		!strings.HasPrefix(plan.Manifest, syntheticManifestHead) || len(plan.Manifest) == len(syntheticManifestHead) {
		return ErrPlan
	}
	if len(plan.Endpoints) != 3 || plan.SensorKey == "" {
		return fmt.Errorf("%w: %d endpoints", ErrPlan, len(plan.Endpoints))
	}
	root, agg, sensor := plan.Endpoints[0], plan.Endpoints[1], plan.Endpoints[2]
	if root.Role != adoption.RoleRoot || agg.Role != adoption.RoleAggregator || sensor.Role != adoption.RoleSensor ||
		sensor.Key != plan.SensorKey || len(agg.Parts) != 1 || agg.Parts[0] != plan.SensorKey {
		return fmt.Errorf("%w: roles or parts", ErrPlan)
	}
	if len(agg.DeviceTypes) != 1 || !definition(agg.DeviceTypes[0], uint32(device.AggregatorID), device.AggregatorRevision) {
		return fmt.Errorf("%w: Aggregator", ErrRevision)
	}
	if len(sensor.DeviceTypes) != 2 || sensor.DeviceTypes[0].Revision != device.BridgedNodeRevision ||
		sensor.DeviceTypes[0].RevisionEvidence != adoption.EvidenceProven ||
		!definition(sensor.DeviceTypes[1], uint32(device.ElectricalSensorID), device.ElectricalSensorRevision) {
		return fmt.Errorf("%w: sensor device types", ErrRevision)
	}
	if len(sensor.Servers) != 4 || sensor.Servers[0].ID != descriptordef.ClusterID ||
		!definition(sensor.Servers[1], bridgedinfodef.ClusterID, bridgedinfodef.Revision) ||
		!definition(sensor.Servers[2], ptdef.ClusterID, ptdef.Revision) ||
		!definition(sensor.Servers[3], epmdef.ClusterID, epmdef.Revision) {
		return fmt.Errorf("%w: sensor servers", ErrRevision)
	}
	return nil
}

var measurementTypes = map[string]epmdef.MeasurementTypeEnum{
	"Voltage":          epmdef.MeasurementTypeVoltage,
	"ActiveCurrent":    epmdef.MeasurementTypeActiveCurrent,
	"ReactiveCurrent":  epmdef.MeasurementTypeReactiveCurrent,
	"ApparentCurrent":  epmdef.MeasurementTypeApparentCurrent,
	"ActivePower":      epmdef.MeasurementTypeActivePower,
	"ReactivePower":    epmdef.MeasurementTypeReactivePower,
	"ApparentPower":    epmdef.MeasurementTypeApparentPower,
	"RmsVoltage":       epmdef.MeasurementTypeRmsVoltage,
	"RmsCurrent":       epmdef.MeasurementTypeRmsCurrent,
	"RmsPower":         epmdef.MeasurementTypeRmsPower,
	"Frequency":        epmdef.MeasurementTypeFrequency,
	"PowerFactor":      epmdef.MeasurementTypePowerFactor,
	"NeutralCurrent":   epmdef.MeasurementTypeNeutralCurrent,
	"ElectricalEnergy": epmdef.MeasurementTypeElectricalEnergy,
	"ReactiveEnergy":   epmdef.MeasurementTypeReactiveEnergy,
	"ApparentEnergy":   epmdef.MeasurementTypeApparentEnergy,
}

// declarations encodes the separately supplied mandatory metadata. Only the
// values AA09 admits are encodable; ActiveCurrent starts null.
func declarations(m adoption.SyntheticMetadata) (ptdef.Declare, epmdef.Declare, error) {
	if m.Topology.Mode != "NODE" || m.PowerMode.Value != "AC" ||
		len(m.Features.Values) != 1 || m.Features.Values[0] != "AC" {
		return ptdef.Declare{}, epmdef.Declare{}, ErrMetadata
	}
	if m.MeasurementTypes.Value < 1 || m.MeasurementTypes.Value > 255 || len(m.Accuracy) != m.MeasurementTypes.Value {
		return ptdef.Declare{}, epmdef.Declare{}, fmt.Errorf("%w: measurement count", ErrMetadata)
	}
	accuracy := make([]epmdef.MeasurementAccuracyStruct, 0, len(m.Accuracy))
	for _, entry := range m.Accuracy {
		kind, ok := measurementTypes[entry.MeasurementType]
		if !ok || entry.FixedMax < 0 || entry.MinMeasured > entry.MaxMeasured {
			return ptdef.Declare{}, epmdef.Declare{}, fmt.Errorf("%w: accuracy %q", ErrMetadata, entry.MeasurementType)
		}
		fixed := uint64(entry.FixedMax)
		accuracy = append(accuracy, epmdef.MeasurementAccuracyStruct{
			MeasurementType: kind, Measured: entry.Measured,
			MinMeasuredValue: entry.MinMeasured, MaxMeasuredValue: entry.MaxMeasured,
			AccuracyRanges: []epmdef.MeasurementAccuracyRangeStruct{{
				RangeMin: entry.MinMeasured, RangeMax: entry.MaxMeasured, FixedMax: &fixed,
			}},
		})
	}
	if !m.ActivePower.Declared || (m.ActivePower.Null && m.ActivePower.Milliwatts != 0) {
		return ptdef.Declare{}, epmdef.Declare{}, fmt.Errorf("%w: ActivePower", ErrMetadata)
	}
	mode := epmdef.PowerModeAc
	count := uint8(m.MeasurementTypes.Value)
	initial := epmdef.Patch{
		PowerMode: &mode, NumberOfMeasurementTypes: &count, Accuracy: &accuracy,
		Null: []uint32{epmdef.AttrActiveCurrent},
	}
	if m.ActivePower.Null {
		initial.Null = append(initial.Null, epmdef.AttrActivePower)
	} else {
		power := m.ActivePower.Milliwatts
		initial.ActivePower = &power
	}
	return ptdef.Declare{Features: ptdef.FeatureNodeTopology},
		epmdef.Declare{Features: epmdef.FeatureAlternatingCurrent, Initial: initial}, nil
}

type nodeState int

const (
	nodeBuilt nodeState = iota
	nodeRunning
	nodeStopped
)

// Node implements adoption.SDKNode over one launched go-fabric node.
type Node struct {
	port   *Port
	plan   adoption.NodePlan
	cfg    node.Config
	sensor *device.ElectricalSensor

	mu       sync.Mutex
	state    nodeState
	launched Launched
	attached bool

	obsMu       sync.Mutex
	completions []uint8
	windows     []node.WindowSnapshot
}

func (n *Node) bound() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), n.port.cfg.Timeout)
}

func (n *Node) commissioningComplete(fabricIndex uint8) {
	n.obsMu.Lock()
	n.completions = append(n.completions, fabricIndex)
	n.obsMu.Unlock()
}

func (n *Node) windowTransition(w node.WindowSnapshot) {
	n.obsMu.Lock()
	n.windows = append(n.windows, w)
	n.obsMu.Unlock()
}

// Completions returns the fabric index of every CommissioningComplete the SDK
// reported. Discovery, PASE and window transitions never add to it.
func (n *Node) Completions() []uint8 {
	n.obsMu.Lock()
	defer n.obsMu.Unlock()
	return append([]uint8(nil), n.completions...)
}

// WindowTransitions returns every window transition the SDK reported.
func (n *Node) WindowTransitions() []node.WindowSnapshot {
	n.obsMu.Lock()
	defer n.obsMu.Unlock()
	return append([]node.WindowSnapshot(nil), n.windows...)
}

// Sensor is the generated device the node serves.
func (n *Node) Sensor() *device.ElectricalSensor { return n.sensor }

// Start launches the node and attaches the one sensor to its Aggregator.
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state != nodeBuilt {
		return ErrNodeState
	}
	n.port.count(func(s *Stats) { s.Launches++ })
	launched, err := n.port.cfg.Launcher.Launch(n.cfg)
	if err == nil && launched == nil {
		err = ErrLaunch
	}
	if err != nil {
		n.port.count(func(s *Stats) { s.LaunchErrors++ })
		n.state = nodeStopped
		return err
	}
	n.launched, n.state = launched, nodeRunning
	ctx, cancel := n.bound()
	defer cancel()
	if err := launched.Act(func(nd *node.Node) error { return nd.Aggregator().Add(ctx, n.sensor) }); err != nil {
		return err
	}
	n.attached = true
	return nil
}

// Stop stops the launched node, if any. The host calls it exactly once.
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	launched := n.launched
	n.launched, n.state, n.attached = nil, nodeStopped, false
	if launched == nil {
		return nil
	}
	n.port.count(func(s *Stats) { s.Stops++ })
	ctx, cancel := n.bound()
	defer cancel()
	if err := launched.Stop(ctx); err != nil {
		n.port.count(func(s *Stats) { s.StopErrors++ })
		return err
	}
	return nil
}

func (n *Node) running() (Launched, error) {
	if n.state != nodeRunning || n.launched == nil {
		return nil, ErrNodeState
	}
	return n.launched, nil
}

// ApplyTopology detaches or reattaches the one sensor. The Aggregator keeps
// the sensor's endpoint number reserved across a detach.
func (n *Node) ApplyTopology(change adoption.TopologyChange) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	launched, err := n.running()
	if err != nil {
		return err
	}
	if change.Generation != n.plan.Generation || change.SensorKey != n.plan.SensorKey || change.Endpoint.Key != n.plan.SensorKey {
		return ErrChange
	}
	ctx, cancel := n.bound()
	defer cancel()
	switch change.Op {
	case adoption.TopologyDetach:
		err = launched.Act(func(nd *node.Node) error {
			return nd.Aggregator().Remove(ctx, device.StringKey(n.plan.SensorKey))
		})
	case adoption.TopologyAttach:
		err = launched.Act(func(nd *node.Node) error { return nd.Aggregator().Add(ctx, n.sensor) })
	default:
		return fmt.Errorf("%w: %q", ErrChange, change.Op)
	}
	if err != nil {
		return err
	}
	n.attached = change.Op == adoption.TopologyAttach
	return nil
}

// PublishCurrent sets ActiveCurrent and nothing else. A nil return means the
// server stored the value; it is not evidence that any subscriber received it.
func (n *Node) PublishCurrent(update adoption.CurrentUpdate) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, err := n.running(); err != nil {
		return err
	}
	if update.Generation != n.plan.Generation || update.SensorKey != n.plan.SensorKey {
		return ErrChange
	}
	if !n.attached {
		return ErrNodeState
	}
	patch := epmdef.Patch{Null: []uint32{epmdef.AttrActiveCurrent}}
	if !update.Null {
		value := update.Milliamperes
		patch = epmdef.Patch{ActiveCurrent: &value}
	}
	return n.sensor.ElectricalPowerMeasurement().Set(patch)
}

// Window returns the SDK's current commissioning window.
func (n *Node) Window() (node.WindowSnapshot, error) {
	n.mu.Lock()
	launched, err := n.running()
	n.mu.Unlock()
	if err != nil {
		return node.WindowSnapshot{}, err
	}
	return launched.Node().Window(), nil
}

// OpenWindow opens a basic window and succeeds only when the SDK then reports
// it open. Opening is not commissioning.
func (n *Node) OpenWindow(ctx context.Context, creds node.WindowCredentials) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	launched, err := n.running()
	if err != nil {
		return err
	}
	if err := launched.Act(func(nd *node.Node) error { return nd.OpenBasicWindow(ctx, creds) }); err != nil {
		return err
	}
	if !launched.Node().Window().Open {
		return ErrWindowClosed
	}
	return nil
}

// CloseWindow asks the SDK to close the window and succeeds only when the SDK
// then reports no open window. A failed close stays a failure; an established
// PASE session or armed fail-safe is not revoked by it.
func (n *Node) CloseWindow(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	launched, err := n.running()
	if err != nil {
		return err
	}
	if err := launched.Act(func(nd *node.Node) error { return nd.CloseWindow(ctx) }); err != nil {
		return err
	}
	if launched.Node().Window().Open {
		return ErrWindowOpen
	}
	return nil
}

// RequireCommissioned succeeds only after a reported CommissioningComplete
// whose fabric the node still holds, with no window left open.
func (n *Node) RequireCommissioned(ctx context.Context) error {
	n.mu.Lock()
	launched, err := n.running()
	n.mu.Unlock()
	if err != nil {
		return err
	}
	completions := n.Completions()
	if len(completions) == 0 {
		return ErrNotCommissioned
	}
	fabrics, err := launched.Node().FabricInfos(ctx)
	if err != nil {
		return err
	}
	held := false
	for _, f := range fabrics {
		held = held || f.FabricIndex == completions[len(completions)-1]
	}
	if !held {
		return fmt.Errorf("%w: fabric %d is not held", ErrNotCommissioned, completions[len(completions)-1])
	}
	if launched.Node().Window().Open {
		return ErrWindowOpen
	}
	return nil
}
