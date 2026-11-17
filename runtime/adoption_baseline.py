"""Offline proposal parsing against the pinned module identity.

Input schema (version 1) is an object with provider, module, release_tag,
checksum (Go h1 form), model ({name, version}), targets (nonempty string
roster), deferred_features (possibly empty string roster), and missing_evidence
(possibly empty string roster). All fields are required; rosters are ordered,
unique identities. Provider, module, release tag, checksum and model are valid
only when they equal the sdk object in the module pin. Other well-formed
values are rejected.

Optional retained evidence has exactly identity (all seven identity fields)
and retained_inputs (nonempty string references). References are retained as
claims only: this module never reads them or verifies archive bytes.
Even joined evidence leaves provenance and every B1-B8 qualification Unknown.
The adapter always denies admission and also returns the unchanged production
evaluator's independent decision. No allocation occurs.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from runtime import production_readiness


IDENTITY_FIELDS = (
    "provider", "module", "release_tag", "checksum", "model", "targets",
    "deferred_features",
)
_PIN_FIELDS = ("provider", "module", "release_tag", "checksum", "model")


@dataclass(frozen=True)
class Proposal:
    provider: str
    module: str
    release_tag: str
    checksum: str
    model: tuple[str, str]
    targets: tuple[str, ...]
    deferred_features: tuple[str, ...]
    missing_evidence: tuple[str, ...]


@dataclass(frozen=True)
class AdoptionResult:
    proposal: Proposal | None
    errors: tuple[str, ...]
    retained_inputs: tuple[str, ...]
    missing_evidence: tuple[str, ...]
    provenance: str
    qualification: tuple[tuple[str, str], ...]
    production_decision: production_readiness.Decision
    permitted: bool = False

    @property
    def proposal_valid(self) -> bool:
        """Structural validity is separate from provenance and qualification."""
        return self.proposal is not None and not self.errors


def _roster(value: Any, *, nonempty: bool = False) -> bool:
    return (
        isinstance(value, list)
        and (bool(value) or not nonempty)
        and all(isinstance(item, str) and bool(item) and item == item.strip() for item in value)
        and len(set(value)) == len(value)
    )


def module_pin() -> dict[str, Any]:
    """Return the sdk object recorded by the module pin."""
    path = Path(__file__).resolve().parents[1] / "sdk" / "matter-sdk.lock.json"
    sdk = json.loads(path.read_text(encoding="utf-8"))["sdk"]
    if not isinstance(sdk, dict):
        raise ValueError("module pin is not an object")
    return sdk


def _identity_matches(identity: dict, proposal: dict) -> bool:
    return all(identity[field] == proposal[field] for field in IDENTITY_FIELDS)


def evaluate_adoption_baseline(
    proposal: Any = None,
    retained_evidence: Any = None,
    *,
    production_lock: Any = None,
    composition: Any = None,
    build_evidence: Any = None,
) -> AdoptionResult:
    """Parse supplied claims without qualification or admission.

    Missing and malformed proposal fields are errors. Missing retained evidence
    is explicit evidence debt, not a structural error. Mismatched or malformed
    supplied evidence is an error and contributes no retained references. Caller
    build/composition objects go only to the real production evaluator.
    """
    decision = production_readiness.evaluate(production_lock, composition, build_evidence)
    errors: list[str] = []
    parsed = None
    retained: tuple[str, ...] = ()
    missing: tuple[str, ...] = ()
    fields = set(IDENTITY_FIELDS) | {"schema_version", "missing_evidence"}
    if not isinstance(proposal, dict) or set(proposal) != fields:
        errors.append("proposal fields are absent or unsupported")
    else:
        if type(proposal["schema_version"]) is not int or proposal["schema_version"] != 1:
            errors.append("proposal schema is unsupported")
        pin = module_pin()
        for field in _PIN_FIELDS:
            if proposal[field] != pin[field]:
                errors.append(f"proposal {field} must be the pinned value")
        for field in ("targets", "deferred_features", "missing_evidence"):
            if not _roster(proposal[field], nonempty=field == "targets"):
                errors.append(f"proposal {field} must be a unique string roster")
        if not errors:
            missing = tuple(proposal["missing_evidence"])
            parsed = Proposal(
                proposal["provider"], proposal["module"], proposal["release_tag"],
                proposal["checksum"], (proposal["model"]["name"], proposal["model"]["version"]),
                tuple(proposal["targets"]), tuple(proposal["deferred_features"]), missing,
            )
    if retained_evidence is None:
        missing += ("retained proposal evidence is missing",)
    elif not (
        isinstance(retained_evidence, dict)
        and set(retained_evidence) == {"identity", "retained_inputs"}
        and isinstance(retained_evidence["identity"], dict)
        and set(retained_evidence["identity"]) == set(IDENTITY_FIELDS)
        and _roster(retained_evidence["retained_inputs"], nonempty=True)
    ):
        errors.append("retained evidence is malformed")
        missing += ("usable retained proposal evidence is missing",)
    elif parsed is None or not _identity_matches(retained_evidence["identity"], proposal):
        errors.append("retained evidence identity differs from proposal")
        missing += ("joined retained proposal evidence is missing",)
    else:
        retained = tuple(retained_evidence["retained_inputs"])
    missing += ("source and checksum provenance is unverified", "B1-B8 qualification is unverified")
    return AdoptionResult(
        parsed, tuple(errors), retained, missing, "Unknown",
        tuple((f"B{number}", "Unknown") for number in range(1, 9)), decision,
    )
