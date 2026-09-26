---
name: designing-change
description: Resolve a non-trivial implementation or architecture decision before coding. Use when ownership, contracts, failure behavior, rollout, or competing approaches require an explicit decision.
---

# Designing Change

## Mission

Choose the smallest design that satisfies verified requirements and make its ownership, contracts, failure behavior, and verification strategy explicit.

## When to use

Use this skill when the approach is genuinely uncertain, the change crosses an ownership boundary, or a wrong decision would be expensive to reverse. Skip it for obvious Fast work and direct Standard fixes with an established local pattern.

## Inputs

- `change-analysis` evidence
- observable acceptance criteria and non-goals
- existing architecture and local patterns
- compatibility and operational constraints
- unresolved questions that materially change the result

## Decision sequence

1. State the problem without embedding a preferred solution.
2. Separate verified facts, assumptions, and unknowns.
3. Identify the current owner of each behavior and resource.
4. Describe the smallest viable option plus credible alternatives.
5. Evaluate each option for simplicity, ownership, contracts, failure recovery, observability, testing, rollout, and rollback.
6. Reject speculative flexibility and duplicate sources of truth.
7. Select an option and record why the rejected options are inferior for the current requirement.
8. Convert the decision into exact acceptance criteria, paths, tests, and operational steps.

## Decision record

Capture:

- context and constraints
- chosen option
- rejected alternatives and concrete tradeoffs
- ownership and contract changes
- failure, timeout, partial-success, and cleanup behavior
- compatibility and migration behavior
- verification strategy
- rollout, disablement, and rollback
- unresolved decisions requiring user or owner input

## Boundaries

This skill decides the approach but does not implement it. Do not manufacture additional options after one solution clearly satisfies the requirement and repository principles.

## Governed lifecycle

For Governed work, the decision feeds the reserved RFC and must pass independent challenge plus exact-digest confirmation. This skill cannot approve its own decision or authorize implementation.
