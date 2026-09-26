---
name: writing-documentation
description: Create or update repository documentation, runbooks, configuration references, and developer guidance. Use when behavior changes make documentation inaccurate or when a documentation task needs code-grounded commands and paths.
---

# Writing Documentation

## Mission

Keep documentation accurate, executable, concise, and owned by the same change that alters the documented behavior.

## Source of truth

Read the implementation, tests, configuration, and runnable commands before documenting them. Do not infer shipped behavior from a plan or copy stale examples.

## Writing sequence

1. Identify the audience and the decision or task the document must support.
2. Verify every path, command, option, default, and failure behavior against the repository.
3. Update the closest authoritative document instead of creating a competing explanation.
4. Separate current behavior from proposals, future work, and environmental limitations.
5. Include prerequisites, success checks, common failures, and recovery when writing operational guidance.
6. Keep examples free of credentials, private endpoints, and misleading placeholder output.
7. Run `just check-agent-docs` and `just agent-style-check` when agent instructions or skills change.
8. Add to `AGENT_LESSONS.md` only when a concrete recurring defect establishes a durable guard.

## Style

- Lead with the outcome or required action.
- Use repository terminology consistently.
- Prefer short sections and concrete commands.
- Do not restate code line by line.
- Do not use tooling attribution or claim checks that were not run.
- Follow the comment and prose rules in `AGENTS.md`.

## Generated material

Never hand-edit generated documentation. Update its source and run the established generator. If no generator exists, do not invent one without a current reuse need.

## Completion

Report the authoritative sources checked, documentation changed, commands validated, and anything that remains environment-dependent.

## Governed lifecycle

Documentation does not replace Governed evidence. RFC status, confirmation receipts, verification results, and review decisions must remain accurate and traceable. This skill must not self-approve those records.
