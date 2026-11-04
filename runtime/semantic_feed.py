"""Bounded consumer of the Gateway GraphQL semantic snapshot feed.

The feed reads the `semanticSnapshot(assetIds:)` root field through an injected,
already authenticated client. This module opens no connection, performs no HTTP,
applies no authentication policy, and creates no Matter node, endpoint, fabric,
or commissioning state. It retains the exact snapshot identity, revision vector,
fact values and projection loss for each asset, and derives an advisory endpoint
selection input from them.

A feed offer never authorizes allocation. Allocation stays behind the unchanged
readiness evaluator in `runtime.endpoint_selection`; an offer only names a
capability the Portal may present for selection.
"""

from __future__ import annotations

import json
import re
import time
from dataclasses import dataclass
from typing import Any, Callable, Mapping, Protocol


MAX_ASSETS = 32
MAX_TEXT = 256
MAX_DISPOSITIONS = 4096
MAX_FACTS = 4096
MAX_CANDIDATES = 64
MAX_LOSS = 64
MAX_DESCRIPTION = 1024
MAX_ERRORS = 16
MAX_ATTEMPTS = 8
MAX_TIMEOUT_S = 60.0
MAX_BACKOFF_S = 60.0

REVISION_AXES = ("semantic", "identity", "facts", "services", "capabilities")
UNAVAILABLE_REASONS = frozenset(
    "semantic.read.reason." + reason
    for reason in ("asset_unknown", "snapshot_unavailable", "source_unavailable", "publication_withheld")
)
OUTCOMES = frozenset({"exact", "transformed", "withheld", "unrepresentable", "unsupported", "unknown"})
ITEM_KINDS = frozenset({"fact", "relation", "capability", "operation"})
LOSS_KINDS = frozenset({"unit", "range", "precision", "time", "symbol", "provenance", "identity", "capability", "operation", "policy"})

_VALUE = "kind quantity { number { coefficient exponent10 } unit } boolean text symbol { namespace token known } symbols { namespace token known } time { unixNanoseconds clockId uncertaintyNs }"
_KEY = "packId packVersion factId dimensions { id value { " + _VALUE + " } }"
_REVISIONS = "revisions { " + " ".join(REVISION_AXES) + " }"
QUERY = (
    "query SemanticFeed($assetIds: [ID!]!) { semanticSnapshot(assetIds: $assetIds) { assets { "
    "assetId unavailableReason "
    "snapshot { snapshotId " + _REVISIONS + " facts { revision key { " + _KEY + " } "
    "candidates { candidateId revision value { " + _VALUE + " } } } } "
    "projection { snapshotId " + _REVISIONS + " dispositions { kind itemId outcome reason "
    "loss { kind sourceItems description reversible } } } } } }"
)

_UINT64 = re.compile(r"0|[1-9][0-9]{0,19}")
_UINT64_MAX = (1 << 64) - 1
_COEFFICIENT = re.compile(r"-?(0|[1-9][0-9]{0,38})")
_VALUE_KINDS = frozenset({"quantity", "boolean", "text", "symbol", "symbols", "time"})
MAX_DIMENSIONS = 64
MAX_EXPONENT10 = 1000
_WAIT_POLL_S = 0.05
_EXHAUSTED = object()


class TransportError(Exception):
    """Raised by a client whose connection failed; the feed may reconnect and retry."""


class Cancelled(Exception):
    """Raised by a client, or by the feed, when the caller cancelled the read."""


class CancelToken(Protocol):
    def is_set(self) -> bool: ...


class SnapshotClient(Protocol):
    """An already authenticated GraphQL client owned and configured by the caller."""

    def execute(self, document: str, variables: Mapping[str, Any], *, timeout: float, cancel: CancelToken) -> Any: ...

    def reconnect(self, *, timeout: float, cancel: CancelToken) -> None: ...


class _Malformed(Exception):
    pass


@dataclass(frozen=True)
class Revisions:
    semantic: int
    identity: int
    facts: int
    services: int
    capabilities: int

    def axes(self) -> tuple[int, ...]:
        return tuple(getattr(self, axis) for axis in REVISION_AXES)


@dataclass(frozen=True)
class Loss:
    kind: str
    source_items: tuple[str, ...]
    description: str
    reversible: bool


@dataclass(frozen=True)
class Disposition:
    kind: str
    item_id: str
    outcome: str
    reason: str | None
    loss: tuple[Loss, ...]


@dataclass(frozen=True)
class Candidate:
    candidate_id: str
    revision: int
    value: str | None  # canonical JSON of the exact published value; None when absent


@dataclass(frozen=True)
class Fact:
    key: str  # canonical JSON of the complete dimensioned key
    revision: int
    candidates: tuple[Candidate, ...]


@dataclass(frozen=True)
class AssetState:
    asset_id: str
    snapshot_id: str | None
    revisions: Revisions | None
    facts: tuple[Fact, ...]
    dispositions: tuple[Disposition, ...]
    unavailable_reason: str | None = None


@dataclass(frozen=True)
class FeedState:
    assets: tuple[AssetState, ...]
    complete: bool
    current: bool
    faults: tuple[str, ...]


@dataclass(frozen=True)
class Offer:
    capability: str
    asset_id: str
    item_id: str
    snapshot_id: str
    revisions: Revisions


@dataclass(frozen=True)
class Withheld:
    capability: str
    reasons: tuple[str, ...]


@dataclass(frozen=True)
class SelectionInput:
    """Advisory input for Portal selection; it carries no allocation authority."""

    offers: tuple[Offer, ...]
    withheld: tuple[Withheld, ...]


def _text(value: Any, limit: int = MAX_TEXT) -> str:
    if not isinstance(value, str) or not value or value != value.strip() or len(value) > limit:
        raise _Malformed("malformed text member")
    return value


def _optional_text(value: Any) -> str | None:
    return None if value is None else _text(value)


def _uint64(value: Any) -> int:
    if not isinstance(value, str) or _UINT64.fullmatch(value) is None or int(value) > _UINT64_MAX:
        raise _Malformed("malformed revision")
    return int(value)


def _list(value: Any, limit: int) -> list[Any]:
    if not isinstance(value, list) or len(value) > limit:
        raise _Malformed("malformed or oversized list")
    return value


def _object(value: Any) -> Mapping[str, Any]:
    if not isinstance(value, Mapping):
        raise _Malformed("malformed object")
    return value


def _canonical(value: Any) -> str:
    def strip(item: Any) -> Any:
        if isinstance(item, Mapping):
            return {key: strip(member) for key, member in item.items() if member is not None}
        if isinstance(item, list):
            return [strip(member) for member in item]
        return item

    try:
        return json.dumps(strip(value), sort_keys=True, separators=(",", ":"), ensure_ascii=True, allow_nan=False)
    except (TypeError, ValueError, RecursionError) as error:
        raise _Malformed("value is not representable") from error


def _symbol(raw: Any) -> None:
    symbol = _object(raw)
    _text(symbol.get("namespace"))
    _text(symbol.get("token"))
    if not isinstance(symbol.get("known"), bool):
        raise _Malformed("malformed symbol")


def _value(raw: Any) -> Mapping[str, Any]:
    """Validate the structure a published value must have for its declared kind."""
    value = _object(raw)
    kind = value.get("kind")
    if kind not in _VALUE_KINDS:
        raise _Malformed("unknown value kind")
    member = value.get(kind)
    if kind == "quantity":
        quantity = _object(member)
        number = _object(quantity.get("number"))
        coefficient, exponent = number.get("coefficient"), number.get("exponent10")
        if not isinstance(coefficient, str) or _COEFFICIENT.fullmatch(coefficient) is None:
            raise _Malformed("malformed quantity coefficient")
        if not isinstance(exponent, int) or isinstance(exponent, bool) or abs(exponent) > MAX_EXPONENT10:
            raise _Malformed("malformed quantity exponent")
        _text(quantity.get("unit"))
    elif kind == "boolean":
        if not isinstance(member, bool):
            raise _Malformed("malformed boolean value")
    elif kind == "text":
        if not isinstance(member, str) or len(member) > MAX_DESCRIPTION:
            raise _Malformed("malformed text value")
    elif kind == "symbol":
        _symbol(member)
    elif kind == "symbols":
        for item in _list(member, MAX_CANDIDATES):
            _symbol(item)
    else:
        time_value = _object(member)
        if time_value.get("unixNanoseconds") is None:
            raise _Malformed("malformed time value")
    return value


def _fact_key(raw: Any) -> str:
    key = _object(raw)
    for member in ("packId", "packVersion", "factId"):
        _text(key.get(member))
    for dimension in _list(key.get("dimensions"), MAX_DIMENSIONS):
        dimension = _object(dimension)
        _text(dimension.get("id"))
        _value(dimension.get("value"))
    return _canonical(key)


def _revisions(record: Mapping[str, Any]) -> Revisions:
    raw = _object(record.get("revisions"))
    return Revisions(**{axis: _uint64(raw.get(axis)) for axis in REVISION_AXES})


def _parse_asset(raw: Any) -> AssetState:
    record = _object(raw)
    asset_id = _text(record.get("assetId"))
    reason = record.get("unavailableReason")
    snapshot, projection = record.get("snapshot"), record.get("projection")
    if reason is not None:
        if reason not in UNAVAILABLE_REASONS or snapshot is not None or projection is not None:
            raise _Malformed("unavailable asset carries state or an unknown reason")
        return AssetState(asset_id, None, None, (), (), reason)
    snapshot, projection = _object(snapshot), _object(projection)
    snapshot_id = _text(snapshot.get("snapshotId"))
    revisions = _revisions(snapshot)
    if projection.get("snapshotId") != snapshot_id or _revisions(projection) != revisions:
        raise _Malformed("projection and snapshot identities are mixed")

    facts = []
    for envelope in _list(snapshot.get("facts"), MAX_FACTS):
        envelope = _object(envelope)
        candidates = []
        for candidate in _list(envelope.get("candidates"), MAX_CANDIDATES):
            candidate = _object(candidate)
            value = candidate.get("value")
            candidates.append(Candidate(_text(candidate.get("candidateId")), _uint64(candidate.get("revision")), None if value is None else _canonical(_value(value))))
        facts.append(Fact(_fact_key(envelope.get("key")), _uint64(envelope.get("revision")), tuple(candidates)))

    dispositions = []
    seen: set[tuple[str, str]] = set()
    for disposition in _list(projection.get("dispositions"), MAX_DISPOSITIONS):
        disposition = _object(disposition)
        kind, item_id, outcome = disposition.get("kind"), _text(disposition.get("itemId")), disposition.get("outcome")
        if kind not in ITEM_KINDS or outcome not in OUTCOMES or (kind, item_id) in seen:
            raise _Malformed("unknown or duplicate disposition")
        seen.add((kind, item_id))
        losses = []
        for loss in _list(disposition.get("loss"), MAX_LOSS):
            loss = _object(loss)
            description = loss.get("description")
            if loss.get("kind") not in LOSS_KINDS or not isinstance(loss.get("reversible"), bool) or not isinstance(description, str) or len(description) > MAX_DESCRIPTION:
                raise _Malformed("malformed loss detail")
            sources = tuple(_text(item) for item in _list(loss.get("sourceItems"), MAX_LOSS))
            losses.append(Loss(loss["kind"], sources, description, loss["reversible"]))
        dispositions.append(Disposition(kind, item_id, outcome, _optional_text(disposition.get("reason")), tuple(losses)))
    return AssetState(asset_id, snapshot_id, revisions, tuple(facts), tuple(dispositions))


def _error_codes(errors: Any) -> tuple[str, ...]:
    codes = []
    for error in errors[:MAX_ERRORS] if isinstance(errors, list) else [None]:
        code = error.get("extensions", {}).get("code") if isinstance(error, Mapping) and isinstance(error.get("extensions"), Mapping) else None
        codes.append(code if isinstance(code, str) and re.fullmatch(r"[A-Z_]{1,64}", code) else "UNSPECIFIED")
    return tuple(sorted(set(codes)))


def _merge(old: AssetState | None, new: AssetState) -> tuple[AssetState | None, str | None]:
    """Return the retained state for one asset, or a fault that rejects the response."""
    if new.revisions is None:
        if old is None or old.revisions is None:
            return new, None
        # Unavailability withholds the last-known-good state but never lowers its revision floor.
        return AssetState(old.asset_id, old.snapshot_id, old.revisions, old.facts, old.dispositions, new.unavailable_reason), None
    if old is None or old.revisions is None:
        return new, None
    pairs = list(zip(new.revisions.axes(), old.revisions.axes()))
    if any(n < o for n, o in pairs):
        kind = "mixed" if any(n > o for n, o in pairs) else "older"
        return None, f"{new.asset_id} carries {kind} revisions than the retained state"
    if all(n == o for n, o in pairs):
        retained = AssetState(old.asset_id, old.snapshot_id, old.revisions, old.facts, old.dispositions)
        if retained != new:
            return None, f"{new.asset_id} changed content without a revision change"
    return new, None


class SemanticFeed:
    """Retain the newest complete semantic feed state for a fixed asset set."""

    def __init__(
        self,
        client: SnapshotClient,
        asset_ids: list[str] | tuple[str, ...],
        *,
        max_attempts: int = 3,
        timeout_s: float = 10.0,
        backoff_s: tuple[float, ...] = (0.5, 2.0, 8.0),
        sleep: Callable[[float, CancelToken], bool] | None = None,
    ) -> None:
        ids = list(asset_ids)
        if not 1 <= len(ids) <= MAX_ASSETS or len(set(ids)) != len(ids):
            raise ValueError("asset selection must hold 1 to 32 unique identities")
        for asset_id in ids:
            try:
                _text(asset_id)
            except _Malformed:
                raise ValueError("asset identity must be bounded text") from None
            if "://" in asset_id:
                raise ValueError("asset identity must not be a native resource locator")
        if not isinstance(max_attempts, int) or not 1 <= max_attempts <= MAX_ATTEMPTS:
            raise ValueError("attempts must be bounded")
        if not 0 < timeout_s <= MAX_TIMEOUT_S or not backoff_s or any(not 0 <= delay <= MAX_BACKOFF_S for delay in backoff_s):
            raise ValueError("timeout and backoff must be bounded")
        self._client = client
        self._asset_ids = tuple(sorted(ids))
        self._max_attempts = max_attempts
        self._timeout = timeout_s
        self._backoff = tuple(backoff_s)
        self._sleep = sleep if sleep is not None else _wait
        self._assets: dict[str, AssetState] = {}
        self._current = False
        self._faults: tuple[str, ...] = ()

    @property
    def asset_ids(self) -> tuple[str, ...]:
        return self._asset_ids

    def state(self) -> FeedState:
        return FeedState(
            tuple(self._assets[asset_id] for asset_id in self._asset_ids if asset_id in self._assets),
            len(self._assets) == len(self._asset_ids),
            self._current,
            self._faults,
        )

    def refresh(self, cancel: CancelToken) -> FeedState:
        """Read once, reconnecting on transport failure within the attempt bound.

        Cancellation, transport exhaustion, GraphQL errors and malformed or
        regressing responses leave every retained asset unchanged and mark the
        state not current. Only a complete, coherent response is applied.
        """
        self._current = False
        try:
            response = self._read(cancel)
        except Cancelled:
            self._faults = ("cancelled",)
            raise
        except Exception:
            self._faults = ("client failed",)
            raise
        if response is _EXHAUSTED:
            return self.state()
        self._apply(response)
        return self.state()

    def _read(self, cancel: CancelToken) -> Any:
        variables = {"assetIds": list(self._asset_ids)}
        for attempt in range(self._max_attempts):
            if cancel.is_set():
                raise Cancelled()
            try:
                if attempt:
                    self._client.reconnect(timeout=self._timeout, cancel=cancel)
                response = self._client.execute(QUERY, variables, timeout=self._timeout, cancel=cancel)
            except TransportError:
                if attempt + 1 == self._max_attempts:
                    break
                if not self._sleep(self._backoff[min(attempt, len(self._backoff) - 1)], cancel):
                    raise Cancelled() from None
                continue
            if cancel.is_set():
                raise Cancelled()
            return response
        self._faults = (f"transport failed after {self._max_attempts} attempts",)
        return _EXHAUSTED

    def _apply(self, response: Any) -> None:
        if not isinstance(response, Mapping):
            self._faults = ("response is not a GraphQL result",)
            return
        if response.get("errors") is not None:
            self._faults = tuple(f"graphql error {code}" for code in _error_codes(response["errors"]))
            return
        try:
            data = _object(response.get("data"))
            assets = _list(_object(data.get("semanticSnapshot")).get("assets"), MAX_ASSETS)
            parsed = [_parse_asset(asset) for asset in assets]
        except _Malformed as error:
            self._faults = (f"response rejected: {error}",)
            return
        ids = [asset.asset_id for asset in parsed]
        if sorted(ids) != list(self._asset_ids):
            self._faults = ("response omitted, duplicated or substituted an asset",)
            return
        merged: dict[str, AssetState] = {}
        for asset in parsed:
            retained, fault = _merge(self._assets.get(asset.asset_id), asset)
            if fault is not None:
                self._faults = (fault,)
                return
            assert retained is not None
            merged[asset.asset_id] = retained
        self._assets = merged
        self._current = True
        self._faults = ()

    def selection_input(self) -> SelectionInput:
        """Offer only exact, lossless capabilities from a current complete state."""
        state = self.state()
        feed_reasons = []
        if not state.complete:
            feed_reasons.append("feed state is incomplete")
        if not state.current:
            feed_reasons.append("feed state is not current")
        offers: list[Offer] = []
        withheld: list[Withheld] = []
        for asset in state.assets:
            for disposition in asset.dispositions:
                if disposition.kind != "capability":
                    continue
                capability = f"{asset.asset_id}/{disposition.item_id}"
                reasons = list(feed_reasons)
                if len(capability) > MAX_TEXT:
                    reasons.append("capability name exceeds the selection bound")
                if asset.unavailable_reason is not None:
                    reasons.append(asset.unavailable_reason)
                if disposition.outcome != "exact":
                    reasons.append(disposition.reason or f"projection outcome is {disposition.outcome}")
                elif disposition.loss:
                    reasons.append("exact projection carries loss")
                if reasons:
                    withheld.append(Withheld(capability, tuple(reasons)))
                else:
                    assert asset.snapshot_id is not None and asset.revisions is not None
                    offers.append(Offer(capability, asset.asset_id, disposition.item_id, asset.snapshot_id, asset.revisions))
        return SelectionInput(tuple(offers), tuple(withheld))


def _wait(seconds: float, cancel: CancelToken) -> bool:
    """Wait for the backoff; return False when the token cancelled it."""
    waiter = getattr(cancel, "wait", None)
    if callable(waiter):
        return not waiter(seconds)
    deadline = time.monotonic() + seconds
    while not cancel.is_set():
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            return True
        time.sleep(min(remaining, _WAIT_POLL_S))
    return False
