# Offline upstream-release triage

Issue #35 supplies a partial implementation slice of #24 / Project #2. This
finite record reviews a **supplied** previous/new release pair; it neither
selects an SDK nor acquires upstream material. Nothing here is stable before
Helianthus 1.0. No current upstream release or delivered API is asserted.

The local [record validator](../qualification/tagtriage/record.go) owns only
structural validation. The [build qualification contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/go-fabric-build-qualification.md)
and [runtime qualification contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/go-fabric-runtime-qualification.md)
remain separate requirements. Links locate public main documents; they are not
immutable evidence or a claim that remote links were fetched.

## Finite record and review meaning

`Record` contains two distinct, nonempty, unpadded opaque release identities,
retained `Evidence`, and exactly one `Classification` for each axis below.
Release labels identify the supplied comparison, not ordered semantic versions;
the caller must establish upstream origin and previous/new ordering. No source
control identifier is needed. Never reuse evidence from a different release pair.

| Axis | Question for the supplied comparison |
| --- | --- |
| `binding` | Is there an overall impact on the Matter SDK boundary? |
| `api-breaks` | Are binding-facing API removals or incompatible changes relevant? |
| `commissioning` | Are new commissioning APIs relevant to the binding? |
| `ble` | Is there a BLE change relevant to the binding? |
| `target` | Are target/platform changes relevant to the binding? |
| `toolchain` | Are compiler, generator or other toolchain changes relevant? |

Each axis has independent relevance (`relevant`, `not-relevant`, `unknown`) and
verification (`verified`, `unverified`). Relevant means review impact, not
qualification or delivered capability. Not-relevant is an evidenced negative
conclusion, never the default for missing material. Unknown records insufficient
information, even when a note makes an attractive roadmap statement.

An evidence entry has a unique opaque ID, kind (`release-notes` or `source`),
nonempty reference and retained excerpt. References locate public-safe offline
artifacts or public source views; the validator performs no I/O. Each axis cites
entry IDs; dangling and duplicate citations are rejected. A determinate relevance
requires at least one cited nonblank release-note or source excerpt, even when
unverified. No available excerpt means Unknown/Unverified. An empty evidence list
is valid only with all axes Unknown/Unverified. An entry with a missing excerpt
can be retained to record an input gap but cannot support a determinate claim.

Verified is the caller's claim of having reviewed the supplied excerpts for that
conclusion. Every cited entry must have an excerpt, and the conclusion must be
determinate. The validator cannot authenticate content, establish complete source
coverage, or detect a misleading excerpt. A reviewer must reconcile each excerpt,
release identity, reference and conclusion. An excerpt discussing one axis does
not semantically support the other axes merely because structural checks pass.
Unverified determinate conclusions are Hypothesis pending review; independently
reviewed conclusions may be labeled Proven only for their narrow supplied-source
claim. Unknown remains Unknown. These labels never upgrade native qualification,
acquisition, freshness, runtime behavior or physical acceptance.

Missing or duplicate release identities, omitted/duplicate/unknown axes, invalid
enums, malformed evidence entries and evidence-free verified claims are rejected.
Overall binding not-relevant also requires all other axes not-relevant: an API
break marked relevant contradicts it, and an unknown child prevents that negative
aggregate conclusion. A binding relevant or unknown conclusion may coexist with
other unknown axes; relevance does not require pretending all questions are known.

`Validate` returns a detached copy on success, including evidence, classifications
and nested citation slices. Failure returns an empty record and an error, without
mutating the input. It has no admission, scheduling or qualification result.

## Synthetic examples and loss handling

| Supplied fixture | Expected classification |
| --- | --- |
| Excerpt explicitly reviews all six axes without changes | All not-relevant; verified only after excerpt review. |
| Excerpt describes a binding-facing API removal | API breaks and binding relevant; independent evidence needed for other conclusions. |
| Excerpt introduces a commissioning API | Commissioning and binding relevant; no commissioning execution is claimed. |
| Notes predict future BLE delivery without confirming implementation | BLE unknown/unverified; timing remains an expectation. Binding cannot be not-relevant while BLE impact is unknown. |
| Release identities available, notes/source excerpts absent | Every axis unknown/unverified; structurally valid missing-input record. |

A changed release pair requires a fresh comparison. If an excerpt is lost, retain
the prior record as historical evidence; create a new record with affected axes
unknown/unverified and without unsupported citations. Recompute the binding
aggregate conservatively. Never silently upgrade unknown to not-relevant, erase
failed attempts, or overwrite unrelated valid evidence. This package does not
persist records or alter native evidence or partial last-known-good state.

## Later nightly orchestration procedure (proposal only)

1. A later orchestrator receives an explicitly supplied previous/new pair and an
   offline public-safe bundle of release notes and source excerpts. Establish
   origin, pair identity and excerpt coverage outside this validator. Missing
   input creates an unknown record; no fallback to a moving upstream branch.
2. Declare exactly six axes, review each excerpt against its claim, retain
   references and distinguish reviewed claims from unverified hypotheses. BLE
   timing from notes remains an expectation until separate delivered-source and
   qualification evidence exists. Do not invent APIs from roadmap wording.
3. Call `Validate`. Retain invalid attempts and diagnostics separately; withhold
   invalid new records. Keep prior valid comparisons as historical records, not
   fresh conclusions for the new release. Store the detached valid record only
   through a future separately owned persistence boundary.
4. A reviewer reconciles the comparison with the build/runtime contracts and
   records follow-up needs. Relevant or unknown results request investigation;
   not-relevant results close only this bounded source review. No result upgrades
   production readiness or changes a dependency pin.

This procedure installs no schedule, workflow, network fetch, upstream submission,
listener or live integration. It authorizes no deployment or device action.
Production admission remains fail closed under the reviewed lock's absent
accepted composition profile; triage cannot supply it. Native protocol facts
still require their public native contract before semantic/target mapping.
eBUS MCP compatibility, GraphQL parity, deterministic envelopes, G5 D1-D5/O8,
and eeBUS output ownership in helianthus-eebusreg are unchanged. No new namespace
or authorization gate is introduced; authentication/authorization roadmap work
remains 0.11.

## Acceptance boundary

Run only the dependency-free focused check:

```sh
GO111MODULE=off go test ./qualification/tagtriage -count=1
```

Discover tests separately and require nonzero selected/executed counts. Synthetic
tests cover unchanged, breaking, commissioning, BLE-unknown and missing-input
comparisons, every missing axis, every unsupported negative upgrade, invalid
identities/evidence, contradictions and slice detachment. Manual documentation
acceptance reviews source claims, transitions, evidence loss, examples and links;
readability or token matching alone is insufficient.

This slice does not close #24. Independent review and applicable committed-head
checks remain. Full CI, native library/build/runtime qualification, transport
conformance, live-fabric smoke, interoperability, certification and physical
acceptance are unrun and unproven here.
