# Claude Custom Subagents

This folder defines project-local Claude custom subagents.

Available subagents:

- `task-planner`
- `plan-challenger`
- `ui-implementation`
- `ui-unit-tests`
- `backend-implementation`
- `backend-unit-tests`
- `code-reviewer`

Expected flow:

1. Use `development-lifecycle` to select Fast, Standard, or Governed work.
2. Use `task-planner` when Standard or Governed work benefits from delegated investigation.
3. Require `plan-challenger` and RFC confirmation only for Governed work.
4. Delegate implementation or tests only when parallelism materially helps, with disjoint ownership.
5. Run scoped finalization explicitly. Completion hooks never run tests.
6. Validate integration scope using the relevant domain skill validator.
7. Use `code-reviewer` against a fixed candidate boundary when the tier requires review.

Required roles must be identifiable. The code reviewer must not be an implementation owner and must not edit findings directly.
