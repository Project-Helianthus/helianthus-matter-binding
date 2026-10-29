# Isolated fabric-store mechanics

Issue #44 implements the fabric-store portion of the proposed Z20 scenarios in
[the runtime qualification contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/go-fabric-runtime-qualification.md).
It is a partial slice of #24 / Project #2; the parent remains open. Nothing here
is stable before Helianthus 1.0.

## Scope and evidence

The [harness](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/qualification/storeharness/harness.go)
uses an injected store and supplied opaque associations. `MemoryStore` is a
synthetic in-memory fault fixture. Its independent `Unrelated` map represents
shared opaque records; no store operation removes them. There is no storage
backend, endpoint allocator, selection-intent persistence owner or go-fabric
adapter. Report queue mechanics remain in the separate report harness.

**Proven:** focused synthetic tests exercise the transitions below, including
identity fences and unchanged sibling records. **Hypothesis:** a future adapter
could expose corresponding durable-write and completion barriers. **Unknown:**
actual library callback lifetimes, durability, recovery guarantees and resource
behavior. These labels do not upgrade native qualification, freshness, runtime
readiness, physical acceptance, interoperability or Matter certification.

The [Portal lifecycle contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/PORTAL_ENDPOINT_LIFECYCLE.md)
continues to own the persistence design boundary. No protocol facts are added;
those require their public native contract before semantic or target mapping.
This fixture stores no native evidence and cannot modify or promote source
values, availability, confidence, qualification, freshness or provenance.

## Logical barriers and bounds

The caller drives the harness serially. Store methods return promptly and must
not reenter it. `Start` reads once per explicit attempt; no retry, background
worker, timer, listener or unbounded history is created. Store implementations
are fixture dependencies, not trusted production storage adapters.

`Begin` takes the complete supplied key (node, capability, composition) and
association (binding, synthetic slot). It invokes one store write and returns a
completion held by the caller. An identical pending request returns the same
completion without another write or waiter. Only `Complete` releases that
barrier. A completion must equal the retained request and outcome, including
node, capability, composition, generation, token, binding and slot. Tokens
increase across this harness instance; successful starts increase generation.

The bounds are two retained/reserved associations and two pending callbacks.
A caller-declared positive byte cap bounds each record's five strings; thus
record payload is at most twice that cap per bounded map. No callback history
is retained. These are fixture limits, not SDK limits. Oversized records and
exhausted budgets are rejected before writes. Injected input/store snapshots
are caller-owned; the fixture validates them but does not claim to bound an
external store's memory or execution time. Map-returning observers and reads
return detached copies. The harness has no endpoint allocation operation.

## Transitions, recovery and loss

| Barrier or fault | Exposure and retained state |
| --- | --- |
| `Start` read fails, or snapshot is missing/invalid | Start fails, no generation advances or exposure occurs; accepted associations survive. No implicit retry. |
| Valid start | Restore only validated supplied records; unresolved commits stay withheld. An absent accepted record remains retained, without exposure. |
| Before `Complete` | New association has zero exposure, even if the fake store already committed it. Existing siblings remain unchanged. |
| Precommit failure | Fake store does not change its association map; completion exposes nothing. A later attempt must be explicit and gets a new token. |
| Confirmed completion while running | Accept and expose exactly the supplied association. This is mock exposure, never production admission. |
| Ambiguous completion | Fake store records the association, but the harness neither accepts nor exposes it. Automatic replacement or retry is prohibited. |
| Explicit `Reconcile(snapshot)` | With no pending callbacks and while running, validate the entire supplied snapshot before changing state. Restore original validated associations; absent unresolved entries stay withheld. |
| Invalid or conflicting reconciliation | Change no accepted or exposed state. Existing last-known-good siblings survive. |
| `Stop` | Immediately clear exposure and fence new work; retain associations and pending callback ownership. Repeated stop is idempotent. |
| Completion after stop | Release only its exact callback. A confirmed write is unresolved until explicit reconciliation; it cannot publish. |
| Drain | Caller must release callbacks before logical step eight. Deadline failure permanently fences restart, even if a later callback releases bookkeeping. |
| Restart | Read once, advance generation only on success, restore safe records; unresolved entries still need explicit reconciliation. Old completions cannot consume current callbacks. |

Snapshots must contain all five nonempty record fields, fit the bounds and avoid
binding/slot collisions with both supplied and retained records. A snapshot
cannot replace an already accepted or unresolved binding or slot. An invalid
snapshot is rejected as a unit, preserving all accepted state. An empty snapshot
is valid absence: it withholds exposure and does not erase retained records.
This conservative fixture does not implement repair, migration, retirement,
node reset, cross-instance crash metadata or fabric recovery. A caller starting
a new harness must supply authoritative validated storage evidence; no claim of
recovering a process crash or independent external store corruption is made.

For example, supply synthetic metering-family keys `(n-a,c-a,p-a)` and
`(n-a,c-b,p-a)` with distinct opaque associations. An ambiguous write for the
first leaves only the second exposed. A restart still withholds the first.
Explicit reconciliation with both original associations exposes both in the
mock model. A replacement binding or slot is rejected and the second survives.
No controller, credential, fabric token or real endpoint number is involved.

The injected `SDKStore` seam also permits mock withdrawal after an ambiguous
completion. Withdrawal requires the endpoint's pending callback to complete and
retains its identity, reservation and fabric records. The adapter remembers
withdrawal for accepted or uncertain identities, so later exact reconciliation
and restarts of the same harness and adapter do not expose that endpoint.
Before committing a new key, the adapter rejects binding or slot collisions
against every retained record, including withdrawn endpoints hidden from a new
harness using the same adapter. A distinct association remains permitted within
the retained-record budget.
This intent is instance-local; it does not establish cross-process recovery or
authorize any live fabric operation.

## Validation boundary

Run discovery with `GO111MODULE=off go test ./qualification/storeharness -list .`
and the focused check with
`GO111MODULE=off go test ./qualification/storeharness -count=1`.
The tests independently vary node, capability and composition for distinct keys;
and node, capability, composition, generation, token, binding, slot and outcome
for completion rejection. Separate snapshot variants cover each empty field,
changed binding/slot, collisions, absence and missing state. Fault schedules
include clean, precommit-failed and ambiguous restarts, read failure, reordered
callbacks, stale retries, duplicate completion and drain timeout.

Each completion-field variant targets omission of that field from lookup and/or
completion equality. Ambiguity targets treating a committed-but-unconfirmed
write as confirmed; sibling controls target clearing unrelated store records or
accepted associations during a local operation. Executed mutation results are
recorded with the focused run; this documentation does not assert exhaustive
mutation coverage. A finite test watchdog diagnoses hangs only; logical steps
supply the fixture ordering, not elapsed time.

Manual source/claim/transition/loss/example/link review is required. Links above
identify public source locations and were reviewed against local supplied source
content, without fetching moving main. Passing this fixture does not supply a
production composition profile or change the current fail-closed readiness
rule. Full CI, independent review, adapter integration, native qualification,
transport conformance, passive/live-fabric smoke, hardware acceptance and parent
acceptance remain outside this run. No T01..T88 transport matrix is declared.
No MCP or GraphQL envelope/parity, G5 D1-D5/O8, or eeBUS ownership changes occur.
No authorization gate is introduced; roadmap authorization remains 0.11 work.
