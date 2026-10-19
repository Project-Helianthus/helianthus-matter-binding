# Internal Electrical Energy Measurement contribution candidate

This is an internal contribution candidate for this binding's own use and later
review, implementing a partial slice of [issue #37][issue] and advancing
[parent #24][parent]. It is executable candidate logic, never an upstream
proposal, fork or patch submission. It does not claim go-fabric support.
Nothing is stable before Helianthus 1.0.

## Source and authority

The [PV inventory][pv] records a generated lifetime observation separately from
grid imported/exported energy, today's period separately from total energy, and
`counter_continuity_unavailable` as a loss. These are inventory observations of
the cited public sources, not new protocol facts or qualification. The public
[SunSpec native contract][native] remains the native authority; this candidate
does not decode it or establish a native-to-Matter mapping. The
[inventory acceptance contract][inventory] requires native evidence, accepted
semantic evidence and target closure separately. Missing source facts remain
Unknown; a family name or a signed value cannot fill them.

**Proven:** the local validator enforces the synthetic conventions below under
its focused tests. This says nothing about native source qualification.
**Hypothesis:** these distinctions may be useful in a later binding-owned energy
implementation. **Unknown:** the inspected Matter 1.7 energy target definitions,
feature conditions, units, ranges, nullability, device composition, accepted
semantic mapping and go-fabric library implementation. This record assigns no
normative cluster identifiers or feature support and supplies no target encoder.

Required evidence before a target implementation: inspect the energy-related
elements of the exact reviewed locked data-model corpus, their revisions and
mandatory/conditional closure; reconcile the public docs-semantic mapping with
native direction, reset continuity, period and scope; inspect the selected
go-fabric types/encoding and demonstrate matching build and fixture evidence.
An absent definition is Unknown, not a claim of unsupported functionality.

## Candidate contract

[Validate](../../candidates/eem/candidate.go) receives an expected identity and a
reading with independently supplied observed identity. The expected value is a
comparison contract, not an authority token. No self-attested metadata can
establish native truth or production admission.

The exact identity retains owner, profile, fact, scope, explicit import/export,
cumulative/periodic kind, reset epoch and interval metadata. Owner/profile/fact/
scope/reset labels must be nonempty opaque evidence labels, not real device
identifiers. The labels' meaning must eventually come from the owning contracts.
Generated energy is not automatically exported energy. Negative values never
choose direction; the candidate requires a nonnegative magnitude and explicit
direction even for zero.

Periodic readings require `HasInterval`, nonnegative start and strictly greater
end in one supplied synthetic tick frame. Cumulative readings have no interval
and require zero bounds; reset epoch remains mandatory. Kind and interval
presence are therefore structurally coupled. Bounds here are test coordinates,
not capture times or a target time representation. Changed reset epoch or either
bound is a different identity, not a continuity repair. No period, clock frame,
reset history or cumulative total is inferred.

`Magnitude * WhPerUnit` is an exact integer whole-Wh convention local to these
vectors. It is not Matter encoding or a native unit decoder. Both factors must
be representable under the checked signed-integer convention; zero scale,
negative magnitude, scale above the signed limit and multiplication overflow
withhold the vector. Fractional units and rounding are outside this slice and
must not be approximated by callers. Zero magnitude is valid only with known
value, continuity and completeness metadata.

Validation first checks both identities for completeness/structure, then exact
equality, known value, known continuity, completeness, numeric validity and
overflow, in that order. The first failing guard gives a deterministic reason
and no vector. Unknown continuity withholds both kinds; incomplete accumulation
or interval coverage withholds both kinds. A partial interval cannot become a
cumulative value, and even complete intervals are never summed into a total.
There is no aggregation, rollover repair, subtraction or cross-reading state.

Every outcome returns a detached native byte snapshot and the original reading
metadata, including withheld values and loss flags. It neither replaces nor
erases caller-owned partial last-known-good state and supplies no freshness
transition. A valid vector is a separate detached value; it has no production
admission or allocation path. The current [readiness boundary][readiness]
continues to deny production allocation. Qualification, accepted semantic
mapping, target closure, library support, freshness and physical acceptance are
independent outcomes. MCP and GraphQL paths are unchanged.

## Synthetic examples and acceptance

| Input | Internal outcome |
| --- | --- |
| Known complete imported periodic value 7, multiplier 1000, epoch-a, bounds 0..10 | 7000 Wh periodic vector with the same identity |
| Same positive value with explicit export | Separate exported vector, never chosen from sign |
| Same reading expected under epoch-b or bounds 0..11 | Identity mismatch; bytes retained, vector withheld |
| Missing direction, even with positive magnitude | Invalid identity, no sign inference |
| Unknown value whose stored magnitude is zero | Value unknown, no fabricated zero |
| Known lifetime value with continuity unavailable | Continuity loss, not a continuous total |
| Incomplete periodic coverage | Partial period; no cumulative derivation |
| Largest signed magnitude multiplied by two | Overflow; no wrapped result |

[Tests](../../candidates/eem/candidate_test.go) use independent numeric oracles,
all four direction/kind combinations, zero and signed-limit boundaries,
individual malformed fields on either input, valid identity mismatches,
all six permutations of three readings and bidirectional detachment on valid
and withheld results. The guard mutation mapping is:

| Test variant / field | Production mutation it must catch |
| --- | --- |
| owner, profile, fact, scope, direction, reset_epoch, start, end | Omit that individual axis from expected/observed equality |
| kind_and_interval_tuple | Omit kind, interval presence and bounds together from equality; no single valid kind-only variant exists under these structural constraints |
| owner_missing, profile_missing, fact_missing, scope_missing, reset_missing | Remove the corresponding nonempty identity guard |
| direction_missing, direction_generation_not_export | Admit absent direction or generated-as-export; remove direction membership guard |
| kind_missing, kind_daily_not_cumulative | Admit unknown kind via the switch default |
| interval_presence_missing, start_negative, end_equal, end_reversed | Remove the individual interval presence, lower-bound or ordered-end guard |
| cumulative_with_interval, cumulative_presence_only, cumulative_with_start, cumulative_with_end | Remove the individual cumulative interval-presence or zero-bound guard |
| unknown_not_zero | Remove known-value withholding |
| continuity_loss | Remove continuity-loss withholding |
| partial_interval | Remove completeness withholding |
| negative_not_direction, scale_missing | Remove nonnegative magnitude or positive-scale validation |
| multiplication_overflow, scale_overflow | Remove the respective numeric overflow guard |
| partial_cannot_be_total | Collapse cumulative/periodic identity and bypass partial-period withholding |
| positive_sign_cannot_supply_direction | Replace missing direction with a sign-derived value |
| permutation and detachment | Introduce cross-call state, reuse vectors, or alias native bytes |

This table is a reasoned mapping; executed mutation evidence is recorded
separately in the local acceptance report. It does not assert all possible
mutations were executed. Focused discovery and `GO111MODULE=off go test
./candidates/eem -count=1` establish local synthetic behavior only.

Documentation acceptance is manual review of source claims, authority,
transitions, losses, examples and public link destinations. This slice does not
close the parent. Independent review, committed-head gates, publication audit,
accepted mappings, target closure, library integration and any operator-directed
physical acceptance remain separate. No full CI, transport matrix, passive
capture, live-fabric smoke, certification or interoperability result is claimed.
No sibling contribution candidate is a dependency. No authorization gate is
introduced; future authorization planning remains a 0.11 subject.

[issue]: https://github.com/Project-Helianthus/helianthus-matter-binding/issues/37
[parent]: https://github.com/Project-Helianthus/helianthus-matter-binding/issues/24
[pv]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/photovoltaic.md
[inventory]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md
[native]: https://github.com/Project-Helianthus/helianthus-docs-modbus/blob/main/protocols/sunspec/read-only-core-v1.md
[readiness]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/runtime/production_readiness.py
