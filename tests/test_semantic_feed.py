"""Offline tests for the GraphQL semantic snapshot feed adapter.

Only a fake client returning synthetic GraphQL-shaped results is used. Sockets
are patched to fail so that any HTTP or network activity breaks the test; no
Matter node, endpoint, fabric, or commissioning state is created.
"""

from __future__ import annotations

import copy
import json
import socket
import threading
import time
import unittest
from pathlib import Path
from typing import Any
from unittest import mock

from runtime.endpoint_selection import LEDGER_KEY, WITHHELD, EndpointSelector
from runtime.production_readiness import Decision
from runtime.semantic_feed import (
    MAX_ATTEMPTS,
    QUERY,
    REVISION_AXES,
    Cancelled,
    SemanticFeed,
    TransportError,
)


ROOT = Path(__file__).resolve().parents[1]
LOCK = ROOT / "sdk" / "matter-sdk.lock.json"
ASSET_A = "asset:synthetic-heat-generator-family"
ASSET_B = "asset:synthetic-heat-pump-family"
POWER = "capability.synthetic.electrical-power.read"
ENERGY = "capability.synthetic.electrical-energy.read"
FACT = "fact.synthetic.active-power"


def revisions(base: int = 2, **overrides: int) -> dict[str, str]:
    values = {axis: str(base + index) for index, axis in enumerate(REVISION_AXES)}
    values.update({axis: str(value) for axis, value in overrides.items()})
    return values


def quantity(coefficient: str) -> dict[str, Any]:
    # GraphQL returns every selected optional member, absent ones as null.
    return {
        "kind": "quantity",
        "quantity": {"number": {"coefficient": coefficient, "exponent10": 0}, "unit": "unit.synthetic.watt"},
        "boolean": None,
        "text": None,
        "symbol": None,
        "symbols": None,
        "time": None,
    }


def capability(item_id: str, outcome: str = "exact", reason: str | None = None, loss: list | None = None) -> dict[str, Any]:
    return {"kind": "capability", "itemId": item_id, "outcome": outcome, "reason": reason, "loss": loss or []}


def asset(asset_id: str, revs: dict[str, str] | None = None, *, snapshot_id: str | None = None, value: str = "1500", dispositions: list | None = None) -> dict[str, Any]:
    revs = revs or revisions()
    snapshot_id = snapshot_id or f"snapshot:{asset_id}:{revs['semantic']}"
    key = {"packId": "pack.synthetic", "packVersion": "1.0.0", "factId": FACT, "dimensions": [{"id": "dimension.synthetic.phase", "value": {**quantity("1"), "kind": "text", "quantity": None, "text": "total"}}]}
    if dispositions is None:
        dispositions = [
            capability(POWER),
            {"kind": "fact", "itemId": FACT, "outcome": "exact", "reason": None, "loss": []},
        ]
    return {
        "assetId": asset_id,
        "unavailableReason": None,
        "snapshot": {
            "snapshotId": snapshot_id,
            "revisions": dict(revs),
            "facts": [{"revision": revs["facts"], "key": key, "candidates": [{"candidateId": f"candidate:{asset_id}", "revision": revs["facts"], "value": quantity(value)}]}],
        },
        "projection": {"snapshotId": snapshot_id, "revisions": dict(revs), "dispositions": dispositions},
    }


def unavailable(asset_id: str, reason: str = "semantic.read.reason.source_unavailable") -> dict[str, Any]:
    return {"assetId": asset_id, "unavailableReason": reason, "snapshot": None, "projection": None}


def result(*assets: dict[str, Any]) -> dict[str, Any]:
    return {"data": {"semanticSnapshot": {"assets": list(assets)}}}


class FakeClient:
    """Scripted client: each step is a result to return or an exception to raise."""

    def __init__(self, *steps: Any) -> None:
        self.steps = list(steps)
        self.executes: list[tuple[str, dict, float, Any]] = []
        self.reconnects: list[float] = []
        self.on_execute = None

    def execute(self, document, variables, *, timeout, cancel):
        self.executes.append((document, copy.deepcopy(dict(variables)), timeout, cancel))
        if self.on_execute is not None:
            self.on_execute()
        step = self.steps.pop(0)
        if isinstance(step, BaseException):
            raise step
        return copy.deepcopy(step)

    def reconnect(self, *, timeout, cancel):
        self.reconnects.append(timeout)


class Sleeps:
    def __init__(self, cancel_on: int | None = None, token: threading.Event | None = None) -> None:
        self.delays: list[float] = []
        self.cancel_on = cancel_on
        self.token = token

    def __call__(self, seconds: float, cancel) -> bool:
        self.delays.append(seconds)
        if self.cancel_on is not None and len(self.delays) == self.cancel_on:
            assert self.token is not None
            self.token.set()
            return False
        return True


class FakeStore:
    def __init__(self) -> None:
        self.data: dict[str, bytes] = {}
        self.writes: list[str] = []

    def read(self, key):
        return self.data.get(key)

    def write(self, key, data):
        self.writes.append(key)
        self.data[key] = bytes(data)


def no_network(*_args, **_kwargs):
    raise AssertionError("semantic feed tests must not open a socket")


class SemanticFeedTest(unittest.TestCase):
    def setUp(self) -> None:
        for target in ("socket.socket", "socket.create_connection", "socket.getaddrinfo"):
            patcher = mock.patch(target, side_effect=no_network)
            patcher.start()
            self.addCleanup(patcher.stop)
        self.cancel = threading.Event()

    def feed(self, client: FakeClient, ids=(ASSET_B, ASSET_A), **kwargs) -> SemanticFeed:
        kwargs.setdefault("sleep", Sleeps())
        return SemanticFeed(client, list(ids), **kwargs)

    def primed(self, *later: Any, revs_a=None, revs_b=None) -> tuple[SemanticFeed, FakeClient]:
        client = FakeClient(result(asset(ASSET_A, revs_a), asset(ASSET_B, revs_b)), *later)
        feed = self.feed(client)
        state = feed.refresh(self.cancel)
        self.assertTrue(state.complete and state.current, state.faults)
        return feed, client

    def assert_retained(self, feed: SemanticFeed, before) -> None:
        state = feed.state()
        self.assertEqual(state.assets, before.assets)
        self.assertFalse(state.current)
        self.assertTrue(state.faults)
        self.assertEqual(feed.selection_input().offers, ())

    def test_query_reads_only_the_snapshot_root_with_sorted_bounded_variables(self) -> None:
        feed, client = self.primed()
        document, variables, timeout, cancel = client.executes[0]
        self.assertEqual(document, QUERY)
        self.assertEqual(variables, {"assetIds": [ASSET_A, ASSET_B]})
        self.assertEqual(timeout, 10.0)
        self.assertIs(cancel, self.cancel)
        self.assertIn("semanticSnapshot(assetIds: $assetIds)", QUERY)
        self.assertNotIn("mutation", QUERY)
        for member in (*REVISION_AXES, "snapshotId", "unavailableReason", "candidates", "value", "loss", "sourceItems", "reversible", "outcome", "reason"):
            self.assertIn(member, QUERY)
        self.assertEqual(feed.asset_ids, (ASSET_A, ASSET_B))

    def test_complete_state_retains_exact_revision_value_and_loss(self) -> None:
        loss = [{"kind": "precision", "sourceItems": [FACT], "description": "synthetic rounding", "reversible": False}]
        dispositions = [capability(POWER), capability(ENERGY, "transformed", None, loss)]
        client = FakeClient(result(asset(ASSET_A, revisions(7), value="2750", dispositions=dispositions), asset(ASSET_B)))
        state = self.feed(client).refresh(self.cancel)
        a = state.assets[0]
        self.assertEqual(a.asset_id, ASSET_A)
        self.assertEqual(a.snapshot_id, f"snapshot:{ASSET_A}:7")
        self.assertEqual(a.revisions.axes(), (7, 8, 9, 10, 11))
        expected_value = json.dumps({"kind": "quantity", "quantity": {"number": {"coefficient": "2750", "exponent10": 0}, "unit": "unit.synthetic.watt"}}, sort_keys=True, separators=(",", ":"))
        self.assertEqual(a.facts[0].candidates[0].value, expected_value)
        self.assertEqual(a.facts[0].candidates[0].revision, 9)
        self.assertEqual(json.loads(a.facts[0].key)["dimensions"][0]["value"], {"kind": "text", "text": "total"})
        energy = a.dispositions[1]
        self.assertEqual((energy.outcome, energy.loss[0].kind, energy.loss[0].source_items, energy.loss[0].description, energy.loss[0].reversible), ("transformed", "precision", (FACT,), "synthetic rounding", False))
        self.assertTrue(state.complete and state.current and not state.faults)

    def test_newer_complete_state_replaces_and_older_or_mixed_cannot_overwrite_it(self) -> None:
        # Each axis is varied alone (older) and against another axis (mixed);
        # the mutation caught is a comparison that ignores that axis.
        for index, axis in enumerate(REVISION_AXES):
            other = REVISION_AXES[(index + 1) % len(REVISION_AXES)]
            current = revisions(10)
            older = revisions(10, **{axis: int(current[axis]) - 1})
            mixed = revisions(10, **{axis: int(current[axis]) - 1, other: int(current[other]) + 5})
            newer = revisions(10, **{axis: int(current[axis]) + 1})
            for label, revs in (("older", older), ("mixed", mixed)):
                with self.subTest(axis=axis, case=label):
                    feed, _ = self.primed(result(asset(ASSET_A, revs, value="9"), asset(ASSET_B)), revs_a=current)
                    before = feed.state()
                    feed.refresh(self.cancel)
                    self.assert_retained(feed, before)
                    self.assertIn(label, feed.state().faults[0])
                    self.assertEqual(feed.state().assets[0].facts[0].candidates[0].value, before.assets[0].facts[0].candidates[0].value)
            with self.subTest(axis=axis, case="newer"):
                feed, _ = self.primed(result(asset(ASSET_A, newer, value="9"), asset(ASSET_B)), revs_a=current)
                state = feed.refresh(self.cancel)
                self.assertTrue(state.current)
                self.assertEqual(state.assets[0].revisions.axes()[index], int(newer[axis]))
                self.assertIn('"9"', state.assets[0].facts[0].candidates[0].value)

    def test_equal_revisions_with_changed_content_or_identity_are_rejected(self) -> None:
        for label, replacement in (
            ("snapshot identity", asset(ASSET_A, snapshot_id="snapshot:other")),
            ("value", asset(ASSET_A, value="1")),
            ("outcome", asset(ASSET_A, dispositions=[capability(POWER, "withheld", "semantic.read.reason.capability_unavailable")])),
        ):
            with self.subTest(changed=label):
                feed, _ = self.primed(result(replacement, asset(ASSET_B)))
                before = feed.state()
                feed.refresh(self.cancel)
                self.assert_retained(feed, before)
        feed, _ = self.primed(result(asset(ASSET_A), asset(ASSET_B)))
        self.assertTrue(feed.refresh(self.cancel).current)

    def test_response_is_atomic_when_one_asset_regresses(self) -> None:
        feed, _ = self.primed(result(asset(ASSET_A, revisions(20), value="9"), asset(ASSET_B, revisions(1))), revs_b=revisions(5))
        before = feed.state()
        feed.refresh(self.cancel)
        self.assert_retained(feed, before)
        self.assertEqual(feed.state().assets[0].revisions.axes()[0], 2)

    def test_mixed_snapshot_and_projection_identity_within_an_asset_is_rejected(self) -> None:
        bad_revision = asset(ASSET_A, revisions(9))
        bad_revision["projection"]["revisions"]["capabilities"] = "99"
        bad_identity = asset(ASSET_A, revisions(9))
        bad_identity["projection"]["snapshotId"] = "snapshot:other"
        for label, record in (("revision", bad_revision), ("identity", bad_identity)):
            with self.subTest(mixed=label):
                feed, _ = self.primed(result(record, asset(ASSET_B)))
                before = feed.state()
                feed.refresh(self.cancel)
                self.assert_retained(feed, before)

    def test_withheld_and_unsupported_items_stay_withheld(self) -> None:
        loss = [{"kind": "unit", "sourceItems": [FACT], "description": "synthetic", "reversible": True}]
        dispositions = [
            capability(POWER),
            capability(ENERGY, "withheld", "semantic.read.reason.capability_unqualified"),
            capability("capability.synthetic.unsupported", "unsupported"),
            capability("capability.synthetic.unknown", "unknown"),
            capability("capability.synthetic.transformed", "transformed", None, loss),
            capability("capability.synthetic.unrepresentable", "unrepresentable"),
            capability("capability.synthetic.lossy-exact", "exact", None, loss),
            {"kind": "fact", "itemId": FACT, "outcome": "exact", "reason": None, "loss": []},
        ]
        feed, client = self.primed(result(asset(ASSET_A, revisions(3), dispositions=dispositions), asset(ASSET_B, revisions(3))))
        feed.refresh(self.cancel)
        selection = feed.selection_input()
        self.assertEqual([o.capability for o in selection.offers], [f"{ASSET_A}/{POWER}", f"{ASSET_B}/{POWER}"])
        reasons = {w.capability.split("/", 1)[1]: w.reasons for w in selection.withheld}
        self.assertEqual(reasons[ENERGY], ("semantic.read.reason.capability_unqualified",))
        self.assertEqual(reasons["capability.synthetic.unsupported"], ("projection outcome is unsupported",))
        self.assertEqual(reasons["capability.synthetic.unknown"], ("projection outcome is unknown",))
        self.assertEqual(reasons["capability.synthetic.transformed"], ("projection outcome is transformed",))
        self.assertEqual(reasons["capability.synthetic.unrepresentable"], ("projection outcome is unrepresentable",))
        self.assertEqual(reasons["capability.synthetic.lossy-exact"], ("exact projection carries loss",))
        self.assertNotIn(FACT, reasons)
        offer = selection.offers[0]
        self.assertEqual((offer.asset_id, offer.item_id, offer.snapshot_id, offer.revisions.axes()), (ASSET_A, POWER, f"snapshot:{ASSET_A}:3", (3, 4, 5, 6, 7)))

        # A newer withheld revision removes the offer; an older exact one cannot restore it.
        withdrawn = [capability(POWER, "withheld", "semantic.read.reason.capability_unavailable")]
        client.steps += [result(asset(ASSET_A, revisions(4), dispositions=withdrawn), asset(ASSET_B, revisions(4))), result(asset(ASSET_A, revisions(3)), asset(ASSET_B, revisions(4)))]
        feed.refresh(self.cancel)
        self.assertNotIn(f"{ASSET_A}/{POWER}", [o.capability for o in feed.selection_input().offers])
        feed.refresh(self.cancel)
        self.assertFalse(feed.state().current)
        self.assertEqual(feed.selection_input().offers, ())
        self.assertIn(f"{ASSET_A}/{POWER}", [w.capability for w in feed.selection_input().withheld])

    def test_unavailable_asset_withholds_last_known_good_and_keeps_revision_floor(self) -> None:
        feed, client = self.primed(result(unavailable(ASSET_A), asset(ASSET_B)), revs_a=revisions(8))
        state = feed.refresh(self.cancel)
        a = state.assets[0]
        self.assertEqual((a.unavailable_reason, a.revisions.axes()[0]), ("semantic.read.reason.source_unavailable", 8))
        self.assertTrue(a.facts and a.dispositions)
        withheld = {w.capability: w.reasons for w in feed.selection_input().withheld}
        self.assertEqual(withheld[f"{ASSET_A}/{POWER}"], ("semantic.read.reason.source_unavailable",))
        client.steps.append(result(asset(ASSET_A, revisions(7)), asset(ASSET_B)))
        feed.refresh(self.cancel)
        self.assertFalse(feed.state().current)
        self.assertEqual(feed.state().assets[0].revisions.axes()[0], 8)
        client.steps.append(result(asset(ASSET_A, revisions(8)), asset(ASSET_B)))
        self.assertIsNone(feed.refresh(self.cancel).assets[0].unavailable_reason)

    def test_first_read_with_unavailable_asset_is_complete_but_offers_nothing_for_it(self) -> None:
        client = FakeClient(result(unavailable(ASSET_A, "semantic.read.reason.publication_withheld"), asset(ASSET_B)))
        feed = self.feed(client)
        state = feed.refresh(self.cancel)
        self.assertTrue(state.complete)
        self.assertEqual([o.asset_id for o in feed.selection_input().offers], [ASSET_B])

    def test_feed_offers_cannot_authorize_endpoint_allocation(self) -> None:
        feed, _ = self.primed()
        offers = feed.selection_input().offers
        self.assertEqual(len(offers), 2)
        store = FakeStore()
        selector = EndpointSelector(store, "synthetic-node-a")
        for offer in offers:
            selector.select(offer.capability)
        lock = json.loads(LOCK.read_text(encoding="utf-8"))
        # The reviewed lock pins no accepted profile, so no feed state can admit.
        for composition in (None, {"schema_version": 1}, feed.state(), feed.selection_input()):
            reconciliation = selector.reconcile(lock, composition, None)
            self.assertTrue(reconciliation.states)
            self.assertTrue(all(s.state == WITHHELD and s.endpoint is None and s.identity is None for s in reconciliation.states))
        denied = EndpointSelector(store, "synthetic-node-a", readiness=lambda *_: Decision(False, ("synthetic readiness denial",)))
        self.assertTrue(all(s.state == WITHHELD for s in denied.reconcile(lock, feed.selection_input(), feed.state()).states))
        self.assertNotIn(LEDGER_KEY, store.writes)

    def test_partial_errors_keep_last_known_good_and_withhold_offers(self) -> None:
        partial = result(asset(ASSET_A, revisions(30), value="1"), asset(ASSET_B, revisions(30)))
        partial["errors"] = [{"message": "semantic snapshot read: CONFLICT", "extensions": {"code": "CONFLICT"}}, {"message": "x"}]
        nulled = {"data": {"semanticSnapshot": None}, "errors": [{"message": "m", "extensions": {"code": "UNAVAILABLE"}}]}
        for label, response, faults in (
            ("data with errors", partial, ("graphql error CONFLICT", "graphql error UNSPECIFIED")),
            ("null field", nulled, ("graphql error UNAVAILABLE",)),
        ):
            with self.subTest(case=label):
                feed, _ = self.primed(response)
                before = feed.state()
                feed.refresh(self.cancel)
                self.assert_retained(feed, before)
                self.assertEqual(feed.state().faults, faults)
                self.assertTrue(all("feed state is not current" in w.reasons for w in feed.selection_input().withheld))
                self.assertEqual(len(feed.selection_input().withheld), 2)

    def test_asset_set_must_match_the_request_exactly(self) -> None:
        for label, assets in (
            ("omitted", [asset(ASSET_A, revisions(9))]),
            ("duplicated", [asset(ASSET_A, revisions(9)), asset(ASSET_A, revisions(9))]),
            ("substituted", [asset(ASSET_A, revisions(9)), asset("asset:other", revisions(9))]),
            ("added", [asset(ASSET_A, revisions(9)), asset(ASSET_B, revisions(9)), asset("asset:other", revisions(9))]),
        ):
            with self.subTest(case=label):
                feed, _ = self.primed(result(*assets))
                before = feed.state()
                feed.refresh(self.cancel)
                self.assert_retained(feed, before)

    def test_malformed_members_reject_the_response(self) -> None:
        def mutate(path: tuple, value: Any) -> dict:
            record = asset(ASSET_A, revisions(9))
            target = record
            for key in path[:-1]:
                target = target[key]
            target[path[-1]] = value
            return record

        cases = {
            "leading zero revision": mutate(("snapshot", "revisions", "facts"), "09"),
            "negative revision": mutate(("snapshot", "revisions", "semantic"), "-1"),
            "overflow revision": mutate(("snapshot", "revisions", "identity"), str(1 << 64)),
            "integer revision": mutate(("snapshot", "revisions", "services"), 9),
            "missing axis": mutate(("snapshot", "revisions"), {"semantic": "9"}),
            "unknown outcome": mutate(("projection", "dispositions"), [capability(POWER, "approximate")]),
            "unknown kind": mutate(("projection", "dispositions"), [{**capability(POWER), "kind": "endpoint"}]),
            "duplicate disposition": mutate(("projection", "dispositions"), [capability(POWER), capability(POWER)]),
            "unknown loss": mutate(("projection", "dispositions"), [capability(POWER, "transformed", None, [{"kind": "guess", "sourceItems": [], "description": "", "reversible": True}])]),
            "oversized loss text": mutate(("projection", "dispositions"), [capability(POWER, "transformed", None, [{"kind": "unit", "sourceItems": [], "description": "x" * 1025, "reversible": True}])]),
            "missing snapshot": mutate(("snapshot",), None),
            "unknown unavailable reason": unavailable(ASSET_A, "semantic.read.reason.anything"),
            "unavailable with state": {**asset(ASSET_A, revisions(9)), "unavailableReason": "semantic.read.reason.asset_unknown"},
            "empty fact key": mutate(("snapshot", "facts"), [{"revision": "9", "key": {}, "candidates": []}]),
            "invalid quantity coefficient": mutate(("snapshot", "facts", 0, "candidates", 0, "value"), quantity("not-a-number")),
            "quantity without number": mutate(("snapshot", "facts", 0, "candidates", 0, "value"), {**quantity("1"), "quantity": {"unit": "unit.synthetic.watt"}}),
            "unknown value kind": mutate(("snapshot", "facts", 0, "candidates", 0, "value"), {**quantity("1"), "kind": "blob"}),
            "oversized facts": mutate(("snapshot", "facts"), [{"revision": "1", "key": {}, "candidates": []}] * 4097),
        }
        for label, record in cases.items():
            with self.subTest(case=label):
                feed, _ = self.primed(result(record, asset(ASSET_B, revisions(9))))
                before = feed.state()
                feed.refresh(self.cancel)
                self.assert_retained(feed, before)
                self.assertTrue(feed.state().faults[0].startswith("response rejected"), feed.state().faults)
        for response in (None, [], {"data": None}, {"data": {"semanticSnapshot": {"assets": None}}}):
            with self.subTest(response=response):
                feed, _ = self.primed(response)
                before = feed.state()
                feed.refresh(self.cancel)
                self.assert_retained(feed, before)

    def test_cancellation_before_during_and_after_a_read_discards_the_result(self) -> None:
        self.cancel.set()
        client = FakeClient(result(asset(ASSET_A), asset(ASSET_B)))
        feed = self.feed(client)
        with self.assertRaises(Cancelled):
            feed.refresh(self.cancel)
        self.assertEqual(client.executes, [])
        self.assertEqual((feed.state().assets, feed.state().current, feed.state().faults), ((), False, ("cancelled",)))

        self.cancel.clear()
        feed, client = self.primed(result(asset(ASSET_A, revisions(40)), asset(ASSET_B, revisions(40))))
        before = feed.state()
        client.on_execute = self.cancel.set
        with self.assertRaises(Cancelled):
            feed.refresh(self.cancel)
        self.assert_retained(feed, before)

        self.cancel.clear()
        feed, client = self.primed(Cancelled())
        before = feed.state()
        with self.assertRaises(Cancelled):
            feed.refresh(self.cancel)
        self.assert_retained(feed, before)
        self.assertEqual(client.reconnects, [])

    def test_cancellation_during_reconnect_backoff_stops_further_attempts(self) -> None:
        client = FakeClient(TransportError("synthetic"), result(asset(ASSET_A), asset(ASSET_B)))
        sleeps = Sleeps(cancel_on=1, token=self.cancel)
        feed = self.feed(client, sleep=sleeps)
        with self.assertRaises(Cancelled):
            feed.refresh(self.cancel)
        self.assertEqual((len(client.executes), client.reconnects, sleeps.delays), (1, [], [0.5]))

    def test_reconnect_is_bounded_and_retries_on_transport_failure_only(self) -> None:
        client = FakeClient(TransportError("a"), TransportError("b"), result(asset(ASSET_A), asset(ASSET_B)))
        sleeps = Sleeps()
        feed = self.feed(client, sleep=sleeps, timeout_s=2.5)
        state = feed.refresh(self.cancel)
        self.assertTrue(state.current and state.complete)
        self.assertEqual((len(client.executes), client.reconnects, sleeps.delays), (3, [2.5, 2.5], [0.5, 2.0]))

        feed, client = self.primed(TransportError("a"), TransportError("b"), TransportError("c"), result(asset(ASSET_A, revisions(50)), asset(ASSET_B, revisions(50))))
        before = feed.state()
        feed.refresh(self.cancel)
        self.assert_retained(feed, before)
        self.assertEqual(feed.state().faults, ("transport failed after 3 attempts",))
        self.assertEqual((len(client.executes), len(client.reconnects)), (4, 2))
        self.assertTrue(feed.refresh(self.cancel).current)

        partial = {"errors": [{"message": "m", "extensions": {"code": "UNAVAILABLE"}}]}
        feed, client = self.primed(partial, result(asset(ASSET_A), asset(ASSET_B)))
        feed.refresh(self.cancel)
        self.assertEqual((len(client.executes), client.reconnects), (2, []))

        feed, client = self.primed(RuntimeError("synthetic client fault"))
        before = feed.state()
        with self.assertRaises(RuntimeError):
            feed.refresh(self.cancel)
        self.assert_retained(feed, before)
        self.assertEqual(client.reconnects, [])

    def test_default_sleep_waits_on_the_cancel_token(self) -> None:
        client = FakeClient(TransportError("a"), result(asset(ASSET_A), asset(ASSET_B)))
        feed = SemanticFeed(client, [ASSET_A, ASSET_B], backoff_s=(0.0,))
        self.assertTrue(feed.refresh(self.cancel).current)

    def test_default_wait_is_cancellable_without_a_wait_method(self) -> None:
        class PollToken:
            def __init__(self, set_after: int | None = None) -> None:
                self.polls, self.set_after = 0, set_after

            def is_set(self) -> bool:
                self.polls += 1
                return self.set_after is not None and self.polls > self.set_after

        started = time.monotonic()
        client = FakeClient(TransportError("a"), result(asset(ASSET_A), asset(ASSET_B)))
        feed = SemanticFeed(client, [ASSET_A, ASSET_B], backoff_s=(0.2,))
        self.assertTrue(feed.refresh(PollToken()).current)
        self.assertGreaterEqual(time.monotonic() - started, 0.15)

        client = FakeClient(TransportError("a"), result(asset(ASSET_A), asset(ASSET_B)))
        feed = SemanticFeed(client, [ASSET_A, ASSET_B], backoff_s=(30.0,))
        started = time.monotonic()
        with self.assertRaises(Cancelled):
            feed.refresh(PollToken(set_after=2))
        self.assertLess(time.monotonic() - started, 5.0)
        self.assertEqual(client.reconnects, [])

    def test_construction_bounds(self) -> None:
        client = FakeClient()
        invalid = (
            {"asset_ids": []},
            {"asset_ids": [ASSET_A, ASSET_A]},
            {"asset_ids": [f"asset:{i}" for i in range(33)]},
            {"asset_ids": [""]},
            {"asset_ids": [" asset:a"]},
            {"asset_ids": ["x" * 257]},
            {"asset_ids": ["native://resource"]},
            {"asset_ids": [ASSET_A], "max_attempts": 0},
            {"asset_ids": [ASSET_A], "max_attempts": MAX_ATTEMPTS + 1},
            {"asset_ids": [ASSET_A], "timeout_s": 0},
            {"asset_ids": [ASSET_A], "timeout_s": 61},
            {"asset_ids": [ASSET_A], "backoff_s": ()},
            {"asset_ids": [ASSET_A], "backoff_s": (-1.0,)},
            {"asset_ids": [ASSET_A], "backoff_s": (61.0,)},
        )
        for kwargs in invalid:
            with self.subTest(kwargs=str(kwargs)[:80]):
                ids = kwargs.pop("asset_ids")
                with self.assertRaises(ValueError):
                    SemanticFeed(client, ids, **kwargs)
        self.assertEqual(client.executes, [])


if __name__ == "__main__":
    unittest.main()
