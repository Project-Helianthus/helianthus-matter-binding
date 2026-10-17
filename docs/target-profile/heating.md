# Heating source coverage and composition gaps

This operator-directed inventory is a **partial source-side contract slice** for
[eBUS heating coverage](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/11).
It advances [the Matter binding parent](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/8)
without closing it or releasing M42/M43. It inventories system, zone, circuit
and domestic hot water (DHW) facts at the binding boundary. It creates no native
facts, semantic definitions, target mappings, runtime endpoint or admission path.
Nothing here is stable before Helianthus 1.0.

Native interpretation belongs to the public protocol owner [N] and [B]; current
polling/publication belongs to the gateway driver [G]; protocol-neutral fields
belong to SemReg [S]. Accepted Matter target mappings belong to the public
semantic documentation repository named by the lock [L]. The prospective
correspondences below are reconciliation questions, not additions to that
mapping. No gateway, MCP or GraphQL interface changes are proposed. Existing
native evidence, deterministic envelopes, MCP compatibility, GraphQL parity and
partial last-known-good behavior remain requirements for any future consumer.

## Evidence axes and notation

Each row separates four decisions: native evidence, semantic representability,
locked target-model evidence, and production admission. **Proven** means the
cited source establishes the stated code or contract behavior; it never means
an installation is qualified. **Hypothesis** means a proposed interpretation or
correspondence with a stated falsifier. **Unknown** means evidence is absent or
conflicting. Family-level scope is Vaillant eBUS controller, remote-room,
functional-module and direct-boiler families; none is a universal product claim.

All SemReg names below are in `helianthus.pack.thermal`, whose current metadata
version is `1.0.0` [S]; that metadata version is not a Helianthus stability claim.
`system`, `zone`, `circuit` and `dhw` service shorthand means respectively
`thermal.service.system`, `.zone`, `.circuit`, `.dhw`; their identity dimensions
are `thermal.dimension.system`, `.zone`, `.circuit`, `.dhw`. A service identity is
separate from a measurement FactKey and from physical topology.

**C** = source candidate; **W** = withheld; **U** = unavailable from the cited
current path. There are no currently supported heating Matter endpoints in
this inventory. Semantic representability is narrower than supported hardware.
**Q?** in every row means installation qualification Unknown: [Q] has
`evidence_status: Unknown` and `qualified_use_permitted: false`; [F] also has
`runtime_qualification_permitted: false`. Other driver paths have no supplied
accepted thermal binding proof. Successful reads or legacy publication do not
change Q?.

**T? / denied**, repeated in every row, means target device type, cluster,
feature and attribute are all **Unknown**, and production admission is denied.
It is not a negative claim about support in the uninspected Matter schema.
Every row also requires **G0**: locked-corpus reconciliation, evidence-backed
mandatory target composition, independent build evidence, and a future reviewed
accepted profile. None is supplied by this document.

## Locked target metadata, not a heating schema inspection

The lock [L] names data-model version `1.7`. Its explicit target metadata is
Electrical Sensor `0x0510` revision 2, Power Topology `0x009C` revision 2,
Electrical Power Measurement `0x0090` revision 3, and optional `ActiveCurrent`
`0x0005`. These identifiers and revisions are **lock metadata only**; no heating
row below maps to them. The complete locked connectedhomeip data model is not
available for this review. These entries neither prove nor disprove heating
support elsewhere in that corpus.

The baseline [D] describes the existing electrical composition gaps: Descriptor,
Power Topology, non-unknown PowerMode, positive NumberOfMeasurementTypes,
non-empty fixed Accuracy, and evidenced nullable ActivePower. Those are the
existing electrical readiness boundary, not an inferred list of mandatory
heating clusters or attributes. Heating mandatory composition remains Unknown.
The lock's `accepted_runtime_composition_profile_sha256` is null, and
`evaluate` in [R] denies every call before caller profile/build fields can
permit allocation. Q? and T? cannot bypass that rule.

## Source-side matrix

In the owner column, **controller/G**, **remote/G**, **module/G** and **boiler/G**
mean the indicated family in gateway `cmd/gateway/semantic_vaillant.go` [G].
Function names and native selectors identify exact evidence within the linked
files. Except where explicitly marked otherwise, local controller rows concern
B524 local reads; selector numbers below are native register identifiers, never
Matter target identifiers. Qualifiers, codecs and observation limits remain
those of [N], [B], [M] and [F]; no wire contract is widened here.

### System and installation identity

| Fact group / disposition | Family, driver owner and source path | Native evidence / qualification | SemReg field or service; current vs proposed | Target device / cluster / feature / attribute; admission | Mandatory composition gaps beyond G0 | Direction | Loss / withholding; candidate falsifier | Proven / Hypothesis / Unknown; exact evidence locator |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| System service identity — C | controller/G, `refreshSystem` | Current controller snapshot identity; Q? | `system` exists in [S]; qualified asset/service binding still missing | T? / denied | Bind asset, source epoch, driver generation and service revision independently of display IDs | Read metadata | Snapshot identity cannot prove topology; falsifier: controller replacement leaves old service qualification attached | Proven source; Hypothesis binding; [G] `refreshSystem`, [M] “SemReg thermal v1 read-only Vaillant mapping”, [F] applicability |
| Physical installation / circuit ownership — W | controller/module/G, `publishCircuits`, FM5 interpretation | System scheme and module configuration are read; ownership is topology-scoped; Q? | Service dimensions exist; no physical topology fact supplied by [S] | T? / denied | Physical routing, module-role evidence, coherent structural mode/reason/revision, endpoint parts and mandatory facts Unknown | Read metadata | Counts, zone indexes and circuit types do not identify physical equipment; unproven owners remain Unknown | Proven scoped publication; Unknown general topology; [M] circuits ownership note, [G] `publishCircuits`, `deriveFM5Interpretation` |
| Outdoor instantaneous temperature — C | controller/G, `refreshSystem` | [F] exact read-only candidate, local system register `0x0073`; Q? | Current [F]: `thermal.measurement.air_temperature`, exact FactKey `thermal.dimension.temperature=temperature:outdoor_air`; separate `system` binding | T? / denied | Qualified initiator/target context, source receipt/freshness, units and target scaling/nullability | Read | No cross-family float-codec generalization; falsifier: reply context or quantity differs from [F] | Proven contract; Hypothesis qualified use; [N] system row `0x0073`, [F] `mappings[0]` |
| System flow temperature — C | controller/G, `refreshSystem` | [F] exact read-only candidate, local system register `0x004B`; Q? | Current [F]: `thermal.measurement.water_temperature`, exact FactKey `thermal.dimension.temperature=temperature:system_flow`; separate `system` binding | T? / denied | Same contextual/freshness evidence as outdoor row; physical sensor role | Read | Not boiler flow or boiler return; falsifier: selector substituted with B509 or remote heat-source status | Proven contract; Hypothesis qualified use; [N] system row `0x004B`, [F] `mappings[1]` |
| Outdoor rolling average — W | controller/G, `refreshSystem` | Local system `0x0095`, rounded average updated every three hours; Q? | Current [F] has no SemReg fact; no window/aggregate mapping | T? / denied | Aggregate window and measurement-time semantics before target reconciliation | Read | `rolling_average_window_unrepresentable`; cannot become instantaneous air temperature | Proven withholding; [N] system row `0x0095`, [F] `mappings[2]` |
| Cylinder top / bottom temperatures — C | controller/G, `refreshSystem` | Separate top/bottom controller observations; Q? | Proposed `thermal.measurement.water_temperature` associated with `system` or `dhw` only after sensor/service qualification | T? / denied | Distinct sensor role FactKeys, cylinder identity, presence and service attachment | Read | Do not merge top/bottom or equate either with DHW outlet; falsifier: swapped sensor roles produce same fact identity | Proven reads; Hypothesis semantics; [M] system `state.hwc_cylinder_temperature_top/bottom`, [G] `refreshSystem`, [S] fields |
| System off / maintenance / energy-manager state — W, U for manager state | controller/G, `refreshSystem` | Off and maintenance booleans read; manager state listed in [M] but not acquired by this function; Q? | `thermal.mode.system`, `thermal.status.operation`, `thermal.status.fault` exist; no accepted binding for these source states | T? / denied | Full mode/action/fault codebook, unknown states, availability, aggregation and source acquisition for manager state | Read | Off alone does not establish all modes; maintenance is not fault; absence is not fault.clear | Proven read/listing distinction; Unknown semantic equivalence; [M] system table, [G] `refreshSystem`, [S] symbol sets |
| System pressure / heating configuration / bounds — W | controller/G, `refreshSystem`, `SetSystemConfig` | Pressure, adaptive curve, alternative/bivalence points, emergency temperature, max desired flow and max humidity read; derived DHW bound is profile context, not register; Q? | [S] has no exact pressure, curve, bound or installer-configuration field; thresholds are not generic room setpoints | T? / denied | Source-specific quantity roles and accepted semantic definitions; profile discovery/qualification for bound | Read inventory; existing native config writes separate | No thermostat command from configuration reads or policy bounds; do not copy personal installer fields or access codes | Proven code/config contract; Unknown target equivalence; [M] system table and bound discussion, [G] `refreshSystem`, `SetSystemConfig`, [S] fields |

### Zones and remote-room observations

| Fact group / disposition | Family, driver owner and source path | Native evidence / qualification | SemReg field or service; current vs proposed | Target device / cluster / feature / attribute; admission | Mandatory composition gaps beyond G0 | Direction | Loss / withholding; candidate falsifier | Proven / Hypothesis / Unknown; exact evidence locator |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Zone presence / service identity / sensor assignment — C | controller/remote/G, discovery, `publishZones` | Zone assignment, room sensor mapping and remote assignment are distinct observations; Q? | `zone` and read capability exist; proposed binding needs explicit identity, not public zone name/index alone | T? / denied | Qualified presence, sparse-instance handling, sensor relationship, epoch and independent physical topology | Read metadata | No universal zone-to-circuit or zone-to-remote identity; falsifier: two service instances collapse to one key or sensor replacement retains old proof | Proven source structure; Hypothesis binding; [N] zones `0x0013`, `0x001C`, [G] `refreshZoneDiscovery`, `mappedRadioRoomStateForZoneLocked`, [S] services |
| Zone room temperature — C | controller/G, `refreshState` | Local zones `0x000F` read as room temperature; Q? | Proposed `thermal.measurement.air_temperature` on `zone`; no accepted row in [F] | T? / denied | Room-specific FactKey, source sensor, units, freshness, target precision and unavailable handling | Read | Preserve distinction from outdoor and water; falsifier: room measurement published under outdoor FactKey | Proven source read; Hypothesis mapping; [N] zones `0x000F`, [M] zones table, [G] `refreshState`, [S] fields |
| Zone and remote humidity / remote room temperature — C | controller/remote/G, slow config, radio refresh, `mappedRadioRoomStateForZoneLocked` | Current zone humidity read and remote fallback paths exist; connection and mapping required; Q? | Proposed `thermal.measurement.humidity_relative` and `thermal.measurement.air_temperature` on `zone` with separate source identity | T? / denied | Qualified remote assignment, source preference/provenance, valid range and stale/disconnect behavior | Read | A connected remote is not automatically the zone sensor; falsifier: unassigned remote overwrites a zone observation | Proven code path; Hypothesis binding; [N] zones `0x0028`, [G] `refreshZoneSlowConfigFields`, `refreshRadioDevices`, `mappedRadioRoomStateForZoneLocked`, [S] fields |
| Manual / effective / override zone setpoints — W | controller/G, slow config and publication | Driver uses primary `0x0022` and fallback `0x0014`; native catalog distinguishes manual from computed effective; holiday/quick-veto facts separate; Q? | `thermal.setpoint.temperature` exists; no accepted selector/role binding in [F] | T? / denied | Preserve manual/effective/holiday/override roles, units, schedule/override precedence, limits and same-service identity | Read observations | Computed effective setpoint is not user-settable; quick-veto unit is unestablished in candidate write contract; do not normalize all roles into one writable target | Proven driver behavior and catalog distinction; Unknown lossless equivalence; [N] zones `0x0005`, `0x0008`, `0x0014`, `0x0022`, `0x0026`, [M] zones table, [G] `refreshZoneSlowConfigFields` |
| Zone operating mode / preset — W | controller/G, `deriveZoneModeAndPreset` | Code derives off/auto/heat/cool plus schedule/manual/quickveto/away; public prose and code use differing derivation inputs; Q? | `thermal.mode.zone` allows auto/comfort/eco/off; no heat/cool/preset equivalence accepted | T? / denied | Family codebook, raw mode + special-function preservation, correct associated circuit and all unsupported symbols | Read derived | Manual is not comfort; away is not automatically eco; code defaults must not become new qualified facts | Proven derivation; Hypothesis native labels; Unknown semantic mapping; [N] zones and family enumerations, [M] Zone Mode Derivation, [G] `deriveZoneModeAndPreset`, `deriveManualOperatingMode`, [S] symbols |
| Zone action / valve status — C | controller/G, `refreshState`, `deriveZoneHvacAction` | Valve status drives idle/heating/cooling projection with circuit type; Q? | Proposed `thermal.action.state` on `zone`; raw valve fact retained separately | T? / denied | Qualified valve meaning, actual action versus demand, associated circuit, Unknown raw states | Read derived | Valve open does not alone prove heat delivery; falsifier: open valve with no heating is published as qualified heating action | Proven code; Hypothesis action correspondence; [N] zones `0x0012`, [G] `deriveZoneHvacAction`, [S] action symbols |
| Zone schedule / holiday / quick-veto lifecycle — W | controller/G, slow config and B555 schedule functions | Configuration dates, times, duration and schedules exposed; Q? | No schedule/window/preset lifecycle field in [S]; scalar temperature alone loses context | T? / denied | Calendar semantics, slot identity, override transitions, retained native source and separate effect evidence | Read inventory; native writes remain separate | No Matter schedule or override command inferred from a displayed preset or acknowledged native write | Proven functions; Unknown target correspondence; [G] `refreshZoneSlowConfigFields`, `refreshSchedules`, `SetZoneTimeProgram`, [N] zones and B524 calendar selectors |

### Circuits

| Fact group / disposition | Family, driver owner and source path | Native evidence / qualification | SemReg field or service; current vs proposed | Target device / cluster / feature / attribute; admission | Mandatory composition gaps beyond G0 | Direction | Loss / withholding; candidate falsifier | Proven / Hypothesis / Unknown; exact evidence locator |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Circuit service / type / cooling capability — C | controller/module/G, `refreshCircuits` | Type probe and cooling flag read; virtual native-water slot is not physical heating circuit; Q? | `circuit` / read capability exist; proposed identity binding only | T? / denied | Profile-specific presence, raw type vs capability, service identity and independently qualified ownership | Read metadata | Driver pseudo-circuit temperature heuristic is not native presence proof; falsifier: a raw temperature/status creates a physical circuit or type implies cooling without its flag | Proven driver behavior; Unknown generalized topology; [N] Circuits introduction and `0x0002`, `0x0006`, [G] `readDhwPseudoCircuitEvidence`, `refreshCircuits`, [S] services |
| Circuit flow temperature — C | controller/G, `refreshCircuits` | Local circuits `0x0008` is circuit flow sensor observation; Q? | Proposed `thermal.measurement.water_temperature` on `circuit` | T? / denied | Per-circuit sensor FactKey, physical sensor qualification, units and freshness | Read | Never boiler return or system flow; falsifier: two circuit instances overwrite one FactKey or sensor role aliases boiler return | Proven read/catalog role; Hypothesis mapping; [N] circuits `0x0008`, [M] circuits table, [G] `refreshCircuits`, [S] fields |
| Circuit flow setpoint — C | controller/G, `refreshCircuits` | Local circuits `0x0007` read-only state candidate; Q? | Proposed `thermal.setpoint.temperature` only with explicit circuit-flow role; not zone operation | T? / denied | Qualified computed/requested role and per-circuit identity; target setpoint semantics Unknown | Read observation | No command route; falsifier: observed flow setpoint is treated as writable room target | Proven read; Hypothesis role/mapping; [N] circuits `0x0007`, [G] `refreshCircuits`, [S] setpoint and zone-operation service |
| Circuit calculated-flow label — W | controller/G, `refreshCircuits` | Driver/prose read `0x0020` as f32 calculated flow; native catalog records a one-byte raw status; Q? | No defensible water-temperature mapping until conflict resolved | T? / denied | Correct width/codec, family context and semantic corroboration | Read raw evidence | Withhold temperature projection; raw status is not degrees Celsius | Proven source conflict; Unknown interpretation; [N] circuits `0x0020`, [M] `state.calculated_flow_temperature`, [G] `refreshCircuits` |
| Circuit state — C | controller/G, `decodeCircuitStateToken` | `0x001E` state enumeration is naming-based Hypothesis, not correlated; unknown raw values retained as unknown tokens; Q? | Proposed `thermal.action.state` with qualified idle/heating/cooling/DHW correspondence | T? / denied | Native state correlation, circuit service, unmapped values, distinction from boiler state | Read derived | Falsifier: correlated circuit state disagrees with proposed enum or unsupported value becomes idle | Proven code; Hypothesis native and semantic meaning; [N] Circuit State Enum, [M] selector assignment note, [G] `decodeCircuitStateToken`, [S] action symbols |
| Circuit pump status — W | controller/G, `refreshCircuits` | `0x001B` raw pump observation; nonzero-active is Hypothesis; Q? | No exact pump-status field in [S]; do not infer `thermal.status.operation` | T? / denied | Correlated circuit pump evidence and accepted semantic role | Read raw/derived | Separate from `0x001E`; cannot qualify boiler pump or system operation | Proven raw-source distinction; Hypothesis Boolean meaning; [N] circuits `0x001B`, [M] selector assignment, [G] `mergeBoilerMirrorFieldsFromSnapshots` |
| Circuit humidity / dew point — C | controller/G, `refreshCircuits` | Separate humidity `0x0022` and dew-point `0x0023` reads; applicability depends on native configuration; Q? | Proposed `thermal.measurement.humidity_relative`; air-temperature field only if a distinct dew-point role is accepted | T? / denied | Sensor role, cooling/room-control applicability, separate derived-temperature identity | Read | Dew point is not ambient temperature; falsifier: dew point replaces room temperature or disabled applicability yields qualified data | Proven reads; Hypothesis mapping; [N] Gate Conditions and circuit rows, [G] `refreshCircuits`, [S] fields |
| Circuit curve / limits / influence / frost configuration — W | controller/G, `refreshCircuits` | Curve, min/max flow limits, summer limit, room influence, frost threshold read; Q? | No exact [S] configuration schema; scalar setpoint is insufficient for limits and curves | T? / denied | Role-specific fields, units, technical configuration scope; frost threshold is config rather than property | Read inventory | No installer/config mutation through generic thermostat writes; no room setpoint alias | Proven reads; Hypothesis menu correspondence where noted; [N] circuits `0x000F`, `0x0010`, `0x0012`, `0x0014`, `0x0015`, `0x001D`, [M] stale frost path note, [G] `refreshCircuits` |
| Circuit mixer / pump counters — W | controller/G, `refreshCircuits` | Mixer position, pump hours and starts acquired; Q? | No exact mixer-position/counter fields in [S]; percent is not automatically `thermal.demand.level` | T? / denied | Counter units/reset semantics and mechanical position versus thermal demand | Read | Preserve native fields; do not substitute pump activity or position for heat demand | Proven reads; Unknown target/semantic correspondence; [N] circuits `0x0021`, `0x0024`, `0x0025`, [G] `refreshCircuits`, [S] fields |

### DHW, direct boiler boundaries and commands

| Fact group / disposition | Family, driver owner and source path | Native evidence / qualification | SemReg field or service; current vs proposed | Target device / cluster / feature / attribute; admission | Mandatory composition gaps beyond G0 | Direction | Loss / withholding; candidate falsifier | Proven / Hypothesis / Unknown; exact evidence locator |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| DHW service presence / identity — C | controller/G, `refreshDHW` | Native DHW has its own presence predicate and service; Q? | `dhw` / read capability exist; proposed identity binding | T? / denied | Qualified presence result, distinct service identity, separation from virtual circuit and cylinders | Read metadata | Schema object or mirror is not presence proof; falsifier: wrong-width/failed predicate becomes absent or an independent physical cylinder | Proven native predicate contract; Hypothesis binding; [N] Native Domestic Hot Water introduction, [G] `refreshDHW`, [S] services |
| DHW current temperature — C | controller/G, `refreshDHW` | Local DHW `0x0005`, read-only; Q? | Proposed `thermal.measurement.water_temperature` on `dhw`, role-qualified | T? / denied | DHW presence, sensor role, source freshness, nullable/invalid rules | Read | No alias with system cylinder top/bottom; falsifier: DHW mirror creates a second independent sensor | Proven read; Hypothesis mapping; [N] DHW `0x0005`, [M] DHW table, [G] `refreshDHW`, [S] fields |
| DHW target temperature — C | controller/G, `refreshDHWSlowConfig` | Local DHW `0x0004` read observation; candidate native write reaction unconfirmed; Q? | Proposed `thermal.setpoint.dhw_temperature` on `dhw`; no DHW write operation in [S] | T? / denied | Qualified target role, source-specific bounds, service identity and separate effect/command evidence | Read observation | Falsifier: observed target is accepted as a writable thermostat or zone operation | Proven read; Hypothesis mapping; [N] DHW `0x0004`, [G] `refreshDHWSlowConfig`, [S] operations |
| DHW mode / special function / holiday schedule — W | controller/G, `deriveDhwModeAndPreset`, schedule functions | Operating mode and special-function observations have distinct widths/roles; code derives presets; Q? | No DHW mode, special-function or calendar definition in [S]; system mode cannot be reused by label | T? / denied | Native enum/width qualification, one-time load versus configured mode, schedule precedence, target mandatory behaviors Unknown | Read inventory; native schedule writes separate | Do not turn special-function mode into actual loading state, operation authority or target command | Proven code and catalog distinction; Hypothesis native candidate semantics; [N] DHW `0x0003`, `0x000D`, `0x000F`, [G] `deriveDhwModeAndPreset`, `SetDhwTimeProgram`, [S] services/operations |
| Functional-module cylinder temperature — C | module/G, `readCylinderSnapshots`, `publishFM5Semantic` | Live decodable temperature required per instance; config-only replies cannot create cylinders; Q? | Proposed `thermal.measurement.water_temperature` on independently identified `dhw` context | T? / denied | Coherent FM5 interpretation, actual cylinder/service relationship, per-instance sensor role | Read | Preserve structural withdrawal versus transient retention; falsifier: config-only data creates a cylinder or stale data becomes a fresh sample | Proven publication rule; Hypothesis binding; [M] cylinders section, [G] `hasLiveCylinderEvidence`, `publishFM5Semantic`, [S] fields |
| Cylinder max setpoint / hysteresis / charge offset — W | module/G, `readCylinderSnapshots` | Separate configuration quantities retained only with coherent cylinder evidence; Q? | No exact hysteresis/offset/limit binding in [S]; not interchangeable with DHW target | T? / denied | Configuration roles, absolute versus delta units, cylinder presence | Read inventory | No inferred heater capability, flow temperature or writable DHW target | Proven reads/publication; Unknown semantic equivalence; [M] cylinders table, [G] `readCylinderSnapshots`, [S] fields |
| Direct-boiler flow / desired temperatures — C | boiler/G, `refreshBoilerStatusB509` | B509 flow, desired flow/DHW and configured max temperatures are separate candidate register meanings; Q? | Proposed `thermal.measurement.water_temperature` for flow; setpoint fields only after role qualification | T? / denied | Direct-boiler identity, correct selector/codec, distinct measurement/desired/limit roles | Read | Not controller system-flow or circuit-return evidence; falsifier: shared display name merges distinct selectors or limits become observed temperature | Proven driver reads; Hypothesis native/semantic correspondence; [B] Direct and Additional read candidates, [G] `refreshBoilerStatusB509`, [S] fields |
| Boiler state / flame / demand / pump and diagnostics — W | boiler/G, `refreshBoilerStatusB509` | Candidate direct state number, flame/gas valve/pumps/DHW demand; separate counters and diagnostics; Q? | `thermal.status.operation`, `thermal.action.state`, `thermal.status.fault` exist; no accepted collapse of native diagnostics | T? / denied | Native codebook/sentinel semantics, fault coverage, independent boiler evidence and Unknown handling | Read | Flame/pump/off/maintenance are not universal fault or operation facts; do not equate modulation setpoint with measured demand | Proven code/catalog boundaries; Hypothesis native meanings; [B] Direct/Additional/Counter candidates, [G] `refreshBoilerStatusB509`, [S] symbols |
| Controller-mediated boiler faults / return / DHW status — U | controller/boiler/G, boiler refresh | Remote heat-source status/error layouts unverified; validated return and DHW raw-status source absent in [M]; Q? | No qualified `thermal.status.fault` or return-temperature binding supplied | T? / denied | Exact native response layout, correlated fault semantics and sensor role before any mapping | Read unavailable | Do not synthesize fault.clear, zero return temperature or DHW inactive from absent data | Unknown native/semantic evidence; [M] boiler section, [N] OP06 controller-mediated parameters, [G] `refreshBoilerStatusB524` |
| Existing controller/circuit boiler mirrors — W | controller/boiler/G, `mergeBoilerMirrorFieldsFromSnapshots` | Driver still mirrors system flow and circuit state to boiler snapshot; circuit pump mirror is excluded; Q? | No independent boiler SemReg fact from these aliases; DHW mirrors retain original service provenance | T? / denied | Independent direct-boiler evidence and alias removal in owning driver scope | Read legacy mirrors | Exclude boiler/circuit aliasing from this inventory; do not promote `diagnostics.heatingStatusRaw` as boiler state | Proven current mirror; Unknown physical equivalence; [M] boiler boundary, [G] `mergeBoilerMirrorFieldsFromSnapshots`, `mergeBoilerDHWFieldsFromSnapshot` |
| Temperature / mode / schedule commands — W | controller/boiler/G, existing native write functions | Native config/schedule functions exist; inferred FLAGS and candidate encodings do not prove effect or accepted binding; Q? | [S] defines only `thermal.operation.set_temperature` on `zone` and `.set_mode` on `system`; neither establishes a driver operation route | T? / denied | Independent accepted operation binding, exact service/argument/effect, native limits, confirmation and full target behavior | Writes withheld from Matter | Observed setpoint/read capability never implies a writable thermostat; no new credentials, principal, permission or authorization checks | Proven definitions/functions; Unknown qualified operations; [S] operations, [N] candidate writes, [G] `SetSystemConfig`, `SetBoilerConfig`, `SetZoneTimeProgram`, `SetDhwTimeProgram` |

Energy totals, solar-only yield/collector services, ventilation and personal
installer data are outside this heating service slice. Their existence in the
same driver neither qualifies these rows nor establishes full target coverage.
In-scope ancillary pressure, humidity, configuration, counters and schedules
are retained above so representability gaps are visible rather than silently
relabelled as temperature, mode, demand or fault facts.

## Later locked-corpus reconciliation

Perform these searches only in a separately provisioned checkout of the exact
SDK selected by [L]. They are a follow-up evidence request, not commands run for
this inventory. No unrelated SDK or remembered numeric IDs may fill T?.

1. Inspect the exact files named by [L]: `data_model/1.7/spec_sha`,
   `data_model/1.7/spec_tag`, `data_model/1.7/device_types/ElectricalSensor.xml`,
   `data_model/1.7/clusters/ElectricalPowerMeasurement.xml`,
   `zzz_generated/app-common/clusters/ElectricalPowerMeasurement/AttributeIds.h`
   and `examples/all-devices-app/all-devices-common/device/types/electrical-sensor/ElectricalSensor.cpp`.
   This reconciles lock metadata with its existing electrical target only.
2. Find heating candidates without assuming file names:
   `rg -n -i 'thermostat|temperature|water.?heater|hot.?water|heating|pump|humidity' data_model/1.7/device_types data_model/1.7/clusters`.
   Record exact resulting files and definitions for each row; no match is not
   by itself a standards-wide absence claim.
3. In each resulting device/cluster definition, inspect the complete mandatory
   and conditional requirements, revisions, features, attributes, units,
   nullability, ranges, commands, access and conformance dependencies. Search
   `rg -n -i 'mandatory|conformance|feature|attribute|command|revision|access'`
   on those exact resulting files; follow imported definitions and global
   requirements rather than treating the search as a complete schema review.
4. Reconcile generated headers under `zzz_generated/app-common/clusters`,
   actual endpoint composition and locked build/controller evidence. Record
   mandatory facts unavailable from these native paths. Any prospective target
   mapping still needs its public semantic owner and reviewed composition.

Until this work is complete, every target tuple and heating mandatory-cluster
list remains Unknown. No native evidence grade upgrades a target evidence grade.

## Manual contract cases and transition review

These are synthetic review cases, not executable tests or new runtime policy.
No interface or state transition is implemented here.

| Case | Current source behavior / boundary | Prospective binding disposition and falsifier | Source |
| --- | --- | --- | --- |
| Valid instantaneous outdoor observation, qualification still Unknown | Exact candidate exists; qualified use is prohibited | Retain native evidence; no endpoint. Allocating from a valid number alone falsifies the boundary | [F], [Q], [R] |
| Rolling average has the same numeric value as instantaneous temperature | Aggregate remains withheld | Numeric equality must not erase window semantics | [F] `mappings[2]` |
| One circuit read fails while another field succeeds | Driver merges circuit fields non-destructively | Preserve partial last-known-good data and source/freshness; never zero missing fields or renew their receipt time | [G] `mergeCircuitSnapshotNonDestructive`, [M] native boundaries |
| One-byte circuit status arrives for the calculated-flow label | Native catalog contradicts old f32 projection | Keep raw evidence, withhold temperature; accepting raw status as Celsius falsifies mapping | [N] circuits `0x0020`, [G] `refreshCircuits` |
| Manual zone target equals effective target | Catalog roles remain different despite equality | No write support from either observed value; treating equality as identical capability falsifies role preservation | [N] zones `0x0014`, `0x0022`, [S] operations |
| Circuit pump runs while boiler pump does not | Circuit pump is not independent boiler evidence | No boiler pump/action fact from circuit observation | [M] selector assignment, [G] `mergeBoilerMirrorFieldsFromSnapshots` |
| FM5 acquisition degrades, then coherent absence arrives | Transient reason retains last coherent state; structural withdrawal clears affected structure | Distinguish retained data from current evidence and absence; config-only cylinder data cannot resurrect it | [M] solar/cylinders, [G] `publishFM5Semantic`, `applyFM5Acquisition` |
| Identity evidence is malformed, sparse matching, or directly conflicting | [Q] specifies reject-without-mutation, possible retention of existing direct proof, or atomic retirement respectively | These are qualification-contract transitions, not proof a current binding implements them; no supplied accepted proof exists to bootstrap from sparse input | [Q] `transition_contract` |
| Caller supplies successful builds and an accepted-looking profile | Null reviewed profile pin denies before trust in caller fields | Denial remains; this inventory cannot promote profile admission | [L], [R] `evaluate` |

The source mapping [F] preserves receipt and evaluation as separate axes;
evaluation cannot replace receipt, and freshness uses elapsed monotonic time.
Future bindings must preserve native request/reply context, qualification,
evidence reference, asset/service/source identity, source epoch, driver
generation and semantic revision. They must expose loss/unavailability under
their accepted lifecycle contract, preserve usable partial state, and never
substitute another opcode, family, circuit or sensor when evidence is missing.
This binding-local review adds no application lifecycle contract to protocol or
type trees and changes no authentication/authorization behavior.

## Public evidence locators

Links use public `blob/main` destinations for inspection convenience. They are
not immutable qualification artifacts. This inventory was reviewed offline
against supplied source views; link checks establish matching repository/path
and cited content locally, not current network reachability or main freshness.

- [L] [Binding SDK lock](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/sdk/matter-sdk.lock.json): `sdk.data_model`, `target`, accepted profile pin and mapping owner.
- [D] [SDK baseline](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/SDK_BASELINE.md): existing electrical fixture and readiness boundary.
- [R] [Production readiness](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/runtime/production_readiness.py): `evaluate`, `validate_composition`.
- [S] [SemReg thermal definitions](https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/thermal/definitions.go): `fields`, `services`, `capabilities`, `operations`.
- [N] [B524 register catalog](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md): system, native DHW, circuits, zones and family-specific evidence.
- [P] [B524 wire contract](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524.md): complete selectors, read-response correlation and inference limits on FLAGS.
- [B] [Direct boiler register catalog](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B509-boiler-register-map.md): direct-boiler candidates and unresolved meanings.
- [M] [B524 semantic correspondence](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/b524-semantic-mapping.md): source/public-surface correspondence and explicit non-claims.
- [F] [Typed thermal mapping](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/fixtures/vaillant-semreg-thermal-map-v1.json): exact candidates, withholding, lifecycle and qualification.
- [Q] [Controller qualification gap](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/fixtures/vaillant-controller-qualification-v1.json): evidence status, permitted use and transition contract.
- [G] [Current gateway driver](https://github.com/Project-Helianthus/helianthus-ebusgateway/blob/main/cmd/gateway/semantic_vaillant.go): functions named in each matrix row; code proves implementation shape, not native qualification.

Remaining parent acceptance includes full locked target inventory, reconciled
mandatory composition and public target mappings, qualified native/service
bindings, accepted profile/build evidence, independent review and applicable
hosted checks. Physical/live-fabric acceptance is separate and is not performed
or authorized by this inventory. Document readability and offline link inspection
are not semantic, runtime, interoperability, certification or hardware validation.
