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


ADAPTER_JOB = ("gofabric-adapter", "Go adapter (go-fabric, module mode)")
SDK_MARKERS = (
    "apt-get",
    "scripts/checkout_sdk.sh",
    "scripts/bootstrap.sh",
    "scripts/activate.sh",
    "build_examples.py",
    "out/focused-tests",
    "run_upstream_fixture.sh",
    "run_owned_group.sh",
    "connectedhomeip",
)


def split_jobs(workflow: str) -> dict[str, str]:
    """Return each job body keyed by job id, from the literal two-space job layout."""
    _, separator, body = workflow.partition("\njobs:\n")
    if not separator:
        raise ValueError("workflow has no jobs mapping")
    parts = re.split(r"^  ([A-Za-z0-9_-]+):\n", body, flags=re.M)
    if parts[0].strip():
        raise ValueError("unexpected content before the first job")
    jobs: dict[str, str] = {}
    for job_id, job_body in zip(parts[1::2], parts[2::2]):
        if job_id in jobs:
            raise ValueError("duplicate job id")
        jobs[job_id] = job_body
    return jobs


def job_name(job: str) -> str:
    names = re.findall(r"^    name: (.*)$", job, re.M)
    if len(names) != 1:
        raise ValueError("expected one job name")
    return names[0]


def job_steps(job: str) -> list[str]:
    return re.findall(r"^      - (.*?)(?=^      - |\Z)", job, re.M | re.S)


def validate_job_split(workflow: str) -> None:
    """Require only the adapter job, with no external SDK checkout step."""
    jobs = split_jobs(workflow)
    if set(jobs) != {ADAPTER_JOB[0]}:
        raise ValueError("expected only the adapter job")
    adapter = jobs[ADAPTER_JOB[0]]
    if job_name(adapter) != ADAPTER_JOB[1]:
        raise ValueError("job name must state what the adapter job runs")
    if re.findall(r"^    needs: (.*)$", adapter, re.M):
        raise ValueError("adapter job must not wait on another job")
    for step in job_steps(adapter):
        if any(marker in step for marker in SDK_MARKERS):
            raise ValueError("adapter job must not run an external SDK step")
    if any(marker in workflow for marker in SDK_MARKERS):
        raise ValueError("workflow must not select an external SDK checkout")


def validate_adapter_step(workflow: str) -> None:
    """Require an unconditional module-mode step with both build configurations.

    The step must live in the adapter job. Deliberately accept only this small
    literal step shape: expressions, shell indirection, conditions and error
    suppression need explicit contract review.
    """
    jobs = split_jobs(workflow)
    if ADAPTER_JOB[0] not in jobs:
        raise ValueError("adapter job is missing")
    for job_id, job in jobs.items():
        if job_id != ADAPTER_JOB[0] and any(step.splitlines()[0] == "name: " + STEP for step in job_steps(job)):
            raise ValueError("adapter step must not run outside the adapter job")
    steps = job_steps(jobs[ADAPTER_JOB[0]])
    matches = [step for step in steps if step.splitlines()[0] == "name: " + STEP]
    if len(matches) != 1:
        raise ValueError("expected one adapter validation step")
    lines = matches[0].rstrip("\n").splitlines()[1:]
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

    def test_external_fixture_and_compile_only_substitutions_fail(self) -> None:
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
            self.workflow.replace(STEP, "External SDK checks", 1),
        ):
            with self.subTest(changed=changed):
                with self.assertRaises(ValueError):
                    validate_adapter_step(changed)

    def test_duplicate_step_fails(self) -> None:
        start = self.workflow.index("      - name: " + STEP)
        step = self.workflow[start:]
        if not step.endswith("\n"):
            step += "\n"
        with self.assertRaises(ValueError):
            validate_adapter_step(self.workflow + step)

    def test_jobs_are_split_and_named(self) -> None:
        validate_job_split(self.workflow)
        jobs = split_jobs(self.workflow)
        self.assertEqual(set(jobs), {ADAPTER_JOB[0]})
        self.assertEqual(job_name(jobs[ADAPTER_JOB[0]]), ADAPTER_JOB[1])

    def test_each_job_id_and_name_is_required(self) -> None:
        job_id, name = ADAPTER_JOB
        for changed in (
            self.workflow.replace("\n  " + job_id + ":\n", "\n  renamed-job:\n", 1),
            self.workflow.replace("    name: " + name + "\n", "    name: Matter checks\n", 1),
            self.workflow.replace("    name: " + name + "\n", "", 1),
        ):
            with self.subTest(changed=changed):
                with self.assertRaises(ValueError):
                    validate_job_split(changed)

    def test_second_job_fails(self) -> None:
        extra = "\n  other-job:\n    name: Other checks\n    runs-on: ubuntu-24.04\n    steps:\n      - run: python3 tools/validate_lock.py\n"
        with self.assertRaises(ValueError):
            validate_job_split(self.workflow + extra)

    def test_adapter_job_contains_no_sdk_step(self) -> None:
        sdk_step = "      - name: Clean checkout\n        run: scripts/checkout_sdk.sh destination\n"
        changed = self.workflow.replace("    steps:\n", "    steps:\n" + sdk_step, 1)
        with self.assertRaises(ValueError):
            validate_job_split(changed)

    def test_validation_policy_names_actual_commands(self) -> None:
        policy = (ROOT / "AGENTS.md").read_text()
        self.assertIn("export GOWORK=off GO111MODULE=on", policy)
        for command in COMMANDS:
            self.assertIn(command, policy)
        self.assertIn("only compiles", policy)
        self.assertIn("Hosted full tagged execution remains required", policy)
        self.assertIn(f"`{ADAPTER_JOB[0]}`", policy)
        self.assertIn(ADAPTER_JOB[1], policy)
        for marker in SDK_MARKERS:
            self.assertNotIn(marker, policy)


if __name__ == "__main__":
    unittest.main()
