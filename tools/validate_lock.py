#!/usr/bin/env python3
"""Strictly validate the Matter SDK lock and optional checked-out evidence."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
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


SHA_PIN = re.compile(r"^[^@\s'\"]+@[0-9a-f]{40}$")
DIGEST_PIN = re.compile(r"^docker://[^@\s'\"]+@sha256:[0-9a-f]{64}$")
FLOW_END = ",}]"


class WorkflowSyntaxError(ValueError):
    """The workflow uses YAML this fail-closed scanner does not understand."""


class _UsesScanner:
    """Quote-aware scanner for the YAML subset in workflows; reports values of mapping keys named `uses`."""

    def __init__(self, text: str) -> None:
        self.text = text
        self.n = len(text)
        self.found: list[tuple[int, str]] = []

    def line_of(self, i: int) -> int:
        return self.text.count("\n", 0, i) + 1

    def quoted(self, i: int) -> tuple[str, int]:
        quote = self.text[i]
        j = i + 1
        out: list[str] = []
        while j < self.n:
            c = self.text[j]
            if quote == '"' and c == "\\":
                decoded, j = self.escape(j)
                out.append(decoded)
                continue
            if c == quote:
                if quote == "'" and self.text[j + 1 : j + 2] == "'":
                    out.append("'")
                    j += 2
                    continue
                return "".join(out), j + 1
            out.append(c)
            j += 1
        raise WorkflowSyntaxError(f"line {self.line_of(i)}: unterminated quoted string")

    ESCAPES = {"0": "\0", "a": "\a", "b": "\b", "t": "\t", "\t": "\t", "n": "\n", "v": "\v", "f": "\f", "r": "\r",
               "e": "\x1b", " ": " ", '"': '"', "/": "/", "\\": "\\", "N": "\x85", "_": "\xa0",
               "L": "\u2028", "P": "\u2029"}

    def escape(self, j: int) -> tuple[str, int]:
        """Decode the YAML double-quoted escape that starts at the backslash at j."""
        code = self.text[j + 1 : j + 2]
        if code in self.ESCAPES:
            return self.ESCAPES[code], j + 2
        width = {"x": 2, "u": 4, "U": 8}.get(code)
        digits = self.text[j + 2 : j + 2 + width] if width else ""
        if width and len(digits) == width and all(d in "0123456789abcdefABCDEF" for d in digits):
            try:
                return chr(int(digits, 16)), j + 2 + width
            except (ValueError, OverflowError):
                pass
        if code in ("\n", "\r"):
            k = j + 1
            while k < self.n and self.text[k] in " \t\r\n":
                k += 1
            return "", k
        raise WorkflowSyntaxError(f"line {self.line_of(j)}: unsupported escape in quoted string")

    def at_colon(self, j: int, flow: bool, adjacent: bool = False) -> bool:
        if self.text[j : j + 1] != ":":
            return False
        if adjacent:
            return True
        nxt = self.text[j + 1 : j + 2]
        return nxt == "" or nxt in " \t\r\n" or (flow and nxt in FLOW_END)

    def plain(self, i: int, flow: bool) -> tuple[str, int]:
        j = i
        while j < self.n:
            c = self.text[j]
            if c == "\n" or (c == "#" and j > i and self.text[j - 1] in " \t"):
                break
            if flow and c in FLOW_END:
                break
            if self.at_colon(j, flow):
                break
            j += 1
        return self.text[i:j].strip(), j

    def scalar(self, i: int, flow: bool) -> tuple[str, int]:
        if self.text[i] in "'\"":
            return self.quoted(i)
        return self.plain(i, flow)

    def skip(self, i: int, newlines: bool) -> int:
        while i < self.n:
            c = self.text[i]
            if c in " \t\r" or (newlines and c == "\n"):
                i += 1
            elif c == "#" and (i == 0 or self.text[i - 1] in " \t\r\n"):
                while i < self.n and self.text[i] != "\n":
                    i += 1
            else:
                break
        return i

    def props(self, i: int, newlines: bool) -> int:
        """Skip anchors, tags and the explicit-key indicator that may precede a key."""
        while i < self.n and (
            self.text[i] in "&!" or (self.text[i] == "?" and self.text[i + 1 : i + 2] in ("", " ", "\t", "\n"))
        ):
            while i < self.n and self.text[i] not in " \t\r\n":
                i += 1
            i = self.skip(i, newlines)
        return i

    def record(self, key: str, value: str, at: int) -> None:
        if key == "uses":
            self.found.append((self.line_of(at), value))

    def flow(self, i: int) -> int:
        close = "}" if self.text[i] == "{" else "]"
        i += 1
        while True:
            i = self.skip(i, True)
            if i >= self.n:
                raise WorkflowSyntaxError("unterminated flow collection")
            if self.text[i] == close:
                return i + 1
            if self.text[i] == ",":
                i += 1
                continue
            i = self.entry(i, True)

    def entry(self, i: int, flow: bool) -> int:
        """Parse `key: value` or a bare item starting at i inside a flow collection."""
        i = self.props(i, True)
        if i >= self.n:
            raise WorkflowSyntaxError("unterminated flow collection")
        if self.text[i] in "{[":
            return self.flow(i)
        key, j = self.scalar(i, True)
        k = self.skip(j, True)
        if not self.at_colon(k, True, adjacent=self.text[i] in "'\""):
            if j == i:
                raise WorkflowSyntaxError(f"line {self.line_of(i)}: unparseable flow entry")
            return j
        k = self.skip(k + 1, True)
        if k < self.n and self.text[k] in "{[":
            self.record(key, "<collection>", i)
            return self.flow(k)
        value, end = (self.scalar(k, True) if k < self.n and self.text[k] not in FLOW_END else ("", k))
        self.record(key, value, i)
        return end

    def run(self) -> list[tuple[int, str]]:
        i = 0
        skip_over = -1  # block-scalar body: skip lines indented deeper than this column
        while i < self.n:
            eol = self.text.find("\n", i)
            eol = self.n if eol < 0 else eol
            line = self.text[i:eol]
            stripped = line.strip()
            indent = len(line) - len(line.lstrip(" \t"))
            if skip_over >= 0:
                if not stripped or indent > skip_over:
                    i = eol + 1
                    continue
                skip_over = -1
            if not stripped or stripped.startswith("#") or stripped in ("---", "..."):
                i = eol + 1
                continue
            col = i + indent
            while self.text[col : col + 1] == "-" and (col + 1 >= eol or self.text[col + 1] in " \t"):
                col = self.skip(col + 1, False)
            if self.text[col] == "?" and self.text[col + 1 : col + 2] in ("", " ", "\t", "\n"):
                raise WorkflowSyntaxError(f"line {self.line_of(col)}: block explicit keys are not supported")
            col = self.props(col, False)
            if col >= eol:
                i = eol + 1
                continue
            end = eol
            if self.text[col] in "{[":
                end = self.flow(col)
            else:
                key, j = self.scalar(col, False)
                j = self.skip(j, False)
                if self.at_colon(j, False, adjacent=self.text[col] in "'\""):
                    v = self.skip(j + 1, False)
                    if v >= eol:
                        self.record(key, "", col)
                    elif self.text[v] in "|>":
                        skip_over = col - i
                        self.record(key, "", col)
                    elif self.text[v] in "{[":
                        self.record(key, "<collection>", col)
                        end = self.flow(v)
                    else:
                        value, end = self.scalar(v, False)
                        self.record(key, value, col)
            nl = self.text.find("\n", end)
            i = self.n if nl < 0 else nl + 1
        return self.found


def validate_workflow_pins(directory: Path) -> list[str]:
    """Require every mapping key named `uses` (jobs.*.uses and steps[].uses, any YAML form) to be immutable.

    A ref must be a full commit SHA or image digest, or a local `./` path. Values of other keys are never
    inspected. Unparseable workflow YAML fails closed. No YAML dependency is added to this repository.
    """
    reasons: list[str] = []
    files = sorted([*directory.glob("*.yml"), *directory.glob("*.yaml")]) if directory.is_dir() else []
    if not files:
        return [f"no workflow files found in {directory}"]
    for path in files:
        try:
            found = _UsesScanner(path.read_text(encoding="utf-8")).run()
        except (WorkflowSyntaxError, OSError, UnicodeDecodeError) as exc:
            reasons.append(f"{path.name}: cannot scan workflow for action pins: {exc}")
            continue
        for number, value in found:
            if value.startswith("./") or SHA_PIN.match(value) or DIGEST_PIN.match(value):
                continue
            reasons.append(f"{path.name}:{number} action is not pinned by immutable commit SHA: {value or '<empty>'}")
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
    reasons.extend(validate_workflow_pins(ROOT / ".github" / "workflows"))
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
