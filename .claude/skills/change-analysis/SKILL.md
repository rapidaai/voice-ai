---
name: change-analysis
description: Establish current behavior, ownership, affected consumers, and blast radius before planning or editing. Use when scoping a feature, assessing safety, tracing a regression, or deciding which services and contracts a change touches.
---

# Change Analysis

## Mission

Replace assumptions with code-grounded evidence about what the system does today and what the requested change can affect.

## Boundaries

This skill is read-only. It produces scope and evidence, not implementation.

## Analysis sequence

1. Identify the entry point and the owner of the behavior.
2. Trace the primary path from input through selection, state changes, side effects, and output.
3. Locate configuration sources, factories, interfaces, generated contracts, callers, and existing tests.
4. Search for consumers by symbol, route, packet, configuration key, event, or schema field.
5. Inspect recent history only when it explains intent or a compatibility constraint.
6. Compare the requested behavior with observable tests and documented contracts.

## Risk questions

Answer each independently with evidence:

- Does this change a public API, protocol, packet, event, or persisted representation?
- Does it affect authentication, authorization, tenant isolation, secrets, or trust boundaries?
- Does it change data ownership, migrations, cleanup, retries, concurrency, or resource lifetime?
- Does it cross a service, generated-code, deployment, or provider boundary?
- Can it fail partially, time out, or leave state that requires recovery?

Any affirmative Governed trigger must be handed back to `development-lifecycle` for reclassification.

## Integration work

After repository-level scope is known, use `system-understanding` for packet, factory, and UI configuration tracing, followed by the matching integration skill for strict write boundaries.

## Output contract

- Current behavior with `file:line` evidence.
- Primary owner and affected consumers.
- Inputs, state transitions, side effects, outputs, and failure paths.
- Contract and compatibility seams.
- Proposed allowed paths and explicit exclusions.
- Existing coverage and missing regression coverage.
- Risks, unknowns, and the command or decision that would settle each unknown.
- Recommended lifecycle tier.

## Governed lifecycle

For Governed work, this analysis becomes planning evidence. It cannot approve the plan or authorize implementation.
