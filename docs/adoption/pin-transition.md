# go-fabric pin transition dossier

This specification advances [issue #80](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/80),
a partial child of [#59](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/59).
The umbrella remains open. Nothing here is stable before Helianthus 1.0.

## Status and authority

This is a proposal for an owner operation. **The pin is not installed.** The
owner has adopted go-fabric on its Matter 1.6.1 data model. This document records
no owner answer to the exact tag and module question, and it writes no lock,
module, evaluator, gate or workflow file.

Approving or merging this dossier is not evidence that a pin was installed, that
a checksum was verified, or that any B1–B8 or runtime qualification row passed.
Every command below is a specification: none was run for this document. Every
`<placeholder>` stays a placeholder until the owner supplies a verified value.

**Proven** below means only a bounded statement from the supplied graded go-fabric
suitability report, or an observation of this repository's current source.
**Hypothesis** is proposed work. **Unknown** is unchecked or missing. These grades
are separate from the missing/failed/unverified/pass states of the
[build ledger](../go-fabric-build-qualification.md).

## Question for the owner and proposed answer

Which exact go-fabric tag and module identity replace the fork lock?

**Proposed answer: inspect and retain `v0.13.12` first.** It is proposed because
it is the only newly inspected exact tag in the supplied report. It is not assumed
to be the latest release, and it is not qualified. The report observed upstream
main one commit past that tag, and it inspected the tag by connector reads, not
from a retained source bundle.

If the owner selects any other tag, obtain a fresh six-axis triage first. Nothing
from the report's inspection carries over, including model revisions, toolchain,
license findings and runtime risks.

| Identity field | Proposed value | Grade and limit |
| --- | --- | --- |
| Provider | `go-fabric` | Owner decision. |
| Module path | `github.com/SukramJ/go-fabric` | Hypothesis. It is the public locator and the exact value [`runtime/adoption_baseline.py`](../../runtime/adoption_baseline.py) accepts. Confirm it against the `module` directive in the retained tag's `go.mod`. A difference stops the transition. |
| Exact tag | `v0.13.12` | Proposal awaiting the owner. Use the canonical stable form, with no range, alias, moving branch or prerelease. |
| Module zip checksum | `<owner-verified zip h1>` | **Missing.** No value is known to this document. |
| `go.mod` checksum | `<owner-verified go.mod h1>` | **Missing.** No value is known to this document. |
| Data model | Matter 1.6.1 | Owner decision; reported source identity for the proposed tag only. |
| Declared Go version | 1.27.3 | Proven as a reported declaration for the proposed tag. Execution on any target is Unknown. |

Record the owner's answer on the issue before any step below starts.

## Preconditions

All of these must hold before any file changes:

1. The owner has answered the question above on the issue.
2. The checksums have been acquired and verified as specified below.
3. All four retained manifests exist and reconcile.
4. A six-axis [triage record](../go-fabric-upstream-tag-triage.md) built from
   retained excerpts passes the
   [structural validator](../../qualification/tagtriage/record.go). The report's
   own source review claims no such record or validator run, so this applies to
   `v0.13.12` as well.
5. The owner has explicitly authorized the otherwise forbidden lock, module, gate
   and workflow writes for this exact identity.

## Module checksum acquisition and verification

A module checksum shows that bytes match what the checksum database recorded. It
does not establish authorship, rights, a passing build or target support.

**1. Acquire.** Run from an empty directory outside any module, with a disposable
cache and with checksum verification on:

```sh
GOWORK=off GOTOOLCHAIN=local GOFLAGS= \
GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org \
GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= \
GOMODCACHE="$WORK/modcache" GOCACHE="$WORK/gocache" \
  go mod download -json "$MODULE@$TAG"
```

The `Sum` field is the zip checksum and `GoModSum` is the `go.mod` checksum. Retain
the JSON output. Its origin data may hold a source-control identifier: keep that in
the bundle and never in public text.

**2. Cross-check the database.** Retain the checksum database's lookup record for
the same `module@version`. Module paths are case-encoded in proxy, database and
cache paths, so the proposed path appears there as `github.com/!sukram!j/go-fabric`.

**3. Derive independently.** Obtain the exact tag from the public origin by a
separate route, build the module zip with `golang.org/x/mod/zip`, and hash it with
`golang.org/x/mod/sumdb/dirhash`. Retain that tool's identity. Plain archive bytes
are not comparable, because module zips apply their own file-selection rules.

**4. Verify separately.** A verifier repeats steps 1–3 in a clean environment and
records its own results. Producer results alone remain unverified.

**5. Replay offline.** Using a copy of the retained cache:

```sh
GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
GOMODCACHE="$BUNDLE_COPY/modcache" GOCACHE="$WORK/gocache" go mod verify
```

`go mod verify` checks cache consistency only. The `go.sum` and lock comparison
below is what binds the cache to the pinned identity.

| Observation | Assessment | Required action |
| --- | --- | --- |
| Any two of steps 1–4 disagree | failed | Stop. Retain every output. Never choose the value that looks right. |
| The tag resolves differently from an earlier observation | failed | Stop and return the question to the owner. Do not re-pin silently. |
| The tag is retracted or deprecated upstream | Stop for owner review | Record the statement. If the status was not checked, it stays Unknown. |
| A database or origin read is unavailable | unverified | Retain the exact error. Do not substitute a cached value or disable verification. |
| The tagged `go.mod` declares a different module path | failed for this identity | Stop. The exact-module check would reject it, and changing that check needs a fresh owner decision. |

## Retained manifests

The bundle follows the [build qualification contract](../go-fabric-build-qualification.md):
SHA-256 over exact bytes, sorted relative paths, entry types, modes and symlink
targets, and no unlisted inputs. It lives outside this repository, at a location
the owner chooses, and is cited only through public-safe relative references. All
four manifests are **missing** today.

| Manifest | Must contain | Feeds |
| --- | --- | --- |
| Source | Complete tagged bytes, the module zip, origin and acquisition evidence, and an explicit no-patch record or full patch chain. | B1, B7 |
| Dependency | The complete direct, transitive and tool module closure with versions and both checksums; replacements, nested-module exclusions and native requirements, or evidenced absence of them; the retained download cache. The upstream `go.mod` tool directive and the SQLite driver import are in scope. A notice calling SQLite test-only does not overrule the inspected import. | B2, [notices ledger](notices-ledger.md) |
| Toolchain | Exact Go toolchain identity per target tuple, compiler and linker, CGO mode, tags and permitted environment. Linux amd64 and Linux arm64 are separate children. | B3, B4, B5 |
| Generator and input | The embedded `parity/schema.json` snapshot and its extraction inputs; `script/generate_matter_schema.go`, `script/clustergen` and `script/devicegen`; the matter.js and Node inputs; the generated path inventory; and the rights evidence for each. Rebuilding committed Go and refreshing the snapshot are separate activities. Do not copy restricted model XML into deliverables. | B6, notices ledger |

Selecting packages for a real adapter comes later, under the
[reference bridge](reference-bridge.md) work. Until then, inventory the whole tagged
module closure instead of narrowing it by guesswork.

## Owner-only changes traced to current consumers

Every path below is **external owner follow-through**. No batch ticket owns these
files, including `sdk/matter-sdk.lock.json`, the future `go.mod` and `go.sum`,
`runtime/production_readiness.py` and `tools/validate_lock.py`.

| Owner change | Current consumer | What the change must handle |
| --- | --- | --- |
| Replace `sdk/matter-sdk.lock.json` | `tools/validate_lock.py` reads it by default and compares it through `validate_lock`; `tests/test_production_readiness.py` loads it directly | Any edit without a matching evaluator edit denies and fails the validator. They must change together. |
| Add `go.mod` and `go.sum` at the repository root | None today. The repository has neither file, its Go packages import only the standard library, and the gateway's `go.mod` names neither this repository nor go-fabric. Nine documents run `GO111MODULE=off` checks. | Choose this repository's module path; proposed `github.com/Project-Helianthus/helianthus-matter-binding`. Confirm that the `GO111MODULE=off` commands still work, which is a Hypothesis today. |
| Migrate `runtime/production_readiness.py` | Imported by `tools/validate_lock.py`, `runtime/adoption_baseline.py` and both test modules below. `EXPECTED_LOCK` and its pinned constants, `validate_build_evidence`, `REQUIRED_BUILD_TARGETS` and `validate_composition` all assume a commit-pinned C++ SDK. | Follow the evaluator rules below. |
| Migrate `tools/validate_lock.py` | `.github/workflows/sdk-baseline.yml`, `scripts/checkout_sdk.sh`, `AGENTS.md`, `CONTRIBUTING.md`. `validate_checkout` and `--sdk-checkout` read commit and blob fields. `--mapping` reads `mapping.sha256`. | Replace checkout validation with module validation. Leave `validate_workflow_pins` and `tests/test_workflow_pins.py` as they are: that check is not SDK-specific. |
| Reconcile `runtime/adoption_baseline.py` | `tests/test_adoption_baseline.py`; it calls the real evaluator | Its `checksum` field binds the zip `h1`. Its tag check accepts prereleases while the build contract requires a stable tag, so apply the stricter rule at selection. Decide whether to tighten the check in this change. |
| Update `tests/test_production_readiness.py` | Its fixtures use `lock["sdk"]["commit"]`, `target_commit`, a Matter 1.7 profile identifier and the legacy build target names | Map each assertion to a successor or retire it with a stated reason. Do not weaken any denial assertion. |
| Update `tests/test_adoption_baseline.py` | Its real-API test builds composition and build evidence from `EXPECTED_LOCK["sdk"]["commit"]` | Same rule. Its positive control must still pass, or the denial result proves nothing. |
| Retire `scripts/checkout_sdk.sh` | Called by the hosted workflow. It hard-codes the fork remote and commit and calls `--sdk-checkout`. | Remove with the workflow step that calls it. |
| Retire `scripts/run_upstream_fixture.sh` and `scripts/run_owned_group.sh`, with `tests/test_fixture_safety.py` | The hosted workflow runs the fixture; the fixture calls the group runner; the safety test reads both files | Remove the test with its subjects, or keep the group runner and its test if a replacement job uses them. Do not leave a test pointing at a removed file. |
| Replace the SDK steps in `.github/workflows/sdk-baseline.yml` | Runs on every pull request. Its checkout, bootstrap, build and fixture steps all need the fork. | Keep the validator and unit test step. The SDK steps cannot pass once the lock changes, so they leave in the same change. The hosted qualification job replacing them is separately specified owner work under #59 direction 2. |
| Update `examples/runtime-composition.profile.json` | Documented as the required shape in `docs/SDK_BASELINE.md`; carries `target_commit`, the legacy mapping and Electrical Sensor revision 2 | Move to the new identity join. It stays `proposed` with missing evidence. |
| Reconcile documents | `docs/SDK_BASELINE.md`, `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `docs/adoption/matter-161-deferred.md`, `docs/adoption/reference-bridge.md`, `docs/adoption/gateway-library-composition.md`, `docs/PORTAL_ENDPOINT_LIFECYCLE.md`, `docs/go-fabric-build-qualification.md`, `docs/go-fabric-build-record-validator.md`, `docs/go-fabric-report-mechanics-harness.md`, `docs/go-fabric-runtime-qualification.md`, `docs/contribution-candidates/bridge.md`, `docs/contribution-candidates/eem.md` and the `docs/target-profile/` pages that cite the lock or evaluator | Each describes the lock as legacy or the evaluator's current shape. Re-read each one: "installed" must mean only the exact tag, never qualified. |

Two consumers stay **out of this transition**. The gateway's default projection and
feed cite the legacy target and need their own cutover with complete public target
evidence. The public semantic owner keeps the legacy Matter 1.7 target artifact,
and this transition does not change it.

## Proposed lock shape

This shape is a Hypothesis for owner review. It holds no real value:

```json
{
  "schema_version": 2,
  "accepted_runtime_composition_profile_sha256": null,
  "sdk": {
    "provider": "go-fabric",
    "module": "<module path confirmed from the retained go.mod>",
    "release_tag": "<owner-selected exact tag>",
    "checksum": "<owner-verified zip h1>",
    "go_mod_checksum": "<owner-verified go.mod h1>",
    "model": {"name": "Matter", "version": "1.6.1"}
  },
  "mapping": null,
  "target": {
    "device_type": {"name": "Electrical Sensor", "id": "0x0510", "revision": "<read from retained source>"},
    "power_topology_cluster": {"name": "Power Topology", "id": "0x009C", "revision": "<read from retained source>"},
    "electrical_power_measurement_cluster": {"name": "Electrical Power Measurement", "id": "0x0090", "revision": "<read from retained source>"},
    "active_current_attribute": {"name": "ActiveCurrent", "id": "0x0005", "optional": true}
  }
}
```

- **Profile pin:** `accepted_runtime_composition_profile_sha256` stays `null`
  throughout this transition. Setting it is separate later work with its own review.
- **Schema version:** the bump makes any stale reader fail on the version as well
  as on the content. It adds no namespace.
- **Mapping:** use `null` until the public semantic owner publishes an accepted
  Matter 1.6.1 target and the owner supplies its identity. Do not keep the 1.7
  artifact's identity in the new lock, because that would present a legacy target
  as the adopted one.
- **Revisions:** the report gives Electrical Sensor revision 1, Power Topology
  revision 1 and Electrical Power Measurement revision 3 for `v0.13.12`. Re-read
  them from retained source; another tag starts unknown. Never edit a revision
  constant to make a comparison match.
- **Legacy fields:** `linux_app`, `controller`, `upstream_merge`, `data_model` and
  `blobs` have no successor. They describe the fork's checkout.

## Evaluator and checker migration

The migration must not carry over the three current behaviors below. The first two
were observed by calling the unmodified code with synthetic locks; the third is
read from source.

| Current behavior | Required behavior after migration |
| --- | --- |
| `validate_build_evidence` and `validate_composition` raise `KeyError` on the new shape. `evaluate` is safe today only because it denies earlier. | Malformed or legacy-shaped input returns a denial with a reason. It never raises. |
| A `null` mapping in both lock and composition produces no mapping reason, because the comparison only tests equality. | An absent mapping is itself a denial reason. |
| Build evidence and composition join on one `sdk_commit` or `target_commit` value. | Both join on the exact `(module, release_tag, checksum)` tuple. A commit field is rejected as an unreviewed shape. |

The following must also hold:

- **Order:** keep the current order. Lock equality comes first, then the absent
  profile pin, and caller-supplied fields are read only after both.
- **Build targets:** retire `REQUIRED_BUILD_TARGETS`, because those are C++ build
  products. The successor is the B1–B8 result for the same identity on both Linux
  targets. The evaluator can compare that claim but cannot authenticate it.
- **Copies of the pin:** the lock, `EXPECTED_LOCK` and `go.sum` each hold the pin.
  The checker requires all three to agree. There is no "if `go.sum` exists" branch,
  because that branch would fail open.
- **Checker:** `tools/validate_lock.py` stays dependency-free. Its default run
  checks the lock, the evaluator pair, the `go.mod` and `go.sum` entries and the
  workflow pins, with no network. A new option takes a retained module cache,
  recomputes both `h1` values and fails closed on anything missing. Its positive
  control is a synthetic zip whose expected value comes from `dirhash`, not from
  the checker's own code.

The owner's change must add tests that vary each field below alone, with all other
fields valid. None of these tests exists yet, so none has been executed.

| Field varied | Production mutation it must catch |
| --- | --- |
| `sdk.provider` | Equality check narrowed to skip the provider |
| `sdk.module`, including case-only differences | Module compared case-insensitively or by prefix |
| `sdk.release_tag` | Tag compared by major and minor only, or accepted as a range |
| `sdk.checksum` | Zip checksum not compared, or `go.mod` checksum compared in its place |
| `sdk.go_mod_checksum` | `go.mod` checksum not compared |
| `sdk.model` name and version | Model omitted from the comparison |
| `mapping`, absent and differing | Absent mapping treated as matching |
| `target` revisions and identifiers | Target section omitted from the comparison |
| `accepted_runtime_composition_profile_sha256`, with a forged value | Pin gate removed, or caller data trusted before it |
| Evidence `module`, `release_tag` and `checksum`, one at a time | Evidence joined on fewer than all three |
| `go.sum` entry removed or changed | Join skipped when the entry is absent |

## Fork lock retirement

Retirement removes the fork from the active path. It does not rewrite history:

- **Evidence:** legacy results stay attributable to the legacy implementation and
  never qualify go-fabric.
- **Old lock contents:** repository history retains them. Do not keep an archived
  copy in the tree, where a tool or reader could mistake it for a live input.
- **Deferrals:** the [deferred ledger](matter-161-deferred.md) continues to name
  the Matter 1.7 elements, each with its re-entry trigger.
- **Gate change:** the contribution rule requiring the Linux SDK workflow for lock
  changes cannot apply to the change that retires that job. The owner must decide
  this explicitly; it must not be treated as a bypass. Until the replacement job
  runs, every hosted SDK row stays missing and nothing passes by absence.

## Ordering

Each step depends on the previous one. Stop at the first failure and use the
rollback section.

| Step | Performed by | Files | Exit condition |
| --- | --- | --- | --- |
| 1. Answer | Owner | None | Tag and module question answered on the issue, plus write authorization. |
| 2. Acquire and retain | Owner, separate verifier | None in this repository | Checksum steps agree; four manifests reconcile. |
| 3. Triage | Owner | None in this repository | Retained-excerpt record passes the structural validator; a reviewer reconciles it. |
| 4. Dry run | Owner | None | `evaluate_adoption_baseline` with the real values gives a valid proposal, `Unknown` provenance, all B rows `Unknown` and a production denial. |
| 5. Semantic identity | Public semantic owner | None here | An accepted Matter 1.6.1 target identity is supplied, or `mapping` stays `null`. |
| 6. Single change set | Owner | Every path in the trace table | One reviewable squash merge. See below. |
| 7. Confirm denial | Owner | None | Validator passes; tests report nonzero counts; accepted-looking metadata with an allocation spy gives a denial and zero allocations. |
| 8. Later work, each on its own trigger | Owner | Outside this transition | Hosted job, real adapter, qualification, consumer pins, feed cutover. |

**Step 6 is a single change** because no partial order keeps the gate green. Review
it in this order: lock and `go.mod`/`go.sum`, then evaluator, checker, tests,
example profile, scripts and workflow, and finally documents. Merging only part of
it leaves denial intact but the validator red.

**`go mod tidy` removes a requirement that no package imports**, and no package
imports go-fabric until the real adapter exists. Do not run it before then. The
checker's `go.sum` and `go.mod` join catches an accidental removal.

## Rollback to denial

**Denial does not depend on completing a rollback.** The profile pin stays `null`
in every state of this transition, so production admission is denied whether the
transition is untouched, half reviewed, merged or reverted.

| State when the problem appears | Action | Resulting denial |
| --- | --- | --- |
| Steps 1–5, nothing merged | Stop. Retain the attempt and its failure. | Legacy lock unchanged; `null` profile pin denies. |
| Step 6 in review | Close without merging. | Same. |
| Step 6 merged | Revert the entire change set as one unit. Do not revert single files. | Legacy lock and evaluator pair restored; `null` profile pin denies. |
| Step 6 merged and a later importer or consumer pin exists | Revert dependents first, then the change set. | A consumer without a factory stays unavailable, as the [composition contract](gateway-library-composition.md) requires; `null` profile pin denies. |
| A later reviewed change has set a profile pin | First restore `null`, then proceed as above. | Pin gate denies before any caller data is read. |

Triggers include a checksum disagreement, a moved or retracted tag, loss of a
retained input, a failed gate, a discovered rights problem and the owner withdrawing
the selection.

- **What rollback does not do:** it does not select another tag, fall back to a
  moving branch or re-accept the fork. The restored legacy lock is a record, not a
  new SDK decision. The go-fabric direction stands until the owner says otherwise.
- **A different tag** restarts at step 1 with a fresh triage.
- **Evidence:** retain failed attempts and bundles. Current assessments move to
  missing, failed or unverified as the build contract defines, and dependent rows
  reopen. Native evidence and partial last-known-good state are untouched, and no
  value becomes fresh.
- **Confirmation:** rerun step 7 after any rollback, and record exits and counts.
  Zero executed tests is not a pass.

## Boundaries and remaining acceptance

**Unknown after this document:** the real checksums; the module directive as
retained; the complete dependency, generator and rights closure; every B1–B8 row;
all runtime rows; Linux arm64 execution; Pi performance and latency. Matter
conformance, certification and interoperability are not claimed and do not follow
from an installed pin.

This transition requires no device access, live write, commissioning, fabric join,
deployment or upstream submission, and it authorizes none. Its offline evidence is
limited to synthetic and public fixtures, in-memory loopback and injected time. It
adds no real-bus transmit path and no production eBUS arming switch.

Production admission remains fail closed. eBUS MCP compatibility and GraphQL parity
are unchanged. No namespace or authorization gate is added before 0.9;
authentication and authorization remain 0.11 roadmap work.

The actual pin, gate migration and checksum verification remain owner operations
after an approved dossier. This document's own acceptance is manual review of each
trace row against current source.
