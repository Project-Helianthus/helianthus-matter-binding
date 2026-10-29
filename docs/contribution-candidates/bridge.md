# Internal Bridged endpoints contribution candidate

This is executable internal contribution-candidate logic for the binding's own
detached association review and later review under issue #40, a partial slice of
#24. It is never an upstream proposal, fork or patch submission. Nothing here is
stable before Helianthus 1.0. It does not claim go-fabric supports Bridged
endpoints or establish Matter target semantics.

The source contracts are the [Portal lifecycle](../PORTAL_ENDPOINT_LIFECYCLE.md)
and [inventory completeness](../target-profile/inventory-completeness.md).
The [validator](../../candidates/bridge/candidate.go) uses only the Go standard
library. It does not depend on another contribution candidate or invoke the
production readiness evaluator.

## Inputs and structural decision

`Review(previous, proposed, edges)` takes complete caller-supplied snapshots,
not deltas. An association contains opaque `Node`, `Owner`, `Profile`, scoped
`Capability`, `Composition`, `Binding`, positive bookkeeping `Slot`, and
`Retired`. These identifiers are detached synthetic review data, never fabric
identities, network coordinates or actual device identifiers. Slot positivity
is a local review rule, not a normative Matter endpoint range. Composition is
an opaque equality token, not an accepted profile digest or feature definition.

All string fields must be nonempty, unpadded and free of NUL, tab and line
breaks. The validator does not normalize identities or parse capability scope;
the caller must include owner-qualified service, channel or phase distinctions
in the capability token. Absence is invalid; equal unknown values cannot become
a match. An empty composition yields `composition_unknown`.

| Key or comparison | Review rule |
| --- | --- |
| Node, owner, profile, capability | Unique source association within a snapshot; every history key must survive. |
| Node, binding | Unique opaque binding within a node. |
| Node, slot | Unique slot within a node, including retired records. |
| Composition, binding and slot for an existing source key | Must remain equal; changes are withheld for explicit migration or retirement review. |
| Retired | Active may become retired; retired cannot become active or disappear. |
| Edge node, parent binding, child binding | Both endpoints must exist in that node and be active; no duplicate edges, multiple parents or cycles. |

Separate node lifecycles can reuse tokens and slot numbers; substituting a new
node for history in a single review is history loss. Display labels and incoming
values are not inputs. The graph is a proposed forest with zero or more roots;
no mandatory root kind, parent feature, cluster ID or normative topology rule is
inferred. Graphs are complete proposals, not persisted graph histories; changing
parents across reviews is not validated as a runtime migration. Prior graph
continuity would need its own reviewed contract and evidence.

`Retained` is a detached copy of the supplied previous associations, even when
they are malformed. It must not be called validated last-known-good storage.
Any malformed history, loss, alias, unknown composition, reassociation,
composition change, retirement reversal or invalid graph withholds **all**
`Candidate` and `Graph` output. `Issues` gives deterministic, sorted, deduplicated
diagnostics; cycle detection uses a graph-wide diagnostic. Valid output sorts
associations by node/binding and edges by node/parent/child. Retained records
keep their supplied order. Every returned slice is independent of caller slices.
An empty review has no issues and no associations; it grants no runtime rights.

This conservative whole-proposal withholding preserves every prior association
without choosing winners in ambiguous data. It performs no allocation,
persistence, native-state mutation, commissioning or reporting. Native evidence
and partial last-known-good semantic state remain with their existing owners;
the validator neither erases them nor presents retained values as fresh.

## Synthetic example and loss cases

A synthetic electrical family supplies a source tuple
`(synthetic-node, synthetic-native, electrical-family, service-a.power)` with
composition `review-composition-a`, binding `opaque-a` and slot `1`. Adding
`service-b.power` as `opaque-b` at slot `2` and proposing `opaque-a → opaque-b`
passes structural review. These are invented equality tokens, not native facts
or evidenced mappings. Changing only owner, profile, capability or node creates
a distinct source key and cannot replace the old record. Reusing slot `1` for
another capability aliases the retained association even if it is retired.
Changing composition with the same binding is withheld, not silently migrated.
Dropping a record is `history_lost`; retiring it preserves the association.

The [tests](../../candidates/bridge/candidate_test.go) contain independent
expected associations, edges and diagnostics. They vary each source-key axis
independently, both uniqueness axes, binding/slot/composition continuity, every
string field's malformed values, nonpositive slots, retirement and graph
references. Permutations preserve the decision; detached copies prevent caller
and result mutation from crossing the boundary. Disposable mutation evidence
belongs to local review evidence, not a library-support claim.

## Evidence boundaries and required follow-up

**Proven:** only the synthetic structural behaviors exercised by the focused
tests. This does not qualify a native source or establish a semantic mapping.
**Hypothesis:** this local association model may be useful for later SDK-boundary
integration review. **Unknown:** selected Matter 1.6.1 Bridged target closure,
required device/cluster revisions, feature conditions, endpoint constraints and
go-fabric library support. No normative IDs or feature support are supplied.

Required evidence remains inspection of the selected 1.6.1 target definitions
and mandatory dependency closure, reconciled with the earlier lock's deferred
1.7 target; public native contracts first,
then inspected native source and accepted semantic evidence; accepted target
mappings at their public semantic owner; and library APIs plus matching build,
graph/lifecycle, persistence/recovery and reporting evidence for that target.
These are separate from source confidence, native qualification, freshness,
composition acceptance, production admission and physical acceptance. Missing
evidence stays Unknown, never a default target or proof of impossibility.

The [readiness boundary](../../runtime/production_readiness.py) is unchanged:
the reviewed lock has no accepted composition-profile digest and denies
production allocation. An issue-free candidate cannot override that boundary.
No eBUS MCP or GraphQL interface changes occur, preserving their existing parity.
No authorization gate is introduced before 0.9; any new authorization roadmap
work remains a 0.11 subject. No transport matrix, runtime integration, native
qualification, certification, interoperability, physical or live-fabric smoke
result is claimed. This slice does not close parent #24.

## Reference factory admission boundary

The [one-sensor reference and finite factory matrix](../adoption/reference-bridge.md#candidate-factory-validation-matrix-and-execution-order)
is the runtime connection contract under #62 / #59. The adopted SDK direction
uses Matter 1.6.1; the earlier lock's 1.7-only elements are deferred, not
implicitly ported. This changes no pin or executable validator. Reference first,
then Thermostat/HVAC, then EEM, then EPM; bridge history review applies at every
stage. A passing Review alone cannot connect or allocate an endpoint.

The factory must retain the complete source key Node/Owner/Profile/Capability
when mapping to the runtime node/capability/composition key. Never discard scope
or provenance to make those keys match. Composition, Binding and Slot continuity
and Retired history survive shutdown/restart: a retired slot remains reserved
within that node lifecycle. Reusing it for another key, dropping history or
reactivating it is withheld, including after an ambiguous store outcome.

Validate one active sensor plus an independent sibling, then each source-key
axis, each continuity field, retirement loss/reversal/reuse, graph parent/child
references, duplicates, multiple parents and cycles. Expected associations must
be independent of the factory's key builder. Actual SDK graph composition,
store durability, allocation callbacks and teardown remain Unknown. The runtime
row ledger retains all missing/failed/unverified evidence and the separate
zero-production-allocation control. No new test or executed mutant is claimed.
