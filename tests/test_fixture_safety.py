"""Regression tests for bounded fixture cleanup."""

from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
import time
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "scripts" / "run_upstream_fixture.sh"
GROUP_RUNNER = ROOT / "scripts" / "run_owned_group.sh"


class FixtureSafetyTest(unittest.TestCase):
    def test_fixture_does_not_request_broad_factory_reset(self) -> None:
        text = FIXTURE.read_text(encoding="utf-8")
        self.assertNotIn("--factory-reset", text)
        self.assertIn("run_owned_group.sh", text)
        self.assertEqual(text.count('TMPDIR="$work" HOME="$work/home"'), 3)
        self.assertIn('[[ -s "$work/chip_tool_kvs" ]]', text)
        self.assertIn("external_tmp/chip_tool_config.ini", text)

    @unittest.skipUnless(shutil.which("setsid") and shutil.which("timeout"), "requires Linux process-group tools")
    def test_forced_timeout_reaps_descendants_and_preserves_unrelated_tmp_state(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            child_pid_path = root / "child.pid"
            sentinel = Path("/tmp") / f"chip-helianthus-unrelated-{os.getpid()}"
            sentinel.write_text("preserve", encoding="utf-8")
            try:
                result = subprocess.run(
                    [
                        str(GROUP_RUNNER),
                        "1",
                        "bash",
                        "-c",
                        'sleep 300 & child=$!; printf "%s" "$child" > "$1"; wait "$child"',
                        "fixture-child",
                        str(child_pid_path),
                    ],
                    check=False,
                    timeout=15,
                )
                self.assertEqual(result.returncode, 124)
                self.assertEqual(sentinel.read_text(encoding="utf-8"), "preserve")
                child_pid = int(child_pid_path.read_text(encoding="utf-8"))
                for _ in range(30):
                    try:
                        os.kill(child_pid, 0)
                    except ProcessLookupError:
                        break
                    time.sleep(0.1)
                else:
                    self.fail(f"owned descendant {child_pid} survived forced timeout")
            finally:
                sentinel.unlink(missing_ok=True)


if __name__ == "__main__":
    unittest.main()
