# RFC NNNN: Title

- Status: Draft
- Owner: Team or individual
- Created: YYYY-MM-DD
- Updated: YYYY-MM-DD
- Reviewers: Names or roles

## Summary

Describe the decision and intended outcome in a few sentences.

## Context

Explain the current behavior, verified evidence, constraints, and why a change is needed.
Link to relevant code, incidents, issues, metrics, or prior RFCs.

## Goals

- State measurable outcomes this RFC must achieve.

## Non-Goals

- State adjacent work that is explicitly excluded.

## Scope and Ownership

### Allowed Paths

- `path/owned/by-this-change/` — owner

### Out-of-Scope Paths

- `path/not/owned/`

## Proposed Design

Describe the smallest complete design, its state transitions, data flow, and ownership.

## Contracts and Compatibility

- Public API, protocol, schema, configuration, and dependency contracts.
- Backward/forward compatibility guarantees and intentional breaking changes.

## Failure and Recovery

- Invalid input, timeout, cancellation, partial failure, retries, cleanup, and idempotency.
- Conditions that block rollout or require operator action.

## Security and Privacy

- Authentication, authorization, tenant isolation, secrets, sensitive data, and least privilege.

## Observability

- Logs, metrics, traces, alerts, and operator diagnostics needed for the changed behavior.

## Data and Migration

- Schema/data changes, ordering, backfill, resumability, rollback limits, and ownership.
- Write `None` when no persistent-data change exists.

## Rollout

- Deployment order, feature flags, compatibility window, validation, and stop conditions.

## Rollback

- Exact rollback or disablement steps and any irreversible effects.

## Alternatives Considered

- Describe simpler alternatives and why they were rejected.

## Testing and Verification

- Required test categories.
- Exact commands and expected evidence.
- Environmental limitations or manual checks.

## Acceptance Criteria

- [ ] Each criterion is observable and independently verifiable.

## Open Questions

- Record unresolved decisions and their owner. Write `None` before acceptance.

## Challenge Resolution

- Summarize blocking findings and how each was resolved.
- Limit revision cycles to two; unresolved issues after that return the RFC to `Draft` or `Rejected`.

## Artifact Index

Store every JSON plan, amendment, challenge, confirmation, review, and operational-readiness
artifact under `rfcs/NNNN-short-name/jsons/`. Use stable descriptive names such as:

- `jsons/plan.json`
- `jsons/confirmation.json`
- `jsons/amendment-01-plan.json`
- `jsons/amendment-01-confirmation.json`
- `jsons/operational-readiness.json`

List the artifacts used for this RFC here with their purpose and status.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| YYYY-MM-DD | Initial proposal | Owner | `jsons/plan.json` |
