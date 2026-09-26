from __future__ import annotations

import json
import os
import subprocess
import sys
import unittest
import uuid

from helpers import ROOT


class PathRulesHookTest(unittest.TestCase):
    def run_hook(self, file_path: str, session_id: str | None) -> subprocess.CompletedProcess[str]:
        payload = {"tool_input": {"file_path": file_path}}
        if session_id is not None:
            payload["session_id"] = session_id
        environment = os.environ.copy()
        environment["CLAUDE_PROJECT_DIR"] = str(ROOT)
        return subprocess.run(
            [sys.executable, str(ROOT / ".claude/hooks/path_rules.py")],
            input=json.dumps(payload),
            capture_output=True,
            check=False,
            text=True,
            env=environment,
        )

    def test_injects_matching_rule_once_per_session(self) -> None:
        session = f"test-{uuid.uuid4().hex}"
        target = ROOT / "api/assistant-api/internal/vad/vad.go"
        first = self.run_hook(str(target), session)
        second = self.run_hook(str(target), session)
        self.assertEqual(first.returncode, 0, first.stderr)
        self.assertIn("Voice activity detection", first.stdout)
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertEqual(second.stdout, "")

    def test_ignores_paths_outside_repository(self) -> None:
        result = self.run_hook("/tmp/outside.go", f"test-{uuid.uuid4().hex}")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "")

    def test_missing_session_does_not_suppress_later_context(self) -> None:
        target = ROOT / "api/assistant-api/internal/vad/vad.go"
        first = self.run_hook(str(target), None)
        second = self.run_hook(str(target), None)
        self.assertIn("Voice activity detection", first.stdout)
        self.assertIn("Voice activity detection", second.stdout)


if __name__ == "__main__":
    unittest.main()
