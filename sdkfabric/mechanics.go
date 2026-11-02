//go:build gofabric

package sdkfabric

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/Project-Helianthus/helianthus-matter-binding/adoption"
	"github.com/Project-Helianthus/helianthus-matter-binding/qualification/commissioning"
	"github.com/Project-Helianthus/helianthus-matter-binding/qualification/reportharness"
	"github.com/SukramJ/go-fabric/node"
)

// The mechanics below connect the commissioning runner and the report harness
// port to one reference Node. They reach the SDK only through that Node's
// methods, on a fixture node an injected launcher started; they construct no
// node, controller or advertiser and admit no route.

var (
	ErrInitiator      = errors.New("sdkfabric: commissioning initiator is missing")
	ErrEndpoint       = errors.New("sdkfabric: SDK endpoint number is absent or not retained")
	ErrEndpointReuse  = errors.New("sdkfabric: endpoint number is reserved for another key")
	ErrEndpointBudget = errors.New("sdkfabric: endpoint reservation budget exhausted")
	ErrRevoked        = errors.New("sdkfabric: route is revoked")
)

// Route names one SDK mechanics route.
type Route string

const (
	RouteCommissioning Route = "commissioning"
	RouteReport        Route = "report"
	RouteStore         Route = "store"
)

// RouteAdmission records whether a route is admitted and, if not, the exact
// unresolved gap. No route is admitted by this revision.
type RouteAdmission struct {
	Route    Route
	Admitted bool
	Gap      string
}

// Admissions lists every route with its gap. A gap closes only on inspected
// SDK evidence of the bound or completion, never on a nominal cap here.
func Admissions() []RouteAdmission {
	return []RouteAdmission{
		{RouteCommissioning, false, "the reference Node keeps every CommissioningComplete and window transition the SDK observers report, without an inspected SDK bound on observer calls"},
		{RouteReport, false, "the SDK's inner subscription, report and event buffers are not inventoried, so the bounded report adapter refuses the route"},
		{RouteStore, false, "no SDK storage or endpoint-store route with an explicit commit acknowledgement is bound; no endpoint-fabric port reaches the SDK"},
	}
}

// Initiator is the injected commissioning initiator, a fixture controller on
// the in-memory site. The route never constructs one.
type Initiator interface {
	PASE(context.Context) error
	Discover(context.Context) (int, error)
}

// CommissioningCompletion is what the runner's Complete barrier takes.
type CommissioningCompletion struct {
	Identity commissioning.Identity
	Success  bool
	Records  []commissioning.Record
}

// CommissioningStats counts staged operations and real SDK or initiator calls.
type CommissioningStats struct {
	Staged, Refused, Revoked, Driven, Failures int
}

type stagedOperation struct {
	op      string
	request commissioning.Request
}

// CommissioningRoute implements commissioning.Ports over one Node. A port call
// only stages its request in one slot and returns; Drive performs it and hands
// the completion back to the caller. Nothing here calls the runner.
type CommissioningRoute struct {
	node      *Node
	initiator Initiator
	creds     node.WindowCredentials

	mu      sync.Mutex
	staged  *stagedOperation
	revoked bool
	stats   CommissioningStats
}

var _ commissioning.Ports = (*CommissioningRoute)(nil)

func NewCommissioningRoute(n *Node, initiator Initiator, creds node.WindowCredentials) (*CommissioningRoute, error) {
	switch {
	case n == nil:
		return nil, ErrPortMissing
	case initiator == nil:
		return nil, ErrInitiator
	}
	return &CommissioningRoute{node: n, initiator: initiator, creds: creds}, nil
}

func (r *CommissioningRoute) stage(op string, q commissioning.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.revoked || r.staged != nil {
		r.stats.Refused++
		return
	}
	r.staged = &stagedOperation{op, q}
	r.stats.Staged++
}

func (r *CommissioningRoute) PASE(q commissioning.Request)       { r.stage("PASE", q) }
func (r *CommissioningRoute) OpenWindow(q commissioning.Request) { r.stage("OpenWindow", q) }
func (r *CommissioningRoute) Discover(q commissioning.Request)   { r.stage("Discover", q) }
func (r *CommissioningRoute) Restart(q commissioning.Request)    { r.stage("Restart", q) }

// Drive performs the one staged operation within the Node's call bound. A
// close succeeds only when the SDK reports no open window and the node holds
// the fabric of an observed CommissioningComplete; a restart succeeds only on
// that same held fabric. Discovery yields one record per match, capped one
// past the runner's limit, so an ambiguous result stays ambiguous.
func (r *CommissioningRoute) Drive() (CommissioningCompletion, bool) {
	r.mu.Lock()
	s := r.staged
	r.staged = nil
	r.mu.Unlock()
	if s == nil {
		return CommissioningCompletion{}, false
	}
	ctx, cancel := r.node.bound()
	defer cancel()
	var err error
	var records []commissioning.Record
	switch s.op {
	case "PASE":
		err = r.initiator.PASE(ctx)
	case "OpenWindow":
		if s.request.Open {
			err = r.node.OpenWindow(ctx, r.creds)
		} else if err = r.node.CloseWindow(ctx); err == nil {
			err = r.node.RequireCommissioned(ctx)
		}
	case "Discover":
		var found int
		if found, err = r.initiator.Discover(ctx); err == nil {
			found = min(found, commissioning.DiscoveryLimit+1)
			for range found {
				records = append(records, commissioning.Record{Node: s.request.Node})
			}
		}
	case "Restart":
		err = r.node.RequireCommissioned(ctx)
	}
	r.mu.Lock()
	r.stats.Driven++
	if err != nil {
		r.stats.Failures++
	}
	r.mu.Unlock()
	return CommissioningCompletion{s.request.Identity, err == nil, records}, true
}

// Revoke drops a staged operation without performing it and refuses later
// ones. It invents no completion: the runner's drain deadline still applies.
func (r *CommissioningRoute) Revoke() (commissioning.Request, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked = true
	s := r.staged
	r.staged = nil
	if s == nil {
		return commissioning.Request{}, false
	}
	r.stats.Revoked++
	return s.request, true
}

func (r *CommissioningRoute) Stats() CommissioningStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

// ControllerReceipt is one ActiveCurrent value a controller application
// received in a subscription report, with the report's data version.
type ControllerReceipt struct {
	Null         bool
	Milliamperes int64
	DataVersion  uint32
}

// ReportStats counts Send calls, real server writes and controller receipts.
type ReportStats struct {
	Sends, Refused, Sets, SetErrors, Dirty, Receipts, Unmatched, Delivered, Revoked int
}

type reportFlight struct {
	request reportharness.SDKReport
	want    ControllerReceipt
	done    func(reportharness.SDKCompletion)
}

// ReportRoute implements reportharness.SDKReportPort over the Node's
// ActiveCurrent. A server write is only SDKDirty; the request completes with
// SDKControllerApplication only through Receive, the controller application
// receipt boundary. It keeps one request in flight and no history.
type ReportRoute struct {
	node *Node

	mu      sync.Mutex
	flight  *reportFlight
	seen    uint32
	seenAny bool
	revoked bool
	stats   ReportStats
}

var _ reportharness.SDKReportPort = (*ReportRoute)(nil)

func NewReportRoute(n *Node) (*ReportRoute, error) {
	if n == nil {
		return nil, ErrPortMissing
	}
	return &ReportRoute{node: n}, nil
}

// Bounds declares no inner buffer inventory: the SDK's own subscription,
// report and event buffers were not inspected, so the declaration is
// incomplete and the bounded adapter refuses it rather than trusting a cap.
func (r *ReportRoute) Bounds() reportharness.SDKRouteBounds {
	return reportharness.SDKRouteBounds{InventoryComplete: false}
}

// current maps a report to ActiveCurrent: "available" carries a decimal
// milliampere Value, "unavailable" is null, anything else is refused.
func current(rep reportharness.Report) (ControllerReceipt, bool) {
	switch rep.State.Availability {
	case "available":
		v, err := strconv.ParseInt(rep.State.Value, 10, 64)
		return ControllerReceipt{Milliamperes: v}, err == nil
	case "unavailable":
		return ControllerReceipt{Null: true}, rep.State.Value == ""
	}
	return ControllerReceipt{}, false
}

// Send writes the value to the server and reports SDKDirty, or SDKFailed if
// the write or the request is refused. It never reports a receipt itself.
func (r *ReportRoute) Send(q reportharness.SDKReport, done func(reportharness.SDKCompletion)) {
	want, ok := current(q.Report)
	r.mu.Lock()
	r.stats.Sends++
	if done == nil {
		r.stats.Refused++
		r.mu.Unlock()
		return
	}
	if !ok || r.revoked || r.flight != nil {
		r.stats.Refused++
		r.mu.Unlock()
		done(reportharness.SDKCompletion{Request: q, Receipt: reportharness.SDKFailed})
		return
	}
	f := &reportFlight{q, want, done}
	r.flight = f
	r.mu.Unlock()
	err := r.node.PublishCurrent(adoption.CurrentUpdate{
		Generation: r.node.plan.Generation, Sequence: q.Report.Sequence, SensorKey: r.node.plan.SensorKey,
		Null: want.Null, Milliamperes: want.Milliamperes, NativeEvidence: q.Report.Evidence,
	})
	r.mu.Lock()
	r.stats.Sets++
	if err != nil {
		r.stats.SetErrors++
		if r.flight == f {
			r.flight = nil
		}
		r.mu.Unlock()
		done(reportharness.SDKCompletion{Request: q, Receipt: reportharness.SDKFailed})
		return
	}
	r.stats.Dirty++
	r.mu.Unlock()
	done(reportharness.SDKCompletion{Request: q, Receipt: reportharness.SDKDirty})
}

// Receive is the controller application receipt boundary. Only a value equal
// to the in-flight request, with a data version newer than every version
// received before it, completes that request; anything else is dropped and
// counted. Receive may run on an SDK goroutine and calls only the completion.
func (r *ReportRoute) Receive(c ControllerReceipt) {
	r.mu.Lock()
	r.stats.Receipts++
	newer := !r.seenAny || c.DataVersion > r.seen
	if newer {
		r.seen, r.seenAny = c.DataVersion, true
	}
	f := r.flight
	if f == nil || !newer || c.Null != f.want.Null || (!c.Null && c.Milliamperes != f.want.Milliamperes) {
		r.stats.Unmatched++
		r.mu.Unlock()
		return
	}
	r.flight = nil
	r.stats.Delivered++
	r.mu.Unlock()
	f.done(reportharness.SDKCompletion{Request: f.request, Receipt: reportharness.SDKControllerApplication})
}

// Revoke drops the in-flight request without completing it and refuses later
// sends. A later receipt is unmatched; the harness keeps the request
// delivery-unknown.
func (r *ReportRoute) Revoke() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked = true
	if r.flight == nil {
		return false
	}
	r.flight = nil
	r.stats.Revoked++
	return true
}

func (r *ReportRoute) Stats() ReportStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

// EndpointLedger retains the SDK endpoint number first observed for each
// sensor key for the whole node lifecycle. A number is never given to another
// key, also after its key is retired, and a key never changes its number.
type EndpointLedger struct {
	limit    int
	byKey    map[string]uint16
	byNumber map[uint16]string
	retired  map[string]bool
}

func NewEndpointLedger(limit int) (*EndpointLedger, error) {
	if limit <= 0 {
		return nil, ErrEndpointBudget
	}
	return &EndpointLedger{limit: limit, byKey: map[string]uint16{}, byNumber: map[uint16]string{}, retired: map[string]bool{}}, nil
}

// Observe retains the endpoint number the SDK assigned to the Node's sensor.
func (l *EndpointLedger) Observe(n *Node) (uint16, error) {
	if n == nil {
		return 0, ErrPortMissing
	}
	number, ok := n.Sensor().EndpointNumber()
	if !ok {
		return 0, ErrEndpoint
	}
	return number, l.Retain(n.plan.SensorKey, number)
}

// Retain records key at number, or confirms the unchanged retained pair.
func (l *EndpointLedger) Retain(key string, number uint16) error {
	if key == "" || number == 0 {
		return ErrEndpoint
	}
	if l.retired[key] {
		return fmt.Errorf("%w: %q is retired", ErrEndpoint, key)
	}
	if held, ok := l.byKey[key]; ok {
		if held != number {
			return fmt.Errorf("%w: %q moved from %d to %d", ErrEndpoint, key, held, number)
		}
		return nil
	}
	if owner, ok := l.byNumber[number]; ok {
		return fmt.Errorf("%w: %d belongs to %q", ErrEndpointReuse, number, owner)
	}
	if len(l.byKey) >= l.limit {
		return ErrEndpointBudget
	}
	l.byKey[key], l.byNumber[number] = number, key
	return nil
}

// Retire keeps key's number reserved for the rest of the node lifecycle.
func (l *EndpointLedger) Retire(key string) error {
	if _, ok := l.byKey[key]; !ok {
		return fmt.Errorf("%w: %q is unknown", ErrEndpoint, key)
	}
	l.retired[key] = true
	return nil
}

// Retained returns a detached copy of every retained key and number.
func (l *EndpointLedger) Retained() map[string]uint16 {
	out := make(map[string]uint16, len(l.byKey))
	for k, v := range l.byKey {
		out[k] = v
	}
	return out
}

// QualifyReference builds the reference qualification from the real Node: a
// held CommissioningComplete fabric with no window open and at least one
// controller application receipt delivered on the Node's report route.
func QualifyReference(n *Node, reports *ReportRoute) (ReferenceQualification, error) {
	if n == nil || reports == nil || reports.node != n {
		return ReferenceQualification{}, ErrReferenceUnqualified
	}
	ctx, cancel := n.bound()
	defer cancel()
	if err := n.RequireCommissioned(ctx); err != nil {
		return ReferenceQualification{}, fmt.Errorf("%w: %w", ErrReferenceUnqualified, err)
	}
	completions := n.Completions()
	return newReferenceQualification(n.plan.Generation, n.plan.SensorKey, completions[len(completions)-1], reports.Stats().Delivered)
}
