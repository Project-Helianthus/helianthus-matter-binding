package reportharness

import (
	"fmt"
	"sync"
)

// SDKBuffer describes an inner SDK buffer, not the harness's outer queue.
// Disabled buffers must have zero limits; enabled buffers need both limits.
type SDKBuffer struct {
	Enabled      bool
	Items, Bytes int
}

// SDKRouteBounds is an assertion by an injected fixture port about its complete
// buffer inventory. Real SDK instrumentation and controller receipts remain
// unqualified; this declaration never enables production admission.
type SDKRouteBounds struct {
	InventoryComplete bool
	Callbacks, Events []SDKBuffer
}

// SDKReport preserves the entire accepted envelope and supplied association.
type SDKReport struct {
	Report      Report
	Association Association
}

type SDKReceipt string

const (
	SDKDirty                 SDKReceipt = "dirty"
	SDKEnqueued              SDKReceipt = "enqueued"
	SDKDatagram              SDKReceipt = "datagram"
	SDKMRPAcknowledged       SDKReceipt = "mrp-acknowledged"
	SDKControllerApplication SDKReceipt = "controller-application-receipt"
	SDKFailed                SDKReceipt = "failed"
)

type SDKCompletion struct {
	Request SDKReport
	Receipt SDKReceipt
}

// SDKReportPort must return promptly, preserve each request independently, and
// use only the declared bounded/disabled buffers. Callbacks may be synchronous
// or concurrent. No upstream SDK API or live delivery is implied by this port.
type SDKReportPort interface {
	Send(SDKReport, func(SDKCompletion))
}

// SDKAdapter implements Sink using one flight mailbox. Harness methods and Poll
// are driven serially. Callbacks only stage an exact terminal completion; they
// never call the harness, dispatch, wait on a channel, or create a goroutine.
// The mutex protects a fixed-size mailbox and is never held across port calls.
type SDKAdapter struct {
	h       *Harness
	port    SDKReportPort
	mu      sync.Mutex
	active  *SDKReport
	pending *SDKCompletion
}

var _ Sink = (*SDKAdapter)(nil)

func NewSDKAdapter(port SDKReportPort, bounds SDKRouteBounds, associations map[Key]Association, byteCap int) (*SDKAdapter, error) {
	if port == nil || !bounds.InventoryComplete || len(bounds.Callbacks) == 0 || len(bounds.Events) == 0 {
		return nil, fmt.Errorf("port and complete explicit inner buffer inventory required")
	}
	for _, buffers := range [][]SDKBuffer{bounds.Callbacks, bounds.Events} {
		for _, b := range buffers {
			if (b.Enabled && (b.Items <= 0 || b.Bytes <= 0)) || (!b.Enabled && (b.Items != 0 || b.Bytes != 0)) {
				return nil, fmt.Errorf("inner buffers must be bounded or explicitly disabled")
			}
		}
	}
	a := &SDKAdapter{port: port}
	h, err := New(a, associations, byteCap)
	if err != nil {
		return nil, err
	}
	a.h = h
	return a, nil
}

// Dispatch is the Sink seam, called only by the owned harness. It refuses
// unsolicited calls and retains one callback closure per dispatched observation.
func (a *SDKAdapter) Dispatch(r Report) {
	if a.h.flight == nil || *a.h.flight != r {
		return
	}
	association, ok := a.h.Association(r.Key)
	if !ok {
		return
	}
	request := SDKReport{r, association}
	a.mu.Lock()
	if a.active != nil {
		a.mu.Unlock()
		return
	}
	a.active = &request
	a.mu.Unlock()
	a.port.Send(request, func(c SDKCompletion) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.active != &request || c.Request != request || a.pending != nil {
			return
		}
		if c.Receipt != SDKControllerApplication && c.Receipt != SDKFailed {
			return
		}
		a.pending = &c
	})
}

// Poll is the explicit receipt barrier. Enqueue, dirty marking, datagrams and
// MRP acknowledgement cannot acknowledge any sample. Cancelled dispatched work
// remains delivery-unknown; a later receipt only releases its original budget.
func (a *SDKAdapter) Poll() []Result {
	a.mu.Lock()
	c := a.pending
	if c != nil {
		a.pending = nil
		a.active = nil
	}
	a.mu.Unlock()
	if c == nil {
		return nil
	}
	return a.h.Complete(c.Request.Report.Identity, c.Receipt == SDKControllerApplication)
}

func (a *SDKAdapter) Start() (uint64, error)                { return a.h.Start() }
func (a *SDKAdapter) Submit(r Report) error                 { return a.h.Submit(r) }
func (a *SDKAdapter) DispatchNext() bool                    { return a.h.Dispatch() }
func (a *SDKAdapter) Cancel(id Identity) []Result           { return a.h.Cancel(id) }
func (a *SDKAdapter) Stop() []Result                        { return a.h.Stop() }
func (a *SDKAdapter) Step() error                           { return a.h.Step() }
func (a *SDKAdapter) Counts() Counters                      { return a.h.Counts() }
func (a *SDKAdapter) HighWater() Counters                   { return a.h.HighWater() }
func (a *SDKAdapter) Source(k Key) (Report, bool)           { return a.h.Source(k) }
func (a *SDKAdapter) Association(k Key) (Association, bool) { return a.h.Association(k) }
