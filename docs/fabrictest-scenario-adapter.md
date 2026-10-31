# Offline commissioning scenario adapter proposal

[Issue #34](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/34)
implements a partial commissioning/restart slice of Z20 for
[issue #24](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/24).
Nothing here is stable before Helianthus 1.0. The standard-library
[runner](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/qualification/commissioning/scenarios.go)
uses local proposed port signatures named PASE, OpenWindow, Discover and Restart.
These names are capability labels supplied by the task, not verified upstream
fabrictest APIs. No upstream library is imported or qualified.

## Executable boundary and trace

A serial caller supplies Ports and a detached Store value, then drives operations
and Complete barriers. Ports must return without reentrant callbacks; asynchronous
library callbacks must later be marshalled onto the same serial scheduler. Each
returned Event contains a logical step, operation, outcome, state, generation,
token, synthetic node identity and pending callback count. The caller owns trace
retention; the runner accumulates no history, waiter list or automatic retries.
Node labels such as n-a and n-b are synthetic, not real device identifiers.
Synthetic metering and charging families are illustrative only.

Successful empty-store trace:

| Ordered input / barrier | Resulting state | Store |
| --- | --- | --- |
| Start | ready | empty |
| PASE; successful matching completion | paired | empty |
| OpenWindow(true); successful matching completion | window | empty |
| Discover; one matching synthetic record | discovered | empty |
| OpenWindow(false); successful matching completion | commissioned | n-a |
| Stop with zero pending callbacks | stopped | n-a |
| Restart; successful matching completion | commissioned, new generation | n-a |

The label commissioned means completion of this fake sequence only. Discovery
alone never establishes it. Explicit window closure must succeed before that
sequence persists its one synthetic association. Missing, duplicate, conflicting
or excessive discovery results are rejected without changing the store. A failed
operation leaves the previous state and store intact, and requires an explicit
retry with a new token. Closing a window without a selected candidate returns to
ready without recording an association. A failed close leaves the window open.
Stop rejects an open window or a pending window operation; the caller must finish
and explicitly close it first. This is a fixture restriction, not a claim that
upstream teardown has those semantics.

Stop fences other pending callbacks. Only the exact pending generation, token and
node can release the registration; completion during drain cannot advance the
state or store. Duplicates and mismatches are ignored without cleanup of another
operation. Restart is rejected until that callback has drained. A completion must
arrive before drain step eight; reaching the deadline permanently fences that
runner even if cleanup arrives later. Clean restart increments generation and
consults only its detached in-memory record. An absent record remains ready;
a conflicting supplied record is rejected at construction. No durable persistence,
crash recovery, ambiguous commit reconciliation or native fabric storage is modeled.
Partial last-known-good state belongs to its source owner; this runner retains its
supplied valid record through operation failures and never projects source data.

## Bounds and later adapter obligations

One pending operation/registration, one store record and one candidate bit bound
retained work. Generation and token counters refuse overflow; tokens increase
across restarts. Node symbols are at most 32 bytes. Discovery accepts exactly one
matching record and declares a two-record input limit; larger slices are rejected
without traversal or retention. The input slice belongs to the caller, so this
does not bound allocation by an external producer. The future adapter must enforce
input object and byte limits before buffering, including malformed discovery
batches, callback queues and payloads. The runner has no background tasks, sockets,
mDNS, BLE, credentials, SDK node or production allocation port. Its zero allocation
counter describes this structural isolation, not a spy on the production evaluator.
The existing admission evaluator and lock remain unchanged and fail closed.

A real fabrictest adapter must first supply reviewed public library sources and
map each proposed operation to the actual entry point, return/error boundary,
callback identity, window-close acknowledgement and resource-release boundary.
It must demonstrate genuine PASE completion independently of discovery, establish
what closing a window guarantees, and validate restored fabric state independently
of cached discovery. The current model trusts injected success flags and its
synthetic store; they are not evidence of any of those library facts. Missing
capabilities remain Unknown. No guessed signature or passing fake may qualify
fabrictest. Unobservable release, hidden buffering or unbounded shutdown blocks
qualification. Retain rejected and timeout traces alongside successes.

## PASE, CASE and window completion

The [one-sensor reference](adoption/reference-bridge.md) and
[runtime row ledger](go-fabric-runtime-qualification.md#row-to-harness-and-adapter-trace)
place this partial model inside the planned adapter. Its `paired` state records
only injected PASE success. `commissioned` records the fake close/discovery
sequence, not operational commissioning or a validated CASE session. A future
adapter must observe PASE completion, operational commissioning completion and
matching CASE communication independently; cached discovery or Restart success
flags are not substitutes for validated fabric state.

Window close and revoke are distinct boundaries to trace in a real adapter.
OpenWindow(false) supplies a fake close acknowledgement only. There is no revoke
port or cancellation guarantee in this runner. A failed close leaves the window
open and Stop rejects it; it does not silently revoke or discard callbacks.
Before real teardown can be accepted, show actual close/revoke acknowledgement,
old-window callback fencing and registration release. Missing observations stay
Unknown. At D=8 logical steps with a callback outstanding, failure stays latched
even when matching cleanup arrives later; restart remains denied.

The integrated reference declares E=2/A=2/C=4/Q=2/F=1, R=1 explicit recovery
attempt, D=8 logical steps and a positive declared byte cap. Here there is only
one pending callback, a 32-byte node-symbol cap and a two-record discovery input
limit. They are fixture budgets, not product capacity or a bound on external/SDK
buffers. The caller must account for every copied input and retained trace.
Actual allocation, durable store reconciliation, CASE and integrated callback
budgets remain unverified. The structural zero-allocation method must be paired
with the separate actual-evaluator denial control before adapter qualification.

## Evidence and remaining acceptance

The [tests](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/qualification/commissioning/scenarios_test.go)
assert literal expected state transitions, ordered dispatch, rejection without
extra dispatch, discovery ambiguity, explicit closure, detached store retention,
restart after quiescence and permanent deadline failure. Callback tests vary
generation, token and synthetic node independently. Their respective mutation
targets remove the generation comparison, token comparison and node comparison
in Complete. A duplicate callback while another operation is pending catches
release of a different registration; a duplicate after cleanup catches dereferencing
or accepting an absent registration. Discovery-only storage mutation is caught by
the empty-store assertion before closing the window. Actual executed mutation
results belong to the per-run evidence report, not a standing qualification claim.

Proven: the tested synthetic mechanics when the focused check passes. Hypothesis:
these ports could be bound to a later real adapter. Unknown: actual library API,
commissioning, cancellation, persistence and teardown behavior. These labels do
not upgrade native qualification, freshness, runtime, physical acceptance,
interoperability or certification. Run only the focused offline check for this
slice: `GO111MODULE=off go test ./qualification/commissioning -count=1`.

[A33 reporting mechanics](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/go-fabric-report-mechanics-harness.md)
remain separate and unchanged. This slice introduces no protocol mappings, no
common production Provider interface, Gateway operation, authorization gate,
public envelope, eBUS MCP or GraphQL change. It changes neither G5 D1-D5/O8 nor
eeBUS output ownership. Authentication and authorization remain 0.11 work; no
new authorization gate is introduced before 0.9. There is no new namespace.
Manual source/claim/transition/loss/example/link review is required. This partial
slice closes no parent acceptance; real adapter integration, production admission,
independent review, committed-head CI, transport/device tests and physical or
live-fabric smoke remain outside its evidence.
