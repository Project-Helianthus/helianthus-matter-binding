# Default-off gateway library composition

This binding-owned companion for issue #64 advances direction 4 of #59. It
defines the composition and lifecycle requirements for an optional in-process
Matter library host. It documents the AA17 gateway specification; it does not
assert that the host, its launcher, or a production SDK backend is running.
This ordinary documentation slice has no dependency on the AA17 code PR.
The in-process-versus-separate-process strategy decision remains owner work
outside this repository.

## Adoption and the current boundary

The owner has adopted go-fabric as the SDK direction, using its Matter 1.6.1
data model. Matter 1.7-only elements are deferred. Adoption does not establish
runtime qualification, target closure, interoperability or certification.
Nothing here is stable before Helianthus 1.0.

The retained [SDK baseline](../SDK_BASELINE.md),
[lock](../../sdk/matter-sdk.lock.json) and
[readiness evaluator](../../runtime/production_readiness.py) still describe the
earlier SDK baseline. This document does not migrate those artifacts, repurpose
their build results as go-fabric evidence, or relax their checks. The evaluator
is an evidence decision only: its reviewed lock has no accepted composition
profile pin, so every current invocation denies allocation. The adopted SDK
direction and retained executable admission baseline must be reconciled through
separately reviewed binding work before any production launcher can exist.

Source provisioning and consumer pinning are separate unresolved boundaries.
No real go-fabric source, adapter linkage, selected consumer dependency pin or
matching retained qualification results are established by this companion.
Providing source alone cannot qualify an adapter or select its consumer pin;
a selected pin alone cannot establish runtime behavior. The real factory link
and consumer pin require the separately owned AA12/AA30 results. Until those
inputs and all admission evidence exist and match, enabled means unavailable.
Missing retained inputs remain missing or unverified.

## Composition states and factory boundary

The following are contract states, not newly defined MCP or GraphQL fields.

| State | Required behavior | Allocation and evidence limits |
| --- | --- | --- |
| Disabled | The explicit in-process option defaults to false. Disabled wins over retained intent or available fixture evidence. | Zero binding factory calls, SDK allocations and fabric-store allocations. No launcher, listener, commissioning or report dispatch. |
| Unavailable | The option is enabled but the factory link, consumer pin or required evidence is absent, incomplete, failed or mismatched. Expose the reason. | Zero factory calls and zero SDK/store allocation; no fallback sample, permissive adapter or automatic retry into production. |
| Fixture-qualified | A declared synthetic host/adapter fixture has passed its bounded checks for its exact inputs. | Only the fixture boundary is qualified. Mock allocation is isolated from production admission; this state cannot enable a real launcher. |

AA17 specifies a narrow injected binding factory and a shared snapshot/evaluation
port. It binds the option on an explicit flag set and supplies the library owner
to the ordinary gateway lifecycle; it must not register through initialization
side effects. These are requirements, not invented go-fabric API names. AA27
owns the separate gateway entry-point integration. No default feed change,
native write route or Portal publication follows from composing the host.

The required ordering is:

1. Resolve the default-false option before constructing the binding or its store.
2. When enabled, validate complete retained tag, build, runtime-qualification
   and accepted target-profile evidence for the exact selected SDK tag, target,
   profile and evidence identities. Check source, binding and native-driver
   generations and the lifecycle token; stale or mismatched inputs cannot pass.
3. Apply the binding's reviewed admission decision before any production
   allocation. Under the current lock this always denies. Caller-supplied
   accepted-looking metadata, a successful fixture or a successful build cannot
   replace a reviewed profile pin and matching mandatory target facts.
4. Only a future qualified and admitted integration may invoke the real factory,
   allocate SDK resources and start reporting within its declared limits.
   There is no production launcher before qualification and admission.

The [Portal lifecycle contract](../PORTAL_ENDPOINT_LIFECYCLE.md) remains the
authority for the exact current evaluator prerequisites. Its required
Descriptor, Power Topology and Electrical Power Measurement facts cannot be
inferred from optional ActiveCurrent, a sample endpoint or library defaults.
Any future adopted-baseline replacement needs its own reviewed target facts
and matching build evidence; the older fixture does not supply them.
Qualification and admission are evidence boundaries. This adds no principal,
credential or authorization gate before 0.9; authentication and authorization
remain 0.11 roadmap work.

## Shared semantic snapshots

The host consumes the shared semantic snapshot/evaluation port through the
AA03 projector. It must not round-trip through HTTP, create a second semantic
truth store, import vendor decoding or reinterpret raw protocol values. The
same detached snapshot and evidence decision must underpin equivalent MCP,
GraphQL and Matter projections; preserve eBUS MCP compatibility and GraphQL
parity. This companion changes no public envelope or namespace.

Preserve value, availability, freshness, confidence, qualification and native
provenance as independent facts. Proven / Hypothesis / Unknown describe claim
support; they do not establish native qualification or Matter admission.
Protocol facts require the public native documentation contract first, then
accepted semantic evidence and the public Matter target mapping. This page
introduces no protocol facts or mappings. See the
[inventory completeness contract](../target-profile/inventory-completeness.md).

Disabled, unavailable, evidence loss and teardown must preserve native evidence
and partial last-known-good semantic state. Retained values must not become
fresh through restart, retry or projection. Withholding one affected capability
must preserve independently valid records and must not select a substitute
target. Reporting cannot upgrade an unknown or unqualified fact.

## Generation-fenced lifecycle

The [runtime qualification scenarios](../go-fabric-runtime-qualification.md)
define the required asynchronous and resource boundaries. Each successful start
has a fresh generation; callbacks must also match their capability/composition
key and operation token. Source, binding and native-driver generations remain
independent checks. An old success or failure cannot publish into a successor
generation, restore a retired registration or overwrite newer evidence.

Stop first fences new work and callbacks, cancels queued work and drains owned
in-flight operations within a declared finite budget. Join tasks and release
registrations and resources before declaring stop complete or allowing restart.
A stale callback may release only its own bookkeeping exactly once. Failure to
quiesce is a qualification failure: keep the old generation fenced and prohibit
successful restart while its resources remain unsafe. Successful stop leaves
zero owned tasks, registrations, queued items and in-flight work. Repeated stop
is idempotent; duplicate start cannot allocate another running host.

Report dispatch and receipt are distinct. Cancellation after dispatch may leave
delivery unknown; do not claim retraction, exactly-once delivery or automatically
replay an ambiguous report after restart. Preserve evidence and loss disposition.
Fixture budgets and fake-sink acknowledgements are not library resource limits
or proof of controller receipt.

Endpoint identity, selection persistence and node-level fabric storage follow
the Portal lifecycle contract. Do not fabricate identity or fabric material from
selection, cached values or fixture tokens. Corrupt or ambiguous restoration
withholds affected publication while retaining valid unaffected records. No
automatic commissioning or rejoin occurs on start, restart, enable, evidence
recovery or fabric removal. Any future real commissioning still requires the
existing action-time operator confirmation in [the safety rules](../../AGENTS.md);
that requirement is separate from qualification and adds no new authorization
gate. No live action is authorized here.

## Evidence and remaining work

Offline acceptance of this document consists of manual comparison against the
Portal lifecycle rules, runtime qualification scenarios and AA17 specification,
plus a standard-library local-link check. It is documentation evidence only.
AA17 calls for isolated controls proving disabled and incomplete-evidence zero
factory/allocation behavior, rejection of stale callbacks, joined teardown and
three start/stop cycles with zero owned resources. Those are required code
checks, not results claimed by this page.

Focused synthetic fixtures, in-memory loopback and injected logical time cannot
establish real SDK, hosted or platform qualification. Linux arm64 support, Pi
performance, latency and physical behavior remain Unknown without matching
evidence. Transport conformance, live-fabric smoke, interoperability and Matter
certification are not established. There is no real-bus transmit path or
production eBUS arming switch introduced by this documentation; answering as a
slave on a real bus would be a live write and is outside this work.

Real factory linking, consumer pin review, retained qualification results,
reviewed target admission and independent integration review remain required.
Selective Portal publication, persistence/UI implementation and closure of #4
remain outside this slice. This does not close the #59 umbrella or establish
release or physical acceptance.
