package adoption

import (
	"errors"
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Spies record every port call in one shared ordered trace. They only count and
// return scripted results; nothing here touches a device, a network or a clock.
type aa09Trace struct {
	mu     sync.Mutex
	calls  []string
	events []LifecycleEvent
}

func (t *aa09Trace) add(call string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls = append(t.calls, call)
}

func (t *aa09Trace) list() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.calls...)
}

func (t *aa09Trace) count(call string) int {
	n := 0
	for _, c := range t.list() {
		if c == call {
			n++
		}
	}
	return n
}

type aa09Node struct {
	trace      *aa09Trace
	startErr   error
	stopErr    error
	topoErr    error
	publishErr error
	onTopology func(TopologyChange)
	changes    []TopologyChange
	updates    []CurrentUpdate
}

func (n *aa09Node) Start() error {
	n.trace.add("start")
	return n.startErr
}

func (n *aa09Node) Stop() error {
	n.trace.add("stop")
	return n.stopErr
}

func (n *aa09Node) ApplyTopology(c TopologyChange) error {
	n.trace.add("topology:" + string(c.Op))
	if n.onTopology != nil {
		n.onTopology(c)
	}
	n.changes = append(n.changes, c)
	return n.topoErr
}

func (n *aa09Node) PublishCurrent(u CurrentUpdate) error {
	n.trace.add("publish")
	n.updates = append(n.updates, u)
	return n.publishErr
}

type aa09SDK struct {
	trace  *aa09Trace
	node   func() *aa09Node
	newErr error
	noNode bool
	plans  []NodePlan
}

func (s *aa09SDK) New(p NodePlan) (SDKNode, error) {
	s.trace.add("new")
	s.plans = append(s.plans, p)
	if s.noNode {
		return nil, s.newErr
	}
	return s.node(), s.newErr
}

type aa09Projection struct {
	trace    *aa09Trace
	obs      CurrentObservation
	err      error
	requests []CurrentRequest
}

func (p *aa09Projection) ReadCurrent(r CurrentRequest) (CurrentObservation, error) {
	p.trace.add("read")
	p.requests = append(p.requests, r)
	return p.obs, p.err
}

type aa09Lifecycle struct{ trace *aa09Trace }

func (l *aa09Lifecycle) Transition(e LifecycleEvent) {
	l.trace.add("lifecycle:" + string(e.State))
	l.trace.mu.Lock()
	defer l.trace.mu.Unlock()
	l.trace.events = append(l.trace.events, e)
}

type aa09Harness struct {
	trace      *aa09Trace
	sdk        *aa09SDK
	projection *aa09Projection
	node       *aa09Node
	host       *ReferenceHost
}

func aa09NewHarness(t *testing.T) *aa09Harness {
	t.Helper()
	trace := &aa09Trace{}
	h := &aa09Harness{trace: trace, node: &aa09Node{trace: trace}, projection: &aa09Projection{trace: trace}}
	h.sdk = &aa09SDK{trace: trace, node: func() *aa09Node { return h.node }}
	host, err := NewReferenceHost(Ports{SDK: h.sdk, Projection: h.projection, Lifecycle: &aa09Lifecycle{trace}})
	if err != nil {
		t.Fatalf("NewReferenceHost: %v", err)
	}
	if calls := trace.list(); len(calls) != 0 {
		t.Fatalf("constructor touched ports: %v", calls)
	}
	h.host = host
	return h
}

func aa09Identity() Identity {
	return Identity{
		Node: "synthetic-node", Owner: "synthetic-owner", Profile: "synthetic-profile",
		Capability: "evse/service-a/phase-a", Composition: "synthetic-composition", Binding: "synthetic-binding", Slot: 7,
	}
}

func aa09Context() FixtureContext {
	return FixtureContext{
		Origin:   OriginSyntheticFixture,
		Manifest: "synthetic:reference-manifest",
		Target: TargetDeclaration{
			DataModel: "1.6.1", AggregatorRevision: 2, BridgedNodeRevision: 3, BridgedBasicRevision: 6,
			ElectricalSensorRevision: 1, PowerTopologyRevision: 1, PowerMeasurementRevision: 3,
		},
		Sensor: aa09Identity(),
		Metadata: SyntheticMetadata{
			PowerMode:        PowerModeFact{"AC", SyntheticProvenance("power-mode")},
			MeasurementTypes: CountFact{1, SyntheticProvenance("count")},
			Accuracy: []AccuracyFact{{
				MeasurementType: "ActiveCurrent", Measured: true, MinMeasured: -1000, MaxMeasured: 1000, FixedMax: 5,
				Provenance: SyntheticProvenance("accuracy"),
			}},
			ActivePower: ActivePowerFact{Declared: true, Null: true, Provenance: SyntheticProvenance("active-power")},
			Features:    FeatureFact{[]string{"AC"}, SyntheticProvenance("features")},
			Topology:    TopologyFact{"NODE", SyntheticProvenance("topology")},
		},
	}
}

func aa09Observation(value string) CurrentObservation {
	return CurrentObservation{
		Fact: "evse.ac.current", Sensor: aa09Identity(), Availability: AvailabilityAvailable, Freshness: FreshnessFresh,
		Qualified: true, Confidence: EvidenceProven, Unit: "mA", Value: value, NativeEvidence: "synthetic:native-ref",
	}
}

func aa09AssertZeroPorts(t *testing.T, h *aa09Harness) {
	t.Helper()
	if calls := h.trace.list(); len(calls) != 0 {
		t.Fatalf("expected zero port calls, got %v", calls)
	}
	if _, built := h.host.Plan(); built {
		t.Fatal("a plan was built before a valid fixture context")
	}
}

func aa09MustStart(t *testing.T, h *aa09Harness) {
	t.Helper()
	if err := h.host.StartFixture(aa09Context()); err != nil {
		t.Fatalf("StartFixture: %v", err)
	}
}

func TestAA09NewHostRequiresEveryPort(t *testing.T) {
	trace := &aa09Trace{}
	sdk, projection, lifecycle := &aa09SDK{trace: trace}, &aa09Projection{trace: trace}, &aa09Lifecycle{trace}
	for name, ports := range map[string]Ports{
		"sdk":        {Projection: projection, Lifecycle: lifecycle},
		"projection": {SDK: sdk, Lifecycle: lifecycle},
		"lifecycle":  {SDK: sdk, Projection: projection},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewReferenceHost(ports); !errors.Is(err, ErrPortMissing) {
				t.Fatalf("got %v", err)
			}
		})
	}
	if calls := trace.list(); len(calls) != 0 {
		t.Fatalf("port touched: %v", calls)
	}
}

func TestAA09ModelRevisionsAreIndependentlyExact(t *testing.T) {
	for _, tc := range []struct {
		got              Definition
		name             string
		id               uint32
		idEvidence       Evidence
		revision         uint16
		revisionEvidence Evidence
	}{
		{rootNodeType(), "Root Node", 0, EvidenceUnknown, 0, EvidenceUnknown},
		{aggregatorType(), "Aggregator", 0x000E, EvidenceProven, 2, EvidenceProven},
		{bridgedNodeType(), "Bridged Node", 0, EvidenceUnknown, 3, EvidenceProven},
		{electricalSensorType(), "Electrical Sensor", 0x0510, EvidenceProven, 1, EvidenceProven},
		{descriptorServer(), "Descriptor", 0x001D, EvidenceProven, 0, EvidenceUnknown},
		{bridgedBasicServer(), "Bridged Device Basic Information", 0x0039, EvidenceProven, 6, EvidenceProven},
		{powerTopologyServer(), "Power Topology", 0x009C, EvidenceProven, 1, EvidenceProven},
		{powerMeasurementServer(), "Electrical Power Measurement", 0x0090, EvidenceProven, 3, EvidenceProven},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := Definition{tc.name, tc.id, tc.idEvidence, tc.revision, tc.revisionEvidence}
			if tc.got != want {
				t.Fatalf("got %+v, want %+v", tc.got, want)
			}
		})
	}
	if DataModelVersion != "1.6.1" {
		t.Fatalf("data model %q", DataModelVersion)
	}
}

func TestAA09PlanHasExactDescriptorPartsAndServerLists(t *testing.T) {
	h := aa09NewHarness(t)
	aa09MustStart(t, h)
	plan := h.sdk.plans[0]
	sensorKey := StableKey(aa09Identity())
	if plan.DataModel != "1.6.1" || plan.Generation != 1 || plan.SensorKey != sensorKey || plan.Manifest != "synthetic:reference-manifest" {
		t.Fatalf("plan header %+v", plan)
	}
	descriptor := Definition{"Descriptor", 0x001D, EvidenceProven, 0, EvidenceUnknown}
	want := []EndpointPlan{
		{
			Key: "root", Role: RoleRoot,
			DeviceTypes:            []Definition{{"Root Node", 0, EvidenceUnknown, 0, EvidenceUnknown}},
			Parts:                  []string{"aggregator", sensorKey},
			Servers:                []Definition{descriptor},
			SDKOwnedServersUnknown: true,
		},
		{
			Key: "aggregator", Role: RoleAggregator,
			DeviceTypes: []Definition{{"Aggregator", 0x000E, EvidenceProven, 2, EvidenceProven}},
			Parts:       []string{sensorKey},
			Servers:     []Definition{descriptor},
		},
		{
			Key: sensorKey, Role: RoleSensor,
			DeviceTypes: []Definition{
				{"Bridged Node", 0, EvidenceUnknown, 3, EvidenceProven},
				{"Electrical Sensor", 0x0510, EvidenceProven, 1, EvidenceProven},
			},
			Parts: []string{},
			Servers: []Definition{
				descriptor,
				{"Bridged Device Basic Information", 0x0039, EvidenceProven, 6, EvidenceProven},
				{"Power Topology", 0x009C, EvidenceProven, 1, EvidenceProven},
				{"Electrical Power Measurement", 0x0090, EvidenceProven, 3, EvidenceProven},
			},
		},
	}
	if len(plan.Endpoints) != len(want) {
		t.Fatalf("endpoints: got %d want %d", len(plan.Endpoints), len(want))
	}
	sensors := 0
	for i := range want {
		got := plan.Endpoints[i]
		if got.Key != want[i].Key || got.Role != want[i].Role || got.SDKOwnedServersUnknown != want[i].SDKOwnedServersUnknown {
			t.Errorf("endpoint %d header %+v", i, got)
		}
		if !reflect.DeepEqual(got.DeviceTypes, want[i].DeviceTypes) {
			t.Errorf("endpoint %d device types %+v", i, got.DeviceTypes)
		}
		if strings.Join(got.Parts, ",") != strings.Join(want[i].Parts, ",") {
			t.Errorf("endpoint %d parts %v", i, got.Parts)
		}
		if !reflect.DeepEqual(got.Servers, want[i].Servers) {
			t.Errorf("endpoint %d servers %+v", i, got.Servers)
		}
		for _, deviceType := range got.DeviceTypes {
			if deviceType.Name == "Electrical Sensor" {
				sensors++
			}
		}
	}
	if sensors != 1 {
		t.Fatalf("electrical sensors: %d", sensors)
	}
	if !reflect.DeepEqual(plan.Metadata, aa09Context().Metadata) {
		t.Fatalf("metadata changed in the plan: %+v", plan.Metadata)
	}
}

func TestAA09InvalidContextReachesNoPort(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*FixtureContext)
		want   error
	}{
		// Production and non-synthetic origins, with otherwise complete metadata.
		{"origin zero", func(c *FixtureContext) { c.Origin = "" }, ErrContextNotSynthetic},
		{"origin production", func(c *FixtureContext) { c.Origin = OriginProduction }, ErrContextNotSynthetic},
		{"origin near miss", func(c *FixtureContext) { c.Origin = "synthetic" }, ErrContextNotSynthetic},
		{"manifest empty", func(c *FixtureContext) { c.Manifest = "" }, ErrManifest},
		{"manifest not synthetic", func(c *FixtureContext) { c.Manifest = "reference-manifest" }, ErrManifest},
		{"manifest bare prefix", func(c *FixtureContext) { c.Manifest = "synthetic:" }, ErrManifest},
		// Each declared revision, one at a time, at old or wrong values.
		{"data model 1.7", func(c *FixtureContext) { c.Target.DataModel = "1.7" }, ErrTarget},
		{"old sensor revision", func(c *FixtureContext) { c.Target.ElectricalSensorRevision = 2 }, ErrTarget},
		{"old topology revision", func(c *FixtureContext) { c.Target.PowerTopologyRevision = 2 }, ErrTarget},
		{"measurement revision", func(c *FixtureContext) { c.Target.PowerMeasurementRevision = 2 }, ErrTarget},
		{"aggregator revision", func(c *FixtureContext) { c.Target.AggregatorRevision = 1 }, ErrTarget},
		{"bridged node revision", func(c *FixtureContext) { c.Target.BridgedNodeRevision = 2 }, ErrTarget},
		{"bridged basic revision", func(c *FixtureContext) { c.Target.BridgedBasicRevision = 5 }, ErrTarget},
		// Each identity axis.
		{"identity node", func(c *FixtureContext) { c.Sensor.Node = "" }, ErrIdentity},
		{"identity owner", func(c *FixtureContext) { c.Sensor.Owner = "" }, ErrIdentity},
		{"identity profile", func(c *FixtureContext) { c.Sensor.Profile = "" }, ErrIdentity},
		{"identity capability", func(c *FixtureContext) { c.Sensor.Capability = "" }, ErrIdentity},
		{"identity composition", func(c *FixtureContext) { c.Sensor.Composition = "" }, ErrIdentity},
		{"identity binding", func(c *FixtureContext) { c.Sensor.Binding = "" }, ErrIdentity},
		{"identity zero slot", func(c *FixtureContext) { c.Sensor.Slot = 0 }, ErrIdentity},
		{"identity negative slot", func(c *FixtureContext) { c.Sensor.Slot = -1 }, ErrIdentity},
		{"identity whitespace", func(c *FixtureContext) { c.Sensor.Node = " x" }, ErrIdentity},
		{"identity control", func(c *FixtureContext) { c.Sensor.Node = "a\nb" }, ErrIdentity},
		{"identity oversized", func(c *FixtureContext) { c.Sensor.Node = strings.Repeat("a", 129) }, ErrIdentity},
		// PowerMode: missing, Unknown and unsupported are distinct inputs.
		{"power mode missing", func(c *FixtureContext) { c.Metadata.PowerMode.Value = "" }, ErrPowerMode},
		{"power mode unknown", func(c *FixtureContext) { c.Metadata.PowerMode.Value = "Unknown" }, ErrPowerMode},
		{"power mode dc", func(c *FixtureContext) { c.Metadata.PowerMode.Value = "DC" }, ErrPowerMode},
		{"power mode case", func(c *FixtureContext) { c.Metadata.PowerMode.Value = "ac" }, ErrPowerMode},
		{"power mode absent fact", func(c *FixtureContext) { c.Metadata.PowerMode = PowerModeFact{} }, ErrPowerMode},
		// Count: zero, negative, bound and mismatch.
		{"count zero", func(c *FixtureContext) { c.Metadata.MeasurementTypes.Value = 0 }, ErrMeasurementCount},
		{"count negative", func(c *FixtureContext) { c.Metadata.MeasurementTypes.Value = -1 }, ErrMeasurementCount},
		{"count over bound", func(c *FixtureContext) { c.Metadata.MeasurementTypes.Value = 9 }, ErrMeasurementCount},
		{"count exceeds accuracy", func(c *FixtureContext) { c.Metadata.MeasurementTypes.Value = 2 }, ErrAccuracy},
		// Accuracy: missing, empty and malformed entries.
		{"accuracy nil", func(c *FixtureContext) { c.Metadata.Accuracy = nil }, ErrAccuracy},
		{"accuracy empty", func(c *FixtureContext) { c.Metadata.Accuracy = []AccuracyFact{} }, ErrAccuracy},
		{"accuracy no type", func(c *FixtureContext) { c.Metadata.Accuracy[0].MeasurementType = "" }, ErrAccuracy},
		{"accuracy inverted", func(c *FixtureContext) { c.Metadata.Accuracy[0].MinMeasured = 2000 }, ErrAccuracy},
		{"accuracy negative fixed", func(c *FixtureContext) { c.Metadata.Accuracy[0].FixedMax = -1 }, ErrAccuracy},
		{"accuracy duplicate", func(c *FixtureContext) {
			c.Metadata.MeasurementTypes.Value = 2
			c.Metadata.Accuracy = append(c.Metadata.Accuracy, c.Metadata.Accuracy[0])
		}, ErrAccuracy},
		// ActivePower: undeclared differs from declared null and known zero.
		{"active power undeclared", func(c *FixtureContext) { c.Metadata.ActivePower.Declared = false }, ErrActivePower},
		{"active power absent fact", func(c *FixtureContext) { c.Metadata.ActivePower = ActivePowerFact{} }, ErrActivePower},
		{"active power null with value", func(c *FixtureContext) { c.Metadata.ActivePower.Milliwatts = 1 }, ErrActivePower},
		// Features and topology metadata.
		{"features missing", func(c *FixtureContext) { c.Metadata.Features.Values = nil }, ErrFeatures},
		{"features dc", func(c *FixtureContext) { c.Metadata.Features.Values = []string{"DC"} }, ErrFeatures},
		{"features extra", func(c *FixtureContext) { c.Metadata.Features.Values = []string{"AC", "POLY"} }, ErrFeatures},
		{"topology missing", func(c *FixtureContext) { c.Metadata.Topology.Mode = "" }, ErrTopologyFact},
		{"topology tree", func(c *FixtureContext) { c.Metadata.Topology.Mode = "TREE" }, ErrTopologyFact},
		// Each fact's provenance independently, including a current-derived label.
		{"provenance power mode", func(c *FixtureContext) {
			c.Metadata.PowerMode.Provenance = Provenance{"evse.ac.current", "synthetic:x"}
		}, ErrProvenance},
		{"provenance count", func(c *FixtureContext) { c.Metadata.MeasurementTypes.Provenance.Source = "native" }, ErrProvenance},
		{"provenance accuracy", func(c *FixtureContext) { c.Metadata.Accuracy[0].Provenance.Reference = "native:x" }, ErrProvenance},
		{"provenance active power", func(c *FixtureContext) { c.Metadata.ActivePower.Provenance = Provenance{} }, ErrProvenance},
		{"provenance features", func(c *FixtureContext) { c.Metadata.Features.Provenance.Reference = "synthetic:" }, ErrProvenance},
		{"provenance topology", func(c *FixtureContext) { c.Metadata.Topology.Provenance.Source = "inferred" }, ErrProvenance},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A valid current must not be able to repair the missing fact.
			h := aa09NewHarness(t)
			h.projection.obs = aa09Observation("16000")
			ctx := aa09Context()
			tc.mutate(&ctx)
			if err := h.host.StartFixture(ctx); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			aa09AssertZeroPorts(t, h)
		})
	}
}

func TestAA09ZeroContextAllocatesNothing(t *testing.T) {
	h := aa09NewHarness(t)
	if err := h.host.StartFixture(FixtureContext{}); !errors.Is(err, ErrContextNotSynthetic) {
		t.Fatalf("got %v", err)
	}
	aa09AssertZeroPorts(t, h)
	snapshot := h.host.Snapshot()
	if snapshot.Running || snapshot.Generation != 0 {
		t.Fatalf("snapshot %+v", snapshot)
	}
}

// The production launcher seam is the same narrow SDK port: every spy supplied
// with a production-labelled but otherwise complete context stays at zero.
func TestAA09ProductionContextLeavesLauncherSpiesAtZero(t *testing.T) {
	production := aa09NewHarness(t)
	ctx := aa09Context()
	ctx.Origin = OriginProduction
	if err := production.host.StartFixture(ctx); !errors.Is(err, ErrContextNotSynthetic) {
		t.Fatalf("got %v", err)
	}
	aa09AssertZeroPorts(t, production)
	// A rejected production attempt also must not unlock the later fixture path.
	if _, err := production.host.RefreshCurrent(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("refresh: %v", err)
	}
	if err := production.host.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	aa09AssertZeroPorts(t, production)
}

func TestAA09SourceHasNoProductionOrForeignDependency(t *testing.T) {
	file, err := parser.ParseFile(gotoken.NewFileSet(), "reference.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	allowed := map[string]bool{"errors": true, "fmt": true, "strconv": true, "strings": true, "sync": true, "unicode": true, "unicode/utf8": true}
	for _, spec := range file.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		if !allowed[path] {
			t.Errorf("unexpected import %q", path)
		}
	}
	full, err := parser.ParseFile(gotoken.NewFileSet(), "reference.go", nil, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var entries []string
	for _, decl := range full.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() {
			continue
		}
		if strings.Contains(strings.ToLower(fn.Name.Name), "production") || strings.Contains(strings.ToLower(fn.Name.Name), "launch") {
			t.Errorf("production entry point %s", fn.Name.Name)
		}
		if strings.HasPrefix(fn.Name.Name, "Start") {
			entries = append(entries, fn.Name.Name)
		}
	}
	if strings.Join(entries, ",") != "StartFixture" {
		t.Fatalf("start entry points: %v", entries)
	}
}

func TestAA09StartOrderValidatesThenConstructsThenStarts(t *testing.T) {
	h := aa09NewHarness(t)
	aa09MustStart(t, h)
	want := []string{"lifecycle:starting", "new", "start", "lifecycle:running"}
	if got := h.trace.list(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// Starting reads no projection, so no mandatory fact can see a current.
	if h.projection.requests != nil {
		t.Fatalf("projection read during start: %+v", h.projection.requests)
	}
	snapshot := h.host.Snapshot()
	if !snapshot.Running || !snapshot.Attached || !snapshot.Published.Null || snapshot.LastKnownGood != nil {
		t.Fatalf("snapshot %+v", snapshot)
	}
}

func TestAA09StopsEveryReturnedNode(t *testing.T) {
	boom := errors.New("synthetic failure")
	for _, tc := range []struct {
		name      string
		arrange   func(*aa09Harness)
		wantErr   error
		wantTrace []string
	}{
		{
			"constructor error without node",
			func(h *aa09Harness) { h.sdk.noNode, h.sdk.newErr = true, boom },
			boom,
			[]string{"lifecycle:starting", "new", "lifecycle:start-failed"},
		},
		{
			"constructor error with node",
			func(h *aa09Harness) { h.sdk.newErr = boom },
			boom,
			[]string{"lifecycle:starting", "new", "lifecycle:stopping", "stop", "lifecycle:stopped", "lifecycle:start-failed"},
		},
		{
			"constructor returns nothing",
			func(h *aa09Harness) { h.sdk.noNode = true },
			ErrNoNode,
			[]string{"lifecycle:starting", "new", "lifecycle:start-failed"},
		},
		{
			"start failure",
			func(h *aa09Harness) { h.node.startErr = boom },
			boom,
			[]string{"lifecycle:starting", "new", "start", "lifecycle:stopping", "stop", "lifecycle:stopped", "lifecycle:start-failed"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := aa09NewHarness(t)
			tc.arrange(h)
			if err := h.host.StartFixture(aa09Context()); !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
			if got := h.trace.list(); !reflect.DeepEqual(got, tc.wantTrace) {
				t.Fatalf("got %v, want %v", got, tc.wantTrace)
			}
			if h.host.Snapshot().Running {
				t.Fatal("reported running after a failed start")
			}
			// The failed start is clean: a later stop has nothing left to stop.
			before := h.trace.count("stop")
			if err := h.host.Stop(); err != nil {
				t.Fatalf("stop: %v", err)
			}
			if h.trace.count("stop") != before {
				t.Fatal("stopped a node twice")
			}
		})
	}
}

func TestAA09StopAfterSuccessIsOnceAndIdempotent(t *testing.T) {
	h := aa09NewHarness(t)
	aa09MustStart(t, h)
	for range 3 {
		if err := h.host.Stop(); err != nil {
			t.Fatalf("stop: %v", err)
		}
	}
	if got := h.trace.count("stop"); got != 1 {
		t.Fatalf("stops: %d", got)
	}
	if h.host.Snapshot().Running {
		t.Fatal("still running")
	}
}

func TestAA09FailedStopLatchesAndProhibitsRestart(t *testing.T) {
	boom := errors.New("synthetic stop failure")
	t.Run("stop", func(t *testing.T) {
		h := aa09NewHarness(t)
		h.node.stopErr = boom
		aa09MustStart(t, h)
		if err := h.host.Stop(); !errors.Is(err, ErrQuiesce) || !errors.Is(err, boom) {
			t.Fatalf("got %v", err)
		}
		if err := h.host.Stop(); !errors.Is(err, ErrQuiesce) {
			t.Fatalf("second stop %v", err)
		}
		if got := h.trace.count("stop"); got != 1 {
			t.Fatalf("stops: %d", got)
		}
		if !h.host.Snapshot().Failed {
			t.Fatal("failure not latched")
		}
		if err := h.host.StartFixture(aa09Context()); !errors.Is(err, ErrQuiesce) {
			t.Fatalf("restart %v", err)
		}
		if got := h.trace.count("new"); got != 1 {
			t.Fatalf("constructed again: %d", got)
		}
	})
	t.Run("start failure then stop failure", func(t *testing.T) {
		h := aa09NewHarness(t)
		startErr := errors.New("synthetic start failure")
		h.node.startErr, h.node.stopErr = startErr, boom
		err := h.host.StartFixture(aa09Context())
		if !errors.Is(err, startErr) || !errors.Is(err, ErrQuiesce) || !errors.Is(err, boom) {
			t.Fatalf("got %v", err)
		}
		if err := h.host.StartFixture(aa09Context()); !errors.Is(err, ErrQuiesce) {
			t.Fatalf("restart %v", err)
		}
		if got := h.trace.count("new"); got != 1 {
			t.Fatalf("constructed again: %d", got)
		}
	})
}

func TestAA09DuplicateStartAllocatesNothing(t *testing.T) {
	h := aa09NewHarness(t)
	aa09MustStart(t, h)
	if err := h.host.StartFixture(aa09Context()); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("got %v", err)
	}
	if got := h.trace.count("new"); got != 1 {
		t.Fatalf("new: %d", got)
	}
}

func TestAA09RestartCyclesUseFreshGenerations(t *testing.T) {
	h := aa09NewHarness(t)
	for cycle := uint64(1); cycle <= 3; cycle++ {
		aa09MustStart(t, h)
		if got := h.sdk.plans[cycle-1].Generation; got != cycle {
			t.Fatalf("plan generation %d in cycle %d", got, cycle)
		}
		if err := h.host.Stop(); err != nil {
			t.Fatalf("stop: %v", err)
		}
	}
	if h.trace.count("new") != 3 || h.trace.count("stop") != 3 {
		t.Fatalf("trace %v", h.trace.list())
	}
	h.trace.mu.Lock()
	defer h.trace.mu.Unlock()
	if len(h.trace.events) != 12 {
		t.Fatalf("events %d", len(h.trace.events))
	}
	for i, event := range h.trace.events {
		if event.Generation != uint64(i/4+1) {
			t.Fatalf("event %d generation %d", i, event.Generation)
		}
	}
}

// aa09Axes varies every identity field independently of the others. The field
// names map to one production mutation each: dropping that field from StableKey
// (key test) or relaxing the struct comparison (admission and echo tests).
func aa09Axes() map[string]func(*Identity) {
	return map[string]func(*Identity){
		"node":        func(i *Identity) { i.Node = "synthetic-node-b" },
		"owner":       func(i *Identity) { i.Owner = "synthetic-owner-b" },
		"profile":     func(i *Identity) { i.Profile = "synthetic-profile-b" },
		"capability":  func(i *Identity) { i.Capability = "evse/service-a/phase-b" },
		"composition": func(i *Identity) { i.Composition = "synthetic-composition-b" },
		"binding":     func(i *Identity) { i.Binding = "synthetic-binding-b" },
		"slot":        func(i *Identity) { i.Slot = 8 },
	}
}

func TestAA09StableKeyVariesWithEveryAxis(t *testing.T) {
	base := StableKey(aa09Identity())
	seen := map[string]string{base: "base"}
	for name, vary := range aa09Axes() {
		id := aa09Identity()
		vary(&id)
		key := StableKey(id)
		if prior, dup := seen[key]; dup {
			t.Errorf("axis %s collides with %s", name, prior)
		}
		seen[key] = name
	}
	left, right := aa09Identity(), aa09Identity()
	left.Node, left.Owner = "ab", "c"
	right.Node, right.Owner = "a", "bc"
	if StableKey(left) == StableKey(right) {
		t.Fatal("adjacent axes can trade bytes")
	}
	left, right = aa09Identity(), aa09Identity()
	left.Binding, right.Binding = "b1", "b"
	left.Slot, right.Slot = 2, 12
	if StableKey(left) == StableKey(right) {
		t.Fatal("binding and slot can trade bytes")
	}
}

func TestAA09SecondSensorIsRejectedOnEveryAxis(t *testing.T) {
	for name, vary := range aa09Axes() {
		t.Run(name, func(t *testing.T) {
			h := aa09NewHarness(t)
			aa09MustStart(t, h)
			id := aa09Identity()
			vary(&id)
			if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyAttach, Sensor: id}); !errors.Is(err, ErrSecondSensor) {
				t.Fatalf("attach: %v", err)
			}
			if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyDetach, Sensor: id}); !errors.Is(err, ErrSecondSensor) {
				t.Fatalf("detach: %v", err)
			}
			if h.trace.count("topology:attach")+h.trace.count("topology:detach") != 0 {
				t.Fatalf("reached SDK: %v", h.trace.list())
			}
			// Restarting with another identity is the same second sensor.
			if err := h.host.Stop(); err != nil {
				t.Fatalf("stop: %v", err)
			}
			ctx := aa09Context()
			ctx.Sensor = id
			if err := h.host.StartFixture(ctx); !errors.Is(err, ErrSecondSensor) {
				t.Fatalf("restart: %v", err)
			}
			if got := h.trace.count("new"); got != 1 {
				t.Fatalf("new: %d", got)
			}
		})
	}
}

func TestAA09TopologyChangeFlow(t *testing.T) {
	h := aa09NewHarness(t)
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyDetach, Sensor: aa09Identity()}); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("before start: %v", err)
	}
	aa09MustStart(t, h)
	if err := h.host.ChangeTopology(TopologyRequest{Op: "retire", Sensor: aa09Identity()}); !errors.Is(err, ErrTopologyOp) {
		t.Fatalf("bad op: %v", err)
	}
	// Attaching an attached sensor is an exact duplicate.
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyAttach, Sensor: aa09Identity()}); err != nil {
		t.Fatalf("duplicate attach: %v", err)
	}
	if got := h.trace.count("topology:attach"); got != 0 {
		t.Fatalf("duplicate reached SDK: %d", got)
	}
	h.projection.obs = aa09Observation("16000")
	if _, err := h.host.RefreshCurrent(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyDetach, Sensor: aa09Identity()}); err != nil {
		t.Fatalf("detach: %v", err)
	}
	snapshot := h.host.Snapshot()
	if snapshot.Attached || !snapshot.Published.Null || snapshot.LastKnownGood == nil || snapshot.LastKnownGood.Milliamperes != 16000 {
		t.Fatalf("after detach %+v", snapshot)
	}
	reads := h.trace.count("read")
	if _, err := h.host.RefreshCurrent(); !errors.Is(err, ErrSensorDetached) {
		t.Fatalf("refresh detached: %v", err)
	}
	if h.trace.count("read") != reads {
		t.Fatal("read while detached")
	}
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyAttach, Sensor: aa09Identity()}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if len(h.node.changes) != 2 {
		t.Fatalf("changes %+v", h.node.changes)
	}
	key := StableKey(aa09Identity())
	for i, change := range h.node.changes {
		if change.Generation != 1 || change.SensorKey != key || change.Endpoint.Key != key || change.Endpoint.Role != RoleSensor {
			t.Fatalf("change %d %+v", i, change)
		}
	}
	if h.node.changes[0].Op != TopologyDetach || h.node.changes[1].Op != TopologyAttach ||
		h.node.changes[0].Sequence >= h.node.changes[1].Sequence {
		t.Fatalf("order %+v", h.node.changes)
	}
}

func TestAA09FailedTopologyChangeFencesUntilStop(t *testing.T) {
	boom := errors.New("synthetic topology failure")
	h := aa09NewHarness(t)
	h.node.topoErr = boom
	aa09MustStart(t, h)
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyDetach, Sensor: aa09Identity()}); !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyAttach, Sensor: aa09Identity()}); !errors.Is(err, ErrTopologyUnresolved) {
		t.Fatalf("second change %v", err)
	}
	if _, err := h.host.RefreshCurrent(); !errors.Is(err, ErrTopologyUnresolved) {
		t.Fatalf("refresh %v", err)
	}
	if err := h.host.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if got := h.trace.count("stop"); got != 1 {
		t.Fatalf("stops %d", got)
	}
	h.node = &aa09Node{trace: h.trace}
	aa09MustStart(t, h)
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyDetach, Sensor: aa09Identity()}); err != nil {
		t.Fatalf("fresh generation still fenced: %v", err)
	}
}

// TryLock inside the port is deterministic: a host that does not hold its lock
// across the SDK call fails on the first change, with no timing involved.
func TestAA09TopologyChangesHoldTheSerializationLock(t *testing.T) {
	h := aa09NewHarness(t)
	calls := 0
	h.node.onTopology = func(TopologyChange) {
		calls++
		if h.host.mu.TryLock() {
			h.host.mu.Unlock()
			t.Error("lock not held across the SDK topology call")
		}
	}
	aa09MustStart(t, h)
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyDetach, Sensor: aa09Identity()}); err != nil {
		t.Fatalf("detach: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls %d", calls)
	}
}

func TestAA09ConcurrentTopologyChangesNeverOverlap(t *testing.T) {
	h := aa09NewHarness(t)
	var mu sync.Mutex
	inFlight, peak := 0, 0
	h.node.onTopology = func(TopologyChange) {
		mu.Lock()
		inFlight++
		if inFlight > peak {
			peak = inFlight
		}
		mu.Unlock()
		mu.Lock()
		inFlight--
		mu.Unlock()
	}
	aa09MustStart(t, h)
	var wg sync.WaitGroup
	for i := range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op := TopologyAttach
			if i%2 == 0 {
				op = TopologyDetach
			}
			if err := h.host.ChangeTopology(TopologyRequest{Op: op, Sensor: aa09Identity()}); err != nil {
				t.Errorf("change: %v", err)
			}
		}()
	}
	wg.Wait()
	if peak > 1 {
		t.Fatalf("overlapping topology changes: %d", peak)
	}
	for i := 1; i < len(h.node.changes); i++ {
		if h.node.changes[i].Sequence <= h.node.changes[i-1].Sequence {
			t.Fatalf("sequence order %+v", h.node.changes)
		}
		if h.node.changes[i].Op == h.node.changes[i-1].Op {
			t.Fatalf("duplicate transition applied %+v", h.node.changes)
		}
	}
}

func TestAA09CurrentValueCases(t *testing.T) {
	limit := strconv.FormatUint(1<<62, 10)
	over := strconv.FormatUint(1<<62+1, 10)
	for _, tc := range []struct {
		name   string
		obs    func(*CurrentObservation)
		null   bool
		value  int64
		reason string
		err    error
	}{
		{name: "positive", obs: func(o *CurrentObservation) { o.Value = "16000" }, value: 16000},
		{name: "known zero is not null", obs: func(o *CurrentObservation) { o.Value = "0" }, value: 0},
		{name: "negative", obs: func(o *CurrentObservation) { o.Value = "-250" }, value: -250},
		{name: "upper bound", obs: func(o *CurrentObservation) { o.Value = limit }, value: 1 << 62},
		{name: "lower bound", obs: func(o *CurrentObservation) { o.Value = "-" + limit }, value: -(1 << 62)},

		{name: "unavailable", obs: func(o *CurrentObservation) { o.Availability, o.Value = AvailabilityUnavailable, "" }, null: true, reason: "unavailable"},
		{name: "unknown availability", obs: func(o *CurrentObservation) { o.Availability, o.Value = AvailabilityUnknown, "" }, null: true, reason: "unknown"},
		{name: "unavailable with value", obs: func(o *CurrentObservation) { o.Availability, o.Value = AvailabilityUnavailable, "1" }, null: true, reason: "invalid-value", err: ErrProjectionValue},
		{name: "unknown with zero", obs: func(o *CurrentObservation) { o.Availability, o.Value = AvailabilityUnknown, "0" }, null: true, reason: "invalid-value", err: ErrProjectionValue},
		{name: "stale", obs: func(o *CurrentObservation) { o.Freshness = FreshnessStale }, null: true, reason: "freshness-stale"},
		{name: "unknown freshness", obs: func(o *CurrentObservation) { o.Freshness = FreshnessUnknown }, null: true, reason: "freshness-unknown"},
		{name: "unqualified", obs: func(o *CurrentObservation) { o.Qualified = false }, null: true, reason: "unqualified"},
		{name: "hypothesis", obs: func(o *CurrentObservation) { o.Confidence = EvidenceHypothesis }, null: true, reason: "confidence-hypothesis"},
		{name: "unknown confidence", obs: func(o *CurrentObservation) { o.Confidence = EvidenceUnknown }, null: true, reason: "confidence-unknown"},

		{name: "available without value", obs: func(o *CurrentObservation) { o.Value = "" }, null: true, reason: "invalid-value", err: ErrProjectionValue},
		{name: "literal null", obs: func(o *CurrentObservation) { o.Value = "null" }, null: true, reason: "invalid-value", err: ErrProjectionValue},
		{name: "fraction", obs: func(o *CurrentObservation) { o.Value = "1.5" }, null: true, reason: "invalid-value", err: ErrProjectionValue},
		{name: "explicit plus", obs: func(o *CurrentObservation) { o.Value = "+1" }, null: true, reason: "invalid-value", err: ErrProjectionValue},
		{name: "leading zero", obs: func(o *CurrentObservation) { o.Value = "01" }, null: true, reason: "invalid-value", err: ErrProjectionValue},
		{name: "negative zero", obs: func(o *CurrentObservation) { o.Value = "-0" }, null: true, reason: "invalid-value", err: ErrProjectionValue},

		{name: "unit amperes", obs: func(o *CurrentObservation) { o.Unit = "A" }, null: true, reason: "invalid-unit", err: ErrProjectionUnit},
		{name: "unit missing", obs: func(o *CurrentObservation) { o.Unit = "" }, null: true, reason: "invalid-unit", err: ErrProjectionUnit},
		{name: "unit power", obs: func(o *CurrentObservation) { o.Unit = "mW" }, null: true, reason: "invalid-unit", err: ErrProjectionUnit},

		{name: "over upper bound", obs: func(o *CurrentObservation) { o.Value = over }, null: true, reason: "out-of-range", err: ErrProjectionRange},
		{name: "under lower bound", obs: func(o *CurrentObservation) { o.Value = "-" + over }, null: true, reason: "out-of-range", err: ErrProjectionRange},
		{name: "over int64", obs: func(o *CurrentObservation) { o.Value = strconv.FormatUint(1<<63, 10) }, null: true, reason: "out-of-range", err: ErrProjectionRange},

		{name: "other fact", obs: func(o *CurrentObservation) { o.Fact = "evse.ac.power" }, null: true, reason: "invalid-fact", err: ErrProjectionFact},
		{name: "zero observation", obs: func(o *CurrentObservation) { *o = CurrentObservation{} }, null: true, reason: "invalid-fact", err: ErrProjectionFact},
		{name: "invalid availability", obs: func(o *CurrentObservation) { o.Availability = "" }, null: true, reason: "invalid-state", err: ErrProjectionState},
		{name: "invalid freshness", obs: func(o *CurrentObservation) { o.Freshness = "" }, null: true, reason: "invalid-state", err: ErrProjectionState},
		{name: "invalid confidence", obs: func(o *CurrentObservation) { o.Confidence = "" }, null: true, reason: "invalid-state", err: ErrProjectionState},
		{name: "no native evidence", obs: func(o *CurrentObservation) { o.NativeEvidence = "" }, null: true, reason: "no-evidence", err: ErrProjectionEvidence},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := aa09NewHarness(t)
			aa09MustStart(t, h)
			h.projection.obs = aa09Observation("16000")
			tc.obs(&h.projection.obs)
			update, err := h.host.RefreshCurrent()
			if tc.err == nil && err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) {
				t.Fatalf("got %v, want %v", err, tc.err)
			}
			if update.Null != tc.null || update.Milliamperes != tc.value || update.WithheldReason != tc.reason {
				t.Fatalf("update %+v", update)
			}
			if tc.null && update.NativeEvidence != "" {
				t.Fatalf("withheld update carries evidence %+v", update)
			}
			if !tc.null && update.NativeEvidence != "synthetic:native-ref" {
				t.Fatalf("native evidence lost %+v", update)
			}
			if len(h.node.updates) != 1 || h.node.updates[0] != update {
				t.Fatalf("sdk updates %+v", h.node.updates)
			}
			snapshot := h.host.Snapshot()
			if (snapshot.LastKnownGood != nil) == tc.null {
				t.Fatalf("last known good %+v", snapshot.LastKnownGood)
			}
		})
	}
}

func TestAA09ProjectionRequestAndEchoCoverEveryAxis(t *testing.T) {
	for name, vary := range aa09Axes() {
		t.Run(name, func(t *testing.T) {
			h := aa09NewHarness(t)
			aa09MustStart(t, h)
			h.projection.obs = aa09Observation("16000")
			vary(&h.projection.obs.Sensor)
			update, err := h.host.RefreshCurrent()
			if !errors.Is(err, ErrProjectionIdentity) || !update.Null {
				t.Fatalf("update %+v err %v", update, err)
			}
			if h.projection.requests[0] != (CurrentRequest{Fact: "evse.ac.current", Sensor: aa09Identity()}) {
				t.Fatalf("request %+v", h.projection.requests[0])
			}
		})
	}
}

func TestAA09CurrentCannotChangeMandatoryFacts(t *testing.T) {
	h := aa09NewHarness(t)
	aa09MustStart(t, h)
	before, _ := h.host.Plan()
	for _, value := range []string{"0", "16000", "-250"} {
		h.projection.obs = aa09Observation(value)
		if _, err := h.host.RefreshCurrent(); err != nil {
			t.Fatalf("refresh: %v", err)
		}
	}
	after, _ := h.host.Plan()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("plan changed: %+v", after)
	}
	if after.Metadata.ActivePower != aa09Context().Metadata.ActivePower || !after.Metadata.ActivePower.Null {
		t.Fatalf("ActivePower was derived: %+v", after.Metadata.ActivePower)
	}
	// The SDK sees only the current DTO, and only after the node started.
	calls := h.trace.list()
	if calls[1] != "new" || calls[2] != "start" {
		t.Fatalf("trace %v", calls)
	}
	// Each mandatory value is carried as supplied, not normalized to a default.
	h2 := aa09NewHarness(t)
	ctx := aa09Context()
	ctx.Metadata.ActivePower = ActivePowerFact{Declared: true, Milliwatts: 0, Provenance: SyntheticProvenance("known-zero")}
	ctx.Metadata.MeasurementTypes.Value = 2
	ctx.Metadata.Accuracy = append(ctx.Metadata.Accuracy, AccuracyFact{
		MeasurementType: "ActivePower", Measured: true, MinMeasured: -9, MaxMeasured: 9, FixedMax: 1, Provenance: SyntheticProvenance("accuracy-2"),
	})
	if err := h2.host.StartFixture(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !reflect.DeepEqual(h2.sdk.plans[0].Metadata, ctx.Metadata) {
		t.Fatalf("metadata %+v", h2.sdk.plans[0].Metadata)
	}
	if h2.sdk.plans[0].Metadata.ActivePower.Null {
		t.Fatal("known zero ActivePower became null")
	}
}

func TestAA09RetainedValueIsNotRepublishedAsCurrent(t *testing.T) {
	h := aa09NewHarness(t)
	aa09MustStart(t, h)
	h.projection.obs = aa09Observation("16000")
	if _, err := h.host.RefreshCurrent(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	h.projection.obs.Freshness = FreshnessStale
	update, err := h.host.RefreshCurrent()
	if err != nil || !update.Null {
		t.Fatalf("stale update %+v %v", update, err)
	}
	snapshot := h.host.Snapshot()
	if !snapshot.Published.Null || snapshot.LastKnownGood == nil || snapshot.LastKnownGood.Milliamperes != 16000 {
		t.Fatalf("snapshot %+v", snapshot)
	}
	boom := errors.New("synthetic read failure")
	h.projection.err = boom
	update, err = h.host.RefreshCurrent()
	if !errors.Is(err, ErrProjectionRead) || !errors.Is(err, boom) || !update.Null || update.WithheldReason != "read-failed" {
		t.Fatalf("read failure %+v %v", update, err)
	}
	if err := h.host.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	h.node = &aa09Node{trace: h.trace}
	aa09MustStart(t, h)
	snapshot = h.host.Snapshot()
	if !snapshot.Published.Null || snapshot.LastKnownGood == nil || snapshot.LastKnownGood.Generation != 1 {
		t.Fatalf("restart made a retained value current: %+v", snapshot)
	}
}

func TestAA09PublishFailureKeepsPublishedState(t *testing.T) {
	boom := errors.New("synthetic publish failure")
	h := aa09NewHarness(t)
	h.node.publishErr = boom
	aa09MustStart(t, h)
	h.projection.obs = aa09Observation("16000")
	if _, err := h.host.RefreshCurrent(); !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	snapshot := h.host.Snapshot()
	if !snapshot.Published.Null || snapshot.LastKnownGood != nil || !snapshot.Running {
		t.Fatalf("snapshot %+v", snapshot)
	}
}

func TestAA09RefreshBeforeStartReadsNothing(t *testing.T) {
	h := aa09NewHarness(t)
	if _, err := h.host.RefreshCurrent(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("got %v", err)
	}
	aa09AssertZeroPorts(t, h)
}

func TestAA09PlansAndContextsAreDetached(t *testing.T) {
	h := aa09NewHarness(t)
	ctx := aa09Context()
	if err := h.host.StartFixture(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	want := aa09Context().Metadata
	// The caller, the SDK and a Plan reader each hold separate storage.
	ctx.Metadata.Accuracy[0].MeasurementType = "changed-by-caller"
	ctx.Metadata.Features.Values[0] = "changed-by-caller"
	sdkPlan := h.sdk.plans[0]
	sdkPlan.Endpoints[2].Servers[0].Name = "changed-by-sdk"
	sdkPlan.Endpoints[0].Parts[0] = "changed-by-sdk"
	sdkPlan.Endpoints[1].DeviceTypes[0].Revision = 99
	sdkPlan.Metadata.Accuracy[0].MeasurementType = "changed-by-sdk"
	reader, _ := h.host.Plan()
	reader.Endpoints[2].Servers[1].Name = "changed-by-reader"
	reader.Metadata.Features.Values[0] = "changed-by-reader"
	plan, ok := h.host.Plan()
	if !ok || !reflect.DeepEqual(plan.Metadata, want) {
		t.Fatalf("metadata aliased: %+v", plan.Metadata)
	}
	if plan.Endpoints[2].Servers[0].Name != "Descriptor" || plan.Endpoints[2].Servers[1].Name != "Bridged Device Basic Information" ||
		plan.Endpoints[0].Parts[0] != "aggregator" || plan.Endpoints[1].DeviceTypes[0].Revision != 2 {
		t.Fatalf("endpoints aliased: %+v", plan.Endpoints)
	}
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyDetach, Sensor: aa09Identity()}); err != nil {
		t.Fatalf("detach: %v", err)
	}
	h.node.changes[0].Endpoint.Servers[0].Name = "changed-by-sdk"
	plan, _ = h.host.Plan()
	if plan.Endpoints[2].Servers[0].Name != "Descriptor" {
		t.Fatal("topology change aliased the plan")
	}
	h.projection.obs = aa09Observation("16000")
	if err := h.host.ChangeTopology(TopologyRequest{Op: TopologyAttach, Sensor: aa09Identity()}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if _, err := h.host.RefreshCurrent(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	snapshot := h.host.Snapshot()
	snapshot.LastKnownGood.Milliamperes = 1
	if h.host.Snapshot().LastKnownGood.Milliamperes != 16000 {
		t.Fatal("snapshot aliased the retained value")
	}
}
