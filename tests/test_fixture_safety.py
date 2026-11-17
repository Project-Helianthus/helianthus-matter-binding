"""The hosted adapter workflow does not keep an external SDK checkout."""

from __future__ import annotations

import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REMOVED = (
    "scripts/checkout_sdk.sh",
    "scripts/run_upstream_fixture.sh",
    "scripts/run_owned_group.sh",
)


class CheckoutScriptTest(unittest.TestCase):
    def test_checkout_scripts_are_absent(self) -> None:
        for relative in REMOVED:
            self.assertFalse((ROOT / relative).exists(), relative)
        workflow = (ROOT / ".github" / "workflows" / "sdk-baseline.yml").read_text(encoding="utf-8")
        for relative in REMOVED:
            self.assertNotIn(Path(relative).name, workflow)


if __name__ == "__main__":
    unittest.main()
