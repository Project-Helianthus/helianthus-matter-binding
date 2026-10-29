# Thermostat families and heating/cooling roles

This is the bounded source inventory for [issue #42](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/42),
a partial slice of [#8](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/8)
and [#23](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/23)
under Project #2. It separates thermostat-related observations and candidate
operations; it does not choose a minimal endpoint or complete either parent.
Nothing here is stable before Helianthus 1.0.


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
| R01–R05 room/remote temperature, humidity and sensor association | Unknown sensor target identities and features; source evidence unchanged, unavailable remote values never become zero or fresh. |
| R06–R26 manual/effective/setback/cooling/holiday/override, mode/action and capability roles; O01–O06 operations | Unknown prior identity → Thermostat revision 11 source presence. Exact features, bounds, loss, native write effect and mandatory closure remain Unknown; observations and operations stay separate. |

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

## Scope and evidence

The inspected public native contract is the B524 register catalog [N], followed
by the thermal semantic definitions [S]. The supplied heating inventory [H]
provides reconciliation references to driver behavior; the driver itself was
not supplied for direct inspection. Accordingly, driver behavior cited through
[H] is secondary evidence, not independently verified acquisition or operation
support. This page adds no protocol facts or accepted semantic mappings.

The source families are Vaillant VRC720-family and VRC700-family controllers,
and their remote-room sensor/control families. Family labels do not qualify
individual profiles, codecs, presence, or installations. Other thermostat driver
families have no supplied public native/driver evidence here: their membership,
roles and support remain **Unknown**, not unsupported. Circuit/module facts are
included only where they constrain room heating/cooling roles. General system,
DHW, hydraulic, energy, schedule and lifecycle inventories remain in [H] and the
existing general inventories; this page does not reproduce X07–X13.

**Proven** below means the cited document or definition states that contract
fact. **Hypothesis** marks candidate meanings or joins. **Unknown** marks absent
or conflicting evidence. None means installation qualification. **Q?** in every
row means native qualification Unknown; no accepted thermostat binding proof is
supplied. Catalog categories, historical read/write labels and FLAGS do not
establish writability: [N] “Table Legend” withdraws those labels as public B524
semantics. Read rows inventory observations, never permission to operate.

**T?** in every row means blocked pending target evidence: candidate target role
only; device type/revision, cluster/revision, feature, attribute/command/event,
mandatory dependencies, units/precision/nullability and accepted public semantic
target mapping are Unknown. The retained legacy lock selects Matter 1.7; the adopted complete Matter
1.6.1 normative corpus and compatible target mapping remain missing [C]. No
normative thermostat identifiers, required feature list or representability
claim can be derived from similarly named source fields. Every T? needs that
adopted-corpus inspection and complete mandatory-feature closure. Unknown-filled
rows record gaps and never establish completeness or a target exclusion.

## Ledger identity and direction

Apply [C]'s join key `(domain, native owner/profile, capability/fact, direction,
target component)`. The domain here is Thermal. Each row ID below is a review
locator, not a source identity or endpoint ID. Its canonical source reference is
[N] plus the family-qualified profile, opcode, group, instance and register,
with the measurement/service role retained. Exact qualified profile and service
binding remain unresolved. `Z` means local `OP=0x02, GG=0x03, II=<native zone>`;
`Cct` means local `OP=0x02, GG=0x02, II=<native circuit>`; `Rem` means remote
`OP=0x06, GG=0x0A, II=<native remote>`. Each RR is hexadecimal. An operation
references the same fact where appropriate but has a different direction key.
In operation rows, Z locates the read catalog entry; it does not assert the
transmission opcode or response contract of a write. Candidate write evidence
is limited to [N] “Candidate write encodings” and the cited zone row notes.
No local GG0A/OP02 selector is an alias for Rem.

Read direction is **native-to-Matter observation candidate**; operation direction
is **Matter-to-native operation candidate**, with no implemented route asserted.
An absent operation row is an evidence gap, not an API. Preserve native owner,
profile, source/service identity, sensor assignment, opcode, instance, register,
source epoch and driver generation; display names and equal values cannot join
facts. Replacing any join-key field requires a separately reviewed join [C].

### Room measurements and sensor identity — read direction

| Row / source reference in [N] | Source role / unit | Confidence; qualification | Semantic evidence [S]; driver evidence | Target evidence | Loss / withholding | Falsifier |
| --- | --- | --- | --- | --- | --- | --- |
| R01 — Z RR000F, “Zones” | Room air observation; f32 °C | Proven catalog role; Q? | `thermal.measurement.air_temperature` exists; [H] zone room row cites `refreshState`, secondary only | T? room measurement | Not outdoor/water temperature, target, demand or writable control | Room reading appears as outdoor temperature or enables a write |
| R02 — Z RR0028, “Zones” | Room humidity; f32 % | Proven catalog role; Q? | `thermal.measurement.humidity_relative` exists; [H] humidity row cites slow-config acquisition, secondary only | T? humidity measurement | No humidity control or humidity setpoint from a sensor reading | Humidity reading enables dehumidification or a set-humidity operation |
| R03 — Rem RR000F, “Remote Control Thermostats” | Remote room air observation; f32 °C; NaN if no sensor | Proven catalog role; Q? | Air-temperature field exists; [H] remote fallback path secondary only | T? remote/zone measurement relationship | Remote observation is distinct from zone R01 until assignment/provenance is qualified; NaN is not zero | Unassigned remote overwrites a zone or NaN becomes a valid temperature |
| R04 — Rem RR0007, same section | Remote relative humidity; f32 %; NaN if no sensor | Proven catalog role; Q? | Humidity field exists; [H] remote fallback secondary only | T? humidity measurement | No automatic alias to R02 or circuit RR0022; sensor and service scopes differ | Same-unit readings from different sensors share one canonical fact without proof |
| R05 — Z RR0013/RR001C; Rem RR0025/RR0001, “Zones”, “Remote Control Thermostats” | Sensor binding enum / circuit assignment raw / remote zone assignment count / connection bool; no physical unit | Proven distinct catalog entries; candidate label/range interpretations Hypothesis; Q? | Zone/circuit service dimensions exist; exact driver binding uninspected; [H] identity row secondary only | T? relationship metadata | Keep all selectors distinct; no universal +1 remap, zone=circuit=remote identity, or physical absence from disconnected status | Disconnected remote erases retained evidence, or an index alone establishes physical topology |

### Setpoint roles — read direction

| Row / source reference in [N] | Source role / unit | Confidence; qualification | Semantic evidence [S]; driver evidence | Target evidence | Loss / withholding | Falsifier |
| --- | --- | --- | --- | --- | --- | --- |
| R06 — Z RR0022, “Zones” | Heating manual room target; f32 °C catalog | Proven catalog role; Q? | `thermal.setpoint.temperature` exists; [H] reports primary selector in slow config, secondary only | T? manual heating target observation | Driver fallback must retain role; not the effective/computed R08 | Primary and fallback values silently share one manual-role claim |
| R07 — Z RR0007, “Zones” | Heating comfort candidate; codec and °C candidate | Hypothesis; Q? | Temperature setpoint field alone supplies no comfort binding; driver evidence absent here | T? comfort target observation | Do not alias RR0007 to manual RR0022 across families | Same label or number collapses family-specific comfort/manual selectors |
| R21 — Z RR0009, “Zones” | Heating setback target; f32 °C catalog | Proven catalog role; Q? | Temperature setpoint field supplies no preset; driver evidence absent here | T? setback target observation | Setback is not automatically semantic eco | Setback target alone creates an eco preset |
| R08 — Z RR0014, “Zones” | Effective heating room target; f32 °C, computed output | Proven catalog role; Q? | Setpoint field exists; [H] reports fallback use for R06, secondary only | T? computed heating target observation | Preserve computed/manual distinction even when equal; never user-settable from this evidence | Effective target is used as proof of a native set-temperature route |
| R09 — Z RR0002, “Zones” | Cooling room target; f32 °C catalog; cooling-enabled gate | Proven catalog role; Q? | Setpoint field exists without a cooling-role mapping; driver evidence absent here | T? cooling target observation | Not heating target or cooling capability proof by itself | Presence/value alone enables a cooling feature |
| R10 — Z RR0023, “Zones” | Manual cooling room target; f32 °C catalog; cooling gate | Proven catalog role; Q? | Setpoint field exists; driver evidence absent here | T? manual cooling target observation | Not an alias to R09 or effective R11 without precedence evidence | All cooling targets collapse to one writable value |
| R11 — Z RR0015, “Zones” | Effective cooling room target; f32 °C, computed output; cooling gate | Proven catalog role; Q? | Setpoint field exists; driver evidence absent here | T? computed cooling target observation | Computation is distinct from manual target and write support | Read of R11 is treated as command capability |
| R12 — Z RR0005, “Zones”, “Open questions” | Holiday room target; catalog f32 °C, unit not established in candidate contract | Proven catalog entry; unit/accepted interpretation Unknown; Q? | Scalar setpoint field cannot represent holiday interval/lifecycle; driver evidence only secondary [H] | T? holiday target/context | Do not silently choose catalog °C over unresolved unit evidence or infer away preset | Holiday target loses interval or conflicting unit qualification |
| R13 — Z RR0008, same sections | Quick-veto target; catalog f32 °C, unit not established | Proven catalog entry; unit and accepted interpretation Unknown; Q? | Setpoint field has no override lifecycle; driver evidence secondary [H] | T? override target observation | Retain target separately from duration and special-function state | Scalar target alone becomes a preset |
| R22 — Z RR0026, same sections | Quick-veto duration; catalog f32 hrs, unit not established | Proven catalog entry; unit and accepted interpretation Unknown; Q? | No duration definition in supplied [S]; driver evidence secondary [H] | T? override duration observation | Preserve duration qualification and lifecycle; no assumed hours conversion | Unqualified duration yields an expiry or an expired override stays active |

### Modes, action and capabilities — read direction

| Row / source reference in [N] | Source role / unit | Confidence; qualification | Semantic evidence [S]; driver evidence | Target evidence | Loss / withholding | Falsifier |
| --- | --- | --- | --- | --- | --- | --- |
| R14 — Z RR0006, “Zones”, “Enumerations that differ by family” | Heating mode; u16 enum, no unit | Proven catalog entry; family labels Hypothesis, range conflict Unknown; Q? | Zone symbols auto/comfort/eco/off differ from native labels; [H] derived mode path secondary only | T? heating mode observation | VRC720-family TimeControlled/Manual versus VRC700-family Auto/Day/Setback; value 3 conflicts with catalog 0..2; manual is not comfort | Native code is mapped without family or unsupported value defaults to a qualified mode |
| R15 — Z RR0001, same sections | Cooling mode; u16 enum; cooling gate | Proven catalog role; family labels Hypothesis; Q? | `thermal.mode.system` has cool, but different system scope; no accepted zone equivalence; driver uninspected | T? cooling mode observation | Cooling mode, heating mode and cooling capability remain separate | A system cool symbol is attached to a zone without a scoped mapping |
| R16 — Z RR000E, “sfmode”, “Open questions” | Special-function control observation; u8 enum; no unit | Proven entry/conflict; labels Hypothesis; Q? | No preset field in supplied [S]; [H] derivation secondary only | T? special-function control observation | Value 3 quick-veto write conflicts with away read label; do not infer presets from names | Raw code is discarded and away/quick-veto is asserted as qualified |
| R23 — Z RR001B, “Enumerations that differ by family”, “Open questions” | Special-function status; catalog u16, candidate width one or two bytes; no unit | Proven entry/conflict; labels Hypothesis, width Unknown; Q? | No preset field in supplied [S]; driver evidence absent here | T? special-function status observation | Distinct selector/codebook from control R16; preserve unresolved width and family labels | Status and control are decoded interchangeably |
| R17 — Z RR0012, “Zones” | Valve position u16 bool; no unit | Proven catalog role; Q? | `thermal.action.state` exists; [H] zone action derivation secondary only | T? valve/action relationship | Open valve is not delivered heating; keep raw valve state | Open valve proves actual heating/cooling delivery |
| R24 — Cct RR001E, “Circuit State Enum” | Circuit action candidate; u16 enum; no unit | Proven entry; action labels Hypothesis; Q? | Action field exists; driver evidence secondary [H] | T? circuit action observation | Keep circuit scope and raw state; distinguish action from demand/capability | Guessed circuit state proves delivered heating/cooling at a zone |
| R18 — Cct RR0002 with Z RR001C, “Circuits”, “mctype” | Heating capability context: circuit type u16 enum and native assignment raw; no unit | Proven contract dependency; family labels/applicability Hypothesis; Q? | Read-zone/read-circuit capabilities are metadata, not heating hardware proof; driver evidence secondary [H] | T? heating capability | Heating requires supporting circuit type/qualified assignment; exclude virtual native-water identity from physical circuit inference | Room temperature, active index or virtual water slot alone proves a heating thermostat |
| R19 — Cct RR0006, “Gate Conditions”, “mctype” | Cooling-enabled u8 bool; no unit | Proven catalog gate; Q? | No cooling-hardware capability supplied by [S]; driver evidence secondary [H] | T? cooling capability | Circuit cooling gate is independent of circuit type; false/unknown differ | Heating type implies cooling, or an unavailable flag becomes false/current |
| R25 — Z RR0029, “Zones” | Zone cooling-enabled flag; catalog u16, decode unspecified | Proven entry; full meaning Unknown; Q? | No accepted flag mapping in supplied [S]; driver evidence absent here | T? zone cooling flag | Not interchangeable with circuit gate R19 or active state R26 | Zone flag silently substitutes for qualified circuit cooling capability |
| R26 — Z RR0027, “Zones”, “Open questions” | Manual-cooling-active state; catalog u16, candidate width one or two bytes; unit unspecified | Proven entry/conflict; interpretation and width Unknown; Q? | No accepted manual-cooling state mapping in supplied [S]; driver evidence absent here | T? manual cooling active-state observation | Active-state label does not establish persistent cooling capability | Active state enables cooling capability despite missing R19 evidence |
| R20 — Cct RR0015/RR0006, “Gate Conditions”; Z RR0028 | Room-control mode u16 enum, cooling bool, humidity % | Proven documented dew-point prerequisites; installation applicability Unknown; Q? | No dew-point control definition in supplied [S]; driver evidence absent here | T? safety-related cooling context | Cooling enabled AND room-control mode not off are documented gates, not sufficient dew-point protection or safe operation proof | Humidity alone establishes safe cooling or bypasses missing room-control evidence |

### Candidate operations — separate operation direction

These rows inventory documented candidates or explicit absence. They do not
supply accepted native effects, limits, readback, authorization or a live route.
Configuration categories and technical write encodings cannot establish these.

| Row / source reference in [N] | Source operation role / unit | Confidence; qualification | Semantic evidence [S]; driver evidence | Target evidence | Loss / withholding | Falsifier |
| --- | --- | --- | --- | --- | --- | --- |
| O01 — Z RR0022 / RR0007, “Zones” | Manual room target write candidate; VRC720-family RR0022 versus VRC700-family RR0007; float32 LE, °C catalog/candidate respectively | Hypothesis encoding/applicability; acceptance/effect Unknown; Q? | `thermal.operation.set_temperature` requires zone set-temperature capability; metadata does not prove either native route; driver uninspected | T? temperature command | Preserve family selector, role, units, constraints and effect evidence; no operation from R01/R08 | A measured or effective temperature, or family-mismatched selector, enables this command |
| O02 — Z RR0006, “Zones” | Heating mode write candidate; u16 LE enum, candidate codes 0 and 2; no unit | Hypothesis; acceptance and zone/circuit interpretation Unknown; Q? | `thermal.operation.set_mode` is system-scoped, not a zone-mode operation; driver uninspected | T? mode command | No system/zone coercion; read labels and write acceptance remain separate | System semantic operation is routed to a zone using equal-looking mode labels |
| O03 — Z RR0008/RR0026/RR000E, “Zones”, “sfmode”, “Open questions” | Quick-veto target/duration and control candidates; float32/float32/u8; target/duration units not established | Hypothesis sequence/labels; accepted effect Unknown; Q? | No matching override/preset operation in supplied [S]; driver uninspected | T? override command | Family-specific sequence and cancel semantics unresolved; do not infer an away/eco preset | A scalar target or write acknowledgement is accepted as completed override effect |
| O04 — Z RR0005, “Zones”, “Open questions” | Holiday target operation unresolved; candidate °C, no qualified effect contract | Unknown operation support; Q? | No matching holiday operation in supplied [S]; driver evidence absent here | T? holiday command | R12 and configuration classification are not operation evidence; interval/unit closure absent | Holiday reading alone enables a holiday command |
| O05 — Z RR0001/RR0002/RR0023, “Zones” | Cooling mode/target operations absent from supplied accepted contract; read units enum/°C do not qualify writes | Unknown operation support; Q? | Generic semantic definitions do not establish cooling operation roles; driver evidence absent here | T? cooling commands | Each future operation needs its own selector/role and effect evidence; no automatic reverse mapping from R09/R10/R15 | Cooling observation or heating write support enables a cooling command |
| O06 — Z RR0028; Rem RR0007/RR000F, measurement rows above | Humidity/measurement operations absent; observed %/°C are not command units | Unknown operation support; Q? | No set-humidity operation in supplied [S]; temperature operation does not turn measurement into control | T? measurement/control separation | Sensor/control family name supplies no humidity, calibration or room-control operation | Remote measurement is published as writable merely because the family has a UI |

## Reconciliation and review cases

These are role refinements of [H]'s zone/remote rows, not additional source facts
for general inventory totals. Canonical references retain the full native key.
R01 and R03, and R02 and R04, are possible assigned-sensor relationships, not
proven aliases. R06/R08 are a reported driver fallback conflict, not two names
for one manual target. R09/R10/R11 retain separate cooling roles. R05's mapping
selectors remain distinct from each other and from remote assignment. R18/R19
are independent capability claims. Read/operation joins count separately, while
repeated selectors count once as canonical source facts after alias review [C].
Unresolved profiles, aliases, driver acquisition/effect evidence, other-family
coverage and all target joins remain in the parent gap register; no coverage
total, full thermostat roster or completed target sweep is asserted.

The following invented examples exercise review decisions only; they are not
protocol observations, automated tests or installation evidence:

| Synthetic case | Required disposition / falsifier |
| --- | --- |
| Family A reports manual and effective targets both as 21 °C | Keep R06/R08 roles and direction separate. Equality must not create a write route; merging them falsifies the ledger rule. |
| Family B has a qualified heating role but no cooling evidence | Heating and cooling remain separate joins. Cooling is Unknown, not false and not inferred from heating; enabling cooling falsifies the rule. |
| Assigned remote becomes disconnected while a zone read succeeds | Preserve native remote evidence and partial last-known-good state with its original freshness; keep the fresh zone read distinct. Erasure or renewal of retained data falsifies the rule. |
| Candidate manual operation receives an acknowledgement without qualified effect/readback | Preserve the acknowledgement as native evidence; operation completion remains Unknown. Reporting applied target/preset falsifies the rule. |

For each row, review source selector/meaning, family and service identity,
direction, units, evidence confidence separately from qualification, semantic
scope, target gap, loss and falsifier. Reopen affected joins when source evidence
changes. Missing evidence cannot transition a row to representable. This review
completes only the named source inventory slice; [C]'s driver roster,
source-to-target and target-to-source sweeps, alias resolution, accepted public
semantic mappings and mandatory closure still govern parent acceptance.

## Public locators and unchanged boundary

- [N] [Public B524 register contract](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md): sections and native keys identified per row; “Table Legend” and “Open questions” qualify all candidate use.
- [S] [Thermal definitions](https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/thermal/definitions.go): `fields`, `services`, `capabilities`, `operations`; definitions establish semantic vocabulary, not native mappings.
- [H] [Heating source inventory](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/heating.md): “Zones and remote-room observations”, “Circuits”, and “Public evidence locators”; driver references here were reviewed only through that supplied inventory.
- [C] [Inventory completeness contract](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md): “Proposed reconciliation record”, “Mandatory-feature closure and shared facts”, and “Later two-way sweep and inventory checkpoint”.

Public main links locate the reviewed supplied views; they are not immutable
qualification evidence. Offline review checks repository/path and cited content,
not network reachability or current main freshness. Missing adopted target
contents cannot be replaced with current upstream material or SDK fixtures.

No SDK selection, accepted profile, allocation, runtime interface, live test or
upstream action is added. Fail-closed admission remains unchanged. Native
ownership, detached data, deterministic envelopes, eBUS MCP compatibility,
GraphQL parity and partial last-known-good state remain required. Qualification,
freshness, build evidence, runtime acceptance, interoperability, certification
and physical acceptance are separate from this document review. No transport or
hardware/live-fabric smoke PASS is implied. No authorization gate is introduced
before 0.9; any new authorization roadmap work belongs to 0.11. Independent
review, applicable committed-head checks and remaining parent acceptance remain
outstanding.
