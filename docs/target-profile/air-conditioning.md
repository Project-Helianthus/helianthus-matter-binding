# Air-conditioning source coverage and command gaps

This operator-directed inventory is a **partial source-side contract** for
thermal cooling, room temperature, relative humidity, mode and demand
observations. It advances issue #10 and [parent #8][parent]; it cannot close the
parent or release M42/M43. Nothing is stable before Helianthus 1.0.

Current behavior below comes from the public native register catalog first,
then the documented gateway projection and SemReg definitions. Proposed
binding dispositions are inventory decisions, not implemented adapters, new
native facts or universal Matter mappings. Protocol-neutral contracts remain
with SemReg; Matter target mappings remain with the public semantic documentation
owner named in the [lock][lock]. This file owns binding coverage and gaps only.

## Evidence and disposition key

Each row keeps four independent axes:

- Native **Proven** means the cited catalog documents corroborated evidence in
  its characterized family scope; it does not qualify another installation.
  **Hypothesis** marks an interpretation requiring corroboration. **Unknown**
  means evidence is missing. Catalog FLAGS and historical RW/RO labels do not
  prove writability; byte order and presence require profile qualification. Proven below describes
  the public catalog evidence, not independently replayed telegrams or inspected
  driver code; annotations and conflicts override the catalog default.
- **Candidate** means an existing SemReg vocabulary could represent the fact
  after the stated conditions. **Withheld** means a semantic mismatch or
  unresolved interpretation prevents that projection. **Unavailable** means
  the source owner or implementation evidence was not inspected. None means
  currently composed hardware support.
- **T?** in every target cell means device type / cluster / feature / attribute,
  their IDs, revisions, types, scaling, nullability and mandatory composition
  are **Unknown** for this domain in the locked corpus. Suggested search terms
  below are reconciliation work, not target-support claims.
- **A0** in every admission cell means production allocation is denied. The
  reviewed lock has `accepted_runtime_composition_profile_sha256: null`;
  [runtime `evaluate`][admission] denies before trusting caller composition or
  build evidence. No row changes that boundary or proves qualification,
  interoperability, certification, fabric readiness or hardware acceptance.

`R` means observation toward a possible output binding; `W withheld` means a
possible command vocabulary exists but no command is offered here. All SemReg
references below belong to `helianthus.pack.thermal`; [definitions][thermal]
prove vocabulary, not installed driver-to-pack bindings. `zone`, `circuit` and
`system` abbreviate `thermal.service.zone`, `.circuit` and `.system`.

Common mandatory-fact gaps are **C**: qualified source identity and scope,
zone/sensor/circuit association, actual capabilities, accepted target model,
mandatory features/attributes/commands, endpoint composition and matching build
and reporting evidence. Thermostat composition remains Unknown. **S** adds
sensor placement, validity, unit/range and unavailable-value handling. **M**
adds mode enum correspondence, capability versus action and scheduling/preset
loss. **D** adds demand units, extent and meaning. **W** adds native write
acceptance, limits, response correlation, readback/effect and partial-failure
semantics. These are evidence gaps, not new authorization checks.

## Inspected eBUS controller families: observations

Owner for these rows: `helianthus-ebusgateway`, `semantic_vaillant.go` (driver
source path reported by [the mapping][mapping], implementation not supplied),
with native contract in `helianthus-docs-ebus`,
`protocols/vaillant/ebus-vaillant-B524-register-map.md`. OP02 selectors below
use the characterized instance, never a universal index assumption.

| Native capability/fact group and source path | Native evidence / qualification | SemReg field / service and disposition | Target type / cluster / feature / attribute | Mandatory gaps | Direction | Loss / withholding and concrete candidate falsifier | Grade | Admission | Exact public evidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Zone room temperature: OP02/GG03/RR000F; `refreshState` | Catalog-corroborated f32 °C; current `[].state.current_temp_c`; installation qualification open | `thermal.measurement.air_temperature` / zone; candidate | T? | C, S | R | Keep measured room scope; falsified if sensor binding is unresolved or sample is not valid room air temperature | Proven source; Hypothesis projection | A0 | [Native zone rows][zones]; [current zones mapping][zone-map]; [field][thermal] |
| Zone room relative humidity: OP02/GG03/RR0028; `refreshState` | Catalog f32 %; current `[].state.current_humidity_pct`; no universal family qualification | `thermal.measurement.humidity_relative` / zone; candidate | T? | C, S | R | Preserve zone/sensor provenance; falsified by non-relative quantity, invalid sample or unknown binding | Proven source; Hypothesis projection | A0 | [Native zones][zones]; [current zones mapping][zone-map]; [field][thermal] |
| Circuit room humidity: OP02/GG02/RR0022; `refreshCircuits` | Catalog f32 %, from room sensor; current circuit humidity projection | `thermal.measurement.humidity_relative` / circuit; candidate | T? | C, S | R | Do not duplicate a zone sensor as independent hardware; falsified if circuit-to-room association cannot be evidenced | Proven source; Hypothesis projection | A0 | [Native circuits][circuits]; [current circuits mapping][circuit-map]; [field][thermal] |
| Circuit cooling enabled: OP02/GG02/RR0006; `refreshCircuits` | Catalog u8 configuration flag; current raw `[].config.cooling_enabled`; not a validated HA configuration contract | No exact capability-flag field established; withhold substitution into `thermal.mode.system` / system | T? | C, M | R | Enabled capability does not mean actively cooling or selected cool mode; native fact retained | Proven scoped source; Unknown exact projection | A0 | [Native circuits][circuits]; [mapping caveat][circuit-map]; [mode vocabulary][thermal] |
| Circuit action: OP02/GG02/RR001E; `refreshCircuits` | Enum assignment naming-based; no correlated native observation of selector; current `circuit_state` candidate | `thermal.action.state` / circuit; candidate only after qualification | T? | C, M | R | Raw 2 is candidate cooling; unknown values retained; falsified by a correlated state disagreement or a non-enum layout | Hypothesis native and projection | A0 | [Circuit State Enum][circuit-state]; [current mapping][circuit-map]; [action field][thermal] |
| Circuit pump/status: OP02/GG02/RR001B; `refreshCircuits` | Raw observation; idle alone cannot distinguish competing meanings; nonzero=active is a candidate | `thermal.status.operation` / circuit withheld | T? | C, M | R | Do not translate pump state into cooling action or qualify RR001E using RR001B | Proven raw observation; Hypothesis interpretation | A0 | [Selector conflict][circuit-state]; [current mapping][circuit-map] |
| System energy-manager action/demand state: OP02/GG00/RR0048; `refreshSystem` | Current `state.energy_manager_state`; catalog medium confidence for 0–2; DHW inference remains hypothesis | `thermal.action.state` / system; candidate | T? | C, M, D | R | No percent demand or per-zone action inferred; falsified if system state cannot distinguish cooling or scope differs from fact key | Hypothesis action interpretation; no installation qualification | A0 | [Native state and transitions][manager]; [system projection][system-map]; [action field][thermal] |
| Circuit external demand: OP02/GG02/RR0019 | Native u16 external heat-source status; not listed in current circuits poll table | `thermal.demand.level` / circuit withheld | T? | C, D | R | Raw state is not 0–100% demand; no Boolean-to-percent conversion, no current publication claimed | Proven catalog state; Unknown quantified demand | A0 | [Native circuits][circuits]; [current poll inventory][circuit-map]; [demand field][thermal] |
| Zone operating mode and presets: OP02/GG03/RR0006 + RR000E, associated GG02/RR0006; `refreshState` / `refreshDiscovery` | Current derived mode/preset; family enum and quick-veto label conflicts remain | `thermal.mode.zone` / zone, `thermal.mode.system` / system withheld pending exact correspondence | T? | C, M | R | Manual is not automatically comfort; away/quick-veto must not silently become off/auto; cooling capability is not selected mode | Proven documented derivation; Hypothesis family interpretation | A0 | [Current mode derivation][zone-map]; [native enums][enums]; [family conflicts][families]; [fields][thermal] |
| Dedicated cooling mode: OP02/GG03/RR0001 and teleswitch RR000B | Catalog mode / unknown teleswitch interpretation; not current zones mapping inputs | `thermal.mode.system` / system withheld | T? | C, M | R | No evidence for zone-to-system collapse or installed cooling-mode projection | Proven catalog entry; Hypothesis enum applicability; Unknown teleswitch layout | A0 | [Native zones][zones]; [family enum candidates][families]; [current inputs][zone-map] |
| Observed room target: OP02/GG03/RR0022 primary, RR0014 fallback; `refreshState` | Current `[].config.target_temp_c`; manual heating target versus computed heating target distinguished in native catalog | `thermal.setpoint.temperature` / zone; candidate observation | T? | C, S, M | R | Preserve primary/fallback origin; not an accepted cooling command target; falsified if active-mode meaning or priority cannot be retained | Proven documented projection; Hypothesis binding | A0 | [Current target selection][zone-map]; [native target meanings][zones]; [field][thermal] |
| Cooling configured/manual room targets: OP02/GG03/RR0002, RR0023 | Catalog f32 °C conditioned on cooling; not listed as current zone target sources | `thermal.setpoint.temperature` / zone; candidate observation, acquisition unavailable in supplied current mapping | T? | C, S, M | R | Keep configured versus manual role distinct; falsified if mode/role cannot be preserved or evidence lacks a valid value | Proven catalog; Hypothesis future projection | A0 | [Native zones][zones]; [current target sources][zone-map]; [field][thermal] |
| Computed cooling effective target: OP02/GG03/RR0015 | Catalog computed f32 °C output; distinct from manual target and current RR0014 fallback | `thermal.setpoint.temperature` / zone; candidate observation | T? | C, S, M | R | Never offer this computed output as a setter; falsified if effective-target role is lost or cooling condition unproved | Proven catalog; Hypothesis projection | A0 | [Native zones][zones]; [current mapping][zone-map]; [field][thermal] |
| Zone valve, schedules and cutoff/action hints: GG03/RR0012, RR0019, RR001A, RR0027, RR0029, RR002A–RR002E, RR001B | Catalog flags/raw codes; manual-cooling width and special-function width unresolved in family matrix; not current zones mapping observations | `thermal.action.state`, `thermal.status.operation` / zone withheld | T? | C, M, D | R | An open valve, schedule, cooling-enabled flag or cutoff is not observed cooling action; retain each selector independently | Proven catalog observations; Hypothesis interpretations; Unknown composition | A0 | [Native zones][zones]; [family width conflicts][families]; [current mapping][zone-map] |
| Zone/circuit and sensor association: GG03/RR001C, RR0013; `refreshDiscovery` | Raw assignment is authoritative; profile-scoped observations; signed interpretation / wider labels candidates | No new thermal field; association prerequisite for all zone candidates | T? | C, S | R | No +1 renumbering; no physical thermostat inferred from a name or association byte | Proven raw scope; Hypothesis interpreted topology | A0 | [Native zone topology][zones]; [family binding candidates][families] |
| Circuit flow measurement / observed target: GG02/RR0008, RR0007 | Current flow measurement and flow setpoint; physical menu correspondence for setpoint still candidate | `thermal.measurement.water_temperature` / circuit candidate for RR0008; RR0007 room-setpoint substitution withheld | T? | C, S, M | R | Flow sensor is not room air or boiler return; RR0008 candidate falsified by unresolved circuit identity or invalid water measurement | Proven measurement source; Hypothesis projection / target role | A0 | [Native circuits][circuits]; [current poll table][circuit-map]; [water field][thermal] |
| Circuit dew point, calculated-flow conflict and cooling limits: GG02/RR0023, RR0020, RR0011, RR001F | Current dew-point and alleged calculated-flow fields; native RR0020 is one-byte raw status, contradicting older f32 mapping; cooling limits catalogued | No exact dew-point/limit role established; withhold air/water measurement or room target substitution | T? | C, S, M | R | Dew point is not measured room temperature; RR0020 is not a temperature; cooling limits do not establish thermostat bounds | Proven native raw conflict; Unknown suitable projection | A0 | [Native circuit rows][circuits]; [conflicting current table][circuit-map]; [field inventory][thermal] |
| System cooling controls / humidity ceiling: GG00/RR0006–RR0009, RR000E, RR0038, RR00DA–RR00DC, RR00F8, RR00FA | Catalog manual-cooling count, holiday/automatic-cooling flags, system-off switch, humidity ceiling, cooling threshold, candidate dates/control mode, abort and external-demand mode; current table includes system-off and humidity ceiling | No measured humidity, action or percent-demand substitution; withheld | T? | C, S, M, D, W | R; W withheld | Settings are not measurements or action; count units, system-off polarity, date/control-mode interpretations and command effects unresolved; CSV-only flags remain Hypothesis | Proven catalog entries where corroborated; Hypothesis candidate meanings; Unknown effects | A0 | [Native system rows][native-system]; [current system mapping][system-map]; [candidate writes][writes] |
| Additional system mode/schedule labels: OP02/GG00/RR007A–RR007D; zone override/enable context GG03/RR000A, RR000E, RR0011, RR001B | System labels and teleswitch meanings are candidates with unknown representation; special-function/enable widths and labels conflict across source sections | `thermal.mode.zone`, `thermal.mode.system`, `thermal.status.operation` withheld for these raw contexts | T? | C, M | R; W withheld | No collapse of schedule, screen mode, special-function state or enabled status into selected cooling; preserve raw distinctions | Hypothesis meanings; Unknown exact projection | A0 | [Native system][native-system]; [native zones][zones]; [enum conflicts][enums]; [family widths][families] |

## Remote sensor families and uninspected owner gaps

For eBUS remote rows the source owner remains `helianthus-ebusgateway`; the
native catalog is the inspected source path. No current remote-to-SemReg
publication implementation was supplied. OP06/GG09 and OP06/GG0A are distinct
from identically numbered OP02 local groups; retain opcode, group, instance,
register, profile and evidence in any proposed association.

| Family / driver owner and source path | Native evidence / qualification | SemReg field / service and disposition | Target type / cluster / feature / attribute | Mandatory gaps | Direction | Loss / withholding and concrete candidate falsifier | Grade | Admission | Exact public evidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| eBUS remote regulator family; native OP06/GG09/RR000F | Catalog f32 °C; NaN when no sensor; characterized slots only; publication unavailable | `thermal.measurement.air_temperature` / zone candidate | T? | C, S | R | No slot-to-zone inference; falsified by NaN, unqualified profile or absent sensor association | Proven catalog; Hypothesis projection | A0 | [Remote regulator rows][remote-regulator]; [field][thermal] |
| eBUS remote regulator family; native OP06/GG09/RR0007 | Catalog f32 %; NaN when no sensor; publication unavailable | `thermal.measurement.humidity_relative` / zone candidate | T? | C, S | R | No zero for missing sensor; falsified by invalid humidity or missing zone association | Proven catalog; Hypothesis projection | A0 | [Remote regulator rows][remote-regulator]; [field][thermal] |
| eBUS remote thermostat/sensor family; native OP06/GG0A/RR000F | Catalog corroborated f32 °C; NaN when no sensor; publication unavailable | `thermal.measurement.air_temperature` / zone candidate | T? | C, S | R | Sensor family does not prove thermostat composition; falsified by invalid sample or ambiguous sensor binding | Proven catalog; Hypothesis projection | A0 | [Remote sensor rows][remote-sensor]; [field][thermal] |
| eBUS remote thermostat/sensor family; native OP06/GG0A/RR0007 | Catalog corroborated f32 %; NaN when no sensor; publication unavailable | `thermal.measurement.humidity_relative` / zone candidate | T? | C, S | R | Keep separate native origin from zone/circuit mirrors; falsified by non-relative quantity or unqualified source | Proven catalog; Hypothesis projection | A0 | [Remote sensor rows][remote-sensor]; [field][thermal] |
| eBUS remote calibration offsets; OP06/GG09 or GG0A/RR000E | Catalog calibration setting, not current measured temperature; native write qualification absent | Air-temperature substitution withheld; no matching calibration operation established | T? | C, S, W | R; W withheld | Never add offset twice or present offset as room measurement; no setter inferred from FLAGS | Proven catalog role; Unknown command behavior | A0 | [Regulator offset][remote-regulator]; [sensor offset][remote-sensor]; [operations][thermal] |
| Gree CAN family cooling/action; driver owner and source path not supplied | Uninspected source-owner gap, qualification unknown | `thermal.action.state` vocabulary only; unavailable | T? | C, M | Unknown | Require public native contract and exact driver locator before any decoding or mapping | Unknown | A0 | Issue #10 requests this gap; no native public locator supplied; [vocabulary only][thermal] |
| Gree CAN family room temperature; driver owner and source path not supplied | Uninspected; no native temperature evidence | `thermal.measurement.air_temperature` vocabulary only; unavailable | T? | C, S | Unknown | No assumed unit, codec, sensor identity or current field | Unknown | A0 | Issue #10 scope; native locator missing; [vocabulary only][thermal] |
| Gree CAN family humidity; driver owner and source path not supplied | Uninspected; no evidence of humidity capability or absence | `thermal.measurement.humidity_relative` vocabulary only; unavailable | T? | C, S | Unknown | Missing source is not a negative target-support or hardware-support claim | Unknown | A0 | Issue #10 scope; native locator missing; [vocabulary only][thermal] |
| Gree CAN family mode/demand; driver owner and source path not supplied | Uninspected; observation/control and demand units unknown | `thermal.mode.system`, `thermal.demand.level` vocabulary only; unavailable | T? | C, M, D | Unknown | No assumed mode enumeration, percent demand or device composition | Unknown | A0 | Issue #10 scope; native locator missing; [vocabulary only][thermal] |
| Gree CAN family set-mode/setpoint; driver owner and source path not supplied | Uninspected native operations and effects | `thermal.operation.set_mode`, `.set_temperature` vocabulary only; unavailable | T? | C, W | W withheld | Read facts cannot establish commands; need public contract, limits, acceptance and effect evidence | Unknown | A0 | Issue #10 scope; native locator missing; [operations only][thermal] |

## Command inventory and transition review

| Family / owner and source path | Native evidence / qualification | SemReg field / service and disposition | Target type / cluster / feature / attribute | Mandatory gaps | Direction | Loss / withholding | Grade | Admission | Exact public evidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| eBUS controller families; `semantic_vaillant.go` path reported by mapping; GG03/RR0006, RR0001 and GG00 cooling/system flags | Read/derived modes exist; candidate write encodings have no established ACK, response correlation or persistence | `thermal.operation.set_mode` / `thermal.capability.set_mode` / system, argument `thermal.mode.system`, effect `thermal.effect.set_mode`; withheld | T? | C, M, W | W withheld | No reverse mapping from observed cooling to mode selection; no qualified cooling setter established | Hypothesis native commands; Proven vocabulary; Unknown implementation | A0 | [Native mode rows][zones]; [candidate write boundary][writes]; [operation table][thermal] |
| eBUS controller families; same owner; GG03/RR0007, RR0022, RR0008 and cooling RR0002/RR0023; effective RR0014/RR0015 separate | Family-specific room-control encodings and quick-veto sequence candidates; computed targets are observations | `thermal.operation.set_temperature` / `thermal.capability.set_temperature` / zone, argument `thermal.setpoint.temperature`, effect `thermal.effect.set_temperature`; withheld | T? | C, S, M, W | W withheld | Accepted native limits, effect, units for sequence steps and partial completion unresolved; a generic setter cannot select manual/cooling/effective roles by analogy | Hypothesis native commands; Proven vocabulary; Unknown implementation | A0 | [Native target rows][zones]; [write qualification][writes]; [operation table][thermal] |

The current documented energy-manager transitions describe demand-related
states. Circuit transitions remain Hypothesis because RR001E itself lacks
correlation. This inventory proposes no state machine, command interface or
native transition. A reported cooling action must never trigger a set-mode or
setpoint operation. Neither SemReg operation availability nor a completed
synthetic case would establish native write acceptance.

Manual synthetic review cases (reasoned dispositions, not executed tests):

| Case | Proposed disposition and source review |
| --- | --- |
| Qualified zone room value 21 °C and relative humidity 45% | Candidate scalar facts only with matched zone/sensor identity; [native zone rows][zones] and [SemReg units][thermal] support vocabulary, T? and A0 persist |
| Remote sensor NaN, or empty native reply | Explicit unavailable/withheld disposition; never zero, never a new live sample; [remote rows][remote-sensor] distinguish missing sensor |
| Circuit RR001E raw 2; RR001B raw 0 | Preserve both native observations; cooling interpretation remains Hypothesis under [selector conflict][circuit-state]; no command follows |
| Circuit RR0020 raw byte 00 | Retain raw status; reject calculated-temperature interpretation under [native catalog][circuits] despite older [mapping table][circuit-map] |
| Cooling enabled while circuit is idle | Keep capability and action separate under [native circuits][circuits]; do not synthesize cool mode or cooling demand |
| External demand raw 1 | Preserve raw status; do not invent 100% for `thermal.demand.level`, whose [quantity definition][thermal] is percent |
| Mode code 3 or conflicting quick-veto label | Preserve family/raw value and explicit loss; do not coerce to a supported SemReg symbol; [enum conflicts][enums] remain unresolved |
| Effective cooling target exists, setter requested | Withhold setter: computed observation and operation acceptance are independent; [zone roles][zones] and [write boundary][writes] apply |
| Temporary partial acquisition gap after a valid observation | Preserve native evidence and partial last-known-good state under the source lifecycle policy; do not promote retained values to fresh, clear unaffected fields or invent fallback precedence |

No MCP or GraphQL behavior changes. Any later consumer must preserve the same
source identity, raw evidence, qualification, retention and explicit loss across
both surfaces, with deterministic envelopes. The existing [mapping's unavailable
observation boundary][mapping] does not authorize a new fallback on stale,
conflicting or unqualified data. No credential, principal or permission checks
are introduced; authentication/authorization remain 0.11 subjects, with no new
authorization gates before 0.9.

## Locked target evidence and remaining reconciliation

The [lock][lock] explicitly records data-model version `1.7`, Electrical Sensor
`0x0510` revision 2, Power Topology `0x009C` revision 2, Electrical Power
Measurement `0x0090` revision 3 and optional ActiveCurrent `0x0005`.
These are **lock metadata**, not an inspected complete schema and not an
air-conditioning target assignment. No thermal target IDs or revisions are
claimed. The [SDK baseline][baseline] describes an Electrical Sensor sample
fixture; its result cannot supply thermostat or humidity-sensor composition.
The full locked connectedhomeip data model was unavailable for this inventory.

Later reconciliation must use that exact locked checkout, without substituting
another SDK or inferring mandatory requirements from names:

1. Inspect `data_model/1.7/spec_sha`, `data_model/1.7/spec_tag`,
   `data_model/1.7/device_types/ElectricalSensor.xml` and
   `data_model/1.7/clusters/ElectricalPowerMeasurement.xml` identified by the lock.
   These paths are lock evidence, not domain coverage.
2. Search `data_model/1.7/device_types` for `Thermostat`, `Temperature`,
   `Humidity`, `AirCondition` and `Room`; search `data_model/1.7/clusters` for
   `Thermostat`, `TemperatureMeasurement`, `RelativeHumidityMeasurement`,
   `SystemMode`, `RunningState`, `Cooling` and `Demand`. These are exact search
   terms and directories for finding candidate files; their existence and
   applicability remain unverified. Record actual matched file paths before
   proposing target cells.
3. For each actual match, inspect mandatory/optional features, attributes,
   commands, enum values, bounds, units, scaling, nullability and revisions;
   follow referenced device-composition requirements and descriptor topology.
   Inspect corresponding generated files under
   `zzz_generated/app-common/clusters` and server implementations under
   `src/app/clusters`, then relevant controller tests. No target absence claim
   follows from the small lock metadata surface.
4. Obtain the semantic mapping artifact at the path recorded in `mapping.path`
   in the lock and its source-owner approval; it was not supplied for inspection.
   Resolve each candidate falsifier and source qualification separately. Extend
   accepted evidence through the owning repositories before any profile proposal.

Cooling energy counters, ventilation control, DHW, boiler behavior, schedules as
commands and unrelated thermal capabilities do not acquire target mappings here.
The inspected sources also do not establish a complete current-driver catalog:
Gree owner discovery and current implementation inspection remain explicit gaps.
Full target inventory, mandatory composition, locked-corpus verification, matching
build evidence, independent review and any separately directed hardware acceptance
remain parent work. This document cannot establish runtime admission, release
M42/M43 or close #8.

## Public evidence locators

Links use public `blob/main` file destinations. They identify the reviewed
source paths; a moving branch link is not an immutable evidence pin. No network
verification was performed. Native selectors and section names above narrow each
claim within these files; line locators reflect the inspected source views.

[parent]: https://github.com/Project-Helianthus/helianthus-matter-binding/issues/8
[lock]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/sdk/matter-sdk.lock.json
[baseline]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/SDK_BASELINE.md
[admission]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/runtime/production_readiness.py#L166-L180
[thermal]: https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/thermal/definitions.go#L37-L82
[mapping]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/b524-semantic-mapping.md
[system-map]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/b524-semantic-mapping.md#L17-L41
[circuit-map]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/b524-semantic-mapping.md#L153-L217
[zone-map]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/architecture/b524-semantic-mapping.md#L219-L236
[native-system]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L292-L532
[circuits]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L572-L633
[zones]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L637-L699
[remote-regulator]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L958-L985
[remote-sensor]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L1008-L1029
[enums]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L1185-L1214
[circuit-state]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L1252-L1286
[manager]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L1384-L1416
[families]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L1444-L1494
[writes]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md#L1496-L1522
