"""Offline tests for the publication rule and its use by the endpoint selector.

Only a synthetic roster, a synthetic Matter target report, a fake store and an
allowing readiness evaluator are used; the production evaluator is exercised
unchanged to show that it still withholds every resource.
"""

from __future__ import annotations

import copy
import json
import re
import tempfile
import unittest
from pathlib import Path

from runtime import production_readiness
from runtime.endpoint_selection import ELIGIBLE, LEDGER_KEY, WITHHELD, EndpointSelector, Offer, TargetReport
from runtime.production_readiness import Decision
from runtime.publication_rule import (
    EXCLUSIONS_PATH,
    KIND_RELATION,
    REASON_CAPACITY,
    REASON_EXCLUDED,
    REASON_EXHAUSTED,
    REASON_NO_DISPOSITION,
    REASON_RELATION,
    REASON_RESOURCE_ID,
    REASON_UNMAPPED,
    REASON_UNRECOGNISED,
    STORE_CAPACITY,
    TRANSFORMED,
    UNKNOWN,
    UNSUPPORTED,
    WITHHELD as WITHHELD_DISPOSITION,
    ExclusionListError,
    PublicationRule,
    RosterRow,
    allocation_refusal,
    load_exclusions,
    parse_exclusions,
)


ROOT = Path(__file__).resolve().parents[1]
GO_RULE = ROOT / "publication" / "rule.go"

CURRENT = "synthetic.heat-pump-family.electrical-current"
POWER = "synthetic.heat-pump-family.electrical-power"
STATUS = "synthetic.heat-generator-family.operating-status"
EXCLUDED = "synthetic.ventilation-family.fan-speed"
SERVICE = "synthetic.heat-generator-family.service"
CONTAINS = "synthetic.heat-generator-family.contains"
BLANK = "synthetic.heat-generator-family.no-disposition"
HOSTS = "synthetic.heat-pump-family.hosts"
UNMAPPED = "synthetic.heat-pump-family.unmapped"

RES_A = "synthetic-resource-a"
RES_B = "synthetic-resource-b"
RES_C = "synthetic-resource-c"
RES_D = "synthetic-resource-d"
NODE = "synthetic-node-a"
LENS = "synthetic.lens.matter-electrical"
COMPOSITION = {"profile_id": "synthetic-allowing-profile"}

ROWS = (
    RosterRow(CURRENT, "fact", TRANSFORMED),
    RosterRow(POWER, "fact", TRANSFORMED),
    RosterRow(STATUS, "fact", UNKNOWN),
    RosterRow(EXCLUDED, "capability", TRANSFORMED),
    RosterRow(SERVICE, "service", UNSUPPORTED),
    RosterRow(CONTAINS, KIND_RELATION, WITHHELD_DISPOSITION),
    RosterRow(BLANK, "fact", ""),
    RosterRow(HOSTS, KIND_RELATION, TRANSFORMED),
)


def allowing(*_args) -> Decision:
    return Decision(True, ())


class FakeStore:
    def __init__(self, initial: dict[str, bytes] | None = None) -> None:
        self.data: dict[str, bytes] = dict(initial or {})
        self.writes: list[str] = []

    def read(self, key: str) -> bytes | None:
        return self.data.get(key)

    def write(self, key: str, data: bytes) -> None:
        self.writes.append(key)
        self.data[key] = bytes(data)


def offer(element: str, resource_id: str) -> Offer:
    return Offer(element, resource_id, "synthetic-anchor")


def report(*offers: Offer) -> TargetReport:
    return TargetReport(LENS, tuple(offers), ())


def outcome(result) -> list[tuple]:
    return [(s.element, s.resource_id, s.state, s.endpoint, s.reasons) for s in result.states]


def ledger(count: int) -> bytes:
    digest = production_readiness._composition_digest(COMPOSITION)
    records = [{"resource_id": f"synthetic-resource-{index:03d}", "composition_sha256": digest, "endpoint": index} for index in range(1, count + 1)]
    return json.dumps({"schema_version": 1, "node_id": NODE, "records": records}).encode("utf-8")


class ExclusionListTest(unittest.TestCase):
    def test_committed_list_is_empty_and_shared_with_the_go_package(self) -> None:
        self.assertEqual(EXCLUSIONS_PATH, ROOT / "publication" / "exclusions.json")
        self.assertEqual(load_exclusions(), frozenset())
        self.assertEqual(json.loads(EXCLUSIONS_PATH.read_text(encoding="utf-8")), {"elements": []})
        source = GO_RULE.read_text(encoding="utf-8")
        self.assertIn("//go:embed exclusions.json\n", source)
        self.assertEqual(source.count("//go:embed"), 1)

    def test_rule_reads_the_committed_list_by_default(self) -> None:
        rule = PublicationRule(ROWS)
        self.assertIsNone(rule.withhold_reason(EXCLUDED, RES_A))

    def test_a_listed_element_is_excluded(self) -> None:
        with tempfile.TemporaryDirectory(dir=ROOT / ".tmp" if (ROOT / ".tmp").is_dir() else None) as directory:
            path = Path(directory) / "exclusions.json"
            path.write_text(json.dumps({"elements": [EXCLUDED]}), encoding="utf-8")
            rule = PublicationRule(ROWS, load_exclusions(path))
        self.assertEqual(rule.withhold_reason(EXCLUDED, RES_A), REASON_EXCLUDED)
        self.assertIsNone(rule.withhold_reason(CURRENT, RES_A))

    def test_malformed_list_is_refused(self) -> None:
        variants = {
            "not an object": b'["x"]',
            "no elements": b"{}",
            "null elements": b'{"elements":null}',
            "extra field": b'{"elements":[],"resources":[]}',
            "duplicate key": b'{"elements":[],"elements":[]}',
            "duplicate element": b'{"elements":["x","x"]}',
            "empty element": b'{"elements":[""]}',
            "padded element": b'{"elements":[" x"]}',
            "not text": b'{"elements":[1]}',
            "trailing data": b'{"elements":[]} {}',
            "empty document": b"",
            "invalid utf-8": b"\xff",
        }
        for name, data in variants.items():
            with self.subTest(name), self.assertRaises(ExclusionListError):
                parse_exclusions(data)


class PublicationRuleTest(unittest.TestCase):
    def setUp(self) -> None:
        self.rule = PublicationRule(ROWS, frozenset({EXCLUDED}))

    def test_every_disposition_has_its_named_reason(self) -> None:
        cases = {
            CURRENT: None,
            POWER: None,
            STATUS: "element disposition is unknown",
            SERVICE: "element disposition is unsupported",
            CONTAINS: "element disposition is withheld",
            BLANK: REASON_NO_DISPOSITION,
            HOSTS: REASON_RELATION,
            EXCLUDED: REASON_EXCLUDED,
            UNMAPPED: REASON_UNMAPPED,
            "": REASON_UNMAPPED,
        }
        for element, reason in cases.items():
            with self.subTest(element=element):
                self.assertEqual(self.rule.withhold_reason(element, RES_A), reason)

    def test_empty_or_malformed_resource_id_is_withheld(self) -> None:
        for resource_id in ("", " " + RES_A, RES_A + "\n", None):
            with self.subTest(resource_id=resource_id):
                self.assertEqual(self.rule.withhold_reason(CURRENT, resource_id), REASON_RESOURCE_ID)

    def test_every_row_field_decides(self) -> None:
        """Element: a rule keyed by anything else would publish the renamed row.
        Kind: a rule that ignores the kind would publish the relation.
        Disposition: a rule that treats any row as mapped would publish it."""
        rows = {
            "element": (RosterRow(CURRENT + ".other", "fact", TRANSFORMED), REASON_UNMAPPED),
            "kind": (RosterRow(CURRENT, KIND_RELATION, TRANSFORMED), REASON_RELATION),
            "disposition": (RosterRow(CURRENT, "fact", UNKNOWN), "element disposition is unknown"),
            "unrecognised disposition": (RosterRow(CURRENT, "fact", "proposed"), REASON_UNRECOGNISED),
        }
        for name, (row, reason) in rows.items():
            with self.subTest(field=name):
                self.assertEqual(PublicationRule((row,), frozenset()).withhold_reason(CURRENT, RES_A), reason)

    def test_ambiguous_roster_is_refused(self) -> None:
        cases = {
            "duplicate element": (RosterRow(CURRENT, "fact", TRANSFORMED), RosterRow(CURRENT, "fact", UNKNOWN)),
            "empty element": (RosterRow("", "fact", TRANSFORMED),),
            "padded element": (RosterRow(" " + CURRENT, "fact", TRANSFORMED),),
            "not a row": ((CURRENT, "fact", TRANSFORMED),),
        }
        for name, rows in cases.items():
            with self.subTest(name), self.assertRaises(ValueError):
                PublicationRule(rows, frozenset())

    def test_reasons_match_the_go_package(self) -> None:
        source = GO_RULE.read_text(encoding="utf-8")
        constants = dict(re.findall(r'^\t(Reason\w+)\s*=\s*"([^"]*)"$', source, re.MULTILINE))
        python = {
            "ReasonResourceID": REASON_RESOURCE_ID,
            "ReasonUnmapped": REASON_UNMAPPED,
            "ReasonExcluded": REASON_EXCLUDED,
            "ReasonRelation": REASON_RELATION,
            "ReasonNoDisposition": REASON_NO_DISPOSITION,
            "ReasonCapacity": REASON_CAPACITY,
        }
        for name, reason in python.items():
            with self.subTest(name):
                self.assertEqual(constants.get(name), reason)
        self.assertIn(f'"{REASON_UNRECOGNISED}"', source)
        self.assertIn(f"const StoreCapacity = {STORE_CAPACITY}\n", source)

    def test_capacity_name_has_no_digit_and_refuses_beyond_it(self) -> None:
        self.assertIsNone(re.search(r"\d", "STORE_CAPACITY"))
        self.assertEqual(STORE_CAPACITY, 64)
        self.assertIsNone(allocation_refusal(STORE_CAPACITY - 1, 1, 0xFFFE))
        self.assertEqual(allocation_refusal(STORE_CAPACITY, 1, 0xFFFE), REASON_CAPACITY)
        self.assertEqual(allocation_refusal(0, 0xFFFF, 0xFFFE), REASON_EXHAUSTED)


class SelectorAppliesTheRuleTest(unittest.TestCase):
    def setUp(self) -> None:
        self.store = FakeStore()
        self.rule = PublicationRule(ROWS, frozenset({EXCLUDED}))

    def selector(self, readiness=allowing) -> EndpointSelector:
        return EndpointSelector(self.store, NODE, rule=self.rule, readiness=readiness)

    def test_two_resources_share_one_transformed_element_and_the_rest_is_withheld(self) -> None:
        live = report(offer(CURRENT, RES_A), offer(CURRENT, RES_B), offer(STATUS, RES_C), offer(EXCLUDED, RES_D), offer(CURRENT, ""))
        result = self.selector().reconcile(None, COMPOSITION, None, live)
        self.assertEqual(
            outcome(result),
            [
                (STATUS, RES_C, WITHHELD, None, ("element disposition is unknown",)),
                (CURRENT, "", WITHHELD, None, (REASON_RESOURCE_ID,)),
                (CURRENT, RES_A, ELIGIBLE, 1, ()),
                (CURRENT, RES_B, ELIGIBLE, 2, ()),
                (EXCLUDED, RES_D, WITHHELD, None, (REASON_EXCLUDED,)),
            ],
        )
        records = json.loads(self.store.data[LEDGER_KEY])["records"]
        self.assertEqual([(r["resource_id"], r["endpoint"]) for r in records], [(RES_A, 1), (RES_B, 2)])

    def test_several_transformed_facts_of_one_resource_are_one_endpoint(self) -> None:
        live = report(offer(CURRENT, RES_A), offer(POWER, RES_A), offer(HOSTS, RES_A), offer(UNMAPPED, RES_A))
        result = self.selector().reconcile(None, COMPOSITION, None, live)
        self.assertEqual(
            outcome(result),
            [
                (CURRENT, RES_A, ELIGIBLE, 1, ()),
                (POWER, RES_A, ELIGIBLE, 1, ()),
                (HOSTS, RES_A, WITHHELD, None, (REASON_RELATION,)),
                (UNMAPPED, RES_A, WITHHELD, None, (REASON_UNMAPPED,)),
            ],
        )
        self.assertEqual(len(json.loads(self.store.data[LEDGER_KEY])["records"]), 1)

    def test_unmapped_or_excluded_resource_gets_no_number(self) -> None:
        live = report(offer(UNMAPPED, RES_A), offer(EXCLUDED, RES_B), offer(SERVICE, RES_C), offer(BLANK, RES_D))
        result = self.selector().reconcile(None, COMPOSITION, None, live)
        self.assertTrue(all(state.state == WITHHELD and state.endpoint is None for state in result.states))
        self.assertEqual(self.store.writes, [])

    def test_production_evaluator_still_withholds_every_resource(self) -> None:
        lock = copy.deepcopy(production_readiness.EXPECTED_LOCK)
        result = EndpointSelector(self.store, NODE, rule=self.rule).reconcile(lock, COMPOSITION, None, report(offer(CURRENT, RES_A), offer(POWER, RES_B)))
        self.assertEqual([state.state for state in result.states], [WITHHELD, WITHHELD])
        for state in result.states:
            self.assertIn("no reviewed accepted runtime-composition profile is pinned", state.reasons)
        self.assertEqual(self.store.writes, [])

    def test_a_new_resource_beyond_the_capacity_is_withheld(self) -> None:
        self.store.data[LEDGER_KEY] = ledger(STORE_CAPACITY - 1)
        live = report(offer(CURRENT, "synthetic-resource-001"), offer(CURRENT, RES_A), offer(CURRENT, RES_B))
        result = self.selector().reconcile(None, COMPOSITION, None, live)
        self.assertEqual(
            outcome(result),
            [
                (CURRENT, "synthetic-resource-001", ELIGIBLE, 1, ()),
                (CURRENT, RES_A, ELIGIBLE, STORE_CAPACITY, ()),
                (CURRENT, RES_B, WITHHELD, None, (REASON_CAPACITY,)),
            ],
        )
        persisted = self.store.data[LEDGER_KEY]
        self.assertEqual(len(json.loads(persisted)["records"]), STORE_CAPACITY)
        again = self.selector().reconcile(None, COMPOSITION, None, report(offer(CURRENT, RES_B), offer(CURRENT, RES_A)))
        self.assertEqual(outcome(again), [(CURRENT, RES_A, ELIGIBLE, STORE_CAPACITY, ()), (CURRENT, RES_B, WITHHELD, None, (REASON_CAPACITY,))])
        self.assertEqual(self.store.data[LEDGER_KEY], persisted)


if __name__ == "__main__":
    unittest.main()
