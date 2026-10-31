# EVSE source coverage and Matter composition gaps

This binding-local inventory advances [issue 13](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/13)
as a partial source-side slice of [parent 8](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/8).
It cannot close the parent or release M42/M43. It inventories Tesla Gen3 HSC
family read acquisition and separately retained current-limit outcomes. It
creates no protocol fact, semantic definition, target mapping, endpoint, or
production admission. Nothing here is stable before Helianthus 1.0.

Native lifecycle rules belong to the [read contract][R] and [current-limit
contract][C]; semantic definitions belong to [SemReg][S]. This document records
coverage and missing evidence at the binding boundary. Proposed Matter mappings
must be reconciled in the public semantic documentation owner before use.

## Evidence vocabulary and common boundaries

**Proven** means directly supported by the cited source contract or code, not
executed or hardware-qualified by this inventory. **Hypothesis** means a proposed
mapping with a stated falsifier. **Unknown** means missing source or locked-model
evidence. These labels are separate from native qualification, semantic
representability, target-model evidence, and production admission.

Every matrix row has production admission **denied**: the [lock][L] has
`accepted_runtime_composition_profile_sha256: null`, and [runtime evaluate][P]
returns denial before trusting caller composition or build metadata. No row
represents currently composed Helianthus hardware. Native version-scoped
qualification predicates and code's `QualificationQualified` semantic annotation
do not establish hardware, live-fabric, interoperability, or certification results.

In the tables, **G** is the unresolved Electrical Sensor composition gap recorded
by [SDK baseline, Production gate][B]: evidenced Descriptor and Power Topology
composition, non-unknown `PowerMode`, positive `NumberOfMeasurementTypes`, non-empty
fixed `Accuracy` matching that count, and an evidenced nullable `ActivePower`
fact. This is the repository's documented readiness boundary; the complete adopted
schema has not been inspected here. A nullable fact still needs evidence: absence
cannot silently become an evidenced null. SDK sample Accuracy or PowerMode values
are not Tesla device facts. No native payload or SemReg field name fills G.

**U** in a target cell means device type, cluster, feature, attribute/command,
IDs, revisions, units, and conformance are **Unknown** in the supplied adopted
corpus. It means withheld pending reconciliation, never a claim that Matter lacks
support. **U/G** also retains G if the Electrical Sensor route is considered.
Direction is from the native source toward a possible passive output unless
explicitly marked as an unimplemented command candidate.

## Retained legacy lock metadata, not a full schema inspection

Only these target identifiers/revisions are asserted, all from [lock `target`][L]:

| Lock key | Explicit metadata | Limitation |
| --- | --- | --- |
| `device_type` | Electrical Sensor, `0x0510`, revision 2 | No EVSE device type established by this key. |
| `power_topology_cluster` | Power Topology, `0x009C`, revision 2 | Mandatory composition remains unevidenced. |
| `electrical_power_measurement_cluster` | Electrical Power Measurement, `0x0090`, revision 3 | Full feature/conformance and encoding rules not inspected. |
| `active_current_attribute` | `ActiveCurrent`, `0x0005`, optional | Explicit target metadata only; no source/current-to-endpoint implementation. |
| `sdk.data_model.version` | `1.7` | A version label does not supply the missing data-model corpus. |

The baseline reports the accepted mapping has only `evse.ac.current`. The mapping
file named by the lock is not among the supplied views, so its full transformation
and conformance are Unknown here. Optional ActiveCurrent cannot discharge G.

## Native acquisition coverage

Driver **D** below is the Tesla Gen3 HSC family, owned by helianthus-modbusreg,
[`tesla_gen3_read_acquisition.go`][D], especially
`teslaGen3ReadQualifiedOperation`, `TerminalBody`, `retainLocked`, and `Retained`.
Each row cites [R, Operations in scope][R] and D. R is an operation/lifecycle
contract, not new wire evidence. Its referenced native body contracts require
separate inspection before any scalar extraction; this inventory invents none.
The source contracts bind these operations to `wc3_24_44_3`; known version labels
and compatible candidates do not extend that qualification.

| Family / driver owner and source | Native evidence / qualification | SemReg pack / field / service | Target device / cluster / feature / attribute | Mandatory gaps | Direction | Status / loss or withholding | Proven / Hypothesis / Unknown | Production admission | Exact public evidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Tesla Gen3 HSC / D: common system information | Family 4, tags 2 → 3, empty request; ACK retains native body; exact version scoped, no live qualification | No promotion demonstrated; metadata is not an EVSE field | U | G + body interpretation and identity/redaction evidence | Read → native retention | Supported native acquisition contract; withheld semantic/target extraction | Proven lifecycle; Unknown scalar semantics/target | Denied | [R: Operations in scope; Evidence record][R], [D: operation selector; TerminalBody][D] |
| Tesla Gen3 HSC / D: vitals | Family 6, tags 1 → 2; same native-only evidence boundary | Measurement/status fields below are representable definitions, not demonstrated vitals promotion | U | G + field/unit/phase/status mapping evidence | Read → native retention | Supported native acquisition contract; no inferred power/current/status | Proven lifecycle; Unknown extraction/target | Denied | [R: Operations in scope; What this contract does not claim][R], [D: operation selector][D] |
| Tesla Gen3 HSC / D: lifetime statistics | Family 6, tags 3 → 4; ACK body, no live qualification | `evse.energy.lifetime` definition is not a decoder | U | G + counter unit, scale, scope and reset evidence | Read → native retention | Supported native acquisition contract; withheld energy promotion | Proven lifecycle; Unknown energy correspondence/target | Denied | [R: Operations in scope][R], [D: operation selector; TerminalBody][D], [S: fields][S] |
| Tesla Gen3 HSC / D: system information | Family 6, tags 9 → 10; ACK body, no live qualification | No pack field demonstrated for native system metadata | U | G + explicit semantic owner and safe metadata projection | Read → native retention | Supported native acquisition contract; no device-identity publication | Proven lifecycle; Unknown semantic/target mapping | Denied | [R: Operations in scope][R], [D: operation selector][D] |
| Tesla Gen3 HSC / D: load-sharing state | Family 6, tags 11 → 12; ACK body, no live qualification | No demonstrated mapping to limits or status fields | U | G + allocation/effective-current distinction and body evidence | Read → native retention | Supported native acquisition contract; do not infer allocated or actual current | Proven lifecycle; Unknown scalar/target mapping | Denied | [R: Operations in scope; What this contract does not claim][R], [D: operation selector][D] |

For all five operations, only acknowledged, fresh, current records can describe
present state. `rejected`, `timeout`, `indeterminate`, and `no_send` carry no value.
Last attempt and last-known-good are separate per operation; failure never
replaces last-known-good. Old epochs may retain historical evidence while losing
current status. Recovery requires a successor generation and reactivation; a
decoder-rejected success retires the endpoint. R records an unresolved production
activation witness boundary, so implementation existence does not prove a usable
live acquisition composition. Acknowledging a read never establishes actuation.

## Semantic representability and retained current-limit coverage

All fields below are in `helianthus.pack.evse`, with definitions and metadata in
[`semreg/v1/packs/evse/definitions.go`][S]. Pack definition versions are existing
source metadata, not a Helianthus stability promise. **Q** denotes the existing
Tesla Gen3 injected-record publication owned by helianthus-ebusgateway,
[`mcp/tesla_gen3_evse_semantic.go`][Q], specifically `batch`, `provisional`,
`candidate`, and `teslaGen3EVSEPublicCurrentAt`. Q does not open an endpoint or
execute a current-limit operation. Native requirements are [C][C]; no new setter
or authentication/authorization check is proposed.

| Family / driver owner and source | Native evidence / qualification | SemReg field / service | Target device / cluster / feature / attribute | Mandatory gaps | Direction | Status / loss or withholding | Proven / Hypothesis / Unknown | Production admission | Exact public evidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Tesla Gen3 HSC / D, S | No typed current extraction demonstrated by supplied D; no hardware qualification | `evse.ac.current`, ampere / `evse.service.phase` | Electrical Sensor / Electrical Power Measurement / feature Unknown / optional ActiveCurrent, lock metadata above | G + phase identity, scale, null/freshness and feature reconciliation | Candidate passive read | Candidate; baseline's sole accepted mapping does not compose hardware | Proven definition and lock metadata; Hypothesis source extraction; Unknown full schema | Denied | [S: fields/services][S], [B: Production gate][B], [L: target][L], [D: TerminalBody][D] |
| Tesla Gen3 HSC / D, S | No typed power extraction demonstrated; no hardware qualification | `evse.ac.active_power`, watt / phase | U; ActivePower named only by repository readiness contract | G + phase, sign, scale, nullable value evidence | Candidate passive read | Unavailable source promotion; no derived current × voltage power | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [B: Production gate][B], [D: TerminalBody][D] |
| Tesla Gen3 HSC / D, S | No typed voltage extraction demonstrated; no hardware qualification | `evse.ac.voltage`, volt / phase | U/G | Phase/scale/conformance plus G | Candidate passive read | Unavailable source promotion | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields/services][S], [R: What this contract does not claim][R] |
| Tesla Gen3 HSC / D, S | No typed frequency extraction demonstrated; no hardware qualification | `evse.ac.frequency`, hertz / EVSE | U/G | Scope/scale/conformance plus G | Candidate passive read | Unavailable source promotion | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields/services][S], [R: What this contract does not claim][R] |
| Tesla Gen3 HSC / D, S | Lifetime body native only; no session derivation or hardware qualification | `evse.energy.lifetime`, kilowatt-hour / meter | U/G | Counter scope/reset/scale and target energy model plus G | Candidate passive read | Withheld; native body is not typed energy | Proven definition; Hypothesis lifetime correspondence; Unknown target | Denied | [S: fields/services][S], [R: Operations in scope][R] |
| Tesla Gen3 HSC / D, S | No session evidence demonstrated; no hardware qualification | `evse.energy.session`, kilowatt-hour / session | U/G | Session identity/boundary, counter semantics plus G | Candidate passive read | Unavailable; do not subtract lifetime counters to invent session state | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields/services][S], [R: What this contract does not claim][R] |
| Tesla Gen3 HSC / Q | Persistent returned settings retained; native version scoped, no live setter qualification | `evse.limit.configured_current`, ampere / EVSE | U/G; never ActiveCurrent | Configured-limit semantics, target conformance plus G | Retained read → passive candidate | Supported semantic projection; target withheld; settings are not measured current | Proven Q projection; Hypothesis target correspondence; Unknown target | Denied | [C: Outcomes and confirmation limits][C], [Q: batch; persistent record][Q], [S: fields][S] |
| Tesla Gen3 HSC / Q | Complete retained provisional set/ACK/readback payloads, exact version, uninhibited and unexpired; no causal/physical proof | `evse.limit.allocated_current`, ampere / connector | U/G; never ActiveCurrent | Host timeout, connector scope, target allocation semantics plus G | Retained read → passive candidate | Supported conditional semantic projection; retain native evidence and comparison limits | Proven Q projection; Hypothesis target correspondence; Unknown target | Denied | [Q: batch; provisional; public current][Q], [C: Outcomes and confirmation limits][C] |
| Tesla Gen3 HSC / Q | Missing, malformed/correlation-mismatched, inhibited, zero/out-of-range timeout, or expired provisional input | Same allocated-current field; no available value | U/G | Same gaps as allocated row | Retained read → withholding | Withheld with `withheld_provisional_` reason; Q can withdraw prior candidate; no zero fill | Proven withholding policy; Unknown target | Denied | [Q: provisional; batch; public current][Q] |
| Tesla Gen3 HSC / S, C | No demonstrated pilot/effective-current fact; returned provisional triple is insufficient | `evse.limit.actual_current`, ampere / connector | U/G | Independent effective-current evidence plus G | Candidate passive read | Unavailable; do not equate allocation, configured or measured current | Proven definition and native distinction; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [C: Outcomes and confirmation limits][C] |
| Tesla Gen3 HSC / S, C | No demonstrated advertised/pilot-current fact or hardware qualification | `evse.limit.advertised_current`, ampere / connector | U/G | Independent advertised-current semantics plus G | Candidate passive read | Unavailable; ACK/readback cannot supply pilot evidence | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [C: Outcomes and confirmation limits][C] |
| Tesla Gen3 HSC / D, S | Native bodies only; no charging-state interpretation demonstrated | `evse.status.charging` / connector | U/G | Exact charging/complete/idle interpretation plus G | Candidate passive read | Unavailable; nonzero current or ACK cannot define charging state | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [R: What this contract does not claim][R] |
| Tesla Gen3 HSC / D, S | No connection-state interpretation demonstrated | `evse.status.connection` / connector | U/G | Connected/disconnected evidence plus G | Candidate passive read | Unavailable; no default disconnected value | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [R: What this contract does not claim][R] |
| Tesla Gen3 HSC / D, S | No fault-state interpretation demonstrated | `evse.status.fault` / EVSE | U/G | Present/clear evidence plus G | Candidate passive read | Unavailable; transport health is not device fault-clear | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [C: Admission preconditions][C] |
| Tesla Gen3 HSC / S, C | Inhibit flag is not a qualified interlock-state map | `evse.status.interlock` / connector | U/G | Active/clear native interlock map plus G | Candidate passive read | Unavailable; do not reinterpret inhibit as an interlock | Proven definition and gap; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [C: Admission preconditions][C] |
| Tesla Gen3 HSC / D, S | Lifecycle ready is not EVSE readiness evidence | `evse.status.readiness` / EVSE | U/G | Device ready/not_ready interpretation plus G | Candidate passive read | Unavailable; no lifecycle-to-device-state substitution | Proven definition; Hypothesis future correspondence; Unknown target | Denied | [S: fields][S], [R: Lifecycle states and transitions][R] |
| Tesla Gen3 HSC / S, Q, C | Pack operation exists; Q is read-only and exposes no execution method | `evse.operation.set_allocated_current`, capability and effect of same suffix / connector | U; possible EVSE command route entirely unverified | Locked command schema, effect semantics, source executor/qualification; G if sensor route retained | Proposed command candidate only | Unavailable binding execution; no command support from read capability | Proven definition/read boundary; Hypothesis future command route; Unknown target | Denied | [S: capabilities/operations][S], [Q: provider interface][Q], [C: Read-only and state-changing boundary][C] |
| Tesla Gen3 HSC / C, Q | t28's separate configured-current value is native returned state, not effective/pilot current; no live qualification | Q's configured fact comes from persistent settings; separate t28 configured value is not demonstrated as a fact in Q | U/G | Separate provenance/comparison and target semantics plus G | Retained read | Withheld separate projection; do not silently replace persistent configured evidence | Proven native distinction and Q field source; Unknown additional projection/target | Denied | [C: Registry-owned request and response framing][C], [Q: batch][Q] |
| Tesla Gen3 HSC / C, Q | Provisional timeout and inhibit are retained native members, not observed physical expiry or safety state | No dedicated timeout/inhibit field in S; Q uses them to withhold allocated current | U/G | Independent device effect/interlock evidence and target model plus G | Retained read | Supported native retention/policy use; withheld scalar/status substitution | Proven retention/policy; Unknown physical effect/target | Denied | [C: Outcomes and confirmation limits][C], [Q: provisional record; provisional][Q], [S: fields][S] |
| Tesla Gen3 HSC / C, Q | ACK, READBACK, TIMEOUT, INDETERMINATE, REJECTED; requested/returned, numeric statuses and phase/certainty evidence stay separate; no live qualification | No generic outcome/comparison field in S; Q is a selected limit-fact view, not the full operation record | U/G | Outcome/effect representation, evidence references and loss disclosure plus G | Historical read | Withheld as target values; preserve ACK after failed readback and native unknown members; never Boolean success | Proven C contract; Unknown complete executor coverage/target | Denied | [C: Outcomes and confirmation limits; Immutable correlated evidence][C], [Q: publication evidence; batch][Q], [S: fields][S] |

## Outcome preservation, falsifiers and synthetic review cases

These are manual contract review cases, not executed tests or proposed API
changes. C's requested values, returned values, comparison, transmission
certainty and effect certainty stay distinct. Persistent t7 → t8 is returned
settings, not ACK. Provisional t25 → t26 ACK alone has no applied value; t27 →
t28 reports returned state without causal application proof. None is one of D's
five acquisitions. Timeout/indeterminate retain available bytes and earlier ACK;
late or cross-generation evidence cannot complete the original operation. No
automatic mutation retry or rollback follows.

Semantic scalar views omit native bytes, unknown members and phase/comparison
detail. A future Matter projection must disclose that loss and retain evidence
references; Q's `ProjectionExact` for a selected semantic scalar is not a claim
that it reproduces the full native operation record. Reviewing the cited contract
does not verify every executor implementation behind that record.

| Candidate / case | Required interpretation and concrete falsifier | Current basis |
| --- | --- | --- |
| Current → ActiveCurrent | Proposed only after native phase/unit/scale and locked feature/null rules agree. Falsified by mapping a configured limit or untyped vitals body as measured current, or by a locked feature rule that the source cannot satisfy. | S fields; D TerminalBody; L target; B Production gate |
| Active power, voltage, frequency candidates | Independently establish signed power in watts, phase voltage in volts, EVSE frequency in hertz. Falsified respectively by unsupported sign/derived power, mismatched phase identity/scale, or incompatible frequency scope/unit; any incompatible locked attribute rule also falsifies. | S fields/services; R native-only boundary |
| Lifetime/session energy candidates | Need meter reset/scale and session identity/boundaries respectively. Falsified by counter reset treated as consumption, or by lifetime subtraction without a session contract; incompatible locked energy conformance also falsifies. | S fields/services; R native-only boundary |
| Configured/allocated limit candidates | Preserve EVSE versus connector scope and timeout. Falsified by either becoming ActiveCurrent, by conflating persistent settings with provisional allocation, or by target command/attribute semantics that cannot preserve their distinctions. | Q batch/provisional; C outcomes |
| Actual/advertised limit candidates | Need independent effective/pilot evidence. Falsified by using t28 configured/provisional values for either, or by incompatible locked semantics. | C outcomes; S fields |
| Charging/connection/fault/interlock/readiness candidates | Require explicit native symbol maps for each. Falsified respectively by ACK → charging, missing value → disconnected, transport success → clear, inhibit → interlock, or acquisition ready → device ready; incompatible locked symbol domains also falsify. | S fields; R states; C preconditions |
| Future allocated-current command | Pack definition does not authorize or implement I/O. Falsified by treating a read-only provider or ACK as execution/effect confirmation, or by locked command effects incompatible with retained outcomes. | S operations; Q provider; C outcomes |
| Synthetic ACK followed by failed readback | Retain ACK, report timeout or indeterminate with phase details; allocated current cannot be fabricated. Error terminal after ACK does not prove mutation rejection. | C Outcomes and confirmation limits |
| Synthetic inhibited/expired provisional record alongside retained persistent settings | Withhold allocated fact with reason while preserving separately evidenced configured fact and native evidence; never reset both to zero. | Q batch/provisional/public current |
| Synthetic failed acquisition after earlier ACK, then successor epoch | Preserve per-operation last-known-good until retention expires, but never call an earlier epoch current; non-ACK body is not a value. | D retainLocked/Retained/TerminalBody; R Freshness and retention |
| Synthetic SDK sample reads ActiveCurrent successfully | Establishes only sample behavior; copying sample Accuracy/PowerMode or admitting a production endpoint falsifies this inventory's boundary. | B Hosted reproduction/Production gate; P evaluate |

Q's existing deterministic evaluated tuple (`Snapshot`, `Evaluation`,
`Selections`, `Projection`) and native evidence references remain the passive
consumer boundary. This document changes neither MCP compatibility nor GraphQL
behavior; GraphQL implementation/parity has not been independently inspected in
the supplied views. No new envelope, transition, permission, principal, credential,
or authentication gate is introduced. Application lifecycle stays outside
protocol/type CC0 trees.

## Matter 1.6.1 target reconciliation

The adopted SDK is [go-fabric](https://github.com/SukramJ/go-fabric) on
**Matter 1.6.1**, as recorded in the [baseline](../SDK_BASELINE.md).
The checked-in lock, validator, mapping reference and hosted job remain legacy
Matter 1.7 artifacts pending owner-controlled replacement; they have not been
migrated. Any lock metadata below describes that retained legacy state only.
The [deferred ledger](../adoption/matter-161-deferred.md) separates Sensor
Electrical Alarm and Power Topology ElectricalCircuit deferrals from the
absent DEM PowerRangeAdjustment upstream snapshot gap.

This table applies to every existing source row and target search on this page.
**Changed** and **Unchanged** describe only the stated identity comparison;
**Unknown** retains uninspected differences. Reported source presence is Proven
only within the supplied graded inspection result, not independent replay here,
full normative-corpus inspection, native qualification or an accepted join.
Source qualification, direction, loss/withholding and incomplete mandatory
closure remain independent and unchanged. No row is deleted or promoted. In retained row text, locked target rules
mean unresolved requirements for the future reviewed adopted baseline, never
authority to publish against the legacy 1.7 model.

| Existing target references and row coverage | Adopted disposition and remaining evidence |
| --- | --- |
| Electrical Sensor / Power Topology / EPM / ActiveCurrent | Changed Sensor/Topology to revision 1; unchanged EPM revision 3. The sole accepted evse.ac.current mapping is unchanged; optional ActiveCurrent cannot discharge composition gaps. |
| EnergyEVSE / EVSE; charging, connection, fault, readiness, limits and command candidates | Unknown prior identity → Energy EVSE revision 4 source presence. Exact device/features/fields and native effect/readback remain Unknown; no extra mapping accepted. |
| ElectricalEnergyMeasurement, PowerTopology, Descriptor and electrical quantities | Unknown prior EEM identity → revision 2 source presence; Descriptor rules, session/lifetime semantics, phase identity, precision and loss remain unresolved. |

For each candidate, obtain the retained Matter 1.6.1 normative device/cluster
corpus and inspect the adopted SDK definitions, referenced features, attributes,
commands, events, access, units, ranges, nullability and transitive mandatory
composition. Public [upstream source](https://github.com/SukramJ/go-fabric/tree/main)
locates the implementation; it does not supply the missing retained model corpus.
Reconcile generated behavior and accepted public semantic target evidence with
each native fact, direction, loss and falsifier. Do not substitute legacy
connectedhomeip samples or the sole EVSE current mapping for another family.
Actual lock migration and matching build/reporting evidence remain owner work.

Descriptor and Power Topology composition, PowerMode, NumberOfMeasurementTypes,
non-empty matching Accuracy and evidenced nullable ActivePower remain unresolved
for the electrical route. No full normative corpus, Linux arm64 build, accepted
production profile, conformance, certification or interoperability is established.
Production admission remains denied. Native evidence and partial last-known-good
state, eBUS MCP compatibility and GraphQL parity remain unchanged.

[L]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/sdk/matter-sdk.lock.json
[B]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/SDK_BASELINE.md
[P]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/runtime/production_readiness.py
[S]: https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/evse/definitions.go
[D]: https://github.com/Project-Helianthus/helianthus-modbusreg/blob/main/tesla_gen3_read_acquisition.go
[R]: https://github.com/Project-Helianthus/helianthus-docs-modbus/blob/main/protocols/tesla/gen3-read-acquisition-v1.md
[C]: https://github.com/Project-Helianthus/helianthus-docs-modbus/blob/main/protocols/tesla/gen3-current-limit-operation-v1.md
[Q]: https://github.com/Project-Helianthus/helianthus-ebusgateway/blob/main/mcp/tesla_gen3_evse_semantic.go
