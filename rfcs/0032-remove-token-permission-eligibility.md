# RFC 0032: Remove Token Permission Eligibility

- Status: Superseded
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-08
- Supersedes: RFC 0031

Superseded by [RFC 0033](0033-remove-role-permission-subject-copy.md).

## Context

The role catalog marks selected project permissions as eligible for project access tokens and
duplicates that flag on role-permission rows. The repository owner decided that a project token may
receive any project-scoped permission, making the eligibility flag and its filter redundant.

## Decision

Remove `token_eligible` from permission definitions and role permissions, including entity fields,
database constraints, seed values, API filters, protobuf fields, generated clients, service
validation, and tests. Scope remains mandatory: `PROJECT_TOKEN` roles remain project-scoped and
cannot receive organization permissions.

Built-in project-token roles keep their existing least-privilege permission bundles. Custom
project-token roles may select any active project permission. Assignment still requires an
explicit role definition; removing the flag does not grant permissions automatically.

All other RFC 0031 and inherited access-management decisions remain unchanged.

## Contracts and Ownership

- `scope_type` is the sole permission-to-role compatibility discriminator.
- `subject_type` continues restricting project tokens to project roles.
- The permission API no longer accepts a token-eligibility filter or returns an eligibility field.
- The role-permission table no longer duplicates permission eligibility.

## Alternatives Rejected

- Keep the flag always true: rejected because it would be redundant state.
- Keep a hard-coded service allowlist: rejected because the owner permits every project permission.
- Expand built-in token roles automatically: rejected because eligibility and default grants are
  separate decisions.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Generated clients lose the unreleased token-eligibility request and response fields. |
| Failure and recovery | N/A. Existing transaction and error behavior remain unchanged. |
| Security and privacy | A deliberately configured token role may receive sensitive project permissions. Explicit role assignment remains the control. |
| Data and migration | Migration 000014 omits both eligibility columns and their constraints. |
| Concurrency and resources | N/A. No lock, transaction, goroutine, or resource behavior changes. |
| Operations and observability | N/A. Audit identity and logging remain unchanged. |

## Rollout and Rollback

Ship the revised migration, proto, generated artifacts, and API together before the unreleased
catalog is deployed. Rollback uses migration 000014 down and the prior generated contract. No
eligibility data requires conversion.

## Verification

- Schema verification proves both eligibility columns are absent and scope foreign keys remain.
- Service tests prove a project-token custom role accepts a formerly ineligible project permission.
- API descriptor tests prove the removed fields are absent.
- Run `bash bin/artifacts-generate.sh`.
- Run `go test ./api/web-api/internal/entity ./api/web-api/internal/service/accessmanagement ./api/web-api/api`.
- Run `bash bin/verify-role-catalog-migration.sh` and `just ci-contracts`.
- Run the required scoped `just agent-finalize` command.

## Review and Approval

- Challenger: Repository owner
- Outcome: Approved
- Resolved findings: All project permissions may be assigned to project-token roles; remove redundant eligibility state.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-08
