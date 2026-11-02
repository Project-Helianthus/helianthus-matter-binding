//go:build gofabric

package sdkfabric_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Project-Helianthus/helianthus-matter-binding/adoption"
	"github.com/Project-Helianthus/helianthus-matter-binding/qualification/commissioning"
	"github.com/Project-Helianthus/helianthus-matter-binding/qualification/reportharness"
	"github.com/Project-Helianthus/helianthus-matter-binding/sdkfabric"
	"github.com/SukramJ/go-fabric/client/discovery"
	epmdef "github.com/SukramJ/go-fabric/cluster/spec/electricalpowermeasurement"
	"github.com/SukramJ/go-fabric/controller/client"
	"github.com/SukramJ/go-fabric/fabrictest"
	"github.com/SukramJ/go-fabric/node"
)

// aa30Run is one reference host on the in-memory site, with the real node it
// launched through the site's AddNode route.
type aa30Run struct {
	site     *fabrictest.Site
	host     *adoption.ReferenceHost
	port     *sdkfabric.Port
	launcher *siteLauncher
	fn       *sdkfabric.Node
	dev      *fabrictest.Device
	ctx      context.Context

	mu   sync.Mutex
	last sdkfabric.ControllerReceipt
}

func aa30Start(t *testing.T) *aa30Run {
	t.Helper()
	site := fabrictest.NewSite(t, fabrictest.Options{})
	if site.Net() == nil || site.Loopback() {
		t.Fatal("the site is not on the in-memory network")
	}
	adv := &spyAdvertiser{}
	launcher := &siteLauncher{site: site, adv: adv}
	port, err := sdkfabric.NewPort(fixtureConfig(launcher, adv))
	if err != nil {
		t.Fatal(err)
	}
	host, err := adoption.NewReferenceHost(adoption.Ports{SDK: port, Projection: &scriptedProjection{}, Lifecycle: &lifecycleLog{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := host.StartFixture(fixtureContext()); err != nil {
		t.Fatalf("StartFixture: %v", err)
	}
	t.Cleanup(func() { _ = host.Stop() })
	if len(port.Nodes()) != 1 || len(launcher.devices) != 1 {
		t.Fatalf("nodes %d, devices %d", len(port.Nodes()), len(launcher.devices))
	}
	return &aa30Run{site: site, host: host, port: port, launcher: launcher, fn: port.Nodes()[0], dev: launcher.devices[0], ctx: bounded(t)}
}

// commission runs the real CASE commissioning from a fixture controller.
func (r *aa30Run) commission(t *testing.T, i *aa30Initiator) *fabrictest.Pair {
	t.Helper()
	pair, err := i.commission(r.ctx)
	if err != nil || !pair.Commissioned() {
		t.Fatalf("commission: %v", err)
	}
	return pair
}

// aa30Initiator is one fixture controller on the site behind the
// commissioning route; the same controller later commissions the node.
type aa30Initiator struct {
	pase       func(context.Context) error
	discover   func(context.Context) (int, error)
	commission func(context.Context) (*fabrictest.Pair, error)

	mu               sync.Mutex
	pases, discovers int
}

func (r *aa30Run) initiator() *aa30Initiator {
	ctl, dev := r.site.AddController(fabrictest.ControllerOptions{}), r.dev
	return &aa30Initiator{
		pase: func(ctx context.Context) error {
			_, err := ctl.PASE(ctx, dev)
			return err
		},
		discover: func(ctx context.Context) (int, error) {
			found, err := ctl.Discover(ctx, discovery.ByLongDiscriminator(dev.Discriminator()), time.Minute)
			return len(found), err
		},
		commission: func(ctx context.Context) (*fabrictest.Pair, error) {
			return ctl.Commission(ctx, dev, fabrictest.CommissionOptions{})
		},
	}
}

func (i *aa30Initiator) PASE(ctx context.Context) error {
	i.mu.Lock()
	i.pases++
	i.mu.Unlock()
	return i.pase(ctx)
}

func (i *aa30Initiator) Discover(ctx context.Context) (int, error) {
	i.mu.Lock()
	i.discovers++
	i.mu.Unlock()
	return i.discover(ctx)
}

func (i *aa30Initiator) counts() (pases, discovers int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.pases, i.discovers
}

func aa30Creds(dev *fabrictest.Device) node.WindowCredentials {
	return node.WindowCredentials{
		Discriminator: dev.Discriminator(), Passcode: fabrictest.DefaultPasscode, PBKDFIterations: fabrictest.DefaultPBKDFIterations,
		Salt: []byte("SPAKE2P Key Salt"), Timeout: 3 * time.Minute,
	}
}

// drive performs the staged operation and hands its completion to the runner.
func drive(t *testing.T, route *sdkfabric.CommissioningRoute, runner *commissioning.Runner, want string) sdkfabric.CommissioningCompletion {
	t.Helper()
	c, ok := route.Drive()
	if !ok {
		t.Fatal("no staged operation")
	}
	if ev := runner.Complete(c.Identity, c.Success, c.Records); ev.Outcome != want {
		t.Fatalf("completion %+v gave %+v, want %s", c, ev, want)
	}
	return c
}

// TestAA30CommissioningRunner drives the existing commissioning runner over the
// real node: PASE and discovery through the fixture controller, the window
// through the SDK, and commissioned only after a real CommissioningComplete.
// Completion identity axes generation, token and node are each varied alone
// (catches: a runner or route that drops one from the fence), duplicates and
// a held completion from an earlier generation are ignored across stop and
// restart (catches: completion by operation rather than identity).
func TestAA30CommissioningRunner(t *testing.T) {
	r := aa30Start(t)
	initiator := r.initiator()
	if _, err := sdkfabric.NewCommissioningRoute(r.fn, nil, aa30Creds(r.dev)); !errors.Is(err, sdkfabric.ErrInitiator) {
		t.Fatalf("nil initiator: %v", err)
	}
	route, err := sdkfabric.NewCommissioningRoute(r.fn, initiator, aa30Creds(r.dev))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := commissioning.New(route, "node-a", commissioning.Store{})
	if err != nil {
		t.Fatal(err)
	}
	if ev := runner.Start(); ev.Outcome != "accepted" {
		t.Fatalf("start %+v", ev)
	}

	// PASE is only staged by the port call: nothing reaches the SDK until Drive.
	ev := runner.PASE()
	if pases, _ := initiator.counts(); ev.Outcome != "pending" || pases != 0 || route.Stats() != (sdkfabric.CommissioningStats{Staged: 1}) {
		t.Fatalf("PASE %+v, initiator %d, stats %+v", ev, pases, route.Stats())
	}
	route.OpenWindow(commissioning.Request{Identity: ev.Identity, Open: true})
	if got := route.Stats(); got.Refused != 1 || got.Staged != 1 {
		t.Fatalf("a second staged operation was kept: %+v", got)
	}
	c, ok := route.Drive()
	if pases, _ := initiator.counts(); !ok || !c.Success || pases != 1 {
		t.Fatalf("PASE drive %+v, %v, initiator %d", c, ok, pases)
	}
	for name, id := range map[string]commissioning.Identity{
		"generation": {Generation: c.Identity.Generation + 1, Token: c.Identity.Token, Node: c.Identity.Node},
		"token":      {Generation: c.Identity.Generation, Token: c.Identity.Token + 1, Node: c.Identity.Node},
		"node":       {Generation: c.Identity.Generation, Token: c.Identity.Token, Node: "node-b"},
	} {
		if ev := runner.Complete(id, true, nil); ev.Outcome != "ignored" {
			t.Fatalf("%s varied: %+v", name, ev)
		}
	}
	first := c
	if ev := runner.Complete(c.Identity, c.Success, c.Records); ev.Outcome != "succeeded" || ev.State != "paired" {
		t.Fatalf("PASE completion %+v", ev)
	}
	if ev := runner.Complete(c.Identity, c.Success, c.Records); ev.Outcome != "ignored" {
		t.Fatalf("duplicate completion %+v", ev)
	}
	if len(r.fn.Completions()) != 0 {
		t.Fatalf("PASE completed commissioning: %v", r.fn.Completions())
	}

	if ev := runner.OpenWindow(true); ev.Outcome != "pending" {
		t.Fatalf("open %+v", ev)
	}
	drive(t, route, runner, "succeeded")
	if w, err := r.fn.Window(); err != nil || !w.Open {
		t.Fatalf("window after open %+v, %v", w, err)
	}
	if ev := runner.Discover(); ev.Outcome != "pending" {
		t.Fatalf("discover %+v", ev)
	}
	if c := drive(t, route, runner, "succeeded"); len(c.Records) != 1 {
		t.Fatalf("discovery %+v", c)
	}
	if _, discovers := initiator.counts(); discovers != 1 {
		t.Fatalf("initiator discoveries %d", discovers)
	}

	// The real commissioning is the controller's act; the runner records it
	// only when the SDK reports the window closed on the completed fabric.
	r.commission(t, initiator)
	if ev := runner.OpenWindow(false); ev.Outcome != "pending" {
		t.Fatalf("close %+v", ev)
	}
	drive(t, route, runner, "succeeded")
	if got := runner.Snapshot().Commissioned.Node; got != "node-a" || len(r.fn.Completions()) != 1 {
		t.Fatalf("commissioned %q, completions %v", got, r.fn.Completions())
	}

	// Restart: a completion held from the earlier generation cannot consume
	// the restart's callback; the restart completes only on the held fabric.
	if ev := runner.Stop(); ev.Outcome != "accepted" {
		t.Fatalf("stop %+v", ev)
	}
	if ev := runner.Restart(); ev.Outcome != "pending" {
		t.Fatalf("restart %+v", ev)
	}
	if ev := runner.Complete(first.Identity, true, nil); ev.Outcome != "ignored" {
		t.Fatalf("held earlier completion %+v", ev)
	}
	drive(t, route, runner, "succeeded")
	if got := route.Stats(); got.Driven != 5 || got.Failures != 0 {
		t.Fatalf("route stats %+v", got)
	}
	if r.site.Net().Sent() == 0 || runner.ProductionAllocations() != 0 {
		t.Fatalf("datagrams %d, allocations %d", r.site.Net().Sent(), runner.ProductionAllocations())
	}
}

// TestAA30CommissioningCloseNeedsFabric: without a CommissioningComplete the
// SDK reopens the node's own window, so the close completes as a failure and
// the runner keeps the window and refuses Stop. Catches: completing a close on
// the CloseWindow return alone.
func TestAA30CommissioningCloseNeedsFabric(t *testing.T) {
	r := aa30Start(t)
	route, err := sdkfabric.NewCommissioningRoute(r.fn, r.initiator(), aa30Creds(r.dev))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := commissioning.New(route, "node-a", commissioning.Store{})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start()
	runner.PASE()
	drive(t, route, runner, "succeeded")
	runner.OpenWindow(true)
	drive(t, route, runner, "succeeded")
	runner.Discover()
	drive(t, route, runner, "succeeded")
	runner.OpenWindow(false)
	drive(t, route, runner, "failed")
	if runner.Snapshot().Commissioned.Node != "" {
		t.Fatal("commissioned without a CommissioningComplete")
	}
	if ev := runner.Stop(); ev.Outcome != "rejected" {
		t.Fatalf("stop with the window open %+v", ev)
	}
	if got := route.Stats(); got.Driven != 4 || got.Failures != 1 {
		t.Fatalf("route stats %+v", got)
	}
}

// TestAA30CommissioningDrainFence: a revoked route invents no completion, so
// work left undrained at D=8 latches the runner and restart stays refused,
// also after the late completion. Catches: Revoke completing the request, or
// a drain deadline cleared by late cleanup.
func TestAA30CommissioningDrainFence(t *testing.T) {
	r := aa30Start(t)
	initiator := r.initiator()
	route, err := sdkfabric.NewCommissioningRoute(r.fn, initiator, aa30Creds(r.dev))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := commissioning.New(route, "node-a", commissioning.Store{})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start()
	pending := runner.PASE()
	if ev := runner.Stop(); ev.Outcome != "accepted" {
		t.Fatalf("stop %+v", ev)
	}
	q, ok := route.Revoke()
	if !ok || q.Identity != pending.Identity {
		t.Fatalf("revoked %+v, %v", q, ok)
	}
	if _, ok := route.Drive(); ok {
		t.Fatal("a revoked operation was driven")
	}
	route.Discover(q)
	if got := route.Stats(); got.Revoked != 1 || got.Refused != 1 || got.Driven != 0 {
		t.Fatalf("route stats %+v", got)
	}
	for step := 1; step < commissioning.DrainSteps; step++ {
		if ev := runner.Step(); ev.Outcome != "accepted" {
			t.Fatalf("step %d %+v", step, ev)
		}
	}
	if ev := runner.Step(); ev.Outcome != "drain-failed" {
		t.Fatalf("step D %+v", ev)
	}
	if ev := runner.Restart(); ev.Outcome != "rejected" {
		t.Fatalf("restart after drain failure %+v", ev)
	}
	if ev := runner.Complete(q.Identity, true, nil); ev.Outcome != "drained" {
		t.Fatalf("late completion %+v", ev)
	}
	if ev := runner.Restart(); ev.Outcome != "rejected" {
		t.Fatalf("restart after late completion %+v", ev)
	}
	if pases, _ := initiator.counts(); pases != 0 || r.port.Stats().Launches != 1 {
		t.Fatalf("initiator %d, port %+v", pases, r.port.Stats())
	}
}

// aa30Sink is the oracle's sink: it sends each dispatched report through the
// route and stages only a controller application receipt or a failure for the
// exact request, as the bounded adapter does, so the harness completes nothing
// on a dirty write.
type aa30Sink struct {
	route *sdkfabric.ReportRoute
	assoc map[reportharness.Key]reportharness.Association

	mu     sync.Mutex
	active *reportharness.SDKReport
	staged *reportharness.SDKCompletion
	dirty  int
	// delivered is the route's controller receipt count when a completion was
	// staged: a completion staged before its receipt arrived shows here.
	delivered int
}

func (s *aa30Sink) Dispatch(r reportharness.Report) {
	q := reportharness.SDKReport{Report: r, Association: s.assoc[r.Key]}
	s.mu.Lock()
	s.active = &q
	s.mu.Unlock()
	s.route.Send(q, func(c reportharness.SDKCompletion) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if c.Receipt == reportharness.SDKDirty {
			s.dirty++
		}
		if s.active == nil || c.Request != *s.active || s.staged != nil {
			return
		}
		if c.Receipt == reportharness.SDKControllerApplication || c.Receipt == reportharness.SDKFailed {
			s.staged, s.delivered = &c, s.route.Stats().Delivered
		}
	})
}

func (s *aa30Sink) take() (*reportharness.SDKCompletion, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.staged
	if c != nil {
		s.staged, s.active = nil, nil
	}
	return c, s.delivered
}

func (s *aa30Sink) dirties() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirty
}

func (s *aa30Sink) waiting() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.staged != nil
}

var aa30Key = reportharness.Key{Node: "node-a", Capability: "electrical.current", Composition: "composition-a"}

func aa30Report(gen, token, seq uint64, availability, value string) reportharness.Report {
	return reportharness.Report{
		Identity: reportharness.Identity{Key: aa30Key, Generation: gen, Token: token, Sequence: seq, Evidence: "synthetic:native-" + value},
		State:    reportharness.State{Value: value, Availability: availability, Confidence: "Proven", Qualification: "Unknown", Freshness: "fresh"},
	}
}

// subscribe feeds every ActiveCurrent value the controller application
// receives into the route's receipt boundary.
func (r *aa30Run) subscribe(t *testing.T, pair *fabrictest.Pair, route *sdkfabric.ReportRoute) {
	t.Helper()
	ep, ok := r.fn.Sensor().EndpointNumber()
	if !ok {
		t.Fatal("sensor has no endpoint number")
	}
	ceiling := time.Minute
	sub, err := fabrictest.Subscribe(r.ctx, pair, fabrictest.SubscribeRequest{
		Attributes:         []fabrictest.AttributePath{client.AttributeAt(ep, epmdef.ClusterID, epmdef.AttrActiveCurrent)},
		MaxIntervalCeiling: &ceiling,
		Updated: func(rep *fabrictest.Report) {
			for _, a := range rep.Attributes {
				if a.Path.Attribute == epmdef.AttrActiveCurrent {
					c := sdkfabric.ControllerReceipt{Null: a.Value.El.IsNull, Milliamperes: a.Value.El.Int, DataVersion: a.DataVersion}
					r.mu.Lock()
					r.last = c
					r.mu.Unlock()
					route.Receive(c)
				}
			}
		},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	t.Cleanup(sub.Close)
	if err := r.site.Eventually(r.ctx, "priming report", func() bool { return route.Stats().Receipts >= 1 }); err != nil {
		t.Fatal(err)
	}
}

func (r *aa30Run) lastReceipt() sdkfabric.ControllerReceipt {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last
}

// receipt waits for the staged completion and requires that the route had
// delivered the given number of controller receipts when it was staged.
func (r *aa30Run) receipt(t *testing.T, sink *aa30Sink, delivered int) *reportharness.SDKCompletion {
	t.Helper()
	if err := r.site.Eventually(r.ctx, "controller receipt", sink.waiting); err != nil {
		t.Fatal(err)
	}
	c, at := sink.take()
	if at != delivered {
		t.Fatalf("completion %+v staged at %d controller receipts, want %d", c, at, delivered)
	}
	return c
}

// TestAA30ReportRoute runs the existing report harness as the oracle over the
// real server write and subscription. Catches: completion on dirty instead of
// controller receipt, an admitted route without inspected inner buffers, lost
// or reordered FIFO observations, a dropped identity axis in the receipt
// fence, a second request in flight, a receipt revising a stopped request, a
// revoked request completing, and restart after D=8 undrained work.
func TestAA30ReportRoute(t *testing.T) {
	r := aa30Start(t)
	route, err := sdkfabric.NewReportRoute(r.fn)
	if err != nil {
		t.Fatal(err)
	}
	assoc := map[reportharness.Key]reportharness.Association{aa30Key: {Binding: "binding-a", Slot: "slot-a"}}

	// The route is refused by the bounded adapter: its inner buffers are not
	// inventoried. No route is admitted and each names its gap.
	if a, err := reportharness.NewSDKAdapter(route, route.Bounds(), assoc, 512); err == nil || a != nil {
		t.Fatal("the bounded adapter admitted an uninventoried route")
	}
	for _, a := range sdkfabric.Admissions() {
		if a.Admitted || a.Gap == "" {
			t.Fatalf("admission %+v", a)
		}
	}
	if _, err := sdkfabric.QualifyReference(r.fn, route); !errors.Is(err, sdkfabric.ErrReferenceUnqualified) {
		t.Fatalf("reference before commissioning: %v", err)
	}

	pair := r.commission(t, r.initiator())
	r.subscribe(t, pair, route)
	if _, err := sdkfabric.QualifyReference(r.fn, route); !errors.Is(err, sdkfabric.ErrReferenceUnqualified) {
		t.Fatalf("reference before any controller receipt: %v", err)
	}

	sink := &aa30Sink{route: route, assoc: assoc}
	h, err := reportharness.New(sink, assoc, 512)
	if err != nil {
		t.Fatal(err)
	}
	gen, err := h.Start()
	if err != nil {
		t.Fatal(err)
	}
	r1, r2, r3 := aa30Report(gen, 1, 1, "available", "16000"), aa30Report(gen, 2, 2, "available", "0"), aa30Report(gen, 3, 3, "available", "7")
	for _, rep := range []reportharness.Report{r1, r2} {
		if err := h.Submit(rep); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.Submit(r3); err == nil {
		t.Fatal("Q=2 exceeded")
	}

	if !h.Dispatch() || h.Dispatch() {
		t.Fatal("F=1 not held")
	}
	// The server write returned: dirty, not received.
	if sink.dirties() != 1 || route.Stats().Sets != 1 {
		t.Fatalf("dirty %d, stats %+v", sink.dirties(), route.Stats())
	}
	route.Send(reportharness.SDKReport{Report: r2, Association: assoc[aa30Key]}, func(c reportharness.SDKCompletion) {
		if c.Receipt != reportharness.SDKFailed {
			t.Errorf("second flight %+v", c)
		}
	})
	if got := route.Stats(); got.Refused != 1 || got.Sets != 1 {
		t.Fatalf("second flight reached the server: %+v", got)
	}

	c := r.receipt(t, sink, 1)
	if c.Receipt != reportharness.SDKControllerApplication || c.Request.Report != r1 {
		t.Fatalf("receipt %+v", c)
	}
	id := c.Request.Report.Identity
	for name, v := range map[string]reportharness.Identity{
		"node":        {Key: reportharness.Key{Node: "node-b", Capability: id.Key.Capability, Composition: id.Key.Composition}, Generation: id.Generation, Token: id.Token, Sequence: id.Sequence, Evidence: id.Evidence},
		"capability":  {Key: reportharness.Key{Node: id.Key.Node, Capability: "electrical.power", Composition: id.Key.Composition}, Generation: id.Generation, Token: id.Token, Sequence: id.Sequence, Evidence: id.Evidence},
		"composition": {Key: reportharness.Key{Node: id.Key.Node, Capability: id.Key.Capability, Composition: "composition-b"}, Generation: id.Generation, Token: id.Token, Sequence: id.Sequence, Evidence: id.Evidence},
		"generation":  {Key: id.Key, Generation: id.Generation + 1, Token: id.Token, Sequence: id.Sequence, Evidence: id.Evidence},
		"token":       {Key: id.Key, Generation: id.Generation, Token: id.Token + 1, Sequence: id.Sequence, Evidence: id.Evidence},
		"sequence":    {Key: id.Key, Generation: id.Generation, Token: id.Token, Sequence: id.Sequence + 1, Evidence: id.Evidence},
		"evidence":    {Key: id.Key, Generation: id.Generation, Token: id.Token, Sequence: id.Sequence, Evidence: "synthetic:other"},
	} {
		if got := h.Complete(v, true); got != nil {
			t.Fatalf("%s varied completed %+v", name, got)
		}
	}
	var results []reportharness.Result
	results = append(results, h.Complete(id, true)...)
	// A duplicate of the delivered controller receipt completes nothing.
	before := route.Stats()
	route.Receive(r.lastReceipt())
	if got := route.Stats(); got.Delivered != before.Delivered || got.Unmatched != before.Unmatched+1 || sink.waiting() {
		t.Fatalf("duplicate receipt %+v", got)
	}

	if !h.Dispatch() {
		t.Fatal("second dispatch")
	}
	c = r.receipt(t, sink, 2)
	results = append(results, h.Complete(c.Request.Report.Identity, c.Receipt == reportharness.SDKControllerApplication)...)
	if len(results) != 2 || results[0] != (reportharness.Result{Report: r1, Outcome: "received"}) || results[1] != (reportharness.Result{Report: r2, Outcome: "received"}) {
		t.Fatalf("FIFO results %+v", results)
	}

	q, err := sdkfabric.QualifyReference(r.fn, route)
	if err != nil {
		t.Fatalf("reference after receipts: %v", err)
	}
	if s, err := sdkfabric.NewStager(q); err != nil || s.Next() != sdkfabric.FamilyHVAC {
		t.Fatalf("stager %v", err)
	}

	// Stop with a request in flight: it stays delivery-unknown, and its later
	// receipt only releases the flight, so restart is allowed.
	if err := h.Submit(r3); err != nil || !h.Dispatch() {
		t.Fatalf("third report: %v", err)
	}
	if got := h.Stop(); len(got) != 1 || got[0] != (reportharness.Result{Report: r3, Outcome: "delivery-unknown"}) {
		t.Fatalf("stop %+v", got)
	}
	c = r.receipt(t, sink, 3)
	if got := h.Complete(c.Request.Report.Identity, true); got != nil {
		t.Fatalf("receipt after stop revised the outcome: %+v", got)
	}
	gen2, err := h.Start()
	if err != nil || gen2 != gen+1 {
		t.Fatalf("restart %d, %v", gen2, err)
	}

	// Revoke: the in-flight request never completes, its receipt is
	// unmatched, and D=8 undrained steps latch the restart refusal.
	r4 := aa30Report(gen2, 1, 4, "unavailable", "")
	if err := h.Submit(r4); err != nil || !h.Dispatch() {
		t.Fatalf("fourth report: %v", err)
	}
	if !route.Revoke() {
		t.Fatal("nothing in flight to revoke")
	}
	unmatched := route.Stats().Unmatched
	if err := r.site.Eventually(r.ctx, "receipt of the revoked value", func() bool { return route.Stats().Unmatched > unmatched }); err != nil {
		t.Fatal(err)
	}
	if sink.waiting() {
		t.Fatal("a revoked request completed")
	}
	if got := h.Stop(); len(got) != 1 || got[0].Outcome != "delivery-unknown" {
		t.Fatalf("stop after revoke %+v", got)
	}
	for step := 1; step < reportharness.DrainSteps; step++ {
		if err := h.Step(); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	if err := h.Step(); err == nil {
		t.Fatal("D=8 passed with undrained work")
	}
	if _, err := h.Start(); err == nil {
		t.Fatal("restart after drain failure")
	}
	route.Send(reportharness.SDKReport{Report: r4, Association: assoc[aa30Key]}, func(c reportharness.SDKCompletion) {
		if c.Receipt != reportharness.SDKFailed {
			t.Errorf("send after revoke %+v", c)
		}
	})

	got := route.Stats()
	if got.Sets != 4 || got.SetErrors != 0 || got.Delivered != 3 || got.Revoked != 1 || r.site.Net().Sent() == 0 {
		t.Fatalf("route stats %+v, datagrams %d", got, r.site.Net().Sent())
	}
}

// TestAA30ReportBudgets exceeds the byte budget alone, with Q and F free, and
// requires no server write. Catches: a byte check after dispatch.
func TestAA30ReportBudgets(t *testing.T) {
	r := aa30Start(t)
	route, err := sdkfabric.NewReportRoute(r.fn)
	if err != nil {
		t.Fatal(err)
	}
	assoc := map[reportharness.Key]reportharness.Association{aa30Key: {Binding: "binding-a", Slot: "slot-a"}}
	rep := aa30Report(1, 1, 1, "available", "16000")
	small, err := reportharness.New(&aa30Sink{route: route, assoc: assoc}, assoc, 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := small.Start(); err != nil {
		t.Fatal(err)
	}
	if err := small.Submit(rep); err == nil {
		t.Fatal("byte budget exceeded")
	}
	if small.Dispatch() || route.Stats() != (sdkfabric.ReportStats{}) {
		t.Fatalf("a refused report reached the route: %+v", route.Stats())
	}
	// Positive control: the same report fits a larger budget and is written.
	large, err := reportharness.New(&aa30Sink{route: route, assoc: assoc}, assoc, 512)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := large.Start(); err != nil {
		t.Fatal(err)
	}
	if err := large.Submit(rep); err != nil || !large.Dispatch() || route.Stats().Sets != 1 {
		t.Fatalf("control: %v, %+v", err, route.Stats())
	}
}

// TestAA30EndpointRetention retains the real SDK endpoint number across
// detach and reattach, keeps it after retirement, and preserves an unrelated
// sentinel. Catches: retirement releasing the number, a key moving to a new
// number, a missing E budget, and a retire that clears another key.
func TestAA30EndpointRetention(t *testing.T) {
	r := aa30Start(t)
	ledger, err := sdkfabric.NewEndpointLedger(2)
	if err != nil {
		t.Fatal(err)
	}
	number, err := ledger.Observe(r.fn)
	if err != nil || number == 0 {
		t.Fatalf("observe %d, %v", number, err)
	}
	sensor := fixtureContext().Sensor
	for _, op := range []adoption.TopologyOp{adoption.TopologyDetach, adoption.TopologyAttach} {
		if err := r.host.ChangeTopology(adoption.TopologyRequest{Op: op, Sensor: sensor}); err != nil {
			t.Fatalf("%s: %v", op, err)
		}
	}
	if again, err := ledger.Observe(r.fn); err != nil || again != number {
		t.Fatalf("reattached number %d, %v, want %d", again, err, number)
	}
	key := ""
	for k := range ledger.Retained() {
		key = k
	}
	if err := ledger.Retain(key, number+1); !errors.Is(err, sdkfabric.ErrEndpoint) {
		t.Fatalf("moved number: %v", err)
	}
	sentinel := number + 7
	if err := ledger.Retain("sentinel", sentinel); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Retain("third", number+8); !errors.Is(err, sdkfabric.ErrEndpointBudget) {
		t.Fatalf("E=2 exceeded: %v", err)
	}
	if err := ledger.Retire(key); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Observe(r.fn); !errors.Is(err, sdkfabric.ErrEndpoint) {
		t.Fatalf("retired key observed: %v", err)
	}
	if err := ledger.Retain("other", number); !errors.Is(err, sdkfabric.ErrEndpointReuse) {
		t.Fatalf("retired number reused: %v", err)
	}
	if got := ledger.Retained(); len(got) != 2 || got[key] != number || got["sentinel"] != sentinel {
		t.Fatalf("retained %v", got)
	}
}

// TestAA30MechanicsStaticBoundary pins what the mechanics may reach: the
// harness packages, the adoption contract and the SDK node package only, with
// no node construction, advertiser, test support or transport package.
func TestAA30MechanicsStaticBoundary(t *testing.T) {
	f := parseSource(t, "mechanics.go")
	if got := buildTag(f); got != "gofabric" {
		t.Fatalf("mechanics.go build constraint %q, want gofabric", got)
	}
	allowed := []string{
		"context", "errors", "fmt", "strconv", "sync",
		"github.com/Project-Helianthus/helianthus-matter-binding/adoption",
		"github.com/Project-Helianthus/helianthus-matter-binding/qualification/commissioning",
		"github.com/Project-Helianthus/helianthus-matter-binding/qualification/reportharness",
		fabricModule + "node",
	}
	for _, path := range imports(t, f) {
		if !slices.Contains(allowed, path) {
			t.Errorf("mechanics.go imports %q", path)
		}
	}
	used := selectors(t, f)
	for _, banned := range []string{fabricModule + "node.New", fabricModule + "node.Run"} {
		if used[banned] {
			t.Errorf("mechanics.go calls %s", banned)
		}
	}
}
