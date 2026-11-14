"""Ratchet the repository's Python naming, complexity and statement-count debt.

Ruff runs with ruff.toml as its only configuration, with suppression comments
ignored, over every Python file it discovers from the repository root. Each
finding is keyed by repository-relative path, rule and qualified symbol. The
measured value is Ruff's observed complexity (C901) or statement count
(PLR0915), or the number of naming findings (N) in that symbol.

--check fails on a new key, on a measured value above its baseline record and
on a baseline record that no longer matches a finding. --regenerate rewrites the
baseline from the current tree; review every baseline change by hand.
"""

from __future__ import annotations

import argparse
import ast
import json
import re
import subprocess
import sys
import tomllib
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[1]
BASELINE = "scripts/python-rules-baseline.json"
CONFIG = "ruff.toml"
VERSION = "0.15.10"
RULES = ["N", "C90", "PLR0915"]
THRESHOLDS = {"max-complexity": 10, "max-statements": 50}
MEASURED_RULES = {"C901", "PLR0915"}
REGENERATE = "python3 scripts/check_python_rules.py --regenerate"
RECORD_KEYS = ("path", "rule", "symbol", "measured")
DOCUMENT_KEYS = {"schema", "generator", "rules", "thresholds", "regenerate", "records"}
EXPECTED_CONFIG = {
    "required-version": "==" + VERSION,
    "target-version": "py311",
    "extend-exclude": [".tmp"],
    "lint": {
        "select": RULES,
        "mccabe": {"max-complexity": THRESHOLDS["max-complexity"]},
        "pylint": {"max-statements": THRESHOLDS["max-statements"]},
    },
}


def load_config(path: Path) -> dict[str, int]:
    """Accept only the reviewed configuration; anything else could hide findings."""
    try:
        config = tomllib.loads(path.read_text(encoding="utf-8"))
    except tomllib.TOMLDecodeError as exc:
        raise ValueError(f"{path.name} is not valid TOML") from exc
    if config != EXPECTED_CONFIG:
        raise ValueError(f"{path.name} differs from the reviewed rule selection and thresholds")
    return dict(THRESHOLDS)


def unique_object(pairs: list[tuple[str, object]]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def read_json(text: str) -> object:
    try:
        return json.loads(text, object_pairs_hook=unique_object)
    except (TypeError, json.JSONDecodeError) as exc:
        raise ValueError("malformed JSON") from exc


def known_rule(rule: object) -> bool:
    return isinstance(rule, str) and (rule in MEASURED_RULES or re.fullmatch(r"N\d{3}", rule) is not None)


def valid_path(path: object) -> bool:
    if not isinstance(path, str) or not path or "\\" in path:
        return False
    pure = PurePosixPath(path)
    return not pure.is_absolute() and ".." not in pure.parts and pure.as_posix() == path and path.endswith(".py")


def index(records: object) -> dict[tuple[str, str, str], int]:
    if not isinstance(records, list):
        raise ValueError("records must be an array")
    result: dict[tuple[str, str, str], int] = {}
    for item in records:
        if not isinstance(item, dict) or set(item) != set(RECORD_KEYS):
            raise ValueError("malformed record")
        path, rule, symbol, measured = (item[key] for key in RECORD_KEYS)
        valid_symbol = isinstance(symbol, str) and bool(symbol.strip())
        valid_measure = type(measured) is int and measured >= 1
        if not (valid_path(path) and known_rule(rule) and valid_symbol and valid_measure):
            raise ValueError(f"invalid record identity or measured value: {item}")
        key = (path, rule, symbol)
        if key in result:
            raise ValueError(f"duplicate record: {key}")
        result[key] = measured
    return result


def compare(current: object, baseline: object) -> None:
    observed, allowed = index(current), index(baseline)
    errors = [f"unused baseline record: {key}" for key in allowed.keys() - observed.keys()]
    for key, measured in observed.items():
        if key not in allowed or measured > allowed[key]:
            errors.append(f"new or increased debt: {key}, measured {measured}")
    if errors:
        raise ValueError("\n".join(sorted(errors)))


def baseline_document(records: list[dict[str, object]], thresholds: dict[str, int]) -> dict[str, object]:
    return {
        "schema": 1,
        "generator": {"tool": "ruff", "version": VERSION},
        "rules": list(RULES),
        "thresholds": dict(thresholds),
        "regenerate": REGENERATE,
        "records": records,
    }


def load_baseline(path: Path, thresholds: dict[str, int]) -> list[dict[str, object]]:
    document = read_json(path.read_text(encoding="utf-8"))
    if not isinstance(document, dict) or set(document) != DOCUMENT_KEYS:
        raise ValueError("unsupported baseline schema")
    expected = baseline_document(document["records"], thresholds)
    for key in sorted(DOCUMENT_KEYS - {"records"}):
        if document[key] != expected[key] or type(document[key]) is not type(expected[key]):
            raise ValueError(f"baseline {key} does not match the configured generator")
    keys = list(index(document["records"]))
    if keys != sorted(keys):
        raise ValueError("baseline records must be sorted by path, rule and symbol")
    return document["records"]


def parse_output(returncode: int, output: str) -> list[dict[str, object]]:
    if returncode not in (0, 1):
        raise ValueError(f"Ruff failed with exit {returncode}")
    diagnostics = read_json(output)
    if not isinstance(diagnostics, list):
        raise ValueError("Ruff output must be an array")
    if bool(diagnostics) != bool(returncode):
        raise ValueError("Ruff status disagrees with its diagnostics")
    for item in diagnostics:
        location = item.get("location") if isinstance(item, dict) else None
        fields = [item.get(key) for key in ("code", "filename", "message")] if isinstance(item, dict) else [None]
        row = location.get("row") if isinstance(location, dict) else None
        if not all(isinstance(field, str) for field in fields) or type(row) is not int or row < 1:
            raise ValueError("malformed Ruff diagnostic")
    return diagnostics


def ruff(arguments: list[str], root: Path, timeout: int) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(
            ["ruff", *arguments], cwd=root, capture_output=True, text=True, timeout=timeout, check=False
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ValueError(f"Ruff {VERSION} unavailable: {type(exc).__name__}") from exc


def checked_command(config: Path) -> list[str]:
    return ["check", ".", "--config", str(config), "--no-cache", "--no-fix"]


def require_version(root: Path) -> None:
    version = ruff(["--version"], root, 30)
    if version.returncode or version.stdout.strip() != f"ruff {VERSION}":
        raise ValueError(f"Ruff {VERSION} is required")


def run_ruff(root: Path, config: Path) -> list[dict[str, object]]:
    require_version(root)
    result = ruff([*checked_command(config), "--ignore-noqa", "--output-format", "json"], root, 150)
    return parse_output(result.returncode, result.stdout)


def relative(root: Path, filename: str) -> str:
    try:
        return Path(filename).resolve().relative_to(root.resolve()).as_posix()
    except ValueError as exc:
        raise ValueError(f"Ruff path outside the repository: {filename}") from exc


def checked_files(root: Path, config: Path) -> list[str]:
    require_version(root)
    result = ruff([*checked_command(config), "--show-files"], root, 150)
    if result.returncode:
        raise ValueError(f"Ruff failed with exit {result.returncode}")
    return sorted(relative(root, line) for line in result.stdout.splitlines() if line.strip())


def symbol_at(source: str, row: int) -> str:
    names: list[str] = []

    def visit(node: ast.AST) -> None:
        for child in ast.iter_child_nodes(node):
            if isinstance(child, (ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)):
                if child.lineno <= row <= child.end_lineno:
                    names.append(child.name)
                    visit(child)
            else:
                visit(child)

    visit(ast.parse(source))
    return ".".join(names) or "<module>"


def measured_value(rule: str, message: str) -> int:
    if rule not in MEASURED_RULES:
        return 1
    match = re.search(r"\((\d+) > \d+\)", message)
    if not match:
        raise ValueError(f"missing measured value in {rule} finding")
    return int(match[1])


def collect(diagnostics: list[dict[str, object]], root: Path) -> list[dict[str, object]]:
    totals: dict[tuple[str, str, str], int] = {}
    for item in diagnostics:
        rule = item["code"]
        if not known_rule(rule):
            raise ValueError(f"unexpected Ruff rule: {rule}")
        path = relative(root, item["filename"])
        source = (root / path).read_text(encoding="utf-8")
        row = item["location"]["row"]
        if row > len(source.splitlines()):
            raise ValueError("Ruff row outside the source")
        key = (path, rule, symbol_at(source, row))
        totals[key] = totals.get(key, 0) + measured_value(rule, item["message"])
    return [dict(zip(RECORD_KEYS, (*key, value))) for key, value in sorted(totals.items())]


def check(root: Path, baseline_path: Path) -> int:
    thresholds = load_config(root / CONFIG)
    baseline = load_baseline(baseline_path, thresholds)
    records = collect(run_ruff(root, root / CONFIG), root)
    compare(records, baseline)
    return len(records)


def write_baseline(root: Path, baseline_path: Path) -> int:
    thresholds = load_config(root / CONFIG)
    records = collect(run_ruff(root, root / CONFIG), root)
    text = json.dumps(baseline_document(records, thresholds), indent=2) + "\n"
    baseline_path.write_text(text, encoding="utf-8")
    return len(records)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true", help="compare the tree with the baseline")
    mode.add_argument("--regenerate", action="store_true", help="rewrite the baseline from the tree")
    args = parser.parse_args()
    try:
        if args.regenerate:
            count = write_baseline(ROOT, ROOT / BASELINE)
            print(f"Python rule baseline regenerated: {count} records")
        else:
            count = check(ROOT, ROOT / BASELINE)
            print(f"Python rule check passed: {count} recorded debt records")
    except (ValueError, OSError, SyntaxError) as exc:
        print(f"Python rule check failed: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
