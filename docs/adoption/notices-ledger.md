# Adoption notices and generated-input rights ledger

Issue #61 supplies an empty evidence template for #59 direction 2. Nothing here
is stable before Helianthus 1.0. There are no populated component records or
accepted rights outcomes. **Unknown** is the initial outcome for every category;
it is not evidence of permission, prohibition or license closure.

The [existing B1–B8 contract](../go-fabric-build-qualification.md) owns qualification;
the [generic validator](../go-fabric-build-record-validator.md) checks supplied
structure only. This ledger supports B2, B6 and cross-row B7/B8 closure. It is not
a second B ledger, a legal clearance result, a dependency pin or runtime admission.
The owner-selected exact tag and actual retained closure are missing. AA08 owns
actual cache closure, hosted follow-through and producer/verifier runs.

## Required categories awaiting evidence

| Category | Outcome | Missing evidence and disposition |
| --- | --- | --- |
| Direct/transitive dependencies, replacements and native libraries | Unknown | Exact selected closure, retained source/license/NOTICE files, origin and distribution obligations; B2 missing. |
| SQLite route and transitive inputs | Unknown | Inspect actual selected imports and link inputs; reconcile any test-only notice with production storage routes, licenses and shipped notices; B2 missing. |
| Extracted Matter model, schemas and upstream generated source | Unknown | Exact input origins, extraction rights, license/attribution/redistribution conditions and mapping to generated paths; B6 missing. |
| Generators, tools and their inputs | Unknown | Tool/source versions, complete tool dependencies, input rights, invocation and output manifests, notices and independent replay; B2/B6 missing. |

## Per-component record template (unpopulated)

Copy this template only when retained public-safe inputs exist. Until then retain
the category's Unknown outcome and explicit missing reason. Do not fill guessed
versions, licenses, paths, checksums or approvals.

| Field | Required entry |
| --- | --- |
| Component and category | Public component name and category above; dependency/tool/generated-input relationship. |
| Attempt identity | Owner-supplied exact tag; source, dependency, toolchain and generator/input manifest references; target tuple and selection. |
| Retained origin | Public origin and acquisition evidence; bundle-relative source and manifest references, including patch chain. |
| Selected use | Direct/transitive, runtime/test/generation use, actual imports/link inputs and generated or embedded output paths. |
| License evidence | Retained license text, NOTICE and third-party notices; exact scope, exceptions and unresolved contradictions. |
| Obligations | Attribution, notice delivery, modification marking, redistribution and other applicable conditions; explicit evidenced non-applicability where appropriate. |
| Generated-input chain | Extracted model/schema origin, extraction inputs, generator/tool identity and dependencies, commands and output manifests; include supplied generated code. |
| Distribution closure | Which notices accompany which shipped outputs, retained notice references and comparison to actual output inventory. |
| Producer evidence | Relative artifact references, exact commands/exits/counts and missing-input reasons; compute content digests over retained bytes rather than paste literal digests here. |
| Verifier replay | Separate retained review and clean replay references for the same identity; disagreements and unresolved obligations. |
| Outcome and confidence | Initially Unknown; evidence-backed Proven/Hypothesis/Unknown claim, separately from missing/unverified/failed/pass qualification state. |
| Loss and follow-up | Missing evidence, affected B rows/children, correction as a linked new attempt, and retained earlier evidence. |

## Closure rules

Source availability, top-level Apache-2.0, upstream green metadata and literal
module checksums alone cannot establish dependency or generated-input rights.
Inspect the actual closure for each declared ABI, CGO mode, toolchain and tags;
never infer arm64 execution or license closure from source availability. A
producer assertion without independent verification remains unverified. Missing
required inputs remain missing, with obligations Unknown; dependent B2/B6/B7/B8
cannot pass. Record contradictory evidence as failed where the qualification
contract requires it. Preserve prior attempts and partial last-known-good native
state; loss of evidence never refreshes a claim or permits allocation.

This template executes no SDK or controller, accesses no device and authorizes
no live write, deployment or fabric operation. Synthetic/public fixtures and
in-memory loopback are the bounded offline scope. Independent-controller evidence,
platform execution, physical acceptance, interoperability, conformance and
certification remain separate. Production admission stays fail closed. No new
namespace or authorization gate is introduced before 0.9; roadmap wording remains
0.11. Protocol facts require their public native documentation contract first;
eBUS MCP compatibility and GraphQL parity are unchanged.
