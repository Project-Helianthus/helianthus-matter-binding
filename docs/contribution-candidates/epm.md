# Internal Electrical Power Measurement contribution candidate

This is an internal contribution candidate for the binding's own use and later
review, advancing [issue #36](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/36)
and partially advancing [parent #24](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/24).
It is executable local arithmetic and envelope validation. It is never an
upstream proposal, fork or patch submission, and asserts no go-fabric library
support. Nothing here is stable before Helianthus 1.0.

## Source and claim boundary

The supplied [metering inventory](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/metering.md)
records source-side unit labels, signed power, phase versus aggregate scope,
and gaps in Accuracy and composition evidence. Its proposed projection rules
require retaining source evidence, zero versus unavailable and independent facts
when another reading is rejected. The supplied
[inventory contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md)
requires public native documentation first, then accepted semantic evidence and
the public semantic owner's target mapping. This validator supplies none of
those acceptances. Moving public links locate files; no online verification or
immutable source verification is claimed.

**Proven:** the synthetic tests demonstrate the local conversion and retention
rules below. The supplied inventories document their own source coverage and
evidence gaps. Neither is native execution evidence.

**Hypothesis:** these local vectors may help later review of an Electrical Power
Measurement implementation. Internal arithmetic labels do not establish a
measurement type, sign convention, physical topology or accepted mapping.

**Unknown:** exact Matter 1.7 target definitions and go-fabric representation,
ranges, reserved values, null encoding, units/scales, features and transitive
composition closure. Required evidence is inspection of the reviewed locked
`data_model/1.7/clusters/ElectricalPowerMeasurement.xml`, its referenced types
and generated representation, Electrical Sensor requirements, Descriptor and
Power Topology dependencies, plus the matching go-fabric types, codecs and
build/test evidence. The accepted public semantic target mapping and public
native contract for each source fact must also be inspected. Source labels or
an SDK sample cannot supply absent evidence. No normative identifiers or
feature-support declarations are introduced.

Native source qualification, semantic mapping acceptance, target closure,
library support, runtime admission, freshness and physical acceptance remain
separate outcomes. A result of `exact` establishes only local arithmetic.

## Local contract

[`Validate`](../../candidates/epm/candidate.go) is pure standard-library logic.
It accepts no provider, callback, endpoint allocator, fabric or runtime handle.
There is no dependency on another contribution ticket.

Each input's key is the tuple `(Owner, Profile, Source, Scope, Phase, Fact)`.
All six fields are required, independent and case-sensitive. Each is 1–128
ASCII letters, digits or `._-/:`; there is no trimming, aliasing or default
phase. A supplied `none` token can describe a non-phase fact; it does not infer
an aggregate scope. These are local opaque labels, not a new semantic schema,
physical identity scheme or endpoint persistence key. Direction is fixed to
observation by this candidate's scope; there is no operation path.

`Decimal * 10^Scale` denotes the caller's measurement in `Unit`.
`OutputScale` selects an internal integer grid in the **same** unit. Both
scales are bounded to −18 through 18 for local resource control, independently
of target bounds. Decimal syntax is optional `-`, one or more ASCII digits,
and an optional decimal point followed by one or more digits, at most 128
bytes total. There is no binary float, exponent syntax, rounding or saturation.
Leading and trailing zeroes are retained in `Native`; negative zero converts
to integer zero with its original spelling retained.

The local unit vocabulary is `W`, `var`, `VA`, `A`, `V`, `Hz`, `ratio`, matching
labels discussed in the source inventory. The requested output unit must equal
the input unit exactly. This validates arithmetic labels only, without assigning
facts to units. Cross-unit conversions, PF percent conventions, reactive sign
conventions, current-kind inference, line-pair mapping and energy are withheld
from this slice. The native documentation and semantic owners must establish
their contracts first. No unit label becomes Accuracy or a measurement count.

| Synthetic input | Local grid | Result |
| --- | --- | --- |
| `-250` W, scale 0, phase a | W, scale 0 | `exact`, integer −250; key and sign retained |
| `12.345` W, scale 0 | W, scale −3 | `exact`, integer 12345 |
| `-2500` W, scale −1 | W, scale −3 | `exact`, integer −250000 |
| `0` W, available | W, scale −3 | `exact`, non-null integer zero |
| empty decimal, explicitly unavailable | W, scale 0 | `unavailable`, no integer |
| `0.1` W, scale 0 | W, scale 0 | `withheld`, `precision-loss`; no truncated zero |
| W requested as VA | VA, scale 0 | `withheld`, `unit-unknown-or-mismatch` |

Conversion uses bounded decimal parsing and arbitrary-precision intermediate
integer arithmetic, checks the division remainder, then checks signed 64-bit
fit. Any nonzero remainder is explicit `precision-loss`, and out-of-range
integers are `integer-overflow`; neither returns a usable integer.

Availability is explicitly `available` or `unavailable`. Unknown/missing
availability and unavailable-with-value conflicts are withheld. An available
row with missing/malformed decimal is withheld rather than treated as null.
Units and scales must be valid even for unavailable rows. Each row returns
`Native` including opaque `Evidence` and caller `Accuracy` bytes, detached from
the caller and other results. Nil and empty metadata remain distinct; metadata
is neither authenticated nor schema-validated. PowerMode and measurement counts
are absent from this API. Accuracy is never populated from scale or magnitude.

Duplicate complete keys withhold every conflicting row, including identical
duplicates. Distinct source, profile, phase, scope or fact never collide just
because values match. Results have a deterministic complete order over their
original fields, including rejected bytes and nil/empty metadata. This order
is an internal envelope convention, not a persisted identity encoding.

Validation is per-row: a malformed phase does not erase a valid sibling.
The function has no retained state, last-known-good store or age clock. It
returns evidence for an owner to reconcile; it neither removes nor refreshes
the owner's partial last-known-good facts. Withholding does not turn old facts
into fresh observations. MCP compatibility, GraphQL parity, Gateway native
decoding and eeBUS output ownership are unchanged because no integration is
added. Existing production admission remains fail closed and unchanged.

## Verification and review

The focused command is `GO111MODULE=off go test ./candidates/epm -count=1`.
[`candidate_test.go`](../../candidates/epm/candidate_test.go) uses independent
integer and envelope expectations. Key tests separately vary Owner, Profile,
Source, Scope, Phase and Fact; each catches dropping its named field from the
production duplicate key or returned envelope. Every field also has separate
empty, whitespace, control, Unicode, invalid-byte and oversized rejection cases.
Malformed value cases cover unit mismatch/unknown, both scale bounds,
availability, decimal syntax, loss and signed overflow. Permutations exercise
all 24 orders of a four-row vector with a duplicate conflict, an independent
zero and an independently withheld precision-loss row. Detachment tests mutate
caller buffers, result buffers, integer pointers and shared sibling buffers.

Specific guard mutations are removal of remainder rejection (must fail the
positive, negative and scaled loss cases), removal of `IsInt64` (must fail both
overflow signs and scaled overflow), accepting unknown availability or unit
(must fail those withheld cases), and replacing unavailable by zero (must fail
the nullable case). Removing either metadata clone must fail detachment. These
are test-to-mutation mappings; executed disposable mutant evidence is recorded
separately in the local work report.

Documentation acceptance requires manual source, claim, transition, loss,
example and link review. No token test establishes semantic acceptance.
Focused tests do not establish full CI, native qualification, runtime
integration, transport behavior, passive capture evidence, hardware smoke,
interoperability, certification or production readiness. Independent review,
matching committed-head checks and the parent's remaining integration and
physical acceptance are outside this slice. No special merge hold is added.
