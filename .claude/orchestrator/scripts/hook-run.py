#!/usr/bin/env python3
import argparse
import hashlib
import hmac
import json
import os
import sys
from pathlib import Path

import jsonschema


def err(code: str, message: str, **extra):
    out = {"code": code, "message": message}
    out.update(extra)
    return out


def validate_stage_schema(data: dict, stage: str) -> list[dict]:
    schema_dir = Path(__file__).resolve().parent.parent / "schemas"
    schema_paths = [schema_dir / "envelope.schema.json", schema_dir / f"{stage}-input.schema.json"]
    for schema_path in schema_paths:
        try:
            schema = json.loads(schema_path.read_text(encoding="utf-8"))
            jsonschema.validate(instance=data, schema=schema)
        except jsonschema.ValidationError as exc:
            location = ".".join(str(part) for part in exc.absolute_path)
            return [
                err(
                    "SCHEMA_VALIDATION_FAILED",
                    exc.message,
                    path=location,
                    schema=str(schema_path),
                )
            ]
        except (OSError, json.JSONDecodeError, jsonschema.SchemaError) as exc:
            return [err("SCHEMA_LOAD_FAILED", str(exc), schema=str(schema_path))]
    return []


def load_approved_plan(data: dict) -> tuple[dict | None, list[dict]]:
    artifacts = data.get("artifacts") or {}
    plan_file = str(artifacts.get("approved_plan_file", "")).strip()
    expected_sha256 = str(artifacts.get("approved_plan_sha256", "")).strip()
    expected_hmac = str(artifacts.get("approved_plan_hmac", "")).strip()
    gate_key = os.environ.get("DEVELOPMENT_GATE_KEY", "")
    if not plan_file or not expected_sha256 or not expected_hmac:
        return None, [err("MISSING_PLAN_ARTIFACT", "Approved plan file, SHA-256, and HMAC are required")]
    if not gate_key:
        return None, [
            err(
                "MISSING_GATE_KEY",
                "DEVELOPMENT_GATE_KEY is required and must be held by the coordinator",
            )
        ]

    plan_path = Path(plan_file)
    if not plan_path.is_absolute():
        plan_path = Path(str(data.get("repo_root", ""))) / plan_path

    try:
        raw_plan = plan_path.read_bytes()
        approved_plan = json.loads(raw_plan.decode("utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        return None, [err("PLAN_ARTIFACT_LOAD_FAILED", str(exc), file=str(plan_path))]

    actual_sha256 = hashlib.sha256(raw_plan).hexdigest()
    if actual_sha256 != expected_sha256:
        return None, [
            err(
                "PLAN_ARTIFACT_DIGEST_MISMATCH",
                "Approved plan artifact does not match its recorded SHA-256",
                expected=expected_sha256,
                actual=actual_sha256,
            )
        ]
    run_id = str(data.get("run_id", "")).strip()
    signed_message = f"{run_id}:{actual_sha256}".encode("utf-8")
    actual_hmac = hmac.new(gate_key.encode("utf-8"), signed_message, hashlib.sha256).hexdigest()
    if not hmac.compare_digest(actual_hmac, expected_hmac):
        return None, [
            err(
                "PLAN_ATTESTATION_INVALID",
                "Approved plan HMAC does not match coordinator attestation",
            )
        ]
    if approved_plan != data.get("task_plan"):
        return None, [
            err(
                "PLAN_ARTIFACT_MISMATCH",
                "task_plan differs from the approved plan artifact",
                file=str(plan_path),
            )
        ]
    return approved_plan, []


def _is_in_allowed(path: str, allowed_paths: list[str]) -> bool:
    for p in allowed_paths:
        if p.endswith("/"):
            if path.startswith(p):
                return True
        elif path == p:
            return True
    return False


def _is_in_blocked(path: str, blocked_paths: list[str]) -> bool:
    for p in blocked_paths:
        if p.endswith("/"):
            if path.startswith(p):
                return True
        elif path == p:
            return True
    return False


def _required_provider_config_path(data: dict, plan: dict) -> str:
    provider = str((data.get("task") or {}).get("provider", "")).strip()
    if not provider:
        return ""

    explicit = str(plan.get("required_provider_config", "")).strip()
    if explicit:
        return explicit.replace("{provider}", provider)

    skill = str((data.get("task") or {}).get("skill", "")).strip()
    mapping = {
        "noise-reduction-integration": "ui/src/providers/{provider}/noise.json",
        "vad-integration": "ui/src/providers/{provider}/vad.json",
        "end-of-speech-integration": "ui/src/providers/{provider}/eos.json",
        "stt-integration": "ui/src/providers/{provider}/stt.json",
        "tts-integration": "ui/src/providers/{provider}/tts.json",
    }
    pattern = mapping.get(skill, "")
    if not pattern:
        return ""
    return pattern.format(provider=provider)


def _base_envelope_checks(data: dict) -> list[dict]:
    errors = []
    if not isinstance(data.get("task"), dict):
        errors.append(err("MISSING_TASK", "task is required"))
        return errors
    task = data["task"]
    for k in ("id", "type", "skill"):
        if not str(task.get(k, "")).strip():
            errors.append(err("MISSING_FIELD", f"task.{k} is required"))

    if str(task.get("type", "")).strip() == "integration" and not str(task.get("provider", "")).strip():
        errors.append(err("MISSING_PROVIDER", "task.provider is required for integration tasks"))
    return errors


def run_pre_implementation(data: dict) -> dict:
    errors = _base_envelope_checks(data)
    warnings = []
    checks = {
        "plan_presence": "fail",
        "scope_declared": "fail",
        "tests_declared": "fail",
        "commands_declared": "fail",
        "principles_declared": "fail",
        "ownership_declared": "fail",
        "discussion_approved": "fail",
    }

    approved_plan, artifact_errors = load_approved_plan(data)
    errors.extend(artifact_errors)
    plan = approved_plan if approved_plan is not None else data.get("task_plan")
    if not isinstance(plan, dict):
        errors.append(err("MISSING_PLAN", "task_plan is required"))
        return {"status": "fail", "errors": errors, "warnings": warnings, "checks": checks}

    checks["plan_presence"] = "pass"

    allowed = plan.get("allowed_paths") or []
    out_scope = plan.get("out_of_scope_paths") or []
    req_tests = plan.get("required_tests") or []
    req_cmds = plan.get("required_commands") or []
    acceptance = plan.get("acceptance_criteria") or []
    non_goals = plan.get("non_goals") or []
    ownership = plan.get("ownership") or {}
    principles = plan.get("principles") or {}
    discussion = plan.get("discussion") or {}

    if allowed and "out_of_scope_paths" in plan and isinstance(out_scope, list):
        checks["scope_declared"] = "pass"
    else:
        errors.append(err("MISSING_SCOPE", "allowed_paths and out_of_scope_paths are required"))

    if req_tests:
        checks["tests_declared"] = "pass"
    else:
        errors.append(err("MISSING_TESTS", "required_tests must not be empty"))

    if req_cmds:
        checks["commands_declared"] = "pass"
    else:
        errors.append(err("MISSING_COMMANDS", "required_commands must not be empty"))

    required_principles = {
        "kiss",
        "yagni",
        "single_source_of_truth",
        "contracts",
        "fail_safe",
        "security",
        "observability",
        "reversibility",
    }
    missing_principles = sorted(
        key for key in required_principles if not str(principles.get(key, "")).strip()
    )
    if acceptance and non_goals and not missing_principles:
        checks["principles_declared"] = "pass"
    else:
        errors.append(
            err(
                "INCOMPLETE_PRINCIPLE_PLAN",
                "acceptance_criteria, non_goals, and all principle decisions are required",
                missing_principles=missing_principles,
            )
        )

    invalid_owners = [path for path, owner in ownership.items() if not str(owner).strip()]
    missing_owners = [path for path in allowed if path not in ownership]
    if ownership and not invalid_owners and not missing_owners:
        checks["ownership_declared"] = "pass"
    else:
        errors.append(
            err(
                "MISSING_OWNERSHIP",
                "task_plan.ownership must assign one owner to every writable path",
                invalid_paths=invalid_owners,
                missing_paths=missing_owners,
            )
        )

    planner = str(discussion.get("planner", "")).strip()
    challenger = str(discussion.get("challenger", "")).strip()
    approved_by = str(discussion.get("approved_by", "")).strip()
    decision = str(discussion.get("decision", "")).strip()
    open_blockers = discussion.get("open_blockers") or []
    if (
        planner
        and challenger
        and planner != challenger
        and approved_by
        and decision == "approved"
        and not open_blockers
    ):
        checks["discussion_approved"] = "pass"
    else:
        errors.append(
            err(
                "PLAN_NOT_APPROVED",
                "An independent challenge and explicit approval are required before implementation",
            )
        )

    has_strict_validator = any("--check-diff" in c and "--provider" in c for c in req_cmds)
    if not has_strict_validator and data.get("task", {}).get("type") == "integration":
        warnings.append(err("MISSING_STRICT_VALIDATOR", "No strict validator command with --check-diff --provider"))

    status = "pass" if not errors else "fail"
    return {
        "status": status,
        "errors": errors,
        "warnings": warnings,
        "checks": checks,
        "normalized_plan": {
            "allowed_paths": allowed,
            "out_of_scope_paths": out_scope,
            "required_tests": req_tests,
            "required_commands": req_cmds,
            "acceptance_criteria": acceptance,
            "non_goals": non_goals,
            "ownership": ownership,
            "principles": principles,
            "discussion": discussion,
        },
    }


def run_post_implementation(data: dict) -> dict:
    pre_result = run_pre_implementation(data)
    errors = list(pre_result.get("errors", []))
    checks = {
        "scope_guard": "fail",
        "required_test_presence": "fail",
        "provider_file_presence": "fail",
    }

    plan = data.get("task_plan") or {}
    impl = data.get("implementation")
    if not isinstance(impl, dict):
        errors.append(err("MISSING_IMPLEMENTATION", "implementation is required"))
        return {"status": "fail", "errors": errors, "checks": checks}

    changed_files = impl.get("changed_files") or []
    tests_covered = set(impl.get("tests_covered") or [])
    allowed = plan.get("allowed_paths") or []
    blocked = plan.get("out_of_scope_paths") or []
    required_tests = set(plan.get("required_tests") or [])

    if not changed_files:
        errors.append(err("NO_CHANGED_FILES", "implementation.changed_files must not be empty"))

    out_of_scope = []
    blocked_hits = []
    for f in changed_files:
        if blocked and _is_in_blocked(f, blocked):
            blocked_hits.append(f)
            continue
        if allowed and not _is_in_allowed(f, allowed):
            out_of_scope.append(f)

    if not out_of_scope and not blocked_hits:
        checks["scope_guard"] = "pass"
    else:
        for f in blocked_hits:
            errors.append(err("BLOCKED_PATH_CHANGED", "Changed file is explicitly out of scope", file=f))
        for f in out_of_scope:
            errors.append(err("OUT_OF_SCOPE_FILE", "Changed file is outside allowed scope", file=f))

    missing_tests = sorted(list(required_tests - tests_covered))
    if not missing_tests:
        checks["required_test_presence"] = "pass"
    else:
        errors.append(err("MISSING_TEST_CATEGORIES", "Required test categories missing", missing=missing_tests))

    expected_provider_config = _required_provider_config_path(data, plan)
    if expected_provider_config:
        if any(f == expected_provider_config for f in changed_files):
            checks["provider_file_presence"] = "pass"
        else:
            errors.append(
                err(
                    "MISSING_PROVIDER_CONFIG",
                    "Expected provider config file was not changed",
                    expected=expected_provider_config,
                )
            )
    else:
        checks["provider_file_presence"] = "pass"

    return {"status": "pass" if not errors else "fail", "errors": errors, "checks": checks}


def run_post_verification(data: dict) -> dict:
    implementation_result = run_post_implementation(data)
    errors = list(implementation_result.get("errors", []))
    issues = []

    plan = data.get("task_plan") or {}
    verification = data.get("verification")
    if not isinstance(verification, dict):
        errors.append(err("MISSING_VERIFICATION", "verification is required"))
        return {
            "status": "fail",
            "final_decision": "block",
            "issues": errors,
            "reroute_payload": {},
        }

    commands = verification.get("commands") or []
    coverage = verification.get("coverage") or {}
    required_tests = set(plan.get("required_tests") or [])
    passed_categories = set(coverage.get("required_categories_passed") or [])

    failed_cmds = [
        command
        for command in commands
        if not isinstance(command, dict) or int(command.get("exit_code", 1)) != 0
    ]
    for c in failed_cmds:
        command = c.get("cmd", "") if isinstance(c, dict) else ""
        issues.append(err("TEST_FAIL", "Verification command failed", command=command))

    successful_commands = {
        str(command.get("cmd", "")).strip()
        for command in commands
        if isinstance(command, dict) and int(command.get("exit_code", 1)) == 0
    }
    missing_commands = [
        command for command in plan.get("required_commands") or [] if command not in successful_commands
    ]
    if missing_commands:
        issues.append(
            err(
                "REQUIRED_COMMANDS_MISSING",
                "Required verification commands were not successfully executed",
                missing=missing_commands,
            )
        )

    if coverage.get("unit_tests_present") is False:
        issues.append(err("UNIT_TESTS_MISSING", "coverage.unit_tests_present is false"))

    missing_cats = sorted(list(required_tests - passed_categories))
    if missing_cats:
        issues.append(err("REQUIRED_CATEGORIES_FAILED", "Missing required test categories", missing=missing_cats))

    if errors:
        issues = errors + issues

    if issues:
        return {
            "status": "fail",
            "final_decision": "reroute_to_implementer",
            "issues": issues,
            "reroute_payload": {
                "target_agent": "implementer",
                "fix_only": True,
                "todo": [i.get("message", "") for i in issues],
            },
        }

    return {
        "status": "pass",
        "final_decision": "ready_for_review",
        "issues": [],
        "reroute_payload": {},
    }


def run_post_review(data: dict) -> dict:
    verification_result = run_post_verification(data)
    errors = list(verification_result.get("issues", []))
    review = data.get("review")
    if not isinstance(review, dict):
        errors.append(err("MISSING_REVIEW", "review is required"))
        return {
            "status": "fail",
            "final_decision": "block",
            "issues": errors,
            "reroute_payload": {},
        }

    reviewer = str((data.get("meta") or {}).get("triggered_by", "")).strip()
    claimed_reviewer = str(review.get("reviewer", "")).strip()
    implementation_owners = {
        str(owner).strip()
        for owner in (data.get("task_plan") or {}).get("ownership", {}).values()
        if str(owner).strip()
    }
    findings = review.get("findings") or []
    decision = str(review.get("decision", "")).strip()
    evidence = review.get("evidence") or {}
    orchestration = data.get("orchestration") or {}

    if not reviewer:
        errors.append(err("MISSING_REVIEWER", "meta.triggered_by is required"))
    elif claimed_reviewer != reviewer:
        errors.append(
            err(
                "REVIEWER_IDENTITY_MISMATCH",
                "review.reviewer must match meta.triggered_by",
                reviewer=claimed_reviewer,
                triggered_by=reviewer,
            )
        )
    elif reviewer in implementation_owners:
        errors.append(
            err(
                "REVIEWER_NOT_INDEPENDENT",
                "The code reviewer must not be an implementation owner",
                reviewer=reviewer,
            )
        )

    if not implementation_owners:
        errors.append(err("MISSING_IMPLEMENTATION_OWNERS", "task_plan.ownership must not be empty"))

    required_evidence = {"plan", "implementation", "verification", "diff"}
    missing_evidence = sorted(
        key for key in required_evidence if not str(evidence.get(key, "")).strip()
    )
    if missing_evidence:
        errors.append(
            err(
                "MISSING_REVIEW_EVIDENCE",
                "Review must reference plan, implementation, verification, and diff evidence",
                missing=missing_evidence,
            )
        )

    required_orchestration = {"run_id", "task_id", "dispatch_id"}
    missing_orchestration = sorted(
        key for key in required_orchestration if not str(orchestration.get(key, "")).strip()
    )
    if missing_orchestration:
        errors.append(
            err(
                "MISSING_ORCA_PROVENANCE",
                "Orca run, task, and dispatch provenance are required for final review",
                missing=missing_orchestration,
            )
        )
    elif str(orchestration.get("run_id", "")).strip() != str(data.get("run_id", "")).strip():
        errors.append(
            err(
                "ORCA_RUN_MISMATCH",
                "Orca provenance run_id must match the coordinator-attested run_id",
                envelope_run_id=data.get("run_id", ""),
                orchestration_run_id=orchestration.get("run_id", ""),
            )
        )

    blocking_findings = [
        finding
        for finding in findings
        if isinstance(finding, dict)
        and str(finding.get("severity", "")).strip() in {"critical", "major"}
        and str(finding.get("status", "")).strip() != "resolved"
    ]
    invalid_findings = [finding for finding in findings if not isinstance(finding, dict)]
    if invalid_findings:
        errors.append(err("INVALID_REVIEW_FINDING", "Every review finding must be an object"))
    if blocking_findings:
        errors.append(
            err(
                "BLOCKING_REVIEW_FINDINGS",
                "Critical or major review findings remain unresolved",
                count=len(blocking_findings),
            )
        )

    if decision != "approved":
        errors.append(err("REVIEW_NOT_APPROVED", "review.decision must be approved"))

    if errors:
        return {
            "status": "fail",
            "final_decision": "reroute_to_implementer",
            "issues": errors,
            "reroute_payload": {
                "target_agent": "implementer",
                "fix_only": True,
                "todo": [issue.get("message", "") for issue in errors],
            },
        }

    return {
        "status": "pass",
        "final_decision": "ready_to_ship",
        "issues": [],
        "reroute_payload": {},
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run orchestrator stage hooks.")
    parser.add_argument(
        "--stage",
        required=True,
        choices=["pre-implementation", "post-implementation", "post-verification", "post-review"],
    )
    parser.add_argument("--input", required=True, help="Path to JSON input envelope")
    parser.add_argument("--output", required=True, help="Path to JSON output")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        input_path = Path(args.input)
        data = json.loads(input_path.read_text(encoding="utf-8"))
    except Exception as exc:  # noqa: BLE001
        print(f"Invalid input JSON: {exc}", file=sys.stderr)
        return 2

    try:
        if args.stage == "pre-implementation":
            runner = run_pre_implementation
        elif args.stage == "post-implementation":
            runner = run_post_implementation
        elif args.stage == "post-verification":
            runner = run_post_verification
        else:
            runner = run_post_review

        schema_errors = validate_stage_schema(data, args.stage)
        if schema_errors:
            result = {
                "status": "fail",
                "final_decision": "block",
                "issues": schema_errors,
                "reroute_payload": {},
            }
        else:
            result = runner(data)

        output_path = Path(args.output)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
        return 0
    except Exception as exc:  # noqa: BLE001
        print(f"Hook execution error: {exc}", file=sys.stderr)
        return 3


if __name__ == "__main__":
    raise SystemExit(main())
