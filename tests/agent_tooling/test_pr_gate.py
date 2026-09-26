from __future__ import annotations

import unittest

from helpers import ROOT, load_script, run_hook


pr_gate = load_script("pr_gate", ".claude/hooks/pr_gate.py")


class PullRequestGateTest(unittest.TestCase):
    def test_ignores_read_only_pull_request_command(self) -> None:
        self.assertEqual(pr_gate.invocations("gh pr view 123"), [])

    def test_finds_publish_command_in_compound_shell(self) -> None:
        calls = pr_gate.invocations("git status && gh pr create --title 'feat: add policy' --body-file pr.md")
        self.assertEqual(calls[0][0], "create")

    def test_create_requires_body_file(self) -> None:
        error = pr_gate.validate_call(ROOT, ROOT, "create", ["--title", "feat: add policy"])
        self.assertIn("requires --body-file", error or "")

    def test_inline_body_is_rejected(self) -> None:
        error = pr_gate.validate_call(ROOT, ROOT, "edit", ["--body", "text"])
        self.assertIn("must use --body-file", error or "")

    def test_nested_publish_command_fails_closed(self) -> None:
        result = run_hook(".claude/hooks/pr_gate.py", 'bash -c "gh pr create --fill"')
        self.assertEqual(result.returncode, 2)
        self.assertIn("cannot safely inspect", result.stderr)


if __name__ == "__main__":
    unittest.main()
