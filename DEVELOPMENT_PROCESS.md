# Development Process

This repository uses a principle-driven, multi-agent lifecycle designed for Orca worktrees.

## Non-negotiable Gates

The required sequence is:

`understand -> plan -> discuss -> approve -> implement -> verify -> review -> ship`

No agent may collapse planning, implementation, verification, and final code review into one self-approved action for a non-trivial change.

## Roles

### Coordinator

- Owns the Orca Run, task breakdown, dependencies, and decision gates.
- Ensures each worker has an explicit and disjoint write scope.
- Preserves plan, implementation, verification, and review evidence.
- Does not mark a task ready while a blocking finding remains open.

### Task Planner

- Investigates the system before proposing changes.
- Produces acceptance criteria, scope, ownership, risks, test commands, and rollback strategy.
- Identifies assumptions and questions instead of presenting guesses as facts.

### Plan Challenger

- Is independent from the planner.
- Looks for a smaller solution, missing contracts, hidden coupling, unsafe rollout, and unverifiable acceptance criteria.
- Recommends `approve`, `revise`, or `block` with reasons.

### Implementer

- Owns only assigned files or modules.
- Implements the approved plan without silently expanding scope.
- Adds or updates behavior-focused tests with the production change.

### Verifier

- Runs the declared commands and checks acceptance criteria independently.
- Records exact commands, exit codes, coverage categories, and environmental limitations.
- Routes failures back to the implementer.

### Code Reviewer

- Must not be an implementation author for the reviewed change.
- Uses read-only tools and reviews the complete diff after verification.
- Never fixes findings directly; the implementation owner makes corrections.
- Reviews correctness, simplicity, ownership, contracts, compatibility, concurrency, resource lifecycle, failure behavior, security, observability, tests, and rollback safety.
- Blocks shipping for unresolved critical or major findings.

## Task Contract

Before implementation, the approved plan must contain:

- Problem statement and verified system context.
- Acceptance criteria and explicit non-goals.
- Allowed and out-of-scope paths.
- One owner for every writable file or module.
- KISS and YAGNI decisions, including rejected alternatives.
- Contract, compatibility, security, operational, and data risks.
- Required test categories and exact verification commands.
- Rollback, disablement, or migration strategy.
- Plan challenge outcome and explicit approval.

The approved plan is stored as a separate JSON artifact. Its SHA-256 and a coordinator-generated HMAC are recorded in the cumulative lifecycle envelope, and every gate verifies that the envelope plan still matches that artifact. The HMAC signs `<run_id>:<plan_sha256>` using `DEVELOPMENT_GATE_KEY`. That key belongs only to the coordinator or CI gate runner and must not be exposed to implementation or review workers.

After approving the plan, the coordinator creates the attestation:

```bash
DEVELOPMENT_GATE_KEY="..." make sign-approved-plan RUN_ID="<orca-run-id>" PLAN="<approved-plan.json>"
```

Workers produce evidence envelopes; the coordinator or CI gate runner executes lifecycle hooks with the key.

Use `.codex/orchestrator/templates/task-plan.md` or the equivalent `.claude` template.

## Orca Workflow

Start a governed Run from an Orca-managed terminal:

```bash
make orca-development-run OBJECTIVE="describe the desired outcome" AGENT=codex
```

The command creates the Orca Run, dependent planning, challenge, implementation, verification, and code-review tasks, an approval gate blocking implementation, and a supervised planner worker. Orca must be running with orchestration enabled.

Generate the development panel from a lifecycle envelope:

```bash
make orca-panel PANEL_INPUT="path/to/lifecycle-input.json"
```

Open it as a browser tab in the active Orca worktree:

```bash
make orca-panel-open PANEL_INPUT="path/to/lifecycle-input.json"
```

The panel displays stage readiness, principle decisions, ownership, exact verification commands, review findings, final-gate issues, and Orca Run/Task/Dispatch provenance. With `DEVELOPMENT_GATE_KEY` available to the coordinator, it also executes the final gate and shows trusted ship readiness.

1. Create one Orca Run for the objective.
2. Create investigation and planning tasks before implementation tasks.
3. Dispatch the planner and plan challenger independently.
4. Resolve discussion through messages and an explicit decision gate.
5. Create implementation tasks with disjoint file ownership.
6. Dispatch verification after implementation settles.
7. Dispatch the code reviewer only after verification passes.
8. Route review findings to the original implementation owner.
9. Re-run affected verification after fixes.
10. Ship only after the final review decision is `approved`.

The final review envelope must include the Orca Run, review Task, and review Dispatch identifiers. Reviewer identity comes from the execution metadata, while implementation ownership comes from the coordinator-attested approved-plan artifact.

Use Orca worktree comments to record decisions and keep each implementation task isolated in its own worktree. Parallel implementation is allowed only when write sets do not overlap.

## Review Severity

- Critical: security, data loss, tenant isolation, production outage, or fundamentally incorrect behavior. Blocks shipping.
- Major: broken acceptance criteria, compatibility regression, unsafe ownership, missing failure handling, or material test gap. Blocks shipping.
- Minor: maintainability or clarity issue that does not invalidate behavior. May become a tracked follow-up.
- Note: optional improvement or question. Does not block shipping.

## Required Evidence

A completed change retains:

- Approved plan and challenge decision.
- Implementation summary and changed-file list.
- Verification commands with results.
- Independent code-review report.
- Resolution for every critical and major finding.
- PR summary, operational impact, and rollback notes.
