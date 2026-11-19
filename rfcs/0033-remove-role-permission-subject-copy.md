# RFC 0033: Remove Role-Permission Subject Copy

- Status: Superseded
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-08
- Supersedes: RFC 0032

Superseded by [RFC 0034](0034-paginate-permission-catalog.md).

## Context

`role_permissions.subject_type` copied the subject from its role definition so token-eligibility
constraints could validate grants. RFC 0032 removed token eligibility, leaving the copied subject
without an independent invariant. The repository owner approved removing this redundant column.

## Decision

Remove `subject_type` from `role_permissions`, its entity, service writes, seed data, constraints,
verification, and tests. Keep `subject_type` on `role_definitions`, where it determines whether the
role can be assigned to a user or project token.

Keep `scope_type` on `role_permissions`. Its composite foreign keys ensure that the role and
permission share the same organization or project boundary. Change the role reference constraint
from `(id, scope_type, subject_type)` to `(id, scope_type)`.

All other RFC 0032 and inherited access-management decisions remain unchanged.

## Contracts and Ownership

- `role_definitions.subject_type` remains the single source of truth for the assignable principal.
- `role_permissions.scope_type` remains the database-enforced scope compatibility value.
- No public API or protobuf field changes.

## Alternatives Rejected

- Keep the copied subject: rejected because it no longer enforces a distinct rule.
- Remove scope as well: rejected because scope still enforces role-to-permission compatibility in
  PostgreSQL without triggers.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The unreleased role-permission schema loses one redundant column. Public contracts are unchanged. |
| Failure and recovery | N/A. Mutation and error behavior remain unchanged. |
| Security and privacy | Role subject validation remains on the authoritative role definition. |
| Data and migration | Migration 000014 and its seed omit the copied subject column. |
| Concurrency and resources | N/A. No transaction or lock behavior changes. |
| Operations and observability | N/A. Audit and logging behavior remain unchanged. |

## Rollout and Rollback

Ship the revised migration before the unreleased catalog is deployed. Rollback continues dropping
the role catalog tables. No subject data conversion is required.

## Verification

- Entity and schema tests prove `RolePermission.SubjectType` is absent.
- Migration verification proves scope mismatch rejection and valid project-token grants.
- Run `go test ./api/web-api/internal/entity ./api/web-api/internal/service/accessmanagement`.
- Run `bash bin/verify-role-catalog-migration.sh` and the required scoped `just agent-finalize` command.

## Review and Approval

- Challenger: Repository owner
- Outcome: Approved
- Resolved findings: Remove the redundant subject copy while retaining database scope enforcement.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-08
