---
name: development-lifecycle
description: Route repository changes through the Fast, Standard, or Governed lifecycle. Use when implementing a feature, fixing a bug, changing behavior, or deciding which planning, verification, and review steps are required.
---

# Development Lifecycle

## Mission

Choose the lightest safe workflow, establish a testable change contract, and route the work through the skills that own each phase.

## Start here

1. Read `AGENTS.md` and classify the request as Fast, Standard, or Governed before creating workers or editing files.
2. Record the requested outcome, acceptance criteria, allowed paths, explicit non-goals, risks, and exact verification commands.
3. Use `change-analysis` when current behavior, ownership, consumers, or blast radius is not already proven.
4. Use `designing-change` when the approach has material tradeoffs or crosses an ownership boundary.
5. For voice integrations, use `system-understanding` and the matching domain skill before `developing-change`.
6. Use `debugging` for unexplained failures and `writing-documentation` for documentation owed by the change.
7. Use `reviewing-change` after verification when the selected tier requires review, then `responding-to-review` for any findings.
8. Use `preparing-delivery` only when the user asks for a commit, push, or pull request.
9. Read `agent-tooling-manifest.json` when capability availability or coverage ownership affects the plan.

## Tier behavior

### Fast

- Keep the contract brief and proceed directly from understanding to implementation.
- Run the narrowest relevant checks.
- Do not add planning or review ceremony unless the discovered risk changes the classification.

### Standard

- Write a concise plan before editing.
- Keep one owner for each writable path and use parallel workers only for disjoint scopes.
- Add behavior-focused tests and run explicit finalization.
- Obtain independent review before merging production behavior changes.

### Governed

- Follow the complete workflow in `DEVELOPMENT_PROCESS.md`.
- Reserve the RFC path and complete challenge plus exact-digest confirmation before implementation.
- Preserve the approved scope, verification evidence, reviewer independence, and correction-round limits.

## Reclassification

Reclassify immediately when discovery reveals a public contract, authentication, authorization, schema, migration, cross-service, irreversible, or high-risk rollout concern. Never silently continue under a lighter tier.

## Change contract

Before editing, make these facts visible:

- outcome and observable acceptance criteria
- allowed paths and out-of-scope paths
- current owner of each affected behavior
- compatibility, security, data, concurrency, and operational risks
- success, fallback, and regression coverage
- exact validation commands
- rollback or safe disablement when behavior changes

## Governed lifecycle

This skill routes the lifecycle but does not replace the Governed roles or gates. It must not approve its own plan, challenge, implementation, verification, or review evidence.

## Completion

Report the selected tier, changed files, verification results, review status, remaining risks, and whether delivery actions were requested. Evidence takes precedence over confidence.
