# Offline build-record structural validation

Issues #29 and #33 implement partial slices of #24 / Project #2. Issue #61
retargets their guidance for #59 direction 2 without changing the generic API.
The isolated
[`qualification/buildrecord`](../qualification/buildrecord/record.go) package
uses only the Go standard library and supplied in-memory records. It is
scaffolding, not a decision about runtime layout, persistence or registration.
Nothing here is stable before Helianthus 1.0.

## Authority and implemented subset

The [offline build qualification contract](go-fabric-build-qualification.md)
owns Z19 B1–B8 and the per-target B4–B6 evidence requirements. The
[runtime qualification contract](go-fabric-runtime-qualification.md) and
[inventory completeness contract](target-profile/inventory-completeness.md)
keep mechanics, source facts, target closure and admission separate. Protocol
facts require the public native documentation contract first. This package adds
no protocol facts, source-derived go-fabric API/test names or platform support claims.

`Validate(Record)` checks a predeclared `Scope` against supplied `Results`.
The finite scope must contain explicit input manifests, a ledger selection,
separate producer/verifier role tokens and a nonempty list of unique target
identities. Both roles require exactly one B1–B8 parent and one B4, B5 and B6
child for every declared target. Parents carry the ledger selection and zero
target tuple; children carry their declared build, test or generation selection.
Unknown rows, roles, targets, duplicate records and duplicate target tuples fail.
No target outside this roster is covered. There is no inference of supported
platforms, nor an API for silently excluding a required child.

`Inputs.UpstreamRelease` must declare one canonical exact stable release in
`vMAJOR.MINOR.PATCH` form, with unsigned decimal components (no leading zeros),
at or above `v0.8.0`. Each component must fit an unsigned 64-bit integer.
Missing/older releases, ranges, `latest`, wildcards, prereleases and build metadata
fail the declaration. A later exact release needs a new matching scope and results;
selection alone supplies no authenticity, build success or platform support.
This is the existing generic validation rule and historical fixture baseline,
not the current adoption pin. The owner must supply the exact adoption tag; it
is missing here. Actual source inspection for that attempt remains Unknown.
No SDK lock, module or dependency pin changes.

For adoption, Linux `amd64` and `arm64` are both mandatory qualification
targets. Records explicitly
supply ABI, CGO, compiler/linker, tags/features and toolchain manifest for each
attempt. They are not supported-platform assertions or automatic target defaults.
The validator permits other explicitly declared tuples without claiming support.

AA06's adoption wrapper applies a narrower policy over these same generic
records, not a parallel B ledger. It must enforce the owner-supplied exact tag,
both Linux architectures and matching B1–B8 parents plus B4–B6 children for each
role/target. The generic API itself permits other rosters; its structural success
cannot prove that adoption policy was applied. A minimal adoption attempt has
28 results, with further children for additional declared tuples. See the
[initial B1–B8 assessment](go-fabric-build-qualification.md#adoption-wrapper-and-initial-assessment):
all real rows and required children are missing until retained evidence exists;
producer-only claims stay unverified. No real wrapper execution is asserted here.

Declare the OS, architecture, ABI, CGO mode, compiler/linker, tags/features,
toolchain, exact source-derived selected tests and execution environment per
attempt. Native B5 execution on each architecture is distinct from B4
cross-compilation; producing an arm64 test binary on amd64 is not arm64 execution.
Label emulation separately; it cannot replace required native execution. Explicitly
exclude and track the independent-controller leg separately from strict offline
B5. No upstream green metadata or module checksum alone can promote a row.

Producer and verifier must each replay the same bounded plan from retained inputs
in clean outputs, retaining actual commands, exits, counts, logs and manifests.
Role-token inequality alone does not prove independent execution. B2 dependency
and SQLite closure and B6 extracted-model/generator/tool obligations must reconcile
with the [notices ledger](adoption/notices-ledger.md). Missing rights evidence
remains Unknown and prevents qualification even if generic structure is complete.
Actual cache closure, hosted workflow replacement and real producer/verifier runs
are AA08's follow-through; this documentation performs none of them.

Every result joins exactly on upstream release, source, dependency, toolchain
and generator/input manifest digests; OS, architecture, ABI, CGO, compiler, linker, tags and features;
and relative working directory, arguments, permitted environment, exact test IDs,
fixture digests, repeat count and timeout. Order is significant for slices;
environment maps compare by exact keys and values. Optional collections must be
explicit non-nil collections in declarations; an empty collection is permitted
where applicable. No host defaults fill missing declarations. CGO is `on` or
`off`; other non-applicability facts must be explicit evidenced strings.
Paths use clean bundle-relative slash syntax (`.` is permitted), without absolute
paths, backslashes, drive prefixes or parent traversal. This is a lexical check,
not filesystem or symlink validation.

Digest fields accept lowercase SHA-256 encodings. The validator does not hash,
read or authenticate their contents. Parent evidence references are opaque claims
of retained supporting material; their row-specific adequacy still needs review.
Each child requires exactly the declared number of supplied run records, with
completion, exit, log and output references. B5 requires a nonempty unique test
selection and consistent nonnegative selected/executed/passed/failed/skipped
counts on every run. Every selected test is required: zero execution and any skip
prevent completeness. B6 requires at least two runs per role and equal supplied
output digests across all repeats and both roles for that target. This subset
requires explicit generation records; it provides no automatic no-generation
shortcut. A future evidenced non-applicability design remains outside this API.

Role tokens distinguish supplied producer and verifier responsibilities. They
are synthetic role labels, not contributor identities, authentication or proof
of independence. A result actor must match its declared role token, and the two
tokens must differ. No permissions or authorization gates are introduced.

## Assessment, transitions and evidence loss

The return value contains only a state and keyed findings:

| State | Structural meaning |
| --- | --- |
| `structurally-complete` | All required supplied declarations, identities, outcomes, references and run summaries satisfy these structural checks. |
| `unresolved` | Required records/references/repeats are absent, a claim is missing or unverified, execution is incomplete/blocked/timed out, or required tests were skipped. |
| `failed` | The supplied structure contradicts the declaration: malformed scope/identity/reference, duplicate or undeclared key, actor conflict, observed failure/nonzero exit, inconsistent counts or generator disagreement. |

Failure takes precedence over unresolved, but every finding is retained in
stable declaration/result order. A timeout or blocked run remains unresolved;
a separately supplied nonzero exit is also a failure and takes precedence.
Zero selected tests contradict the declared nonempty selection and fail; zero
executed tests with all selected tests skipped remain unresolved.

Each call reassesses the current supplied snapshot. Removing a previously
required record/log/output reopens the assessment; adding a mismatch fails it.
A new identity needs a new scope and matching results. The function does not
mutate inputs, keep a hidden pass cache, persist history, supersede attempts or
erase failed evidence. The caller must retain prior attempts separately and
must not merge records from old identities. Native evidence and partial
last-known-good state remain untouched, and no retained value becomes fresh.
eBUS MCP compatibility, GraphQL parity, G5 D1–D5/O8 and driver ownership of the
eeBUS output binding remain unchanged.

## Synthetic example and tests

The [tests](../qualification/buildrecord/record_test.go) construct historical generic
synthetic records selecting `v0.8.0` and declaring Linux `amd64` and `arm64`
with synthetic
ABI/compiler/linker/toolchain evidence, eight ledger parents and six target
children per role: 28 results total.
Every child has two runs; each B5 run declares two selected, executed and passed
tests. All digests are computed from public synthetic strings with
`sha256.Sum256`; no literal digest fixture identifies real source or content.
This shape returns `structurally-complete`. Changing just a compile result's
source manifest fails its identity join. Removing a verifier child is unresolved;
changing a B6 output fails; losing a log after completeness reopens the result.
The historical `v0.8.0` examples are not a new pin or adoption tag.
None of these examples executes go-fabric or grants admission.

`TestBuildRecordIdentity` varies each of the 20 fields independently for B4–B6
and both roles. `TestBuildRecordLedgerIdentity` also varies every field for
all eight parents. Required-row tests omit and duplicate each of the 28 records.
Outcome, execution and evidence-loss controls use independent explicit expected
states. A mutation that normalizes one result field to the declared field before
comparison is the reasoned production mutation caught by each corresponding
identity case. Release cases cover consistent missing/old/invalid declarations,
explicit later releases and mismatched producer/verifier selections. Target cases
change or omit every tuple field independently; required-row tests omit every
child. Version-only claims cannot substitute for declarations or evidence.
These controls catch dropping release validation/equality, target equality or
field completeness, missing-child detection, or returning success solely for
`v0.8.0`. Executed mutation evidence belongs in the change's review record.

The focused offline check is:

```sh
GO111MODULE=off go test ./qualification/buildrecord -count=1
```

The existing hosted workflow does not discover this isolated Go package; future
registration belongs to its owner. This change adds no workflow, module, SDK
import, dependency pin, lock change or gate override.

## Remaining evidence and claim limits

Structural completeness does not establish authentic source origin, acquisition,
patch-chain correctness, dependency closure, actual retained bytes, manifest
encoding/path safety, log truth, source-derived selection adequacy or independent
verifier reproduction. B1–B3 and B7–B8 accept typed identity/outcome/reference
claims; they do not inspect their underlying row content. B4 compile output
reproduction and justified normalization, B5 per-test logs/expected values and
actual repeat execution, B6 generated-path inventory and exact bytes, and cross-row
artifact/dependency closure still require the full qualification procedure.
Fabricated but consistent summaries can pass structure; they cannot prove facts.

Proven / Hypothesis / Unknown label individual supported claims, separately from
qualification, freshness and admission. The tests prove only fixture behavior;
actual go-fabric source, APIs, target coverage and reproduction remain Unknown.
There is no library-qualified or production-ready return state. The current
[readiness evaluator](../runtime/production_readiness.py) still denies admission
because the reviewed lock has no accepted composition profile. This validator
has no runtime integration or allocation operation and cannot change that rule.

There is no source authenticity verifier, filesystem traversal, command execution,
build runner, fetch, listener, live fabric or hardware action. Full CI, transport,
physical smoke, interoperability and certification remain separate acceptance.
Parent #24 stays open for actual qualification and integration. No authorization
gates are added before 0.9; authentication and authorization remain 0.11 work.
Documentation acceptance requires manual source, claim, transition, loss, example
and link review, independently of test success.
