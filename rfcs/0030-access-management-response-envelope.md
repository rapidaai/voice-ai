# RFC 0030: Access Management Response Envelope

- Status: Superseded
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-07
- Supersedes: RFC 0029

Superseded by [RFC 0031](0031-remove-role-reduction-evidence.md).

## Context

RFC 0029 defined access-management responses without the response envelope used by the existing
web API. Before this new API is released, the repository owner directed that every response expose
the standard `code`, `success`, and `error` fields and that the redundant internal gRPC constructor
be removed.

## Decision

Every `AccessManagementService` response uses field 1 for `int32 code`, field 2 for `bool success`,
field 3 for its primary payload, and field 4 for `Error error`. Responses with secondary pagination
or correlation metadata use field 5 for that metadata. Successful handlers return code 200,
success true, and no error value. Existing canonical gRPC status behavior for authentication,
authorization, validation, domain, cancellation, deadline, and persistence failures remains
unchanged.

`NewAccessManagementGRPC` directly constructs the production server. No second constructor or
mapping utility is introduced. Tests construct the server struct directly when injecting a fake
service.

All other RFC 0029 role lifecycle, authorization, persistence, replay, audit, rollout, and rollback
decisions remain unchanged.

## Contracts and Ownership

- `protos/artifacts/access-management-api.proto` owns the response field layout.
- `api/web-api/api/access_management.go` owns successful envelope values and existing gRPC status
  mapping.
- Generated Go and Python artifacts remain generator-owned.
- The response field renumbering intentionally replaces the unreleased RFC 0029 draft contract.

## Alternatives Rejected

- Keep bare gRPC responses: rejected because the repository owner requires parity with existing web
  API response envelopes.
- Embed `BaseResponse`: rejected because existing contracts expose the three envelope fields
  directly and payload field 3 remains clearer.
- Add another response-mapping abstraction: rejected because direct handler construction follows
  the approved minimal implementation style.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Generated clients must use the new unreleased field numbers. No released consumer exists. |
| Failure and recovery | Failure behavior remains canonical gRPC status. Envelope fields describe successful responses. |
| Security and privacy | N/A. Authentication, authorization, tenant filtering, and error detail remain unchanged. |
| Data and migration | N/A. No database or persisted-data change. |
| Concurrency and resources | N/A. No transaction, lock, goroutine, or resource ownership change. |
| Operations and observability | Existing logging, correlation, and gRPC status behavior remain unchanged. |

## Rollout and Rollback

Generate and ship the proto artifacts with the web API implementation before any SDK or UI client
adopts the service. Roll back the API and generated artifacts together before client adoption. No
data rollback is required.

## Verification

- Every response descriptor exposes `code=1`, `success=2`, its primary payload at 3, and `error=4`.
- Secondary response metadata uses field 5.
- Every successful RPC test asserts code 200 and success true.
- The production constructor registers the service without a second constructor.
- Run `bash bin/artifacts-generate.sh`.
- Run `go test ./api/web-api/api ./api/web-api/router`.
- Run `just ci-contracts`.
- Run `just agent-finalize` with the changed paths.

## Review and Approval

- Challenger: Repository owner
- Outcome: Approved
- Resolved findings: Use the existing direct envelope field pattern and remove the extra constructor.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-07
