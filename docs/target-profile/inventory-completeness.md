# Inventory completeness and evidence joins

This is the proposed binding-local inventory acceptance contract for issue #15,
a partial slice of parent #8 / Project #2. It defines how the six domain
inventories must reconcile, independently of their future page contents. It is
not a seventh inventory, a minimal target subset, a semantic mapping format, or
an endpoint persistence design. Nothing here is stable before Helianthus 1.0.
No inventory checkpoint is released by this document.

## Current authority and limits

Helianthus adopts [go-fabric](https://github.com/SukramJ/go-fabric) and its
Matter **1.6.1** data model. The [reviewed lock](../../sdk/matter-sdk.lock.json),
validator and hosted job still select the legacy connectedhomeip fork's Matter
1.7 target; owner-controlled replacement remains outstanding. The legacy lock
is evidence of retained tooling, not the adopted model authority. Its sample
and generated headers cannot qualify go-fabric or supply native semantics.
Public main/tag links locate sources, not immutable evidence. The complete
retained normative model corpus and compatible accepted semantic target evidence
are still missing; reported source identities below do not replace either.

The [SDK baseline](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/SDK_BASELINE.md)
records a bounded Electrical Sensor fixture and the current mapping limitation:
`evse.ac.current` alone does not supply the mandatory target composition.
A sample controller success does not prove native facts, full inventory coverage,
a Helianthus endpoint, interoperability, certification, or physical qualification.

The [semantic metadata registry](https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/metadata.go)
composes Thermal, PV, storage, EVSE and infrastructure pack validators. That
registration does not enumerate all driver groups or prove any capability's
native evidence. In particular, six domain inventories do not imply six accepted
packs or a separate accepted metering pack. For reconciliation, the domain
partition is Thermal, PV, storage, EVSE, metering and infrastructure; membership
and coverage still require the sweeps below.

The supplied sources do not include the complete adopted normative model
contents or compatible accepted docs-semantic target evidence. These remain explicit completion
gaps. The [native CAN source sweep](can-driver-coverage.md) now records supplied
Growatt V1.04, common shared-field and Gree raw/candidate source coverage, with
qualification and target gaps kept separate. Lock metadata does not replace
inspecting target contents. No exhaustive adopted feature list, full CAN
capability list, or completed six-domain mapping is claimed here. No other
domain inventory page is an input or prerequisite to these row rules and cases.

## Full-family transition ledger

The [baseline](../SDK_BASELINE.md) lists nine source identities supported by
the supplied graded inspection result: Electrical Sensor revision 1, Power
Topology revision 1, EPM revision 3, EEM revision 2, Thermostat revision 11,
DEM revision 4, Energy EVSE revision 4, Aggregator revision 2 and Bridged Node
revision 3. **Proven** is limited to reported source presence; this is not
independent inspection or qualification. Feature and mandatory composition
closure remain **Unknown**. [Named deferrals](../adoption/matter-161-deferred.md)
cover Sensor Electrical Alarm, Power Topology ElectricalCircuit and the separate
DEM PowerRangeAdjustment snapshot gap; no universal 1.7 DEM delta is inferred.

Every original source row is retained in the following pages. Their new
reconciliation tables apply to every target reference, including unresolved
searches and reverse-direction candidates. **Changed** / **Unchanged** describe
only a stated identity comparison; **Unknown** preserves uninspected differences.
Source evidence, qualification, direction, loss, join acceptance and runtime
admission remain independent. No native sweep is repeated.

| Inventory / retained family coverage | Target disposition under the adopted baseline |
| --- | --- |
| [Storage](storage.md): RS-485 battery, optional battery, unavailable CAN source slots | Changed Sensor/Topology context; unchanged EPM revision; Unknown battery/Power Source/SOC/cell/temperature/status/limits and capacity joins; EEM/DEM source presence does not close them |
| [Air conditioning](air-conditioning.md): eBUS controller/remote sensors and Gree gaps | Unknown comparison for Thermostat revision 11; Unknown air-conditioner, temperature/humidity sensor, fan, mode/demand/setpoint and operation closure |
| [Heating](heating.md): system, zone, circuit, DHW, direct boiler, module/cylinder/solar | Unknown Thermostat/sensor/water-heater/pump/valve/mixer/counter/configuration joins; electrical context changed only as above |
| [PV](photovoltaic.md): SunSpec/Fronius, Growatt and Huawei | Changed Sensor/Topology; unchanged EPM revision; EEM revision 2 reported present, but sign/phase/energy/status/temperature and reset joins Unknown |
| [EVSE](evse.md): charging family reads and retained limit outcomes | Changed Sensor/Topology; unchanged EPM revision and sole accepted evse.ac.current mapping; Energy EVSE revision 4 and EEM revision 2 presence leave all additional joins Unknown |
| [Metering/infrastructure](metering.md): semantic fields, SunSpec meters, eeBUS raw values | Changed Sensor/Topology; unchanged EPM revision; EEM revision 2 presence, but sign/quantity/flow/phase/topology/status and counter closure Unknown |
| [Pumps](pumps.md): P1–P9 circuit, solar and boiler provenance | Unknown pump device/cluster/feature and command correspondence; no deletion or inferred off state |
| [Thermostats](thermostats.md): R01–R26 and O01–O06 room/remote/control roles | Unknown prior identity → Thermostat revision 11 source presence; exact sensor, heating/cooling, mode/preset and operation closure Unknown |
| [B524 matrix](b524-device-types.md): B01–B16 and RC01–RC08, T-S/T-H/T-U | Unknown sensor, thermostat, pump, valve/mixer, water and energy-management joins; DEM revision 4 does not establish activation/effect |
| [CAN sweep](can-driver-coverage.md): four selected-profile, four common and seven Gree groups | Changed electrical context; EEM/DEM/Thermostat reported present, while battery/HVAC/fan/diagnostic and reverse joins stay Unknown |
| Cross-family bridge composition | Unknown prior identities → Aggregator revision 2 / Bridged Node revision 3 source presence; Descriptor, parts/topology, identity continuity, reporting and mandatory closure Unknown |

All six domains and every representable family remain in scope. The last row
adds composition reconciliation, not a new native source or a replacement for
any inventory. No first-bridge subset can release the full-inventory checkpoint.

## Proposed reconciliation record

These are review columns, not a new API, serialization schema, or mapping file.
Use one row for each candidate source-fact to target-component join. A row's key
is the ordered tuple:

`(domain, native owner/profile, capability/fact, direction, target component)`

| Key field | Required interpretation |
| --- | --- |
| Domain | One of the six review domains above; a cross-domain reference does not change fact identity. |
| Native owner/profile | Public native-contract owner plus exact profile/variant within that owner; a family label alone is insufficient. If unavailable, use an explicit unresolved source slot and track its replacement. |
| Capability/fact | Owner-qualified fact identity and scope, such as service/measurement role, channel or phase where evidenced. Do not collapse distinct facts because their display names match. |
| Direction | Native-to-Matter observation, Matter-to-native operation, or independently evidenced directions in separate rows. Read evidence never proves a write path. |
| Target component | Candidate device type and revision, cluster and revision, feature selection and attribute/command/event role. An unresolved candidate is explicit, never an implicit Electrical Sensor default. |

An unresolved slot permits a gap row, not a match between unknown facts. Resolve
it through reviewed evidence and keep a trace from the old row to the new key.
Changing any key field creates a different join; document supersession rather
than silently overwriting an earlier decision. No row key is an endpoint number,
fabric identity, credential, or persistence key.

| Evidence columns | Required contents and separation |
| --- | --- |
| Source evidence | Public native contract location first; native owner/profile and driver group; inspected driver source location; supported fact, direction, units/scaling, scope and availability semantics; retained native evidence reference. Missing items remain explicit. |
| Source confidence and qualification | Proven / Hypothesis / Unknown for each claim, with its supporting evidence and limits; native qualification recorded separately. A synthetic case is not native Proven evidence. |
| Target evidence | Reviewed lock reference; inspected adopted data-model file and element; device/cluster revisions and feature conditions; mandatory closure ledger; public docs-semantic target mapping reference and its compatibility with the adopted baseline and future reviewed lock. |
| Join decision | Representable, blocked pending evidence, or not representable under the adopted target, with rationale, unresolved facts and public owning follow-up. These are proposed review dispositions, not runtime return values. |
| Reconciliation references | Canonical source-fact reference, cross-domain aliases, related joins, evidence conflicts, and the driver-group and target-sweep entries covered by this row. |
| Separate outcomes | Source qualification, inventory review disposition, composition-profile acceptance, matching build evidence and runtime admission each remain distinct. No one column upgrades another. |

A source fact is not admitted into a mapping solely because a driver emits a
value. First establish the public native documentation contract, then accepted
semantic evidence, then the target mapping at its public semantic owner. The
binding records the join and its limitations. It does not invent protocol facts
or move semantic contracts into this repository. Missing target evidence must
not erase native evidence or partial last-known-good state, or mark retained
values as fresh. No MCP or GraphQL behavior changes; their parity is preserved.

## Mandatory-feature closure and shared facts

For each candidate target, review the adopted device-type requirements and
all referenced clusters. Expand every unconditional requirement and every
requirement activated by the proposed feature selection, including dependencies
on other features, clusters and composition. Continue until there are no
unreviewed dependencies. Record the condition, applicability decision, evidence
for each required fact, and the dependent rows. Optional facts do not satisfy
unrelated mandatory requirements. A nullable mandatory fact still needs an
evidenced nullable representation; omission is not evidence of null.

A representable disposition requires both source semantics and complete target
closure, including an accepted public target mapping. A condition whose
applicability is Unknown blocks closure. Unavailable source or target evidence
is blocked, not proof of impossibility. A not-representable decision needs a
specific evidenced incompatibility; it is not a shortcut for an unavailable
source view. Alternative target selection needs its own fully evidenced join.

Use a canonical source-fact reference independent of domain to reconcile shared
readings. Its identity retains native owner/profile, service or measurement
scope and fact identity; direction remains explicit in each join. Two readings
with different scopes, phases or native owners are not duplicates merely because
the units agree. Cross-domain aliases refer to one canonical fact and its closure
ledger. Count distinct source facts once in source totals, and distinct target
components once in target totals; count domain row coverage separately. If a
fact supports multiple target components, review each join's closure separately.
This accounting neither allocates endpoints nor chooses endpoint sharing.
Conflicting purported aliases stay unresolved until evidence reconciles them;
never select the more convenient value or silently discard either source.

## Self-contained synthetic joins

All cases below are invented review examples. They assert no driver support,
protocol behavior, accepted mapping beyond the stated current baseline, or
runtime result. Identifiers describe synthetic families and facts only.

| Case and row key(s) | Source side | Target side and proposed disposition |
| --- | --- | --- |
| EVSE current: `(EVSE, synthetic-native/charging, ac.current, native-to-Matter observation, Electrical Sensor rev 1 / Electrical Power Measurement rev 3 / ActiveCurrent)` | Assume an evidenced current reading for a synthetic charging family solely for this example. The current baseline identifies `evse.ac.current`; the assumption supplies no other facts. | ActiveCurrent is optional. Missing evidenced Descriptor and Power Topology composition, non-unknown PowerMode, positive NumberOfMeasurementTypes, matching non-empty Accuracy and evidenced nullable ActivePower prevent closure. Record blocked pending evidence; never fill these from a sample or current magnitude. |
| Shared electrical reading: `(PV, synthetic-native/electrical, service-a.active-power, native-to-Matter observation, candidate power attribute)` and `(metering, synthetic-native/electrical, service-a.active-power, native-to-Matter observation, candidate power attribute)` | Assume both rows cite the same public native fact and service scope. Make the metering row an alias to the canonical fact. If scopes differ, retain two facts instead. | Record two domain rows but one distinct source fact. One identical target candidate is counted once; mapping, locked feature conditions and mandatory closure remain Unknown, so both joins are blocked. No second physical measurement, accepted target or endpoint is inferred. |
| Thermal modes: `(Thermal, synthetic-native/thermal, service-b.cooling, native-to-Matter observation, unresolved cooling component)` and `(Thermal, synthetic-native/thermal, service-b.heating, native-to-Matter observation, unresolved heating component)` | One synthetic Thermal service is assumed to evidence heating only. Its cooling capability is Unknown; shared service identity does not establish cooling support. | Keep separate rows and confidence claims. Heating still needs its own target mapping and closure; cooling is blocked for absent source evidence as well. Neither a shared service nor a target feature supplies cooling evidence. A future operation would require a separate direction row and native operation contract. |
| Absent driver source: `(infrastructure, unresolved-native/synthetic-family, unresolved fact, native-to-Matter observation, unresolved target component)` | Register the driver group as awaiting inspection. Assume driver source and its public native fact contract are unavailable; no capability is invented. | Block the source sweep entry and join. This placeholder is not a counted representable fact, a negative representability proof, or a completed driver assessment. Reconcile its key and any actual domain membership only after the source is inspected. |

The EVSE mandatory list above is supported by the baseline and
[readiness evaluator](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/runtime/production_readiness.py).
The full adopted normative data-model inspection is still required to establish exhaustive
closure. The shared-reading and Thermal cases apply proposed identity and
evidence rules; the metadata registry alone cannot prove their native facts.
The absent-source case is hypothetical; current bounded CAN source coverage
is recorded in the [native CAN source sweep](can-driver-coverage.md).

## Later two-way sweep and inventory checkpoint

The proposed review proceeds from unreviewed to dispositioned rows; evidence
changes reopen affected decisions and their dependent closure ledgers. These
are document-review transitions only. Incomplete evidence cannot transition a
row to representable. A checkpoint candidate returns to incomplete whenever
coverage, evidence conflicts or required closure remains unresolved.

1. **Source-to-target sweep:** enumerate the current driver groups from public
   driver/source views with their native contract and semantic owner. Account
   for every current representable group and its facts across the six domains.
   Each group gets a disposition and row references, or an evidenced reason it
   has no representable facts. Include unavailable groups in an explicit gap
   register so that missing source views cannot disappear from the denominator.
   The five metadata registrations alone cannot establish this denominator.
2. **Target-to-source sweep:** enumerate every relevant device type and cluster
   from the adopted Matter 1.6.1 normative model corpus, including mandatory dependencies and
   relevant optional features. Define and review relevance across all six
   domains; every excluded candidate needs a reason so a minimal Electrical
   Sensor subset cannot masquerade as complete coverage. Give each relevant
   target a representability decision, joined rows and closure ledger, or an
   evidenced incompatibility. Missing inspection stays blocked.
3. **Reconcile:** join the two ledgers through the row keys and canonical fact
   references. Resolve orphan groups, orphan target candidates, duplicate facts,
   conflicting aliases and unclosed mandatory dependencies. Report per-domain
   coverage and unique source/target totals separately, with explicit exclusions
   and unresolved counts. Six populated or partial pages do not establish this.

Releasing the full-inventory checkpoint requires a reviewable evidence bundle:

- A bounded, reviewed source-group roster and adopted target-tree roster, their
  inclusion/exclusion rationale, and complete two-way dispositions across all
  six domains; no unresolved availability or coverage gaps.
- Public native documentation and inspected driver evidence, accepted semantic
  evidence and docs-semantic mappings for representable joins; each target's
  fully evidenced mandatory-feature closure and revision compatibility.
- Canonical fact/alias reconciliation with no unresolved conflicting evidence,
  reproducible coverage accounting and reasoned negative representability
  decisions. Proven / Hypothesis / Unknown and native qualification remain
  visible, with no Hypothesis or Unknown standing in for required proof.
- Independent review of that bundle and an explicit recorded inventory decision
  by the owning operator/reviewer against the full scope of parent #8. Any
  deliberately unsupported target has an evidenced disposition, not a hidden
  reduction of scope. This issue alone cannot close that parent.

The missing adopted normative data-model and compatible accepted semantic contents still prevent
that release. The [native CAN source sweep](can-driver-coverage.md) supplies a
bounded source roster, not full target reconciliation or its missing evidence. A released inventory checkpoint would
mean completeness of the reviewed reconciliation, not that every candidate is
representable or admitted for production.

## B524 matrix coverage reference

The [B524 device-type matrix](b524-device-types.md) adds a bounded reverse
reconciliation under #23 / parent #8. Its B01–B16 observation bundles and
RC01–RC08 reverse bundles reuse the canonical [pump](pumps.md),
[thermostat](thermostats.md) and domain facts. The reverse slice accounts for
five family searches: thermostat/room control, temperature/humidity sensors,
pumps, valves/mixers and energy management. Acquisition references, operation
candidates and supported operations are separately dispositioned; no supported
reverse operation or accepted representable join is established by that slice.

Bundle counts are review coverage only. They are not unique source/target totals
or an exhaustive denominator. Repeated selectors are canonical references, not
extra facts; a new operation direction requires its own join and effect evidence.
Absent injection/actuation mappings, remaining groups/opcode families, other
profiles, solar scope conflicts, native/driver qualification and all adopted
target device/cluster/feature/mandatory-attribute closure remain explicit gaps.
Every other representable family remains required. Qualification against native
Go go-fabric on Matter 1.6.1 stays transport-agnostic and separate from native
semantics, inventory review, runtime and physical acceptance; no complete retained normative target corpus or independent SDK qualification
is supplied here. These counts and cross-links
do not lower the checkpoint threshold above or release either parent.

## Runtime and lifecycle remain unchanged

The current evaluator denies allocation before trusting caller metadata because
the reviewed lock has no accepted composition-profile digest. Passing inventory
review does not change the lock, accept a profile, fabricate build results, or
invoke the evaluator. Future profile acceptance and matching build evidence
remain separate work under the existing readiness contract.

The [Portal endpoint lifecycle contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/PORTAL_ENDPOINT_LIFECYCLE.md)
keeps selection intent separate from allocation and preserves native evidence
and partial last-known-good state when publication is withheld. Its proposed
lifecycle states and identity design remain unchanged. M41 persistence and
M42/M43 runtime work are unchanged; this document adds no runtime interface,
endpoint ledger, storage migration or launcher transition.

The inventory decision is an evidence review, not a runtime credential,
principal, permission or authorization check. No authorization gate is added
before 0.9; authentication and authorization remain 0.11 subjects. No new major
or experimental namespace is introduced. This document authorizes no live
operation, commissioning, deployment or device write.

Review of these columns, transitions and synthetic cases is contract-only.
Readability and offline link checks do not validate semantics, execute a
runtime, qualify a device, release the checkpoint or complete parent acceptance.
No transport matrix or hardware/live-fabric smoke result is claimed.
