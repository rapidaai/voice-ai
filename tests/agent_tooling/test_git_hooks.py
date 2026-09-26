from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path

from helpers import ROOT


class CommitMessageHookTest(unittest.TestCase):
    def run_hook(self, message: str) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "message"
            path.write_text(message, encoding="utf-8")
            return subprocess.run(
                [str(ROOT / "githooks/commit-msg"), str(path)],
                cwd=ROOT,
                capture_output=True,
                check=False,
                text=True,
            )

    def test_accepts_repository_commit_format(self) -> None:
        result = self.run_hook("feat(agent): add policy checks\n")
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_rejects_unknown_commit_type(self) -> None:
        result = self.run_hook("feature: add policy checks\n")
        self.assertEqual(result.returncode, 1)

    def test_rejects_upper_case_subject(self) -> None:
        result = self.run_hook("feat: Add policy checks\n")
        self.assertEqual(result.returncode, 1)


if __name__ == "__main__":
    unittest.main()
