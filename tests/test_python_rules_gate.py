"""Fail-closed tests for the Python naming, complexity and statement-count ratchet."""

from __future__ import annotations

import importlib.util
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("check_python_rules", ROOT / "scripts" / "check_python_rules.py")
gate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(gate)


def record(**changes: object) -> dict[str, object]:
    return {"path": "pkg/sample.py", "rule": "C901", "symbol": "work", "measured": 12} | changes


def baseline_payload(records: list[dict[str, object]]) -> dict[str, object]:
    return gate.baseline_document(records, gate.THRESHOLDS)


def complex_function(name: str, branches: int) -> str:
    body = "".join(f"    if value == {index}:\n        return {index}\n" for index in range(branches))
    return f"def {name}(value):\n{body}    return -1\n"


def long_function(name: str, statements: int) -> str:
    body = "".join(f"    item_{index} = {index}\n" for index in range(statements))
    return f"def {name}():\n{body}    return 0\n"


class CompareTest(unittest.TestCase):
    """The ratchet itself, over synthetic records."""

    def test_clean_tree_and_recorded_debt_pass(self) -> None:
        gate.compare([], [])
        gate.compare([record()], [record()])

    def test_new_naming_violation_fails_in_file_with_recorded_debt(self) -> None:
        with self.assertRaisesRegex(ValueError, "new or increased"):
            gate.compare([record(), record(rule="N802", symbol="doWork", measured=1)], [record()])

    def test_new_complex_function_fails_in_file_with_recorded_debt(self) -> None:
        with self.assertRaisesRegex(ValueError, "new or increased"):
            gate.compare([record(), record(symbol="other", measured=11)], [record()])

    def test_new_statement_count_violation_fails_in_file_with_recorded_debt(self) -> None:
        with self.assertRaisesRegex(ValueError, "new or increased"):
            gate.compare([record(), record(rule="PLR0915", measured=51)], [record()])

    def test_grown_measured_value_fails(self) -> None:
        with self.assertRaisesRegex(ValueError, "new or increased"):
            gate.compare([record(measured=13)], [record()])

    def test_reduced_measured_value_passes(self) -> None:
        gate.compare([record(measured=11)], [record()])

    def test_removed_violation_must_leave_the_baseline(self) -> None:
        with self.assertRaisesRegex(ValueError, "unused baseline record"):
            gate.compare([], [record()])

    def test_every_identity_field_is_enforced(self) -> None:
        # path: keying without the path would let debt move to another file.
        # rule: keying without the rule would let C901 debt cover a PLR0915 violation.
        # symbol: keying without the symbol would let one function's debt cover another.
        for changes in ({"path": "pkg/other.py"}, {"rule": "PLR0915"}, {"symbol": "Holder.work"}):
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                gate.compare([record(**changes)], [record()])

    def test_duplicate_records_fail(self) -> None:
        for current, baseline in (([record()] * 2, [record()]), ([record()], [record()] * 2)):
            with self.subTest(current=current, baseline=baseline), self.assertRaises(ValueError):
                gate.compare(current, baseline)

    def test_malformed_records_fail(self) -> None:
        for changes in (
            {"path": "../sample.py"},
            {"path": "/sample.py"},
            {"path": "pkg\\sample.py"},
            {"path": "pkg/./sample.py"},
            {"path": "pkg/sample.txt"},
            {"symbol": " "},
            {"rule": "F401"},
            {"rule": "N8021"},
            {"measured": True},
            {"measured": 0},
            {"measured": 1.5},
        ):
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                gate.compare([record(**changes)], [])
        extra = record() | {"note": "x"}
        missing = {key: value for key, value in record().items() if key != "measured"}
        for item in (extra, missing, "record"):
            with self.subTest(item=item), self.assertRaises(ValueError):
                gate.compare([item], [])


class BaselineDocumentTest(unittest.TestCase):
    """The committed baseline must state how and under which settings it was generated."""

    def load(self, payload: object) -> list[dict[str, object]]:
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "baseline.json"
            path.write_text(payload if isinstance(payload, str) else json.dumps(payload), encoding="utf-8")
            return gate.load_baseline(path, gate.THRESHOLDS)

    def test_generated_document_round_trips(self) -> None:
        self.assertEqual(self.load(baseline_payload([record()])), [record()])

    def test_document_records_generator_thresholds_and_regeneration(self) -> None:
        document = baseline_payload([])
        self.assertEqual(document["generator"], {"tool": "ruff", "version": "0.15.10"})
        self.assertEqual(document["rules"], ["N", "C90", "PLR0915"])
        self.assertEqual(document["thresholds"], {"max-complexity": 10, "max-statements": 50})
        self.assertEqual(document["regenerate"], "python3 scripts/check_python_rules.py --regenerate")

    def test_stale_or_altered_metadata_fails(self) -> None:
        for key, value in (
            ("schema", 2),
            ("generator", {"tool": "ruff", "version": "0.15.9"}),
            ("rules", ["N", "C90"]),
            ("thresholds", {"max-complexity": 12, "max-statements": 50}),
            ("thresholds", {"max-complexity": 10, "max-statements": 60}),
            ("regenerate", "ruff check ."),
        ):
            with self.subTest(key=key, value=value), self.assertRaises(ValueError):
                self.load(baseline_payload([]) | {key: value})

    def test_malformed_documents_fail(self) -> None:
        for text in (
            "not json",
            "[]",
            "{}",
            json.dumps(baseline_payload([]) | {"extra": 0}),
            json.dumps(baseline_payload([]) | {"schema": True}),
            json.dumps(baseline_payload([]) | {"records": {}}),
            json.dumps(baseline_payload([])).replace('"records"', '"records": [], "records"', 1),
        ):
            with self.subTest(text=text), self.assertRaises(ValueError):
                self.load(text)

    def test_records_must_be_sorted(self) -> None:
        records = [record(symbol="work"), record(symbol="alpha")]
        with self.assertRaisesRegex(ValueError, "sorted"):
            self.load(baseline_payload([]) | {"records": records})


class ConfigurationTest(unittest.TestCase):
    """ruff.toml may only carry the reviewed rule selection and thresholds."""

    def load(self, text: str) -> dict[str, int]:
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "ruff.toml"
            path.write_text(text, encoding="utf-8")
            return gate.load_config(path)

    def test_repository_configuration_is_accepted(self) -> None:
        self.assertEqual(gate.load_config(ROOT / "ruff.toml"), gate.THRESHOLDS)

    def test_widened_or_suppressing_configuration_fails(self) -> None:
        text = (ROOT / "ruff.toml").read_text(encoding="utf-8")
        for changed in (
            text.replace("max-complexity = 10", "max-complexity = 11"),
            text.replace("max-statements = 50", "max-statements = 80"),
            text.replace('"PLR0915"]', '"PLR0915", "E"]'),
            text.replace('select = ["N", "C90", "PLR0915"]', 'select = ["N", "C90"]'),
            text.replace('"==0.15.10"', '">=0.15"'),
            text.replace('extend-exclude = [".tmp"]', 'extend-exclude = [".tmp", "tools"]'),
            text + '\n[lint.per-file-ignores]\n"tools/*" = ["C901"]\n',
            text + '\n[lint.pep8-naming]\nignore-names = ["*"]\n',
            "[lint\n",
        ):
            with self.subTest(changed=changed), self.assertRaises(ValueError):
                self.load(changed)


class RuffOutputTest(unittest.TestCase):
    """Tool failures, malformed output and foreign rules fail closed."""

    def test_tool_failure_fails_even_with_json(self) -> None:
        with self.assertRaises(ValueError):
            gate.parse_output(2, "[]")

    def test_status_must_agree_with_diagnostics(self) -> None:
        diagnostic = {"code": "N802", "filename": "a.py", "message": "m", "location": {"row": 1}}
        for status, diagnostics in ((0, [diagnostic]), (1, [])):
            with self.subTest(status=status), self.assertRaises(ValueError):
                gate.parse_output(status, json.dumps(diagnostics))

    def test_malformed_output_fails(self) -> None:
        for value in ("not json", "{}", "[{}]", '[{"code": null}]', '[{"code": "N802", "filename": "a.py", '
                      '"message": "m", "location": {"row": 0}}]'):
            with self.subTest(value=value), self.assertRaises(ValueError):
                gate.parse_output(1, value)

    def test_missing_wrong_or_hung_tool_fails(self) -> None:
        for effect in (
            FileNotFoundError(),
            subprocess.TimeoutExpired("ruff", 30),
        ):
            with self.subTest(effect=effect), patch.object(gate.subprocess, "run", side_effect=effect):
                with self.assertRaises(ValueError):
                    gate.run_ruff(ROOT, ROOT / "ruff.toml")
        for result in (
            subprocess.CompletedProcess([], 0, "ruff 0.15.9\n", ""),
            subprocess.CompletedProcess([], 1, "ruff 0.15.10\n", ""),
        ):
            with self.subTest(result=result), patch.object(gate.subprocess, "run", return_value=result):
                with self.assertRaises(ValueError):
                    gate.run_ruff(ROOT, ROOT / "ruff.toml")

    def test_measured_values_and_naming_occurrences(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary).resolve()
            (root / "pkg").mkdir()
            path = root / "pkg" / "sample.py"
            path.write_text("def work():\n    pass\n", encoding="utf-8")
            diagnostic = {
                "code": "C901",
                "filename": str(path),
                "location": {"row": 1},
                "message": "`work` is too complex (12 > 10)",
            }
            self.assertEqual(gate.collect([diagnostic], root), [record()])
            naming = diagnostic | {"code": "N806", "message": "Variable `myValue` in function should be lowercase"}
            self.assertEqual(gate.collect([naming, naming], root), [record(rule="N806", measured=2)])
            for changed in (
                diagnostic | {"code": "F401"},
                diagnostic | {"message": "`work` is too complex"},
                diagnostic | {"location": {"row": 3}},
                diagnostic | {"filename": str(Path(temporary).resolve().parent / "outside.py")},
            ):
                with self.subTest(changed=changed), self.assertRaises(ValueError):
                    gate.collect([changed], root)

    def test_symbols_are_qualified_and_position_independent(self) -> None:
        source = "def work():\n    pass\n"
        self.assertEqual(gate.symbol_at(source, 1), gate.symbol_at("\n\n" + source, 3))
        nested = "class Holder:\n    def work(self):\n        pass\n"
        self.assertEqual(gate.symbol_at(nested, 2), "Holder.work")
        self.assertEqual(gate.symbol_at("value = 1\n", 1), "<module>")


class RuffEndToEndTest(unittest.TestCase):
    """The pinned Ruff and the committed configuration detect each regression class."""

    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary.name).resolve()
        shutil.copyfile(ROOT / "ruff.toml", self.root / "ruff.toml")
        (self.root / "pkg").mkdir()
        self.source = self.root / "pkg" / "sample.py"
        self.baseline = self.root / "baseline.json"
        self.debt = complex_function("work", 11)
        self.source.write_text(self.debt, encoding="utf-8")
        gate.write_baseline(self.root, self.baseline)

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def check(self) -> int:
        return gate.check(self.root, self.baseline)

    def assert_rejected(self, text: str, rule: str) -> None:
        self.source.write_text(text, encoding="utf-8")
        with self.assertRaisesRegex(ValueError, f"new or increased debt: .*'{rule}'"):
            self.check()

    def test_recorded_debt_passes(self) -> None:
        self.assertEqual(gate.load_baseline(self.baseline, gate.THRESHOLDS), [record(measured=12)])
        self.assertEqual(self.check(), 1)

    def test_new_naming_violation_fails_in_file_with_recorded_debt(self) -> None:
        self.assert_rejected(self.debt + "\n\ndef doWork():\n    return 0\n", "N802")

    def test_new_complex_function_fails_in_file_with_recorded_debt(self) -> None:
        self.assert_rejected(self.debt + "\n\n" + complex_function("other", 11), "C901")

    def test_new_statement_count_violation_fails_in_file_with_recorded_debt(self) -> None:
        self.assert_rejected(self.debt + "\n\n" + long_function("other", 51), "PLR0915")

    def test_grown_measured_value_fails(self) -> None:
        self.assert_rejected(complex_function("work", 12), "C901")

    def test_suppression_comment_does_not_hide_a_violation(self) -> None:
        self.assert_rejected(self.debt + "\n\ndef doWork():  # noqa: N802\n    return 0\n", "N802")

    def test_nested_configuration_does_not_hide_a_violation(self) -> None:
        (self.root / "pkg" / "ruff.toml").write_text('[lint]\nselect = ["E"]\n', encoding="utf-8")
        self.assert_rejected(self.debt + "\n\ndef doWork():\n    return 0\n", "N802")

    def test_removed_violation_must_leave_the_baseline(self) -> None:
        self.source.write_text("def work(value):\n    return value\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "unused baseline record"):
            self.check()
        gate.write_baseline(self.root, self.baseline)
        self.assertEqual(self.check(), 0)


class RepositoryTest(unittest.TestCase):
    """The committed tree passes with its recorded debt and Ruff sees every Python directory."""

    def test_repository_passes_with_recorded_debt(self) -> None:
        records = gate.load_baseline(ROOT / gate.BASELINE, gate.THRESHOLDS)
        self.assertEqual(gate.check(ROOT, ROOT / gate.BASELINE), len(records))

    def test_ruff_checks_the_complete_python_inventory(self) -> None:
        checked = set(gate.checked_files(ROOT, ROOT / "ruff.toml"))
        inventory = {
            Path(directory, name).relative_to(ROOT).as_posix()
            for top in ("runtime", "scripts", "tests", "tools")
            for directory, _, names in os.walk(ROOT / top)
            for name in names
            if name.endswith(".py")
        }
        self.assertTrue({"scripts/check_python_rules.py", "tests/test_python_rules_gate.py"} <= inventory)
        self.assertLessEqual(inventory, checked)


if __name__ == "__main__":
    unittest.main()
