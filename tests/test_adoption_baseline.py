"""Synthetic proposal identity, evidence debt and independent admission checks."""

from __future__ import annotations

import base64
import copy
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest.mock import Mock, patch

from runtime import adoption_baseline as baseline
from runtime import production_readiness

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
import validate_lock as pin_checker  # noqa: E402
from validate_lock import validate_module_pin  # noqa: E402

CHECKOUT_KEYS = ("repository", "commit", "upstream_merge", "data_model", "blobs")


def direct_requirement(root: Path = ROOT) -> tuple[str, str]:
    """Read the single direct module requirement. A block entry is not that pin."""
    found = [
        (parts[1], parts[2])
        for line in (root / "go.mod").read_text(encoding="utf-8").splitlines()
        if (parts := line.split()) and len(parts) == 3 and parts[0] == "require"
    ]
    if len(found) != 1:
        raise AssertionError("direct module requirement must be exactly one require line")
    return found[0]


def sum_checksum(module: str, tag: str, *, go_mod: bool, root: Path = ROOT) -> str:
    """Return the one zip or go.mod h1 recorded for the module requirement."""
    version = f"{tag}/go.mod" if go_mod else tag
    values: list[str] = []
    for line in (root / "go.sum").read_text(encoding="utf-8").splitlines():
        parts = line.split()
        if len(parts) == 3 and parts[0] == module and parts[1] == version and parts[2].startswith("h1:"):
            values.append(parts[2])
    if len(values) != 1:
        raise AssertionError("module sum must record exactly one matching checksum")
    return values[0]


# Targets and retained references are synthetic. Identity fields come from the module pin.
def checksum(label: str) -> str:
    return "h1:" + base64.b64encode(hashlib.sha256(label.encode()).digest()).decode()


def proposal() -> dict:
    sdk = json.loads((ROOT / "sdk" / "matter-sdk.lock.json").read_text(encoding="utf-8"))["sdk"]
    return {
        "schema_version": 1,
        "provider": sdk["provider"],
        "module": sdk["module"],
        "release_tag": sdk["release_tag"],
        "checksum": sdk["checksum"],
        "model": copy.deepcopy(sdk["model"]),
        "targets": ["synthetic-electrical-family", "synthetic-topology-family"],
        "deferred_features": ["synthetic-future-model-feature"],
        "missing_evidence": ["independent source replay", "platform results"],
    }


# Independent field inventory: do not derive expectations from production constants.
KEY_FIELDS = (
    "provider", "module", "release_tag", "checksum", "model", "targets", "deferred_features",
)
CHANGED = {
    "provider": "other-provider",
    "module": "github.com/other/go-fabric",
    "release_tag": "v0.0.2-synthetic",
    "checksum": checksum("different synthetic archive"),
    "model": {"name": "Matter", "version": "1.7"},
    "targets": ["different-synthetic-family"],
    "deferred_features": ["different-synthetic-feature"],
}


def evidence(item: dict) -> dict:
    return {
        "identity": {key: copy.deepcopy(item[key]) for key in KEY_FIELDS},
        "retained_inputs": ["synthetic/source-record", "synthetic/build-record"],
    }


class AdoptionBaselineTest(unittest.TestCase):
    def assert_nonadmission(self, result) -> None:
        self.assertFalse(result.permitted)
        self.assertEqual(result.provenance, "Unknown")
        self.assertEqual(result.qualification, tuple((f"B{n}", "Unknown") for n in range(1, 9)))
        self.assertIn("source and checksum provenance is unverified", result.missing_evidence)
        self.assertIn("B1-B8 qualification is unverified", result.missing_evidence)

    def test_valid_join_is_only_structure_and_preserves_missing_evidence(self) -> None:
        item = proposal()
        result = baseline.evaluate_adoption_baseline(item, evidence(item))
        self.assertTrue(result.proposal_valid)
        self.assertEqual(result.errors, ())
        self.assertEqual(result.retained_inputs, ("synthetic/source-record", "synthetic/build-record"))
        self.assertEqual(result.proposal.provider, "go-fabric")
        self.assertEqual(result.proposal.module, "github.com/SukramJ/go-fabric")
        self.assertEqual(result.proposal.release_tag, item["release_tag"])
        self.assertEqual(result.proposal.checksum, item["checksum"])
        self.assertEqual(result.proposal.model, ("Matter", "1.6.1"))
        self.assertEqual(result.proposal.targets, tuple(item["targets"]))
        self.assertEqual(result.proposal.deferred_features, tuple(item["deferred_features"]))
        self.assertEqual(result.proposal.missing_evidence, tuple(item["missing_evidence"]))
        self.assertEqual(result.missing_evidence[:2], ("independent source replay", "platform results"))
        self.assert_nonadmission(result)

    def test_no_successful_default_or_inferred_retained_inputs(self) -> None:
        result = baseline.evaluate_adoption_baseline()
        self.assertFalse(result.proposal_valid)
        self.assertIsNone(result.proposal)
        self.assertEqual(result.retained_inputs, ())
        self.assert_nonadmission(result)
        result = baseline.evaluate_adoption_baseline(proposal())
        self.assertTrue(result.proposal_valid)
        self.assertIn("retained proposal evidence is missing", result.missing_evidence)
        self.assert_nonadmission(result)

    def test_each_required_proposal_field_is_required(self) -> None:
        for field in (*KEY_FIELDS, "schema_version", "missing_evidence"):
            with self.subTest(field=field):
                item = proposal()
                del item[field]
                self.assertFalse(baseline.evaluate_adoption_baseline(item).proposal_valid)

    def test_wrong_provider_module_and_each_model_field_even_when_joined(self) -> None:
        changes = [
            ("provider", CHANGED["provider"]), ("module", CHANGED["module"]),
            ("model", {"name": "Other", "version": "1.6.1"}),
            ("model", {"name": "Matter", "version": "1.7"}),
            ("model", {"name": "Matter", "version": "1.6.1", "extra": True}),
        ]
        for field, value in changes:
            with self.subTest(field=field, value=value):
                item = proposal()
                item[field] = value
                result = baseline.evaluate_adoption_baseline(item, evidence(item))
                self.assertFalse(result.proposal_valid)
                self.assert_nonadmission(result)

    def assert_identity_contract(self, evaluate) -> None:
        item = proposal()
        self.assertTrue(evaluate(item, evidence(item)).proposal_valid)
        for field, value in CHANGED.items():
            supplied = evidence(item)
            supplied["identity"][field] = value
            result = evaluate(item, supplied)
            self.assertFalse(result.proposal_valid, field)
            self.assertIn("retained evidence identity differs from proposal", result.errors)
            self.assertEqual(result.retained_inputs, ())
        for model_field, value in (("name", "Other"), ("version", "1.7")):
            supplied = evidence(item)
            supplied["identity"]["model"][model_field] = value
            self.assertFalse(evaluate(item, supplied).proposal_valid, model_field)

    def test_each_identity_join_independently(self) -> None:
        self.assert_identity_contract(baseline.evaluate_adoption_baseline)
        item = proposal()
        for field in KEY_FIELDS:
            with self.subTest(missing=field):
                supplied = evidence(item)
                del supplied["identity"][field]
                self.assertFalse(baseline.evaluate_adoption_baseline(item, supplied).proposal_valid)
        for field in ("targets", "deferred_features"):
            for value in ([], [*proposal()[field], "another-synthetic-feature"], list(reversed(proposal()[field]))):
                if value == proposal()[field]:
                    continue
                with self.subTest(field=field, value=value):
                    supplied = evidence(item)
                    supplied["identity"][field] = value
                    self.assertFalse(baseline.evaluate_adoption_baseline(item, supplied).proposal_valid)

    def test_tag_checksum_and_schema_are_strict_supplied_claims(self) -> None:
        bad = {
            "release_tag": [None, "", "latest", "main", "v1.2", "^v1.2.3", "v01.2.3", "v1.2.3-01", "v1.2.3\n"],
            "checksum": [None, "", "h1:broken", "h1:" + base64.b64encode(b"short").decode(), checksum("x") + "\n"],
            "schema_version": [None, True, 1.0, 2],
        }
        for field, values in bad.items():
            for value in values:
                with self.subTest(field=field, value=value):
                    item = proposal()
                    item[field] = value
                    self.assertFalse(baseline.evaluate_adoption_baseline(item).proposal_valid)
        module, tag = direct_requirement()
        for value in ("v0.0.2", "v1.2.3-rc.1", ".".join(["v" + tag.split(".")[0].removeprefix("v"), *tag.split(".")[1:2]])):
            item = proposal()
            item["release_tag"] = value
            item["checksum"] = checksum(value)
            self.assertFalse(baseline.evaluate_adoption_baseline(item, evidence(item)).proposal_valid)
        item = proposal()
        item["module"] = module.casefold()
        self.assertFalse(baseline.evaluate_adoption_baseline(item, evidence(item)).proposal_valid)
        item = proposal()
        item["checksum"] = sum_checksum(module, tag, go_mod=True)
        self.assertFalse(baseline.evaluate_adoption_baseline(item, evidence(item)).proposal_valid)

    def test_rosters_and_explicit_missing_inventory(self) -> None:
        for field in ("targets", "deferred_features", "missing_evidence"):
            for value in (None, "not-a-list", [""], [" spaced"], ["duplicate", "duplicate"], [{}]):
                with self.subTest(field=field, value=value):
                    item = proposal()
                    item[field] = value
                    self.assertFalse(baseline.evaluate_adoption_baseline(item).proposal_valid)
        item = proposal()
        item["targets"] = []
        self.assertFalse(baseline.evaluate_adoption_baseline(item).proposal_valid)
        item = proposal()
        item["deferred_features"] = []
        item["missing_evidence"] = []
        result = baseline.evaluate_adoption_baseline(item, evidence(item))
        self.assertTrue(result.proposal_valid)
        self.assert_nonadmission(result)

    def test_malformed_evidence_and_self_asserted_passes_cannot_qualify(self) -> None:
        item = proposal()
        for supplied in ({}, {"identity": {}}, [], {**evidence(item), "B1": "passed"},
                         {**evidence(item), "retained_inputs": []}):
            with self.subTest(supplied=supplied):
                result = baseline.evaluate_adoption_baseline(item, supplied)
                self.assertFalse(result.proposal_valid)
                self.assertEqual(result.retained_inputs, ())
                self.assert_nonadmission(result)
        item["qualification"] = {f"B{n}": "passed" for n in range(1, 9)}
        self.assertFalse(baseline.evaluate_adoption_baseline(item).proposal_valid)

    def test_detached_results_preserve_caller_inputs(self) -> None:
        item = proposal()
        supplied = evidence(item)
        original = copy.deepcopy((item, supplied))
        result = baseline.evaluate_adoption_baseline(item, supplied)
        self.assertEqual((item, supplied), original)
        item["targets"].clear()
        item["model"]["version"] = "changed"
        item["missing_evidence"].clear()
        supplied["retained_inputs"].clear()
        self.assertEqual(result.proposal.targets, ("synthetic-electrical-family", "synthetic-topology-family"))
        self.assertEqual(result.proposal.model, ("Matter", "1.6.1"))
        self.assertEqual(result.proposal.missing_evidence, ("independent source replay", "platform results"))
        self.assertEqual(result.retained_inputs, ("synthetic/source-record", "synthetic/build-record"))

    def test_real_production_api_and_independent_allocation_spy(self) -> None:
        lock = copy.deepcopy(production_readiness.EXPECTED_LOCK)
        composition = {
            "schema_version": 1, "status": "accepted", "profile_id": "synthetic-profile",
            "target_commit": lock["sdk"]["commit"], "mapping": copy.deepcopy(lock["mapping"]),
            "device_type": copy.deepcopy(lock["target"]["device_type"]),
            "clusters": [{"id": key, "name": value, "evidence": "synthetic-source"}
                         for key, value in production_readiness.REQUIRED_CLUSTERS.items()],
            "electrical_power_measurement": {
                "power_mode": {"value": "ac", "evidence": "synthetic-source"},
                "number_of_measurement_types": {"value": 1, "evidence": "synthetic-source"},
                "accuracy": [{"measurement_type": "active_power", "evidence": "synthetic-source"}],
                "active_power": {"value": None, "evidence": "synthetic-source"},
            },
        }
        build = {"schema_version": 1, "sdk_commit": lock["sdk"]["commit"],
                 "targets": {name: "passed" for name in production_readiness.REQUIRED_BUILD_TARGETS}}
        self.assertEqual(production_readiness.validate_composition(lock, composition), ())
        self.assertEqual(production_readiness.validate_build_evidence(lock, build), ())
        original = copy.deepcopy((lock, composition, build))
        allocate = Mock()
        item = proposal()
        for forge_pin in (False, True):
            supplied_lock = copy.deepcopy(lock)
            if forge_pin:
                supplied_lock["accepted_runtime_composition_profile_sha256"] = hashlib.sha256(b"synthetic profile").hexdigest()
            with patch.object(production_readiness, "evaluate", wraps=production_readiness.evaluate) as real:
                result = baseline.evaluate_adoption_baseline(
                    item, evidence(item), production_lock=supplied_lock,
                    composition=composition, build_evidence=build,
                )
                real.assert_called_once_with(supplied_lock, composition, build)
            self.assertTrue(result.proposal_valid)
            self.assert_nonadmission(result)
            self.assertFalse(result.production_decision.permitted)
            if not forge_pin:
                self.assertEqual(result.production_decision.reasons, ("no reviewed accepted runtime-composition profile is pinned",))
            if result.permitted or result.production_decision.permitted:
                allocate()
        allocate.assert_not_called()
        self.assertEqual((lock, composition, build), original)

    def test_lock_names_module_path_tag_and_zip_checksum(self) -> None:
        """The lock pin is the module path, tag and zip checksum, with no checkout identity.

        Fields varied, and the production comparison a wrong value must fail:
        provider — equality with the provider name skipped.
        module — case-insensitive or prefix comparison.
        release_tag — major.minor comparison, or a range.
        checksum — zip checksum skipped, or the go.mod checksum accepted in its place.
        model name and model version — model omitted from the comparison.
        checkout keys — a repository or commit key treated as part of the pin.
        """
        lock_text = (ROOT / "sdk" / "matter-sdk.lock.json").read_text(encoding="utf-8")
        sdk = json.loads(lock_text)["sdk"]
        module, tag = direct_requirement()
        self.assertEqual(sdk.get("provider"), "go-fabric")
        self.assertEqual(sdk.get("module"), module)
        self.assertNotEqual(sdk.get("module"), module.casefold())
        self.assertFalse(str(sdk.get("module", "")).startswith(module + "/"))
        self.assertEqual(sdk.get("release_tag"), tag)
        self.assertNotEqual(sdk.get("release_tag"), ".".join(tag.removeprefix("v").split(".")[:2]))
        self.assertEqual(sdk.get("checksum"), sum_checksum(module, tag, go_mod=False))
        self.assertNotEqual(sdk.get("checksum"), sum_checksum(module, tag, go_mod=True))
        self.assertEqual(sdk.get("model"), {"name": "Matter", "version": "1.6.1"})
        self.assertNotIn("connectedhomeip", lock_text)
        for key in CHECKOUT_KEYS:
            self.assertNotIn(key, sdk)

    def test_disposable_owned_source_mutants_are_killed(self) -> None:
        source = Path(baseline.__file__).read_text()
        mutations = {
            **{f"drop join {field}": (
                "for field in IDENTITY_FIELDS)", f'for field in IDENTITY_FIELDS if field != "{field}")'
            ) for field in KEY_FIELDS},
            **{f"skip pinned {field}": (
                "for field in _PIN_FIELDS:",
                f'for field in tuple(name for name in _PIN_FIELDS if name != "{field}"):',
            ) for field in ("provider", "module", "release_tag", "checksum", "model")},
            "casefold pinned fields": (
                "if proposal[field] != pin[field]:",
                "if str(proposal[field]).casefold() != str(pin[field]).casefold():",
            ),
            "structure qualifies B1-B8": ('(f"B{number}", "Unknown")', '(f"B{number}", "Proven")'),
            "structure admits": ("permitted: bool = False", "permitted: bool = True"),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(mutant=name), tempfile.TemporaryDirectory() as directory:
                self.assertEqual(source.count(old), 1)
                path = Path(directory) / "adoption_baseline.py"
                path.write_text(source.replace(old, new))
                module = types.ModuleType("runtime._adoption_mutant")
                module.__dict__["__file__"] = baseline.__file__
                with patch.dict(sys.modules, {module.__name__: module}):
                    exec(compile(path.read_text(), str(path), "exec"), module.__dict__)
                    evaluate = module.evaluate_adoption_baseline
                    if name.startswith("drop join"):
                        with self.assertRaises(AssertionError):
                            self.assert_identity_contract(evaluate)
                    elif name.startswith("skip pinned") or name == "casefold pinned fields":
                        field = "module" if name == "casefold pinned fields" else name.removeprefix("skip pinned ")
                        item = proposal()
                        item[field] = item["module"].casefold() if field == "module" and name.startswith("casefold") else CHANGED[field]
                        with self.assertRaises(AssertionError):
                            self.assertFalse(evaluate(item, evidence(item)).proposal_valid)
                    else:
                        item = proposal()
                        with self.assertRaises(AssertionError):
                            self.assert_nonadmission(evaluate(item, evidence(item)))


def shifted_tag(tag: str) -> str:
    """Return the same module line with only the last numeric component changed."""
    numbers = [int(part) for part in tag.removeprefix("v").split(".")]
    numbers[-1] += 1
    return "v" + ".".join(str(number) for number in numbers)


def with_value(document: dict, path: tuple[str, ...], value: object) -> dict:
    changed = copy.deepcopy(document)
    cursor = changed
    for key in path[:-1]:
        cursor = cursor[key]
    cursor[path[-1]] = value
    return changed


class ModulePinTest(unittest.TestCase):
    """Each identity field is rejected alone, and each listed mutation lets that field through."""

    def setUp(self) -> None:
        self.lock = json.loads((ROOT / "sdk" / "matter-sdk.lock.json").read_text(encoding="utf-8"))
        self.module, self.tag = direct_requirement()
        self.assertEqual(validate_module_pin(self.lock, ROOT), [])

    def test_each_identity_and_target_field_is_required(self) -> None:
        """Vary provider, module, release_tag, checksum, model name, model version and each target field."""
        digest = hashlib.sha256(b"synthetic profile").hexdigest()
        samples = {
            "provider": with_value(self.lock, ("sdk", "provider"), "other-provider"),
            "module case": with_value(self.lock, ("sdk", "module"), self.module.casefold()),
            "module prefix": with_value(self.lock, ("sdk", "module"), self.module + "/extra"),
            "release_tag patch": with_value(self.lock, ("sdk", "release_tag"), shifted_tag(self.tag)),
            "release_tag range": with_value(self.lock, ("sdk", "release_tag"), ">=" + self.tag),
            "zip checksum": with_value(self.lock, ("sdk", "checksum"), sum_checksum(self.module, self.tag, go_mod=True)),
            "model name": with_value(self.lock, ("sdk", "model", "name"), "Other"),
            "model version": with_value(self.lock, ("sdk", "model", "version"), "1.7"),
            "mapping": with_value(self.lock, ("mapping",), {"path": "synthetic"}),
            "profile pin": with_value(self.lock, ("accepted_runtime_composition_profile_sha256",), digest),
            "checkout key": with_value(self.lock, ("sdk", "commit"), "synthetic"),
        }
        for name, section, field, value in (
            ("device name", "device_type", "name", "Other Sensor"),
            ("device id", "device_type", "id", "0x0511"),
            ("device revision", "device_type", "revision", 2),
            ("topology name", "power_topology_cluster", "name", "Other Topology"),
            ("topology id", "power_topology_cluster", "id", "0x009D"),
            ("topology revision", "power_topology_cluster", "revision", 2),
            ("measurement name", "electrical_power_measurement_cluster", "name", "Other Measurement"),
            ("measurement id", "electrical_power_measurement_cluster", "id", "0x0091"),
            ("measurement revision", "electrical_power_measurement_cluster", "revision", 4),
            ("current name", "active_current_attribute", "name", "OtherCurrent"),
            ("current id", "active_current_attribute", "id", "0x0006"),
            ("current optional", "active_current_attribute", "optional", False),
        ):
            samples[name] = with_value(self.lock, ("target", section, field), value)
        for name, sample in samples.items():
            with self.subTest(field=name):
                self.assertTrue(validate_module_pin(sample, ROOT), name)

    def test_absent_or_replaced_checksum_mutation_is_killed(self) -> None:
        """checksum — comparison skipped when the zip line is absent, or the go.mod line used in its place."""
        source = Path(pin_checker.__file__).read_text(encoding="utf-8")
        mutations = {
            "absent zip line": (
                'if checksum is None:\n        reasons.append("module zip checksum is absent")',
                "if checksum is None:\n        return []",
            ),
            "go.mod checksum": (
                "recorded_checksum(sums, module, tag)",
                'recorded_checksum(sums, module, f"{tag}/go.mod")',
            ),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(mutant=name), tempfile.TemporaryDirectory() as directory:
                self.assertEqual(source.count(old), 1)
                checker = load_checker(source.replace(old, new))
                if name == "go.mod checksum":
                    with self.assertRaises(AssertionError):
                        self.assertEqual(checker(self.lock, ROOT), [])
                    continue
                root = directory_without_zip(Path(directory), self.module, self.tag)
                with self.assertRaises(AssertionError):
                    self.assertTrue(checker(self.lock, root))

    def test_identity_comparison_mutants_are_killed(self) -> None:
        """provider, module, release_tag, model, mapping, profile pin, checkout key and target identity."""
        source = Path(pin_checker.__file__).read_text(encoding="utf-8")
        digest = hashlib.sha256(b"synthetic profile").hexdigest()
        mutations = {
            "skip provider": ('sdk["provider"] != "go-fabric"', "False"),
            "casefold module": ("sdk[\"module\"] != module", "sdk[\"module\"].casefold() != module.casefold()"),
            "prefix module": ("sdk[\"module\"] != module", "not sdk[\"module\"].startswith(module)"),
            "major minor tag": (
                'sdk["release_tag"] != tag',
                'sdk["release_tag"].split(".")[:2] != tag.split(".")[:2]',
            ),
            "skip model": ('sdk["model"] != {"name": "Matter", "version": "1.6.1"}', "False"),
            "allow extra sdk keys": ("set(lock[\"sdk\"]) != SDK_FIELDS", "not SDK_FIELDS <= set(lock[\"sdk\"])"),
            "skip mapping": ('lock["mapping"] is not None', "False"),
            "skip profile pin": (
                'lock["accepted_runtime_composition_profile_sha256"] is not None',
                "False",
            ),
            "revision only target": (
                "if target.get(key) != expected:",
                "if isinstance(expected, dict) and expected.get(\"revision\") is not None and "
                "target.get(key, {}).get(\"revision\") != expected.get(\"revision\"):",
            ),
        }
        samples = {
            "skip provider": with_value(self.lock, ("sdk", "provider"), "other-provider"),
            "casefold module": with_value(self.lock, ("sdk", "module"), self.module.casefold()),
            "prefix module": with_value(self.lock, ("sdk", "module"), self.module + "/extra"),
            "major minor tag": with_value(self.lock, ("sdk", "release_tag"), shifted_tag(self.tag)),
            "skip model": with_value(self.lock, ("sdk", "model", "version"), "1.7"),
            "allow extra sdk keys": with_value(self.lock, ("sdk", "commit"), "synthetic"),
            "skip mapping": with_value(self.lock, ("mapping",), {"path": "synthetic"}),
            "skip profile pin": with_value(self.lock, ("accepted_runtime_composition_profile_sha256",), digest),
            "revision only target": with_value(self.lock, ("target", "device_type", "name"), "Other Sensor"),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(mutant=name):
                self.assertEqual(source.count(old), 1)
                checker = load_checker(source.replace(old, new))
                with self.assertRaises(AssertionError):
                    self.assertTrue(checker(samples[name], ROOT))


def load_checker(source: str):
    module = types.ModuleType("tools._pin_mutant")
    module.__dict__["__file__"] = pin_checker.__file__
    exec(compile(source, pin_checker.__file__, "exec"), module.__dict__)
    return module.validate_module_pin


def directory_without_zip(root: Path, module: str, tag: str) -> Path:
    (root / "go.mod").write_text((ROOT / "go.mod").read_text(encoding="utf-8"), encoding="utf-8")
    kept = [
        line
        for line in (ROOT / "go.sum").read_text(encoding="utf-8").splitlines()
        if line.split()[:2] != [module, tag]
    ]
    (root / "go.sum").write_text("\n".join(kept) + "\n", encoding="utf-8")
    return root


if __name__ == "__main__":
    unittest.main()
