# Contributing

Use an issue-scoped branch from the repository default branch and preserve other
contributors' work. Keep a change within the issue acceptance criteria. Before
opening a PR, run:

```bash
python3 tools/validate_lock.py
python3 -m unittest discover -s tests -v
```

The Linux SDK workflow is required before merge when a change affects the lock,
fixture, target contract, or readiness logic. Record commands and outcomes in
the PR. Request an independent review of the exact final head; resolve P0-P2
findings before merge.

Workflow authors must stay within the YAML subset that `tools/validate_lock.py` accepts (block mappings and sequences, plain or quoted scalars, block scalars, and property-free flow collections; no anchors, aliases, tags, merge or explicit keys, BOM, or extra document markers), because the workflow action-pin check rejects anything else.

Do not add credentials, fabric storage, attestation material, private captures,
personal device identifiers, network coordinates, or a real-device test result.
No change authorizes production commissioning, deployment, endpoint allocation,
or a live Gateway operation.

For go-fabric consumption, local patches and defect tracking, follow the
[upstream contribution workflow](docs/adoption/upstream-contribution-workflow.md).
SDK adoption does not authorize an upstream contribution: owner CLA review and
per-contribution authorization remain pending. Issue #67 supplies process material
only; do not contact the maintainer or prepare an actual submission in this issue.
