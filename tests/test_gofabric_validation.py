"""Guard the hosted adapter command selection independently of SDK fixtures."""

from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
STEP = "Vet and test the adopted Go adapter in module mode"
COMMANDS = (
    "go vet ./...",
    "go test -race -count=1 -timeout=5m ./...",
    "go vet -tags gofabric ./...",
    "go test -race -tags gofabric -count=1 -timeout=5m ./...",
)


def validate_adapter_step(workflow: str) -> None:
    """Require an unconditional module-mode step with both build configurations.

    Deliberately accept only this small literal step shape: expressions, shell
    indirection, conditions and error suppression need explicit contract review.
    """
    steps = re.findall(r"^      - name: (.*?)(?=^      - |\Z)", workflow, re.M | re.S)
    matches = [step for step in steps if step.splitlines()[0] == STEP]
    if len(matches) != 1:
        raise ValueError("expected one adapter validation step")
    lines = matches[0].splitlines()[1:]
    expected = [
        "        env:",
        '          GOWORK: "off"',
        '          GO111MODULE: "on"',
        "        run: |",
        *("          " + command for command in COMMANDS),
    ]
    if lines != expected:
        raise ValueError("adapter step must execute both configurations in module mode")


class GoFabricValidationTest(unittest.TestCase):
    def setUp(self) -> None:
        self.workflow = (ROOT / ".github/workflows/sdk-baseline.yml").read_text()

    def test_hosted_adapter_commands(self) -> None:
        validate_adapter_step(self.workflow)

    def test_each_command_is_required(self) -> None:
        for command in COMMANDS:
            with self.subTest(command=command):
                changed = self.workflow.replace("          " + command + "\n", "", 1)
                with self.assertRaises(ValueError):
                    validate_adapter_step(changed)

    def test_each_tagged_command_requires_the_tag(self) -> None:
        for command in COMMANDS[2:]:
            with self.subTest(command=command):
                changed = self.workflow.replace(command, command.replace(" -tags gofabric", ""), 1)
                with self.assertRaises(ValueError):
                    validate_adapter_step(changed)

    def test_module_and_workspace_settings_are_required(self) -> None:
        for setting, replacement in (
            ('          GOWORK: "off"\n', ""),
            ('          GOWORK: "off"', '          GOWORK: "auto"'),
            ('          GO111MODULE: "on"\n', ""),
            ('          GO111MODULE: "on"', '          GO111MODULE: "off"'),
        ):
            with self.subTest(setting=setting, replacement=replacement):
                with self.assertRaises(ValueError):
                    validate_adapter_step(self.workflow.replace(setting, replacement, 1))

    def test_legacy_only_and_compile_only_substitutions_fail(self) -> None:
        for replacement in (
            "GO111MODULE=off go test -race -count=1 ./...",
            "scripts/run_upstream_fixture.sh sdk",
            "go test -tags gofabric -c ./sdkfabric",
            "go test -race -tags gofabric -run '^$' ./...",
        ):
            with self.subTest(replacement=replacement):
                with self.assertRaises(ValueError):
                    validate_adapter_step(self.workflow.replace(COMMANDS[-1], replacement, 1))

    def test_comments_conditions_and_failure_suppression_are_not_execution(self) -> None:
        for changed in (
            self.workflow.replace(COMMANDS[-1], "# " + COMMANDS[-1], 1),
            self.workflow.replace(COMMANDS[-1], COMMANDS[-1] + " || true", 1),
            self.workflow.replace("        env:\n", "        if: false\n        env:\n", 1),
            self.workflow.replace("        env:\n", "        continue-on-error: true\n        env:\n", 1),
            self.workflow.replace(STEP, "Legacy SDK checks", 1),
        ):
            with self.subTest(changed=changed):
                with self.assertRaises(ValueError):
                    validate_adapter_step(changed)

    def test_duplicate_step_fails(self) -> None:
        start = self.workflow.index("      - name: " + STEP)
        end = self.workflow.index("      - name: Install public Linux", start)
        with self.assertRaises(ValueError):
            validate_adapter_step(self.workflow + self.workflow[start:end])

    def test_validation_policy_names_actual_commands(self) -> None:
        policy = (ROOT / "AGENTS.md").read_text()
        self.assertIn("export GOWORK=off GO111MODULE=on", policy)
        for command in COMMANDS:
            self.assertIn(command, policy)
        self.assertIn("only compiles", policy)
        self.assertIn("Hosted full tagged execution remains required", policy)


if __name__ == "__main__":
    unittest.main()
