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

    def test_flow_and_nested_forms_fail(self) -> None:
        for line in (
            "- { uses: actions/checkout@v4 }",
            "steps: [{uses: actions/checkout@v4}]",
            "call: {uses: owner/repo/.github/workflows/x.yml@main}",
            '- { "uses": "actions/checkout@v4" }',
            "- {uses: 'actions/checkout@v4', name: x}",
            "- { uses: }",
            "- uses:",
        ):
            with self.subTest(line=line):
                self.assertTrue(check(line))

    def test_pinned_flow_and_reusable_workflow_pass(self) -> None:
        self.assertEqual(check(f"- {{ uses: actions/checkout@{SHA} }}"), [])
        self.assertEqual(check(f'- {{ "uses": "actions/checkout@{SHA}", name: x }}'), [])
        self.assertEqual(check(f"call: {{uses: owner/repo/.github/workflows/x.yml@{SHA}}}"), [])
        self.assertEqual(check(f"uses: owner/repo/.github/workflows/x.yml@{SHA}"), [])

    def test_quoted_hash_does_not_hide_uses(self) -> None:
        self.assertTrue(check('- {name: "x # y", uses: actions/checkout@v4}'))
        self.assertTrue(check("- {name: 'x # y', uses: actions/checkout@v4}"))
        self.assertTrue(check('call: {name: "a # b", uses: owner/repo/.github/workflows/x.yml@main}'))
        self.assertEqual(check(f'- {{name: "x # y", uses: actions/checkout@{SHA}}}'), [])

    def test_values_of_other_keys_are_never_inspected(self) -> None:
        self.assertEqual(check('- run: echo "uses: actions/checkout@v4"'), [])
        self.assertEqual(check("- name: uses: actions/checkout@v4"), [])
        self.assertEqual(check("- run: |", "    uses: actions/checkout@v4", "    echo ok"), [])
        self.assertEqual(check("- {run: 'echo uses: a@v4', name: x}"), [])

    def test_multiline_flow_mapping(self) -> None:
        self.assertEqual(check("- {", f"    uses: actions/checkout@{SHA},", "    name: x", "  }"), [])
        self.assertEqual(check("- {", f"    uses:", f"      actions/checkout@{SHA}", "  }"), [])
        self.assertTrue(check("- {", "    name: x,", "    uses: actions/checkout@v4", "  }"))
        self.assertTrue(check("- [", "    {uses: actions/checkout@main}", "  ]"))

    def test_mutable_reusable_workflow_flow_form_fails(self) -> None:
        self.assertTrue(check("j: { uses: owner/repo/.github/workflows/x.yml@v1, with: {a: 1} }"))

    def test_unterminated_input_fails_closed(self) -> None:
        self.assertTrue(check('- { name: "oops, uses: actions/checkout@v4 }'))
        self.assertTrue(check("- { uses: actions/checkout@v4"))

    def test_json_style_and_escaped_keys(self) -> None:
        for line in (
            '- {"uses":"actions/checkout@v4"}',
            '- {"uses":actions/checkout@v4}',
            "- {'uses':'actions/checkout@v4'}",
            '- {"uses":{}}',
            '- {"\\u0075ses": "actions/checkout@v4"}',
            '- {"u\\x73es": "actions/checkout@v4"}',
            '- {"\\165ses": "actions/checkout@v4"}',
            '- "uses":"actions/checkout@v4"',
            '- "\\u0075ses": actions/checkout@v4',
        ):
            with self.subTest(line=line):
                self.assertTrue(check(line))

    def test_json_style_pinned_passes(self) -> None:
        self.assertEqual(check(f'- {{"uses":"actions/checkout@{SHA}","name":"x"}}'), [])
        self.assertEqual(check(f'- {{"\\u0075ses": "actions/checkout@{SHA}"}}'), [])

    def test_adversarial_shapes(self) -> None:
        self.assertTrue(check("- [{a: [{b: {uses: actions/checkout@v4}}]}]"))
        self.assertTrue(check("- uses   : actions/checkout@v4"))
        self.assertTrue(check("- { uses   : actions/checkout@v4 }"))
        self.assertTrue(check("\t- uses: actions/checkout@v4"))
        self.assertTrue(check("- {a: 1,\tuses: actions/checkout@v4}"))
        self.assertEqual(check(f"- uses   : actions/checkout@{SHA}"), [])

    def test_anchor_tag_and_explicit_key_prefixes(self) -> None:
        for line in ("- &a uses: actions/checkout@v4", "- !!str uses: actions/checkout@v4", "- ? uses\n  : actions/checkout@v4",
                     "- {&a uses: actions/checkout@v4}", "- {? uses : actions/checkout@v4}"):
            with self.subTest(line=line):
                self.assertTrue(check(line))

    def test_unsupported_constructs_are_rejected_not_skipped(self) -> None:
        pinned = f"actions/checkout@{SHA}"
        cases = {
            "bom": "\ufeffjobs: {}\n",
            "bom with root flow": "\ufeff{jobs: {j: {steps: [{uses: actions/checkout@v4}]}}}\n",
            "anchor on flow value": "steps: &steps [{uses: actions/checkout@v4}]\n",
            "anchor in merge": "- <<: &base {uses: actions/checkout@v4}\n",
            "anchor on scalar": f"- uses: &a {pinned}\n",
            "alias": "a: *base\n",
            "alias item": "- *base\n",
            "tag": f"- uses: !!str {pinned}\n",
            "tag on key": f"- !!str uses: {pinned}\n",
            "merge key": "- <<: {uses: actions/checkout@v4}\n",
            "merge key alias": "- <<: *base\n",
            "explicit key": "- ? uses\n  : actions/checkout@v4\n",
            "explicit key in flow": "- {? uses : actions/checkout@v4}\n",
            "root flow mapping": "{jobs: {j: {steps: [{uses: actions/checkout@v4}]}}}\n",
            "root flow sequence": "[{uses: actions/checkout@v4}]\n",
            "free-standing flow": "steps:\n  [{uses: actions/checkout@v4}]\n",
            "directive": "%YAML 1.2\n---\na: 1\n",
            "second document marker": "a: 1\n---\nb: 2\n",
            "document end marker": "a: 1\n...\n",
            "content after marker": "--- {uses: actions/checkout@v4}\n",
            "tab indent": "a:\n\t- uses: " + pinned + "\n",
            "unterminated quote": 'a: "oops\n',
            "unterminated flow": "a: {uses: " + pinned + "\n",
        }
        for name, text in cases.items():
            with self.subTest(case=name):
                with tempfile.TemporaryDirectory() as temporary:
                    (Path(temporary) / "w.yml").write_text(text, encoding="utf-8")
                    reasons = validate_workflow_pins(Path(temporary))
                self.assertEqual(len(reasons), 1, reasons)
                self.assertIn("cannot scan workflow", reasons[0])

    def test_single_leading_document_marker_is_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            (Path(temporary) / "w.yml").write_text(f"---\nsteps:\n  - uses: actions/checkout@{SHA}\n", encoding="utf-8")
            self.assertEqual(validate_workflow_pins(Path(temporary)), [])

    def test_comment_mentions_are_ignored(self) -> None:
        self.assertEqual(check("# uses: actions/checkout@v4", f"- uses: a/b@{SHA} # uses: x@v1"), [])

    def test_missing_workflow_directory_fails(self) -> None:
        self.assertTrue(validate_workflow_pins(ROOT / "no-such-dir"))


if __name__ == "__main__":
    unittest.main()
