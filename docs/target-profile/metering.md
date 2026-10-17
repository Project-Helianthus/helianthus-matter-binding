# Metering source coverage and composition gaps

This is a partial, binding-local source-side inventory for issue #14, advancing
matter-binding#8. It cannot close that parent or release M42/M43. It proposes
coverage review, not a protocol-neutral contract or an accepted Matter mapping.
Matter mappings remain owned by the public semantic documentation repository.
Nothing here is stable before Helianthus 1.0.

## Evidence and reading conventions

The reviewed public source locators are:

- [S: infrastructure definitions](https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/infrastructure/definitions.go): `fields`, `dimensions`, `services`, `capabilities`, `index`, `Metadata`.
- [M: SunSpec meter definitions](https://github.com/Project-Helianthus/helianthus-modbusreg/blob/main/sunspec_models_meter.go): `integerMeterSunSpecPoints`, `floatMeterSunSpecPoints`, `meterMandatoryPoints`, `meterFieldID`, `meterEventSymbols`.
- [E: eeBUS raw typed values](https://github.com/Project-Helianthus/helianthus-eebusreg/blob/main/eebusraw/typed_value_v1.go): `NewTypedValueV1`, `DecodeTypedValueV1`, `canonicalTypedValueV1WithDepth`, `Value`, `Clone`, `ComputeHash`.
- [L: reviewed SDK lock](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/sdk/matter-sdk.lock.json): `sdk.data_model`, `sdk.blobs`, `target`, `accepted_runtime_composition_profile_sha256`.
- [B: baseline and readiness description](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/SDK_BASELINE.md): “Production gate” and “Hosted reproduction”.
- [R: admission implementation](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/runtime/production_readiness.py): `validate_composition`, `evaluate`.

These main-branch links locate public files; they are not immutable evidence or
proof that a remote page was checked online. Source claims below were reviewed
against the supplied source views. S proves field definitions, M proves native
point-definition coverage, and E proves a generic raw value representation.
None proves an installed driver, actual readings, Gateway publication, MCP or
GraphQL exposure, device qualification, interoperability, certification, or a
working Matter endpoint. Native decoder execution and hardware acceptance are
unverified in this contract review.

In the tables, `S`, `M`, `E`, `L`, `B`, and `R` refer to the exact public files
above; the adjacent symbol or point name is the evidence locator within that
file. `infrastructure.*` fields belong to `helianthus.pack.infrastructure`.
Service shorthand `phase`, `meter`, `grid_connection`, `circuit`, `site`, and
`feeder` means `infrastructure.service.<name>` and its corresponding
`infrastructure.dimension.<name>`. These are text dimensions, not evidence of
physical wiring. S defines read capabilities and empty Operations/EffectRules;
no row implies control or a write operation.

Each row separates native evidence, semantic representability, target evidence,
and admission. **Proven** means only the cited source fact; **Hypothesis** marks
an unaccepted mapping candidate; **Unknown** marks absent evidence. The status
“supported” is restricted to source representation. “Withheld” retains evidence
without projecting it; “unavailable” means unavailable in these reviewed sources,
not universally unsupported by Matter or any driver.

## Locked target boundary and shared gaps

L explicitly records data-model version `1.7`; Electrical Sensor `0x0510`
revision 2; Power Topology `0x009C` revision 2; Electrical Power Measurement
(EPM) `0x0090` revision 3; and optional `ActiveCurrent` `0x0005`. These are
**lock metadata**, not conclusions from inspecting the complete SDK schema.
No other target numeric identifiers, feature bits, attribute encodings, energy
cluster selection, or mandatory-cluster list are asserted here. Every table
entry `T?` means device type / cluster / feature / attribute composition is
**Unknown**; the lock's Electrical Sensor/EPM identifiers are only possible
review context. `TC?` adds the optional ActiveCurrent lock metadata as a
candidate, without proving feature applicability or a source mapping.

Every row has admission **Denied**: L pins no accepted composition-profile
digest and R `evaluate` returns denial before trusting caller composition or
build evidence. Representable data and passing SDK sample fixtures cannot
change this decision. No currently composed Helianthus hardware is established
by any row.

Every row carries shared gap **G**: evidenced Descriptor and Power Topology
composition, non-unknown PowerMode, positive NumberOfMeasurementTypes, non-empty
Accuracy matching that count and identifying each measurement type, nullable
ActivePower with evidence, accepted semantic mapping, and matching build/profile
evidence. B and R document/check these requirements; full schema reconciliation
remains open. They are not claimed as an independently inspected SDK mandatory
set. No measurement unit, point count, scale factor, event bit, text dimension,
or family name may substitute for Accuracy, topology, or measurement-type
metadata. Exact required range, resolution, error, feature and nullability
rules remain Unknown pending the locked corpus.

## Infrastructure semantic inventory

These rows inventory all 19 current S fields, grouped only where their
representation and gaps coincide. Native producers are unavailable in S;
Gateway publication is Unknown throughout. Direction is read-only (RO);
import/export below denotes energy flow, not a command direction. All rows have
G and Denied admission as defined above.

| Family / owner and source | Native evidence / qualification | SemReg pack field / service | Target evidence | Mandatory gaps | Direction | Status; loss or withholding reason | Proven / Hypothesis / Unknown; concrete falsifier | Public evidence locator |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.ac.active_power`, W, signed; phase | T?; ActivePower shape only in R | G; phase reference, sign convention | RO | Candidate; preserve negative values | Proven signed definition; Hypothesis projection; falsified by unsigned target or opposite sign convention | S `fields`, bounds ±10^7 |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.ac.reactive_power`, var, signed; phase | T? | G; reactive convention | RO | Candidate; no absolute-value conversion | Proven definition; Hypothesis; falsified by loss of sign or incompatible reactive convention | S `fields`, bounds ±10^7 |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.ac.apparent_power`, VA; phase | T? | G; phase association | RO | Candidate; VA is not W | Proven definition; Hypothesis; falsified by target requiring active power instead of apparent | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.ac.current`, A; phase | TC? | G; current kind and phase association | RO | Candidate; AC current is not automatically ActiveCurrent | Proven definition; Hypothesis; falsified by required active-current component absent from source | S `fields`; L `target.active_current_attribute` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.ac.voltage`, V; phase | T? | G; line-neutral versus line-line | RO | Candidate; reference must be evidenced | Proven definition; Hypothesis; falsified by target requiring a different voltage reference | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.ac.frequency`, Hz; grid_connection | T? | G; measurement location | RO | Candidate; no duplicate per-phase inference | Proven definition; Hypothesis; falsified by incompatible target scope | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.ac.power_factor`, ratio ±1; grid_connection | T? | G; factor/sign interpretation | RO | Candidate; ratio is not percent | Proven definition; Hypothesis; falsified by differing target factor convention | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.energy.import`, `infrastructure.energy.export`, kWh; meter | T?; energy target Unknown | G; accumulation/reset basis, flow reference | RO; import/export separate | Candidate; never merge counters | Proven definitions; Hypothesis; falsified by swapped direction or incompatible accumulation interval | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.power.apparent`, VA; grid_connection | T? | G; aggregate scope | RO | Candidate; not the phase field | Proven definition; Hypothesis; falsified by target requiring phase-local rather than aggregate data | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.power.import_active`, `infrastructure.power.export_active`, nonnegative W; grid_connection | T? | G; flow convention and derivation | RO; import/export separate | Candidate; not interchangeable with signed phase power | Proven definitions; Hypothesis; falsified by source net power without a documented split rule | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.power.import_reactive`, `infrastructure.power.export_reactive`, nonnegative var; grid_connection | T? | G; reactive flow convention | RO; import/export separate | Candidate; preserve direction | Proven definitions; Hypothesis; falsified by unresolved quadrant or sign convention | S `fields` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.status.breaker` closed/open, `infrastructure.status.interlock` active/clear; circuit | T? | G; source status interpretation | RO | Withheld from metering; status is not command support | Proven symbols; Unknown target/publication | S `fields`, `index` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.status.connection` connected/disconnected, `infrastructure.status.readiness` not_ready/ready; grid_connection | T? | G; status provenance | RO | Withheld; readiness does not grant endpoint admission | Proven symbols; Unknown target/publication | S `fields`; R `evaluate` |
| Electrical infrastructure / SemReg, S | Native unavailable; unqualified | `infrastructure.status.fault` clear/present; site | T? | G; fault interpretation | RO | Withheld; do not collapse arbitrary native events | Proven symbols; Unknown native mapping/target | S `fields` |
| Infrastructure grouping / SemReg, S | No physical topology evidence; unqualified | site/grid_connection/feeder/circuit/phase/meter services and read capabilities | T?; Power Topology is lock metadata only | G; physical relations and endpoint allocation | RO | Supported semantic grouping; physical topology unavailable | Proven text dimensions/services; Unknown hardware graph | S `dimensions`, `services`, `capabilities` |

## SunSpec meter native coverage

M supplies integer/scaled and floating meter families, with mandatory-point
sets for integer models 201–204 and floating models 211–214. These are public
protocol model identifiers, not product models. A definition list is not proof
of live support, successful decoding, or publication. Mandatory native points
vary by family and do not establish target mandatory facts. All rows are RO,
unqualified, Gateway publication Unknown, target T? unless TC? is stated, and
admission Denied with G. SemReg columns are proposed correspondences, not
existing driver-to-SemReg adapters.

| Family / owner and source | Native evidence / qualification | SemReg field / service | Target evidence | Mandatory gaps | Direction | Status; loss or withholding reason | Proven / Hypothesis / Unknown; concrete falsifier | Public evidence locator |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| SunSpec integer/scaled and float meter / modbusreg, M | Defined `A`, `AphA/B/C`, A; unqualified | Phase points candidate `infrastructure.ac.current` / phase; total has no evidenced aggregate-current correspondence | TC? | G; total versus phase; current kind | RO | Candidate phase; total withheld; no total-to-phase relabeling | Proven points; Hypothesis mapping; falsified by negative current outside S bounds or non-active current semantics | M point builders, `meterFieldID` |
| SunSpec meter / modbusreg, M | `PhV`, `PhVphA/B/C`, V; unqualified | Phase candidate `infrastructure.ac.voltage` / phase; average withheld | T? | G; neutral reference, phase identity | RO | Candidate phase; no average-as-phase default | Proven line-neutral IDs; Hypothesis; falsified by missing neutral/reference identity | M `meterFieldID`, point builders |
| SunSpec meter / modbusreg, M | `PPV`, `PPVphAB/BC/CA` and `PhVphAB/BC/CA` aliases, V; unqualified | No explicit line-pair field in S; phase voltage requires further contract | T? | G; line-pair representation | RO | Withheld; line-line cannot silently become line-neutral | Proven alias IDs; Unknown safe semantic/target mapping | M `integerMeterSunSpecPoints`, `meterFieldID` |
| SunSpec meter / modbusreg, M | `Hz`, Hz; unqualified | Candidate `infrastructure.ac.frequency` / grid_connection | T? | G; grid connection identity | RO | Candidate; preserve scope | Proven point; Hypothesis; falsified by an unassociated measurement location | M point builders, `meterFieldID` |
| SunSpec meter / modbusreg, M | `W`, `WphA/B/C`, W, signed integer or float definitions; unqualified | Phase candidate `infrastructure.ac.active_power`; total not a direct nonnegative import/export field | T? | G; sign/flow convention; total split | RO | Candidate phase; total split withheld | Proven typed points; Hypothesis; falsified by sign loss or applying total to one phase | M point builders, `meterFieldID`; S `fields` |
| SunSpec meter / modbusreg, M | `VA`, `VAphA/B/C`, VA; unqualified | Candidate `infrastructure.power.apparent` / grid_connection and `infrastructure.ac.apparent_power` / phase | T? | G; scope, nonnegative validation | RO | Candidate; retain total/phase distinction | Proven points; Hypothesis; falsified by negative source or wrong aggregation scope | M point builders, `meterFieldID`; S `fields` |
| SunSpec meter / modbusreg, M | `VAR`, `VARphA/B/C`, var; unqualified | Phase candidate `infrastructure.ac.reactive_power`; total directional split not established | T? | G; reactive sign/flow | RO | Candidate phase; total split withheld | Proven points; Hypothesis; falsified by unresolved sign convention or fabricated import/export split | M point builders, `meterFieldID` |
| SunSpec meter / modbusreg, M | `PF`, `PFphA/B/C`; integer unit Pct, float unit PF; unqualified | Total candidate `infrastructure.ac.power_factor` / grid_connection; phase-specific field unavailable in S | T? | G; percent-to-ratio and float convention | RO | Candidate total only with public conversion contract; phase withheld | Proven unit difference; Hypothesis; falsified by treating integer percent as ratio or assuming float convention from its label | M point builders; S `fields` |
| SunSpec meter / modbusreg, M | `TotWhImp/Exp`, total and phase A/B/C, Wh; unqualified | Total candidate `infrastructure.energy.import/export`, kWh / meter; phase dimension not represented by those fields | T?; energy target Unknown | G; counter/reset basis; exact Wh-to-kWh conversion | RO; separate import/export | Candidate totals; phases withheld unless a dimension-preserving contract is supplied | Proven points/units; Hypothesis; falsified by Wh labeled kWh, reversed direction, or collapsed phases | M point builders, `meterFieldID`; S `fields` |
| SunSpec meter / modbusreg, M | `TotVAhImp/Exp`, total and phase A/B/C, VAh; unqualified | No apparent-energy field in S | T? | G; apparent-energy semantics | RO; separate import/export | Withheld; not active kWh | Proven point definitions; Unknown target/semantic mapping | M point builders, `meterFieldID` |
| SunSpec meter / modbusreg, M | `TotVArhImpQ1/Q2`, `TotVArhExpQ3/Q4`, total and phase A/B/C, varh; unqualified | No reactive-energy/quadrant field in S | T? | G; quadrant and accumulation basis | RO; separate quadrants/flows | Withheld; preserve `Ph`/`ph` native aliases and canonical phase IDs | Proven point and alias definitions; Unknown target mapping | M float/integer builders, `meterFieldID` |
| SunSpec integer meter / modbusreg, M | `A_SF`, `V_SF`, `Hz_SF`, `W_SF`, `VA_SF`, `VAR_SF`, `PF_SF`, `TotWh_SF`, `TotVAh_SF`, `TotVArh_SF`; unqualified | No standalone measurement correspondence | T? | G; decode/scale evidence | RO metadata | Supported native metadata; withheld as measurements; scale is not Accuracy | Proven scale links; Unknown executed decoding/target | M `integerMeterSunSpecPoints` |
| SunSpec meter / modbusreg, M | `ID`, `L` header and family-specific required flags; unqualified | No physical source identity or topology derived | T? | G; source/model instance identity | RO metadata | Supported native layout; topology unavailable | Proven header/mandatory sets; Unknown physical composition | M `meterHeaderSunSpecPoints`, `markMeterMandatory`, `meterMandatoryPoints` |
| SunSpec meter / modbusreg, M | `Evt` bitfield; Power_Failure, Under_Voltage, Low_PF, Over_Current, Over_Voltage, Missing_Sensor, reserved/OEM symbols; unqualified | `infrastructure.status.fault` is not an accepted event mapping | T? | G; event interpretation, unknown bits | RO | Withheld; preserve raw event bits and symbols rather than boolean collapse | Proven symbolic definitions; Unknown target/status mapping | M `meterEventSymbols`, `meterFieldID` |

The supplied M file does not establish decoder sentinel behavior, supported
registry dispatch, physical wiring, accuracy classes, or installed device
coverage. Those need their public native contracts and execution evidence
before acceptance; this inventory creates no native facts by extrapolation.

## eeBUS raw readings and unavailable composition facts

E supplies a bounded typed-value container, not a catalog of electrical
readings, units, phases, features or driver publication. These rows cover its
representable fact groups without assigning electrical semantics to JSON shape.
All admission remains Denied; all qualification and Gateway publication remain
unverified. eeBUS output binding implementation belongs to its driver owner.

| Family / owner and source | Native evidence / qualification | SemReg field / service | Target evidence | Mandatory gaps | Direction | Status; loss or withholding reason | Proven / Hypothesis / Unknown; concrete falsifier | Public evidence locator |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| eeBUS raw typed readings / eebusreg, E | Null, bool, string, safe signed integers; decimal/exponent JSON numbers and out-of-safe-range integers preserved as strings; Go floats rejected; unqualified | None evidenced; numeric text alone is not infrastructure quantity | T? | G; native measurement type, unit, phase, direction | RO representation | Supported raw representation; electrical projection withheld | Proven container rules; Unknown electrical interpretation | E `canonicalTypedValueV1WithDepth` |
| eeBUS raw structured readings / eebusreg, E | Bounded arrays/string-keyed objects, nested typed values; clone access and canonical hashing; unqualified | None evidenced; source association must accompany value | T? | G; source/feature/address association and public native contract | RO representation | Supported raw structure; no endpoint identity from value hash | Proven `Value`, `Clone`, `ComputeHash`; Unknown producer/publication | E named methods, `decodeTypedJSONTokenV1` |
| eeBUS unavailable/invalid raw input / eebusreg, E | Duplicate keys, invalid Unicode, excess bounds and unsupported values rejected; null is representable; unqualified | No missing-to-zero rule | T? | G; explicit missing/invalid/stale semantics | RO representation | Withheld on rejection; no fabricated fresh reading | Proven validation; Unknown downstream state transition | E `NewTypedValueV1`, `decodeTypedJSONV1WithDepth` |
| Electrical composition metadata / all source owners, S/M/E | Accuracy, enumerated measurement types/count, physical topology and target PowerMode not established by these views; unqualified | Text dimensions and units cannot supply these facts | T? | G in full | RO metadata | Unavailable in reviewed source slice; do not invent defaults | Proven evidence gap in these views; Unknown broader support | S metadata tables; M builders; E container; B “Production gate”; R `validate_composition` |

E's existing secret rejection is a representation safety behavior; this ticket
adds no credential, principal, authentication or permission check. It introduces
no authorization gate; authentication/authorization remain later lifecycle work.

## Proposed projection rules and manual synthetic review

Current behavior is limited to the source definitions and fail-closed admission
cited above. No new interface, adapter, transition, endpoint or mutation is
implemented. The following are proposed acceptance conditions for later mapping
work, manually reviewed here, not executed tests or native qualification:

| Synthetic case | Proposed result and falsifier | Source basis / remaining evidence |
| --- | --- | --- |
| A phase's signed power is −250 W | Preserve sign, phase, source and unit; withhold if target requires a different sign convention. Never turn it into an unsigned import/export reading without a reviewed rule. | S active-power signed bounds; M W phase IDs. Native sign convention and target encoding Unknown. |
| Aggregate import energy is 2500 Wh | Candidate 2.5 kWh only after exact unit conversion and counter semantics are accepted; export remains a separate counter. A result of 2500 kWh falsifies conversion. | M Wh, S kWh; target energy attributes and reset basis Unknown. |
| Line-pair AB voltage and phase-A line-neutral voltage coexist | Retain distinct native keys; withhold line-pair projection until representable. Equal numeric values do not justify merging. | M `meterFieldID`; S phase field has no explicit line-pair discriminator. |
| Integer PF is 50 Pct | Candidate ratio 0.5 only after native convention is accepted; raw 50 in ratio field fails its range. Float unit PF needs separate evidence. | M unit declarations; S ratio bounds. No conversion code claimed. |
| Two synthetic sources report the same raw value | Preserve distinct source associations; do not use E's value-only hash as physical identity. | E hashes canonical value only; full source envelope not supplied. |
| A valid phase reading is followed by an invalid or absent reading for another phase | Proposed adapter retains native evidence and unaffected last-known-good facts, explicitly marking missing/stale state; it must not zero, erase, or refresh unrelated facts. | E rejects invalid containers; S/M do not define lifecycle. Downstream transition and envelope evidence remain Unknown, not current behavior proven here. |
| A caller supplies complete-looking composition and successful build fields | Admission remains denied with the current lock; a permit falsifies the current boundary. | L null accepted profile pin; R `evaluate`. Manual source review only. |

Later projections must preserve deterministic envelopes, phase, flow direction,
signed power, energy unit and source identity, and keep MCP and GraphQL parity.
Those integration contracts are not present in these views and are not claimed
as implemented. Lifecycle contracts belong outside protocol/type CC0 trees.
Public native documentation must establish protocol facts before semantic or
target mapping acceptance; this inventory neither writes that contract nor
substitutes source labels for it.

## Locked-corpus reconciliation still required

Obtain the exact locked SDK corpus through the existing controlled reproduction
process, outside this offline inventory. Do not reconcile against another SDK or
an unpinned current tree. Required inspections/searches are:

1. `data_model/1.7/spec_sha`, `data_model/1.7/spec_tag` and
   `data_model/1.7/device_types/ElectricalSensor.xml`: verify corpus identity,
   device revision, required clusters and conformance expressions.
2. `data_model/1.7/clusters/ElectricalPowerMeasurement.xml` and
   `zzz_generated/app-common/clusters/ElectricalPowerMeasurement/AttributeIds.h`:
   search `PowerMode`, `NumberOfMeasurementTypes`, `Accuracy`, `ActivePower`,
   `ActiveCurrent`, feature/conformance declarations, measurement types, units,
   ranges, nullability, scaling and generated attribute identifiers.
3. Search `data_model/1.7/clusters` for `PowerTopology`, `Descriptor` and energy
   measurement declarations; resolve actual filenames and transitive types
   from that corpus before asserting attributes, revisions or mandatory facts.
   Search for voltage, frequency, apparent/reactive power, power factor,
   imported/exported energy, phase and line-pair representation; absence from
   supplied views is not a negative target-support finding.
4. `examples/all-devices-app/all-devices-common/device/types/electrical-sensor/ElectricalSensor.cpp`:
   inspect sample composition separately from schema requirements and hardware.
5. Resolve L's `mapping.path` in its declared semantic owner and reconcile the
   accepted mapping with this inventory. B reports only `evse.ac.current` in the
   accepted mapping; that is not an infrastructure-wide mapping or source
   qualification, and the mapping file itself was not supplied for review.

Parent acceptance remains open for full target inventory, native contracts,
producer identity and publication evidence, semantic mapping acceptance,
composition metadata, exact locked-schema/build evidence, independent review,
and any separately authorized hardware acceptance. Readability and offline
public-locator checks provide documentation evidence only. They cannot establish
transport behavior, passive capture evidence, production admission or release.
