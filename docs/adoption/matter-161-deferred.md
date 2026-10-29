# Matter 1.6.1 adoption: deferrals and snapshot gaps

Helianthus has adopted [go-fabric](https://github.com/SukramJ/go-fabric) and its
Matter 1.6.1 data model. The checked-in [legacy lock](../../sdk/matter-sdk.lock.json)
and [hosted job](../../.github/workflows/sdk-baseline.yml) still target the
connectedhomeip fork's Matter 1.7 model pending owner-controlled replacement.
This ledger does not change those artifacts, accept a composition profile or
establish a complete normative model comparison. Nothing is stable before
Helianthus 1.0. Matter 1.7 remains draft direction for later pre-1.0 review.

**Proven** means only the bounded source-inspection statement attributed to the
supplied graded report; that inspection was not rerun here. **Hypothesis** is a
candidate correspondence; **Unknown** marks uninspected or missing evidence.
These grades never imply qualification, conformance, certification, ecosystem
interoperability, Linux arm64 execution, Pi performance or latency.

Public [go-fabric source](https://github.com/SukramJ/go-fabric/tree/main) and
[connectedhomeip model sources](https://github.com/project-chip/connectedhomeip/tree/master/data_model)
are locators only. No online verification or complete retained normative corpus
was supplied. The [baseline](../SDK_BASELINE.md) records the bounded source
identities; the [full-family inventory](../target-profile/inventory-completeness.md)
keeps all existing native rows and independent qualification/direction/loss gaps.

| Field / feature | Old identity / expectation | Adopted identity / disposition | Consequence | Evidence grade and limit | Re-entry trigger |
| --- | --- | --- | --- | --- | --- |
| Electrical Sensor: Electrical Alarm support | Legacy Matter 1.7 Electrical Sensor revision 2; Electrical Alarm addition in that target | Matter 1.6.1 Electrical Sensor revision 1; Changed, 1.7 addition deferred | No Electrical Alarm support claim or invented native alarm mapping; baseline current reading still needs complete composition | Proven reported Sensor revision 1 source presence; alarm applicability and full normative closure Unknown here | Reviewed later SDK/model support, retained feature/mandatory-composition evidence, accepted public semantic mapping and matching build/reporting evidence |
| Power Topology: ElectricalCircuit | Legacy Matter 1.7 Power Topology revision 2; ElectricalCircuit addition in that target | Matter 1.6.1 Power Topology revision 1; Changed, 1.7 addition deferred | No ElectricalCircuit topology claim; native circuit identity does not supply physical target topology | Proven reported Power Topology revision 1 source presence; complete feature and topology requirements Unknown here | Reviewed later model/SDK support plus retained ElectricalCircuit definition, native topology evidence, accepted target composition and matching build evidence |
| Device Energy Management: PowerRangeAdjustment | Prior inventory energy-management candidate; exact old feature identity/availability Unknown | DEM revision 4 present; PowerRangeAdjustment absent in the inspected upstream snapshot; snapshot gap, not a universal Matter 1.7 delta | Withhold this feature and any dependent range-adjustment operation; do not infer absence of all DEM or limits observation support | Proven bounded reported absence in upstream snapshot; standard-wide version attribution Unknown | Upstream snapshot supplies the feature, followed by retained model/source inspection, public native operation/effect evidence, semantic acceptance and matching build tests |
| Other uninspected model/feature/attribute/command/event differences | Legacy full Matter 1.7 corpus and accepted semantic target contents unavailable for comparison | Adopted complete Matter 1.6.1 normative corpus not retained; Unknown comparison, including EEM, Thermostat, Energy EVSE, Aggregator, Bridged Node and all remaining families | No exhaustive delta, negative-support claim or first-bridge scope reduction; retain every source fact and incomplete target join | Unknown; reported component revisions and unchanged EPM revision do not prove field/feature equivalence | Provision complete retained public model and semantic evidence; reconcile every relevant family and dependency with independent review and SDK build/reporting evidence |

EPM revision 3 is unchanged at the revision level; EEM revision 2, Thermostat
revision 11, DEM revision 4, Energy EVSE revision 4, Aggregator revision 2 and
Bridged Node revision 3 have reported source presence. Their old complete
identities were not inspected, so comparisons stay Unknown. Neither a matching
revision nor a similarly named field proves semantic equivalence.

The sole accepted mapping remains `evse.ac.current`; optional ActiveCurrent
cannot provide Descriptor and Power Topology composition, PowerMode,
NumberOfMeasurementTypes, non-empty matching Accuracy or an evidenced nullable
ActivePower fact. No accepted production composition-profile digest is pinned,
so admission remains denied regardless of caller metadata or build success.
Native evidence and partial last-known-good state remain intact; eBUS MCP
compatibility and GraphQL parity are unchanged. No live operation, device access,
commissioning, fabric join or deployment is authorized. Inventory review adds
no authorization gate before 0.9; authentication/authorization remain 0.11 work.
