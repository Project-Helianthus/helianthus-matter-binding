# Matter SDK baseline and offline fixture

## Adopted SDK and transition boundary

Helianthus adopts [SukramJ/go-fabric](https://github.com/SukramJ/go-fabric)
under Apache-2.0: a pure-Go semantic port of matter.js using the Matter **1.6.1**
data model. This is the selected SDK direction now, not a future candidate or
an implementation-readiness verdict. It is not certified. Public source links
locate upstream material; no remote source was fetched for this reconciliation.

The checked-in [lock](../sdk/matter-sdk.lock.json), validator and
[hosted workflow](../.github/workflows/sdk-baseline.yml) remain **legacy**:
they select the [connectedhomeip fork](https://github.com/AryaHassanli/connectedhomeip)
and its Matter 1.7 target. The legacy semantic artifact remains at its
[public owner](https://github.com/Project-Helianthus/helianthus-docs-semantic/blob/main/api/v1/targets/matter-1.7-ballot-0.9-v1.json).
No dependency pin, workflow, runtime admission code or accepted semantic mapping
is migrated by this documentation. Legacy evidence remains attributable to that
implementation; it cannot qualify the adopted SDK.

## Bounded source presence, separate from qualification

The supplied graded source-inspection result supports the following identities.
**Proven** here means reported source presence only; this worker did not rerun
that inspection. Full normative requirements, feature applicability, mandatory
composition, target mappings, generated behavior and build results remain
**Unknown** unless separately evidenced. Public
[go-fabric source](https://github.com/SukramJ/go-fabric/tree/main) is a locator,
not retained normative-corpus evidence or independent replay.

| Component | Legacy identity / prior inventory | Adopted source identity | Disposition and limit |
| --- | --- | --- | --- |
| Electrical Sensor `0x0510` | revision 2 in legacy lock | revision 1 | Changed; Electrical Alarm support deferred |
| Power Topology `0x009C` | revision 2 in legacy lock | revision 1 | Changed; ElectricalCircuit deferred |
| Electrical Power Measurement `0x0090` | revision 3 in legacy lock | revision 3 | Unchanged revision; features and composition still need evidence |
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

## Hosted reproduction — retained legacy procedure

`.github/workflows/sdk-baseline.yml` clones the exact fork object into a clean
Ubuntu runner, initializes recursive submodules, verifies detached HEAD and all
locked blobs, then uses that checkout's scripts to build:

1. `linux-x64-all-devices-clang`, including code-driven
   `examples/all-devices-app/posix`;
2. `linux-x64-chip-tool-clang`;
3. focused Electrical Power Measurement and Power Topology cluster test roots.

The fixture starts `all-devices-app --device electrical-sensor:1` with a unique
temporary KVS path, the SDK's public test passcode and discriminator, and a
temporary controller storage path. It then invokes the SDK's EPM test which
commissions the sample and verifies an `ActiveCurrent` read after the sample's
test event. The fixture does not use the SDK runner's broad `--factory-reset`;
all explicit sample fabric and controller state belongs to its unique temporary
directory. The direct `chip-tool` calls bind `TMPDIR` to that directory, assert
that their default and alpha commissioner storage plus platform KVS were created
there, and verify an external sentinel was unchanged. The pinned Linux binaries
also use compile-time `/tmp`
defaults for generic factory, configuration, and counter bookkeeping; the
workflow therefore runs on a fresh hosted runner and does not treat those files
as reusable Helianthus state.
The script removes owned storage, terminates the directly started app through an EXIT
trap, and runs SDK helpers in a separately owned process group that is terminated
and reaped on success, failure, or timeout. It never reads a private network,
fabric, or credential.

The fixture proves only an upstream SDK sample's descriptor/device/cluster
composition and controller interaction. It cannot qualify a Helianthus runtime,
make a production profile accepted, prove a real fabric path, or establish
Matter conformance or certification.

Legacy build success is not established by this documentation. The retained
hosted procedure and any results apply only to the legacy lock, not go-fabric.
No hosted, Linux arm64, platform or physical qualification was performed here.

## Production gate

The retained electrical readiness contract requires Descriptor and Power
Topology composition. Electrical Power Measurement requires a non-unknown `PowerMode`, a positive
`NumberOfMeasurementTypes`, non-empty fixed `Accuracy`, and a nullable
`ActivePower` fact. The accepted mapping currently has only `evse.ac.current`.
It cannot supply the missing facts by analogy or default.

`runtime/production_readiness.py` has no production-admission path in this
revision. The reviewed lock fixes
`accepted_runtime_composition_profile_sha256` to `null`, so every call is
denied before caller-supplied profile or build fields can be trusted. A future
reviewed revision may add an exact canonical-profile digest; only then would it
require all of:

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
retained normative model corpus, accepted semantic target evidence and actual
lock migration remain external provisioning/owner work. Source inspection,
legacy sample success and documentation checks establish neither Matter
conformance nor certification, interoperability, arm64 behavior or production
admission. Native evidence and partial last-known-good state remain preserved;
eBUS MCP compatibility and GraphQL parity are unchanged.
