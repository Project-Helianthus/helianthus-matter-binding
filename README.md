# Helianthus Matter Binding

This is the public runtime owner for the Helianthus Matter output boundary. It
owns Matter SDK selection, node and endpoint allocation, fabric-storage and
commissioning integration, reporting integration, and the offline controller
fixture. It does not own protocol-neutral semantics, Gateway native decoding,
vendor protocol behavior, credentials, deployed nodes, or certification.

The baseline is intentionally an admission layer, not a running production
bridge. The currently accepted mapping supplies only `evse.ac.current`; it does
not supply the mandatory Electrical Sensor composition facts. Therefore no
Helianthus Electrical Sensor endpoint can be enabled by this repository today.

## Exact baseline

[`sdk/matter-sdk.lock.json`](sdk/matter-sdk.lock.json) records the accepted
public target:

- fork target `AryaHassanli/connectedhomeip` at
  `29b4768a513cf566011ab8cd60df1bc495204953`;
- upstream Matter 1.7 merge [project-chip/connectedhomeip#73842](https://github.com/project-chip/connectedhomeip/pull/73842)
  at `bd73b5141729af92f3d021aeecaa53d9e0cadb89`;
- semantic mapping revision `30f5e5c79ac6da3a7c7c10c990599906d1dfd0cb`,
  `api/v1/targets/matter-1.7-ballot-0.9-v1.json`, SHA-256
  `5ae81d5e0971d25ead08f982fdf31caf47ada4838d0ee6f2b3e719d11a6df39c`.

The lock binds Electrical Sensor `0x0510`, Power Topology `0x009C`, Electrical
Power Measurement `0x0090`, and optional `ActiveCurrent` `0x0005`.

The strict validator checks the reviewed lock shape and, when given an SDK
checkout, checks detached HEAD and every locked blob. It refuses changed or
additional lock content.

```bash
python3 tools/validate_lock.py
python3 tools/validate_lock.py --sdk-checkout /path/to/connectedhomeip
python3 -m unittest discover -s tests -v
```

The hosted job creates the only intended build evidence: a clean recursive Linux
checkout at the exact SHA, builds the code-driven `all-devices-app` and
`chip-tool`, builds focused Electrical Power Measurement and Power Topology
tests, and runs the upstream Electrical Sensor fixture with temporary SDK test
state. See [the baseline procedure](docs/SDK_BASELINE.md).

## Fail-closed production admission

`runtime.production_readiness.evaluate()` has no production-admission path in
this revision. The reviewed lock pins
`accepted_runtime_composition_profile_sha256` to `null`, so every invocation
denies endpoint allocation, including calls that provide self-asserted
`accepted` metadata or passed build JSON. A future reviewed change may replace
that immutable absent state with an exact canonical profile digest, after which
the unmodified lock, exact-SHA build/fixture result, and matching accepted
profile would all be required. That profile must provide evidenced Descriptor
and Power Topology composition plus Electrical Power Measurement `PowerMode`,
`NumberOfMeasurementTypes`, a non-empty `Accuracy` list, and its nullable
`ActivePower` fact. `ActiveCurrent` remains optional at the Matter target level.

No accepted production composition profile is supplied here. The template is
deliberately `proposed`, and cannot open the gate even if edited to assert
acceptance. A future accepted profile must be backed by public native/SemReg
evidence and separately reviewed with its exact digest added to the lock.

The closure choices remain open: extend accepted native/SemReg evidence for the
mandatory metadata; choose another standards-valid, evidence-backed target; or
keep this mapping withheld. The upstream fixture is a complete SDK sample; its
metadata must never become Helianthus production semantics.

## Safety and non-claims

This repository does not commission or join a real fabric, use production
credentials, call a live Gateway, deploy a node, prove Matter conformance or
certification, establish ecosystem interoperability, or qualify real hardware.
Its fixture uses temporary test storage and cleans itself up. See
[AGENTS.md](AGENTS.md) and [CONTRIBUTING.md](CONTRIBUTING.md).
