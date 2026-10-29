# One-sensor reference bridge and adapter contract

This is the documentation companion for [#62](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/62),
a partial child of [#59](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/59).
It specifies the reference that later candidate factories must follow; it does
not report an implemented bridge. No code PR is a dependency of this document.
Nothing here is stable before Helianthus 1.0.

## Direction, authority and evidence

Helianthus adopts [go-fabric](https://github.com/SukramJ/go-fabric) as its Matter
SDK direction, using its Matter 1.6.1 data model. The selection does not qualify
an adapter or any target. Matter 1.7-only elements are deferred within the
pre-1.0 cycle. The checked-in [lock](../../sdk/matter-sdk.lock.json) and
[SDK baseline](../SDK_BASELINE.md) still describe the earlier connectedhomeip
fixture and its 1.7 targets; they are not a go-fabric lock or evidence of 1.6.1
target closure. This documentation does not change pins or grant admission.

**Proven:** the existing local source defines separate synthetic commissioning,
report and store mechanics. Focused executions prove only the exercised mechanic.
**Hypothesis:** a real SDK adapter can expose the boundaries below.
**Unknown:** real SDK completion/lifetimes, peer/controller receipt, platform
behavior and full target closure. The [runtime row ledger](../go-fabric-runtime-qualification.md#row-to-harness-and-adapter-trace)
retains missing coverage; no row becomes passing through this companion.

Protocol facts require their public native documentation contract first, then
accepted semantic evidence and target mapping in the public semantic owner.
This document supplies synthetic metadata, never a substitute native fact.
Confidence Proven/Hypothesis/Unknown, source qualification, availability,
freshness, mapping acceptance and runtime qualification are independent axes.
Keep native bytes/provenance and partial last-known-good source state when a
projection is withheld; neither retries nor SDK writes make retained data fresh.
MCP compatibility and GraphQL parity remain unchanged.

## Reference composition and mandatory metadata

Use one synthetic sensor beneath one proposed bridge, with one supplied opaque
association. This is a reference shape, not a normative device composition or
production endpoint. A second synthetic key is used only for isolation and
budget negative controls. Display electrical/thermal families, never models.

The fixture manifest must declare before execution:

| Input | Required content and rejection rule |
| --- | --- |
| Identity and history | Node, owner, profile, scoped capability, compatible composition, opaque binding and synthetic slot; retained retirement history. Missing, aliased or incompatible keys withhold publication. |
| Target facts | Explicit device/cluster definitions and revisions, Descriptor relationships, mandatory and conditional features/attributes, types, ranges, units/scales and nullability for the selected 1.6.1 composition. Missing facts stay Unknown. |
| Electrical example | Synthetic non-unknown PowerMode, positive NumberOfMeasurementTypes, matching nonempty evidenced Accuracy and evidenced nullable ActivePower, plus Descriptor and Power Topology closure. Optional ActiveCurrent alone is insufficient. These are fixture inputs, not inferred source metadata or proven 1.6.1 encodings. |
| Evidence and loss | Public-contract reference and synthetic native evidence reference/bytes; value, availability, confidence, qualification, freshness and reviewed loss separately. Preserve unavailable versus known zero and unrelated valid facts. |
| Execution | Adapter/library identity, declared completion and receipt boundaries, budget/byte accounting, injected scheduler, independent expected trace, store fault schedule and cleanup ownership. Missing instrumentation blocks qualification. |

The old lock identifies Electrical Sensor revision 2, Power Topology revision 2
and Electrical Power Measurement revision 3 under its 1.7 target. These are
historical lock values only; do not copy them into a 1.6.1 manifest as accepted
revisions. Target encoders must inspect the selected public definitions and
matching library representation. No 1.6.1 normative revision or codec is invented.

## Fixture budgets and ownership

Declare E=2 endpoint slots, A=2 pending allocations, C=4 registered callbacks,
Q=2 queued reports, F=1 in-flight report, D=8 logical drain steps and R=1 explicit
recovery attempt per explicit start. Declare a positive byte cap B before any
execution (for example B=4096 bytes in a new synthetic manifest). That example is
a fixture choice, not an existing test result or SDK/product capacity.

These are integrated reference budgets. The commissioning runner owns only one
pending callback and a 32-byte node symbol, with a two-record discovery input
limit. The store harness bounds two retained/reserved associations and two
callbacks, with B per five-string record. The report harness bounds total pending
envelope string bytes by B, with Q/F and one retained source envelope per supplied
key; retained envelopes require separate accounting up to N times B for N keys.
They do not jointly demonstrate C=4 or endpoint allocation. An integrated adapter
must count all registrations, including commissioning and store callbacks, against
C without simply adding harness maxima.

Reject oversized input before buffering; count bytes for SDK encoding, callbacks,
queued work, retained source/history and evidence traces separately. Bound the
observer's trace and fake store/sink logs too. No boundedness claim covers
uninstrumented SDK buffers, retransmission queues, callback closures, tasks or
external producers. An unobservable buffer or release boundary stays unverified.

The intent key K=(node, capability, compatible composition) is not the full
candidate review key. The factory must preserve owner/profile/scope axes in the
capability association without flattening collisions. Allocation order must not
depend on asynchronous completion. Exact duplicate intent reuses one pending
token or retained association, with no waiter growth. Retired slots remain
reserved for that K throughout the node lifecycle; neither removal nor a changed
composition permits reuse. The bridge candidate reviews retirement history;
the store harness itself implements no retirement or migration operation.

## Real and fake completion boundaries

The [commissioning runner](../fabrictest-scenario-adapter.md) has injected success
flags and an in-memory record. Its `commissioned` state is a fake sequence label.
PASE success establishes only the modeled pairing step. It does not establish
CASE, operational commissioning, discovery identity, durable fabric restoration
or controller receipt. A real adapter must independently observe commissioning
completion and a matching operational CASE session/read after validating fabric
state; a PASE/discovery callback cannot stand in for those boundaries.

Window open, successful close and revoke/cancellation are separate observations.
The fake requires successful explicit close before stop and has no revoke port.
A real adapter must show close/revoke acknowledgement, fence discovery and
commissioning callbacks from the prior window, and release registrations before
claiming quiescence. A failed close stays unresolved; teardown may not assume
revocation. Unsupported or unobservable revocation remains missing evidence.

The [report harness](../go-fabric-report-mechanics-harness.md) separates Submit,
Dispatch and Complete. Queue acceptance, attribute mutation and dispatch return
are not receipt. A receipt must be actor-independent: an observer outside the
submitting adapter must identify the actual receiving boundary and match node,
capability, composition, generation, token, sequence and native evidence reference.
The fake sink's injected boolean proves no peer receipt. A real SDK that coalesces
latest attributes does not provide a FIFO observation receipt. Either demonstrate
the declared observation contract at a separately instrumented boundary or leave
that runtime row unverified; never reinterpret latest-state delivery as delivery
of every accepted observation.

Queued cancellation prevents dispatch. Cancellation after dispatch returns
delivery-unknown until ownership drains; late success cannot revise that terminal
ambiguity or trigger replay. Each completion cleans only its exact operation once.
Old generations cannot mutate a restarted run. Logical step eight with outstanding
resources latches failure: late cleanup does not make restart safe or clear the
failed qualification. A clean stop has zero owned tasks, registrations, queue and
flight work while retaining valid source evidence and durable associations.

The [store harness](../go-fabric-store-mechanics-harness.md) separates write,
completion and exposure. Precommit failure exposes nothing. A committed but
unconfirmed write remains ambiguous; no blind retry, new identity or exposure is
allowed. Confirmed completion after stop also requires reconciliation. Explicit
recovery validates an authoritative snapshot, exact key/binding/slot and unrelated
records before exposure; absent/corrupt/conflicting state stays withheld. R=1
means no hidden retry loop. The in-memory fixture proves no disk durability or
cross-process recovery.

Run a separate control through the actual [readiness evaluator](../../runtime/production_readiness.py)
with accepted-looking metadata and an allocation spy. The current lock has no
accepted composition-profile digest: zero production allocation is mandatory
even if every isolated mechanic succeeds. A structural fake allocation counter
cannot replace this evaluator-boundary control.

## Candidate factory validation matrix and execution order

AA30 can connect a candidate only after the one-sensor reference has explicit
evidence for its adapter boundaries. This is an implementation order, not a
code-PR dependency for this documentation ticket. Run reference first, then
Thermostat/HVAC, then EEM, then EPM; the bridge history validator constrains every
stage. Candidate arithmetic/labels are inputs to a separately reviewed encoder,
never upstream APIs or already encoded Matter attributes.

| Stage / local candidate | Finite positive control | Required negative and isolation cases before connection | Encoding/execution boundary still required |
| --- | --- | --- | --- |
| Reference + [bridge](../contribution-candidates/bridge.md) | One sensor, preserved association and active acyclic edge; clean stop/restart | Each node/owner/profile/capability axis independently; composition/binding/slot continuity; retired-slot reuse/reactivation/history loss; duplicate edge, missing parent, cycle; late completion and allocation-denial control | Actual Descriptor/bridge composition, SDK allocation/store/teardown mapping; no production admission |
| 1. [Thermostat/HVAC](../contribution-candidates/hvac.md) | 21.5 C measured-room; separately evidenced heating/cooling target and complete selected-mode codebook | Every seven-field key axis; nil versus zero, nonfinite values, bounds, Unknown capability, incomplete/unselected codebook entry, loss, operation direction and valid sibling retention | Inspect 1.6.1 temperature scale/range/null and mode enums; encode only evidenced observation roles; no inferred setter |
| 2. [EEM](../contribution-candidates/eem.md) | All four import/export and cumulative/periodic combinations; known complete 7 times 1000 gives 7000 Wh | Owner/profile/fact/scope/direction/reset epoch and each interval bound; structurally valid kind/interval tuple; unknown value/continuity, partial interval, zero, overflow and sibling detachment | Inspect energy units/scales and nullable/interval/reset representation; local whole-Wh convention is not a target wire format; never sum periods into totals |
| 3. [EPM](../contribution-candidates/epm.md) | Signed -250 W, exact 12.345 W to integer 12345 at scale -3, known zero and unavailable | Every Owner/Profile/Source/Scope/Phase/Fact axis; duplicates, precision loss, overflow, unavailable-with-value, unit/scale mismatch, independent valid phase/sibling | Inspect selected power units/scales/ranges/null, Accuracy, measurement count, PowerMode and transitive composition; arithmetic cannot synthesize mandatory metadata |

For each stage, retain separate structural-vector, target-encoding, real-adapter,
receipt, store/restart and teardown results with pass/fail/missing/unverified
status and nonzero execution counts. Vary identity fields independently while
other fields remain valid; where structure couples fields (EEM kind/interval),
record the valid tuple case separately. A valid local vector does not satisfy
the encoder or runtime column. Missing retained inputs stay missing. Failed
traces stay failed; later success does not erase them.

No device access, live commissioning, live Gateway call, real-bus slave answer,
deployment or production eBUS arming switch is introduced. Only synthetic/public
fixtures, in-memory loopback and injected time are in scope. No conformance,
certification, interoperability, Linux arm64, Pi performance, latency, transport
or physical acceptance follows. No authorization gate is introduced before 0.9;
authorization roadmap work remains 0.11. Actual peer/network/controller evidence,
independent review and applicable repository gates remain separate.
