"""Production endpoint admission is deliberately fail-closed.

This module does not create a Matter node, open a fabric, or transform a
Gateway value. It only evaluates evidence a future launcher must supply before
it is allowed to create a Helianthus Electrical Sensor endpoint.
"""

from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass
from typing import Any


SDK_COMMIT = "29b4768a513cf566011ab8cd60df1bc495204953"
UPSTREAM_MERGE_COMMIT = "bd73b5141729af92f3d021aeecaa53d9e0cadb89"
DOCS_SEMANTIC_COMMIT = "30f5e5c79ac6da3a7c7c10c990599906d1dfd0cb"
MAPPING_SHA256 = "5ae81d5e0971d25ead08f982fdf31caf47ada4838d0ee6f2b3e719d11a6df39c"

EXPECTED_LOCK: dict[str, Any] = {
    "schema_version": 1,
    "accepted_runtime_composition_profile_sha256": None,
    "sdk": {
        "repository": "https://github.com/AryaHassanli/connectedhomeip.git",
        "commit": SDK_COMMIT,
        "upstream_merge": {
            "repository": "https://github.com/project-chip/connectedhomeip.git",
            "pull_request": 73842,
            "commit": UPSTREAM_MERGE_COMMIT,
        },
        "data_model": {
            "version": "1.7",
            "spec_sha": {"path": "data_model/1.7/spec_sha", "blob": "86ef3bbd871805600c751bd84c639cfb0af64ed9"},
            "spec_tag": {"path": "data_model/1.7/spec_tag", "blob": "df6ffc2f98235eea1a41493e971ef53e69084425"},
        },
        "blobs": [
            {"path": "data_model/1.7/device_types/ElectricalSensor.xml", "blob": "b6235fc64f44d4db2c42d05dc1e9b1f6f51dedfd"},
            {"path": "data_model/1.7/clusters/ElectricalPowerMeasurement.xml", "blob": "3349108215d56ad61a950a3ad874ea9773499cb5"},
            {"path": "zzz_generated/app-common/clusters/ElectricalPowerMeasurement/AttributeIds.h", "blob": "8f95703c7a97c8c86752695446220a900819fd28"},
            {"path": "examples/all-devices-app/all-devices-common/device/types/electrical-sensor/ElectricalSensor.cpp", "blob": "3f937a355870efc38f512ee40a38007ca4cc6bd3"},
        ],
    },
    "mapping": {
        "repository": "https://github.com/Project-Helianthus/helianthus-docs-semantic.git",
        "commit": DOCS_SEMANTIC_COMMIT,
        "path": "api/v1/targets/matter-1.7-ballot-0.9-v1.json",
        "sha256": MAPPING_SHA256,
    },
    "target": {
        "device_type": {"name": "Electrical Sensor", "id": "0x0510", "revision": 2},
        "power_topology_cluster": {"name": "Power Topology", "id": "0x009C", "revision": 2},
        "electrical_power_measurement_cluster": {"name": "Electrical Power Measurement", "id": "0x0090", "revision": 3},
        "active_current_attribute": {"name": "ActiveCurrent", "id": "0x0005", "optional": True},
        "linux_app": "examples/all-devices-app/posix",
        "controller": "examples/chip-tool",
    },
}

REQUIRED_BUILD_TARGETS = frozenset({"all_devices_app", "chip_tool", "electrical_power_measurement_tests", "power_topology_tests", "upstream_electrical_sensor_fixture"})
REQUIRED_CLUSTERS = {"0x001D": "Descriptor", "0x009C": "Power Topology", "0x0090": "Electrical Power Measurement"}


@dataclass(frozen=True)
class Decision:
    """A decision suitable for a caller to log without exposing credentials."""

    permitted: bool
    reasons: tuple[str, ...]


def _is_nonempty_text(value: Any) -> bool:
    return isinstance(value, str) and bool(value.strip())


def validate_lock(lock: Any) -> tuple[str, ...]:
    """Return drift reasons; strict equality prevents unreviewed lock changes."""
    if not isinstance(lock, dict):
        return ("lock is not an object",)
    if lock == EXPECTED_LOCK:
        return ()
    return ("lock differs from the reviewed Matter SDK baseline",)


def validate_build_evidence(lock: dict[str, Any], evidence: Any) -> tuple[str, ...]:
    if not isinstance(evidence, dict):
        return ("build evidence is absent or malformed",)
    if evidence.get("schema_version") != 1:
        return ("build evidence schema is unsupported",)
    if evidence.get("sdk_commit") != lock["sdk"]["commit"]:
        return ("build evidence is for a different SDK commit",)
    targets = evidence.get("targets")
    if not isinstance(targets, dict):
        return ("build evidence has no target results",)
    missing = REQUIRED_BUILD_TARGETS - set(targets)
    failed = sorted(name for name in REQUIRED_BUILD_TARGETS if targets.get(name) != "passed")
    reasons = [f"build evidence omits {name}" for name in sorted(missing)]
    reasons.extend(f"build target did not pass: {name}" for name in failed if name not in missing)
    return tuple(reasons)


def validate_composition(lock: dict[str, Any], composition: Any) -> tuple[str, ...]:
    if not isinstance(composition, dict):
        return ("accepted runtime-composition profile is absent or malformed",)
    required = {"schema_version", "status", "profile_id", "target_commit", "mapping", "device_type", "clusters", "electrical_power_measurement"}
    if set(composition) != required:
        return ("runtime-composition profile has an unreviewed shape",)
    reasons: list[str] = []
    if composition["schema_version"] != 1:
        reasons.append("runtime-composition schema is unsupported")
    if composition["status"] != "accepted":
        reasons.append("runtime-composition profile is not accepted")
    if not _is_nonempty_text(composition["profile_id"]):
        reasons.append("runtime-composition profile has no identity")
    if composition["target_commit"] != lock["sdk"]["commit"]:
        reasons.append("runtime-composition profile pins a different SDK commit")
    if composition["mapping"] != lock["mapping"]:
        reasons.append("runtime-composition profile pins a different mapping revision")
    if composition["device_type"] != lock["target"]["device_type"]:
        reasons.append("runtime-composition profile pins a different device type")

    clusters = composition["clusters"]
    if not isinstance(clusters, list):
        reasons.append("runtime-composition clusters are malformed")
    else:
        found = {item.get("id"): item for item in clusters if isinstance(item, dict) and isinstance(item.get("id"), str)}
        for cluster_id, name in REQUIRED_CLUSTERS.items():
            cluster = found.get(cluster_id)
            if not isinstance(cluster, dict) or cluster.get("name") != name or not _is_nonempty_text(cluster.get("evidence")):
                reasons.append(f"mandatory {name} composition metadata is absent or unevidenced")

    epm = composition["electrical_power_measurement"]
    if not isinstance(epm, dict) or set(epm) != {"power_mode", "number_of_measurement_types", "accuracy", "active_power"}:
        return tuple(reasons + ["Electrical Power Measurement metadata is malformed"])
    power_mode = epm["power_mode"]
    if not isinstance(power_mode, dict) or not _is_nonempty_text(power_mode.get("value")) or power_mode["value"].lower() == "unknown" or not _is_nonempty_text(power_mode.get("evidence")):
        reasons.append("mandatory PowerMode is absent, unknown, or unevidenced")
    count = epm["number_of_measurement_types"]
    if not isinstance(count, dict) or isinstance(count.get("value"), bool) or not isinstance(count.get("value"), int) or count["value"] < 1 or not _is_nonempty_text(count.get("evidence")):
        reasons.append("mandatory NumberOfMeasurementTypes is absent or unevidenced")
        expected_accuracy_count = None
    else:
        expected_accuracy_count = count["value"]
    accuracy = epm["accuracy"]
    if not isinstance(accuracy, list) or not accuracy:
        reasons.append("mandatory Accuracy list is empty or malformed")
    else:
        if expected_accuracy_count is not None and len(accuracy) != expected_accuracy_count:
            reasons.append("Accuracy does not match NumberOfMeasurementTypes")
        for item in accuracy:
            if not isinstance(item, dict) or not _is_nonempty_text(item.get("measurement_type")) or not _is_nonempty_text(item.get("evidence")):
                reasons.append("mandatory Accuracy entry is absent or unevidenced")
                break
    active_power = epm["active_power"]
    if not isinstance(active_power, dict) or set(active_power) != {"value", "evidence"} or not _is_nonempty_text(active_power.get("evidence")):
        reasons.append("mandatory nullable ActivePower metadata is absent or unevidenced")
    return tuple(reasons)


def _composition_digest(composition: Any) -> str:
    """Hash an exact canonical profile; a future reviewed lock may pin it."""
    canonical = json.dumps(composition, sort_keys=True, separators=(",", ":"), ensure_ascii=True)
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()


def evaluate(lock: Any, composition: Any, build_evidence: Any) -> Decision:
    """Permit endpoint allocation only when all reviewed evidence agrees."""
    lock_reasons = validate_lock(lock)
    if lock_reasons:
        return Decision(False, lock_reasons)
    assert isinstance(lock, dict)
    profile_pin = lock["accepted_runtime_composition_profile_sha256"]
    if profile_pin is None:
        return Decision(False, ("no reviewed accepted runtime-composition profile is pinned",))
    if not isinstance(profile_pin, str) or len(profile_pin) != 64:
        return Decision(False, ("accepted runtime-composition profile pin is malformed",))
    if _composition_digest(composition) != profile_pin:
        return Decision(False, ("runtime-composition profile differs from the reviewed pinned profile",))
    reasons = list(validate_build_evidence(lock, build_evidence))
    reasons.extend(validate_composition(lock, composition))
    return Decision(not reasons, tuple(reasons))
