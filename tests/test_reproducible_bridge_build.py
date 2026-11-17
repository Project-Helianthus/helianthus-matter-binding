"""The bridge build page states the standalone gateway record contract."""

import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
PAGE = ROOT / "docs" / "reproducible-bridge-build.md"

REQUIRED_SENTENCES = (
    "The bridge binary is the standalone gateway built without the nomatter profile.",
    "This repository builds no second executable.",
    "The targets are linux/arm64, and linux/arm with GOARM=7.",
    "Each target is built twice from a clean state with trimpath, "
    "version-control stamping off and an empty build id.",
    "The two digests of one target must be equal.",
    "Each recorded run carries the Go toolchain, the module checksum of this "
    "module, the module checksum of go-fabric, and an SBOM for that binary.",
    "The official add-on build is not this job.",
    "A bench run cites the digest and is a separate session.",
)


class ReproducibleBridgeBuildPageTest(unittest.TestCase):
    def test_page_states_each_build_record_sentence(self):
        text = PAGE.read_text(encoding="utf-8")
        for sentence in REQUIRED_SENTENCES:
            self.assertIn(sentence, text)


if __name__ == "__main__":
    unittest.main()
