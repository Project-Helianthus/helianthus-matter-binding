# Changelog

## Unreleased

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
