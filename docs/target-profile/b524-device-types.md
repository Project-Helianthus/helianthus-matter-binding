# B524 source groups and candidate Matter device types

This is the named source-inventory slice for [issue #43][issue], advancing
[the device-type matrix #23][matrix] and [parent #8][parent]. It reconciles
B524 system, zone, circuit, DHW, cylinder, solar and associated pump/valve
observations with candidate target searches. It does not close either parent,
select a minimal subset, or duplicate the general domain inventories.
Nothing here is stable before Helianthus 1.0.

The canonical [heating inventory][heating] and [air-conditioning inventory][ac]
retain their individual fact rows, command gaps and driver locators. This page
adds the group-to-candidate ledger, including solar groups excluded from the
heating slice. Native interpretation belongs to the public [B524 catalog][native];
protocol-neutral definitions remain in [SemReg][thermal], and accepted Matter
target mappings remain with the public semantic documentation owner identified
by the reviewed lock. This page is binding-local reconciliation, not an accepted
target mapping or a native decoding contract.


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
| T-S; B01–B16 and RC03/RC04 sensor/measurement/context joins | Unknown temperature/humidity device and cluster identities/features; retain all bundles and role/alias conflicts. |
| T-H; RC01/RC02 thermostat, SystemMode, RunningState and Demand | Unknown prior identity → Thermostat revision 11 source presence; command, heating/cooling and preset closure Unknown. |
| T-U; RC05/RC06 pumps, valves/mixers; RC07/RC08 energy management/storage and water/configuration/counters | Unknown joins and actuator targets; DEM revision 4 / EEM revision 2 source presence supplies neither activation effects nor energy semantics. No reverse operation accepted. |

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

## Evidence, identity and bounded coverage

**Proven** describes only what the supplied public source establishes.
**Hypothesis** identifies a proposed meaning or correspondence requiring the
listed falsifier review. **Unknown** records absent or conflicting evidence.
Native installation qualification is **Unknown in every row**; no row permits
qualified use. In particular, the catalog's solar `†` annotations mean
CSV value-matching only, with false-positive risk, not independently confirmed
solar semantics. Historical FLAGS/RW labels do not establish write behavior.

The B01–B16 ledger directions are **native-to-Matter observation candidates**,
abbreviated **R**. Configuration and setpoints in R rows are read observations
only. The reverse reconciliation below separately inventories acquisition
references, operation candidates and absent supported operations. Source family scope is eBUS
controller and functional-module families, with profile-specific applicability;
remote-room observations are cross-references to the existing air-conditioning inventory,
not a new survey of remote device types.

Each source reference below is rooted in the public native catalog first.
`GG/RR` abbreviates `OP=0x02, OT=0x00, GG, II, RR`; retain the full native
owner/profile, opcode, operation role, group, instance, register, sensor/service
role and evidence context. System and native DHW use their documented singleton
context; circuits, zones and cylinders retain their characterized instance.
Other opcodes and profile variants are never aliases merely because GG/RR match.
Source receipt, validity, qualification and freshness remain separate.

Apply the ordered join key and alias rules in
[inventory-completeness][completeness]: domain, native owner/profile,
capability/fact, direction and target component. The group rows below are
bounded review bundles of canonical facts, not new FactKeys or endpoint keys.
Split each bundle into individual joins when actual target components are
known; changing a key field creates a new join with a supersession reference.
An unresolved profile or target is a gap slot, never a match between unknowns.

## Target evidence and closure ledger

**T0** applies to every source row: the adopted complete Matter 1.6.1 normative model contents and
compatible accepted public semantic target evidence were not supplied. Device
type, cluster, revision, feature selection, attributes, commands, events, types,
scaling, nullability and mandatory closure are **Unknown** for these joins.
All rows are **blocked pending evidence**, with zero representable joins and no
negative claim that the standard lacks a suitable target. No numeric normative
target identifiers are inferred from memory or from native register numbers.

The following labels are search candidates already requested by the canonical
inventories, not verified schema elements. No additional pump, valve, solar or
water-heater type is asserted without the adopted normative corpus.

| Closure reference | Candidate type / cluster / feature search and supported rationale | Evidence still required for closure |
| --- | --- | --- |
| T-S | Temperature/Humidity sensor searches; `TemperatureMeasurement` and `RelativeHumidityMeasurement` search terms from air-conditioning “Locked target evidence and remaining reconciliation”. Source quantities motivate investigation only. | T0; actual device and cluster definitions, revisions, sensor applicability, unit conversion, ranges, unavailable/null behavior, reporting and every mandatory dependency. Water, air and collector roles require separate joins. |
| T-H | `Thermostat`, heating/cooling, `SystemMode`, `RunningState` and `Demand` searches from the canonical inventories. Modes/setpoints/action candidates motivate investigation only. | T0; evidenced heating versus cooling capabilities, enum correspondence, mandatory feature conditions, setpoint roles and bounds, commands/effects, composition and reporting. Observations cannot satisfy mandatory command behavior by analogy. |
| T-U | Unresolved target slot for topology, water services, configuration, energy/counters and actuators. Native group/type names alone supply no Matter device type, cluster or feature. | T0 plus accepted role-specific semantic evidence before target selection. A counter, boolean, percentage or temperature alone cannot close a physical device type. |

For each eventual target, follow every unconditional and selected-feature
dependency through the adopted normative device/cluster definitions until no
requirement is unreviewed. Record condition, applicability, required fact and
joined source row. Unknown applicability blocks closure; an optional fact does
not satisfy an unrelated mandatory one, and omission is not an evidenced null.
Inspect the public semantic mapping and its compatibility with the lock as a
separate requirement. The electrical fixture described by the canonical
inventories supplies no heating, cooling or solar closure.

## Source-to-target ledger

Q? means native qualification Unknown throughout. Driver implementation was
not independently inspected here: function locators below are **reported by
the canonical inventories**, not fresh code evidence. Where those inventories
do not establish acquisition, the driver gap is explicit. Each target cell
inherits T0 and its stated closure reference.

| Row / native source reference and canonical row | R: source role / unit; identity and driver evidence | Native confidence / qualification | Target evidence / closure | Loss or withholding | Concrete falsifier for the proposed join |
| --- | --- | --- | --- | --- | --- |
| B01 system identity/configuration: [native “System”][system]; heating “System service identity”, “Physical installation / circuit ownership”, “System pressure / heating configuration / bounds” | R metadata/configuration; units vary, not one measurement. Keep system/profile and module interpretation; heating reports `refreshSystem`, `publishCircuits`. | Proven catalog/inventory structure; Unknown physical topology; Q? | T0 / T-U; system is not a verified Matter type | Scheme, counts, bounds and display identity cannot allocate devices or establish plumbing. Remaining system facts stay in the canonical inventory. | A module/scheme change leaves the same supposedly qualified physical ownership. |
| B02 system instantaneous vs aggregate: [GG00/RR0073, RR004B, RR0095][system]; heating outdoor instantaneous, system flow and outdoor rolling-average rows | R f32 °C; respectively outdoor air, system water flow and daily-average label (rounded average updated every three hours). Separate sensor/aggregate roles; `refreshSystem` reported. | Proven catalog roles; Hypothesis future sensor joins; Q? | T0 / T-S for instantaneous candidates; T-U for aggregate role | No B509 flow or OP06 heat-source alias. Average is withheld from instantaneous temperature; profile-qualified float codec required. | Equal values cause average and instantaneous readings to collapse into one fact. |
| B03 zone identity/room observations: [GG03/RR0013, RR001C, RR000F, RR0028][zones]; heating zone presence/room/humidity and AC zone rows | R association metadata, f32 °C air and % relative humidity. Retain zone, sensor, assignment and profile; `refreshState` / discovery reported. | Proven catalog observations; Hypothesis sensor binding; Q? | T0 / T-S; no inferred physical thermostat | Remote and circuit humidity mirrors require alias evidence, not independent sensor counts; a display index is not physical ownership. | An unassigned remote overwrites a zone value or two zone instances overwrite one key. |
| B04 zone targets/modes: [GG03/RR0022, RR0014, RR0023, RR0015, RR0006, RR0001, RR000E][zones]; canonical manual/effective/override and mode rows | R f32 °C targets and raw mode/special-function codes. Manual heating, effective heating, manual cooling and effective cooling remain distinct roles; `refreshZoneSlowConfigFields` / derivation reported. | Proven catalog role distinctions; Hypothesis family enum applicability; Q? | T0 / T-H | No setter, schedule, heat/cool or preset equivalence follows from a read. Preserve primary/fallback origin and unsupported symbols. | An effective target is offered as writable, or manual/away is silently relabelled comfort/eco. |
| B05 circuits presence/capability: [native “Circuits” introduction, GG02/RR0002, RR0006][circuits]; heating service/type/cooling row | R type enum and cooling configuration bool, no physical unit. Per-profile circuit instance; `refreshCircuits` reported. | Proven scoped discovery observations; Unknown generalized topology; Q? | T0 / T-H or T-U; type enum is no target type | Virtual native-water slot is not a physical heating circuit; temperature/status is not its presence predicate. Enabled cooling is not action. | A virtual slot becomes a physical endpoint from a temperature, or enabled cooling becomes running cooling. |
| B06 circuit flow/control context: [GG02/RR0008, RR0007, RR0022, RR0023, RR0020][circuits]; heating flow/setpoint/calculated-flow/humidity rows | R flow measurement/observed target °C, humidity %, dew point °C; RR0020 is u8 raw. Retain circuit and quantity role; `refreshCircuits` reported. | Proven catalog distinctions and documented raw-width conflict; Hypothesis joins; Q? | T0 / T-S; T-H only a search for control context | No boiler return alias, no dew-point-as-room-air, no calculated Celsius from RR0020. Limits/curves remain in canonical configuration rows. | Raw RR0020 becomes a temperature, or a circuit-flow target becomes a room target. |
| B07 circuit pump/state/counters: [GG02/RR001B, RR001E, RR0024, RR0025][circuits] and [Circuit State Enum][state]; heating pump/state/counter rows | R u16 raw pump observation versus candidate state enum, u32 hours and starts count. Local circuit owner; `refreshCircuits` reported. | Proven raw distinction/counter catalog; Hypothesis pump Boolean and RR001E state meanings; Q? | T0 / T-U; T-H is only an action search for RR001E after correlation | No exact generic pump/counter SemReg fields supplied; do not publish boiler pump, system operation or cooling from RR001B. | Pump off while another circuit state is reported forces both selectors into the same action, or circuit pump data creates boiler-pump evidence. |
| B08 mixer and zone valve: [GG02/RR001A, RR0021][circuits]; [GG03/RR0012][zones]; heating mixer/counter and zone-action rows | R mixer movement signed f32 % (closing/opening), position f32 %, zone valve u16 boolean label. Distinct mechanical facts and instances; mixer position and zone action acquisition reported, movement acquisition not established here. | Proven catalog distinctions; Hypothesis action correspondence; Q? | T0 / T-U; T-H action search cannot establish valve type | Movement is not position; neither percentage is percent thermal demand. Open valve does not prove delivered heat. | A closing movement is published as absolute position, or open valve alone becomes qualified heating action. |
| B09 native DHW presence/temperature/target: [GG01/RR0001, RR0005, RR0004, RR0008][dhw]; heating DHW presence/current/target rows | R exact two-byte nonzero UIN presence predicate; f32 °C measured DHW, observed DHW target and flow target. Keep singleton DHW service and role; `refreshDHW` / slow config reported for current and target, flow-target acquisition not established. | Proven documented predicate and quantity roles; Hypothesis joins; Q? | T0 / T-S or T-U; no water-service type established | Wrong-width/failed/unmatched predicate is Unknown, not absent. No zone setter or automatic cylinder sensor alias. | Wrong-width input is treated as absence, or observed target enables a write. |
| B10 DHW pumps and lifecycle: [GG01/RR0002, RR0006, RR0003, RR000D, RR000F][dhw]; [GG02/RR0018][circuits]; heating DHW mode/special-function and circuit rows | R distinct u8 circulation/DHW pump booleans, mode/special-function codes and circuit-local u16 circulation status. Preserve native DHW vs external circuit service. Canonical mode derivation reported; exact pump acquisition/semantic binding Unknown here. | Proven catalog distinctions; Hypothesis special-function interpretation; Q? | T0 / T-U | Pump observation is not heat delivery, selected mode, one-time loading command acceptance or independent equipment presence. | Native and circuit-local circulation observations collapse without alias proof, or special-function state triggers a write. |
| B11 system cylinder sensors: [GG00/RR0067, RR0068, RR009D–RR00A0][system]; heating cylinder top/bottom row | R f32; units not established in RR0067/68 catalog cells, NaN denotes no module; RR009D–A0 explicitly °C for DHW/heating top/bottom. Preserve register, role and cylinder identity; top/bottom reads reported without proving every selector's acquisition. | Proven catalog role/unit distinctions; Unknown cross-selector equivalence; Q? | T0 / T-S only after quantity and sensor qualification | Never assume generic cylinder pair aliases DHW/heating pairs or GG05. No top/bottom merge or outlet-temperature substitution. | Two distinct sensor roles get one FactKey, or unknown-unit observations are asserted Celsius from their names. |
| B12 solar-loaded cylinder: [native “Solar Loaded Cylinder”, GG05/RR0001–0004][cylinders]; heating functional-module cylinder and configuration rows | R per-instance temperature °C, maximum °C, start/stop differences K; fm5_config ≤ 2. Live decodable RR0004 required by documented presence rule, not config-only replies. `readCylinderSnapshots` / FM5 publication reported. | Proven documented presence rule; Hypothesis CSV-only quantity correspondences; Q? | T0 / T-S temperature candidate; T-U configuration | No cylinder count from static settings; max/delta are not DHW targets; physical association with B11 remains unresolved. | Configuration alone creates a cylinder, or a K difference becomes an absolute temperature. |
| B13 solar collector/yield temperature: [GG04/RR0003, RR0007][solar] | R f32 °C candidate collector/yield temperatures, singleton, fm5_config ≤ 2. Exact driver acquisition/semantic attachment not established by supplied inventories. | Hypothesis CSV-only meanings; Q? | T0 / T-S search only after semantic fluid/sensor-role evidence | Collector temperature is not established room air or water temperature solely by unit; no solar service defined in supplied Thermal services. | A collector name alone selects air/water measurement or binds it to a DHW service. |
| B14 solar flow/yield: [GG04/RR0009, RR000A][solar]; [GG00/RR0019, RR003D, RR00D2–00D5][system] | R RR0009 f32 raw, RR000A representation/unit Unknown; RR0019 f32 unit unestablished; RR003D u32 kWh total; monthly/yearly f32 units unestablished. Exact acquisition absent here. | Hypothesis CSV-only/explicit candidate meanings where annotated; Proven catalog RR003D unit; Q? | T0 / T-U | Raw flow cannot supply `thermal.measurement.flow_rate` in litre/minute; kWh energy cannot supply watt power. No metering/PV duplication or invented integration window. | Raw flow is labelled litre/minute, or energy total is published as instantaneous power. |
| B15 solar pump/valve: [GG04/RR0002, RR0008, RR000B][solar]; [GG00/RR00B8][system] | R kick-enable setting vs pump-active u8 candidate, runtime u32 hours; diverter u8 unit/codebook Unknown. Retain source group/role; exact driver acquisition absent here. | Hypothesis CSV-only solar meanings; Unknown diverter semantics beyond catalog label/raw observation; Q? | T0 / T-U | Kick-enable is not current status. No actuator target, binary valve position or command acceptance inferred. | Kick-enabled creates running pump state, or a diverter code becomes a qualified open/closed valve. |
| B16 solar/cylinder settings and commands: [GG04/RR0001, RR0004–0006, RR000C][solar], [GG05/RR0001–0003][cylinders], [GG00/RR000B, RR000D, RR0027, RR0029, RR0080–0081, RR0089–008A][system]; heating cylinder configuration row | R catalogued limits °C/K where explicit, kick/bleed/offset settings with unresolved units retained as such. GG04/RR000C representation Unknown. Reset command labels are inventory only; no write execution. Exact additional acquisition absent here. | Hypothesis annotated meanings; Unknown unresolved codecs/units/effects; Q? | T0 / T-U | Do not reinterpret protection limits as measured temperatures, power-limit names as measured watts, or reset labels as safe operations. General settings remain in canonical inventories/catalog. | A reset/configuration label alone enables an operation or a protection limit replaces a sensor reading. |

## Reverse reconciliation: fabric consumers to native candidates

This partial continuation of [#23][matrix] and [#8][parent] adds the requested
five family searches. A **reverse search** starts with a fabric-consumed role;
it is not an implemented fabric-to-B524 adapter. An **acquisition mapping**
locates native read evidence, an **operation candidate** locates a published
possible write encoding, and a **supported operation** requires independently
accepted native capability, constraints, response correlation and effect/readback
evidence. None of the rows below establishes a supported operation. Receiving a
fabric sensor value supplies no native injection or setter contract. Protocol
§4.8 explicitly documents no template or sequence as safe to send: an ACK
does not prove device state, and separately established safety procedure and
target qualification remain necessary. This document authorizes no write.

The public [register catalog][native] and [B524 protocol][protocol] are the native
authority first. Read `L(GG,II,RR)` as exact `OP=0x02, OT=0x00` with the listed
group, instance and register; `D(GG,II,RR)` as `OP=0x06, OT=0x00`. `Z` is the
concrete zone instance, `C` the concrete physical circuit, and `M` the concrete
remote instance. Their identity must be retained, not replaced by a display
index: characterized zone bounds are `0x00..0x0A`, physical circuits
`0x00..0x08` (virtual `0x09` excluded), and remote slots `0x01..0x08` for the
characterized profile only. Other profiles and actual presence remain separate
evidence gaps. Singleton reads use `II=0x00`; write `II=0xFF` below is a literal
candidate write selector, never an alias for the read instance.

Every row inherits **T0**, native qualification **Unknown**, and admission
denied. Proven here means a published catalog/template fact only; candidate
meaning/effect remains Hypothesis or Unknown as specified. Qualification stays
transport-agnostic and requires native Go **go-fabric on the adopted Matter 1.6.1 model** evidence;
no complete retained normative target corpus or independent qualification is supplied. This is a
qualification direction, not proof that the library supports these families,
not a dependency-pin change, and not a replacement for native qualification.

| Reverse row / family search and canonical references | Acquisition reference: exact selector and role | Operation candidate / existing native boundary | Unresolved target obligations and rejection case |
| --- | --- | --- | --- |
| RC01 thermostat / room target: [thermostats](thermostats.md) R06–R08, O01; B04; catalog “Zones”; protocol §4.10 | `L(0x03,Z,0x0022)` manual heating target, f32 °C; `L(0x03,Z,0x0007)` comfort target, codec/°C Hypothesis; `L(0x03,Z,0x0014)` effective computed target, f32 °C. Assignment reads `L(0x03,Z,0x0013/0x001C)` retain sensor-binding/circuit-index roles. | Published candidates: `OP=0x02, OT=0x01, GG=0x03, II=Z, RR=0x0022` for the VRC720 family, versus `RR=0x0007` for the VRC700 family, float32 LE. Hypothesis applicability/encoding; accepted effects Unknown. Protocol orders mode before setpoint but proves neither acceptance nor zone=circuit identity. SemReg zone `set_temperature` vocabulary does not implement either route. | T-H / T0; exact target command/attribute access, heating/cooling roles, bounds, precision, composition and mandatory dependencies. Measured RC03 or effective RR0014 must never enable a setter. Family selectors cannot substitute for each other. |
| RC02 thermostat / room modes and overrides: thermostats R09–R26, O02–O05; B04–B06; catalog “Zones”, “Open questions”; protocol §§4.9–4.10 | `L(0x03,Z,0x0006)` heating mode; `L(0x03,Z,0x0001)` cooling mode; `L(0x03,Z,0x0002/0x0023/0x0015)` configured/manual/effective cooling targets; `L(0x02,C,0x0006)` cooling configuration. These are distinct reads, not capabilities to write. | Mode candidate `OP=0x02, OT=0x01, GG=0x03, II=Z, RR=0x0006`, u16 LE (candidate codes 0/2); meaning/effect Hypothesis/Unknown. Quick-veto candidates `OP=0x02, OT=0x01, GG=0x03, II=Z, RR=0x0008/0x0026/0x000E` (float32/float32/u8) remain with thermostats O03 and protocol §4.9: family-specific sequence, units and effects unresolved. Holiday/cooling routes lack accepted operation contracts (O04/O05). SemReg system `set_mode` cannot be coerced into a zone operation. | T-H / T0; feature applicability, enum/preset semantics, scheduling, cooling capability, command completion and all mandatory facts. Heating evidence cannot establish cooling; an acknowledgement alone cannot establish effect. |
| RC03 temperature sensor consumption: thermostats R01/R03/R05; [air conditioning](air-conditioning.md); B02/B03/B06/B09/B11–B13; catalog “Zones”, “Remote Control Thermostats” | Room air `L(0x03,Z,0x000F)` and `D(0x0A,M,0x000F)`, f32 °C, remain assignment-qualified separate sources; remote NaN is unavailable. Other water/outdoor/collector roles stay in their canonical B rows. | No published accepted fabric-sensor injection, measurement setter or calibration operation in these references. Measurement is acquisition evidence only; manual target candidates RC01 have a different role/direction. | T-S / T0; actual sensor device/cluster, placement, ranges, scaling, nullability, reporting and mandatory composition. Equal Celsius values do not alias room air, water and outdoor sensors or create a control path. |
| RC04 humidity sensor consumption: thermostats R02/R04/R05/O06; AC humidity rows; B03/B06; catalog “Zones”, “Circuits”, “Remote Control Thermostats” | `L(0x03,Z,0x0028)`, `L(0x02,C,0x0022)`, `D(0x0A,M,0x0007)`, f32 relative %. Zone/circuit/remote assignment remains unresolved; remote NaN is unavailable. | No accepted humidity setter, sensor injection or dehumidification operation. Thermal humidity vocabulary proves no native write or safe cooling function. | T-S / T0; sensor identity and alias reconciliation, unavailable representation, reporting and every mandatory feature/attribute. Shared % units establish neither common sensor identity nor humidity control. |
| RC05 pump consumption: [pumps](pumps.md) P1–P9; B07/B10/B15/B16; catalog “Circuits”, “Solar Circuit”, “Native Domestic Hot Water” | Circuit `L(0x02,C,0x001B)` raw u16 pump status, distinct RR001E state, RR0024 hours and RR0025 starts. Solar `L(0x04,0x00,0x0008/0x000B)` candidate activity/runtime; DHW `L(0x01,0x00,0x0002/0x0006)` and external-DHW `L(0x02,C,0x0018)` retain separate roles. | No accepted pump start/stop/speed operation. Solar `L(0x04,0x00,0x0001/0x0002)` are read catalog reset-command/kick-configuration roles, not established write encodings. Solar meanings Hypothesis (CSV-only); solar instance conflict remains with P5–P8. Direct-boiler pump mapping is explicitly absent (P9), not supplied by circuit data. | T-U / T0; target actuator tuple, status/control distinction, counters, limits and mandatory composition. Raw status, reset labels and enabled kick cannot establish actuator support, running state or accepted reset effect. |
| RC06 valve / mixer consumption: heating mixer and zone-action rows; B08/B15; catalog “Circuits”, “Zones”, “System” | `L(0x02,C,0x001A)` signed movement %, `L(0x02,C,0x0021)` position %, `L(0x03,Z,0x0012)` u16 valve Boolean label; `L(0x00,0x00,0x00B8)` u8 diverter with Unknown codebook/unit. | Concrete read references exist; accepted position/open/close/movement write mapping is absent. Catalog configuration/FLAGS labels prove no write route. Movement, position and thermal demand remain distinct roles. | T-U / T0; valve versus mixer target, travel/position semantics, commands, failure state and mandatory feature/attribute closure. A percentage cannot select an actuator type; open valve cannot prove heat delivery. |
| RC07 energy-management activation: AC energy-manager row; catalog “System”; protocol §4.11 | `L(0x00,0x00,0x00B6)` external-energy-management-enabled u8 read; `L(0x00,0x00,0x0048)` energy-manager action/demand state is separate, not quantified demand. | Published write candidate `OP=0x02, OT=0x01, GG=0x00, II=0xFF, RR=0x00B6`, u8 0/1. Encoding/meaning Hypothesis; controller reaction and meaning of 0/1 Unknown. No accepted energy-management native operation or supported effect follows. | T-U / T0; actual energy-management device/cluster/features, command scope, constraints, reporting and all mandatory dependencies. Candidate activation must not imply supported dispatch, power limits, demand percentage or fabric authority. |
| RC08 energy / thermal storage context: B14/B16; [metering](metering.md), [PV](photovoltaic.md), [storage](storage.md), [EVSE](evse.md); catalog “System”; protocol §4.11 | `L(0x00,0x00,0x003D)` solar total u32 kWh; RR00D2–00D5 f32 period totals with units unresolved. `L(0x00,0x00,0x00A2)` buffer-offset configuration remains distinct from measured energy or power. | Published buffer-offset candidate `OP=0x02, OT=0x01, GG=0x00, II=0xFF, RR=0x00A2`, float32 LE, K candidate; controller reaction Unknown. No accepted energy dispatch, counter-reset or demand-response mapping. Other domains remain in their canonical inventories, not aliases of this B524 context. | T-U / T0; target energy/power interval, flow convention, reset/rollover, offset units/limits and mandatory composition. kWh cannot fill watts; a buffer offset cannot establish an energy-management function. |

### Reverse coverage accounting and carried gaps

RC01–RC08 are **eight review bundles across five requested family searches**
(thermostat/room control, temperature/humidity sensors, pumps, valves/mixers
and energy management). They record concrete read references or explicit
absent mappings, not eight unique facts or eight target types. Repeated selectors point back to
B01–B16 and the pump/thermostat/domain inventories. Count canonical source
facts once after alias review; operation directions require separate join keys.
There are **zero accepted representable joins and zero established supported
reverse operations in this slice**. The full fact/target denominator is Unknown;
no percentage of full coverage or completed reverse target sweep is claimed.

Remaining gaps include other thermostat/controller/module profiles, remote
regulator roles, the unresolved solar instance scope, direct-boiler sources,
all remaining B524 groups and opcode families, schedules/time programs,
errors/service state, and all other representable device families. OP00 system
information, OP01/07 descriptions, OP03/04 programs, OP05 unknown layout and
OP08–0C event/status candidates require their own native reconciliation;
OP02/06 group numbers never alias these selectors. The full every-group and
every-opcode matrix remains required by #23. Missing adopted Matter 1.6.1 normative device,
cluster, feature and mandatory-attribute contents and accepted semantic mappings
block the entire target sweep, even where a native read or candidate write is
named. Unknown-filled rows cannot close #23, #8 or the inventory checkpoint.

## Alias reconciliation and remaining sweeps

B01–B12 reference the existing heating/air-conditioning canonical rows rather
than adding another domain inventory. B13–B16 supply the solar group gap in
this bounded roster; they do not inventory photovoltaic generation or electrical
metering. Cross-domain reuse of a solar total must cite B14's native fact and
retain its energy/time role. Count a proven alias once as a source fact, and
count each target join separately. The sixteen bundles are not a count of
unique facts, supported types, physical devices or closed targets.

Unresolved aliases remain gaps: zone/circuit/remote humidity; system versus
native DHW versus solar-cylinder temperatures; virtual native-water versus
physical circuits; controller mirrors versus direct-boiler evidence; pump
versus circuit state. Equal values and similar names prove none of these.
Remote OP06 sources remain in the AC remote rows; B509 and OP06 heat-source
behavior remain in heating's boiler boundary. They are not reclassified as
local B524 evidence. Ventilation, DeltaT, unknown local registers and other
controller-mediated groups are outside this named roster, not assessed as
unrepresentable and not removed from the full inventory denominator.

The completeness contract still requires an inspected driver-group sweep,
adopted target-tree sweep, all aliases/conflicts reconciled and mandatory closure
for each individual join. Reported function names do not replace driver source
inspection. Missing solar acquisition, role-specific semantic contracts, adopted
target definitions and public mappings remain explicit follow-ups to their
owning repositories. Populating Unknown cells completes neither sweep nor the
full-inventory checkpoint. No sibling output is a build dependency here.

## Manual examples and unchanged runtime boundary

These are reasoned synthetic review cases, not executed runtime tests:

| Example | Required review disposition |
| --- | --- |
| A zone and circuit each report 45% humidity | B03/B06 retain separate provenance until sensor association proves an alias; numeric equality alone neither deduplicates nor creates two sensors. |
| Cylinder configuration succeeds but its temperature is unavailable | B12 cannot establish new presence. A transient acquisition gap preserves applicable partial last-known-good data with its original freshness; coherent structural withdrawal remains distinct from transient failure. |
| A collector reports 40 °C and a raw flow scalar of 2 | B13/B14 remain blocked: collector role and raw flow unit do not establish water-temperature/flow-rate facts or a solar target composition. |
| A pump-active observation accompanies a closed zone valve | B07/B08/B10/B15 preserve each native owner and raw role. Neither observation alone establishes thermal action or a pump/valve endpoint. |
| All sixteen bundles are filled and a sample build succeeds | T0 remains unresolved; no representable join, full inventory completion or production admission follows. |

As recorded by the canonical inventories, the reviewed lock has no accepted
runtime composition profile and admission fails closed before trusting caller
metadata/build claims. This document changes no lock, SDK selection, readiness
rule, interface or endpoint allocation. Future consumers must preserve detached
native evidence, source epochs, partial last-known-good state, deterministic
envelopes, eBUS MCP compatibility and GraphQL parity; unavailable observations
must not become zero/false/fresh readings. No authorization gate is added before
0.9; authentication/authorization remain 0.11 subjects.

This source/meaning/identity/loss/example/public-link review uses supplied public
source views only. Public `blob/main` links locate files, not immutable evidence
or verified current network content. There is no full CI, transport, runtime,
native qualification, live-fabric, interoperability, certification or hardware
PASS. Independent review, applicable hosted checks and remaining parent
acceptance are separate; no commissioning, deployment or upstream action is
authorized or performed.

[issue]: https://github.com/Project-Helianthus/helianthus-matter-binding/issues/43
[matrix]: https://github.com/Project-Helianthus/helianthus-matter-binding/issues/23
[parent]: https://github.com/Project-Helianthus/helianthus-matter-binding/issues/8
[heating]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/heating.md
[ac]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/air-conditioning.md
[completeness]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md
[thermal]: https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/thermal/definitions.go
[native]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md
[system]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#gg0x00--system
[dhw]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#gg0x01--native-domestic-hot-water
[circuits]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#gg0x02--heating-circuits-multi-instance
[zones]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#gg0x03--zones-multi-instance
[solar]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#gg0x04--solar-circuit
[cylinders]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#gg0x05--cylinders-multi-instance
[state]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#circuit-state-enum

[protocol]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524.md
