# Helianthus Matter Binding

This public repository owns the Matter SDK boundary for Helianthus output
bindings: SDK revision selection, node and endpoint allocation, fabric-storage
integration, commissioning integration, reporting integration, controller
fixtures, and the admission rule for a future production endpoint.

It does not own protocol-neutral semantics, Gateway native decoding, vendor
protocol behavior, Matter certification, production fabrics, credentials, or
deployment. Do not infer a production endpoint, conformance, certification, or
interoperability claim from a passing SDK fixture.

## Workflow

Use a scoped issue and an `issue/<number>-<slug>` branch from the remote default
branch. Preserve concurrent work. Keep changes within the owning issue, run the
commands below, document material runtime or public-contract changes, and open a
linked PR. Do not merge until applicable hosted checks and an independent
exact-HEAD review have passed.

## Validation

The local, dependency-free baseline is:

```bash
python3 tools/validate_lock.py
python3 -m unittest discover -s tests -v
```

The hosted workflow performs the exact-SHA recursive SDK checkout, checks its
tracked blobs, builds `all-devices-app` and `chip-tool`, runs focused Electrical
Power Measurement and Power Topology tests, then runs the bounded upstream
Electrical Sensor controller fixture. These commands require a Linux runner and
are intentionally not a requirement for local contributors.

Documentation is required for changed public ownership, Matter target semantics,
readiness rules, fixtures, or safety boundaries. This repository has no declared
T01..T88 transport matrix. Physical or live-fabric smoke is outside normal CI.

## Safety and evidence

Never commit credentials, fabric storage, attestation material, personal network
coordinates, device identifiers, private captures, or generated local SDK state.
Do not commission a real device, join a real fabric, deploy a node, call a live
Gateway operation, or perform a safety-relevant write without explicit operator
confirmation at action time.

The runtime admission code must fail closed. This revision has no production
admission path: the reviewed lock has no accepted composition-profile digest,
so no caller-provided metadata or build result can permit allocation. A future
reviewed lock may pin an exact versioned composition profile only after it
supplies all mandatory target facts and matching build evidence.

## Documentation destinations

Keep binding ownership, the lock, fixture procedure, and readiness boundary in
this repository. Put protocol-neutral semantic contracts in their explicit public
owner and Matter target mappings in the public semantic documentation repository;
do not copy private material here.
