# Isolated logical-step report mechanics

This implements only Z20 for [issue #30](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/30),
a partial slice of [#24](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/24)
and Project #2. The [runtime qualification contract](go-fabric-runtime-qualification.md)
defines the broader scenarios. Nothing here is stable before Helianthus 1.0.

## Boundary and evidence

The standard-library-only [harness](../qualification/reportharness/harness.go)
accepts preassigned synthetic keys and opaque associations. It allocates no node
or endpoint, persists nothing, and has no SDK or go-fabric adapter. Synthetic
metering and charging families are display context only. It implements no other
qualification scenario, fabric store, recovery algorithm, runtime registration,
M42/M43 replacement, or C++ sidecar.

Proven means the specific synthetic mechanics exercised by the
[tests](../qualification/reportharness/harness_test.go), when those tests pass.
Hypothesis: a future adapter can expose equivalent bounded dispatch, receipt and
cleanup boundaries. Unknown: actual library behavior, resource lifetime and live
delivery. These claim labels remain separate from native qualification,
availability, freshness and admission. A fake sink receipt establishes only that
fixture boundary; it does not establish controller receipt or physical effect.

The [production readiness evaluator](../runtime/production_readiness.py) remains
unchanged and fail closed: the reviewed lock has no accepted composition-profile
digest. No harness result or caller metadata permits production allocation.
The [Portal lifecycle contract](PORTAL_ENDPOINT_LIFECYCLE.md) retains ownership of
selection and durable association design. Protocol facts require the public
native contract first, then semantic evidence and target mapping; this fixture
introduces none. Native trust, deterministic public envelopes, eBUS MCP
compatibility, GraphQL parity, G5 D1-D5/O8 and driver-owned eeBUS output are
unchanged. No authorization gate is introduced before 0.9; that roadmap remains
0.11 work.

## Scheduling and bounded ownership

Construct with an injected fake `Sink`, a detached map of synthetic `Key` to
`Association`, and a positive byte cap B. Missing or conflicting associations
reject construction; this is fixture validation, not a persistence qualification
scenario. Keys contain node, capability and compatible-composition symbols;
binding and slot are opaque supplied symbols. The harness never infers actual
composition compatibility. Call methods serially. `Sink.Dispatch` must return
without blocking or reentering the harness; the fake sink owns its observation
log. A blocking or real SDK sink is outside this fixture's contract.

`Start` returns a fresh generation. `Submit` accepts bounded work without calling
the sink. Tokens must be positive and strictly increase within a run; sequences
must be positive and strictly increase per key across clean restarts. Evidence
references must be nonempty. Rejection returns an error before changing source,
token, sequence or queue state; the producer retains rejected input. Accepted
source state is retained per supplied key and is never projected or refreshed
by delivery. This includes value, availability, confidence, qualification,
freshness and the opaque native evidence reference.

Q=2 queued reports, F=1 dispatched report and D=8 logical drain steps are fixture
constants, not Matter limits. B bounds the sum of pending envelope string bytes,
including payload, key, evidence and every source-state axis; thus it also bounds
each individual report. Fixed scalar overhead is bounded by Q/F. `Counts` and
`HighWater` expose queue, flight and pending bytes. `RetainedBytes` separately
counts one last accepted source envelope per supplied key, including after stop;
for N supplied keys this is bounded by N times B. Association memory is fixed by
the caller's finite input map. There are no harness goroutines, callback
registrations, hidden waiters, terminal-history list or retry buffer. Result
slices and the fake sink log are caller-owned evidence, outside pending work.

`Dispatch` releases the FIFO head only if the flight slot is free. The explicit
`Complete(identity, received)` receipt barrier releases that slot; it does not
dispatch the next report. Every key field, generation, token, sequence and native
evidence reference must match the flight identity. Nonmatching and duplicate
receipts return no result and cannot release newer work or overwrite source
state. Observers record input calls, dispatch calls, returned results and counters
in their own bounded fixture trace; the harness does not accumulate a trace.

## Terminal outcomes and loss

| Transition | Result and retained ownership |
| --- | --- |
| Full queue, exhausted bytes, invalid or stale submission | Explicit error; no enqueue, sink call or source replacement. |
| Cancel exact queued identity | One `cancelled` result, remove queued work; never invoke sink. |
| Cancel exact dispatched identity | One `delivery-unknown` result; retain flight until matching receipt. No guaranteed retraction. |
| Matching receipt for active work | One `received` or `failed` result and release flight. |
| Late receipt after cancellation or stop | Release that flight only; no second terminal result or revision of ambiguity. |
| Stop | Fence submissions/dispatch, cancel all queued work, mark unresolved flight delivery-unknown once. |
| Drain remains pending at step eight | Nonzero error, qualification failure latched; restart remains fenced even after late cleanup. |

Repeated cancel and stop are idempotent. Before the deadline, matching cleanup
makes a stop clean with zero owned work (queue, flight and pending bytes), while
associations and partial last-known-good evidence remain. A clean explicit start
advances generation and never replays reports. An explicit later submission must
carry a new sequence and token; it cannot turn retained stale data fresh. Old
callbacks cannot affect the newer run. Do not infer exactly-once remote delivery.

Example: dispatch sequence 1, enqueue 2 and 3, reject 4 while full. Receipt 1
releases the flight; explicit dispatch/receipt barriers then deliver 2 and 3 in
order. Alternatively cancel queued 2 and dispatched 1: 2 never reaches the sink;
1 remains delivery-unknown even after a late success receipt. Neither schedule
replays automatically or changes source freshness.

## Focused evidence and remaining gates

Run offline with worktree-local caches and disabled module/toolchain lookup:

```sh
GO111MODULE=off go test ./qualification/reportharness -run '^TestReportHarness' -count=1
```

The tests check separate dispatch/receipt barriers, independent expected FIFO and
byte counts, all identity fences, rejection without side effects, terminal
uniqueness, source-axis preservation, repeated clean stops and the deadline
negative control. Test discovery must collect a nonzero set. Mutant results and
exact command exits belong in the implementation evidence, not a library
qualification claim. Manual source/claim/transition/loss/example/link review is
also required; document token checks alone are insufficient.

Current hosted discovery registers Python lock/readiness tests and the SDK
fixture, with no Go harness invocation. Registration is a future binding-owner
task; no workflow is changed here. Full committed-head gates and independent
review remain required. Transport conformance, passive/live-fabric smoke,
hardware acceptance, interoperability and certification are unrun and unproven.
The parent remains open for actual adapter integration, library qualification,
admission controls, target closure and the remaining lifecycle scenarios.
