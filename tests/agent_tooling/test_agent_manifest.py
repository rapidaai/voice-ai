from __future__ import annotations

import json
import subprocess
import sys
import unittest

from helpers import ROOT, load_script


manifest = load_script("check_agent_manifest", "bin/check-agent-manifest")


class AgentManifestTest(unittest.TestCase):
    def test_repository_manifest_is_valid(self) -> None:
        document = json.loads((ROOT / "agent-tooling-manifest.json").read_text(encoding="utf-8"))
        failures = manifest.validate_document(ROOT, document, manifest.just_recipes(ROOT))
        self.assertEqual(failures, [])

    def test_duplicate_capability_is_rejected(self) -> None:
        document = {
            "version": 1,
            "state_definitions": {state: state for state in manifest.VALID_STATES},
            "capabilities": [
                {
                    "id": "duplicate",
                    "state": "implemented",
                    "owner": "AGENTS.md",
                    "evidence": ["AGENTS.md"],
                    "verification": ["just test-agent-hooks"],
                    "limitation": "fixture",
                },
                {
                    "id": "duplicate",
                    "state": "designed",
                    "owner": "AGENTS.md",
                    "evidence": ["AGENTS.md"],
                    "verification": ["just test-agent-hooks"],
                    "limitation": "fixture",
                },
            ],
            "coverage_contracts": [
                {
                    "id": "fixture",
                    "paths": ["AGENTS.md"],
                    "contract": "fixture",
                    "checks": ["just test-agent-hooks"],
                }
            ],
            "review_policy": {
                "standard": {"minimum_reviewers": 1},
                "high": {"minimum_reviewers": 2, "distinct_model_families": True},
            },
        }
        failures = manifest.validate_document(ROOT, document, manifest.just_recipes(ROOT))
        self.assertIn("duplicate capability id: duplicate", failures)

    def test_installer_help_is_available(self) -> None:
        result = subprocess.run(
            [str(ROOT / "bin/install-agent-tooling"), "--help"],
            cwd=ROOT,
            capture_output=True,
            check=False,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("--check", result.stdout)


if __name__ == "__main__":
    unittest.main()
