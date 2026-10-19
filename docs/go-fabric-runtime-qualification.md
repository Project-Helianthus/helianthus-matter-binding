# Proposed go-fabric runtime qualification scenarios

This document defines the library-neutral offline harness contract for issue
#26, a partial contract slice of #24 / Project #2. It specifies evidence a
future go-fabric adapter must produce. The offline commissioning slice below
and the separate reporting harness implement only synthetic mechanics.
Nothing here is stable before Helianthus 1.0. Abstract event names below are
fixture symbols, not upstream Go APIs or a new public interface.

## Authority and current evidence

The current [readiness evaluator](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/runtime/production_readiness.py)
only evaluates evidence. It creates no node, endpoint or fabric. Its reviewed
lock has no accepted runtime composition-profile digest, so all current calls
deny allocation before caller profile or build metadata can permit it. Changing
caller metadata or producing passing harness results cannot change that decision.
A future reviewed profile must match the reviewed lock and mapping, provide
mandatory composition evidence, and have matching required build results.

The [SDK baseline](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/SDK_BASELINE.md)
describes a bounded upstream Electrical Sensor sample fixture. It does not
qualify a Helianthus runtime or go-fabric. Optional ActiveCurrent alone cannot
supply the required Descriptor, Power Topology and Electrical Power Measurement
composition, including non-unknown PowerMode, positive NumberOfMeasurementTypes,
matching non-empty evidenced Accuracy and evidenced nullable ActivePower.
This scenario contract neither supplies those facts nor substitutes a library's
sample target for them. These are baseline constraints, not an exhaustive review
of the locked data-model tree.

The [Portal endpoint lifecycle contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/PORTAL_ENDPOINT_LIFECYCLE.md)
owns selection persistence and the proposed durable opaque endpoint association.
It requires preservation of native evidence and partial last-known-good state
when publication is withheld. This harness consumes supplied intent and ledger
snapshots; it does not choose selections, persist Portal state, define migration,
or implement M42/M43. Its mock ledger is an observation aid, not a replacement
persistence design.

The [inventory completeness contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md)
keeps source confidence, native qualification, target closure, profile acceptance,
build evidence and admission separate. Protocol facts require the public native
contract first, then accepted semantic evidence and the public target mapping.
No such fact is introduced here. Public main links locate the reviewed sources;
they are not evidence that a moving branch matches the reviewed lock. Links
were compared with the supplied source documents offline, not fetched.

**Current implementation:** fail-closed evidence evaluation and the documented
upstream fixture boundary. **Offline implementation:** the [commissioning scenario model](fabrictest-scenario-adapter.md)
implements a bounded subset of commissioning and restart transitions for issue
#34; A33 reporting remains separate. **Proposal:** the broader harness events,
transitions, limits and scenarios below. **Missing evidence:** library sources,
a real adapter event mapping and library resource/lifetime measurements. No
upstream Go API behavior or library readiness is asserted. Proven / Hypothesis /
Unknown labels apply to individual claims with their support; a passing
synthetic trace proves only that tested mechanic in that fixture. It cannot
upgrade native qualification, acquisition, freshness or production admission.

## Isolated fixture model

Use a deterministic scheduler with explicit barriers and logical steps, an
in-memory ledger, an injected fake fabric store containing opaque test tokens,
and a fake report sink. No network listener, live commissioning, fabric
credentials, real SDK node, Gateway operation or command relay is permitted.
Allocation means a recorded mock allocation request only; endpoint slots are
synthetic labels, never real Matter endpoint numbers. A mechanics-only fixture
must not stub a successful admission decision into the production launcher.
A separate admission-denial control records zero allocation requests under the
current reviewed lock even when every mechanics scenario succeeds.

Each run declares finite budgets before starting: endpoint slots E=2, pending
allocation requests A=2, report queue Q=2, in-flight reports F=1, registered
callbacks C=4, recovery attempts R=1 per explicit start, and drain deadline D=8
logical scheduler steps. These small bounds are proposed fixture parameters,
not Matter or library limits. A future adapter must expose measurable equivalent
limits and a bounded completion mechanism; inability to do so is missing
qualification evidence. No hidden unbounded queue, waiter list, retry or retained
callback payload may sit outside the counters. Count bytes as well as objects,
with a declared per-payload size cap; reject oversized input before admission to
a queue. An adapter must declare its byte cap before execution.

Use only synthetic opaque symbols, such as n-a, c-a, c-b, p-a and b-a. Families
such as synthetic charging or metering are display labels, not identity keys.
The endpoint intent key K is (binding-node identity, semantic-capability identity,
compatible composition identity). A supplied ledger maps K to (opaque binding
identity, synthetic endpoint slot). Distinct K values cannot alias one slot;
retired slots cannot be reassigned to another K in the same node lifecycle.
Unknown or incompatible composition is withheld, never defaulted. Adapter
mechanics cannot establish actual composition compatibility.

Every asynchronous event carries (run generation, K, operation token); reports
also carry a monotonically ordered per-K observation sequence and an opaque
native-evidence reference. The generation changes on each successful start;
operation tokens distinguish retries in the same generation. Synthetic source
state includes independent fields for value, availability, confidence,
qualification, freshness and native provenance. Preserve them unchanged;
retained values must not become fresh merely because a report is retried.

Fixture events are Start(snapshot), Intent(K), AllocateDone(token, result),
Report(K, sequence, evidence), DeliverDone(token, result), Cancel(token), Stop,
and StoreDone(token, result). These are abstract labels. Completion is observable
in a trace containing input order, callback order, generation, key, token,
sequence, result, ledger before/after and resource counters. Use logical steps,
not real capture times. Failure results distinguish rejected, cancelled, failed,
and delivery-unknown; queue acceptance is not controller receipt.

## Proposed transitions and loss disposition

Start validates supplied snapshots before entering running. Missing, corrupt,
conflicting or incompatible state withholds the affected publication; valid
unaffected records and source evidence survive. A store-wide failure stops all
work dependent on that store. Start cannot infer commissioned state from intent,
a cached value or opaque store tokens. No recovery retry occurs automatically
beyond the declared budget; a later explicit start is a separate attempt.

In running, accept only bounded work for compatible fixture keys. An exact
repeat intent returns the same pending token or retained mock association,
without allocating another slot or accumulating duplicate waiters. A changed
composition is a different key requiring upstream migration/retirement review,
not silent reuse. Exposure in the model occurs only after durable-store success;
a failed or ambiguous commit never exposes an unverified new association.

Stop fences new work and callbacks, cancels queued work, and drains owned
in-flight work within D. A callback may release its own bookkeeping exactly once
but cannot mutate a stopped generation or newer operation. Report cancellation
before dispatch prevents sink invocation. Cancellation after dispatch cannot
promise retraction: if receipt is not established, return delivery-unknown and
record that ambiguity without retrying automatically. Never claim exactly-once
remote delivery. At D an unquiesced adapter fails qualification; it must remain
fenced and cannot restart successfully while old resources remain unsafe.
Successful stop leaves zero owned tasks, registrations, queue items and in-flight
work, while preserving supplied durable associations and source state.

Reports use bounded FIFO per endpoint and a single in-flight item in this
fixture. Do not coalesce or silently drop accepted observations. Reject a full
queue explicitly so the producer retains its state; any later resubmission is
an explicit new attempt. Late or duplicate delivery callbacks cannot regress
acknowledgement sequence, overwrite newer source state, or convert failure into
success. A success acknowledgement concerns the declared fake-sink boundary
only, not physical effect. Rejected, cancelled, failed and delivery-unknown work
remains visible in the trace; none erases native evidence or partial
last-known-good state. Restart does not automatically replay reports whose
receipt was ambiguous.

## Required synthetic scenarios

All rows describe required adapter checks, not executed library results. The
[offline commissioning subset](fabrictest-scenario-adapter.md) exercises only its
documented logical transitions; it does not complete any full adapter row. Each starts with a fresh
isolated fixture unless it explicitly continues through stop or restart. Adapter
evidence must map every abstract callback to the actual library lifetime and
completion boundary, without guessing unsupported APIs.

| Scenario and input events | Observable result | Retained-state invariant | Required eventual adapter evidence |
| --- | --- | --- | --- |
| Admission control: current lock, accepted-looking caller metadata and passing mechanics summary; request publication. | Existing evaluator denies; zero production or mock-launcher allocation calls. Mechanics runs remain isolated. | No new identity, endpoint or fabric state; source state unchanged. | Trace from the real evaluator boundary and an allocation spy; no permissive replacement evaluator. |
| Deterministic requests: supply K-a and K-b with preassigned slots slot-a and slot-b; submit intents in both orders; reverse completion order. Repeat with an empty ledger and two identical ordered input streams. | Restored requests retain supplied associations regardless of callback order; identical ordered streams yield identical request traces. For an empty ledger the declared deterministic scheduler assigns the first free synthetic slot before asynchronous completion. | No callback-order-dependent identity or cross-key slot reuse; no claim that different discovery orders produce the same fresh assignment. | Predeclared expected request/ledger trace, both completion schedules and an independent allocation spy. This mock ordering is not a Portal allocation algorithm. |
| Duplicate intent: submit K-a twice before completion, then again after store success. Supply a conflicting record claiming its slot for K-b. | One pending request and one association for exact duplicates; conflict withholds affected publication. | No extra endpoint, waiter or token accumulation; valid unrelated records survive. | Request counts, token equality, conflict reason and resource high-water counters. |
| Reordered and stale operations: create operation u-a, cancel it, then create u-b in the same generation; complete u-b before u-a. Repeat with stale failure after new success and duplicate completions. | Only the current operation may change state; stale/duplicate events are recorded and ignored except one-time cleanup. | u-a cannot overwrite u-b, free its slot or acknowledge its reports. | Barrier-controlled schedules, per-token cleanup counts and ledger/source snapshots. |
| Late callbacks: stop generation g-a with allocation, delivery and store completions held; release each before D. Start g-b after quiescence; inject synthetic duplicate g-a callbacks. | Stop completes once; old completions cannot publish, restore a removed registration, or affect g-b. A deliberately withheld callback past D fails qualification and prevents successful restart. | Durable valid records and source evidence survive; no freed resource is accessed by a late callback. | Callback registration/release trace, lifetime instrumentation, generation fence checks and timeout negative control. |
| Queue backpressure: hold one report in flight, enqueue sequences 2 and 3 for K-a, then submit 4; include an oversized payload. Release delivery barriers. | Sequence 4 is explicitly rejected while full; oversized input rejected; accepted sequence order is preserved. No automatic resubmission. | Rejection leaves producer state and evidence intact; pending accepted items are neither replaced nor silently dropped. | Sink invocation order, queue/byte high-water counters, rejection results and absence of hidden buffers. |
| Cancellation: cancel a queued report twice, then cancel a dispatched report before its completion; release a late success. | Queued report never invokes the sink; repeated cancel has one terminal result. Dispatched report is delivery-unknown if receipt is unresolved; late success cannot reopen a cancelled operation. | No sequence rollback, automatic replay or freshness upgrade; ambiguity remains recorded. | Separate dispatch and receipt barriers, terminal-result counts, sink spy and cancellation lifetime counters. |
| Repeated start/stop: start twice while running, stop twice, then perform three start/stop cycles with the same valid snapshot. | Second running start is explicitly rejected with no new generation or registrations; repeated stop is idempotent. Each later successful start has a fresh generation. | Same compatible durable association restored; zero owned resources after each successful stop; no report replay. | Per-cycle counters and identities, generation trace, duplicate-start result and deterministic drain completion. |
| Bounded exhaustion: fill E, A and C independently, then request one more of each; exercise Q, F and byte bounds through held callbacks. | Excess allocation or registration is explicitly rejected before side effects; F-full reports may enter only the bounded queue, otherwise are rejected. No counter exceeds its bound. Release barriers and retry explicitly. | Existing valid associations and accepted work survive; failed requests cannot consume slots permanently. | Separate boundary traces for every budget, byte accounting, failure reasons and resource recovery after drain. |
| Fabric-store failure: inject read failure, failed write before commit, and ambiguous commit after recording the synthetic ledger association but before success callback. Also fail an unrelated synthetic store key. | Read failure blocks dependent start; failed/ambiguous write prevents new exposure. No blind retry or replacement identity; unrelated valid state is retained and only independently safe work may proceed. | No fabricated commissioned state, deletion of shared tokens or overwrite of committed state. Previously valid source and ledger records survive. | Fault position relative to commit, store operation trace, untouched-key sentinel, pre/post snapshots and bounded recovery attempts. |
| Restart/recovery: restart after clean stop, failed write and ambiguous commit; for each, exercise valid, absent, corrupt and conflicting snapshot variants. Inject an old completion after recovery. | Only validated compatible durable records restore their original association. An ambiguous commit must be reconciled from storage before exposure; unresolved state remains withheld. No cached report automatically replays. | No guessed new identity, retired-slot reuse or old-generation mutation. Valid unaffected records and partial last-known-good evidence remain intact. | Crash-point snapshots, recovery decisions for every variant, old-callback trace and zero exposure before validation. |

## Evidence review and completion boundary

For each row, the future adapter bundle must include declared budgets and sink
semantics, independent expected outcomes, reproducible scheduler input, actual
trace, resource accounting before/peak/after, and explicit pass/fail/missing
status. A disconnected fake alone cannot qualify the library: show the adapter
actually invokes and observes the library's allocation, cancellation, callback,
store and cleanup boundaries. Missing hooks or unbounded teardown are unresolved
qualification blockers. Preserve failed traces rather than reporting only
successful schedules. No library runtime measurements have been collected here.

Identity tests must vary each axis independently: node, capability and composition
in K (catch omitted-key-field aliasing); binding identity and slot in the supplied
ledger (catch unchecked association conflicts); generation (catch stale-run
admission); operation token (catch stale retry admission); per-K sequence (catch
acknowledgement regression); native-evidence reference and provenance fields
(catch evidence substitution or loss). Vary value, availability, confidence,
qualification and freshness independently to detect unintended projection or
freshness promotion. These are reasoned mutation targets, not executed mutant
kills. Expected traces must be independent of the adapter's key construction;
include detached controls with no adapter dispatch and unchanged-state sentinels.

This contract adds no principal, credential, permission or authorization gate
before 0.9; authentication and authorization remain 0.11 work. Existing trust and
admission checks remain unchanged. It changes no deterministic public envelope,
eBUS MCP compatibility or GraphQL behavior/parity, G5 D1-D5/O8, or driver ownership
of the eeBUS output binding. It introduces no major or experimental namespace.

Manual source/claim/transition/loss/example/link review is the contract gate;
readability alone is not semantic validation or runtime proof. Existing CI
registers lock checks, Python tests and the upstream SDK fixture, not this
proposed adapter harness. Executable adapter integration and its CI registration
remain future binding-owner work. This document closes no parent acceptance:
production target closure and profile acceptance, library implementation and
qualification, independent review and applicable committed-head gates remain.
Transport conformance, live-fabric smoke, hardware acceptance, interoperability
and Matter certification are neither run nor established by this offline slice.
