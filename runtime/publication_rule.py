"""The publication rule for live resources on the Matter bridge.

A live resource is published only when its catalog element has the disposition
``transformed`` on the derived Matter roster and that element is not named in
the single exclusion list, ``publication/exclusions.json``. The Go publication
package embeds the same file; there is no second list and no per-resource
switch. Every other disposition, an unmapped element and a malformed resource
id are withheld with a named reason. Several published elements of one
resource are one endpoint. A relation allocates no endpoint.

The endpoint store holds at most ``STORE_CAPACITY`` resource ids for the life
of a node; a further new id is withheld. This module creates no Matter node,
endpoint, fabric or commissioning state and is not an admission decision.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable, Mapping


EXCLUSIONS_PATH = Path(__file__).resolve().parents[1] / "publication" / "exclusions.json"

TRANSFORMED = "transformed"
UNKNOWN = "unknown"
UNSUPPORTED = "unsupported"
WITHHELD = "withheld"
KIND_RELATION = "relation"

REASON_RESOURCE_ID = "resource id is empty or malformed"
REASON_UNMAPPED = "element has no row on the derived roster"
REASON_EXCLUDED = "element is on the exclusion list"
REASON_RELATION = "a relation allocates no endpoint"
REASON_NO_DISPOSITION = "element has no disposition"
REASON_UNRECOGNISED = "element disposition is not recognised"
REASON_CAPACITY = "endpoint store holds its full capacity of resource ids"
REASON_EXHAUSTED = "endpoint number space is exhausted for this node lifecycle"

STORE_CAPACITY = 64

_NAMED = frozenset({UNKNOWN, UNSUPPORTED, WITHHELD})
_TOKEN = re.compile(r"[^\x00\r\n\t]+")


class ExclusionListError(ValueError):
    """The exclusion list is malformed; nothing is published from it."""


@dataclass(frozen=True)
class RosterRow:
    """One row of the derived Matter roster."""

    element: str
    kind: str
    disposition: str


def is_token(value: Any) -> bool:
    """Non-empty text without surrounding blanks or control separators."""
    return isinstance(value, str) and value == value.strip() and _TOKEN.fullmatch(value) is not None


def _unique_pairs(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    document: dict[str, Any] = {}
    for key, value in pairs:
        if key in document:
            raise ExclusionListError("exclusion list has a duplicate key")
        document[key] = value
    return document


def parse_exclusions(data: bytes) -> frozenset[str]:
    """Parse an exclusion list: one object whose only member is the array ``elements``."""
    try:
        document = json.loads(data.decode("utf-8"), object_pairs_hook=_unique_pairs)
    except (ValueError, UnicodeDecodeError, RecursionError) as error:
        raise ExclusionListError(f"exclusion list is not a JSON document: {error}") from error
    if not isinstance(document, dict) or set(document) != {"elements"} or not isinstance(document["elements"], list):
        raise ExclusionListError("exclusion list must be an object with one array 'elements'")
    elements = document["elements"]
    if not all(is_token(element) for element in elements) or len(set(elements)) != len(elements):
        raise ExclusionListError("exclusion list has an empty, malformed or duplicate element")
    return frozenset(elements)


def load_exclusions(path: Path = EXCLUSIONS_PATH) -> frozenset[str]:
    """Read the committed exclusion list."""
    return parse_exclusions(path.read_bytes())


class PublicationRule:
    """The rule over one derived roster and one exclusion list."""

    def __init__(self, rows: Iterable[RosterRow], exclusions: frozenset[str] | None = None) -> None:
        indexed: dict[str, RosterRow] = {}
        for row in rows:
            if not isinstance(row, RosterRow) or not is_token(row.element):
                raise ValueError("roster row has an empty or malformed element")
            if row.element in indexed:
                raise ValueError(f"roster has more than one row for {row.element!r}")
            indexed[row.element] = row
        self._rows: Mapping[str, RosterRow] = indexed
        self._excluded = load_exclusions() if exclusions is None else frozenset(exclusions)

    def withhold_reason(self, element: Any, resource_id: Any) -> str | None:
        """Return the named reason a live element is withheld, or None when it is published."""
        if not is_token(resource_id):
            return REASON_RESOURCE_ID
        row = self._rows.get(element) if isinstance(element, str) else None
        if row is None:
            return REASON_UNMAPPED
        if element in self._excluded:
            return REASON_EXCLUDED
        if row.disposition == TRANSFORMED:
            return REASON_RELATION if row.kind == KIND_RELATION else None
        if not row.disposition:
            return REASON_NO_DISPOSITION
        if row.disposition in _NAMED:
            return f"element disposition is {row.disposition}"
        return REASON_UNRECOGNISED


def allocation_refusal(stored_ids: int, next_endpoint: int, last_endpoint: int) -> str | None:
    """Refuse a new number once the store holds its capacity or the number space is spent."""
    if stored_ids >= STORE_CAPACITY:
        return REASON_CAPACITY
    if next_endpoint > last_endpoint:
        return REASON_EXHAUSTED
    return None
