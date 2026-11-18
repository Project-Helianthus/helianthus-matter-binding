"""Endpoint publication behind the publication rule and the fail-closed readiness gate.

The endpoint ledger lives in an injected store. This module creates no Matter
node, endpoint, fabric, or commissioning state. A Matter target report offers
catalog elements on canonical resources. The publication rule decides first:
only an element with a transformed roster row that the exclusion list does not
name can make its resource eligible. The ledger keys one endpoint number to
each resource's ResourceID and records the composition digest it was admitted
under. Parent and child endpoints are derived on every reconciliation from the
relation instances the report cites and are never persisted. Nothing is read
through the Portal catalog.

An eligible result only means that the unchanged readiness evaluator permitted
allocation and that the endpoint number was durably recorded first. The
reviewed lock pins no accepted profile, so today every resource is withheld and
nothing is allocated.
"""

from __future__ import annotations

import json
import re
from collections import Counter
from dataclasses import dataclass, field
from typing import Any, Callable, Protocol

from runtime.production_readiness import Decision, _composition_digest, evaluate
from runtime.publication_rule import PublicationRule, allocation_refusal


LEDGER_KEY = "endpoint-ledger"
SCHEMA_VERSION = 1
FIRST_ENDPOINT = 1
LAST_ENDPOINT = 0xFFFE
LEDGER_FIELDS = frozenset({"schema_version", "node_id", "records"})
RECORD_FIELDS = frozenset({"resource_id", "composition_sha256", "endpoint"})
# Forest core types: each resource is the part of at most one instance.
PARENT_CORE_TYPES = frozenset({"contains", "hosts"})
COHERENT = "coherent"

WITHHELD = "withheld"
ELIGIBLE = "eligible"

_RESOURCE_ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9._:/@-]{0,255}")
_DIGEST = re.compile(r"[0-9a-f]{64}")
_MAX_TEXT = 256
_NO_PARENT = "presented under no parent"


class StoreError(Exception):
    """Raised by a store that cannot durably read or write a document."""


class EndpointStore(Protocol):
    """Durable key-value storage; each write must replace one key atomically."""

    def read(self, key: str) -> bytes | None: ...

    def write(self, key: str, data: bytes) -> None: ...


@dataclass(frozen=True)
class RelationKey:
    """One published relation instance; every member is part of the key."""

    pack_id: str
    pack_version: str
    definition: str
    version: str
    from_resource: str
    to_resource: str


@dataclass(frozen=True)
class RelationInstance:
    """A relation instance in the report's cut, with its core type and coherence."""

    key: RelationKey
    core_type: str
    coherence: str


@dataclass(frozen=True)
class Offer:
    """A target-report disposition offering one catalog element on one resource.

    The anchor is the resource's identity evidence as published. It is never a
    ledger key: an anchor supersession keeps the ResourceID and so the endpoint.
    """

    element: str
    resource_id: str
    anchor: str
    relations: tuple[RelationKey, ...] = ()


@dataclass(frozen=True)
class TargetReport:
    """The Matter target report over a named lens."""

    lens: str
    offers: tuple[Offer, ...]
    relations: tuple[RelationInstance, ...] = ()


@dataclass(frozen=True)
class ElementState:
    """The outcome for one offered element on one resource."""

    element: str
    resource_id: str
    state: str
    reasons: tuple[str, ...]
    endpoint: int | None = None
    parent_endpoint: int | None = None


@dataclass(frozen=True)
class Reconciliation:
    states: tuple[ElementState, ...]
    faults: tuple[str, ...]


@dataclass
class _Ledger:
    records: list[dict[str, Any]] = field(default_factory=list)
    valid: dict[str, dict[str, Any]] = field(default_factory=dict)
    withheld: dict[str, set[str]] = field(default_factory=dict)
    highest_endpoint: int = FIRST_ENDPOINT - 1
    allocation_fault: str | None = None


def _is_text(value: Any) -> bool:
    return isinstance(value, str) and bool(value.strip()) and value == value.strip() and len(value) <= _MAX_TEXT


def _is_resource(value: Any) -> bool:
    return isinstance(value, str) and _RESOURCE_ID.fullmatch(value) is not None


def _encode(document: dict[str, Any]) -> bytes:
    return json.dumps(document, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode("utf-8")


def _unique_pairs(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    document: dict[str, Any] = {}
    for key, value in pairs:
        if key in document:
            raise ValueError("duplicate key")
        document[key] = value
    return document


def _decode(data: Any) -> Any:
    if not isinstance(data, (bytes, bytearray)):
        raise ValueError("stored document is not bytes")
    return json.loads(bytes(data).decode("utf-8"), object_pairs_hook=_unique_pairs)


def _read(store: EndpointStore, key: str) -> tuple[Any, str | None]:
    try:
        data = store.read(key)
    except (StoreError, OSError):
        return None, f"{key} could not be read"
    if data is None:
        return None, None
    try:
        return _decode(data), None
    except (ValueError, UnicodeDecodeError, RecursionError):
        return None, f"{key} is corrupt"


def _valid_record(record: Any) -> bool:
    if not isinstance(record, dict) or set(record) != RECORD_FIELDS:
        return False
    endpoint = record["endpoint"]
    return (
        _is_resource(record["resource_id"])
        and isinstance(record["composition_sha256"], str)
        and _DIGEST.fullmatch(record["composition_sha256"]) is not None
        and isinstance(endpoint, int)
        and not isinstance(endpoint, bool)
        and FIRST_ENDPOINT <= endpoint <= LAST_ENDPOINT
    )


def _parse_ledger(store: EndpointStore, node_id: str) -> tuple[_Ledger | None, str | None]:
    document, fault = _read(store, LEDGER_KEY)
    if fault:
        return None, fault
    if document is None:
        return _Ledger(), None
    if not isinstance(document, dict) or set(document) != LEDGER_FIELDS or document["schema_version"] != SCHEMA_VERSION:
        return None, f"{LEDGER_KEY} has an unsupported shape"
    if not _is_text(document["node_id"]) or not isinstance(document["records"], list):
        return None, f"{LEDGER_KEY} has a malformed node identity or record list"
    if document["node_id"] != node_id:
        return None, f"{LEDGER_KEY} belongs to a different node lifecycle; a reviewed migration is required"

    ledger = _Ledger(records=list(document["records"]))
    well_formed: list[dict[str, Any]] = []
    for record in ledger.records:
        if _valid_record(record):
            well_formed.append(record)
            continue
        ledger.allocation_fault = "endpoint ledger has a malformed record; new allocation is withheld"
        if isinstance(record, dict) and _is_resource(record.get("resource_id")):
            ledger.withheld.setdefault(record["resource_id"], set()).add("persisted endpoint record is malformed")

    resources = Counter(record["resource_id"] for record in well_formed)
    endpoints = Counter(record["endpoint"] for record in well_formed)
    for record in well_formed:
        resource_id = record["resource_id"]
        reasons = ledger.withheld.get(resource_id, set())
        if resources[resource_id] > 1:
            reasons.add("resource has duplicate persisted endpoint records")
        if endpoints[record["endpoint"]] > 1:
            reasons.add("persisted endpoint number is shared with another record")
        if reasons:
            ledger.withheld[resource_id] = reasons
        ledger.highest_endpoint = max(ledger.highest_endpoint, record["endpoint"])
    for record in well_formed:
        if record["resource_id"] not in ledger.withheld:
            ledger.valid[record["resource_id"]] = record
    return ledger, None


def _valid_key(key: Any) -> bool:
    return (
        isinstance(key, RelationKey)
        and all(_is_text(value) for value in (key.pack_id, key.pack_version, key.definition, key.version))
        and _is_resource(key.from_resource)
        and _is_resource(key.to_resource)
    )


def _report_is_malformed(report: Any) -> bool:
    return (
        not isinstance(report, TargetReport)
        or not _is_text(report.lens)
        or not isinstance(report.offers, tuple)
        or not isinstance(report.relations, tuple)
        or not all(isinstance(offer, Offer) and _is_text(offer.element) and isinstance(offer.resource_id, str) for offer in report.offers)
    )


def _offer_fault(offer: Offer) -> str | None:
    if not _is_resource(offer.resource_id) or not _is_text(offer.anchor):
        return "offer cites no well-formed resource"
    if not isinstance(offer.relations, tuple) or not all(_valid_key(key) for key in offer.relations):
        return "offer cites a malformed relation key"
    return None


_OfferKey = tuple[str, str]


def _resolve_offers(report: TargetReport, rule: PublicationRule) -> tuple[dict[_OfferKey, Offer], dict[_OfferKey, ElementState]]:
    """Ask the publication rule about every offer, then keep the well-formed, unambiguous ones."""
    counts = Counter((offer.element, offer.resource_id) for offer in report.offers)
    offers: dict[_OfferKey, Offer] = {}
    withheld: dict[_OfferKey, ElementState] = {}
    for offer in report.offers:
        key = (offer.element, offer.resource_id)
        fault = rule.withhold_reason(offer.element, offer.resource_id)
        if fault is None and counts[key] > 1:
            fault = "Matter target report has more than one offer for this element on this resource"
        fault = fault or _offer_fault(offer)
        if fault:
            withheld[key] = ElementState(offer.element, offer.resource_id, WITHHELD, (fault,))
        else:
            offers[key] = offer
    return offers, withheld


def _ordered(states: dict[_OfferKey, ElementState]) -> tuple[ElementState, ...]:
    return tuple(states[key] for key in sorted(states))


def _published(relations: tuple[Any, ...]) -> dict[RelationKey, RelationInstance]:
    """Index well-formed coherent instances; a key published twice is unusable."""
    counts = Counter(item.key for item in relations if isinstance(item, RelationInstance) and _valid_key(item.key))
    return {
        item.key: item
        for item in relations
        if isinstance(item, RelationInstance)
        and _valid_key(item.key)
        and counts[item.key] == 1
        and item.coherence == COHERENT
        and _is_text(item.core_type)
    }


def _cited_parent(resource_id: str, cited: set[RelationKey], published: dict[RelationKey, RelationInstance], endpoints: dict[str, int]) -> tuple[str | None, str | None]:
    parents: set[str] = set()
    for key in cited:
        if key.to_resource != resource_id:
            continue
        instance = published.get(key)
        if instance is None:
            return None, f"a cited relation is not a coherent published instance; {_NO_PARENT}"
        if instance.core_type in PARENT_CORE_TYPES:
            parents.add(key.from_resource)
    if len(parents) > 1:
        return None, f"cited relations name more than one parent; {_NO_PARENT}"
    if not parents:
        return None, None
    parent = parents.pop()
    if parent not in endpoints:
        return None, f"parent resource has no eligible endpoint; {_NO_PARENT}"
    return parent, None


def _compose(report: TargetReport, offers: dict[_OfferKey, Offer], endpoints: dict[str, int]) -> dict[str, tuple[int | None, tuple[str, ...]]]:
    """Derive each eligible resource's parent endpoint from the cited relations."""
    published = _published(report.relations)
    cited = {key for offer in offers.values() for key in offer.relations}
    parent_of: dict[str, str] = {}
    notes: dict[str, tuple[str, ...]] = {}
    for resource_id in endpoints:
        parent, reason = _cited_parent(resource_id, cited, published, endpoints)
        if parent is not None:
            parent_of[resource_id] = parent
        notes[resource_id] = (reason,) if reason else ()
    for resource_id in list(parent_of):
        seen, current = {resource_id}, parent_of[resource_id]
        while current in parent_of and current not in seen:
            seen.add(current)
            current = parent_of[current]
        if current == resource_id:
            notes[resource_id] = (f"cited parent relations form a cycle; {_NO_PARENT}",)
    composed = {r: (None, notes[r]) for r in endpoints}
    for resource_id, parent in parent_of.items():
        if not notes[resource_id]:
            composed[resource_id] = (endpoints[parent], ())
    return composed


class EndpointSelector:
    """Apply the publication rule and keep the resource endpoint ledger for one node lifecycle."""

    def __init__(
        self,
        store: EndpointStore,
        node_id: str,
        *,
        rule: PublicationRule,
        readiness: Callable[[Any, Any, Any], Decision] = evaluate,
    ) -> None:
        if not _is_text(node_id):
            raise ValueError("node identity must be non-empty text")
        if not isinstance(rule, PublicationRule):
            raise ValueError("a publication rule is required")
        self._store = store
        self._node_id = node_id
        self._rule = rule
        self._readiness = readiness

    def reconcile(self, lock: Any, composition: Any, build_evidence: Any, report: Any) -> Reconciliation:
        """Withhold, restore, or durably allocate an endpoint for every resource the rule publishes."""
        if _report_is_malformed(report):
            return Reconciliation((), ("Matter target report is absent or malformed",))
        offers, states = _resolve_offers(report, self._rule)
        if not offers:
            return Reconciliation(_ordered(states), ())

        decision = self._readiness(lock, composition, build_evidence)
        if not isinstance(decision, Decision) or decision.permitted is not True:
            reasons = decision.reasons if isinstance(decision, Decision) and decision.reasons else ("readiness evaluator did not permit allocation",)
            states.update((key, ElementState(*key, WITHHELD, tuple(reasons))) for key in offers)
            return Reconciliation(_ordered(states), ())

        ledger, fault = _parse_ledger(self._store, self._node_id)
        if ledger is None:
            assert fault is not None
            states.update((key, ElementState(*key, WITHHELD, (fault,))) for key in offers)
            return Reconciliation(_ordered(states), (fault,))
        faults = (ledger.allocation_fault,) if ledger.allocation_fault else ()
        digest = _composition_digest(composition)

        endpoints: dict[str, int] = {}
        withheld: dict[str, tuple[str, ...]] = {}
        pending: list[dict[str, Any]] = []
        next_endpoint = ledger.highest_endpoint + 1
        for resource_id in sorted({offer.resource_id for offer in offers.values()}):
            record = ledger.valid.get(resource_id)
            if resource_id in ledger.withheld:
                withheld[resource_id] = tuple(sorted(ledger.withheld[resource_id]))
            elif record is not None and record["composition_sha256"] != digest:
                withheld[resource_id] = ("persisted endpoint belongs to a different composition; a reviewed migration is required",)
            elif record is not None:
                endpoints[resource_id] = record["endpoint"]
            elif ledger.allocation_fault:
                withheld[resource_id] = (ledger.allocation_fault,)
            elif refusal := allocation_refusal(len(ledger.records) + len(pending), next_endpoint, LAST_ENDPOINT):
                withheld[resource_id] = (refusal,)
            else:
                pending.append({"resource_id": resource_id, "composition_sha256": digest, "endpoint": next_endpoint})
                next_endpoint += 1

        if pending:
            document = {"schema_version": SCHEMA_VERSION, "node_id": self._node_id, "records": ledger.records + pending}
            try:
                self._store.write(LEDGER_KEY, _encode(document))
            except (StoreError, OSError):
                withheld.update((record["resource_id"], ("endpoint ledger could not be persisted",)) for record in pending)
                faults += (f"{LEDGER_KEY} could not be written",)
            else:
                endpoints.update((record["resource_id"], record["endpoint"]) for record in pending)

        composed = _compose(report, offers, endpoints) if endpoints else {}
        for key, offer in offers.items():
            resource_id = offer.resource_id
            if resource_id in endpoints:
                parent_endpoint, notes = composed[resource_id]
                states[key] = ElementState(*key, ELIGIBLE, notes, endpoints[resource_id], parent_endpoint)
            else:
                states[key] = ElementState(*key, WITHHELD, withheld[resource_id])
        return Reconciliation(_ordered(states), faults)
