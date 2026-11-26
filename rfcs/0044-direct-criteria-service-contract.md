# RFC 0044: Direct Criteria Service Contract

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-09
- Supersedes: RFC 0042 typed service-filter boundary decision only

## Context

RFC 0042 requires controllers to convert validated criteria into new typed filter structures before
calling services. Existing repository list services, including `GetAllAssistant`, pass
`[]*protos.Criteria` through the service boundary and map allowlisted keys with fixed switch cases.
Adding credential and access-management filter types creates a second internal pattern without a
separate behavioral need.

## Decision

Project credential and access-management list service methods accept `[]*protos.Criteria` and
`*protos.Paginate` directly, matching established list-service signatures. Do not introduce
`ProjectCredentialFilter`, `PermissionFilter`, or `RoleFilter` types.

Controllers remain the sole validation owner defined by RFC 0042. They reject invalid counts, nil
criteria, unsupported keys or logic, duplicate keys, invalid enum values, blank or oversized text,
and NUL before calling a service. Services map the already validated slice through explicit key
switches and fixed parameterized predicates. Unsupported keys have no predicate and no caller value
is ever used as a SQL identifier or operator. Criteria are combined with AND.

This changes only the internal call shape. RFC 0042's public criteria contracts, allowlists, error
behavior, authorization, query semantics, UI, SDK, rollout, and verification remain unchanged.

## Contracts and Ownership

- Controllers own all request and criteria validation.
- Services own tenant predicates, explicit criterion-to-column mapping, pagination, ordering, and
  persistence reads.
- Protobuf `Criteria` is the shared boundary type; no additional filter type duplicates it.

## Alternatives Rejected

- Add typed filter structures: rejected because they duplicate the established criteria boundary
  without adding safety once the service uses fixed key switches.
- Build SQL from criterion keys or logic: rejected because it permits caller-controlled SQL syntax.
- Validate independently in both layers: rejected because it duplicates policy ownership.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Internal Go interfaces change within the same coordinated feature. Public contracts remain RFC 0042. |
| Failure and recovery | Invalid criteria stop in controllers. Read failures remain retryable and use existing error mapping. |
| Security and privacy | Fixed service switches and parameterized values prevent SQL identifier and operator injection. Tenant predicates remain mandatory. |
| Data and migration | N/A. No persistent data changes. |
| Concurrency and resources | N/A. Existing count, page, and transaction behavior remains. |
| Operations and observability | Existing list error monitoring remains unchanged. |

## Rollout and Rollback

Ship controller and service signature changes together. Roll back both layers together. No data
rollback or compatibility window is required.

## Verification

Controller tests cover every invalid criterion class and prove the service is not called. Service
tests cover every allowlisted predicate, AND composition, literal wildcard handling, tenant scope,
count consistency, and omitted criteria. Run the complete RFC 0042 verification suite.

## Review and Approval

- Challenger: Codex, task `/root/criteria_filter_rfc_challenge`
- Reviewed digest: SHA-256 `227af0c878279c72aff1a076e313df065b7a4405ab51891b00fad34e82493248`
- Outcome: Approved
- Resolved findings: None
- Open questions: None
- Approval reference: Repository owner instruction to follow the established GetAllAssistant criteria contract without additional types, 2026-11-09
