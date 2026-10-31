# go-fabric contribution and defect-tracking workflow

This process advances [issue #67](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/67),
a partial child of [#59](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/59).
The umbrella remains open. Nothing here is stable before Helianthus 1.0.

## Consumption and contribution authority

The owner has adopted SukramJ/go-fabric as the Matter SDK on its Matter 1.6.1
data model. Matter 1.7-only elements are deferred. This direction does not
establish a selected dependency tag, complete license closure, passing build or
runtime qualification, or production admission. Updating the lock and its
baseline documentation is separately owned work; this process changes neither.

Consuming the Apache-2.0 library and granting rights to a contribution are separate
decisions. The supplied owner brief says the go-fabric CLA grants relicensing,
including commercial terms. Although the owner earlier agreed in principle,
the latest brief leaves owner CLA review pending before the first contribution.
Do not infer assent, sign, accept terms, or authorize a submission from adoption,
that earlier agreement, a local fix, a passing fixture or this document.

Issue #67 authorizes only this process and a blank template. It prohibits
contacting the maintainer, preparing an actual upstream-targeted report or patch
submission, and submitting anything upstream. The first real fix/report remains
carried owner work: owner review of the applicable CLA terms and explicit
per-contribution authorization must precede it. A similar contributor governance
agreement for Helianthus drivers is also carried owner work, outside 0.7; this
process neither adopts that agreement nor marks it complete.

## License and retained-input review

Before a later authorized contribution or SDK update, retain the supplied exact
release tag, origin evidence, applicable license/CLA text and review disposition.
Review Apache-2.0 obligations, copyright and attribution, required notices,
modifications and redistribution for the selected code and dependencies. Check
contributor authority over every proposed change independently of library use.
Record unresolved rights as Unknown and withhold affected redistribution or
contribution pending owner disposition; do not infer license closure from a
repository-level Apache notice.

Review generated output, generator versions, embedded matter.js model inputs,
third-party notices and any connectedhomeip XML used for cross-checks separately.
Record each input's provenance, applicable terms, transformation and output
obligations. Public availability does not establish permission to copy an input;
do not copy restricted XML into deliverables. Missing retained inputs, terms or
notices remain explicitly missing/unverified. This process performs no legal
assent or completed transitive/generated-input license audit.

## Exact-tag intake and update review

Use the existing [six-axis triage contract](../go-fabric-upstream-tag-triage.md)
and its [structural validator](../../qualification/tagtriage/record.go).
Receive an explicitly supplied previous/new exact-tag pair with public-safe
retained release notes and source excerpts. Establish origin and ordering outside
the validator; its opaque release identities do not authenticate a tag. Never
substitute moving main or evidence from another pair for missing inputs.

Review exactly `binding`, `api-breaks`, `commissioning`, `ble`, `target` and
`toolchain`. Each axis independently records relevance (`relevant`,
`not-relevant`, `unknown`), verification (`verified`, `unverified`) and evidence
IDs. Evidence has a unique ID, kind (`release-notes` or `source`), reference and
retained excerpt. Determinate relevance requires a nonblank supporting excerpt;
verified claims require determinate conclusions and excerpts for every citation.
Missing excerpts yield Unknown/Unverified, never an unsupported negative.
Overall binding can be not-relevant only when all five other axes are also
not-relevant. Reject duplicate/dangling citations and invalid records; retain
failed attempts and diagnostics separately. Validation checks structure, not
source authenticity, semantic sufficiency or qualification.

Use Hypothesis for unverified determinate conclusions, Proven only for narrowly
supported independently reviewed source claims, and Unknown for missing evidence.
None of these labels upgrades build, runtime, platform or physical qualification.
If evidence is lost, retain the old record historically and create a new record
with affected axes Unknown/Unverified and a conservative binding aggregate.
Preserve unrelated valid evidence and partial last-known-good state.

Every update needs a fresh comparison, patch reconciliation, license review and
review of the [build qualification](../go-fabric-build-qualification.md) and
[runtime qualification](../go-fabric-runtime-qualification.md) requirements.
Repeat affected qualification against the exact candidate tag and composition;
do not carry forward results across changed SDK, generated inputs, toolchain,
platform or adapter behavior without evidence. Retain attempted commands,
nonzero selected/executed counts, failures, skips and finite timeouts. Missing
inputs, zero executed tests or unavailable platform evidence remain unverified.
A source-triage result cannot approve a pin change or production allocation.

## Minimal local patches and defect ledger

Use an internal scoped issue for each evidence-backed candidate. Keep the
smallest reviewable patch needed for the binding requirement, with an independent
reproduction, negative control, rollback/removal condition and bounded maintenance
cost. Avoid opportunistic refactors, speculative APIs or a second SDK abstraction
that hides upstream behavior. Retain the unmodified exact-tag reproduction and
the patched comparison; a passing local model is evidence only for that model.

For each candidate, the internal defect ledger must retain:

| Field | Required content |
| --- | --- |
| Identity and scope | Opaque ledger ID, internal issue, affected exact tag and binding requirement; distinguish local adapter, model mismatch and suspected SDK defect. |
| Claim and grade | Expected versus observed behavior, narrow Proven/Hypothesis/Unknown label, evidence references and missing inputs; qualification status separately. |
| Reproduction | Public/synthetic fixture, independent expectation, finite budgets, injected schedule/time, commands, toolchain/platform tuple, nonzero counts, actual trace and failure output. |
| State and lifetime | Native evidence and partial last-known-good before/after, callbacks/resources/queues/bytes, cancellation/drain and cleanup observations where relevant. |
| Patch disposition | Minimal local diff reference, reason, review, regression evidence, rollback, maintenance owner and condition for removal. |
| Rights and authority | Code/generated-input provenance and notice review, unresolved rights, owner CLA review and per-contribution authorization status. |
| History and next action | Attempts, source-review links, affected qualification rows, review disposition, unresolved edges and next evidence needed. |

Candidate states are internal investigation, reproduced locally, locally mitigated,
awaiting evidence/owner review, or closed with retained rationale. Only later
separately authorized work may add upstream submission, acceptance or decline
status with actual evidence. Do not label a local patch as upstream accepted.
When a fix is delivered upstream, rebase/reconcile and rerun affected checks before
removing the local patch; retain both failed and successful histories. If upstream
declines, a fork requires an explicit owner disposition on scope and maintenance.
Do not silently fork or describe a local fork as upstream.

Internal boundedness, model and lifecycle gaps are candidates for investigation,
not upstream-ready reports. Examples to investigate include callback/event queue
bounds, dirty-path coalescing versus required observation delivery, endpoint
retirement/reuse, model/revision differences, and callback/store shutdown or
recovery behavior. Compare the actual selected source and independent reproducer
before attributing a defect. A stricter Helianthus requirement is not by itself
an upstream bug. See the existing [bridge candidate](../contribution-candidates/bridge.md)
for a bounded local proposal, not evidence of delivered SDK behavior.

## Evidence and safety boundary

Retain only public or synthetic reproducible inputs. Use device families rather
than models; exclude secrets, private captures, personal identifiers, serials,
network coordinates, capture times and contributor identities. Use public branch
or repository-relative file links rather than revision identifiers in published
material. Retained references must actually locate available evidence; a link
alone is not a retained input or a successful remote check.

Native protocol facts require their public native documentation contract before
semantic or Matter mapping. Preserve provenance, qualification, freshness and
partial last-known-good state without promotion. Production admission remains
fail closed under the current reviewed lock's absent accepted composition profile;
no triage record, local patch or passing fixture supplies admission authority.
eBUS MCP compatibility and GraphQL parity remain unchanged.

This process adds no device access, live writes, commissioning, deployment,
real-bus transmit path or production arming switch. A slave answer on a real bus
is a live write. Reproductions here use synthetic/public fixtures, in-memory
loopback and injected time only. Offline checks imply no real SDK, hosted,
Linux arm64, Pi performance/latency, transport, passive/physical, interoperability,
Matter conformance or certification result. Unknown remains Unknown. No new major
or experimental namespace or authorization gate is introduced; authorization
roadmap wording remains 0.11, with no new gate before 0.9.

## Blank internal report template

Leave this template unfilled in this issue. It is generic process material, not
an upstream-targeted report, maintainer message, CLA assent or submission recipe.
Do not fill it or turn it into an actual submission under issue #67.

```text
Internal ledger ID / internal issue:
Binding requirement and candidate category:
Supplied exact tag / origin evidence / affected composition:
Expected behavior and independent expectation source:
Observed behavior / claim grade (Proven, Hypothesis, Unknown):
Qualification status (separate from claim grade):
Public or synthetic retained inputs / provenance / missing inputs:
Reproduction procedure / finite budgets / injected schedule:
Toolchain and platform tuple / commands / exits / nonzero counts:
Unmodified result / patched comparison / negative control:
Failure logs / trace / resource and cleanup accounting:
Native evidence and partial last-known-good preservation:
Minimal local patch reference / rollback / removal condition:
Six-axis tag comparison reference / affected requalification:
License / generated-input / notice review and unresolved rights:
Owner CLA review status / per-contribution authority status:
Reviewer disposition / remaining evidence / next internal action:
```

Completion of this document does not complete the first upstream fix/report,
driver-contributor governance agreement, independent review, repository gates or
any qualification acceptance under #59.
