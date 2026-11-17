# Matter SDK baseline and offline fixture

## Adopted SDK

Helianthus adopts [go-fabric](https://github.com/SukramJ/go-fabric)
under Apache-2.0: a pure-Go semantic port of matter.js using the Matter **1.6.1**
data model. This is the selected SDK, not an implementation-readiness verdict.
It is not certified. Public source links locate upstream material; no remote
source was fetched for this reconciliation.

The checked-in [lock](../sdk/matter-sdk.lock.json) is the module pin.
`sdk.provider` is `go-fabric`, `sdk.module` is `github.com/SukramJ/go-fabric`,
`sdk.release_tag` is `v0.22.2`, and `sdk.checksum` is the module zip checksum
on the matching `go.sum` line (not the `go.mod` checksum). `sdk.model` is
Matter 1.6.1. The [validator](../tools/validate_lock.py) requires those fields
to match the module files exactly and rejects any other identity.
`accepted_runtime_composition_profile_sha256` stays unset, and `mapping` stays
unset. The public semantic owner has not published an accepted Matter 1.6.1
target. The [hosted workflow](../.github/workflows/sdk-baseline.yml) runs the
Go adapter job only.

## Bounded source presence, separate from qualification

The supplied graded source-inspection result supports the following identities.
**Proven** here means reported source presence only; this worker did not rerun
that inspection. Full normative requirements, feature applicability, mandatory
composition, target mappings, generated behavior and build results remain
**Unknown** unless separately evidenced. Public
[go-fabric source](https://github.com/SukramJ/go-fabric/tree/main) is a locator,
not retained normative-corpus evidence or independent replay.

| Component | Previously recorded identity | Adopted source identity | Disposition and limit |
| --- | --- | --- | --- |
| Electrical Sensor `0x0510` | revision 2 | revision 1 | Changed; Electrical Alarm support deferred |
| Power Topology `0x009C` | revision 2 | revision 1 | Changed; ElectricalCircuit deferred |
| Electrical Power Measurement `0x0090` | revision 3 | revision 3 | Unchanged revision; features and composition still need evidence |
| Electrical Energy Measurement | prior full identity Unknown | revision 2 | Unknown comparison; source presence does not accept energy mappings |
| Thermostat | prior full identity Unknown | revision 11 | Unknown comparison; heating/cooling and command closure unresolved |
| Device Energy Management (DEM) | prior full identity Unknown | revision 4 | Unknown comparison; PowerRangeAdjustment absent in inspected snapshot |
| Energy EVSE | prior full identity Unknown | revision 4 | Unknown comparison; no current-limit or charging-state mapping accepted |
| Aggregator | prior full identity Unknown | revision 2 | Unknown comparison; bridge composition remains unaccepted |
| Bridged Node | prior full identity Unknown | revision 3 | Unknown comparison; Descriptor, identity and lifecycle closure unresolved |

These are not nine qualified targets. Sensor/Topology deferrals, the separate
DEM snapshot gap and all uninspected differences are recorded in the
[deferred ledger](adoption/matter-161-deferred.md). Every existing family and
source row remains in the [full inventory](target-profile/inventory-completeness.md).
The sole accepted mapping remains `evse.ac.current`, with optional
`ActiveCurrent` `0x0005`; it supplies no complete endpoint composition.

## Hosted adapter job

`.github/workflows/sdk-baseline.yml` runs `gofabric-adapter` ("Go adapter
(go-fabric, module mode)"). That job installs the pinned Ruff, runs the lock
validator, the unit tests and the Python rule check, then runs the module-mode
adapter commands with and without the `gofabric` build tag. The commands are
listed in [AGENTS.md](../AGENTS.md). There is no second job.

The job does not check out another SDK, build a sample application or run a
controller fixture. A green job is Go adapter evidence for the pinned module.
It cannot qualify a Helianthus runtime, make a production profile accepted,
prove a real fabric path, or establish Matter conformance or certification.
No hosted, Linux arm64, platform or physical qualification was performed here.

## Production gate

The retained electrical readiness contract requires Descriptor and Power
Topology composition. Electrical Power Measurement requires a non-unknown `PowerMode`, a positive
`NumberOfMeasurementTypes`, non-empty fixed `Accuracy`, and a nullable
`ActivePower` fact. The accepted mapping currently has only `evse.ac.current`.
It cannot supply the missing facts by analogy or default.

`runtime/production_readiness.py` has no production-admission path in this
revision. Its embedded baseline keeps
`accepted_runtime_composition_profile_sha256` unset, so a lock that matches
that baseline is denied before caller-supplied profile or build fields can be
trusted. The module pin is a different object and does not open admission.
A future reviewed revision may add an exact canonical-profile digest to that
baseline; only then would it require all of:

- the unmodified lock;
- matching independently verified SDK build and fixture evidence;
- an `accepted` composition profile matching the reviewed lock and accepted mapping;
- evidenced Descriptor, Power Topology, and Electrical Power Measurement facts.

[`examples/runtime-composition.profile.json`](../examples/runtime-composition.profile.json)
is intentionally a `proposed` template with missing evidence. It documents the
required shape but cannot pass the gate even if a caller changes its status. The
admissible closure options are to extend accepted evidence, select an
evidence-backed standards-valid target, or withhold the existing mapping.

The adopted source identities above do not relax this boundary. The complete
retained normative model corpus and accepted semantic target evidence remain
external provisioning work. Source inspection and documentation checks establish
neither Matter conformance nor certification, interoperability, arm64 behavior
or production admission. Native evidence and partial last-known-good state
remain preserved; eBUS MCP compatibility and GraphQL parity are unchanged.
