from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

from helpers import ROOT, load_script


review = load_script("agent_review", "bin/agent-review")


def reviewer(reviewer_id: str, family: str) -> object:
    return review.Reviewer(
        reviewer_id=reviewer_id,
        model_family=family,
        transport="local",
        input_mode="stdin",
        command=(sys.executable, "-c", 'print(\'{"decision":"approve","summary":"ok","findings":[]}\')'),
        timeout_seconds=10,
    )


class AgentReviewTest(unittest.TestCase):
    def test_high_risk_requires_distinct_model_families(self) -> None:
        reviewers = [reviewer("first", "same-family"), reviewer("second", "same-family")]
        with self.assertRaisesRegex(ValueError, "distinct model families"):
            review.select_reviewers(reviewers, "high")

    def test_rejects_approval_with_blocking_finding(self) -> None:
        response = {
            "decision": "approve",
            "summary": "Looks good.",
            "findings": [
                {
                    "severity": "major",
                    "location": "bin/example:1",
                    "message": "A required check is skipped.",
                    "evidence": "The branch returns before validation.",
                    "remedy": "Run the check before returning.",
                }
            ],
        }
        with self.assertRaisesRegex(ValueError, "approving review"):
            review.validate_response(response)

    def test_runs_local_reviewer_with_stdin_contract(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            prompt = Path(directory) / "prompt.txt"
            prompt.write_text("review this", encoding="utf-8")
            result = review.run_reviewer(ROOT, reviewer("local", "family-a"), prompt, False)
        self.assertEqual(result["decision"], "approve")
        self.assertEqual(result["id"], "local")

    def test_external_reviewer_requires_explicit_authorization(self) -> None:
        configured = reviewer("external", "family-b")
        configured = review.Reviewer(
            reviewer_id=configured.reviewer_id,
            model_family=configured.model_family,
            transport="external",
            input_mode=configured.input_mode,
            command=configured.command,
            timeout_seconds=configured.timeout_seconds,
        )
        with tempfile.TemporaryDirectory() as directory:
            prompt = Path(directory) / "prompt.txt"
            prompt.write_text("review this", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "requires --allow-external"):
                review.run_reviewer(ROOT, configured, prompt, False)

    def test_panel_prefers_strongest_decision(self) -> None:
        reviews = [{"decision": "approve"}, {"decision": "revise"}, {"decision": "block"}]
        self.assertEqual(review.panel_decision(reviews), "block")

    def test_example_configuration_is_valid_but_disabled(self) -> None:
        reviewers = review.load_reviewers(ROOT / "agent-reviewers.example.json")
        self.assertEqual(reviewers, [])


if __name__ == "__main__":
    unittest.main()
