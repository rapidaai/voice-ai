# Codex Custom Subagents

This folder defines project-local Codex custom subagents.

Files use markdown with YAML frontmatter:

- `name`
- `description`
- `tools`

Available subagents:

- `task-planner`
- `plan-challenger`
- `ui-implementation`
- `ui-unit-tests`
- `backend-implementation`
- `backend-unit-tests`
- `code-reviewer`

Expected flow:

1. Delegate investigation and planning to `task-planner`.
2. Require `plan-challenger` to approve or revise the plan.
3. Delegate implementation to UI/backend implementation agents with disjoint ownership.
4. Delegate tests to UI/backend test agents.
5. Enforce stop-time checks using `.codex/hooks/*`.
6. Validate integration scope using the relevant skill strict validator.
7. Require `code-reviewer` to approve the verified complete diff.

The planner, challenger, implementer, verifier, and code reviewer must be identifiable. The code reviewer must not be an implementation owner and must not edit findings directly.
