# RFC 0046: Access Management Controller Validation and Errors

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-10
- Supersedes: RFC 0030 access-management failure mapping, as inherited through RFC 0037

## Context

The unreleased access-management handlers currently return gRPC `InvalidArgument` for request
validation while their responses already expose the Web API `code`, `success`, and `error`
envelope. They also define public messages in the controller, define domain sentinels in the
service package, and route failures through a shared status-mapping method. This differs from the
existing Web API convention and obscures which layer owns validation and public errors.

## Decision

Access-management request validation belongs to the controller and uses the existing
`pkg/validator` package where it provides the required predicate. Validation failures return a
failure response with gRPC status `OK`, `code=400`, `success=false`, and an operation-specific
`pkg/errors.PlatformError`. The controller performs each mapping directly in the affected handler.
It does not introduce response builders, status mappers, validation helpers, or utility functions.

The error catalog is fixed before release:

| Condition | `code` | `error.errorCode` | Machine message | Human message |
| --- | ---: | ---: | --- | --- |
| Invalid permission criteria | 400 | 1026301 | `invalid permission criteria` | `Use a supported permission filter and try again.` |
| Invalid role criteria | 400 | 1026302 | `invalid role criteria` | `Use supported role filters and try again.` |
| Invalid pagination | 400 | 1026303 | `invalid pagination` | `Choose a page and page size between 1 and 100.` |
| Invalid role identifier | 400 | 1026304 | `invalid role identifier` | `Select a valid role and try again.` |
| Invalid role scope | 400 | 1026305 | `invalid role scope` | `Select an organization or project scope and try again.` |
| Invalid role subject | 400 | 1026306 | `invalid role subject` | `Select console or API access and try again.` |
| API-access role without Project scope | 400 | 1026307 | `API access roles require project scope` | `Use project scope for API access roles.` |
| Invalid role key | 400 | 1026308 | `invalid role key` | `Use 3 to 64 lowercase letters, numbers, or underscores, starting with a letter.` |
| Invalid role name | 400 | 1026309 | `invalid role name` | `Enter a role name with 200 characters or fewer.` |
| Invalid role description | 400 | 1026310 | `invalid role description` | `Use 400 characters or fewer for the role description.` |
| Invalid permission identifiers or incompatible permission selection | 400 | 1026311 | `invalid role permission selection` | `Select valid permissions for the role scope and try again.` |
| Invalid role-change reason | 400 | 1026312 | `invalid role change reason` | `Enter a reason with 400 characters or fewer.` |
| Unknown, archived, or tenant-hidden role | 404 | 1026313 | `role not found` | `The selected role is unavailable.` |
| Duplicate built-in or same-Organization custom key | 409 | 1026314 | `role key already exists` | `Choose a different role key and try again.` |
| Built-in, already changed, assigned, or otherwise immutable role state | 400 | 1026315 | `role change is not allowed` | `This role cannot be changed in its current state.` |

`1026301..1026315` are reserved for these errors and must be unique in `pkg/errors`. Missing or
malformed server correlation context is not caller validation and remains an internal gRPC error.

The service does not validate request shape. It owns authentication invariants, authorization,
tenant visibility, database-backed role and permission eligibility, transactions, and state
transitions. Typed domain errors and public platform errors are defined in `pkg/errors`; the
access-management service package defines no sentinel errors. Database-backed missing,
conflicting, or immutable state is mapped directly by each controller handler.

The domain mapping is exact: authorization denial uses a typed `pkg/errors` error and maps to gRPC
`PermissionDenied`; hidden role state maps to `1026313`; duplicate role key maps to `1026314`;
unknown or cross-scope permission selection maps to `1026311`; and immutable or conflicting role
state maps to `1026315`. Controllers use `errors.As` for these types and do not publish service
error text.

Authentication, authorization, cancellation, deadline, and unexpected persistence failures keep
their existing non-`OK` gRPC status and return no response message. Successful response contracts
and protobuf fields do not change. Project credential handlers retain RFC 0040 and RFC 0041:
their existing validator-based request checks and platform-error envelopes already follow this
decision.

## Contracts and Ownership

- Access-management controllers own request parsing, trimming, structural validation, and direct
  construction of success and expected-failure responses.
- `pkg/errors` owns access-management domain error types, application codes, machine messages, and
  human messages.
- The access-management service owns authorization and database behavior, without public request
  validation or public error text.
- Existing access-management helper methods and free functions for status mapping, membership
  checks, role lookup/query construction, authentication extraction, permission-ID projection, and
  context-error substitution are removed. Their logic is kept directly in each owning handler or
  service method. The two dependency constructors remain because they are composition boundaries,
  not behavior utilities.
- Invalid access-management requests intentionally change from non-`OK` gRPC `InvalidArgument` to
  gRPC `OK` with a `400` failure envelope before release.

## Alternatives Rejected

- Keep gRPC `InvalidArgument`: rejected because the Web API exposes expected request failures
  through its standard response envelope.
- Keep service-local sentinels or public controller strings: rejected because `pkg/errors` is the
  repository error source of truth.
- Add shared builders, mappers, or validation functions: rejected because the handlers are the
  explicit owners and the extra indirection is not reused outside this unreleased API.
- Retain the existing new-code service helpers: rejected because they split short operation flows
  across generic functions and hide operation-specific authorization and error behavior.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Unreleased invalid-input behavior changes from gRPC `InvalidArgument` to a `400` response envelope. Success fields and endpoint signatures are unchanged. |
| Failure and recovery | Expected validation and domain failures are returned to generated clients as responses. Cancellation, deadline, and unexpected failures remain transport errors. |
| Security and privacy | Tenant-hidden resources retain indistinguishable domain errors. Controller responses do not expose database or cross-tenant detail. |
| Data and migration | N/A. No schema, migration, or persisted value changes. |
| Concurrency and resources | N/A. Transaction boundaries, locks, and resource lifetimes do not change. |
| Operations and observability | Existing internal failure logging and correlation data remain. Application error codes make expected client failures distinguishable without parsing text. |

## Rollout and Rollback

Ship the error catalog, controller mappings, service error replacement, tests, and clients before
the unreleased access-management API is deployed. Stop if an application code collides, a request
validation path reaches the service, or a hidden-resource response becomes distinguishable.
Rollback reverts the backend correction and client expectations together; there is no data
rollback.

## Verification

- Controller tests cover every validation family, prove the service is not called, and assert the
  exact `400` platform-error envelope.
- Service tests prove authorization and database domain failures use `pkg/errors` types and that
  request-shape validation is absent.
- Source checks prove no access-management status mapper, validation helper, response builder, or
  service-local sentinel remains, and no application error code from `1026301..1026315` collides.
- Run `go test ./pkg/errors ./api/web-api/api ./api/web-api/internal/service/accessmanagement`.
- Run `go test ./api/web-api/...`.
- Run `just validate-rfc-layout`, `git diff --check`, and the scoped `just agent-finalize` command.

## Review and Approval

- Challenger: Codex independent review, task `/root/access_error_rfc_challenge`
- Outcome: Approved
- Resolved findings: Corrected the superseded backend decision, fixed the complete public error
  catalog and domain mapping, and made removal of every existing new-code helper explicit.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-10
