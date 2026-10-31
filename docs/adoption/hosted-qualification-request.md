# Owner-run hosted adoption qualification request

This request implements the specification slice of
[#79](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/79),
a partial child of [#59](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/59).
The umbrella stays open. Nothing here is stable before Helianthus 1.0.

It specifies one finite job for the owner to run against one selected go-fabric
tag and retained bundle. Nothing has run: this document performs no network,
hosted, build, controller or device action, and it edits no workflow, pin or gate.
The [B1–B8 contract](../go-fabric-build-qualification.md) owns the rows and their
states, the [record validator](../go-fabric-build-record-validator.md) owns record
structure, and the [notices ledger](notices-ledger.md) owns rights outcomes. This
request adds no second ledger.

**Proven:** the existing synthetic validator produces the outcomes quoted in the
[rehearsal](#manual-rehearsal-against-b1b8) for those synthetic records.
**Hypothesis:** the proposed replacement scope and stage layout can be realized on
a hosted platform. **Unknown:** every real result below: producer and verifier
logs, native arm64 execution, generator reproducibility, license closure and both
controller outcomes. They stay Unknown until owner-run evidence exists.

## Owner inputs and preflight

The owner has not supplied these values to this revision, so each is **missing**.
None has a default. This document names no runner, image, tag, toolchain, controller
pin or go-fabric test, and none may be filled from an example, a moving branch,
upstream green metadata or a research comparison.

| Input | Required value | Current state |
| --- | --- | --- |
| O1 Tag | One exact stable `vMAJOR.MINOR.PATCH` release | missing |
| O2 Retained bundle | Location and computed digests of the source, dependency, toolchain and generator/input manifests, plus the finite notice inventory | missing |
| O3 Workflow scope | Replace the legacy job, or add a separate job and retire the legacy one later | missing |
| O4 Linux amd64 tuple | OS, architecture, ABI, CGO mode, compiler, linker, tags, features, toolchain manifest | missing |
| O5 Linux arm64 tuple | The same fields, separately | missing |
| O6 Execution environments | Runner for each tuple, including the native arm64 runner, and how its architecture is evidenced | missing |
| O7 Role separation | How producer and verifier stages are kept apart on the platform | missing |
| O8 Controller bundle | Retained Go-controller and chip-tool identities, their tuples, the retained reference host and the inspected store locations of each | missing |
| O9 Isolation mechanisms | How the no-network leg, the synthetic controller network and process containment are enforced and evidenced | missing |
| O10 Bounds | Every timeout, repeat count and window in this document, and a total job bound | missing |
| O11 Retention | Where the bundle is kept, for how long, and the public-safe sanitization rule | missing |

The owner writes these into one request record, which is retained first and
addressed by its computed digest. Its digest is listed in `Scope.Ledger.Fixtures`,
so a changed request changes every parent identity. The `AdoptionProfile` is
built from the request and never from results, as
[`qualification/buildrecord/adoption.go`](../../qualification/buildrecord/adoption.go)
requires. Stage A rebuilds it from the request and fails any difference.

Stage S0 runs before any source executes. It stops the job, with no later stage
started, when:

- any input is absent, non-canonical or a placeholder;
- the bundle's declared tag is not exactly O1, or a retained manifest does not match
  its O2 digest;
- either mandatory Linux architecture is undeclared;
- a bound is not a positive integer, or the stage bounds exceed the total job bound;
- a tuple or environment field is left to a host default.

The stop is recorded as missing (absent input) or failed (contradictory input),
with every dependent row and child listed. It is never reported as a build result.

## Finite stage plan

Each stage runs once. There are no automatic retries, background services that
outlive their stage, or stages added at run time. A rerun is a new attempt linked
to the same request and retains the earlier attempt, as the
[transition rules](../go-fabric-build-qualification.md#states-transitions-and-evidence-loss)
require. A cancelled or expired stage is unverified with its incomplete counts.

| Stage | Role | Runs on | Produces |
| --- | --- | --- | --- |
| S0 Preflight | Job | Any declared runner; no source execution | Accepted request or stop record |
| P1 Inputs | Producer | Clean runner | B1, B2, B3 evidence and frozen selections |
| P2-amd64 | Producer | Declared native amd64 runner | B4, B5, B6 children for that tuple |
| P2-arm64 | Producer | Declared native arm64 runner | B4, B5, B6 children for that tuple |
| P3 Closure | Producer | Clean runner | B7, B8 parents and notice records |
| V1 Inputs | Verifier | Clean runner | Independent B1, B2, B3 reconciliation and rediscovery |
| V2-amd64 | Verifier | Declared native amd64 runner | Replayed B4, B5, B6 children |
| V2-arm64 | Verifier | Declared native arm64 runner | Replayed B4, B5, B6 children |
| V3 Closure | Verifier | Clean runner | B7, B8 parents, notice records, comparison |
| C1 Go controller | Controller leg | Declared in O8 | Separate leg record |
| C2 chip-tool | Controller leg | Declared in O8 | Separate leg record |
| A Assessment | Job | Clean runner, no network | Assessment of all records; no admission output |

The minimal roster is the 28 results of the
[adoption wrapper](../go-fabric-build-qualification.md#adoption-wrapper-and-initial-assessment):
16 parents and 12 children. Every additional declared tuple adds children and
cannot replace either mandatory architecture. C1 and C2 are not B rows, do not
enter B5 and do not feed `AssessAdoption`. A B failure does not erase a leg result,
and a passing leg does not repair a B row.

### Producer and verifier separation

The verifier starts from the request and the retained bundle only. It uses a clean
runner and workspace, with no cache, output or environment shared with a producer
stage. It repeats discovery, build, tests and generation before reading any producer
log, then compares. Disagreement prevents pass. A producer-only result stays
unverified, and an absent verifier stage leaves every row unverified or missing.
Role tokens are labels: token inequality does not prove independence, so the
separation evidence required by O7 is retained with the record. This adds no
principal, credential or authorization gate.

### Offline execution environment

All B stages use a cleared environment that holds only the declared allowlist.
They set `GOWORK=off`, `GOPROXY=off`, `GOSUMDB=off` and `GOTOOLCHAIN=local`, keep
caches and toolchain inside the disposable build area, and use relative working
directories. A step that needs an undeclared download fails its row as a hidden
input. The job holds no secrets and needs none.

## Target execution

A tuple's B4, B5 and B6 children all run on its own declared native runner. Before
those children start, the stage retains architecture evidence: the kernel-reported
machine, the toolchain's host and target architecture, the ELF machine of the
executed test binaries, and any registered instruction-translation handlers.
Exclude serial numbers and host identifiers from that evidence.

`TargetExecution.Method` is `executed-target` only with that evidence joined to
the same B5 identity. A run under translation is labelled `emulated`. A produced but
unexecuted test binary is labelled `cross-compile`. Neither label satisfies the
native requirement. Cross-compilation may add B4 evidence only. If the evidence
cannot distinguish native from translated execution, B5 for that tuple is unverified.

### Source-derived test discovery

No test name is known here. For each tuple and role, discovery runs on the native
runner against the retained source with the tuple's tags and CGO mode:

1. List packages and their test closure with `go list`, and reconcile them with B2.
2. List top-level tests per package with `go test -list` on the native test binary.
3. Inspect the source of each candidate. Classify it as selected, or as excluded
   with a reason, covering clock, seed, locale, filesystem-order and concurrency
   assumptions.
4. Exclude anything needing a listener, socket, fabric, device or hardware. Exclusion
   is not coverage, and that behavior stays in the broader denominator.
5. Freeze the identities as `import/path.TestName` in `Command.TestIDs`, with fixture
   digests and rationale, before any selected test runs.

Selections for the two tuples may differ only where source constraints differ, and
each difference is listed. The verifier rediscovers independently and compares:
a differing candidate list or an unjustified exclusion fails B3 and B5.

Counts come from `go test -json` events, not from the exit status, because a run
selector that matches nothing still exits zero. Selected, executed, passed, failed
and skipped count declared top-level identities, so `Selected` equals
`len(TestIDs)`. Subtests stay in the log. Each repeat is a separate clean invocation
with `-count=1`. An executed test outside the selection is a failed count mismatch.
A skip of a selected test leaves B5 unverified, whatever the reason.

### No-network deterministic B5 leg

B5 runs inside the O9 no-network mechanism: no non-loopback interface, no route, no
discovery daemon and no controller process. The stage retains the interface and
route listings from inside that environment, before and after the tests. Missing
listings leave B5 unverified. A selected test that needs the network is reclassified
as excluded in a new linked attempt. Isolation is never relaxed to make it pass, and
an in-memory transport counts only where source inspection shows it opens no socket.

### Two clean generator runs

B6 first inventories every generated path, and separates upstream-supplied generated
code from locally regenerated code. Then each role, on each tuple, runs generation
twice from retained inputs in two new empty directories and compares exact bytes
across both runs and both roles. Any difference from upstream-supplied output is
recorded and explained. `Generate.Repeats` is at least 2.

Absent generator inputs or tools leave B6 missing and no substitute is fetched. There
is no no-generation shortcut, because the validator has none and an absent inventory
is not evidence of absence. Output shared across platforms is reconciled into both
tuple identities explicitly. Inputs reconcile with the
[notices ledger](notices-ledger.md); Unknown rights keep B2, B6, B7 and B8 from pass.

## Records

Each child run fills the existing `Run` and `Result` fields. The job retains the
following alongside them, addressed by computed digests rather than literal ones.

| Record | Required content |
| --- | --- |
| Identity | The full seven-part identity of the [contract](../go-fabric-build-qualification.md#report-identity-producers-and-verification), joined to the request digest |
| Exit | `Completed`, `Exit`, `TimedOut`, `Blocked`, the declared bound and the termination sequence |
| Counts | Selected, executed, passed, failed and skipped per run; declared repeats against actual repeats |
| Logs | Raw log digest and sanitized log; commands, counts, exits and identity survive sanitization |
| Artifacts | Sorted manifest of outputs, with types, modes and digests; the retention limit of any platform-hosted artifact |
| Environment | Allowlist values, architecture evidence and isolation listings, with no host identifiers |
| Exclusions | Every excluded test, target and leg, with its reason and affected denominator |
| Gaps | Every missing input and blocked step with its exact error |

If a log cannot be retained safely, withhold that result rather than replace the log.
Expiry or loss of any required artifact returns the affected row to missing and
reopens B8. Published summaries cite repository-relative artifact references and
never carry commit identifiers, private paths, network coordinates, capture times,
serials or personal environment dumps.

## Controller legs

C1 and C2 are separate from strict offline B5 and separate from each other. Each
starts on a fresh runner or equally clean environment, with its own synthetic
network, stores and sentinels. They exercise one
[reference](reference-bridge.md) synthetic sensor with Descriptor, Power Topology
and Electrical Power Measurement, using synthetic data only. No such reference host
exists today, so both legs are **missing** until O8 supplies a retained host built
from the same identity. An upstream sample does not substitute for it.

C1 shares a stack with the device, so a shared defect can pass both sides. It is a
separately retained outcome, not independent-implementation evidence. C2 is the
separately implemented controller. Other controllers are outside this request. A
pass in one leg does not stand in for the other, and a leg on one tuple says
nothing about another.

### Synthetic isolated network

The O9 mechanism gives each leg a network with no route to any real network, device
or fabric, using only addresses declared in the request. The leg retains interface
and route listings inside that network before W1 and after W7. If isolation cannot
be evidenced, the leg does not start and is unverified. Probing a real address to
test isolation is forbidden. Any discovery daemon must be started for the leg and
stopped with it; a shared host daemon is undeclared state and fails the leg. Setup
codes are synthetic public test values, never production credentials.

### Windows

Every window has an O10 bound with no default and ends on an observed condition.
A fixed sleep is not synchronization. These are test bounds: durations do not support
any latency claim, and they are unrelated to the logical D=8 drain budget of the
[runtime contract](../go-fabric-runtime-qualification.md).

| Window | Opens on | Pass condition | On expiry or error |
| --- | --- | --- | --- |
| W0 Setup | Stage start | Network, empty stores and sentinels exist, with digests recorded | Unverified; nothing starts |
| W1 Host ready | Host start | Host-side readiness observed | Unverified, or failed on host error |
| W2 Commissioning open | Readiness | Explicit window open, acknowledged, with its declared duration | Failed on refusal; unverified on expiry |
| W3 Commission | Open window | PASE, operational commissioning and a matching CASE exchange, each observed separately | Failed on controller error; unverified on expiry |
| W4 Read | Commissioned | Device types and server lists read through the controller and compared with the declared composition | Failed on mismatch |
| W5 Subscribe | Read done | Subscription established with declared intervals; first report, then a report of one synthetic value change, each seen by the controller | Failed on wrong value; unverified on no report |
| W6 Close | Reports seen | Subscription ended and commissioning window close observed | Unverified; no revocation is assumed |
| W7 Terminate | Any earlier end | Group termination and reaping below | Failed on survivor |
| W8 Inspect | After W7 | Store and sentinel checks below, then store removal | Failed on sentinel change |

W7 and W8 run on every path, including after failure. Host-side dispatch or attribute
mutation is not receipt: only controller-observed output counts. Latest-value
reporting is not evidence of per-observation delivery, as the
[reference boundary](reference-bridge.md#real-and-fake-completion-boundaries) states.

### Disposable stores

Each process receives its own new store, `HOME` and `TMPDIR` inside the leg's
disposable area. A store is never reused between legs, attempts or roles, and no
broad factory reset is used. After W3, every store location inspected for O8 must be
nonempty inside that area. An empty store means the process may have written
elsewhere, so the leg is unverified. Before-and-after listings of the declared scan
roots show anything written outside the area, and an unexplained write fails the leg.
Stores are never uploaded, as they hold fabric material even when synthetic. Only
names, types and sizes are retained, and the job records store removal. A failed
removal fails the leg.

### External sentinels

Before W1, the leg creates sentinels outside the disposable area: one at each
controller's inspected default store name in a separate external directory, and one
in the runner's original `HOME` and temporary directory. Their digest, type, mode
and size are recorded after W0, after W3 and after W7. Any change or absence fails
the leg even if every window passed. A detached control first runs the checker
against a modified copy, and the leg does not start unless the checker reports the
change.

### Process-group termination and reaping

Host, controller and helpers each run in an owned process group under an O10 bound.
[`scripts/run_owned_group.sh`](../../scripts/run_owned_group.sh) shows the required
shape: a finite timeout, then TERM, a bounded grace period, KILL and `wait`. Its
[safety test](../../tests/test_fixture_safety.py) covers only its own synthetic case.

Stop the controller before the host, and reap every process before W8 inspects
anything. A descendant that starts its own session leaves the process group, so the
leg also runs in the O9 containment and enumerates survivors after KILL. A survivor
fails the leg and retires that runner: no later stage reuses it. Keep exit status,
signal and timeout flag separately for each process. The legacy
[fixture script](../../scripts/run_upstream_fixture.sh) stops its directly started
app through a trap, which is not sufficient for these legs.

### Leg outcomes and failure records

A leg uses the four contract states, kept apart from B rows and from
Proven/Hypothesis/Unknown confidence. It passes only when every window passes, no
step is skipped, W7 and W8 are clean and independent review of the retained record
agrees; until that review it is unverified. An absent bundle is missing. A blocked step, timeout or unprovable
isolation is unverified with the exact cause. An observed assertion failure, survivor,
sentinel change or composition mismatch is failed. A socket-bind denial or missing
tool is an environment result, not a product failure or a pass.

A failure record is written on every non-pass path and contains: request digest,
attempt link, leg and window; expected and observed result; exit, signal and timeout
per process; termination sequence and survivors; store and sentinel results;
isolation listings; log references; and every window not reached, marked as such. A
later success links to it and never replaces it.

## Manual rehearsal against B1–B8

The validator column was observed by applying each edit once to the existing
synthetic adoption fixture in a disposable copy of the package. No real SDK, runner or
controller was involved. The state columns name the current assessment for the
affected row; none may become pass.

| Variant | Affected | Ledger state | `AssessAdoption` state | Job behavior |
| --- | --- | --- | --- | --- |
| Zero selected tests | B5, B7, B8 | unverified | failed: selection count mismatch | Exit zero is ignored; counts decide |
| Zero executed, all skipped | B5, B7, B8 | unverified | unresolved | No test success inferred |
| Required skip | B5, B7, B8 | unverified | unresolved | Skip stays in the record |
| Inventory kind missing | B2, B6, B7, B8 | missing | unresolved: missing inventory kind | Closure gap listed; nothing fetched |
| Rights Unknown | B2, B6, B7, B8 | unverified | unresolved: input rights | Ledger outcome stays Unknown |
| No inventory | B2, B7, B8 | missing | failed: inventory bound | Stage stops; dependents listed |
| Stale tag in results only | All | failed | failed: scope and join mismatch | Attempt retained as failed |
| Stale tag in profile and results | All | failed | **structurally-complete** | S0 and A compare with O1 and fail it; the validator cannot detect this |
| Verifier records absent | Every row | unverified | unresolved: 14 results missing | Producer claims stay unverified |
| Verifier role undeclared | Every row | failed | failed: invalid scope | S0 stops |
| Cross-compile-only arm64 | B5 arm64, B7, B8 | unverified | unresolved | B4 may stand; B5 cannot |
| Emulated arm64 | B5 arm64, B7, B8 | unverified | unresolved | Labelled, never substituted |
| Arm64 execution record absent | B5 arm64, B7, B8 | missing | unresolved | Gap recorded |
| Cross-compile labelled `executed-target` | B5 arm64, B7, B8 | failed | **structurally-complete** | Architecture evidence contradicts it; without evidence it stays unverified |
| Generator children absent | B6, B7, B8 | missing | unresolved: 4 results missing | No substitute output |
| Generator inputs undeclared | B6, B7, B8 | missing | failed: invalid inputs | S0 stops |
| One generator run | B6, B7, B8 | unverified | unresolved: incomplete repeats | Second clean run still required |
| Generator drift | B6, B7, B8 | failed | failed: output disagreement | Both outputs retained |

The two bold rows are limits of a structural check that takes supplied claims. Both
are closed by the S0, A and architecture-evidence requirements, not by the validator,
and the job must not treat `structurally-complete` as a qualification result. The
ledger and validator states differ by design for zero selected tests: the
[contract](../go-fabric-build-qualification.md#states-transitions-and-evidence-loss)
leaves the row unverified, the validator also reports the count contradiction, and
the record keeps both.

Controller legs have no validator. On review of this document alone, each one is
missing, because no reference host, bundle or run exists. A missing network listing is
unverified, and a changed sentinel or a survivor is failed, whatever the windows
report.

## Hosted and gate follow-through (not owned here)

The legacy workflow is **Matter SDK baseline**, job `exact-sdk-linux`, in
[`.github/workflows/sdk-baseline.yml`](../../.github/workflows/sdk-baseline.yml). It
builds the legacy connectedhomeip fork on one amd64 runner, and its results cannot
qualify go-fabric. Editing it is an external owner action, as are the tool and
controller pins and the run itself. No ticket owns these, and this request changes
none of them.

The proposed O3 answer (Hypothesis, not a selection) replaces that job with the stage
plan above. It is manually started by the owner, takes the request record as its only
input, holds read-only permissions and no secrets, and does not run on pull requests.
Moving the lock checks and fixture away from the legacy job belongs to the same owner
change, so that repository checks are not lost in the replacement.

| Follow-through item | Why it is affected |
| --- | --- |
| [`tools/validate_lock.py`](../../tools/validate_lock.py) and [`tests/test_workflow_pins.py`](../../tests/test_workflow_pins.py) | The pin scan covers every workflow file and fails closed; new actions need immutable references |
| [`sdk/matter-sdk.lock.json`](../../sdk/matter-sdk.lock.json) and [`scripts/checkout_sdk.sh`](../../scripts/checkout_sdk.sh) | They still select the legacy SDK; the pin migration is separate owner work |
| [`scripts/run_upstream_fixture.sh`](../../scripts/run_upstream_fixture.sh) and [`tests/test_fixture_safety.py`](../../tests/test_fixture_safety.py) | Tests assert the legacy fixture's text; retiring it changes them together |
| [`qualification/buildrecord`](../../qualification/buildrecord/adoption.go) | Stage A needs a small owner-provided caller, because the package has no command entry point |
| [`AGENTS.md`](../../AGENTS.md), [`README.md`](../../README.md) and the [SDK baseline](../SDK_BASELINE.md) | They describe the legacy hosted job and must change when it does |
| The [B1–B8 contract](../go-fabric-build-qualification.md), [record validator](../go-fabric-build-record-validator.md) and [notices ledger](notices-ledger.md) | They attribute closure and runs to this ticket's follow-through; the ticket delivers this specification only, and those runs remain owner actions |

## Claim boundaries

A complete run could support Proven only for its exact observed claims and tuples. It
would not establish a supported platform, interoperability, Matter conformance or
certification, Pi performance or latency, and no such claim exists today. It cannot
admit a production endpoint: the
[readiness evaluator](../../runtime/production_readiness.py) still denies allocation
because the reviewed lock has no accepted composition profile.

All fixtures are synthetic or public. The legs use no real device, real fabric, live
Gateway operation, deployment or upstream submission, and this request adds no
real-bus transmit path or eBUS arming switch. A slave answer on a real bus would be a
live write and is out of scope. Protocol facts still require their public native
documentation contract first. Native evidence and partial last-known-good state are
untouched, and evidence loss never refreshes a value. eBUS MCP compatibility and
GraphQL parity are unchanged. No namespace and no authorization gate is added before
0.9; authentication and authorization remain 0.11 work.

This document's check is manual source, claim, transition, loss, rehearsal and link
review. Its acceptance closes no parent criterion. Remaining: every owner input above,
the workflow and pin edits, the hosted run, independent review of its evidence, and
the broader #59 acceptance.
