# RFC 0031: Remove Role Reduction Evidence

- Status: Superseded
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-08
- Supersedes: RFC 0030

Superseded by [RFC 0032](0032-remove-token-permission-eligibility.md).

## Context

RFC 0029 introduced persisted reduction evidence and transient audit-actor display-name fields.
Before release, the repository owner determined that these fields are not required and directed
their complete removal. RFC 0030 changed only the response envelope and otherwise inherited the
RFC 0029 behavior.

## Decision

Remove reduction reason, correlation, and assurance columns from `role_definitions` and
`role_permissions`. Remove the corresponding entity fields, database constraints, persistence,
exact replay queries, migration verification, and tests. Role and role-permission archival keeps
the existing status, update timestamp, update actor type, and update actor ID audit fields.

Remove the transient created-actor and updated-actor display-name fields and their user lookup.
Audit actors in API responses contain only their persisted type and ID.

The existing request reason validation and response correlation identifier remain transport
contracts, but they are not persisted. Consequently, retrying an already-completed permission
reduction returns `FailedPrecondition`, and retrying an already-completed role archive returns
`NotFound`. All other RFC 0030 and inherited role-management decisions remain unchanged.

## Contracts and Ownership

- Migration 000015 owns only database-generated role-definition and role-permission IDs.
- Existing audit entity fields own archival actor and timestamp evidence.
- The access-management service owns one-attempt reduction and archival behavior without replay
  evidence.
- The API continues returning audit actor identity without a display name.

## Alternatives Rejected

- Keep nullable evidence columns: rejected by the repository owner because the fields are not
  required.
- Keep transient display-name fields: rejected because audit actor type and ID are sufficient.
- Add another audit table: rejected because no persisted reduction evidence is required.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The API no longer guarantees idempotent mutation replay or actor display names. |
| Failure and recovery | Ambiguous mutation results must be reconciled with `GetRole` or `ListRoles`; retries may fail after a successful commit. |
| Security and privacy | Less user metadata is loaded and returned. Actor type and ID remain auditable. |
| Data and migration | Migration 000015 creates sequences only. No evidence columns are added or removed. |
| Concurrency and resources | Fewer transactional queries are required; lock ordering remains unchanged. |
| Operations and observability | Request correlation remains in transport responses and internal error logs but is not stored with archived rows. |

## Rollout and Rollback

Ship the revised migration and service before migration 000015 is applied anywhere. Rollback uses
the unchanged sequence down migration. No evidence data exists to preserve or restore.

## Verification

- Entity tests prove the removed fields do not exist.
- Migration verification proves sequence allocation and rollback without evidence columns.
- Service tests prove reduction and archive mutate only status and standard audit fields.
- API tests prove actor display names are omitted.
- Run `go test ./api/web-api/internal/entity ./api/web-api/internal/service/accessmanagement ./api/web-api/api`.
- Run `bash bin/verify-role-catalog-migration.sh`.
- Run `just validate-rfc-layout` and the required scoped `just agent-finalize` command.

## Review and Approval

- Challenger: Repository owner
- Outcome: Approved
- Resolved findings: Remove all listed persisted and transient fields without replacement storage.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-08
