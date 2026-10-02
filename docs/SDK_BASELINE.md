# Matter SDK baseline and offline fixture

## Evidence recorded by the lock

The reviewed target is the public fork commit
`29b4768a513cf566011ab8cd60df1bc495204953`. Its relevant Matter 1.7 source
identities are fixed in `sdk/matter-sdk.lock.json`:

| Artifact | Locked blob |
| --- | --- |
| `data_model/1.7/device_types/ElectricalSensor.xml` | `b6235fc64f44d4db2c42d05dc1e9b1f6f51dedfd` |
| `data_model/1.7/clusters/ElectricalPowerMeasurement.xml` | `3349108215d56ad61a950a3ad874ea9773499cb5` |
| `data_model/1.7/spec_sha` | `86ef3bbd871805600c751bd84c639cfb0af64ed9` |
| `data_model/1.7/spec_tag` | `df6ffc2f98235eea1a41493e971ef53e69084425` |
| generated Electrical Power Measurement `AttributeIds.h` | `8f95703c7a97c8c86752695446220a900819fd28` |
| code-driven Electrical Sensor implementation | `3f937a355870efc38f512ee40a38007ca4cc6bd3` |

The upstream merge provenance is `project-chip/connectedhomeip#73842` at
`bd73b5141729af92f3d021aeecaa53d9e0cadb89`. This documents identical accepted
target data-model artifacts; it is not a build result for the fork target.

The mapping is bound to the public semantic repository main revision
`30f5e5c79ac6da3a7c7c10c990599906d1dfd0cb`, path
`api/v1/targets/matter-1.7-ballot-0.9-v1.json`, SHA-256
`5ae81d5e0971d25ead08f982fdf31caf47ada4838d0ee6f2b3e719d11a6df39c`.

The locked target identifiers are Electrical Sensor `0x0510` revision 2, Power
Topology `0x009C` revision 2, Electrical Power Measurement `0x0090` revision
3, and optional `ActiveCurrent` `0x0005`.

## Hosted reproduction

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
all KVS and controller state belongs to its unique temporary directory. The
direct `chip-tool` calls bind `TMPDIR` to that directory, assert that their
configuration was created there, and verify an external sentinel was unchanged.
The script removes owned storage, terminates the directly started app through an EXIT
trap, and runs SDK helpers in a separately owned process group that is terminated
and reaped on success, failure, or timeout. It never reads a private network,
fabric, or credential.

The fixture proves only an upstream SDK sample's descriptor/device/cluster
composition and controller interaction. It cannot qualify a Helianthus runtime,
make a production profile accepted, prove a real fabric path, or establish
Matter conformance or certification.

The exact fork SHA had no own check run when this baseline was authored. Build
success remains unproven until the hosted workflow runs successfully at the
exact repository commit containing this lock.

## Production gate

Electrical Sensor revision 2 requires Power Topology. Electrical Power
Measurement revision 3 requires a non-unknown `PowerMode`, a positive
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
- exact-SHA build evidence whose five required targets are all `passed`;
- an `accepted` composition profile matching the lock and mapping revision;
- evidenced Descriptor, Power Topology, and Electrical Power Measurement facts.

[`examples/runtime-composition.profile.json`](../examples/runtime-composition.profile.json)
is intentionally a `proposed` template with missing evidence. It documents the
required shape but cannot pass the gate even if a caller changes its status. The
admissible closure options are to extend accepted evidence, select an
evidence-backed standards-valid target, or withhold the existing mapping.
