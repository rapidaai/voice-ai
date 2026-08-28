# Orchestrator Hooks (Codex)

This folder provides the contract and runner for Governed-tier subagent gates across four stages:

1. `pre-implementation`
2. `post-implementation`
3. `post-verification`
4. `post-review`

## Layout

- `schemas/envelope.schema.json`: shared run envelope shape
- `schemas/pre-implementation-input.schema.json`
- `schemas/post-implementation-input.schema.json`
- `schemas/post-verification-input.schema.json`
- `schemas/post-review-input.schema.json`
- `scripts/hook-run.py`: stage runner
- `examples/lifecycle-input.json`: one full evidence envelope used across all gates

## CLI

```bash
DEVELOPMENT_GATE_KEY="<coordinator-key>" python3 .codex/orchestrator/scripts/hook-run.py \
  --stage pre-implementation \
  --input .codex/orchestrator/examples/lifecycle-input.json \
  --output /tmp/hook-out.json
```

Valid stages:

- `pre-implementation`
- `post-implementation`
- `post-verification`
- `post-review`

Exit codes:

- `0`: hook executed (check `status` inside output json)
- `2`: invalid input arguments/json
- `3`: internal hook error

## Contract summary

- `pre-implementation` validates plan completeness and required test/command declarations.
- `pre-implementation` also requires principle decisions, explicit ownership, an independent plan challenge, and approval.
- `post-implementation` enforces file scope guard and test-category presence.
- `post-verification` enforces command success and routes the task to independent code review.
- `post-review` enforces reviewer independence, review evidence, and resolution of critical or major findings before shipping.

Each stage receives the same cumulative lifecycle envelope. Later stages re-run all earlier gates, so review cannot replace an approved plan, implementation report, or successful verification with a standalone boolean.

The plan is loaded from `artifacts.approved_plan_file`, verified against `artifacts.approved_plan_sha256`, validated with the coordinator-held `DEVELOPMENT_GATE_KEY` and `artifacts.approved_plan_hmac`, and compared with `task_plan`. Workers must not receive the coordinator key.

Templates:

- `templates/task-plan.md`
- `templates/code-review.md`

Create the corresponding Orca Run, task DAG, approval gate, and planner worker with:

```bash
just orca-development-run "describe the desired outcome" \
  "rfcs/NNNN-short-name.md" codex
```

Use this only when a Governed trigger from `DEVELOPMENT_PROCESS.md` applies. It creates
only the planner, RFC-author, and challenger tasks and starts the planner.
After the exact challenged RFC bytes contain `- Status: Accepted`, use
`just orca-confirm-rfc-create` to create a dedicated confirmation task and exact-digest gate. Use
`just orca-confirm-rfc-collect` to verify the resolved gate and write its receipt before
creating or starting implementation tasks.

All RFC-related JSON inputs and outputs must be direct children of
`rfcs/NNNN-short-name/jsons/`. The collect command writes `confirmation.json` there by
default.

## Notes

- This is intentionally conservative and can be extended with repo-specific checks.
- Path guard uses exact matches for file paths and prefix matches for paths ending in `/`.
