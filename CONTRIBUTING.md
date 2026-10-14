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

Do not add credentials, fabric storage, attestation material, private captures,
personal device identifiers, network coordinates, or a real-device test result.
No change authorizes production commissioning, deployment, endpoint allocation,
or a live Gateway operation.
