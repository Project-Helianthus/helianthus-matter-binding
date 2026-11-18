"""Offline tests for persistent endpoint selection behind the readiness gate.

Only a fake store, a synthetic Matter target report and generated synthetic
readiness inputs are used. Admitted cases patch the in-memory reviewed baseline
with a digest computed from a synthetic profile; the lock file on disk is never
changed.
"""

from __future__ import annotations

import copy
import dataclasses
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
    Offer,
    RelationInstance,
    RelationKey,
    SelectionStateError,
    StoreError,
    TargetReport,
)


ROOT = Path(__file__).resolve().parents[1]
LOCK = ROOT / "sdk" / "matter-sdk.lock.json"
PIN = "accepted_runtime_composition_profile_sha256"
NODE = "synthetic-node-a"
FABRIC_KEY = "sdk-fabric-storage"
FABRIC_BYTES = b"synthetic opaque sdk-owned bytes"
PORTAL_KEY = "portal-catalog"
PORTAL_BYTES = b'{"contributions":[{"resource_id":"synthetic-resource-a","label":"Synthetic heat generator"}]}'
POWER = "synthetic.heat-generator-family.electrical-power"
CURRENT = "synthetic.heat-pump-family.electrical-power"
THIRD = "synthetic.ventilation-family.electrical-power"
STATUS = "synthetic.heat-generator-family.operating-status"
RES_A = "synthetic-resource-a"
RES_B = "synthetic-resource-b"
RES_C = "synthetic-resource-c"
RES_REPLACEMENT = "synthetic-resource-a-replacement"
LENS = "synthetic.lens.matter-electrical"


def load_lock() -> dict:
    """Admission fixtures use the unchanged in-module baseline. The file pin is separate."""
    return copy.deepcopy(production_readiness.EXPECTED_LOCK)


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


def relation_key(
    whole: str,
    part: str,
    *,
    pack_id: str = "synthetic.pack.physical",
    pack_version: str = "1.0.0",
    definition: str = "synthetic.relation.contains",
    version: str = "1.0.0",
) -> RelationKey:
    return RelationKey(pack_id, pack_version, definition, version, whole, part)


def instance(key: RelationKey, core_type: str = "contains", coherence: str = "coherent") -> RelationInstance:
    return RelationInstance(key, core_type, coherence)


def offer(capability: str, resource_id: str, anchor: str = "synthetic-anchor-operator-slot", relations: tuple = ()) -> Offer:
    return Offer(capability, resource_id, anchor, tuple(relations))


def target_report(*offers: Offer, relations: tuple = (), lens: str = LENS) -> TargetReport:
    return TargetReport(lens, tuple(offers), tuple(relations))


def default_report() -> TargetReport:
    return target_report(offer(POWER, RES_A), offer(CURRENT, RES_B), offer(THIRD, RES_C))


def ledger(records: list[dict], node_id: str = NODE) -> bytes:
    return json.dumps({"schema_version": 1, "node_id": node_id, "records": records}).encode("utf-8")


def record(resource_id: str, digest: str, endpoint: int) -> dict:
    return {"resource_id": resource_id, "composition_sha256": digest, "endpoint": endpoint}


def by_capability(result) -> dict:
    return {state.capability: state for state in result.states}


def placement(state) -> tuple:
    return (state.state, state.resource_id, state.endpoint, state.parent_endpoint)


class EndpointSelectionTest(unittest.TestCase):
    def setUp(self) -> None:
        self.lock = load_lock()
        self.composition = synthetic_composition(self.lock)
        self.build = synthetic_build(self.lock)
        self.digest = production_readiness._composition_digest(self.composition)
        self.store = FakeStore({FABRIC_KEY: FABRIC_BYTES, PORTAL_KEY: PORTAL_BYTES})
        self.report = default_report()

    def selector(self, store: FakeStore | None = None, node_id: str = NODE) -> EndpointSelector:
        return EndpointSelector(store or self.store, node_id)

    def reconcile_admitted(self, selector: EndpointSelector, composition: dict | None = None, report: TargetReport | None = None):
        composition = composition or self.composition
        with admitted(composition) as lock:
            self.assertTrue(production_readiness.evaluate(lock, composition, self.build).permitted)
            return selector.reconcile(lock, composition, self.build, report or self.report)

    def stored_records(self, store: FakeStore | None = None) -> list[dict]:
        return json.loads((store or self.store).data[LEDGER_KEY])["records"]

    # Production readiness stays deny-all.

    def test_withheld_production_readiness_allocates_nothing(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        for _ in range(2):
            result = self.selector().reconcile(self.lock, self.composition, self.build, self.report)
            self.assertEqual([s.capability for s in result.states], sorted([POWER, CURRENT]))
            for state in result.states:
                self.assertEqual(state.state, WITHHELD)
                self.assertIn("no reviewed accepted runtime-composition profile is pinned", state.reasons)
                self.assertEqual((state.resource_id, state.endpoint, state.parent_endpoint), (None, None, None))
        self.assertNotIn(LEDGER_KEY, self.store.data)
        self.assertNotIn(LEDGER_KEY, self.store.writes)
        disk = json.loads(LOCK.read_text(encoding="utf-8"))
        self.assertIsNone(disk[PIN])
        self.assertFalse(production_readiness.evaluate(disk, self.composition, self.build).permitted)

    def test_absent_or_mismatched_inputs_stay_withheld_even_with_an_admitted_pin(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        other = synthetic_composition(self.lock, "synthetic-electrical-sensor-profile-b")
        with admitted(self.composition) as lock:
            for composition, build in ((None, self.build), (self.composition, None), (other, self.build)):
                state = selector.reconcile(lock, composition, build, self.report).states[0]
                self.assertEqual(state.state, WITHHELD)
                self.assertIsNone(state.endpoint)
        self.assertNotIn(LEDGER_KEY, self.store.writes)

    def test_unselected_capabilities_are_not_evaluated_or_allocated(self) -> None:
        calls = []
        selector = EndpointSelector(self.store, NODE, readiness=lambda *args: calls.append(args))
        self.assertEqual(selector.reconcile(self.lock, self.composition, self.build, self.report).states, ())
        self.assertEqual(calls, [])
        self.assertEqual(self.store.writes, [])

    def test_non_decision_readiness_result_is_withheld(self) -> None:
        selector = EndpointSelector(self.store, NODE, readiness=lambda *args: True)
        selector.select(POWER)
        state = selector.reconcile(self.lock, self.composition, self.build, self.report).states[0]
        self.assertEqual(state.state, WITHHELD)
        self.assertNotIn(LEDGER_KEY, self.store.writes)

    # A ledger record carries the resource and the composition.

    def test_ledger_record_carries_resource_and_composition(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        result = self.reconcile_admitted(selector)
        states = by_capability(result)
        self.assertEqual(result.faults, ())
        self.assertEqual(placement(states[POWER]), (ELIGIBLE, RES_A, 1, None))
        self.assertEqual(placement(states[CURRENT]), (ELIGIBLE, RES_B, 2, None))
        self.assertEqual(RECORD_FIELDS, frozenset({"resource_id", "composition_sha256", "endpoint"}))
        stored = json.loads(self.store.data[LEDGER_KEY])
        self.assertEqual(stored["node_id"], NODE)
        self.assertEqual(stored["records"], [record(RES_A, self.digest, 1), record(RES_B, self.digest, 2)])

    def test_capabilities_on_one_resource_share_its_endpoint(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(STATUS)
        report = target_report(offer(POWER, RES_A), offer(STATUS, RES_A))
        states = by_capability(self.reconcile_admitted(selector, report=report))
        self.assertEqual(placement(states[POWER]), (ELIGIBLE, RES_A, 1, None))
        self.assertEqual(placement(states[STATUS]), (ELIGIBLE, RES_A, 1, None))
        self.assertEqual(self.stored_records(), [record(RES_A, self.digest, 1)])

    def test_endpoint_survives_restart(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        writes = list(self.store.writes)
        second = self.reconcile_admitted(self.selector()).states[0]
        self.assertEqual(placement(second), placement(first))
        self.assertEqual(self.store.writes, writes)

    def test_anchor_supersession_keeps_the_endpoint(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        before = target_report(offer(POWER, RES_A, anchor="synthetic-anchor-operator-slot"))
        first = self.reconcile_admitted(selector, report=before).states[0]
        persisted = self.store.data[LEDGER_KEY]
        after = target_report(offer(POWER, RES_A, anchor="synthetic-anchor-native-identity"))
        second = self.reconcile_admitted(self.selector(), report=after).states[0]
        self.assertEqual(placement(second), (ELIGIBLE, RES_A, first.endpoint, None))
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)

    def test_replaced_equipment_gets_a_new_endpoint_and_the_old_number_stays_retired(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        first = by_capability(self.reconcile_admitted(selector))
        replaced = target_report(offer(POWER, RES_REPLACEMENT, anchor="synthetic-anchor-replacement-unit"), offer(CURRENT, RES_B))
        second = by_capability(self.reconcile_admitted(selector, report=replaced))
        self.assertEqual(placement(second[POWER]), (ELIGIBLE, RES_REPLACEMENT, 3, None))
        self.assertEqual(placement(second[CURRENT]), placement(first[CURRENT]))
        self.assertEqual(self.stored_records(), [record(RES_A, self.digest, 1), record(RES_B, self.digest, 2), record(RES_REPLACEMENT, self.digest, 3)])
        back = by_capability(self.reconcile_admitted(selector))
        self.assertEqual(placement(back[POWER]), placement(first[POWER]))

    def test_withheld_after_admission_keeps_the_ledger_intact(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        persisted = self.store.data[LEDGER_KEY]
        withheld = selector.reconcile(self.lock, self.composition, self.build, self.report).states[0]
        self.assertEqual(placement(withheld), (WITHHELD, None, None, None))
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)
        again = self.reconcile_admitted(selector).states[0]
        self.assertEqual(placement(again), placement(first))

    def test_deselect_and_reselect_reuse_the_endpoint(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        persisted = self.store.data[LEDGER_KEY]
        selector.deselect(POWER)
        self.assertEqual(self.reconcile_admitted(selector).states, ())
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)
        selector.select(POWER)
        again = self.reconcile_admitted(self.selector()).states[0]
        self.assertEqual(placement(again), placement(first))

    def test_retired_endpoint_number_is_not_reassigned(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        first = self.reconcile_admitted(selector).states[0]
        selector.deselect(POWER)
        selector.select(CURRENT)
        second = self.reconcile_admitted(selector).states[0]
        self.assertEqual((second.capability, second.resource_id), (CURRENT, RES_B))
        self.assertEqual(second.endpoint, first.endpoint + 1)

    def test_composition_change_withholds_and_keeps_the_record(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        self.reconcile_admitted(selector)
        persisted = self.store.data[LEDGER_KEY]
        other = synthetic_composition(self.lock, "synthetic-electrical-sensor-profile-b")
        state = self.reconcile_admitted(selector, other).states[0]
        self.assertEqual(placement(state), (WITHHELD, None, None, None))
        self.assertIn("different composition", state.reasons[0])
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)

    def test_other_node_lifecycle_ledger_is_withheld_and_preserved(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        self.reconcile_admitted(selector)
        persisted = self.store.data[LEDGER_KEY]
        result = self.reconcile_admitted(self.selector(node_id="synthetic-node-b"))
        self.assertEqual(result.states[0].state, WITHHELD)
        self.assertIn("different node lifecycle", result.faults[0])
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)

    # Parent and child endpoints follow the cited relations.

    def composed_report(self, part_relations: tuple, relations: tuple, whole_relations: tuple = ()) -> TargetReport:
        return target_report(offer(POWER, RES_A, relations=whole_relations), offer(CURRENT, RES_B, relations=part_relations), relations=relations)

    def test_parent_and_child_endpoints_follow_the_cited_relation(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        key = relation_key(RES_A, RES_B)
        states = by_capability(self.reconcile_admitted(selector, report=self.composed_report((key,), (instance(key),))))
        self.assertEqual(placement(states[POWER]), (ELIGIBLE, RES_A, 1, None))
        self.assertEqual(placement(states[CURRENT]), (ELIGIBLE, RES_B, 2, 1))
        self.assertEqual(states[CURRENT].reasons, ())
        self.assertEqual(self.stored_records(), [record(RES_A, self.digest, 1), record(RES_B, self.digest, 2)])

    def test_parent_relation_cited_by_the_whole_also_composes(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        key = relation_key(RES_A, RES_B)
        report = self.composed_report((), (instance(key),), whole_relations=(key,))
        self.assertEqual(by_capability(self.reconcile_admitted(selector, report=report))[CURRENT].parent_endpoint, 1)

    def test_every_relation_key_field_must_match_the_published_instance(self) -> None:
        """Fields varied: pack_id, pack_version, definition, version, from_resource, to_resource."""
        cited = relation_key(RES_A, RES_B)
        published = {
            "pack_id": dataclasses.replace(cited, pack_id="synthetic.pack.other"),
            "pack_version": dataclasses.replace(cited, pack_version="1.0.1"),
            "definition": dataclasses.replace(cited, definition="synthetic.relation.other"),
            "version": dataclasses.replace(cited, version="1.0.1"),
            "from_resource": dataclasses.replace(cited, from_resource=RES_C),
            "to_resource": dataclasses.replace(cited, to_resource=RES_C),
        }
        for name, key in published.items():
            with self.subTest(field=name):
                store = FakeStore()
                selector = self.selector(store)
                for capability in (POWER, CURRENT, THIRD):
                    selector.select(capability)
                report = target_report(offer(POWER, RES_A), offer(CURRENT, RES_B, relations=(cited,)), offer(THIRD, RES_C), relations=(instance(key),))
                states = by_capability(self.reconcile_admitted(selector, report=report))
                self.assertEqual(placement(states[CURRENT]), (ELIGIBLE, RES_B, 2, None))
                self.assertIn("not a coherent published instance", states[CURRENT].reasons[0])
                self.assertEqual(states[THIRD].parent_endpoint, None)

    def test_unusable_parent_relations_present_the_endpoint_under_no_parent(self) -> None:
        key = relation_key(RES_A, RES_B)
        second = relation_key(RES_C, RES_B, definition="synthetic.relation.contains-other")
        cases = {
            "uncited": ((), (instance(key),), None),
            "not published": ((key,), (), "not a coherent published instance"),
            "not coherent": ((key,), (instance(key, coherence="conflict"),), "not a coherent published instance"),
            "duplicate instance": ((key,), (instance(key), instance(key, coherence="candidate")), "not a coherent published instance"),
            "two parents": ((key, second), (instance(key), instance(second)), "more than one parent"),
            "contains and hosts": ((key, second), (instance(key), instance(second, core_type="hosts")), "more than one parent"),
        }
        for name, (cited, relations, reason) in cases.items():
            with self.subTest(name):
                store = FakeStore()
                selector = self.selector(store)
                for capability in (POWER, CURRENT, THIRD):
                    selector.select(capability)
                report = target_report(offer(POWER, RES_A), offer(CURRENT, RES_B, relations=cited), offer(THIRD, RES_C), relations=relations)
                result = self.reconcile_admitted(selector, report=report)
                state = by_capability(result)[CURRENT]
                self.assertEqual(placement(state), (ELIGIBLE, RES_B, 2, None))
                if reason is None:
                    self.assertEqual(state.reasons, ())
                else:
                    self.assertIn(reason, state.reasons[0])
                self.assertEqual(self.stored_records(store), [record(RES_A, self.digest, 1), record(RES_B, self.digest, 2), record(RES_C, self.digest, 3)])

    def test_only_forest_relations_make_a_parent(self) -> None:
        key = relation_key(RES_A, RES_B)
        for core_type, parent in (("contains", 1), ("hosts", 1), ("groups", None), ("measures", None), ("supplies", None), ("controls", None), ("relays", None)):
            with self.subTest(core_type=core_type):
                store = FakeStore()
                selector = self.selector(store)
                selector.select(POWER)
                selector.select(CURRENT)
                report = self.composed_report((key,), (instance(key, core_type=core_type),))
                state = by_capability(self.reconcile_admitted(selector, report=report))[CURRENT]
                self.assertEqual(placement(state), (ELIGIBLE, RES_B, 2, parent))
                self.assertEqual(state.reasons, ())

    def test_parent_without_an_eligible_endpoint_is_not_used(self) -> None:
        selector = self.selector()
        selector.select(CURRENT)
        key = relation_key(RES_A, RES_B)
        state = self.reconcile_admitted(selector, report=self.composed_report((key,), (instance(key),))).states[0]
        self.assertEqual(placement(state), (ELIGIBLE, RES_B, 1, None))
        self.assertIn("parent resource has no eligible endpoint", state.reasons[0])

    def test_relation_cycle_presents_its_members_under_no_parent(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        down, up = relation_key(RES_A, RES_B), relation_key(RES_B, RES_A)
        report = self.composed_report((down,), (instance(down), instance(up)), whole_relations=(up,))
        states = by_capability(self.reconcile_admitted(selector, report=report))
        for capability in (POWER, CURRENT):
            self.assertEqual(states[capability].parent_endpoint, None)
            self.assertIn("cycle", states[capability].reasons[0])

    def test_changed_relations_move_the_child_without_rewriting_the_ledger(self) -> None:
        selector = self.selector()
        for capability in (POWER, CURRENT, THIRD):
            selector.select(capability)
        under_a = relation_key(RES_A, RES_B)
        report_a = target_report(offer(POWER, RES_A), offer(CURRENT, RES_B, relations=(under_a,)), offer(THIRD, RES_C), relations=(instance(under_a),))
        first = by_capability(self.reconcile_admitted(selector, report=report_a))
        persisted = self.store.data[LEDGER_KEY]
        under_c = relation_key(RES_C, RES_B)
        report_c = target_report(offer(POWER, RES_A), offer(CURRENT, RES_B, relations=(under_c,)), offer(THIRD, RES_C), relations=(instance(under_c),))
        second = by_capability(self.reconcile_admitted(selector, report=report_c))
        self.assertEqual(first[CURRENT].parent_endpoint, first[POWER].endpoint)
        self.assertEqual(second[CURRENT].parent_endpoint, second[THIRD].endpoint)
        self.assertEqual(second[CURRENT].endpoint, first[CURRENT].endpoint)
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)

    # Withdrawing a Portal contribution leaves the ledger unchanged.

    def test_withdrawing_a_portal_contribution_leaves_the_ledger_unchanged(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        key = relation_key(RES_A, RES_B)
        report = self.composed_report((key,), (instance(key),))
        first = self.reconcile_admitted(selector, report=report)
        persisted = self.store.data[LEDGER_KEY]
        del self.store.data[PORTAL_KEY]
        second = self.reconcile_admitted(self.selector(), report=report)
        self.store.data[PORTAL_KEY] = b"{not a portal catalog"
        third = self.reconcile_admitted(self.selector(), report=report)
        self.assertEqual(second, first)
        self.assertEqual(third, first)
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)
        self.assertNotIn(PORTAL_KEY, self.store.reads)
        self.assertNotIn(PORTAL_KEY, self.store.writes)

    def test_offer_absent_from_the_target_report_is_withheld_and_keeps_its_record(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        first = by_capability(self.reconcile_admitted(selector))
        persisted = self.store.data[LEDGER_KEY]
        state = by_capability(self.reconcile_admitted(selector, report=target_report(offer(CURRENT, RES_B))))[POWER]
        self.assertEqual(placement(state), (WITHHELD, None, None, None))
        self.assertIn("no offer", state.reasons[0])
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)
        again = by_capability(self.reconcile_admitted(selector))
        self.assertEqual(placement(again[POWER]), placement(first[POWER]))

    def test_malformed_or_ambiguous_offers_are_withheld(self) -> None:
        bad_key = relation_key(RES_A, RES_B, pack_id=" padded")
        cases = {
            "resource not opaque": offer(POWER, "synthetic resource"),
            "resource empty": offer(POWER, ""),
            "anchor empty": offer(POWER, RES_A, anchor=""),
            "relation not a key": Offer(POWER, RES_A, "synthetic-anchor", ("synthetic-relation",)),
            "relation key malformed": offer(POWER, RES_A, relations=(bad_key,)),
            "relations not a tuple": Offer(POWER, RES_A, "synthetic-anchor", [relation_key(RES_A, RES_B)]),
        }
        for name, bad in cases.items():
            with self.subTest(name):
                store = FakeStore()
                selector = self.selector(store)
                selector.select(POWER)
                selector.select(CURRENT)
                states = by_capability(self.reconcile_admitted(selector, report=target_report(bad, offer(CURRENT, RES_B))))
                self.assertEqual(placement(states[POWER]), (WITHHELD, None, None, None))
                self.assertEqual(placement(states[CURRENT]), (ELIGIBLE, RES_B, 1, None))
        store = FakeStore()
        selector = self.selector(store)
        selector.select(POWER)
        duplicate = target_report(offer(POWER, RES_A), offer(POWER, RES_B))
        state = self.reconcile_admitted(selector, report=duplicate).states[0]
        self.assertEqual(state.state, WITHHELD)
        self.assertIn("more than one offer", state.reasons[0])
        self.assertNotIn(LEDGER_KEY, store.writes)

    def test_malformed_target_report_withholds_everything(self) -> None:
        cases = {
            "absent": None,
            "not a report": {"lens": LENS, "offers": [], "relations": []},
            "lens empty": target_report(offer(POWER, RES_A), lens=""),
            "offers not a tuple": TargetReport(LENS, [offer(POWER, RES_A)], ()),
            "offer not an offer": TargetReport(LENS, ({"capability": POWER},), ()),
            "relations not a tuple": TargetReport(LENS, (offer(POWER, RES_A),), []),
        }
        for name, report in cases.items():
            with self.subTest(name):
                store = FakeStore()
                selector = self.selector(store)
                selector.select(POWER)
                with admitted(self.composition) as lock:
                    state = selector.reconcile(lock, self.composition, self.build, report).states[0]
                self.assertEqual(placement(state), (WITHHELD, None, None, None))
                self.assertIn("target report", state.reasons[0])
                self.assertNotIn(LEDGER_KEY, store.writes)

    def test_malformed_relation_instances_are_never_used(self) -> None:
        key = relation_key(RES_A, RES_B)
        for bad in ("synthetic-relation", RelationInstance(key, "", "coherent"), RelationInstance(key, "contains", " "), RelationInstance("synthetic-key", "contains", "coherent")):
            with self.subTest(bad=bad):
                store = FakeStore()
                selector = self.selector(store)
                selector.select(POWER)
                selector.select(CURRENT)
                state = by_capability(self.reconcile_admitted(selector, report=self.composed_report((key,), (bad,))))[CURRENT]
                self.assertEqual(placement(state), (ELIGIBLE, RES_B, 2, None))
                self.assertIn("not a coherent published instance", state.reasons[0])

    # Persisted ledger faults stay fail closed.

    def test_duplicate_resource_records_withhold_only_that_resource(self) -> None:
        records = [record(RES_A, self.digest, 1), record(RES_A, self.digest, 2), record(RES_B, self.digest, 3)]
        self.store.data[LEDGER_KEY] = ledger(records)
        selector = self.selector()
        for capability in (POWER, CURRENT, THIRD):
            selector.select(capability)
        states = by_capability(self.reconcile_admitted(selector))
        self.assertEqual(states[POWER].state, WITHHELD)
        self.assertIn("resource has duplicate persisted endpoint records", states[POWER].reasons)
        self.assertEqual(placement(states[CURRENT]), (ELIGIBLE, RES_B, 3, None))
        self.assertEqual(placement(states[THIRD]), (ELIGIBLE, RES_C, 4, None))
        self.assertEqual(self.stored_records()[:3], records)

    def test_duplicate_endpoint_withholds_both_and_new_number_skips_it(self) -> None:
        self.store.data[LEDGER_KEY] = ledger([record(RES_A, self.digest, 7), record(RES_B, self.digest, 7)])
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
            record(RES_A, self.digest, 1),
            record(RES_A, self.digest, 2),
            {"resource_id": RES_B, "endpoint": 3},
            record(RES_B, self.digest, 4),
        ]
        outcomes = []
        for ordering in (records, list(reversed(records)), records[2:] + records[:2]):
            store = FakeStore({LEDGER_KEY: ledger(ordering)})
            selector = self.selector(store)
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
                selector = self.selector(store)
                selector.select(POWER)
                result = self.reconcile_admitted(selector)
                self.assertEqual(result.states[0].state, WITHHELD)
                self.assertTrue(result.faults)
                self.assertEqual(store.data[LEDGER_KEY], data)
                self.assertNotIn(LEDGER_KEY, store.writes)

    def test_malformed_record_preserves_valid_restore_and_blocks_new_allocation(self) -> None:
        malformed = [
            {"resource_id": RES_B, "composition_sha256": self.digest, "endpoint": True},
            {"resource_id": "synthetic resource", "composition_sha256": self.digest, "endpoint": 2},
            {"resource_id": RES_B, "composition_sha256": self.digest.upper(), "endpoint": 2},
            {"resource_id": RES_B, "composition_sha256": self.digest, "endpoint": 0},
            {"resource_id": RES_B, "composition_sha256": self.digest, "endpoint": 2, "fabric": "x"},
            {"resource_id": "", "composition_sha256": self.digest, "endpoint": 2},
            {"identity": "synthetic-identity-a002", "capability": CURRENT, "composition_sha256": self.digest, "endpoint": 2},
        ]
        for bad in malformed:
            with self.subTest(bad=bad):
                data = ledger([record(RES_A, self.digest, 1), bad])
                store = FakeStore({LEDGER_KEY: data})
                selector = self.selector(store)
                for capability in (POWER, CURRENT, THIRD):
                    selector.select(capability)
                result = self.reconcile_admitted(selector)
                states = by_capability(result)
                self.assertEqual(placement(states[POWER]), (ELIGIBLE, RES_A, 1, None))
                self.assertEqual(states[CURRENT].state, WITHHELD)
                self.assertEqual(states[THIRD].state, WITHHELD)
                self.assertTrue(result.faults)
                self.assertEqual(store.data[LEDGER_KEY], data)

    def test_failed_ledger_write_is_not_exposed(self) -> None:
        selector = self.selector()
        selector.select(POWER)
        self.store.fail_writes.add(LEDGER_KEY)
        result = self.reconcile_admitted(selector)
        self.assertEqual(placement(result.states[0]), (WITHHELD, None, None, None))
        self.assertIn("endpoint ledger could not be persisted", result.states[0].reasons)
        self.assertNotIn(LEDGER_KEY, self.store.data)
        self.store.fail_writes.clear()
        state = self.reconcile_admitted(selector).states[0]
        self.assertEqual(placement(state), (ELIGIBLE, RES_A, 1, None))

    def test_failed_ledger_write_never_presents_a_parent(self) -> None:
        self.store.data[LEDGER_KEY] = ledger([record(RES_A, self.digest, 1)])
        self.store.fail_writes.add(LEDGER_KEY)
        selector = self.selector()
        selector.select(POWER)
        selector.select(CURRENT)
        key = relation_key(RES_A, RES_B)
        states = by_capability(self.reconcile_admitted(selector, report=self.composed_report((key,), (instance(key),))))
        self.assertEqual(placement(states[POWER]), (ELIGIBLE, RES_A, 1, None))
        self.assertEqual(placement(states[CURRENT]), (WITHHELD, None, None, None))

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

    def test_exhausted_endpoint_space_is_withheld(self) -> None:
        self.store.data[LEDGER_KEY] = ledger([record(RES_A, self.digest, LAST_ENDPOINT)])
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
        selector.reconcile(self.lock, self.composition, self.build, self.report)
        self.reconcile_admitted(selector)
        selector.deselect(POWER)
        self.reconcile_admitted(selector)
        selector.select(POWER)
        self.reconcile_admitted(self.selector())
        self.assertEqual(self.store.data[FABRIC_KEY], FABRIC_BYTES)
        self.assertNotIn(FABRIC_KEY, self.store.reads)
        self.assertNotIn(FABRIC_KEY, self.store.writes)
        self.assertEqual(set(self.store.writes), {SELECTION_KEY, LEDGER_KEY})
        self.assertEqual(set(json.loads(self.store.data[LEDGER_KEY])), {"schema_version", "node_id", "records"})


if __name__ == "__main__":
    unittest.main()
