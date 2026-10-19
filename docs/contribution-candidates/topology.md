# Internal topology contribution candidate

This is an **internal contribution candidate for the binding's own use and
later review**, implementing a partial slice of [#38](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/38)
and advancing [#24](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/24).
The parent stays open. It is never an upstream proposal, fork or patch
submission. Nothing here is stable before Helianthus 1.0. Executable validation
does not establish that go-fabric supports Power Topology.

## Source basis and evidence boundaries

The [metering inventory](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/metering.md)
separates source representations from physical topology: text dimensions,
counters, units, names and shared addresses cannot establish wiring or endpoint
associations. The [inventory completeness contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md)
requires native owner/profile and fact scope to remain distinct and missing
evidence to block closure without erasing native or partial last-known-good
state. These supplied contracts motivate the local rules below; they do not
supply a real graph or normative Matter graph rules. Public main links are
locators, not evidence that remote pages were checked during this offline work.

| Claim | Evidence status | Separate outcome |
| --- | --- | --- |
| The standard-library validator checks the supplied synthetic graph under the rules below | Proven only by focused executable vectors when they pass | Native source qualification remains unverified |
| These source-local rules might help a future binding implementation | Hypothesis; requires binding review | Semantic mapping acceptance remains open |
| Exact Matter 1.7 topology attributes, feature conditions and endpoint composition | Unknown in the supplied source views | Target closure remains open |
| go-fabric target definitions and implementation support | Unknown; no library dependency or invocation | Library support remains unverified |
| A valid graph permits production allocation | No such transition exists here | Existing fail-closed admission remains unchanged |

Before using protocol facts, obtain their public native documentation contract
at its owning repository, then qualification and accepted semantic evidence.
For target closure inspect the exact reviewed-lock Matter 1.7 corpus, resolving
Power Topology and Descriptor files, transitive types, device requirements,
feature conditions and mandatory composition. Reconcile the accepted mapping
at its public semantic documentation owner. For library support require actual
go-fabric declarations, revision compatibility, implementation and focused build
evidence for that composition. No numeric cluster IDs, feature support, phase
enumerations or production admission rules are invented here.

## Local executable contract

[Validate](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/candidates/topology/candidate.go)
accepts a finite `Graph` value containing explicit `Sources`, `Nodes` and
directed `Edges`. There is no discovery, provider interface, gateway integration,
SDK call or dependency on another contribution-candidate ticket.

* A `Source` tuple is `(Owner, Profile, Scope)`. Scope is an opaque supplied
  service/measurement scope. A node identity is `(Source, ID)`. Declarations
  must be unique and every node must use an exactly declared source.
* Every identity label must be nonempty valid UTF-8 with no leading/trailing
  Unicode whitespace or control characters. Labels are case-sensitive; no
  normalization, parsing, address interpretation or separator concatenation
  is used. A missing identity is invalid, never an anonymous shared source.
* Each node supplies `Phase` and `Endpoint` labels. Empty means Unknown and
  withholds the candidate; other malformed labels are invalid. Literal `"0"`
  is a known opaque label. Neither field is an enum, physical claim, numeric
  Matter endpoint nor an allocation request.
* Within each source, one phase must associate with exactly one endpoint and
  one endpoint with exactly one phase. Distinct nodes can repeat the same
  association. The rules deliberately describe this local candidate shape;
  they do not assert normative Matter cardinality. Aggregate or shared-phase
  target shapes need a separately reviewed representation, never invented
  phase labels. Independent sources may reuse either label freely.
* Edge identity is the ordered pair `(From, To)` of full node identities. Both
  references must exist and have identical source tuples. Duplicate edges,
  self-loops and directed cycles, including disconnected cycles, are invalid.
  Acyclic fan-out, fan-in and isolated nodes are allowed; neither a tree nor
  physical flow direction is inferred. Edge relations do not propagate or
  manufacture phase/endpoint labels.

`Valid` means local structural success only. It returns a detached graph,
sorted by owner/profile/scope/ID and then full edge endpoints. `Invalid` means
a malformed or conflicting supplied graph. `Withheld` means a required graph,
phase or endpoint fact is absent. No sources or no nodes is withheld, never
an empty successful topology. Invalid dominates withheld if both apply, while
all detected reason codes remain visible, unique and lexically sorted.
Rejection exposes no candidate, so there is no partial successful projection.
Duplicate or absent evidence is never silently dropped into a valid graph.

The validator never mutates the input or retained state. A caller must keep
native evidence and unrelated last-known-good values, marking loss or stale
state explicitly; a rejected graph supplies no replacement or freshness
signal. This pure function has no storage, receipt clock or freshness model.
Validation uses iterative cycle detection with work proportional to supplied
nodes and edges, plus output sorting; it requires no listeners or recursion.
It validates finite in-memory values, not an unbounded ingestion stream.

## Synthetic examples and mutation review

A synthetic metering family can explicitly declare source
`(synthetic-owner, meter-family, service-a)` and nodes `a`, `b`, `c`, each with
its supplied matching phase/endpoint labels. Supplied edges `a -> b` and
`a -> c` form a valid local candidate. Adding `b -> c` is also valid. Adding
`c -> a` makes a cycle. Referencing missing node `d` is invalid. Supplying
`a` with an empty phase withholds the whole graph; it does not delete node `b`
from the caller's retained evidence. Assigning the same phase to different
endpoints, or one endpoint to different phases, is invalid within that source.
Equal labels in a separately declared source do not create a shared topology.
All identifiers here are synthetic; no installed family capability is claimed.

[Tests](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/candidates/topology/candidate_test.go)
use independent expected graphs and reason codes. The following is the
test-to-mutation mapping; an executed mutation campaign is reported separately
and must not be inferred merely from this table.

| Test variants | Exact production mutation they must detect |
| --- | --- |
| `TestIdentityAxesRemainDistinct/owner`, `/profile`, `/scope`, `/id` | Drop that one field from the node identity key, falsely merging distinct nodes |
| Owner/profile/scope variants of the same test | Drop that field from source declaration keys or phase/endpoint association scope, falsely merging independent sources |
| `TestScopedEdgeKeys`, separate owner/profile/scope variants; valid fan-in/fan-out vectors | Drop the named source field from both ends of an edge key, or drop From/To local ID, falsely merging distinct edges |
| `TestReferenceAxes/from` and `/to`, each with owner/profile/scope/id | Ignore that field in reference lookup, accepting a dangling reference |
| `TestMalformedIdentityAxes`, every owner/profile/scope/id and empty/space/padding/control/invalid-UTF-8 variant | Remove the named identity label guard, admitting malformed keys |
| Malformed reference variants for both ends and every identity field | Omit edge identity validation or treat an invalid reference as resolved |
| `TestGraphGuards/cross-source`, owner/profile/scope variants | Drop that source comparison axis, accepting a cross-source relation |
| Duplicate source/node/edge variants | Remove the corresponding duplicate rejection |
| Self-loop, cycle and disconnected-cycle variants | Remove or restrict cycle detection to one connected component |
| Phase-conflict and endpoint-conflict variants | Remove the phase-to-endpoint or endpoint-to-phase contradiction guard |
| Unknown phase/endpoint and `TestWithheldEmpty` variants | Default missing facts, accept empty topology or expose a candidate on withholding |
| `TestAssociationValueAxes`, each phase/endpoint malformed variant | Omit label validation for that association field |
| `TestPermutationAndDetachment`, `TestProblemPermutation` | Return aliased or input-ordered slices, nondeterministic/repeated reasons, or mutate retained input on rejection |

Focused offline commands are `GO111MODULE=off go test ./candidates/topology -list .`
for nonzero discovery and `GO111MODULE=off go test ./candidates/topology -count=1`
for execution. They establish neither full CI nor native qualification, semantic
acceptance, target closure, runtime integration, transport behavior, passive
capture, hardware or live-fabric acceptance. No MCP or GraphQL path changes;
their parity and existing admission boundary are preserved. No authorization
gate is introduced; later authentication/authorization planning remains 0.11.
eeBUS output remains in its registry owner. Independent review and remaining
parent integration/physical acceptance are separate and unrun here.
