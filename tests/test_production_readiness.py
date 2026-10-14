"""Mutation tests for the fail-closed endpoint readiness decision."""

from __future__ import annotations

import copy
import json
import unittest
from pathlib import Path

from runtime.production_readiness import evaluate, validate_build_evidence, validate_composition


ROOT = Path(__file__).resolve().parents[1]
LOCK = ROOT / "sdk" / "matter-sdk.lock.json"


def load_lock() -> dict:
    return json.loads(LOCK.read_text(encoding="utf-8"))


def accepted_composition(lock: dict) -> dict:
    return {
        "schema_version": 1,
        "status": "accepted",
        "profile_id": "matter-1.7-electrical-sensor-v1",
        "target_commit": lock["sdk"]["commit"],
        "mapping": copy.deepcopy(lock["mapping"]),
        "device_type": copy.deepcopy(lock["target"]["device_type"]),
        "clusters": [
            {"id": "0x001D", "name": "Descriptor", "evidence": "accepted-native-profile"},
            {"id": "0x009C", "name": "Power Topology", "evidence": "accepted-native-profile"},
            {"id": "0x0090", "name": "Electrical Power Measurement", "evidence": "accepted-native-profile"},
        ],
        "electrical_power_measurement": {
            "power_mode": {"value": "ac", "evidence": "accepted-native-profile"},
            "number_of_measurement_types": {"value": 1, "evidence": "accepted-native-profile"},
            "accuracy": [{"measurement_type": "active_power", "evidence": "accepted-native-profile"}],
            "active_power": {"value": None, "evidence": "accepted-native-profile"},
        },
    }


def accepted_build(lock: dict) -> dict:
    return {
        "schema_version": 1,
        "sdk_commit": lock["sdk"]["commit"],
        "targets": {
            "all_devices_app": "passed",
            "chip_tool": "passed",
            "electrical_power_measurement_tests": "passed",
            "power_topology_tests": "passed",
            "upstream_electrical_sensor_fixture": "passed",
        },
    }


class ProductionReadinessTest(unittest.TestCase):
    def setUp(self) -> None:
        self.lock = load_lock()
        self.composition = accepted_composition(self.lock)
        self.build = accepted_build(self.lock)

    def test_complete_self_asserted_evidence_cannot_enable_the_gate(self) -> None:
        decision = evaluate(self.lock, self.composition, self.build)
        self.assertFalse(decision.permitted)
        self.assertIn("no reviewed accepted runtime-composition profile is pinned", decision.reasons)

    def test_absent_composition_cannot_enable_the_gate(self) -> None:
        self.assertFalse(evaluate(self.lock, None, self.build).permitted)
        self.assertIn("absent or malformed", validate_composition(self.lock, None)[0])

    def test_proposed_composition_cannot_enable_the_gate(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["status"] = "proposed"
        self.assertFalse(evaluate(self.lock, composition, self.build).permitted)
        self.assertTrue(any("not accepted" in reason for reason in validate_composition(self.lock, composition)))

    def test_mismatched_target_pin_cannot_enable_the_gate(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["target_commit"] = "0" * 40
        self.assertFalse(evaluate(self.lock, composition, self.build).permitted)
        self.assertTrue(any("different SDK commit" in reason for reason in validate_composition(self.lock, composition)))

    def test_missing_power_topology_cannot_enable_the_gate(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["clusters"] = [item for item in composition["clusters"] if item["id"] != "0x009C"]
        self.assertFalse(evaluate(self.lock, composition, self.build).permitted)
        self.assertIn("mandatory Power Topology composition metadata is absent or unevidenced", validate_composition(self.lock, composition))

    def test_empty_accuracy_cannot_enable_the_gate(self) -> None:
        composition = copy.deepcopy(self.composition)
        composition["electrical_power_measurement"]["accuracy"] = []
        self.assertFalse(evaluate(self.lock, composition, self.build).permitted)
        self.assertIn("mandatory Accuracy list is empty or malformed", validate_composition(self.lock, composition))

    def test_missing_build_evidence_cannot_enable_the_gate(self) -> None:
        build = copy.deepcopy(self.build)
        del build["targets"]["upstream_electrical_sensor_fixture"]
        self.assertFalse(evaluate(self.lock, self.composition, build).permitted)
        self.assertIn("build evidence omits upstream_electrical_sensor_fixture", validate_build_evidence(self.lock, build))

    def test_mismatched_attribute_identifier_cannot_enable_the_gate(self) -> None:
        lock = copy.deepcopy(self.lock)
        lock["target"]["active_current_attribute"]["id"] = "0x0006"
        self.assertFalse(evaluate(lock, self.composition, self.build).permitted)



if __name__ == "__main__":
    unittest.main()
