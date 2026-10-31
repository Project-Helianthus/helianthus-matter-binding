# Offline go-fabric build qualification evidence

Issue #61 retargets the existing issues #25 and #33 qualification contract for
#59 direction 2 / Project #2. The owner has selected go-fabric as the Matter SDK
direction now, on its Matter 1.6.1 data model; Matter 1.7-only elements are
deferred. Selection is not build qualification or production admission. This
document neither changes an SDK pin nor completes #59 or the earlier #24 work.
Nothing here is stable before Helianthus 1.0. This is a document review
contract, not an implemented report parser or runtime interface.

## Current authority and explicit unknowns

The [binding ownership rules](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/AGENTS.md)
place SDK selection, allocation and integration at this boundary. The
[reviewed lock](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/sdk/matter-sdk.lock.json)
and [SDK baseline](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/SDK_BASELINE.md)
still describe the existing connectedhomeip implementation and bounded sample
fixture. They have not been retargeted by this documentation slice and do not
supply go-fabric build evidence. The owner decision in #59 supersedes their SDK
direction; AA08 owns the actual cache closure, pin/workflow follow-through and
producer/verifier runs. Existing build targets are not go-fabric qualification
targets. No conformance or certification follows from the selection.

The [readiness evaluator](../runtime/production_readiness.py) checks the reviewed
lock and denies allocation when `accepted_runtime_composition_profile_sha256`
is null, before trusting caller composition or build metadata. This contract
leaves that behavior and the lock untouched. Even a fully passing report below
cannot admit a production endpoint. The
[inventory completeness contract](target-profile/inventory-completeness.md)
separately requires source semantics, target closure and accepted mapping evidence.
Build results supply none of those missing facts by default.

The exact go-fabric tag must be supplied by the owner for the adoption attempt.
It is **missing** here: do not derive it from an old example, upstream green
metadata, a moving branch or a research comparison. Record one canonical exact
stable release (`vMAJOR.MINOR.PATCH`), never `latest`, a range, wildcard,
prerelease or build-metadata alias. Match it across all retained inputs and both
roles. A release label or literal module checksum supplies neither authentic
source bytes, successful compilation, rights closure nor target support.

Linux `amd64` and Linux `arm64` are mandatory adoption targets, not supported
platform claims. Each needs its own explicit OS, architecture, ABI, CGO mode,
compiler, linker, tags/features and exact toolchain manifest, plus both roles'
B4–B6 children. No host defaults fill these fields. Native execution means tests
actually run on the declared architecture; record the execution environment and
its architecture evidence. Cross-compilation, including emission of a test binary,
is only B4 evidence. Emulated execution must be labelled separately and does not
substitute for the required native run. An amd64 host result never proves arm64
execution, Pi performance or latency. A changed tuple or release starts a new
assessment without inherited passes.

No real go-fabric source tree, retained module closure or owner-selected tag is
supplied for this slice. Source-derived APIs, exact test identities, toolchain
requirements, generator inputs and platform outcomes for the adoption attempt
remain **Unknown** until inspected and replayed. The owner selection is current;
the historical suitability report is not a retained build bundle. Moving
`blob/main` links locate public documents, not immutable qualification evidence.

## Adoption wrapper and initial assessment

AA06's adoption wrapper constrains the existing generic
[build records](go-fabric-build-record-validator.md); it is not a second B ledger.
The generic validator checks supplied structure and allows explicitly declared
rosters. The adoption wrapper must require the owner-selected tag and both Linux
architectures, reconcile the same input identity and require the existing eight
parents and B4–B6 children for both roles. For the minimal two-target roster this
is 28 results: 16 parents and 12 children. Additional ABI/CGO/tag tuples require
additional children; they cannot silently replace either mandatory architecture.
This is the wrapper's documentation contract, not evidence that its implementation
or any real SDK run passed. A structurally complete generic record alone cannot
close adoption.

Use the following initial assessment on the existing B1–B8 rows. These are current
gaps, not fabricated producer/verifier results. Every required child begins
missing; any later complete-looking producer claim remains unverified pending
independent replay.

| Existing row | Initial adoption state | Missing evidence / child trace |
| --- | --- | --- |
| B1 | missing | Owner-selected exact tag, retained origin/source and patch manifests. |
| B2 | missing | Actual direct/transitive dependency, SQLite, native-library and tool closure; [notices ledger](adoption/notices-ledger.md) outcomes Unknown. |
| B3 | missing | Exact offline plan, both tuples, native execution environments, source-derived selected tests, exclusions and replay inputs. |
| B4 | missing | Producer and verifier compile/link child for Linux amd64 and Linux arm64, separately keyed to each tuple. |
| B5 | missing | Producer and verifier native test child for Linux amd64 and Linux arm64, exact selections, nonzero execution and repeat logs. |
| B6 | missing | Producer and verifier generator/input child for Linux amd64 and Linux arm64, retained input/tool rights and two clean generation runs per role. |
| B7 | missing | Reconciliation of all parents, children, manifests, logs, outputs and rights evidence. |
| B8 | missing | B1–B7 closure and independent bounded decision; runtime remains denied. |

Before execution, declare exact selected package/test identities from inspected
source, fixtures, expected values, counts, repeats and bounds. Do not invent SDK
APIs or treat generic harness tests as actual SDK tests. Explicitly exclude the
independent-controller leg (for example a separately pinned chip-tool or matter.js
controller) from strict offline B5: it needs its own reviewed plan and retained
interaction evidence and cannot be inferred from an in-memory fixture. Exclusion
does not remove that broader acceptance obligation or authorize any live fabric.

## Report identity, producers and verification

The later producer assembles an offline, public-safe evidence bundle. It contains
one report plus retained inputs, manifests, command logs and outputs addressed by
computed content digests. Use SHA-256 over exact file bytes; directory manifests
list sorted relative paths, entry types, executable modes, symlink targets and
file digests, with an explicit manifest encoding. Reject path escapes, unresolved
external links and unlisted build inputs. Keep raw content identifiers in the
bundle; do not publish source-control commit identifiers, private local paths,
credentials, device identities or personal environment dumps in this report.
Public summaries can cite repository-relative artifact references and explain
how digests are computed rather than paste literal digests.

A report identifies the source origin and acquisition method, exact source bundle
and manifest, any local patches with their base and resulting manifests, complete
dependency closure, toolchain and generator inputs, selected build/test plan and
all result artifacts. A moving branch or package version alone is insufficient.
Record substitutions and replacements explicitly; no silent fallback source is
permitted. Cached acquisition does not establish authentic provenance: a verifier
must reconcile origin evidence with the retained source contents. Missing origin
evidence stays missing even when the source compiles.

A result's identity is the tuple `(exact upstream release, source manifest,
dependency manifest, toolchain manifest, generator/input manifest, target tuple,
command/selection)`.
The target tuple includes OS, architecture, ABI, CGO mode, compiler/linker and
build tags/features. Command/selection includes working directory relative to the
bundle, arguments, permitted environment, test identities, fixture inputs, repeat
count and timeout. Use explicit values or evidenced non-applicability, never
implicit host defaults. Any changed axis creates a new result; do not reuse old
passes under the new identity. These are evidence keys, not persistence keys.

The producer records what actually ran, all exits and selected counts, and all
gaps. A separate verifier checks manifest closure, origin, selection adequacy,
logs and artifact identities, then repeats the bounded offline procedure using
retained inputs in a clean output directory. Record producer results and verifier
results separately; disagreement prevents pass. These are review responsibilities,
not new credential, principal, permission or authorization gates.

## Finite evidence ledger

Use exactly these eight required rows, with per-target children for B4–B6.
The adoption wrapper requires both Linux targets described above; the generic
record format remains reusable for other explicitly bounded scopes.
Declare a finite nonempty target list and a finite nonempty deterministic test
selection for each claimed execution target before running. List exclusions and
why they do not establish coverage. An omitted target is outside the claim, not
supported. The mandatory adoption Linux targets above have no established
qualification results; actual source-derived test selection remains Unknown.

| Row | Required producer evidence | Verifier responsibility and pass criterion |
| --- | --- | --- |
| B1 Source provenance | Public origin evidence, retained source and path manifest, acquisition method, patch chain or explicit no-patch record; include embedded/generated source. | Reconcile origin and exact content, all patch inputs and outputs; no unexplained changes or missing source. |
| B2 Dependencies and tools | Complete direct/transitive module inventory, versions, module metadata and checksums, replacements, vendored/local modules, generator tools, native libraries, compiler/linker and toolchain manifests; mark absent categories with evidence. | Reconcile inventory with actual build inputs and cached bytes; checksums alone do not prove provenance or license status. No undeclared external input or download may be required. |
| B3 Reproduction plan | Exact offline commands, relative directories, bounded timeouts, environment allowlist, cache/toolchain locations within the disposable build area, clean-output procedure, target list and test selection rationale. | Replay without network or services; disable module lookup/download and automatic toolchain acquisition (`GOWORK=off`, `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`). Confirm all inputs are retained and no host-only dependency supplies a hidden success. |
| B4 Compile/link | For each target tuple, exact package/entrypoint selection and build flags, exit status, logs and output manifest from a clean build. | Independently repeat that tuple; require successful compile/link and matching declared inputs. A cross-compile is compile evidence only, even if it emits a test executable. |
| B5 Deterministic tests | Source-derived exact test identities and rationale, fixtures, selected/executed/passed/failed/skipped counts, exits and bounded repeat results; cover intended local behavior and failure paths with synthetic/public fixtures. | Confirm selection against inspected source, nonzero actual execution, independent expected values and detached controls, no required skips and consistent outcomes on repeated clean runs. Command exit zero with zero selected/executed tests is not pass. Network, fabric or hardware tests cannot satisfy this offline row. |
| B6 Generator/input reproducibility | Inventory every generated path, generator/tool identity, exact input manifests, command and generated-output manifests; distinguish upstream supplied generated code from locally regenerated code. | Repeat generation twice in clean directories and require identical generated bytes. If no generation is involved, evidence from source/build inspection must establish that; absence of an inventory is not evidence of absence. Unavailable regeneration inputs keep this row unresolved. |
| B7 Cross-row closure | All result identities, logs and artifacts reconciled to B1–B6, with producer/verifier outcomes and dependency links; account for all declared targets and exclusions. | Reject stale or mixed inputs and missing children. Compare clean build outputs; record nondeterminism and any predeclared, justified metadata normalization without hiding executable/generated-content differences. Unexplained differences fail reproducibility. |
| B8 Bounded decision | Full ledger, unresolved/failure reasons, confidence labels, exact qualified scope, excluded scope and explicit runtime non-admission. | Pass only if B1–B7 and all required children pass for the same identity and declared scope. Record independent review and limits; no broader target, interoperability or certification claim follows. |

B5 does not prescribe invented go-fabric APIs or test names. The later source
inspection must make the selection concrete before results can pass. Relevant
behavior requiring listeners or devices stays unverified in this offline scope;
do not remove it from a broader readiness denominator. For any later follow-up
test claiming identity/key isolation, vary every stated key field and map each
to the production mutation it catches. Executed mutant kills and reasoned
mutation mappings must be reported separately. This prose adds no such test.
Selection rationale must also account for clock, random seed, locale, filesystem
ordering and concurrency assumptions: use controlled inputs and deterministic
synchronization where relevant, and retain repeat counts and failure diagnostics.

The later build plan must say whether byte-for-byte binary reproducibility is
claimed. Successful repeat compilation with explained metadata differences can
establish compile repeatability only; it cannot be labelled byte reproducible.
Generated source still requires the exact-byte B6 comparison. Log sanitization
must retain commands, counts, exits and evidence identity; if safe retention is
impossible, withhold the affected result rather than invent a replacement log.

## Generated-input rights closure

B2 inventories actual compiled/transitive inputs, including SQLite on any selected
storage route; a notice calling it test-only cannot overrule inspected imports.
B6 links every extracted model, schema, generated path, generator/tool and exact
input to retained origin, license, NOTICE/attribution and redistribution evidence.
Use the [notices ledger](adoption/notices-ledger.md) to record obligations and
missing inputs, not as a license clearance declaration. Include upstream-supplied
generated code even when the build never invokes its generator. Source availability,
module checksums, Apache-2.0 on the top-level project and upstream green metadata
do not establish rights for transitive or extracted inputs. Unknown obligations
keep rights closure unresolved and prevent B2/B6, dependent B7 and B8 from pass.
Record independent verifier review of both the closure and shipped notices;
retain superseded or contradictory evidence rather than silently choosing it.

## States, transitions and evidence loss

Each required row/child has one of four review states. These state names are not
runtime return values or the existing SDK build-evidence serialization.

| State | Meaning and disposition |
| --- | --- |
| missing | A required record, input, log, selection or artifact is absent. Record the gap and dependent rows; no qualification. |
| failed | An observed build/test/reproduction assertion failed, or verification found contradictory, stale, malformed or mismatched evidence. Preserve the failure and its reason; no qualification. |
| unverified | A complete-looking claim has not been independently verified, execution is blocked, or the evidence cannot establish the requested outcome. Record the blocker; no qualification. |
| pass | Both producer and verifier satisfy the row criteria for the same identity with retained evidence and no unresolved dependencies. This is a bounded evidence conclusion only. |

At creation, absent evidence is missing. Producer success alone is unverified;
verification can promote it to pass. A failed assertion becomes failed immediately.
A corrected attempt is a new linked attempt, never an overwrite of a failure.
Loss of a required artifact changes the current assessment to missing. Discovery
of a mismatch changes it to failed. Inability to repeat an otherwise retained
claim changes it to unverified. Every such change reopens dependent rows and B8.
Keep the old attempt and its limitations as history; never treat it as a current
pass. A new identity begins a new assessment rather than inheriting results.

Aggregate B8 is failed if any required row fails; otherwise missing if a required
row is missing; otherwise unverified if any is unverified; otherwise pass. Keep
all underlying reasons even when one aggregate state takes precedence. Evidence
loss does not erase native observations or partial last-known-good state, mark
old values fresh, or permit allocation. MCP compatibility and GraphQL parity
remain unchanged. No protocol facts are added: those require their public native
documentation contract, semantic acceptance and target mapping first.

| Failure class | Required treatment |
| --- | --- |
| Source/inventory integrity | Missing bytes are missing; contradictory origin, manifest, checksum or replacement records are failed. Block dependent results. |
| Provisioning/environment | Missing tools/modules, sandbox or socket-bind denial, unavailable offline cache: unverified execution with the exact error. Also mark absent required input rows missing. Never call these product test failures or passes. |
| Execution | Nonzero compile/test assertion exit is failed. A timeout records the bound and observed termination, with execution unverified unless an independent assertion already failed; incomplete work is never pass. |
| Selection | Zero collected/executed tests, required skips or unknown selection leave B5 unverified. A fabricated positive count or a known wrong selection is failed. |
| Reproduction | Generator drift, unexplained output differences, hidden inputs or mismatched evidence identity are failed. Retain both outputs and the comparison. |
| Unsupported/excluded scope | An explicit exclusion is outside the claim, not evidence of support. Excluding a previously required child does not turn that report green; create and review a new bounded scope. |

## Synthetic report examples

These historical synthetic `v0.8.0` examples are preserved from the generic
contract. They are not a new pin, an owner-selected adoption tag or an adoption
roster; the one-target shape below cannot satisfy the adoption wrapper.
These examples are invented, unevaluated review records, not command transcripts,
real artifacts, go-fabric support claims or test results. Artifact names below
are symbolic. For a self-contained digest illustration, compute
`hashlib.sha256(b"synthetic-source-a").hexdigest()` for source A and
`hashlib.sha256(b"synthetic-source-b").hexdigest()` for source B. They identify
different synthetic bytes, not upstream source. Do not substitute labels for
actual retained manifests in a later report.

**Valid bounded report shape:** assume retained synthetic artifacts exist for
exact release `v0.8.0`, source A, dependency D, toolchain T, generator/input G
and command plan C. The
only declared target is synthetic tuple P: `os=synthetic-os`,
`arch=synthetic-arch`, `abi=synthetic-abi`, `cgo=off`,
`compiler=synthetic-compiler`, `linker=synthetic-linker`, `tags=[]`.
C uses relative directory `fixture/`, fixed empty optional environment, the
invented local selection `synthetic-local-case`, input `synthetic-input-a`,
repeat count 2 and a 10-second timeout per action. These are symbolic fixture
operations, not proposed go-fabric commands. The independent synthetic test has
one selected, one executed, one passed, zero failed and zero skipped on each of
two clean producer runs and two clean verifier runs. This does not name or claim
a real go-fabric test or platform.

| Row | Assumed synthetic record | Review outcome under those assumptions |
| --- | --- | --- |
| B1 | Origin reconciled to retained A; manifest and explicit no-patch record checked by verifier. | pass |
| B2 | D includes the full dependency/tool closure; all bytes and replacements reconciled. | pass |
| B3 | C declares P, the exact synthetic test and offline bounded clean replay; verifier repeats it. | pass |
| B4 | Producer and verifier clean compile/link P with exit zero and retained output manifests. | pass for P compilation only |
| B5 | The stated nonzero counts, deterministic expected result and both repeat logs agree for A/D/T/G/P/C. | pass for that synthetic local test only |
| B6 | Both parties regenerate from G twice, retaining identical output bytes and tool/input manifests. | pass |
| B7 | All identities and artifacts reconcile; clean build outputs are byte identical in this example. | pass |
| B8 | B1–B7 pass with independent review, one-target scope and runtime non-admission stated. | pass for the invented offline build/test scope only |

**Invalid reports and loss cases:** each variant changes only the stated evidence
from the assumed valid example; preserve other records and reopen dependents.

| Variant | Required assessment |
| --- | --- |
| Historical generic release is missing/older than `v0.8.0`, or producer and verifier name different exact releases. | Reject the declaration or identity join; B7/B8 cannot pass. |
| Target tuple changes, a required target child is missing, or a version label is the only claimed success. | Reopen or fail affected rows; no prior assessment or target support is inherited. |
| Compile log belongs to source B while report and tests name A. | B4/B7 failed for identity mismatch; B8 failed. |
| Producer says all rows passed but no verifier replay exists. | Claimed passes remain unverified; B8 unverified. |
| Test command exits zero with zero selected/executed tests. | B5 unverified, B8 unverified; no test success inferred. |
| Build exits nonzero with a compiler error; synthetic assertion fails in a separate run. | B4 and B5 failed, each with its own exit/log; B8 failed. |
| Cache input is absent and offline module resolution cannot proceed. | B2 missing, B4/B5 unverified for blocked execution; B8 missing. |
| Generator outputs differ between clean runs. | B6/B7 failed, B8 failed; no silent output selection. |
| A required log is lost after a prior pass. | Affected row missing and B8 missing; retain the earlier assessment as history. |
| A bounded command times out, or a sandbox denies a required operation. | Affected execution unverified with exact cause and incomplete counts; B8 cannot pass. |
| P only cross-compiles; test executable never runs. | B4 may pass; B5 unverified; B8 cannot pass for the declared build/test scope. |
| All eight rows pass, then report asserts a production endpoint or certification. | Reject that conclusion as outside scope; runtime remains denied. Offline pass is not admission evidence. |

## Claim boundaries and remaining acceptance

Proven / Hypothesis / Unknown describe confidence in each specific claim, with
its supporting evidence. They are separate from ledger states, source acquisition,
build qualification, native qualification and runtime admission. The synthetic
examples establish no Proven go-fabric claim. Current go-fabric implementation
claims remain Unknown; proposed expectations remain Hypothesis until evidenced.
A pass can support Proven only for the exact observed bounded claim, never for
an unexecuted feature or a device family by analogy.

Compile/link evidence proves only that the declared sources and inputs produce
the declared artifact for one target tuple. Test evidence proves only the named
executed assertions under the recorded conditions. Supported build-target coverage
requires an explicit reviewed target roster and passing applicable evidence for
every listed tuple; a host pass, cross-compile or one feature set cannot cover
another tuple. This contract claims no such roster or platform support today.

Matter target coverage separately needs the locked data-model inspection,
mandatory composition closure and accepted public semantic mapping described by
the inventory contract. No compile count or local test count measures that
coverage. Interoperability needs a separately reviewed controller/endpoint matrix
and observed interactions for declared implementations and versions; local tests
and the existing upstream sample fixture cannot establish it for Helianthus.
Conformance and certification need their own applicable external assessment and
product evidence; neither follows from offline or hosted green checks.

Parent #24 remains open: actual source qualification, dependency review, concrete
commands and test selection, target results, integration and remaining parent
acceptance are future work. This document supplies no clone, dependency, SDK pin,
workflow, upstream submission or runtime implementation. Official upstream
submission remains separately operator-approved work. No live operation is
authorized. No new namespace or authorization gate is introduced before 0.9;
authentication and authorization remain 0.11 work. Existing trust/admission checks
are unchanged, as are G5 D1–D5/O8 and driver ownership of the eeBUS output binding.

The contract check is manual review of every source claim, transition, loss
disposition, synthetic example and link against the supplied sources. Readability
is not semantic validation or runtime proof. No new test is required for prose.
Full CI, transport execution and physical/live-fabric acceptance remain separate;
this contract reports none of them as passed.
