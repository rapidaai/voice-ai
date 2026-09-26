---
name: developing-change
description: Implement an approved repository change with explicit ownership, minimal scope, test-first evidence, and scoped verification. Use after lifecycle routing and analysis have established the intended behavior and allowed paths.
---

# Developing Change

## Mission

Make the smallest complete change at the owning layer and leave a deterministic evidence trail for verification and review.

## Preconditions

- The lifecycle tier is known.
- Acceptance criteria, allowed paths, risks, and exact verification commands are recorded.
- Current behavior and ownership are understood.
- Governed work has an accepted RFC and approved exact-digest confirmation.
- The matching integration skill is active when changing a provider boundary.

## Ownership

Use one editor for each file and mutable behavior. Parallel work requires disjoint write scopes. Reviewers remain read-only and never repair their own findings.

## Implementation loop

For each observable behavior:

1. Add or update the focused test and confirm it fails for the expected reason.
2. Change the established owner with the smallest direct control flow.
3. Run the focused test.
4. Exercise the relevant fallback or failure path.
5. Remove accidental complexity and unrelated edits.
6. Continue only after the current increment is understood and green.

For documentation or tooling work where a pre-fix test is not meaningful, state the substitute validation before editing.

## Repository rules

- Follow `AGENTS.md` as the canonical code-writing contract.
- Do not hand-edit generated files.
- Do not cross a selected integration boundary without pausing for direction.
- Keep resource creation, cancellation, cleanup, and shutdown under one owner.
- Preserve compatibility unless the accepted scope explicitly changes it.
- Do not commit, push, create a PR, merge, deploy, or release unless explicitly requested.

## Verification handoff

Run the narrowest tests first, then the declared broader checks. Finish with:

```bash
just agent-finalize "comma,separated,changed,paths"
```

Record exact commands, exit results, environmental limitations, changed files, and any acceptance criterion not proven.

## Review handoff

Provide `reviewing-change` with the intent, acceptance criteria, base and head or working-tree boundary, changed files, verification evidence, known risks, and explicit exclusions.

## Governed lifecycle

Implement only the confirmed Governed plan. Scope changes return to the coordinator, challenger, and confirmation gate. This skill must not verify or approve its own work.
