"""Offline tests for persistent endpoint selection behind the readiness gate.

Only a fake store and generated synthetic readiness inputs are used. Admitted
cases patch the in-memory reviewed baseline with a digest computed from a
synthetic profile; the lock file on disk is never changed.
"""

from __future__ import annotations

import copy
import json
import unittest
from contextlib import contextmanager
from pathlib import Path
from typing import Iterator
from unittest import mock

from runtime import production_readiness
from runtime.endpoint_selection import (
    ELIGIBLE,
    LAST_ENDPOINT,
    LEDGER_KEY,
    RECORD_FIELDS,
    SELECTION_KEY,
    WITHHELD,
    EndpointSelector,
    SelectionStateError,
    StoreError,
)


ROOT = Path(__file__).resolve().parents[1]
LOCK = ROOT / "sdk" / "matter-sdk.lock.json"
PIN = "accepted_runtime_composition_profile_sha256"
NODE = "synthetic-node-a"
FABRIC_KEY = "sdk-fabric-storage"
FABRIC_BYTES = b"synthetic opaque sdk-owned bytes"
POWER = "synthetic.heat-generator-family.electrical-power"
CURRENT = "synthetic.heat-pump-family.electrical-power"
THIRD = "synthetic.ventilation-family.electrical-power"


def load_lock() -> dict:
    return json.loads(LOCK.read_text(encoding="utf-8"))


def synthetic_composition(lock: dict, profile_id: str = "synthetic-electrical-sensor-profile-a") -> dict:
    evidence = "synthetic-readiness-input"
    return {
        "schema_version": 1,
        "status": "accepted",
        "profile_id": profile_id,
        "target_commit": lock["sdk"]["commit"],
        "mapping": copy.deepcopy(lock["mapping"]),
        "device_type": copy.deepcopy(lock["target"]["device_type"]),
        "clusters": [
            {"id": "0x001D", "name": "Descriptor", "evidence": evidence},
            {"id": "0x009C", "name": "Power Topology", "evidence": evidence},
            {"id": "0x0090", "name": "Electrical Power Measurement", "evidence": evidence},
        ],
        "electrical_power_measurement": {
            "power_mode": {"value": "ac", "evidence": evidence},
            "number_of_measurement_types": {"value": 1, "evidence": evidence},
            "accuracy": [{"measurement_type": "active_power", "evidence": evidence}],
            "active_power": {"value": None, "evidence": evidence},
        },
    }


def synthetic_build(lock: dict) -> dict:
    return {
        "schema_version": 1,
        "sdk_commit": lock["sdk"]["commit"],
        "targets": {name: "passed" for name in sorted(production_readiness.REQUIRED_BUILD_TARGETS)},
    }


@contextmanager
def admitted(composition: dict) -> Iterator[dict]:
    """Yield a lock whose in-memory reviewed pin matches the synthetic profile."""
    digest = production_readiness._composition_digest(composition)
    lock = load_lock()
    lock[PIN] = digest
    with mock.patch.dict(production_readiness.EXPECTED_LOCK, {PIN: digest}):
        yield lock


class FakeStore:
    def __init__(self, initial: dict[str, bytes] | None = None) -> None:
        self.data: dict[str, bytes] = dict(initial or {})
        self.reads: list[str] = []
        self.writes: list[str] = []
        self.fail_writes: set[str] = set()

    def read(self, key: str) -> bytes | None:
        self.reads.append(key)
        return self.data.get(key)

    def write(self, key: str, data: bytes) -> None:
        self.writes.append(key)
        if key in self.fail_writes:
            raise StoreError("synthetic write failure")
        self.data[key] = bytes(data)


class Identities:
    def __init__(self, prefix: str = "synthetic-identity") -> None:
        self.prefix = prefix
        self.calls = 0

    def __call__(self) -> str:
        self.calls += 1
        return f"{self.prefix}-{self.calls:04d}"


def ledger(records: list[dict], node_id: str = NODE) -> bytes:
    return json.dumps({"schema_version": 1, "node_id": node_id, "records": records}).encode("utf-8")


def record(identity: str, capability: str, digest: str, endpoint: int) -> dict:
    return {"identity": identity, "capability": capability, "composition_sha256": digest, "endpoint": endpoint}


def by_capability(result) -> dict:
    return {state.capability: state for state in result.states}


class EndpointSelectionTest(unittest.TestCase):
    def setUp(self) -> None:
        self.lock = load_lock()
        self.composition = synthetic_composition(self.lock)
        self.build = synthetic_build(self.lock)
        self.digest = production_readiness._composition_digest(self.composition)
        self.store = FakeStore({FABRIC_KEY: FABRIC_BYTES})
        self.identities = Identities()

    def selector(self, store: FakeStore | None = None, node_id: str = NODE, identities: Identities | None = None) -> EndpointSelector:
        return EndpointSelector(store or self.store, node_id, new_identity=identities or self.identities)

    def reconcile_admitted(self, selector: EndpointSelector, composition: dict | None = None):
        composition = composition or self.composition
        with admitted(composition) as lock:
            self.assertTrue(production_readiness.evaluate(lock, composition, self.build).permitted)
            return selector.reconcile(lock, composition, self.build)

    def test_withheld_production_readiness_allocates_nothing(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        for attempt in range(2):
            result = self.selector().reconcile(self.lock, self.composition, self.build)
            self.assertEqual([s.capability for s in result.states], sorted([POWER, CURRENT]))
            for state in result.states:
                self.assertEqual(state.state, WITHHELD)
                self.assertIn("no reviewed accepted runtime-composition profile is pinned", state.reasons)
                self.assertIsNone(state.identity)
                self.assertIsNone(state.endpoint)
        self.assertNotIn(LEDGER_KEY, self.store.data)
        self.assertNotIn(LEDGER_KEY, self.store.writes)
        self.assertEqual(self.identities.calls, 0)
        self.assertIsNone(load_lock()[PIN])

    def test_absent_or_mismatched_inputs_stay_withheld_even_with_an_admitted_pin(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        other = synthetic_composition(self.lock, "synthetic-electrical-sensor-profile-b")
        with admitted(self.composition) as lock:
            for composition, build in ((None, self.build), (self.composition, None), (other, self.build)):
                state = selector.reconcile(lock, composition, build).states[0]
                self.assertEqual(state.state, WITHHELD)
                self.assertIsNone(state.endpoint)
        self.assertNotIn(LEDGER_KEY, self.store.writes)
        self.assertEqual(self.identities.calls, 0)

    def test_unselected_capabilities_are_not_evaluated_or_allocated(self) -> None:
        calls = []
        selector = EndpointSelector(self.store, NODE, readiness=lambda *args: calls.append(args), new_identity=self.identities)
        self.assertEqual(selector.reconcile(self.lock, self.composition, self.build).states, ())
        self.assertEqual(calls, [])
        self.assertEqual(self.store.writes, [])

    def test_non_decision_readiness_result_is_withheld(self) -> None:
        selector = EndpointSelector(self.store, NODE, readiness=lambda *args: True, new_identity=self.identities)
        selector.select(POWER)
        state = selector.reconcile(self.lock, self.composition, self.build).states[0]
        self.assertEqual(state.state, WITHHELD)
        self.assertEqual(self.identities.calls, 0)

    def test_admitted_allocation_is_persisted_before_eligibility(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        result = self.reconcile_admitted(selector)
        states = by_capability(result)
        self.assertEqual(result.faults, ())
        self.assertEqual({s.state for s in result.states}, {ELIGIBLE})
        self.assertNotEqual(states[POWER].identity, states[CURRENT].identity)
        self.assertEqual(sorted(s.endpoint for s in result.states), [1, 2])
        stored = json.loads(self.store.data[LEDGER_KEY])
        self.assertEqual(stored["node_id"], NODE)
        for item in stored["records"]:
            self.assertEqual(set(item), RECORD_FIELDS)
            self.assertEqual(item["composition_sha256"], self.digest)
            self.assertEqual((item["identity"], item["endpoint"]), (states[item["capability"]].identity, states[item["capability"]].endpoint))

    def test_identity_and_endpoint_survive_restart(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        writes = list(self.store.writes)
        restarted = self.selector(identities=Identities("synthetic-restart"))
        second = self.reconcile_admitted(restarted).states[0]
        self.assertEqual((second.state, second.identity, second.endpoint), (ELIGIBLE, first.identity, first.endpoint))
        self.assertEqual(self.store.writes, writes)

    def test_withheld_after_admission_keeps_the_ledger_intact(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        persisted = self.store.data[LEDGER_KEY]
        withheld = selector.reconcile(self.lock, self.composition, self.build).states[0]
        self.assertEqual((withheld.state, withheld.identity, withheld.endpoint), (WITHHELD, None, None))
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)
        again = self.reconcile_admitted(selector).states[0]
        self.assertEqual((again.identity, again.endpoint), (first.identity, first.endpoint))

    def test_deselect_and_reselect_reuse_identity(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        persisted = self.store.data[LEDGER_KEY]
        selector.deselect(POWER)
        self.assertEqual(self.reconcile_admitted(selector).states, ())
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)
        selector.select(POWER)
        again = self.reconcile_admitted(self.selector(identities=Identities("synthetic-reselect"))).states[0]
        self.assertEqual((again.state, again.identity, again.endpoint), (ELIGIBLE, first.identity, first.endpoint))

    def test_retired_endpoint_number_is_not_reassigned(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        selector.deselect(POWER)
        selector.select(CURRENT)
        second = self.reconcile_admitted(selector).states[0]
        self.assertEqual(second.capability, CURRENT)
        self.assertNotEqual(second.identity, first.identity)
        self.assertEqual(second.endpoint, first.endpoint + 1)

    def test_composition_change_withholds_without_new_identity(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        self.reconcile_admitted(selector)
        persisted = self.store.data[LEDGER_KEY]
        calls = self.identities.calls
        other = synthetic_composition(self.lock, "synthetic-electrical-sensor-profile-b")
        state = self.reconcile_admitted(selector, other).states[0]
        self.assertEqual((state.state, state.identity, state.endpoint), (WITHHELD, None, None))
        self.assertIn("different composition", state.reasons[0])
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)
        self.assertEqual(self.identities.calls, calls)

    def test_other_node_lifecycle_ledger_is_withheld_and_preserved(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        self.reconcile_admitted(selector)
        persisted = self.store.data[LEDGER_KEY]
        other_node = self.selector(node_id="synthetic-node-b")
        result = self.reconcile_admitted(other_node)
        self.assertEqual(result.states[0].state, WITHHELD)
        self.assertIn("different node lifecycle", result.faults[0])
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)

    def test_duplicate_capability_records_withhold_only_that_capability(self) -> None:
        records = [
            record("synthetic-identity-a001", POWER, self.digest, 1),
            record("synthetic-identity-a002", POWER, self.digest, 2),
            record("synthetic-identity-a003", CURRENT, self.digest, 3),
        ]
        self.store.data[LEDGER_KEY] = ledger(records)
        selector = self.selector()
        for capability in (POWER, CURRENT, THIRD):
            selector.select(capability)
        states = by_capability(self.reconcile_admitted(selector))
        self.assertEqual(states[POWER].state, WITHHELD)
        self.assertIn("capability has duplicate persisted endpoint records", states[POWER].reasons)
        self.assertEqual((states[CURRENT].state, states[CURRENT].identity, states[CURRENT].endpoint), (ELIGIBLE, "synthetic-identity-a003", 3))
        self.assertEqual((states[THIRD].state, states[THIRD].endpoint), (ELIGIBLE, 4))
        self.assertEqual(json.loads(self.store.data[LEDGER_KEY])["records"][:3], records)

    def test_duplicate_identity_withholds_both_records(self) -> None:
        self.store.data[LEDGER_KEY] = ledger([
            record("synthetic-identity-a001", POWER, self.digest, 1),
            record("synthetic-identity-a001", CURRENT, self.digest, 2),
        ])
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        for state in self.reconcile_admitted(selector).states:
            self.assertEqual(state.state, WITHHELD)
            self.assertIn("persisted identity is shared with another record", state.reasons)

    def test_duplicate_endpoint_withholds_both_and_new_number_skips_it(self) -> None:
        self.store.data[LEDGER_KEY] = ledger([
            record("synthetic-identity-a001", POWER, self.digest, 7),
            record("synthetic-identity-a002", CURRENT, self.digest, 7),
        ])
        selector = self.selector()
        for capability in (POWER, CURRENT, THIRD):
            selector.select(capability)
        states = by_capability(self.reconcile_admitted(selector))
        for capability in (POWER, CURRENT):
            self.assertEqual(states[capability].state, WITHHELD)
            self.assertIn("persisted endpoint number is shared with another record", states[capability].reasons)
        self.assertEqual(states[THIRD].endpoint, 8)

    def test_duplicate_handling_is_independent_of_record_order(self) -> None:
        records = [
            record("synthetic-identity-a001", POWER, self.digest, 1),
            record("synthetic-identity-a002", POWER, self.digest, 1),
            {"identity": "synthetic-identity-a003", "capability": CURRENT, "endpoint": 2},
            record("synthetic-identity-a004", CURRENT, self.digest, 3),
        ]
        outcomes = []
        for ordering in (records, list(reversed(records)), records[2:] + records[:2]):
            store = FakeStore({LEDGER_KEY: ledger(ordering)})
            selector = self.selector(store, identities=Identities())
            selector.select(POWER)
            selector.select(CURRENT)
            outcomes.append(self.reconcile_admitted(selector))
        self.assertEqual(outcomes[0], outcomes[1])
        self.assertEqual(outcomes[0], outcomes[2])

    def test_corrupt_ledger_withholds_everything_and_is_not_overwritten(self) -> None:
        variants = {
            "invalid utf-8": b"\xff\xfe",
            "invalid json": b"{not json",
            "duplicate key": b'{"schema_version":1,"node_id":"synthetic-node-a","node_id":"synthetic-node-a","records":[]}',
            "unsupported schema": json.dumps({"schema_version": 2, "node_id": NODE, "records": []}).encode(),
            "extra field": json.dumps({"schema_version": 1, "node_id": NODE, "records": [], "fabric": "x"}).encode(),
            "records not a list": json.dumps({"schema_version": 1, "node_id": NODE, "records": {}}).encode(),
            "node not text": json.dumps({"schema_version": 1, "node_id": 5, "records": []}).encode(),
        }
        for name, data in variants.items():
            with self.subTest(name):
                store = FakeStore({LEDGER_KEY: data})
                identities = Identities()
                selector = self.selector(store, identities=identities)
                selector.select(POWER)
                result = self.reconcile_admitted(selector)
                self.assertEqual(result.states[0].state, WITHHELD)
                self.assertTrue(result.faults)
                self.assertEqual(store.data[LEDGER_KEY], data)
                self.assertNotIn(LEDGER_KEY, store.writes)
                self.assertEqual(identities.calls, 0)

    def test_malformed_record_preserves_valid_restore_and_blocks_new_allocation(self) -> None:
        malformed = [
            {"identity": "synthetic-identity-a002", "capability": CURRENT, "composition_sha256": self.digest, "endpoint": True},
            {"identity": "short", "capability": CURRENT, "composition_sha256": self.digest, "endpoint": 2},
            {"identity": "synthetic-identity-a002", "capability": CURRENT, "composition_sha256": self.digest.upper(), "endpoint": 2},
            {"identity": "synthetic-identity-a002", "capability": CURRENT, "composition_sha256": self.digest, "endpoint": 0},
            {"identity": "synthetic-identity-a002", "capability": CURRENT, "composition_sha256": self.digest, "endpoint": 2, "fabric": "x"},
            {"identity": "synthetic-identity-a002", "capability": " ", "composition_sha256": self.digest, "endpoint": 2},
        ]
        for bad in malformed:
            with self.subTest(bad=bad):
                data = ledger([record("synthetic-identity-a001", POWER, self.digest, 1), bad])
                store = FakeStore({LEDGER_KEY: data})
                identities = Identities()
                selector = self.selector(store, identities=identities)
                for capability in (POWER, CURRENT, THIRD):
                    selector.select(capability)
                result = self.reconcile_admitted(selector)
                states = by_capability(result)
                self.assertEqual((states[POWER].state, states[POWER].identity, states[POWER].endpoint), (ELIGIBLE, "synthetic-identity-a001", 1))
                self.assertEqual(states[CURRENT].state, WITHHELD)
                self.assertEqual(states[THIRD].state, WITHHELD)
                self.assertTrue(result.faults)
                self.assertEqual(store.data[LEDGER_KEY], data)
                self.assertEqual(identities.calls, 0)

    def test_failed_ledger_write_is_not_exposed(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        self.store.fail_writes.add(LEDGER_KEY)
        result = self.reconcile_admitted(selector)
        self.assertEqual((result.states[0].state, result.states[0].endpoint), (WITHHELD, None))
        self.assertIn("endpoint ledger could not be persisted", result.states[0].reasons)
        self.assertNotIn(LEDGER_KEY, self.store.data)
        self.store.fail_writes.clear()
        state = self.reconcile_admitted(selector).states[0]
        self.assertEqual((state.state, state.endpoint), (ELIGIBLE, 1))

    def test_unreadable_ledger_is_withheld(self) -> None:
        class ReadFailure(FakeStore):
            def read(self, key: str) -> bytes | None:
                if key == LEDGER_KEY:
                    raise StoreError("synthetic read failure")
                return super().read(key)

        store = ReadFailure()
        selector = self.selector(store)
        selector.select(POWER)
        result = self.reconcile_admitted(selector)
        self.assertEqual(result.states[0].state, WITHHELD)
        self.assertNotIn(LEDGER_KEY, store.writes)

    def test_generated_identity_collision_or_malformed_value_is_withheld(self) -> None:
        self.store.data[LEDGER_KEY] = ledger([record("synthetic-identity-0001", POWER, self.digest, 1)])
        for generated in ("synthetic-identity-0001", "bad id"):
            with self.subTest(generated=generated):
                selector = EndpointSelector(self.store, NODE, new_identity=lambda: generated)
                selector.select(CURRENT)
                state = by_capability(self.reconcile_admitted(selector))[CURRENT]
                self.assertEqual((state.state, state.endpoint), (WITHHELD, None))
                self.assertEqual(len(json.loads(self.store.data[LEDGER_KEY])["records"]), 1)

    def test_exhausted_endpoint_space_is_withheld(self) -> None:
        self.store.data[LEDGER_KEY] = ledger([record("synthetic-identity-a001", POWER, self.digest, LAST_ENDPOINT)])
        selector = self.selector()
        selector.select(CURRENT)
        state = by_capability(self.reconcile_admitted(selector))[CURRENT]
        self.assertEqual(state.state, WITHHELD)
        self.assertIn("exhausted", state.reasons[0])

    def test_corrupt_selection_is_not_overwritten(self) -> None:
        variants = (b"{not json", json.dumps({"schema_version": 1, "selected": [POWER, POWER]}).encode(), json.dumps({"schema_version": 1, "selected": [""]}).encode())
        for data in variants:
            with self.subTest(data=data):
                store = FakeStore({SELECTION_KEY: data})
                selector = self.selector(store)
                with self.assertRaises(SelectionStateError):
                    selector.select(POWER)
                with self.assertRaises(SelectionStateError):
                    selector.deselect(POWER)
                result = self.reconcile_admitted(selector)
                self.assertEqual(result.states, ())
                self.assertTrue(result.faults)
                self.assertEqual(store.data[SELECTION_KEY], data)
                self.assertEqual(store.writes, [])

    def test_invalid_capability_or_node_is_rejected(self) -> None:
        for capability in ("", " padded", None, "x" * 257):
            with self.subTest(capability=capability), self.assertRaises(ValueError):
                self.selector().select(capability)  # type: ignore[arg-type]
        with self.assertRaises(ValueError):
            EndpointSelector(self.store, "")
        self.assertEqual(self.store.writes, [])

    def test_lifecycle_never_touches_sdk_fabric_storage(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        selector.reconcile(self.lock, self.composition, self.build)
        self.reconcile_admitted(selector)
        selector.deselect(POWER)
        self.reconcile_admitted(selector)
        selector.select(POWER)
        self.reconcile_admitted(self.selector(identities=Identities("synthetic-restart")))
        self.assertEqual(self.store.data[FABRIC_KEY], FABRIC_BYTES)
        self.assertNotIn(FABRIC_KEY, self.store.reads)
        self.assertNotIn(FABRIC_KEY, self.store.writes)
        self.assertEqual(set(self.store.writes), {SELECTION_KEY, LEDGER_KEY})
        self.assertEqual(set(json.loads(self.store.data[LEDGER_KEY])), {"schema_version", "node_id", "records"})


if __name__ == "__main__":
    unittest.main()
