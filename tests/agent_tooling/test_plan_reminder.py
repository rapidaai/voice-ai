from __future__ import annotations

import json
import os
import subprocess
import sys
import unittest
import uuid

from helpers import ROOT


class PlanReminderHookTest(unittest.TestCase):
    def run_hook(self, file_path: str, session_id: str | None) -> subprocess.CompletedProcess[str]:
        payload = {"tool_input": {"file_path": file_path}}
        if session_id is not None:
            payload["session_id"] = session_id
        environment = os.environ.copy()
        environment["CLAUDE_PROJECT_DIR"] = str(ROOT)
        return subprocess.run(
            [sys.executable, str(ROOT / ".claude/hooks/plan_reminder.py")],
            input=json.dumps(payload),
            capture_output=True,
            check=False,
            text=True,
            env=environment,
        )

    def test_reminds_once_before_source_edit(self) -> None:
        session = f"test-{uuid.uuid4().hex}"
        target = ROOT / "api/assistant-api/internal/vad/vad.go"
        first = self.run_hook(str(target), session)
        second = self.run_hook(str(target), session)
        self.assertEqual(first.returncode, 0, first.stderr)
        self.assertIn("Plan check before editing", first.stdout)
        self.assertEqual(second.stdout, "")

    def test_ignores_documentation_edit(self) -> None:
        result = self.run_hook(str(ROOT / "AGENTS.md"), f"test-{uuid.uuid4().hex}")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "")

    def test_missing_session_repeats_reminder(self) -> None:
        target = ROOT / ".claude/hooks/pr_gate.py"
        first = self.run_hook(str(target), None)
        second = self.run_hook(str(target), None)
        self.assertIn("Plan check before editing", first.stdout)
        self.assertIn("Plan check before editing", second.stdout)


if __name__ == "__main__":
    unittest.main()
