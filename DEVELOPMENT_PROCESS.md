# Development Process

This repository uses a risk-tiered development lifecycle designed for Orca worktrees.

## Workflow Tiers

Choose the lightest safe workflow before creating tasks or agents.

### Fast

Use for documentation, comments, formatting, generated-file refreshes, test-only work,
and isolated low-risk fixes with no public contract, schema, security, concurrency,
deployment, or data impact.

`understand -> implement -> targeted verification`

### Standard

This is the default for ordinary feature and bug-fix work contained within one service or
component.

`understand -> concise plan -> implement -> targeted verification -> review`

The plan can live in the task or PR. An RFC, plan signature, and Orca decision gate are
not required. Production behavior changes receive independent review before merge.

### Governed

Use only for public API/protocol changes, authentication or authorization, schema or data
migrations, cross-service contracts, irreversible operations, high-risk rollouts, or an
explicitly requested RFC.

`understand -> plan -> draft RFC -> challenge -> confirm -> implement -> verify -> review -> ship`

No agent may collapse planning, implementation, verification, and final code review into
one self-approved action in the Governed tier.

If classification is uncertain, use Standard and document the uncertainty. Escalate to
Governed only when a listed trigger is discovered.

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

- Follows the canonical engineering principles and code writing rules in `AGENTS.md`.
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

## Governed Task Contract

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
- Reserved RFC path and the exact accepted RFC SHA-256.
- RFC confirmation gate receipt with Run, Task, Gate, question, resolution, and timestamps.

Create RFCs from `rfcs/TEMPLATE.md`. Keep the RFC Markdown at
`rfcs/NNNN-short-name.md` and store every related JSON artifact under
`rfcs/NNNN-short-name/jsons/`. The standard initial names are `plan.json`,
`challenge.json`, and `confirmation.json`; use `amendment-NN-*.json` for later changes.

The RFC author drafts only the coordinator-reserved path. The challenger reviews the
exact plan and RFC bytes. After all findings are resolved, the final challenged bytes
must already contain the sole metadata line `- Status: Accepted`. The coordinator then
creates a gate whose question includes the RFC path and SHA-256 without editing the RFC.
Implementation may start only after that gate resolves to `approved`. Any
subsequent RFC byte change requires a new challenge and confirmation.

Plan/RFC revision and implementation/review correction loops are limited to two rounds.
After two unsuccessful rounds, mark the task blocked and escalate the unresolved decision
to the user or technical owner. Do not silently create another worker attempt.

The approved plan is stored as a separate JSON artifact. Its SHA-256 and a coordinator-generated HMAC are recorded in the cumulative lifecycle envelope, and every gate verifies that the envelope plan still matches that artifact. The HMAC signs `<run_id>:<plan_sha256>` using `DEVELOPMENT_GATE_KEY`. That key belongs only to the coordinator or CI gate runner and must not be exposed to implementation or review workers.

After approving the plan, the coordinator creates the attestation:

```bash
DEVELOPMENT_GATE_KEY="..." just sign-approved-plan \
  "<orca-run-id>" "rfcs/NNNN-short-name/jsons/plan.json"
```

Workers produce evidence envelopes; the coordinator or CI gate runner executes lifecycle hooks with the key.

Use `.codex/orchestrator/templates/task-plan.md` or the equivalent `.claude` template.

## Orca Governed Workflow

Use Orca orchestration only for Governed work or when supervised coordination is explicitly
needed. Fast and Standard work should use the normal agent flow without creating a Run DAG.

Start a Governed Run from an Orca-managed terminal:

```bash
just orca-development-run "describe the desired outcome" \
  "rfcs/NNNN-short-name.md" codex
```

If a Run is abandoned before its confirmation gate is created, release its reservation:

```bash
just orca-rfc-release "rfcs/NNNN-short-name.md"
```

The command creates the Orca Run and dependent planning, RFC-authoring, and challenge
tasks, then starts only the planner. It intentionally does not create or start an
implementation task.

After challenge approval, the coordinator creates the exact-digest gate on a dedicated
confirmation task. No implementation task is created yet:

```bash
just orca-confirm-rfc-create "<run>" "<challenge-task>" \
  "rfcs/NNNN-short-name/jsons/challenge.json" "rfcs/NNNN-short-name.md"
```

After the gate resolves, collect the authoritative receipt before starting a worker:

```bash
just orca-confirm-rfc-collect "<run>" "<confirmation-task>" "<challenge-task>" \
  "rfcs/NNNN-short-name/jsons/challenge.json" "<gate>" "rfcs/NNNN-short-name.md"
```

Generate the development panel from a lifecycle envelope:

```bash
just orca-panel "path/to/lifecycle-input.json"
```

Open it as a browser tab in the active Orca worktree:

```bash
just orca-panel-open "path/to/lifecycle-input.json"
```

The panel displays stage readiness, principle decisions, ownership, exact verification commands, review findings, final-gate issues, and Orca Run/Task/Dispatch provenance. With `DEVELOPMENT_GATE_KEY` available to the coordinator, it also executes the final gate and shows trusted ship readiness.

1. Create one Orca Run for the objective.
2. Reserve the RFC number and create planning, RFC-authoring, and challenge tasks.
3. Dispatch planner, RFC author, and challenger in dependency order.
4. Resolve challenge findings before marking the RFC `Accepted`.
5. Create an exact-digest confirmation gate on a dedicated confirmation task.
6. Collect the approved receipt and preserve the accepted RFC baseline.
7. Start implementation tasks with disjoint file ownership.
8. Dispatch verification after implementation settles.
9. Dispatch the code reviewer only after verification passes.
10. Route findings to the original implementation owner and re-verify fixes.
11. Ship only after the final review decision is `approved`.

For coordinator waits, process the complete delivery, acknowledge its delivery ID, and
release or deliberately reuse each settled worker before waiting again. Never retry a task
because a wait window timed out; inspect worker state first. Do not exceed two failed
attempts or correction rounds without escalating.

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
- Accepted RFC and exact-digest confirmation receipt.
- Implementation summary and changed-file list.
- Verification commands with results.
- Independent code-review report.
- Resolution for every critical and major finding.
- PR summary, operational impact, and rollback notes.
