"""Synthetic proposal identity, evidence debt and independent admission checks."""

from __future__ import annotations

import base64
import copy
import hashlib
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest.mock import Mock, patch

from runtime import adoption_baseline as baseline
from runtime import production_readiness


# These values are synthetic supplied claims, never a selected SDK pin.
def checksum(label: str) -> str:
    return "h1:" + base64.b64encode(hashlib.sha256(label.encode()).digest()).decode()


def proposal() -> dict:
    return {
        "schema_version": 1,
        "provider": "go-fabric",
        "module": "github.com/SukramJ/go-fabric",
        "release_tag": "v0.0.1-synthetic",
        "checksum": checksum("synthetic archive"),
        "model": {"name": "Matter", "version": "1.6.1"},
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
        for tag in ("v0.0.2", "v1.2.3-rc.1"):
            item = proposal()
            item["release_tag"] = tag
            item["checksum"] = checksum(tag)
            self.assertTrue(baseline.evaluate_adoption_baseline(item, evidence(item)).proposal_valid)

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

    def test_disposable_owned_source_mutants_are_killed(self) -> None:
        source = Path(baseline.__file__).read_text()
        mutations = {
            **{f"drop join {field}": (
                "for field in IDENTITY_FIELDS)", f'for field in IDENTITY_FIELDS if field != "{field}")'
            ) for field in KEY_FIELDS},
            "accept wrong provider": ('proposal["provider"] != "go-fabric"', "False"),
            "accept wrong module": ('proposal["module"] != "github.com/SukramJ/go-fabric"', "False"),
            "accept wrong model": ('proposal["model"] != {"name": "Matter", "version": "1.6.1"}', "False"),
            "structure qualifies B1-B8": ('(f"B{number}", "Unknown")', '(f"B{number}", "Proven")'),
            "structure admits": ("permitted: bool = False", "permitted: bool = True"),
        }
        for name, (old, new) in mutations.items():
            with self.subTest(mutant=name), tempfile.TemporaryDirectory() as directory:
                self.assertEqual(source.count(old), 1)
                path = Path(directory) / "adoption_baseline.py"
                path.write_text(source.replace(old, new))
                module = types.ModuleType("runtime._adoption_mutant")
                with patch.dict(sys.modules, {module.__name__: module}):
                    exec(compile(path.read_text(), str(path), "exec"), module.__dict__)
                    evaluate = module.evaluate_adoption_baseline
                    if name.startswith("drop join"):
                        with self.assertRaises(AssertionError):
                            self.assert_identity_contract(evaluate)
                    elif name.startswith("accept wrong"):
                        field = name.removeprefix("accept wrong ")
                        item = proposal()
                        item[field] = CHANGED[field]
                        with self.assertRaises(AssertionError):
                            self.assertFalse(evaluate(item, evidence(item)).proposal_valid)
                    else:
                        item = proposal()
                        with self.assertRaises(AssertionError):
                            self.assert_nonadmission(evaluate(item, evidence(item)))


if __name__ == "__main__":
    unittest.main()
