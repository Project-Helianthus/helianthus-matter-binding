"""Check the documented upstream engagement boundary and its reference."""

import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


class UpstreamEngagementTest(unittest.TestCase):
    def test_board_decision_and_workflow_link(self):
        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        decision = (
            "Upstream engagement **only if SukramJ invites it** after seeing the "
            "fork and its reasons (Board 2026-11-08): then register Helianthus "
            "as the second consumer (reinstates the suspended deprecation window) "
            "and offer the fork PRs upstream under the CLA, each with the Board's "
            "OK. Until then no contact, no upstream issue or PR."
        )
        self.assertIn(decision, readme)
        self.assertIn("audit report §8 decision 10 and §8.1", readme)
        target = "docs/adoption/upstream-contribution-workflow.md"
        self.assertIn(f"]({target})", readme)
        workflow = (ROOT / target).read_text(encoding="utf-8")
        self.assertIn("Consuming the Apache-2.0 library and granting rights", workflow)
        self.assertIn("owner CLA review pending", workflow)
        self.assertIn("per-contribution authorization must precede it", workflow)
        self.assertIn("a local fix", workflow)
        self.assertIn("contacting the maintainer", workflow)


if __name__ == "__main__":
    unittest.main()
