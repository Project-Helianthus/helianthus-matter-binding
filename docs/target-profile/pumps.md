# Pump-family source inventory and target gaps

This bounded source inventory addresses [issue #41](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/41)
and advances [#8](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/8)
and [#23](https://github.com/Project-Helianthus/helianthus-matter-binding/issues/23)
as a partial contract slice. It distinguishes controller circuit pumps,
controller solar pumps and direct boiler pump provenance. It completes only
this named source inventory slice, not target closure or either parent.
Nothing here is stable before Helianthus 1.0.

The [heating inventory][H] retains the general system, circuit, DHW, zone and
boiler inventories. This page refines pump identity and loss; it does not repeat
the general inventories or create protocol semantics. The public native catalog
[N] comes before semantic correspondence. Protocol-neutral definitions remain
in SemReg [S], and accepted Matter mappings belong to the public semantic
documentation owner identified by the binding lock, as described in [I].

## Evidence and row identity

**Proven** below means that the supplied public source establishes the stated
catalog entry or definition, not that native meaning or an installation has
been qualified. **Hypothesis** marks an interpretation still needing evidence;
**Unknown** marks missing evidence. Native qualification is **Unknown** for all
rows. No row grants qualified use. Solar entries marked `†` by [N] are
value-matched only, with false-positive risk and no independent ebusd coverage.

Every row is a Thermal-domain native-to-Matter **observation candidate**. A
listed configuration or command is inventoried as a distinct native role, never
as an executable Matter operation. Read evidence cannot establish control.
All joins are **blocked pending evidence**, not accepted representable mappings
and not proofs of incompatibility with Matter. Source candidates whose native
value can be retained are distinguished from absent source evidence.

Row keys follow [I]: `(Thermal, native owner/profile, capability/fact,
direction, target component)`. For B524 rows the native owner is [N]; retain
the qualified controller profile, `OP=0x02`, `OT=0x00`, `GG`, concrete `II`
and `RR`, plus service/measurement role. Family names alone do not resolve a
profile. Circuit `II=0x00` is a local circuit identity, not a boiler identity.
[N] bounds ordinary heating candidates to `II00..08` for the characterized
controller family; `II09` is virtual native water, not a physical heating pump.
Other profiles need their own bounds and presence evidence. Missing replies
do not prove absence or an off state.

For solar rows, [N] “GG=0x04 — Solar Circuit” states `II=0x00`, while its
group-topology/current-bounds tables list `II00..01`. This is an unresolved
source-scope conflict, not evidence for a second pump. Keep any concrete
selector distinct; do not expand support or count an extra instance until the
native owner reconciles the conflict. `fm5_config <= 2` is the documented
applicability condition, not sufficient proof of physical presence.

**T?** in each row means the target device type/revision, cluster/revision,
feature, attribute/command/event role and accepted public semantic mapping are
all Unknown. The locked Matter 1.7 target corpus and mapping contents are not
supplied. **M0** is the unresolved mandatory-feature closure procedure below;
it applies separately to every row. No normative pump IDs are inferred from
names, samples or the existing electrical fixture. Admission remains denied.

## Bounded source-to-target ledger

Native integers below use the catalog's little-endian encoding. A catalog unit
does not prove reset, rollover, saturation or observation-time semantics.
Driver evidence in [H] is an indirect source locator, not inspection of the
driver itself; no driver source was supplied for this review.

| Canonical row / exact public source locator | Source direction, role, encoding and unit | Source confidence / native qualification | Semantic candidate and driver evidence | Target evidence / mandatory gap | Loss or withholding | Falsifier requiring rejection or reopened review |
| --- | --- | --- | --- | --- | --- | --- |
| P1: circuit pump status; [N] Circuits `GG=0x02 RR=0x001B`, `circuit_pump_unknown_8`; Circuit State Enum | Observation; circuit-local raw status, `u16`, raw/unitless | Proven catalog identity; pump interpretation is Hypothesis based on naming; nonzero-active is Hypothesis [H]; qualification Unknown | [S] has no exact pump-status field. [H] cites `refreshCircuits` and excludes circuit-pump mirroring in `mergeBoilerMirrorFieldsFromSnapshots`; driver implementation uninspected here | T?; M0 plus exact raw-code semantics, unavailable representation and qualified service binding | Preserve raw codes and selector. Do not turn zero/nonzero into universal idle/active or boiler activity | Another qualified raw code disproves Boolean meaning, or a circuit sample populates a boiler fact |
| P2: circuit state, separate from P1; [N] Circuits `GG=0x02 RR=0x001E`, `circuit_state`; Circuit State Enum | Observation; circuit operating-state candidate, `u16`, enum | Proven distinct selector; standby/heating/cooling/DHW assignment is Hypothesis; qualification Unknown | `thermal.action.state` exists [S], but equivalence and service attachment are unaccepted; [H] is the source inventory, not independent acquisition proof | T?; M0 plus corroborated state codebook and action-versus-pump distinction | Do not relabel this as pump status because ebusd calls it `Hc{hc}PumpStatus`; no automatic action projection | Correlated evidence contradicts an enum assignment, or P1 and P2 collapse to one fact |
| P3: circuit pump hours; [N] Circuits `GG=0x02 RR=0x0024`, `circuit_pump_operating_time_total` | Observation; cumulative circuit-pump runtime, `u32`, hours | Proven catalog entry/unit; counter lifecycle Unknown; qualification Unknown | No exact cumulative-runtime field in [S]; [H] cites `refreshCircuits` acquisition, uninspected driver | T?; M0 plus counter reset/rollover and target width/unit handling | Not instantaneous activity, energy or boiler runtime; no delta across unqualified reset or source replacement | A decrease or replacement is treated as continuous elapsed runtime, or hours are interpreted as starts |
| P4: circuit pump starts; [N] Circuits `GG=0x02 RR=0x0025`, `circuit_pump_start_count_total` | Observation; cumulative circuit-pump starts, `u32`, count | Proven catalog entry/unit; reset/rollover Unknown; qualification Unknown | No exact start-counter field in [S]; [H] cites `refreshCircuits`, uninspected driver | T?; M0 plus event/count semantics, reset and target numeric range | Count is neither hours nor present activity; retain independently from P3 | Equal hours and starts merge into one identity, or counter reset is counted as new starts |
| P5: solar pump state; [N] Solar Circuit `GG=0x04 RR=0x0008`, `solar_pump_active` | Observation; candidate solar-pump state, `u8`, catalog Boolean `0=off, 1=on` | Proven catalog listing; Hypothesis native meaning, value-matched candidate (`†`), not independently confirmed; qualification Unknown | No exact pump-status field in [S]; solar acquisition locator `publishFM5Semantic` / `applyFM5Acquisition` in [H], per-register driver evidence absent | T?; M0 plus independent native corroboration, instance conflict resolution and source availability | Do not generalize to circuit or boiler status; unknown codes must not become true | Independent evidence shows another role/code, or `kick_enabled` is substituted for this selector |
| P6: solar pump hours; [N] Solar Circuit `GG=0x04 RR=0x000B`, `solar_pump_operating_time_total` | Observation; cumulative solar-pump runtime candidate, `u32`, hours | Proven catalog listing; Hypothesis native meaning, value-matched candidate (`†`); reset/rollover Unknown; qualification Unknown | No exact runtime-counter field in [S]; same indirect solar locators as P5, per-register acquisition uninspected | T?; M0 plus P5 scope gaps and counter lifecycle | Never circuit runtime, energy yield or evidence that the pump is currently running | Reset, instance change or coherent absence is silently bridged as continuous runtime |
| P7: solar reset role; [N] Solar Circuit `GG=0x04 RR=0x0001`, `solar_pump_operating_time_total_reset_command` | Inventory observation of command-role candidate; `u8`, catalog Boolean; no operation direction accepted | Proven listing; Hypothesis native meaning, value-matched candidate (`†`); command/effect and qualification unproven | [S] defines only temperature/mode operations, no pump-counter reset; no supplied driver operation proof | T?; M0 plus separate native operation/effect contract before any operation could be considered | A command label is not pump activity, permission or successful reset; no reset requested | A read value or configuration category enables a write or claims the counter was reset |
| P8: solar kick configuration; [N] Solar Circuit `GG=0x04 RR=0x0002`, `solar_pump_kick_enabled` | Observation; configuration candidate, `u8`, catalog Boolean | Proven listing; Hypothesis native meaning, value-matched candidate (`†`); qualification Unknown | No exact pump-kick configuration field in [S]; acquisition/operation evidence absent | T?; M0 plus independent configuration meaning and separate operation contract if ever proposed | Enabled is not active; no control from a configuration read | Enabled kick configuration produces an active-pump fact when P5 says off or is unavailable |
| P9: direct boiler pump provenance; [H] “DHW, direct boiler boundaries and commands”, boiler state/pump row and mirror row | Observation candidate; direct-boiler-local pump role; exact selector, width, unit and codebook Unknown in supplied views | Proven that [H] separates direct candidates from mirrors; native pump meaning and qualification Unknown | [H] names `refreshBoilerStatusB509` and the B509 catalog; neither direct source supplied here. No exact pump field in [S] | T?; M0 plus public direct native contract inspection, exact selector, driver acquisition and service identity | Absent evidence slot, not a representable fact. Circuit zero/active, system flow or circuit state cannot fill it | Any circuit-derived value establishes boiler pump activity or apparent independent boiler evidence |

## Mandatory-feature gaps and reconciliation

M0 is an explicit open ledger for each P row, not an invented mandatory list:

| Required target evidence | Current gap for P1–P9 | Closure evidence required later |
| --- | --- | --- |
| Exact target tuple | Device, cluster, revisions and feature selection Unknown | Inspect the complete locked Matter 1.7 definitions and identify exact files/elements; names alone are insufficient |
| Mandatory and conditional composition | Requirement roster and feature applicability Unknown | Expand all unconditional and feature-dependent requirements, dependencies and composition until none remain unreviewed; record condition, applicability, source fact and dependent row |
| Attribute/command/event contract | Access, units, scaling, ranges, nullability, unavailable states and reporting Unknown | Inspect locked requirements and an accepted public target mapping; match each required fact independently, including evidenced nullable representations |
| Source coverage for those requirements | Cannot determine whether pump state or counters supply required target facts | Join qualified native/service evidence and exact SemReg roles to each requirement; leave unsupported facts withheld |
| Profile and build evidence | No accepted pump composition or matching build evidence supplied | Separate reviewed profile acceptance and matching build evidence from inventory review; the current admission denial remains unchanged |

The existing electrical fixture's mandatory composition does not define a pump
target. Missing corpus evidence blocks closure; it does not prove that Matter
lacks pumps. No row is representable under [I]'s complete-closure definition.
Unknown-filled rows cannot release the full-inventory checkpoint or justify a
minimal subset, endpoint allocation, new SDK selection or upstream action.

P1–P4 refine the corresponding [H] circuit rows and are aliases for those
same native facts, not extra source counts. P2 remains distinct despite the
misleading ebusd pump name; P3/P4 remain distinct despite a shared circuit.
P5/P6 have different native groups and roles from the circuit facts even when
values match. P7/P8 retain command/configuration roles. P9 tracks unavailable
direct evidence, not another counted pump. Do not count unresolved solar
instances as supported services. No completed driver sweep, target sweep,
unique-target total or general inventory completeness is claimed.

Adjacent DHW pump observations (`GG=0x01 RR=0x0002/0x0006`) and circuit
external-DHW status (`GG=0x02 RR=0x0018`) retain their separate catalog roles
and the general heating inventory; none is an alias for P1, P5 or P9.
Collector temperature, raw solar flow, energy yield and hydraulic controls
also remain outside this pump-state/counter slice. In particular, [S]'s
flow-rate and power fields do not make a raw flow value or named pump-power
limit an accepted measurement, nor do they establish pump activity.

## Manual examples and lifecycle review

These invented cases are review examples, not executed native observations:

| Synthetic case | Required disposition / falsifier |
| --- | --- |
| Circuit zero has raw P1 value zero; no direct boiler sample exists | Retain circuit raw zero; P9 stays Unknown. Publishing boiler off or on from that value falsifies provenance |
| Circuit one has a nonzero P1 sample while direct boiler pump evidence is absent | Preserve the circuit sample without declaring boiler activity or universal system operation |
| P3 hours and P4 starts both equal 12 | Keep separate role/selector/unit identities. Equality supplies no alias or active state |
| P8 kick-enabled is one while P5 solar-active is zero | Configuration and candidate activity remain separate. Neither implies writable control |
| P6 hours decrease after a source-epoch change | Keep the evidence discontinuity; no cross-epoch delta, inferred reset command or renewed qualification |
| One circuit read fails while another succeeds | Preserve usable partial last-known-good fields with their original source and receipt/freshness; missing is not zero and reevaluation is not receipt |
| Solar acquisition degrades, then coherent structural absence is established | As described in [H], distinguish retained data during degradation from structural withdrawal; stale counters or configuration cannot recreate a service |

Future joins must retain native evidence, qualified asset/service identity,
source epoch, driver generation, semantic revision and freshness independently.
No gateway, eBUS MCP or GraphQL interface changes are made; parity,
deterministic envelopes, detached ownership and partial last-known-good behavior
remain unchanged. No authorization gate is added before 0.9; authorization
roadmap work remains at 0.11. No new major or experimental namespace is added.

## Evidence references and acceptance boundary

Public links locate files on `main`; offline inspection of the supplied views
does not verify network reachability, current-main freshness or the locked
target revision. Exact section names and selectors above locate the reviewed
source claims without publishing private input locations. Only [N], [S], [H]
and [I] were supplied and inspected; downstream driver/native references named
by [H] remain follow-up evidence, not inspected authority here.

- [N] [B524 register catalog](https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md): Table Legend, Group Topology, current bounds, Circuits, Solar Circuit and Circuit State Enum.
- [S] [Thermal definitions](https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/thermal/definitions.go): `fields`, `dimensions`, `services`, `capabilities`, `operations`.
- [H] [Heating source inventory](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/heating.md): Circuits, DHW, direct boiler boundaries and commands, Manual contract cases and transition review.
- [I] [Inventory completeness](https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md): Proposed reconciliation record, Mandatory-feature closure and shared facts, Later two-way sweep and inventory checkpoint.

Acceptance for this slice is manual per-row source, meaning, identity, unit,
loss, example and public-link review. It is not a prose-token test, native
qualification, runtime integration, transport or physical/live-fabric PASS.
Independent review, applicable hosted checks and parent acceptance remain
separate. The missing locked target corpus, accepted target mappings, direct
boiler source details, driver inspection and unresolved native semantics still
block broader claims. No live test, commissioning, deployment or device write
is performed or authorized by this inventory.

[N]: https://github.com/Project-Helianthus/helianthus-docs-ebus/blob/main/protocols/vaillant/ebus-vaillant-B524-register-map.md
[S]: https://github.com/Project-Helianthus/helianthus-semreg/blob/main/semreg/v1/packs/thermal/definitions.go
[H]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/heating.md
[I]: https://github.com/Project-Helianthus/helianthus-matter-binding/blob/main/docs/target-profile/inventory-completeness.md
