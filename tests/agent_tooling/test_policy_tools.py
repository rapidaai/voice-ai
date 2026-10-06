from __future__ import annotations

import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from helpers import ROOT, load_script


egress = load_script("agent_egress", "bin/agent-egress")
pr_body = load_script("check_pr_body", "bin/check-pr-body")
style = load_script("check_agent_style", "bin/check-agent-style")
required_tests = load_script("run_required_tests", ".codex/hooks/run_required_tests.py")


VALID_BODY = """## Summary

Adds repository policy checks.

- Workflow tier: Standard

## Approved Plan

- Acceptance criteria: Policy checks reject invalid input.
- In scope: Agent tooling.
- Out of scope: Product behavior.
- Ownership: Repository tooling.
- Rollback or disablement: Revert the tooling change.

## RFC

- Accepted RFC: N/A
- Approval: N/A

## Principles

- KISS / smallest complete solution: One policy entry point.
- YAGNI / rejected speculation: No new service abstraction.
- Single source of truth and ownership: Repository policy scripts.
- Contracts and compatibility: Tooling only.
- Failure safety and cleanup: Fail closed before publication.
- Security and least privilege: Read-only validation.
- Observability: Explicit command output.

## Testing

- `just validate-development-toolkit`

## Independent Code Review

- Reviewer: pending
- Decision: pending

## Checklist

- [ ] Review pending.
"""


class PullRequestBodyTest(unittest.TestCase):
    def test_accepts_complete_standard_body(self) -> None:
        self.assertEqual(pr_body.validate_title("feat: add agent policy"), [])
        self.assertEqual(pr_body.validate_body(VALID_BODY), [])

    def test_rejects_template_tier_placeholder(self) -> None:
        body = VALID_BODY.replace("Standard", "Fast / Standard / Governed", 1)
        self.assertIn("Summary must select one workflow tier", pr_body.validate_body(body))

    def test_rejects_empty_plan_field(self) -> None:
        body = VALID_BODY.replace("- Ownership: Repository tooling.", "- Ownership:")
        self.assertIn("Approved Plan must provide Ownership", pr_body.validate_body(body))

    def test_governed_body_requires_rfc_approval(self) -> None:
        body = VALID_BODY.replace("Standard", "Governed", 1)
        failures = pr_body.validate_body(body)
        self.assertIn("Governed work must identify the accepted RFC", failures)
        self.assertIn("Governed work must identify the RFC approval", failures)

        body = body.replace("- Accepted RFC: N/A", "- Accepted RFC: rfcs/0017-example.md")
        body = body.replace("- Approval: N/A", "- Approval: approved by platform owner")
        self.assertEqual(pr_body.validate_body(body), [])


class EgressFilterTest(unittest.TestCase):
    def test_allows_repository_documentation(self) -> None:
        self.assertIsNone(egress.path_refusal(ROOT, "AGENTS.md"))

    def test_rejects_environment_file(self) -> None:
        self.assertEqual(egress.path_refusal(ROOT, "ui/.env.production"), "environment file")

    def test_redacts_secret_looking_content(self) -> None:
        filtered, reasons = egress.filter_text('api_token="abcdefgh12345678"\nCLIENT_SECRET=unquotedsecret123')
        self.assertIn("redacted", filtered)
        self.assertEqual(reasons, ["secret-looking assignment", "secret-looking assignment"])

    def test_withholds_sensitive_diff_section(self) -> None:
        source = "\n".join(
            [
                "diff --git a/ui/.env.production b/ui/.env.production",
                "--- a/ui/.env.production",
                "+++ b/ui/.env.production",
                "+TOKEN=abcdefgh12345678",
            ]
        )
        filtered, manifest = egress.filter_diff(ROOT, source)
        self.assertIn("file withheld", filtered)
        self.assertEqual(len(manifest["withheld"]), 1)

    def test_allows_deleted_file_diff_when_source_no_longer_exists(self) -> None:
        path = "tests/agent_tooling/deleted-egress-fixture.py"
        source = "\n".join(
            [
                f"diff --git a/{path} b/{path}",
                "deleted file mode 100644",
                f"--- a/{path}",
                "+++ /dev/null",
                "-print('deleted')",
            ]
        )
        filtered, manifest = egress.filter_diff(ROOT, source)
        self.assertIn("-print('deleted')", filtered)
        self.assertEqual(manifest["withheld"], [])

    def test_withholds_deleted_sensitive_file_diff(self) -> None:
        source = "\n".join(
            [
                "diff --git a/ui/.env.production b/ui/.env.production",
                "deleted file mode 100644",
                "--- a/ui/.env.production",
                "+++ /dev/null",
                "-TOKEN=abcdefgh12345678",
            ]
        )
        filtered, manifest = egress.filter_diff(ROOT, source)
        self.assertIn("file withheld", filtered)
        self.assertEqual(len(manifest["withheld"]), 1)


class StyleCheckTest(unittest.TestCase):
    def test_rejects_em_dash_in_prose(self) -> None:
        line = style.AddedLine("docs/example.md", 3, "Unsafe prose — replace punctuation.")
        self.assertTrue(style.validate([line]))

    def test_accepts_direct_comment(self) -> None:
        line = style.AddedLine("pkg/example.go", 3, "// Preserve caller cancellation.")
        self.assertEqual(style.validate([line]), [])


class RequiredTestSelectionTest(unittest.TestCase):
    def test_compiles_integration_tag_only_go_directories_without_running_tests(self) -> None:
        with TemporaryDirectory() as root:
            directory = "api/example"
            path = Path(root, directory)
            path.mkdir(parents=True)
            path.joinpath("provider_test.go").write_text(
                "//go:build integration\n\npackage example_test\n"
            )

            self.assertEqual(
                required_tests._go_test_command(directory, root),
                ["go", "test", "-tags=integration", "-run", "^$", "./api/example"],
            )

    def test_runs_normal_go_directories_without_build_tags(self) -> None:
        with TemporaryDirectory() as root:
            directory = "api/example"
            path = Path(root, directory)
            path.mkdir(parents=True)
            path.joinpath("provider_test.go").write_text("package example_test\n")

            self.assertEqual(
                required_tests._go_test_command(directory, root),
                ["go", "test", "./api/example"],
            )

    def test_ui_tests_run_non_interactively(self) -> None:
        self.assertEqual(
            required_tests._ui_test_command(
                [
                    "ui/src/components/Auth.tsx",
                    "ui/src/components/Auth.test.tsx",
                ]
            ),
            [
                "yarn",
                "test",
                "--watch=false",
                "--runInBand",
                "src/components/Auth.test.tsx",
            ],
        )
        self.assertEqual(required_tests.UI_TEST_ENV, {"CI": "true"})

    def test_ui_test_selection_does_not_substitute_provider_tests(self) -> None:
        command = required_tests._ui_test_command(
            [
                "ui/src/components/Auth.tsx",
                "ui/src/components/Auth.test.tsx",
                "ui/src/providers/openai/provider.test.tsx",
            ]
        )
        self.assertIn("src/components/Auth.test.tsx", command or [])
        self.assertIn("src/providers/openai/provider.test.tsx", command or [])
        self.assertNotIn("providers", command or [])

    def test_ignores_generated_ui_policy_files(self) -> None:
        self.assertFalse(required_tests._is_ui_source("ui/src/AGENTS.md"))

    def test_selects_ui_source_files(self) -> None:
        self.assertTrue(required_tests._is_ui_source("ui/src/components/Provider.tsx"))


if __name__ == "__main__":
    unittest.main()
