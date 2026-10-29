# Native CAN source sweep and Matter target gaps

This bounded source ledger advances [issue #8][issue] as a partial slice. It
replaces the CAN-source-absence gap in [inventory completeness][I], without
repeating the [pump](pumps.md), [thermostat](thermostats.md) or
[B524](b524-device-types.md) inventories. Nothing is stable before Helianthus
1.0. No row releases M42/M43, establishes inventory completeness, qualifies an
installation, or permits an endpoint.


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
| O25-V1–V4 and O25-C1–C4 battery/limits/status/electrical/cell/counter/diagnostic searches | Changed Sensor/Topology context to revision 1; unchanged EPM revision 3. EEM revision 2 / DEM revision 4 source presence; battery, DC, limits and diagnostic joins Unknown. |
| O25-G1–G7 thermostat/air-conditioner/setpoint/mode/status/sensor/fan/HVAC searches | Unknown prior Thermostat identity → revision 11 source presence; exact sensor/fan/air-conditioner targets and Gree property semantics Unknown. Raw/candidate/map distinctions unchanged. |
| Reverse thermostat/mode/setpoint/fan and every other representable family | Unknown separate reverse joins; in-memory encoding is no effect evidence. All 15 source groups retained, no native sweep repeated. |

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

## Evidence, identity and direction

The public native contracts [V], [C] and [G] are the authority for native facts;
the inspected canbusreg source [VS], [CS], [GS] records implementation coverage.
The public storage [S] and thermal [H] packs supply vocabulary, not an accepted
CAN-to-pack binding. The public semantic target contract [T] has one narrow
EVSE-current transform; it supplies no accepted Growatt or Gree target join.
Its current public view does not replace the exact locked mapping contents.

**Proven** means a cited public contract or inspected source establishes the
bounded statement. **Hypothesis** means a proposed semantic correspondence or
target search lead. **Unknown** means unavailable or unresolved evidence.
These grades are independent of native qualification and inventory closure.
All rows below are **blocked pending evidence**. None is a negative proof of
Matter representability.

Growatt rows belong to the storage domain; electrical-metering searches are
cross-domain candidates, not extra facts. Gree rows are thermal/HVAC source
candidates; raw diagnostics alone do not establish another domain membership.

Each O25 row expands into separate fact joins under [I]'s key: domain, public
native owner/profile, fact and service scope, direction, target component.
Growatt's exact profile is `growatt.bms.low_voltage.can.v1_04`; the common
projection is a separately caller-selected interpretation, not a revision
profile. Gree's exact candidate is `gree.vrf.canbus.candidate.v1`, with explicit
map selection where applicable. Retain source/interface identity, frame ID,
geometry, sequence and observation context, and Gree's `opaque7`, opcode,
coordinate/cell and selected map. Equal values or units do not merge facts.
Shared Growatt fields are alternative interpretations of the same evidence
only when their source identity and scope are proven equal; they are not extra
measurements or automatic aliases to RS-485 facts in [storage](storage.md).

**R** in every ledger row means native-to-Matter observation **candidate**,
not current Matter publication. **QG** means the V1.04 qualification card [VQ]
records contract-evidenced offline candidate readiness with physical
qualification and live trials false. **QC** means common-projection native
qualification Unknown; it has no revision/equipment admission claim. **QR**
means Gree's card [GQ] records `QUALIFICATION_TEST_READY` for its provider-local
offline check, physical qualification false. No replay was executed here.

**T?** in every row means the candidate device type/revision, cluster/revision,
feature selection, attribute/command/event correspondence and mandatory closure
are **Unknown**. Target names are Hypothesis search terms only. The full adopted
Matter 1.6.1 normative device/cluster corpus is unavailable; the supplied
graded report establishes only the bounded source identities in the baseline. No IDs or
mandatory-feature lists are inferred from names. Exact locked target inspection,
accepted public semantic mapping, source qualification, composition and matching
build/reporting evidence remain required separately for every join. Qualification
against native Go go-fabric on the adopted Matter 1.6.1 model is transport-agnostic and unverified;
there is no C++ sidecar. The [reviewed lock][L] has no accepted composition-profile
digest, so production admission remains denied regardless of these source rows.

## Explicitly selected Growatt V1.04

[V] requires receive-only standard, non-RTR eight-byte frames. Admission needs
valid `0x311`, `0x312`, `0x313` in the most recent 16 accepted observations of
one interface; malformed listed frames reset that interface's window. [VS]
additionally checks raw DLC eight, pack/cell counts 1–254 and SOC at most 100.
[Registry][RS] rejects competing profile matches. Selection, a single ID,
firmware or an observed keepalive never qualifies another revision or family.

| Row / native group and source | Direction / native units and retained evidence | Grade / native qualification / source sweep | Semantic candidate and target search / exact remaining gap |
| --- | --- | --- | --- |
| O25-V1 — `0x311` limits; [V] Frame Map, [VS] `Limits` | R; big-endian unsigned charge voltage 0.1 V, charge/discharge current limits 0.1 A; full `RawStatus` and `LimitsEvidence` retained | Proven contract and projected members; QG; group inspected | [S] interface charge/discharge-limit vocabulary is Hypothesis pending exact role, scaling and qualified binding. T? storage/energy-management limit observation; limits do not establish a set-limit command or an electrical measurement |
| O25-V2 — `0x312` status/metadata; [V] Status and Errors, [VS] `Status` | R; two protection bytes, two warning bytes, pack count, manufacturer bytes and total-cell count; raw/count units, `StatusEvidence` retained | Proven raw projection; named semantic health interpretation Unknown; QG; group inspected | [S] protection/warning candidates need code correspondence and topology. T? battery status/diagnostics; a clear bit proves no healthy state, counts do not identify physical members and manufacturer bytes are not equipment identity |
| O25-V3 — `0x313` measurements; [V] Frame Map, [VS] `Measurements` | R; big-endian signed voltage 0.01 V, signed current 0.1 A, maximum-cell temperature 0.1 °C; SOC percent, SOH low seven bits plus validity bit; `MeasurementsEvidence` retained | Proven members/encoding; semantic joins Hypothesis; QG; group inspected | [S] pack voltage/current, SOC/SOH and cell-temperature candidates. T? electrical sensor, battery status or temperature sensor; DC/sign/reference, pack-or-system scope, cell identity and temperature maximum role need evidence. SOH can be 127 in [VQ], outside [S]'s 0–100 percent bound: no blind promotion, even when the native validity bit is set |
| O25-V4 — `0x314`, optional `0x315`–`0x318`, `0x319`–`0x321`; [V] Frame Map and Optional Frames, [VS] aggregate switch | R; documented capacity in 10 mAh, voltage spread/extrema/cell values in mV, cycle count; request/type flags and indices, build/update metadata remain native. No typed members for these groups in [VS]; retain raw observations, never publish identifying/build-time values here | Proven contract listing and absence from this projection; decoded availability Unknown; QG applies only to its three-frame subset; wider source sweep remains open | T? battery capacity/cell sensor/counter/diagnostics searches; separate source decoding, identity, validity and counter lifecycle evidence required before mapping. Optional absence is unavailable, not zero or fault. Native request/enable bits prove no Matter command or supported write |

[VS] retains three source `Evidence` values, each containing frame, interface,
sequence, monotonic context and exact raw record [RS]. It does not project every
frame documented by [V]. Raw-only frames must remain available through the
native observation owner; this ledger does not assert a new aggregate store or
Gateway API. Malformed transport observations cannot be used to erase valid
retained native values; the native MCP boundary [V] distinguishes transport
errors from clearing an admitted snapshot. Window admission and retained
last-known-good publication are separate contracts.

## Caller-selected common shared fields

[C] and [CS] permit source-local shared fields without selecting V1.04 or V1.08.
Unknown or conflicting revision withholds divergent fields while preserving the
shared values and raw evidence. V1.07 lacks a full layout; firmware cannot select
a revision. Each result retains its single `RawEvidence`; this projector itself
is stateless and supplies no multi-frame last-known-good store.

| Row / native group and source | Direction / native units and retained evidence | Grade / native qualification / source sweep | Semantic candidate and target search / exact remaining gap |
| --- | --- | --- | --- |
| O25-C1 — common `0x311`; [C] Common Field Projection, [CS] `Limits` | R; unsigned big-endian limits in 0.1 V / 0.1 A; status word raw only | Proven shared members; QC; group inspected | [S] limit vocabulary Hypothesis; T? storage/energy-management observation. Same role/scaling/identity gaps as O25-V1; status bits cannot select a revision or grant control |
| O25-C2 — common `0x312`; [C], [CS] `Status` | R; raw protection/warning bytes and pack count; bytes 5–7 raw only | Proven shared members; named conditions Unknown without selected revision; QC; group inspected | [S] alarms/topology candidates withheld pending revision-specific meanings and member identity. T? battery diagnostics; no manufacturer/cell count or derating reason from divergent bytes |
| O25-C3 — common `0x313`; [C], [CS] `Measurements` | R; signed voltage 0.01 V, signed current 0.1 A, SOC percent, low-seven-bit SOH; temperature bytes and SOH high bit raw only | Proven source members; semantic joins Hypothesis; QC; group inspected | [S] voltage/current/SOC/SOH candidates need range, validity, sign/reference and service evidence. T? electrical/battery observations. No maximum/average temperature, SOH validity, measured power or energy inferred |
| O25-C4 — other/malformed observations and revision-divergent fields; [C] Input Boundary, [CS] `Apply` | R; no engineering unit assigned to opaque content; full source-local `RawEvidence` retained, no typed fields | Proven raw-only boundary; meaning Unknown; QC; gap disposition recorded | T? unresolved target; valid geometry alone does not supply a typed fact. Wrong interface is rejected. Wider Growatt revisions, high-voltage and other families require separate contracts and source sweeps |

## Gree raw and candidate observations

[G] defines structural candidates, not final property meaning. [GS] requires
extended format, nonempty matching effective/raw DLC, class `0xf7`, unit `8`,
and opcodes `0x10/0x11/0x52/0x58`, with bounded coordinate spans and actual
candidate updates. Unsupported, malformed or out-of-profile frames remain
raw without a Gree projection. `opaque7` is preserved, never normalized into
an equipment or service identity. [GQ] adds explicit source/interface and map
selection at the [replay boundary][GR].

| Row / native group and source | Direction / native units and retained evidence | Grade / native qualification / source sweep | Semantic candidate and target search / exact remaining gap |
| --- | --- | --- | --- |
| O25-G1 — raw/unqualified context; [G] Scope and Extended Identifier Layout, [RS] `Evidence` | R; no engineering units; identifier partitions, frame/payload/raw DLC, source, sequence, monotonic context and exact raw record remain native | Proven retention contract and evidence shape; HVAC meaning Unknown; QR only for accepted offline path; raw boundary inspected | [H] supplies thermal vocabulary, no correspondence. T? unresolved HVAC target; raw bytes cannot prove temperature, humidity, mode, fan, cooling capability or equipment health |
| O25-G2 — opcode `0x10`; [G] Receive Layout, [GS] `Opcode10Updates` | R; coordinate `0x05` pair to cells `0x0f/0x10`, coordinate `0x20` to cell `0x11`; opaque bytes, no units; original `Evidence` retained | Proven bounded candidate structure; HVAC interpretation Hypothesis at most; QR; group inspected | T? thermostat/air-conditioner/setpoint searches blocked by property identity, unit, range, zone binding and current-versus-target role Unknown. A pair of bytes is not a temperature |
| O25-G3 — opcode `0x52`; [GS] `Opcode52Updates`, [G] packed-bit layout | R; packed bit coordinate `0x59` to cell `0x12`; unitless native bit with source Evidence | Proven structure; HVAC meaning Unknown; QR; group inspected | T? mode/status search; Boolean value establishes neither operating mode nor capability, fault or controllability |
| O25-G4 — opcode `0x58`; [GS] `Opcode58Updates`, [G] byte layout | R; pairs `0x5d/0x5f/0x59/0x5b` to cells `0x13..0x1a`; opaque bytes with source Evidence | Proven structure; HVAC meaning Unknown; QR; group inspected | T? sensor/fan/HVAC search; quantity, endian interpretation, scale, measurement placement and service scope Unknown. No semantic aggregate is inferred |
| O25-G5 — opcode `0x11`; [GS] `SingleByteUpdate`, [G] byte layout | R; coordinate `0x09` to cell `0x1b`; opaque byte with source Evidence | Proven structure; HVAC meaning Unknown; QR; group inspected | T? unresolved HVAC component; property identity, value domain and relation to other cells remain Unknown |
| O25-G6 — selected M94 map; [G] Active Maps and State-Cell Update Rules, [GM] and [GD] | R; 94 ordered rows / 73 destination cells; bit, u8 and u16-LE byte-stream structural sources, transforms/latches/100 slot flags; no engineering units | Proven documented offline map and implementation roster; QR; map group inspected, not replayed | T? thermal/HVAC candidates withheld. Map state and candidate updates are distinct projections, not additional qualified facts. Duplicate `0x220d` rows both apply; labels/transform names do not prove HVAC meaning or actual unit count |
| O25-G7 — selected M115 map; [G], [GM] and [GD] | R; 115 rows / 83 destination cells; ordered transforms and selector-qualified cleanup; no engineering units; [GR] retains candidate Evidence and map result | Proven documented offline map and roster; QR; map group inspected, not replayed | T? thermal/HVAC candidates withheld. Preserve selected map and selector, cleanup and untouched state; no Profile-A meaning/width borrowed. M152 is not active; selectors are not equipment identification |

The full map decoder [GM] covers more opcode/source-key rows than the narrow
candidate gate [GS]. [GR] first applies that gate: a complete offline map table
is not proof every row is reachable through the accepted observation path.
[G]'s frame-rule/transform names remain structural. The [property catalog][GP]
explicitly treats names as profile-qualified diagnostic candidates, and widths
as storage only. The [bridge record][GB] fixes a bounded structural handoff,
not UART identity or a semantic feed; the [opaque-cell boundary][GO] preserves
native byte values without semantic replacement. These are evidence references,
not additional Matter source facts.

## Reverse direction and remaining inventory work

No row establishes Matter-to-native operation support. Growatt is receive-only;
charge limits, force-charge requests and enable bits do not create a command
route. Gree's [command map][GC] is deterministic in-memory encoding only;
equipment writes are disabled and unsafe, and encoding proves no delivery,
acceptance, effect or confirming readback. Fabric-input thermostat bindings,
setpoints, modes, fans and all other representable types still need independent
reverse-direction joins. See [thermostats](thermostats.md) and
[air conditioning](air-conditioning.md); their inventories are not replaced.

The bounded roster is four V1.04 groups, four common-projection groups and
seven Gree groups above: **15 review groups**, not unique facts, targets, devices
or a full six-domain denominator. This closes inspection of the named source
views only; V4 and C4 preserve wider decoding/profile gaps. Source facts remain
separate across profiles, directions and scopes. [Storage](storage.md) and the
existing thermal pages retain their canonical facts; no cross-transport alias
or new endpoint sharing is decided here.

Next review must inspect every relevant adopted target device type and cluster
in both directions, expand feature conditions and mandatory attributes through
[I]'s closure procedure, reconcile exact semantic joins, and record all
exclusions with evidence. Pumps, thermostats, B524 and every other representable
family remain required. No Unknown-filled row establishes completeness or
releases M42/M43. Native raw evidence and permitted partial last-known-good
values remain intact and distinct from freshness/qualification; no eBUS MCP or
GraphQL behavior changes. Full committed-change review/CI, go-fabric build
qualification and physical acceptance remain unverified.

[issue]: https://github.com/Project-Helianthus/helianthus-matter-binding/issues/8
[I]: inventory-completeness.md
[L]: ../../sdk/matter-sdk.lock.json
[V]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/growatt/low-voltage-bms-can-v104.md
[C]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/growatt/growatt-low-voltage-bms-can-common-projection-v1.md
[G]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/gree/vrf-canbus.md
[VQ]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/growatt/growatt-low-voltage-bms-can-v104-qualification-card-v1.md
[GQ]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/gree/gree-vrf-can-qualification-card-v1.md
[VS]: https://github.com/Project-Helianthus/helianthus-canbusreg/blob/main/growatt_v104.go
[CS]: https://github.com/Project-Helianthus/helianthus-canbusreg/blob/main/growatt_common_projection.go
[GS]: https://github.com/Project-Helianthus/helianthus-canbusreg/blob/main/gree_vrf_candidate.go
[RS]: https://github.com/Project-Helianthus/helianthus-canbusreg/blob/main/registry.go
[GM]: https://github.com/Project-Helianthus/helianthus-canbusreg/blob/main/gree_vrf_map.go
[GD]: https://github.com/Project-Helianthus/helianthus-canbusreg/blob/main/gree_vrf_map_data.go
[GR]: https://github.com/Project-Helianthus/helianthus-canbusreg/blob/main/gree_vrf_replay.go
[S]: https://github.com/Project-Helianthus/helianthus-docs-semantic/blob/main/api/v1/packs/storage-bms-v1.md
[H]: https://github.com/Project-Helianthus/helianthus-docs-semantic/blob/main/api/v1/packs/thermal-hvac-v1.md
[T]: https://github.com/Project-Helianthus/helianthus-docs-semantic/blob/main/api/v1/targets/matter-1.7-ballot-0.9-v1.md
[GP]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/gree/gree-vrf-property-catalog.md
[GB]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/gree/gree-vrf-can-bridge-record-v1.md
[GO]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/gree/gree-vrf-can-opaque-cell-mcp-v1.md
[GC]: https://github.com/Project-Helianthus/helianthus-docs-canbus/blob/main/protocols/gree/gree-vrf-command-map.md
