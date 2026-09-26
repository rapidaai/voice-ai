---
name: responding-to-review
description: Evaluate and resolve human or automated review comments with evidence. Use when a pull request or code review has findings that must be verified, fixed, refuted, or deferred.
---

# Responding To Review

## Mission

Disposition every review comment against the exact candidate, make one coherent correction pass, and reply with concise evidence.

## Review each comment

For every comment:

1. Restate the claimed defect or requested outcome precisely.
2. Verify the claim against the current code, contract, and reviewed revision.
3. Classify it as Valid, Invalid, Duplicate, or Out-of-scope.
4. Assign Critical, Major, Minor, or Note using `DEVELOPMENT_PROCESS.md`.
5. For a valid finding, identify the owning layer and smallest safe correction.
6. For an invalid finding, record the code, test, or command result that disproves it.
7. For out-of-scope work, name the follow-up destination and explain why it is separable.

Do not accept a suggestion merely because it came from a reviewer or tool. Do not dismiss it without evidence.

## Correction pass

- Group compatible findings into one scoped pass.
- Return fixes to the original implementation owner.
- Add a regression test for corrected behavior and prove its pre-fix failure when reproducible.
- Run focused checks and repeat any broader verification invalidated by the corrections.
- Request targeted re-review for corrected Critical or Major findings.

## Replies

Keep each response short and factual:

- fixed: state the change and verification
- invalid: cite the evidence that refutes the claim
- duplicate: link to the owning finding
- out-of-scope: name the follow-up destination

Do not resolve a thread until the reply and code agree.

## Closeout

Report counts by disposition and severity, exact commands rerun, remaining blockers, and the final reviewed boundary.

## Governed lifecycle

Governed implementation and review correction cycles are limited to two. A correction that changes accepted scope or RFC bytes returns to challenge and exact-digest confirmation. This skill must not self-approve corrections.
