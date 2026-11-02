//go:build gofabric

package sdkfabric_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Project-Helianthus/helianthus-matter-binding/adoption"
	"github.com/Project-Helianthus/helianthus-matter-binding/sdkfabric"
	"github.com/SukramJ/go-fabric/client/discovery"
	"github.com/SukramJ/go-fabric/cluster/spec"
	descriptordef "github.com/SukramJ/go-fabric/cluster/spec/descriptor"
	epmdef "github.com/SukramJ/go-fabric/cluster/spec/electricalpowermeasurement"
	ptdef "github.com/SukramJ/go-fabric/cluster/spec/powertopology"
	"github.com/SukramJ/go-fabric/controller/client"
	"github.com/SukramJ/go-fabric/fabrictest"
	"github.com/SukramJ/go-fabric/mdns"
	"github.com/SukramJ/go-fabric/node"
)

// siteLauncher is the only launcher the tests admit: every node it starts is
// a fabrictest device on the site's in-memory network and fake clock. It
// refuses a configuration that would let the SDK build its own advertiser if
// launched anywhere but the site, which replaces the advertiser on AddNode.
type siteLauncher struct {
	site *fabrictest.Site
	adv  mdns.Advertiser

	mu       sync.Mutex
	devices  []*fabrictest.Device
	launches int
	stops    int
}

var errUnsafeConfig = errors.New("node configuration leaves advertisement to the SDK")

func (l *siteLauncher) Launch(cfg node.Config) (sdkfabric.Launched, error) {
	l.mu.Lock()
	l.launches++
	l.mu.Unlock()
	if cfg.Advertiser == nil || cfg.Advertiser != l.adv || !cfg.Network.MDNS.Off || !cfg.Network.Reannounce.Off ||
		cfg.Network.TCP != (node.TCPConfig{}) || cfg.Client != nil || cfg.Observers.OnCommissioningComplete == nil {
		return nil, errUnsafeConfig
	}
	d := l.site.AddNode(cfg)
	l.mu.Lock()
	l.devices = append(l.devices, d)
	l.mu.Unlock()
	return &countedDevice{Device: d, l: l}, nil
}

func (l *siteLauncher) counts() (launches, stops int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.launches, l.stops
}

type countedDevice struct {
	*fabrictest.Device
	l *siteLauncher
}

func (d *countedDevice) Stop(ctx context.Context) error {
	d.l.mu.Lock()
	d.l.stops++
	d.l.mu.Unlock()
	return d.Device.Stop(ctx)
}

// logRecorder keeps the messages the site's hosts log.
type logRecorder struct {
	mu   sync.Mutex
	msgs []string
}

func (r *logRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (r *logRecorder) Handle(_ context.Context, rec slog.Record) error {
	r.mu.Lock()
	r.msgs = append(r.msgs, rec.Message)
	r.mu.Unlock()
	return nil
}
func (r *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *logRecorder) WithGroup(string) slog.Handler      { return r }

func (r *logRecorder) has(msg string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.msgs, msg)
}

type scriptedProjection struct {
	mu   sync.Mutex
	next adoption.CurrentObservation
}

func (p *scriptedProjection) set(o adoption.CurrentObservation) {
	p.mu.Lock()
	p.next = o
	p.mu.Unlock()
}

func (p *scriptedProjection) ReadCurrent(adoption.CurrentRequest) (adoption.CurrentObservation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.next, nil
}

type lifecycleLog struct {
	mu     sync.Mutex
	events []adoption.LifecycleEvent
}

func (l *lifecycleLog) Transition(e adoption.LifecycleEvent) {
	l.mu.Lock()
	l.events = append(l.events, e)
	l.mu.Unlock()
}

func available(c adoption.FixtureContext, milliamps string) adoption.CurrentObservation {
	return adoption.CurrentObservation{
		Fact: adoption.CurrentFactID, Sensor: c.Sensor, Availability: adoption.AvailabilityAvailable,
		Freshness: adoption.FreshnessFresh, Qualified: true, Confidence: adoption.EvidenceProven,
		Unit: adoption.CurrentUnit, Value: milliamps, NativeEvidence: "synthetic:native-" + milliamps,
	}
}

func unavailable(c adoption.FixtureContext) adoption.CurrentObservation {
	return adoption.CurrentObservation{
		Fact: adoption.CurrentFactID, Sensor: c.Sensor, Availability: adoption.AvailabilityUnavailable,
		Freshness: adoption.FreshnessUnknown, Confidence: adoption.EvidenceUnknown,
	}
}

func bounded(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func readCluster(ctx context.Context, t *testing.T, pair *fabrictest.Pair, ep uint16, cluster uint32) spec.AttributeValues {
	t.Helper()
	v, err := fabrictest.ReadCluster(ctx, pair, ep, cluster)
	if err != nil {
		t.Fatalf("read %d/0x%04X: %v", ep, cluster, err)
	}
	return v
}

func global(t *testing.T, v spec.AttributeValues, attr uint32) uint64 {
	t.Helper()
	n, ok := v.Value(attr)
	if !ok {
		t.Fatalf("global attribute 0x%04X is absent", attr)
	}
	return n.El.Uint
}

func descriptorOf(ctx context.Context, t *testing.T, pair *fabrictest.Pair, ep uint16) descriptordef.State {
	t.Helper()
	st, _ := descriptordef.StateFrom(readCluster(ctx, t, pair, ep, descriptordef.ClusterID))
	return st
}

// currentReport is one ActiveCurrent value a subscription report carried.
type currentReport struct {
	null    bool
	value   int64
	version uint32
}

func activeCurrent(t *testing.T, data fabrictest.AttributeData) currentReport {
	t.Helper()
	if data.DataVersion == 0 {
		t.Fatal("ActiveCurrent data carries no data version")
	}
	if data.Value.El.IsNull {
		return currentReport{null: true, version: data.DataVersion}
	}
	return currentReport{value: data.Value.El.Int, version: data.DataVersion}
}

// TestAA12CommissionReadSubscribe runs the AA09 host on the real go-fabric
// node through the site's AddNode and AddController route: discovery and PASE
// are observed not to complete commissioning, CommissioningComplete over CASE
// does, the window is closed through the SDK, and descriptor, Power Topology,
// Electrical Power Measurement and a subscribed ActiveCurrent are read over
// the in-memory wire with independently expected values and data versions.
func TestAA12CommissionReadSubscribe(t *testing.T) {
	logs := &logRecorder{}
	site := fabrictest.NewSite(t, fabrictest.Options{Logger: slog.New(logs)})
	if site.Net() == nil || site.Loopback() {
		t.Fatal("the site is not on the in-memory network")
	}
	adv := &spyAdvertiser{}
	launcher := &siteLauncher{site: site, adv: adv}
	port, err := sdkfabric.NewPort(fixtureConfig(launcher, adv))
	if err != nil {
		t.Fatal(err)
	}
	projection := &scriptedProjection{}
	lifecycle := &lifecycleLog{}
	host, err := adoption.NewReferenceHost(adoption.Ports{SDK: port, Projection: projection, Lifecycle: lifecycle})
	if err != nil {
		t.Fatal(err)
	}
	fixture := fixtureContext()
	if err := host.StartFixture(fixture); err != nil {
		t.Fatalf("StartFixture: %v", err)
	}
	t.Cleanup(func() { _ = host.Stop() })
	ctx := bounded(t)

	if got := port.Stats(); got != (sdkfabric.Stats{Plans: 1, Launches: 1}) {
		t.Fatalf("stats after start %+v", got)
	}
	if launches, _ := launcher.counts(); launches != 1 || len(port.Nodes()) != 1 {
		t.Fatalf("launches %d, nodes %d", launches, len(port.Nodes()))
	}
	fn := port.Nodes()[0]
	dev := launcher.devices[0]
	if ip := dev.Addr().IP; !ip.IsLinkLocalUnicast() || ip[0] != 0xfe || ip[1] != 0x80 {
		t.Fatalf("device address %v is not a site address", dev.Addr())
	}
	if w, err := fn.Window(); err != nil || !w.Open || w.Kind != node.WindowOwn {
		t.Fatalf("uncommissioned window %+v, %v", w, err)
	}
	notCommissioned := func(stage string) {
		t.Helper()
		if c := fn.Completions(); len(c) != 0 {
			t.Fatalf("%s: completions %v", stage, c)
		}
		if err := fn.RequireCommissioned(ctx); !errors.Is(err, sdkfabric.ErrNotCommissioned) {
			t.Fatalf("%s: RequireCommissioned %v", stage, err)
		}
		fabrics, err := dev.Node().FabricInfos(ctx)
		if err != nil || len(fabrics) != 0 {
			t.Fatalf("%s: fabrics %v, %v", stage, fabrics, err)
		}
	}
	notCommissioned("after start")

	// Without a fabric the SDK reopens the node's own window after a close: the
	// close call returns, yet the window stays open, so closing has not completed.
	creds := node.WindowCredentials{
		Discriminator: 2222, Passcode: 34567890, PBKDFIterations: fabrictest.DefaultPBKDFIterations,
		Salt: []byte("SPAKE2P Key Salt"), Timeout: 3 * time.Minute,
	}
	if err := fn.OpenWindow(ctx, creds); err != nil {
		t.Fatalf("OpenWindow before commissioning: %v", err)
	}
	if err := fn.CloseWindow(ctx); !errors.Is(err, sdkfabric.ErrWindowOpen) {
		t.Fatalf("CloseWindow without a fabric: %v", err)
	}
	if w, _ := fn.Window(); !w.Open || w.Kind != node.WindowOwn || w.Discriminator != dev.Discriminator() {
		t.Fatalf("window after close without a fabric %+v", w)
	}
	notCommissioned("after an uncommissioned window cycle")

	ctl := site.AddController(fabrictest.ControllerOptions{})
	found, err := ctl.Discover(ctx, discovery.ByLongDiscriminator(dev.Discriminator()), time.Minute)
	if err != nil || len(found) != 1 {
		t.Fatalf("discover: %d found, %v", len(found), err)
	}
	notCommissioned("after discovery")

	pase, err := ctl.PASE(ctx, dev)
	if err != nil {
		t.Fatalf("PASE: %v", err)
	}
	if pase.Commissioned() {
		t.Fatal("a PASE pair reports commissioned")
	}
	notCommissioned("after PASE")

	pair, err := ctl.Commission(ctx, dev, fabrictest.CommissionOptions{})
	if err != nil {
		t.Fatalf("commission: %v", err)
	}
	if !pair.Commissioned() {
		t.Fatal("the commissioned pair holds no CASE peer")
	}
	completions := fn.Completions()
	fabrics, err := dev.Node().FabricInfos(ctx)
	if err != nil || len(completions) != 1 || len(fabrics) != 1 || fabrics[0].FabricIndex != completions[0] {
		t.Fatalf("completions %v, fabrics %v, %v", completions, fabrics, err)
	}
	if fabrics[0].FabricID != fabrictest.DefaultFabricID || fabrics[0].NodeID != pair.Address.NodeID {
		t.Fatalf("fabric %+v, peer %+v", fabrics[0], pair.Address)
	}
	if err := fn.RequireCommissioned(ctx); err != nil {
		t.Fatalf("RequireCommissioned after CommissioningComplete: %v", err)
	}
	if w, _ := fn.Window(); w.Open {
		t.Fatalf("CommissioningComplete left the window open: %+v", w)
	}

	// An explicit window: opening is not commissioning, a failed close keeps it
	// open, and only the SDK's report of no open window counts as closed.
	if err := fn.OpenWindow(ctx, creds); err != nil {
		t.Fatalf("OpenWindow: %v", err)
	}
	if w, _ := fn.Window(); !w.Open || w.Kind != node.WindowBasic || w.Discriminator != 2222 {
		t.Fatalf("opened window %+v", w)
	}
	if len(fn.Completions()) != 1 {
		t.Fatalf("opening a window changed completions: %v", fn.Completions())
	}
	if err := fn.RequireCommissioned(ctx); !errors.Is(err, sdkfabric.ErrWindowOpen) {
		t.Fatalf("RequireCommissioned with an open window: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := fn.CloseWindow(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("CloseWindow with a cancelled context: %v", err)
	}
	if w, _ := fn.Window(); !w.Open {
		t.Fatal("a failed close closed the window")
	}
	transitions := len(fn.WindowTransitions())
	if err := fn.CloseWindow(ctx); err != nil {
		t.Fatalf("CloseWindow: %v", err)
	}
	if w, _ := fn.Window(); w.Open {
		t.Fatalf("closed window %+v", w)
	}
	if after := fn.WindowTransitions(); len(after) <= transitions || after[len(after)-1].Open {
		t.Fatalf("no closing transition was reported: %+v", after[transitions:])
	}
	if err := fn.RequireCommissioned(ctx); err != nil {
		t.Fatalf("RequireCommissioned after close: %v", err)
	}

	// Descriptor composition read over CASE.
	sensorEP, ok := fn.Sensor().EndpointNumber()
	if !ok || sensorEP == 0 {
		t.Fatalf("sensor endpoint %d, %v", sensorEP, ok)
	}
	root := descriptorOf(ctx, t, pair, 0)
	if !slices.Contains(root.PartsList, sensorEP) {
		t.Fatalf("root PartsList %v lacks the sensor %d", root.PartsList, sensorEP)
	}
	var aggEP uint16
	for _, ep := range root.PartsList {
		if ep == sensorEP {
			continue
		}
		if d := descriptorOf(ctx, t, pair, ep); slices.Equal(d.DeviceTypeList, []descriptordef.DeviceTypeStruct{{DeviceType: 0x000E, Revision: 2}}) {
			aggEP = ep
			if !slices.Equal(d.PartsList, []uint16{sensorEP}) {
				t.Fatalf("Aggregator PartsList %v", d.PartsList)
			}
		}
	}
	if aggEP == 0 {
		t.Fatal("no Aggregator endpoint of revision 2")
	}
	sensor := descriptorOf(ctx, t, pair, sensorEP)
	wantTypes := []descriptordef.DeviceTypeStruct{{DeviceType: 0x0013, Revision: 3}, {DeviceType: 0x0510, Revision: 1}}
	gotTypes := slices.Clone(sensor.DeviceTypeList)
	slices.SortFunc(gotTypes, func(a, b descriptordef.DeviceTypeStruct) int { return int(a.DeviceType) - int(b.DeviceType) })
	if !slices.Equal(gotTypes, wantTypes) {
		t.Fatalf("sensor DeviceTypeList %+v, want %+v", sensor.DeviceTypeList, wantTypes)
	}
	for _, id := range []uint32{0x001D, 0x0039, 0x009C, 0x0090} {
		if !slices.Contains(sensor.ServerList, id) {
			t.Errorf("sensor ServerList %v lacks 0x%04X", sensor.ServerList, id)
		}
	}
	if slices.Contains(sensor.ServerList, 0x0091) {
		t.Errorf("sensor serves an undeclared Electrical Energy Measurement: %v", sensor.ServerList)
	}

	pt := readCluster(ctx, t, pair, sensorEP, ptdef.ClusterID)
	if rev, features := global(t, pt, spec.AttrClusterRevision), global(t, pt, spec.AttrFeatureMap); rev != 1 || features != 1 {
		t.Fatalf("Power Topology revision %d, features %#x", rev, features)
	}
	epmValues := readCluster(ctx, t, pair, sensorEP, epmdef.ClusterID)
	if rev, features := global(t, epmValues, spec.AttrClusterRevision), global(t, epmValues, spec.AttrFeatureMap); rev != 3 || features != 2 {
		t.Fatalf("Electrical Power Measurement revision %d, features %#x", rev, features)
	}
	epm, _ := epmdef.StateFrom(epmValues)
	fixed := uint64(100)
	wantAccuracy := []epmdef.MeasurementAccuracyStruct{{
		MeasurementType: 2, Measured: true, MinMeasuredValue: 0, MaxMeasuredValue: 32000,
		AccuracyRanges: []epmdef.MeasurementAccuracyRangeStruct{{RangeMin: 0, RangeMax: 32000, FixedMax: &fixed}},
	}}
	if epm.PowerMode != 2 || epm.NumberOfMeasurementTypes != 1 || epm.ActivePower != nil || epm.ActiveCurrent != nil {
		t.Fatalf("Electrical Power Measurement state %+v", epm)
	}
	if len(epm.Accuracy) != 1 || len(epm.Accuracy[0].AccuracyRanges) != 1 ||
		epm.Accuracy[0].MeasurementType != wantAccuracy[0].MeasurementType || !epm.Accuracy[0].Measured ||
		epm.Accuracy[0].MaxMeasuredValue != 32000 || epm.Accuracy[0].AccuracyRanges[0].RangeMax != 32000 ||
		epm.Accuracy[0].AccuracyRanges[0].FixedMax == nil || *epm.Accuracy[0].AccuracyRanges[0].FixedMax != fixed {
		t.Fatalf("Accuracy %+v, want %+v", epm.Accuracy, wantAccuracy)
	}

	// Subscribed ActiveCurrent: each accepted change must reach the controller
	// as a report with the next data version, and a read must agree.
	currentPath := client.AttributeAt(sensorEP, epmdef.ClusterID, epmdef.AttrActiveCurrent)
	var repMu sync.Mutex
	var reports []currentReport
	ceiling := time.Minute
	sub, err := fabrictest.Subscribe(ctx, pair, fabrictest.SubscribeRequest{
		Attributes:         []fabrictest.AttributePath{currentPath},
		MaxIntervalCeiling: &ceiling,
		Updated: func(r *fabrictest.Report) {
			repMu.Lock()
			defer repMu.Unlock()
			for _, a := range r.Attributes {
				if a.Path.Attribute == epmdef.AttrActiveCurrent {
					reports = append(reports, activeCurrent(t, a))
				}
			}
		},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	t.Cleanup(sub.Close)
	snapshot := func() []currentReport {
		repMu.Lock()
		defer repMu.Unlock()
		return slices.Clone(reports)
	}
	primed := snapshot()
	if len(primed) != 1 || !primed[0].null {
		t.Fatalf("priming reports %+v", primed)
	}
	base := primed[0].version
	readCurrent := func() currentReport {
		t.Helper()
		data, err := fabrictest.Read(ctx, pair, currentPath)
		if err != nil {
			t.Fatalf("read ActiveCurrent: %v", err)
		}
		return activeCurrent(t, data)
	}
	if got := readCurrent(); got != (currentReport{null: true, version: base}) {
		t.Fatalf("read before change %+v, base %d", got, base)
	}

	// A change addressed to another generation or sensor reaches no server.
	if err := fn.PublishCurrent(adoption.CurrentUpdate{Generation: 2, SensorKey: adoption.StableKey(fixture.Sensor), Milliamperes: 1}); !errors.Is(err, sdkfabric.ErrChange) {
		t.Fatalf("wrong generation: %v", err)
	}
	if err := fn.PublishCurrent(adoption.CurrentUpdate{Generation: 1, SensorKey: "other", Milliamperes: 1}); !errors.Is(err, sdkfabric.ErrChange) {
		t.Fatalf("wrong sensor: %v", err)
	}

	steps := []struct {
		observation adoption.CurrentObservation
		want        currentReport
	}{
		{available(fixture, "16000"), currentReport{value: 16000, version: base + 1}},
		{unavailable(fixture), currentReport{null: true, version: base + 2}},
		{available(fixture, "0"), currentReport{value: 0, version: base + 3}},
	}
	for i, step := range steps {
		projection.set(step.observation)
		if _, err := host.RefreshCurrent(); err != nil {
			t.Fatalf("step %d: RefreshCurrent: %v", i, err)
		}
		if err := site.Eventually(ctx, "ActiveCurrent report", func() bool { return len(snapshot()) >= i+2 }); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if got := snapshot(); len(got) != i+2 || got[i+1] != step.want {
			t.Fatalf("step %d: reports %+v, want %+v last", i, got, step.want)
		}
		if got := readCurrent(); got != step.want {
			t.Fatalf("step %d: read %+v, want %+v", i, got, step.want)
		}
	}
	if snap := host.Snapshot(); snap.LastKnownGood == nil || snap.LastKnownGood.Milliamperes != 0 || snap.Published.Null {
		t.Fatalf("host snapshot %+v", snap)
	}

	// Detach and reattach keep the reserved endpoint number.
	if err := host.ChangeTopology(adoption.TopologyRequest{Op: adoption.TopologyDetach, Sensor: fixture.Sensor}); err != nil {
		t.Fatalf("detach: %v", err)
	}
	if parts := descriptorOf(ctx, t, pair, aggEP).PartsList; slices.Contains(parts, sensorEP) {
		t.Fatalf("detached sensor still in Aggregator PartsList %v", parts)
	}
	if err := host.ChangeTopology(adoption.TopologyRequest{Op: adoption.TopologyAttach, Sensor: fixture.Sensor}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if parts := descriptorOf(ctx, t, pair, aggEP).PartsList; !slices.Equal(parts, []uint16{sensorEP}) {
		t.Fatalf("reattached Aggregator PartsList %v, want [%d]", parts, sensorEP)
	}

	// Stop releases the real node exactly once; the SDK then refuses a start.
	sub.Close()
	running := dev.Node()
	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := host.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if err := running.Start(ctx); !errors.Is(err, node.ErrStopped) {
		t.Fatalf("start of the stopped node: %v", err)
	}
	if got := port.Stats(); got != (sdkfabric.Stats{Plans: 1, Launches: 1, Stops: 1}) {
		t.Fatalf("stats after stop %+v", got)
	}
	if launches, stops := launcher.counts(); launches != 1 || stops != 1 {
		t.Fatalf("launcher launches %d, stops %d", launches, stops)
	}
	if got := adv.counts(); got != (advertiserCounts{}) {
		t.Fatalf("the injected advertiser was used on the site: %+v", got)
	}
	if site.Net().Sent() == 0 {
		t.Fatal("no datagram crossed the in-memory network")
	}
	t.Logf("in-memory datagrams %d; group-send socket bind refused by the environment: %v",
		site.Net().Sent(), logs.has("matter.groups.send_socket"))
}

type advertiserCounts struct{ publish, withdraw, closed int }

// spyAdvertiser stands for any real network advertisement: it must never be
// called on the site, and a nil advertiser must never reach the SDK.
type spyAdvertiser struct {
	mu sync.Mutex
	n  advertiserCounts
}

func (a *spyAdvertiser) counts() advertiserCounts {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.n
}

func (a *spyAdvertiser) Publish(context.Context, mdns.Service) error {
	a.mu.Lock()
	a.n.publish++
	a.mu.Unlock()
	return nil
}

func (a *spyAdvertiser) Withdraw(context.Context, string, string) error {
	a.mu.Lock()
	a.n.withdraw++
	a.mu.Unlock()
	return nil
}

func (a *spyAdvertiser) Active() []mdns.Service { return nil }

func (a *spyAdvertiser) Close() error {
	a.mu.Lock()
	a.n.closed++
	a.mu.Unlock()
	return nil
}

// spyLauncher stands for a production node factory: it must never be called.
type spyLauncher struct {
	mu    sync.Mutex
	calls int
}

func (l *spyLauncher) Launch(node.Config) (sdkfabric.Launched, error) {
	l.mu.Lock()
	l.calls++
	l.mu.Unlock()
	return nil, errors.New("production node factory reached")
}

func (l *spyLauncher) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

func fixtureConfig(l sdkfabric.Launcher, adv mdns.Advertiser) sdkfabric.Config {
	return sdkfabric.Config{
		Origin:     adoption.OriginSyntheticFixture,
		Launcher:   l,
		Advertiser: adv,
		Transport:  sdkfabric.TransportInMemory,
		Listen:     "[::1]:0",
		Timeout:    time.Minute,
		Commissioning: node.Commissioning{
			Passcode: fabrictest.DefaultPasscode, Discriminator: 1357,
			PBKDFIterations: fabrictest.DefaultPBKDFIterations, Salt: []byte("SPAKE2P Key Salt"),
		},
		ProductDescription: node.ProductDescription{VendorID: 0xFFF1, ProductID: 0x8001},
		BasicInformation: node.BasicInformation{
			VendorName: "synthetic", ProductName: "reference bridge", NodeLabel: "reference",
			HardwareVersion: 1, HardwareVersionString: "1", SoftwareVersion: 1, SoftwareVersionString: "1",
		},
	}
}

func fixtureContext() adoption.FixtureContext {
	p := adoption.SyntheticProvenance
	return adoption.FixtureContext{
		Origin:   adoption.OriginSyntheticFixture,
		Manifest: "synthetic:aa12-reference",
		Target: adoption.TargetDeclaration{
			DataModel: "1.6.1", AggregatorRevision: 2, BridgedNodeRevision: 3, BridgedBasicRevision: 6,
			ElectricalSensorRevision: 1, PowerTopologyRevision: 1, PowerMeasurementRevision: 3,
		},
		Sensor: adoption.Identity{
			Node: "node-a", Owner: "owner-a", Profile: "profile-a", Capability: "electrical.current",
			Composition: "composition-a", Binding: "binding-a", Slot: 1,
		},
		Metadata: adoption.SyntheticMetadata{
			PowerMode:        adoption.PowerModeFact{Value: "AC", Provenance: p("power-mode")},
			MeasurementTypes: adoption.CountFact{Value: 1, Provenance: p("measurement-types")},
			Accuracy: []adoption.AccuracyFact{{
				MeasurementType: "ActiveCurrent", Measured: true, MinMeasured: 0, MaxMeasured: 32000, FixedMax: 100,
				Provenance: p("accuracy"),
			}},
			ActivePower: adoption.ActivePowerFact{Declared: true, Null: true, Provenance: p("active-power")},
			Features:    adoption.FeatureFact{Values: []string{"AC"}, Provenance: p("features")},
			Topology:    adoption.TopologyFact{Mode: "NODE", Provenance: p("topology")},
		},
	}
}

// capturePort records the plan the real host builds and constructs nothing.
type capturePort struct{ plan *adoption.NodePlan }

func (c *capturePort) New(plan adoption.NodePlan) (adoption.SDKNode, error) {
	c.plan = &plan
	return nil, errors.New("captured")
}

func hostPlan(t *testing.T) adoption.NodePlan {
	t.Helper()
	capture := &capturePort{}
	host, err := adoption.NewReferenceHost(adoption.Ports{SDK: capture, Projection: &scriptedProjection{}, Lifecycle: &lifecycleLog{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := host.StartFixture(fixtureContext()); err == nil || capture.plan == nil {
		t.Fatalf("no plan captured: %v", err)
	}
	return *capture.plan
}

// TestAA12ConfigRefusals varies each configuration field alone, all others
// valid, and requires refusal before any launcher or advertiser call.
func TestAA12ConfigRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*sdkfabric.Config)
		want   error
	}{
		{"production origin", func(c *sdkfabric.Config) { c.Origin = adoption.OriginProduction }, sdkfabric.ErrNotSynthetic},
		{"zero origin", func(c *sdkfabric.Config) { c.Origin = "" }, sdkfabric.ErrNotSynthetic},
		{"nil launcher", func(c *sdkfabric.Config) { c.Launcher = nil }, sdkfabric.ErrPortMissing},
		{"nil advertiser", func(c *sdkfabric.Config) { c.Advertiser = nil }, sdkfabric.ErrAdvertiser},
		{"zero transport", func(c *sdkfabric.Config) { c.Transport = "" }, sdkfabric.ErrTransport},
		{"other transport", func(c *sdkfabric.Config) { c.Transport = "udp" }, sdkfabric.ErrTransport},
		{"empty listen", func(c *sdkfabric.Config) { c.Listen = "" }, sdkfabric.ErrTransport},
		{"zero timeout", func(c *sdkfabric.Config) { c.Timeout = 0 }, sdkfabric.ErrTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			factory := &spyLauncher{}
			adv := &spyAdvertiser{}
			cfg := fixtureConfig(factory, adv)
			tc.mutate(&cfg)
			if port, err := sdkfabric.NewPort(cfg); !errors.Is(err, tc.want) || port != nil {
				t.Fatalf("NewPort: %v, want %v", err, tc.want)
			}
			if factory.count() != 0 || adv.counts() != (advertiserCounts{}) {
				t.Fatalf("launcher %d, advertiser %+v", factory.count(), adv.counts())
			}
		})
	}
	// Positive control: the unmutated configuration is accepted.
	if _, err := sdkfabric.NewPort(fixtureConfig(&spyLauncher{}, &spyAdvertiser{})); err != nil {
		t.Fatalf("valid configuration refused: %v", err)
	}
}

// TestAA12PlanRefusals varies each bound plan field alone and requires
// refusal at New, before the node exists; Start of the unmutated plan is the
// only call that reaches the launcher.
func TestAA12PlanRefusals(t *testing.T) {
	sensor, epm := 2, 3
	cases := []struct {
		name   string
		mutate func(*adoption.NodePlan)
		want   error
	}{
		{"data model", func(p *adoption.NodePlan) { p.DataModel = "1.7" }, sdkfabric.ErrPlan},
		{"generation", func(p *adoption.NodePlan) { p.Generation = 0 }, sdkfabric.ErrPlan},
		{"manifest", func(p *adoption.NodePlan) { p.Manifest = "fixture:aa12" }, sdkfabric.ErrPlan},
		{"sensor key", func(p *adoption.NodePlan) { p.SensorKey = "helianthus.reference/other" }, sdkfabric.ErrPlan},
		{"endpoint count", func(p *adoption.NodePlan) { p.Endpoints = p.Endpoints[:2] }, sdkfabric.ErrPlan},
		{"aggregator revision", func(p *adoption.NodePlan) { p.Endpoints[1].DeviceTypes[0].Revision = 1 }, sdkfabric.ErrRevision},
		{"bridged node revision", func(p *adoption.NodePlan) { p.Endpoints[sensor].DeviceTypes[0].Revision = 2 }, sdkfabric.ErrRevision},
		{"electrical sensor revision 2", func(p *adoption.NodePlan) { p.Endpoints[sensor].DeviceTypes[1].Revision = 2 }, sdkfabric.ErrRevision},
		{"electrical sensor id", func(p *adoption.NodePlan) { p.Endpoints[sensor].DeviceTypes[1].ID = 0x0511 }, sdkfabric.ErrRevision},
		{"electrical sensor evidence", func(p *adoption.NodePlan) {
			p.Endpoints[sensor].DeviceTypes[1].RevisionEvidence = adoption.EvidenceHypothesis
		}, sdkfabric.ErrRevision},
		{"bridged basic revision", func(p *adoption.NodePlan) { p.Endpoints[sensor].Servers[1].Revision = 5 }, sdkfabric.ErrRevision},
		{"power topology revision 2", func(p *adoption.NodePlan) { p.Endpoints[sensor].Servers[2].Revision = 2 }, sdkfabric.ErrRevision},
		{"power measurement revision", func(p *adoption.NodePlan) { p.Endpoints[sensor].Servers[epm].Revision = 2 }, sdkfabric.ErrRevision},
		{"power mode", func(p *adoption.NodePlan) { p.Metadata.PowerMode.Value = "DC" }, sdkfabric.ErrMetadata},
		{"topology mode", func(p *adoption.NodePlan) { p.Metadata.Topology.Mode = "TREE" }, sdkfabric.ErrMetadata},
		{"features", func(p *adoption.NodePlan) { p.Metadata.Features.Values = []string{"DC"} }, sdkfabric.ErrMetadata},
		{"measurement type", func(p *adoption.NodePlan) { p.Metadata.Accuracy[0].MeasurementType = "Unknown" }, sdkfabric.ErrMetadata},
		{"accuracy count", func(p *adoption.NodePlan) { p.Metadata.MeasurementTypes.Value = 2 }, sdkfabric.ErrMetadata},
		{"active power", func(p *adoption.NodePlan) { p.Metadata.ActivePower.Declared = false }, sdkfabric.ErrMetadata},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			factory := &spyLauncher{}
			port, err := sdkfabric.NewPort(fixtureConfig(factory, &spyAdvertiser{}))
			if err != nil {
				t.Fatal(err)
			}
			plan := hostPlan(t)
			tc.mutate(&plan)
			if n, err := port.New(plan); !errors.Is(err, tc.want) || n != nil {
				t.Fatalf("New: %v, want %v", err, tc.want)
			}
			if port.Stats() != (sdkfabric.Stats{}) || factory.count() != 0 || len(port.Nodes()) != 0 {
				t.Fatalf("stats %+v, launcher %d", port.Stats(), factory.count())
			}
		})
	}
	// Positive control: the unmutated plan builds a node and only Start
	// dispatches, here to the production spy, whose refusal is a start failure.
	factory := &spyLauncher{}
	port, err := sdkfabric.NewPort(fixtureConfig(factory, &spyAdvertiser{}))
	if err != nil {
		t.Fatal(err)
	}
	n, err := port.New(hostPlan(t))
	if err != nil || n == nil || factory.count() != 0 {
		t.Fatalf("valid plan: %v, launcher %d", err, factory.count())
	}
	if err := n.Start(); err == nil || factory.count() != 1 {
		t.Fatalf("start through the spy: %v, launcher %d", err, factory.count())
	}
	if got := port.Stats(); got != (sdkfabric.Stats{Plans: 1, Launches: 1, LaunchErrors: 1}) {
		t.Fatalf("stats %+v", got)
	}
}

// TestAA12HostAdmission requires the host's fixture admission to gate the
// port: a production context reaches neither New nor any launcher.
func TestAA12HostAdmission(t *testing.T) {
	factory := &spyLauncher{}
	port, err := sdkfabric.NewPort(fixtureConfig(factory, &spyAdvertiser{}))
	if err != nil {
		t.Fatal(err)
	}
	host, err := adoption.NewReferenceHost(adoption.Ports{SDK: port, Projection: &scriptedProjection{}, Lifecycle: &lifecycleLog{}})
	if err != nil {
		t.Fatal(err)
	}
	c := fixtureContext()
	c.Origin = adoption.OriginProduction
	if err := host.StartFixture(c); !errors.Is(err, adoption.ErrContextNotSynthetic) {
		t.Fatalf("production context: %v", err)
	}
	if port.Stats() != (sdkfabric.Stats{}) || factory.count() != 0 {
		t.Fatalf("stats %+v, launcher %d", port.Stats(), factory.count())
	}
}
