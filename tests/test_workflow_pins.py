"""Fail-closed tests for immutable GitHub Actions pins."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))

from validate_lock import validate_workflow_pins  # noqa: E402

SHA = "11d5960a326750d5838078e36cf38b85af677262"


def check(*uses_lines: str) -> list[str]:
    with tempfile.TemporaryDirectory() as temporary:
        directory = Path(temporary)
        body = "jobs:\n  j:\n    steps:\n" + "".join(f"      {line}\n" for line in uses_lines)
        (directory / "w.yml").write_text(body, encoding="utf-8")
        return validate_workflow_pins(directory)


class WorkflowPinTest(unittest.TestCase):
    def test_repository_workflows_are_pinned(self) -> None:
        self.assertEqual(validate_workflow_pins(ROOT / ".github" / "workflows"), [])

    def test_sha_pin_with_version_comment_passes(self) -> None:
        self.assertEqual(check(f"- uses: actions/checkout@{SHA} # v4.4.0"), [])

    def test_local_action_passes(self) -> None:
        self.assertEqual(check("- uses: ./.github/actions/local"), [])

    def test_mutable_refs_fail(self) -> None:
        for ref in ("v4", "v4.4.0", "main", "11d5960", SHA.upper(), SHA[:-1], SHA + "0"):
            with self.subTest(ref=ref):
                self.assertTrue(check(f"- uses: actions/checkout@{ref}"))

    def test_missing_ref_fails(self) -> None:
        self.assertTrue(check("- uses: actions/checkout"))

    def test_quoted_and_unusual_forms_fail(self) -> None:
        self.assertTrue(check("- uses: 'actions/checkout@v4'"))
        self.assertTrue(check('- "uses": actions/checkout@v4'))
        self.assertTrue(check("uses: docker://alpine:3.19"))

    def test_missing_workflow_directory_fails(self) -> None:
        self.assertTrue(validate_workflow_pins(ROOT / "no-such-dir"))


if __name__ == "__main__":
    unittest.main()
