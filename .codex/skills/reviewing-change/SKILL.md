---
name: reviewing-change
description: Perform an evidence-based review of a fixed change boundary. Use for self-review, independent code review, pre-merge review, or review of production behavior changes after verification.
---

# Reviewing Change

## Mission

Determine whether the exact candidate satisfies its acceptance criteria without introducing correctness, ownership, compatibility, safety, or verification defects.

## Independence and mutation

Review read-only. An independent reviewer must not be an author of the reviewed production change. Findings return to the original implementation owner for correction.

## Freeze the candidate

Before reviewing, record:

- base revision and merge base
- full head revision when committed
- working-tree status and untracked files
- exact changed-file list
- whether the boundary is a commit range or an uncommitted diff

If the candidate changes during review, discard stale conclusions and review the new boundary.

## Review order

1. Acceptance criteria and user-visible behavior.
2. Public and internal contracts, compatibility, and data ownership.
3. Authentication, authorization, tenant isolation, and secret handling.
4. Failure paths, cancellation, timeouts, retries, partial success, and cleanup.
5. Concurrency, resource lifetime, ordering, and shutdown.
6. Observability and operational recovery.
7. Test quality, including success, fallback, regression, and factory selection.
8. KISS, YAGNI, duplicate state, and unnecessary abstraction.

## Finding format

Every finding includes:

- severity: Critical, Major, Minor, or Note
- location: `file:line`
- violated contract or invariant
- concrete consequence and triggering input
- supporting evidence
- smallest safe remedy

Do not report speculative findings as defects. Try to refute every Critical or Major finding against the code and tests before retaining it.

## Finding ledger

Disposition each candidate finding as Valid, Invalid, Duplicate, or Out-of-scope. Invalid findings require refuting evidence. Out-of-scope findings name a concrete follow-up destination.

## Decision

- Block approval for unresolved Critical or Major findings.
- Report verification gaps separately from code defects.
- When no blocking findings remain, state the reviewed boundary, checks considered, residual risks, and decision.

## Executable review panel

When `.agent-reviewers.json` is configured, use `bin/agent-review --working-tree --risk standard`
for one independent reviewer. Use `--risk high` for high-risk changes; it requires two distinct
model families. External transports require explicit user authorization through `--allow-external`.
The filtered diff digest and combined panel are evidence, not permission to merge or ship.

## Governed lifecycle

Governed review must verify reviewer independence, accepted scope, confirmation evidence, successful declared commands, and resolution of all blocking findings. This skill must not edit the candidate or self-approve authored work.
