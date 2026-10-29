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

## Adopted baseline and retained legacy tooling

Helianthus adopts [go-fabric](https://github.com/SukramJ/go-fabric), an
Apache-2.0 pure-Go semantic port of matter.js, on its **Matter 1.6.1** data
model. SDK selection is an owner decision, not qualification or certification.
The full inventory includes storage, thermal/HVAC, PV, EVSE, metering,
infrastructure, pumps, thermostats, valves/mixers and bridge composition; it is
not reduced to an initial Electrical Sensor bridge.

The checked-in [SDK lock](sdk/matter-sdk.lock.json), strict validator and
[hosted job](.github/workflows/sdk-baseline.yml) still describe the legacy
connectedhomeip fork and Matter 1.7 target. Their replacement is owner-controlled
work and has **not** happened here. Legacy build/fixture results cannot qualify
go-fabric. Neither a complete retained normative model corpus, a Linux arm64
build nor an accepted production profile is supplied by this documentation.

See the [baseline and legacy fixture procedure](docs/SDK_BASELINE.md),
[full inventory reconciliation](docs/target-profile/inventory-completeness.md)
and [named deferrals and snapshot gaps](docs/adoption/matter-161-deferred.md).
Matter 1.7 remains draft direction for later pre-1.0 review; nothing is stable
before Helianthus 1.0.

## Fail-closed production admission

`runtime.production_readiness.evaluate()` has no production-admission path in
this revision. The reviewed lock pins
`accepted_runtime_composition_profile_sha256` to `null`, so every invocation
denies endpoint allocation, including calls that provide self-asserted
`accepted` metadata or passed build JSON. A future reviewed change may replace
that immutable absent state with an exact canonical profile digest, after which
the unmodified lock, matching independently verified SDK build/fixture result, and matching accepted
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
