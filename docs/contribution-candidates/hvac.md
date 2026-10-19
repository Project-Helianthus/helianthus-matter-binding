# Internal Thermostat and HVAC contribution candidate

This is an **internal contribution candidate for the binding's own use and
later review**, implementing [issue #39](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/39)
as a partial slice of [parent #24](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/24).
It is never an upstream proposal, fork or patch submission. Executable vector
validation does not establish go-fabric library support. Nothing is stable
before Helianthus 1.0.

## Source authority and scope

Protocol facts belong first to the [public native documentation contract](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md),
with projection context in the [public mapping contract](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/b524-semantic-mapping.md).
The supplied binding [air-conditioning inventory](../target-profile/air-conditioning.md)
records controller and remote sensor family coverage, unknowns and loss. Its
measured-room, configured/manual/effective target, capability, selected-mode and
command distinctions motivate these vectors. This candidate introduces no
native register assignments, codebook, driver behavior or public semantic
mapping. The [inventory join contract](../target-profile/inventory-completeness.md)
requires native contracts before semantic acceptance and target reconciliation.
Public main links locate the evidence owners; their remote contents were not
fetched or independently qualified here.

The implementation in [candidate.go](../../candidates/hvac/candidate.go) uses only
the Go standard library. It has no production Provider interface, Gateway
connection, SDK dependency, command interface, endpoint allocation or admission
integration. No sibling contribution candidate is consumed. eBUS MCP behavior
and GraphQL parity are unchanged.

## Input and loss contract

Each vector identifies one ordered join:

`(Domain, Owner, Profile, Scope, Fact, Direction, Target)`

All seven fields must be explicit, nonblank and free of surrounding whitespace.
Only `Thermal` and `native-to-Matter observation` are accepted here. `Target` is
an opaque unresolved candidate slot, never an asserted normative component.
Different owners, profiles, scopes or facts cannot join merely because their
labels or units match. Identical full keys withhold **all** duplicate rows;
changing any axis cannot overwrite or poison a distinct sibling. These are
review keys, never fabric, endpoint or persistence identities.

`Evidence` retains the public native contract reference, scoped source reference,
source confidence and independently supplied qualification. Source references
must carry the native selector and association axes required by their contract
(for example opcode, group, instance and register when applicable). These opaque
references are retained, not parsed or authenticated. A nonempty raw observation
is required. The validator checks supplied assertions, not their truth; synthetic
`Proven` labels in tests prove no native fact. `Hypothesis` and `Unknown` source
confidence withhold equivalence. Neither a supplied `Proven` label nor a candidate
projection qualifies a family or installation. Qualification remains separate.

`Heating` and `Cooling` each require an explicit `Unknown`, `No` or `Yes` state.
`Yes` and `No` require their own source reference. Unknown cooling cannot be
filled from heating, a service name, a target feature, or a mode label. Capability
does not establish selected mode or observed action.

Temperature vectors carry a finite Celsius observation, unit `C`, and one role:

- `measured-room` needs neither heating nor cooling support.
- `heating-configured-target`, `heating-manual-target` and
  `heating-effective-target` require explicitly evidenced heating support.
- The corresponding `cooling-` roles require separately evidenced cooling support.

Roles remain distinct in the output. Nil is unavailable, zero is a valid supplied
value, and NaN/infinity are withheld rather than converted to zero. No target
range, scaling or null representation is invented. Optional source constraints
supply finite inclusive minimum/maximum bounds and an evidence reference. Bounds
must be ordered and contain the sample. Omitting them asserts no limits; these
bounds cannot establish Thermostat target limits. Flow, dew-point, raw-status or
unknown-role substitutions are withheld. The validator cannot verify whether a
caller has truthfully labelled a source measurement.

Mode vectors carry `selected-mode`, the observed native code, and the complete
**supplied** codebook. Every entry, including unselected entries, requires its
own code, native meaning, explicit equivalent candidate label, reference, explicit
capability requirement (`none`, `heating`, `cooling`, `heating-and-cooling`) and
reviewed loss. No requirement is inferred from a label. Duplicate/blank codes,
missing equivalence, wildcard/default substitutions and unmatched observed codes
withhold the row. Required heating/cooling support must be `Yes` independently.
The caller must evidence codebook completeness; this validator cannot enumerate
an unseen native codebook or establish that an equivalence is semantically true.
Candidate labels are opaque, not Matter enum values. No manual-to-comfort,
away-to-off, schedule-to-auto or cooling-enabled-to-selected-mode rule exists.

Row loss and each codebook entry's loss are mandatory review fields: nil means
unreviewed and withholds; an explicit empty slice asserts no known loss for that
supplied assertion. Nonempty entries retain scheduling, preset, fallback origin,
role or other source distinctions. Loss is never silently discarded to make a
projection appear complete. Original raw bytes, evidence, codebook meanings,
constraints, qualification and loss remain in a detached `Source` even when the
candidate is withheld.

## Synthetic examples

These are invented review fixtures, not native protocol or target assignments.

| Supplied vector | Structural result |
| --- | --- |
| Room observation 21.5 C, both capability states Unknown, complete evidence and reviewed loss | Candidate measured-room value 21.5; all target and library outcomes remain Unknown |
| Native `native-manual` with supplied `candidate-manual-heating` equivalence and heating requirement, heating Yes, cooling No | Candidate supplied label only; source schedule loss retained |
| Same codebook with an unselected entry missing equivalence | Whole mode row withheld; selected entry cannot hide missing codebook evidence |
| Cooling-effective target with cooling Unknown | Withheld despite heating Yes; no inferred cooling or setter |
| Unavailable room-b sample alongside a valid room-a observation | Room-b withheld with raw evidence retained; room-a candidate survives; no retained state becomes fresh |

## Results and transitions

`Validate` returns independent per-row candidates or withholding reasons in a
canonical order, independent of row and codebook permutations. Ordering is an
internal value comparison, not a stable wire or persistence format. Inputs,
outputs, raw buffers, numbers, optional bounds and nested loss slices do not
share mutable storage. A bad row cannot clear a good sibling. There is no state
store or freshness update: callers must preserve partial last-known-good source
state under its owning lifecycle policy; nil/unavailable data is no replacement
sample and retained state must never be marked fresh by this result.

A candidate means only that supplied read-only assertions passed structural
checks. Semantic mapping acceptance, target mandatory-feature closure and library
support always remain `Unknown`; `ProductionAdmitted` always remains false.
A read-derived command, including a setter for an effective target, is rejected
by direction. No read result triggers a command or supports reversing a mapping.
The existing reviewed lock and runtime fail-closed admission remain untouched.

Before any target claim, reviewers still need the exact reviewed Matter 1.7
and go-fabric target definitions, compatible public docs-semantic mapping,
actual matched device/cluster files and revisions, enum correspondence, units,
scaling, nullability, feature applicability, all mandatory dependencies and
composition, plus matching build/reporting and controller evidence. No normative
cluster IDs, features, revisions, library API or target closure is supplied by
these vectors. Native qualification, accepted semantics, target closure, build
qualification, runtime and physical acceptance are separate outcomes.

## Review and bounded evidence

[Tests](../../candidates/hvac/candidate_test.go) use invented family/profile facts,
independent expected projections, malformed axes, withheld cases, all row
permutations, codebook reordering and mutation of detached storage. The
`TestEveryKeyAxis` missing and changed cases separately cover Domain, Owner,
Profile, Scope, Fact, Direction and Target: dropping each completeness check or
omitting each field from duplicate counting must fail its corresponding case.
For Domain and Direction, the changed row is invalid but the valid sibling must
remain a candidate. Optional bound axes, all mode-entry fields, source loss and
Unknown confidence have separate negative cases. No test depends on another
I ticket.

Loss and unknown guards are checked by removing row/entry loss requirements,
introducing a default mode, ignoring an unselected entry's equivalence, inferring
heating/cooling, accepting an operation direction, or upgrading the independent
unknown/admission outcomes. Disposable-copy mutation execution and exact exits
are recorded separately; these mappings describe what tests must catch, not
an assertion that every possible mutant was executed.

The focused offline check is `GO111MODULE=off go test ./candidates/hvac -count=1`.
Documentation acceptance requires manual source, claim, transition, loss, example
and link review. Synthetic checks establish neither native qualification nor
runtime integration, full CI, certification, interoperability, transport/device
acceptance or physical/live-fabric smoke. No authorization gate is introduced;
authentication/authorization remain 0.11 subjects. Ordinary independent review
and applicable repository gates remain necessary; parent #24 stays open.
