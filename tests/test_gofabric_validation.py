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
LEGACY_JOB = ("legacy-sdk-fixture", "Legacy SDK fixture (connectedhomeip)")
GO_MARKERS = ("go vet", "go test", "gofabric", "GOWORK", "GO111MODULE")
SDK_MARKERS = (
    "apt-get",
    "scripts/checkout_sdk.sh",
    "scripts/bootstrap.sh",
    "scripts/activate.sh",
    "build_examples.py",
    "out/focused-tests",
    "run_upstream_fixture.sh",
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
    """Require a go-fabric job and a legacy SDK job with distinct names and disjoint steps."""
    jobs = split_jobs(workflow)
    if set(jobs) != {ADAPTER_JOB[0], LEGACY_JOB[0]}:
        raise ValueError("expected exactly the adapter job and the legacy SDK job")
    adapter, legacy = jobs[ADAPTER_JOB[0]], jobs[LEGACY_JOB[0]]
    if job_name(adapter) != ADAPTER_JOB[1] or job_name(legacy) != LEGACY_JOB[1]:
        raise ValueError("job names must state what each job runs")
    if re.findall(r"^    needs: (.*)$", legacy, re.M) != [ADAPTER_JOB[0]]:
        raise ValueError("legacy SDK job must need the adapter job")
    if re.findall(r"^    timeout-minutes: (.*)$", legacy, re.M) != ["90"]:
        raise ValueError("legacy SDK job keeps its 90-minute bound")
    for step in job_steps(legacy):
        if any(marker in step for marker in GO_MARKERS):
            raise ValueError("legacy SDK job must not run a go-fabric step")
    for step in job_steps(adapter):
        if any(marker in step for marker in SDK_MARKERS):
            raise ValueError("adapter job must not run a legacy SDK step")


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
        end = self.workflow.index("\n  " + LEGACY_JOB[0] + ":\n", start) + 1
        step = self.workflow[start:end]
        with self.assertRaises(ValueError):
            validate_adapter_step(self.workflow[:end] + step + self.workflow[end:])

    def test_jobs_are_split_and_named(self) -> None:
        validate_job_split(self.workflow)
        jobs = split_jobs(self.workflow)
        self.assertNotEqual(job_name(jobs[ADAPTER_JOB[0]]), job_name(jobs[LEGACY_JOB[0]]))

    def test_each_job_id_and_name_is_required(self) -> None:
        for job_id, name in (ADAPTER_JOB, LEGACY_JOB):
            for changed in (
                self.workflow.replace("\n  " + job_id + ":\n", "\n  renamed-job:\n", 1),
                self.workflow.replace("    name: " + name + "\n", "    name: Matter checks\n", 1),
                self.workflow.replace("    name: " + name + "\n", "", 1),
            ):
                with self.subTest(job=job_id, changed=changed):
                    with self.assertRaises(ValueError):
                        validate_job_split(changed)

    def test_identical_job_names_fail(self) -> None:
        changed = self.workflow.replace("    name: " + LEGACY_JOB[1], "    name: " + ADAPTER_JOB[1], 1)
        with self.assertRaises(ValueError):
            validate_job_split(changed)

    def test_merged_single_job_fails(self) -> None:
        changed = re.sub(r"\n  legacy-sdk-fixture:\n(    [^\n]*\n)*?    steps:\n", "\n", self.workflow, count=1)
        self.assertNotEqual(changed, self.workflow)
        with self.assertRaises(ValueError):
            validate_job_split(changed)

    def test_legacy_job_dependency_and_bound_are_required(self) -> None:
        for changed in (
            self.workflow.replace("    needs: " + ADAPTER_JOB[0] + "\n", "", 1),
            self.workflow.replace("    timeout-minutes: 90\n", "    timeout-minutes: 30\n", 1),
        ):
            with self.subTest(changed=changed):
                with self.assertRaises(ValueError):
                    validate_job_split(changed)

    def test_legacy_job_contains_no_go_fabric_step(self) -> None:
        start = self.workflow.index("      - name: " + STEP)
        end = self.workflow.index("\n  " + LEGACY_JOB[0] + ":\n", start) + 1
        step = self.workflow[start:end].rstrip("\n") + "\n"
        changed = self.workflow + step
        with self.assertRaises(ValueError):
            validate_job_split(changed)
        with self.assertRaises(ValueError):
            validate_adapter_step(self.workflow[:start] + self.workflow[end:] + step)

    def test_adapter_job_contains_no_sdk_step(self) -> None:
        sdk_step = (
            "      - name: Clean recursive exact-SHA checkout\n"
            '        run: scripts/checkout_sdk.sh "$RUNNER_TEMP/connectedhomeip"\n'
        )
        end = self.workflow.index("\n  " + LEGACY_JOB[0] + ":\n")
        with self.assertRaises(ValueError):
            validate_job_split(self.workflow[:end] + "\n" + sdk_step + self.workflow[end:])

    def test_validation_policy_names_actual_commands(self) -> None:
        policy = (ROOT / "AGENTS.md").read_text()
        self.assertIn("export GOWORK=off GO111MODULE=on", policy)
        for command in COMMANDS:
            self.assertIn(command, policy)
        self.assertIn("only compiles", policy)
        self.assertIn("Hosted full tagged execution remains required", policy)
        for job_id, name in (ADAPTER_JOB, LEGACY_JOB):
            self.assertIn(f"`{job_id}`", policy)
            self.assertIn(name, policy)


if __name__ == "__main__":
    unittest.main()
