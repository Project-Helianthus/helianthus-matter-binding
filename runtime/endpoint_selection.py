"""Persistent Portal endpoint selection behind the fail-closed readiness gate.

Selection intent and the endpoint identity ledger live in an injected store.
This module creates no Matter node, endpoint, fabric, or commissioning state.
An eligible result only means that the unchanged readiness evaluator permitted
allocation and that a locally generated opaque identity and endpoint number
were durably recorded first. The reviewed lock pins no accepted profile, so
today every selected capability is withheld and nothing is allocated.
"""

from __future__ import annotations

import json
import re
import secrets
from collections import Counter
from dataclasses import dataclass, field
from typing import Any, Callable, Protocol

from runtime.production_readiness import Decision, _composition_digest, evaluate


SELECTION_KEY = "portal-selection"
LEDGER_KEY = "endpoint-ledger"
SCHEMA_VERSION = 1
FIRST_ENDPOINT = 1
LAST_ENDPOINT = 0xFFFE
SELECTION_FIELDS = frozenset({"schema_version", "selected"})
LEDGER_FIELDS = frozenset({"schema_version", "node_id", "records"})
RECORD_FIELDS = frozenset({"identity", "capability", "composition_sha256", "endpoint"})

WITHHELD = "withheld"
ELIGIBLE = "eligible"

_IDENTITY = re.compile(r"[A-Za-z0-9_-]{16,128}")
_DIGEST = re.compile(r"[0-9a-f]{64}")
_MAX_TEXT = 256


class StoreError(Exception):
    """Raised by a store that cannot durably read or write a document."""


class SelectionStateError(Exception):
    """Persisted selection intent is unreadable and must not be overwritten."""


class EndpointStore(Protocol):
    """Durable key-value storage; each write must replace one key atomically."""

    def read(self, key: str) -> bytes | None: ...

    def write(self, key: str, data: bytes) -> None: ...


@dataclass(frozen=True)
class CapabilityState:
    capability: str
    state: str
    reasons: tuple[str, ...]
    identity: str | None = None
    endpoint: int | None = None


@dataclass(frozen=True)
class Reconciliation:
    states: tuple[CapabilityState, ...]
    faults: tuple[str, ...]


@dataclass
class _Ledger:
    records: list[dict[str, Any]] = field(default_factory=list)
    valid: dict[str, dict[str, Any]] = field(default_factory=dict)
    withheld: dict[str, set[str]] = field(default_factory=dict)
    identities: set[str] = field(default_factory=set)
    highest_endpoint: int = FIRST_ENDPOINT - 1
    allocation_fault: str | None = None


def _is_text(value: Any) -> bool:
    return isinstance(value, str) and bool(value.strip()) and value == value.strip() and len(value) <= _MAX_TEXT


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


def _parse_selection(store: EndpointStore) -> tuple[frozenset[str], str | None]:
    document, fault = _read(store, SELECTION_KEY)
    if fault:
        return frozenset(), fault
    if document is None:
        return frozenset(), None
    if not isinstance(document, dict) or set(document) != SELECTION_FIELDS or document["schema_version"] != SCHEMA_VERSION:
        return frozenset(), f"{SELECTION_KEY} has an unsupported shape"
    selected = document["selected"]
    if not isinstance(selected, list) or not all(_is_text(item) for item in selected) or len(set(selected)) != len(selected):
        return frozenset(), f"{SELECTION_KEY} has malformed or duplicate capabilities"
    return frozenset(selected), None


def _valid_record(record: Any) -> bool:
    if not isinstance(record, dict) or set(record) != RECORD_FIELDS:
        return False
    identity, endpoint = record["identity"], record["endpoint"]
    return (
        isinstance(identity, str)
        and _IDENTITY.fullmatch(identity) is not None
        and _is_text(record["capability"])
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
        if isinstance(record, dict) and _is_text(record.get("capability")):
            ledger.withheld.setdefault(record["capability"], set()).add("persisted endpoint record is malformed")

    capabilities = Counter(record["capability"] for record in well_formed)
    identities = Counter(record["identity"] for record in well_formed)
    endpoints = Counter(record["endpoint"] for record in well_formed)
    for record in well_formed:
        capability = record["capability"]
        reasons = ledger.withheld.get(capability, set())
        if capabilities[capability] > 1:
            reasons.add("capability has duplicate persisted endpoint records")
        if identities[record["identity"]] > 1:
            reasons.add("persisted identity is shared with another record")
        if endpoints[record["endpoint"]] > 1:
            reasons.add("persisted endpoint number is shared with another record")
        if reasons:
            ledger.withheld[capability] = reasons
        ledger.identities.add(record["identity"])
        ledger.highest_endpoint = max(ledger.highest_endpoint, record["endpoint"])
    for record in well_formed:
        if record["capability"] not in ledger.withheld:
            ledger.valid[record["capability"]] = record
    return ledger, None


def _new_identity() -> str:
    return secrets.token_hex(16)


class EndpointSelector:
    """Persist Portal selection intent and admitted endpoint identity for one node lifecycle."""

    def __init__(
        self,
        store: EndpointStore,
        node_id: str,
        *,
        readiness: Callable[[Any, Any, Any], Decision] = evaluate,
        new_identity: Callable[[], str] = _new_identity,
    ) -> None:
        if not _is_text(node_id):
            raise ValueError("node identity must be non-empty text")
        self._store = store
        self._node_id = node_id
        self._readiness = readiness
        self._new_identity = new_identity

    def selected(self) -> frozenset[str]:
        selected, fault = _parse_selection(self._store)
        if fault:
            raise SelectionStateError(fault)
        return selected

    def select(self, capability: str) -> None:
        self._update_selection(capability, add=True)

    def deselect(self, capability: str) -> None:
        self._update_selection(capability, add=False)

    def _update_selection(self, capability: str, *, add: bool) -> None:
        if not _is_text(capability):
            raise ValueError("capability must be non-empty text")
        selected = self.selected()
        if (capability in selected) == add:
            return
        updated = selected | {capability} if add else selected - {capability}
        self._store.write(SELECTION_KEY, _encode({"schema_version": SCHEMA_VERSION, "selected": sorted(updated)}))

    def reconcile(self, lock: Any, composition: Any, build_evidence: Any) -> Reconciliation:
        """Withhold, restore, or durably allocate identity for every selected capability."""
        selected, fault = _parse_selection(self._store)
        if fault:
            return Reconciliation((), (fault,))
        if not selected:
            return Reconciliation((), ())
        capabilities = sorted(selected)

        decision = self._readiness(lock, composition, build_evidence)
        if not isinstance(decision, Decision) or decision.permitted is not True:
            reasons = decision.reasons if isinstance(decision, Decision) and decision.reasons else ("readiness evaluator did not permit allocation",)
            return Reconciliation(tuple(CapabilityState(c, WITHHELD, tuple(reasons)) for c in capabilities), ())

        ledger, fault = _parse_ledger(self._store, self._node_id)
        if ledger is None:
            assert fault is not None
            return Reconciliation(tuple(CapabilityState(c, WITHHELD, (fault,)) for c in capabilities), (fault,))
        faults = (ledger.allocation_fault,) if ledger.allocation_fault else ()
        digest = _composition_digest(composition)

        states: dict[str, CapabilityState] = {}
        pending: list[dict[str, Any]] = []
        next_endpoint = ledger.highest_endpoint + 1
        for capability in capabilities:
            if capability in ledger.withheld:
                states[capability] = CapabilityState(capability, WITHHELD, tuple(sorted(ledger.withheld[capability])))
                continue
            record = ledger.valid.get(capability)
            if record is not None:
                if record["composition_sha256"] != digest:
                    states[capability] = CapabilityState(capability, WITHHELD, ("persisted identity belongs to a different composition; a reviewed migration is required",))
                else:
                    states[capability] = CapabilityState(capability, ELIGIBLE, (), record["identity"], record["endpoint"])
                continue
            if ledger.allocation_fault:
                states[capability] = CapabilityState(capability, WITHHELD, (ledger.allocation_fault,))
                continue
            if next_endpoint > LAST_ENDPOINT:
                states[capability] = CapabilityState(capability, WITHHELD, ("endpoint number space is exhausted for this node lifecycle",))
                continue
            identity = self._new_identity()
            if not isinstance(identity, str) or _IDENTITY.fullmatch(identity) is None or identity in ledger.identities:
                states[capability] = CapabilityState(capability, WITHHELD, ("generated identity is malformed or already recorded",))
                continue
            ledger.identities.add(identity)
            pending.append({"identity": identity, "capability": capability, "composition_sha256": digest, "endpoint": next_endpoint})
            next_endpoint += 1

        if pending:
            document = {"schema_version": SCHEMA_VERSION, "node_id": self._node_id, "records": ledger.records + pending}
            try:
                self._store.write(LEDGER_KEY, _encode(document))
            except (StoreError, OSError):
                reason = ("endpoint ledger could not be persisted",)
                for record in pending:
                    states[record["capability"]] = CapabilityState(record["capability"], WITHHELD, reason)
                faults += (f"{LEDGER_KEY} could not be written",)
            else:
                for record in pending:
                    states[record["capability"]] = CapabilityState(record["capability"], ELIGIBLE, (), record["identity"], record["endpoint"])
        return Reconciliation(tuple(states[c] for c in capabilities), faults)
