#!/usr/bin/env python3
"""Strictly validate the Matter SDK lock and optional checked-out evidence."""

from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from runtime.production_readiness import validate_lock  # noqa: E402


def git(checkout: Path, *args: str) -> str:
    result = subprocess.run(["git", "-C", str(checkout), *args], check=False, text=True, capture_output=True)
    if result.returncode:
        raise ValueError(result.stderr.strip() or "git validation failed")
    return result.stdout.strip()


def validate_checkout(lock: dict, checkout: Path) -> list[str]:
    reasons: list[str] = []
    try:
        if git(checkout, "rev-parse", "HEAD") != lock["sdk"]["commit"]:
            reasons.append("SDK checkout HEAD differs from locked commit")
        items = [lock["sdk"]["data_model"]["spec_sha"], lock["sdk"]["data_model"]["spec_tag"], *lock["sdk"]["blobs"]]
        for item in items:
            if git(checkout, "rev-parse", f"HEAD:{item['path']}") != item["blob"]:
                reasons.append(f"SDK blob differs: {item['path']}")
    except (OSError, ValueError) as exc:
        reasons.append(f"SDK checkout cannot be validated: {exc}")
    return reasons


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--lock", type=Path, default=ROOT / "sdk" / "matter-sdk.lock.json")
    parser.add_argument("--sdk-checkout", type=Path)
    parser.add_argument("--mapping", type=Path, help="Optional checked-out mapping file to hash")
    args = parser.parse_args()
    try:
        lock = json.loads(args.lock.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"FAIL: cannot read lock: {exc}", file=sys.stderr)
        return 1
    reasons = list(validate_lock(lock))
    if not reasons and args.sdk_checkout:
        reasons.extend(validate_checkout(lock, args.sdk_checkout))
    if not reasons and args.mapping:
        observed = hashlib.sha256(args.mapping.read_bytes()).hexdigest()
        if observed != lock["mapping"]["sha256"]:
            reasons.append("mapping SHA-256 differs from lock")
    if reasons:
        print("FAIL: " + "; ".join(reasons), file=sys.stderr)
        return 1
    print("PASS: Matter SDK lock and requested evidence match the reviewed baseline")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
