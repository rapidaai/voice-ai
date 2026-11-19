# RFC 0034: Paginate Permission Catalog

- Status: Superseded
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-08
- Supersedes: RFC 0033

## Context

`ListRoles` uses the shared pagination request and response contracts, while `ListPermissions`
returns the complete catalog. The repository owner directed both list APIs to use the same
pagination shape.

## Decision

Add `Paginate paginate` to `ListPermissionsRequest` and `Paginated paginated` to
`ListPermissionsResponse`. A missing pagination request defaults to page 1 with 25 items. A
present request requires a one-based page and a page size from 1 through 100.

The service counts and reads active permissions with the same scope predicate in one repeatable-read,
read-only transaction. Results remain ordered by permission key and then ID. The response reports
the selected page and total filtered count.

All other RFC 0033 and inherited access-management decisions remain unchanged.

## Contracts and Ownership

- The proto owns the shared pagination request and response fields.
- The API owns transport validation and successful response mapping.
- The access-management service owns count and page consistency.

## Alternatives Rejected

- Paginate in memory: rejected because count and page could observe different states and would load
  the full catalog.
- Keep the permission list unpaginated: rejected by the repository owner for list API consistency.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Generated clients gain pagination on the unreleased permission-list contract. |
| Failure and recovery | Invalid page values return `InvalidArgument`; database failures retain existing mapping. |
| Security and privacy | Membership and scope filtering remain unchanged. |
| Data and migration | N/A. No persisted schema changes. |
| Concurrency and resources | Repeatable-read provides count and page consistency without new resources. |
| Operations and observability | Existing list-operation logging remains unchanged. |

## Rollout and Rollback

Ship the proto, generated artifacts, service, and API together before client adoption. Roll back
those files together. No data rollback is required.

## Verification

- API tests cover default and explicit pagination plus invalid bounds.
- Service tests cover count and page selection under the scope filter.
- Descriptor tests verify pagination field numbers.
- Run `bash bin/artifacts-generate.sh`, focused Go tests, `just ci-contracts`, and the required
  scoped `just agent-finalize` command.

## Review and Approval

- Challenger: Repository owner
- Outcome: Approved
- Resolved findings: Match the existing role-list pagination contract.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-08
- Superseded by: RFC 0037. Its pagination contract remains in force; RFC 0037 replaces the inherited hidden user-subject permission compatibility policy.
