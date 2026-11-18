# Changelog

## Unreleased

- Topology and bridge candidates key each node by its resource identifier.
  An edge is admitted only when it cites the relation key of a coherent
  instance, and a cross-source edge published by the core is admitted.
  Native owner and profile stay on the record as provenance. A native-tuple
  node, an uncited edge and an incoherent relation are refused. Reservations
  follow the resource key.

- Docs: `docs/reproducible-bridge-build.md` records the bridge binary as the
  standalone gateway built without the nomatter profile. The targets are
  linux/arm64 and linux/arm with GOARM=7, each built twice from a clean state
  with trimpath, version-control stamping off and an empty build id. Each run
  carries the Go toolchain, the module checksum of this module, the module
  checksum of go-fabric, and an SBOM. This repository builds no second
  executable. The official add-on build is not this job and stays without
  Matter. A bench run cites the digest and is a separate session.

- The SDK lock pins the go-fabric module path, release tag and module zip
  checksum recorded by the module files. The hosted workflow runs the Go
  adapter job only. `scripts/checkout_sdk.sh`, `scripts/run_upstream_fixture.sh`
  and `scripts/run_owned_group.sh` are removed with that job. Admission stays
  fail closed.

- The go-fabric module requirement is v0.22.2, and the Go directive is 1.27.4,
  the version that release declares. The adapter no longer sets a reannounce
  switch: the built-in responder stays off and the periodic interface
  comparison stays off. Subscription report callbacks return the error the
  release requires. Every breaking change from v0.16.0 through this release,
  and how this repository absorbs it, is recorded in
  `docs/adoption/pin-transition.md`. The fixture lock is unchanged.

- Docs: every inventory row under `docs/target-profile/` is now keyed by its
  element in the docs-semantic element catalog and that element's plane
  membership; native owner and profile move to a provenance column. The
  metering page is not a pack: its rows sit under the planes of their
  `helianthus.pack.infrastructure` elements. Rows without a catalog element say
  so, thermal measurement and setpoint rows keep their elements with planes
  undecided and no resource placement, and cross-source relations come from
  the core relation types. Row contents, device-type and cluster choices and
  admission are unchanged.

- CI: Ruff 0.15.10 now checks every Python file for naming (`N`), complexity
  (`C90`, at most 10) and statement-count (`PLR0915`, at most 50) findings
  against the committed `scripts/python-rules-baseline.json`, in the
  `gofabric-adapter` job and in the local validation list. New findings and
  grown measured values fail, including in a file with recorded debt; records
  without a finding must leave the baseline. The six existing findings are
  recorded, not fixed. Class N: every shipped Python file is AST-identical
  without docstrings and every other existing file outside the workflow,
  `AGENTS.md` and this changelog is byte-identical.

- Breaking: the Python semantic feed reader and its tests are removed. Nothing in
  this repository reads the gateway feed; the in-process Matter driver of the
  standalone gateway projects the shared semantic data
  (Project-Helianthus/helianthus-ebusgateway#1492). No replacement reader is
  provided; readiness and allocation stay fail closed.

- CI: the hosted workflow now runs two named jobs, `gofabric-adapter` ("Go adapter
  (go-fabric, module mode)") and `legacy-sdk-fixture` ("Legacy SDK fixture
  (connectedhomeip)"), so a green legacy check is not read as go-fabric
  qualification. `AGENTS.md` lists the go-fabric commands in the required local
  baseline. Each step keeps its commands and environment; the legacy job adds
  its own pinned checkout and Python setup and keeps its 90-minute bound.
- Breaking: the injected semantic snapshot reader now requests
  `semantic_snapshot(asset_ids:)` and accepts only the canonical snake_case
  identity, provenance and time keys recursively. No aliases or migration path
  are provided. Offers remain advisory; readiness and allocation stay fail closed.
- Target profiles adopt `thermal.measurement.humidity`,
  `thermal.setpoint.temperature_setpoint` and
  `thermal.setpoint.dhw_temperature_setpoint` with their existing quantities and
  qualification boundaries, plus the coordinated lowercase native-page URLs.
- This class S artifact change requires renewal of affected security, hardware
  and candidate evidence before release; synthetic checks do not qualify devices.
