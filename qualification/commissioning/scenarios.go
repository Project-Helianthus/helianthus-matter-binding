// Package commissioning models offline commissioning traces, not upstream APIs.
package commissioning

import "fmt"

const DrainSteps = 8
const SymbolBytes = 32
const DiscoveryLimit = 2

type Identity struct {
	Generation, Token uint64
	Node              string
}
type Request struct {
	Identity
	Open bool
}

// Ports must return without callbacks or retained unbounded work. Complete is
// the separate callback barrier; only serial caller-driven use is supported.
type Ports interface {
	PASE(Request)
	OpenWindow(Request)
	Discover(Request)
	Restart(Request)
}
type Record struct{ Node string }
type Store struct{ Commissioned Record }
type Event struct {
	Step                      uint64
	Operation, Outcome, State string
	Identity
	Pending int
}

// Runner owns one pending callback and one synthetic store record. Events are
// returned to the caller, never accumulated in an internal trace or retry queue.
type Runner struct {
	ports                                        Ports
	store                                        Store
	node, state, operation                       string
	generation, token, step                      uint64
	pending                                      *Request
	running, stopping, failed, window, candidate bool
	drain                                        int
}

func New(ports Ports, node string, snapshot Store) (*Runner, error) {
	if ports == nil || !symbol(node) || (snapshot.Commissioned.Node != "" && snapshot.Commissioned.Node != node) {
		return nil, fmt.Errorf("invalid ports, node or conflicting store")
	}
	return &Runner{ports: ports, node: node, store: snapshot, state: "stopped"}, nil
}
func symbol(s string) bool { return len(s) > 0 && len(s) <= SymbolBytes }
func (r *Runner) event(op, outcome string, id Identity) Event {
	r.step++
	p := 0
	if r.pending != nil {
		p = 1
	}
	return Event{r.step, op, outcome, r.state, id, p}
}
func (r *Runner) Start() Event {
	if r.running || r.stopping || r.failed || r.pending != nil || r.generation == ^uint64(0) {
		return r.event("Start", "rejected", Identity{})
	}
	r.generation++
	r.running, r.state = true, "ready"
	return r.event("Start", "accepted", Identity{Generation: r.generation, Node: r.node})
}
func (r *Runner) begin(op string, open bool) Event {
	allowed := (op == "PASE" && r.state == "ready") ||
		(op == "OpenWindow" && ((open && r.state == "paired") || (!open && r.window))) ||
		(op == "Discover" && r.state == "window") ||
		(op == "Restart" && r.state == "ready")
	if !r.running || r.failed || r.pending != nil || !allowed || r.token == ^uint64(0) {
		return r.event(op, "rejected", Identity{})
	}
	r.token++
	q := Request{Identity{r.generation, r.token, r.node}, open}
	r.pending, r.operation = &q, op
	switch op {
	case "PASE":
		r.ports.PASE(q)
	case "OpenWindow":
		r.ports.OpenWindow(q)
	case "Discover":
		r.ports.Discover(q)
	case "Restart":
		r.ports.Restart(q)
	}
	return r.event(op, "pending", q.Identity)
}
func (r *Runner) PASE() Event                { return r.begin("PASE", false) }
func (r *Runner) OpenWindow(open bool) Event { return r.begin("OpenWindow", open) }
func (r *Runner) Discover() Event            { return r.begin("Discover", false) }

// Restart is allowed only after Stop and quiescence; recovery uses the detached
// in-memory record, never discovery, intent or a guessed commissioned identity.
func (r *Runner) Restart() Event {
	if r.running || r.stopping || r.failed || r.pending != nil || r.window {
		return r.event("Restart", "rejected", Identity{})
	}
	if r.generation == ^uint64(0) || r.token == ^uint64(0) {
		return r.event("Restart", "rejected", Identity{})
	}
	r.generation++
	r.running, r.state = true, "ready"
	return r.begin("Restart", false)
}
func (r *Runner) Complete(id Identity, success bool, records []Record) Event {
	if r.pending == nil || id.Generation != r.pending.Generation || id.Token != r.pending.Token || id.Node != r.pending.Node {
		return r.event("Callback", "ignored", id)
	}
	q, op := *r.pending, r.operation
	r.pending = nil
	if r.stopping || r.failed {
		if !r.failed {
			r.stopping = false
		}
		return r.event(op, "drained", id)
	}
	if !success {
		return r.event(op, "failed", id)
	}
	switch op {
	case "PASE":
		r.state = "paired"
	case "OpenWindow":
		if q.Open {
			r.window, r.state = true, "window"
		} else {
			r.window = false
			if r.candidate {
				r.store.Commissioned, r.state = Record{r.node}, "commissioned"
			} else {
				r.state = "ready"
			}
			r.candidate = false
		}
	case "Discover":
		if len(records) != 1 || len(records) > DiscoveryLimit || !symbol(records[0].Node) || records[0].Node != r.node {
			r.candidate = false
			return r.event(op, "ambiguous-or-missing", id)
		}
		r.candidate, r.state = true, "discovered"
	case "Restart":
		if r.store.Commissioned.Node == r.node {
			r.state = "commissioned"
		}
	}
	return r.event(op, "succeeded", id)
}

// Stop requires explicit successful window closure first. Outstanding callbacks
// are fenced and must release ownership before D; no cancellation is invented.
func (r *Runner) Stop() Event {
	if r.window || (r.pending != nil && r.operation == "OpenWindow") {
		return r.event("Stop", "rejected", Identity{})
	}
	if !r.running {
		return r.event("Stop", "unchanged", Identity{})
	}
	r.running, r.stopping, r.state, r.drain = false, r.pending != nil, "stopped", 0
	return r.event("Stop", "accepted", Identity{})
}
func (r *Runner) Step() Event {
	if r.stopping {
		r.drain++
		if r.drain >= DrainSteps {
			r.failed = true
		}
	}
	if r.failed {
		return r.event("Step", "drain-failed", Identity{})
	}
	return r.event("Step", "accepted", Identity{})
}
func (r *Runner) Snapshot() Store { return r.store }

// No allocation port or production launcher is reachable from this model.
func (r *Runner) ProductionAllocations() int { return 0 }
