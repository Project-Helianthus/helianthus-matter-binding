# Changelog

## Unreleased

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
