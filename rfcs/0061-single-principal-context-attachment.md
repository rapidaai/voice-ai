# RFC 0061: Single Principal Context Attachment

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0059 principal-attachment visibility, `ContextWithPrincipal` test-seam
  signature, and no-production-caller verification contract

## Context

RFC 0059 retained an exported context-only `ContextWithPrincipal` function for controller tests and
a private error-returning `contextWithPrincipal` function for middleware. The exported function only
forwards to the private function and suppresses its error. This gives one security operation two
names and two contracts without separating ownership or behavior.

## Decision

Use one authoritative attachment operation:

```go
func ContextWithPrincipal(context.Context, authentication.Principal) (context.Context, error)
```

Remove the private `contextWithPrincipal` function. HTTP and gRPC authentication middleware call
the exported operation and continue to deny invalid or duplicate attachment. Controller tests use
the same operation and must handle attachment errors explicitly.

This replaces RFC 0059's private-production-operation and exported-test-only split. It also replaces
RFC 0059's verification requirement that `ContextWithPrincipal` have no production caller. RFC
0059's write-once attachment, error mapping, and public-route shadowing requirements remain in
force.

Keep `Principal` as the read operation and `contextWithoutPrincipal` as the private public-route
shadowing operation. Do not export the context key, expose a context-only compatibility wrapper, or
inline attachment logic into transport middleware.

## Contracts and Ownership

- `pkg/middlewares/authentication` remains the sole owner of the private context key and principal
  attachment invariants.
- `ContextWithPrincipal` becomes the sole production and test attachment operation.
- Invalid principals and duplicate attachment return the same centralized errors as RFC 0059.
- Middleware continues to map every attachment error to the existing generic authentication
  failure. No HTTP, gRPC, credential, principal, policy, or controller behavior changes.
- All repository Go callers move atomically to the error-returning signature. No compatibility
  wrapper or alias remains.
- RFC 0059's no-production-caller occurrence checks are obsolete and replaced by RFC 0061's checks
  for one exported definition and no remaining private operation.

## Alternatives Rejected

- Keep the exported wrapper and private operation: rejected because it duplicates one behavior and
  hides attachment errors from tests.
- Keep only the context-returning exported function: rejected because middleware could not
  distinguish invalid or duplicate attachment without inspecting context implementation details.
- Inline attachment in each middleware: rejected because it would duplicate the private key and
  write-once security invariant across HTTP, unary gRPC, and streaming gRPC.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Internal Go test and middleware callers must update atomically. The API is unreleased and has no network compatibility impact. |
| Failure and recovery | Existing invalid and duplicate errors remain unchanged. Rollback restores the prior wrapper and callers together. |
| Security and privacy | One attachment owner reduces ambiguity. Principal contents, context-key privacy, and fail-closed behavior are unchanged. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | N/A. Context values remain immutable and request-scoped. |
| Operations and observability | N/A. Runtime status codes, logs, metrics, and deployment configuration are unchanged. |

## Rollout and Rollback

Change the function signature, remove the private function, and update every middleware and test
caller in one candidate. Stop on any stale private caller, ignored attachment error, test failure,
or unresolved Critical or Major finding. Rollback restores the prior two-function API and callers
together. No data or policy rollback is required.

## Verification

```bash
go test ./pkg/middlewares/authentication ./api/web-api/api/... ./cmd/web
go test -race ./pkg/middlewares/authentication ./api/web-api/api/... ./cmd/web
go vet ./pkg/middlewares/authentication ./api/web-api/api/... ./cmd/web
! rg -n -g '*.go' '\bcontextWithPrincipal\b' .
test "$(rg -n -g '*.go' '^func ContextWithPrincipal\(' . | wc -l | tr -d ' ')" = "1"
git diff --check
just agent-finalize "rfcs/0061-single-principal-context-attachment.md,pkg/middlewares/authentication/context.go,pkg/middlewares/authentication/context_test.go,pkg/middlewares/authentication/grpc.go,pkg/middlewares/authentication/grpc_test.go,pkg/middlewares/authentication/http.go,pkg/middlewares/authentication/http_test.go,api/web-api/api/access_management_test.go,api/web-api/api/auth_test.go,api/web-api/api/dashboard/dashboard_test.go,api/web-api/api/organization_test.go,api/web-api/api/product_usage_test.go,api/web-api/api/project_credential_test.go,api/web-api/api/project_test.go,api/web-api/api/proxy/assistant_test.go,api/web-api/api/proxy/document_test.go"
```

## Review and Approval

- Challenger: Independent gpt-6-astra and gpt-5.6-sol review with combined evidence panel
- Outcome: Approved
- Resolved findings: Expanded `Supersedes` and replacement verification to cover RFC 0059's
  private-production and exported-test-only ownership split.
- Open questions: None
- Approval reference: Repository owner directed removal of the private wrapper on 2026-11-13;
  cross-model review approved corrected digest
  `818969be224322b4be4288f101548d51bf65fedcf81dac38d42f0fab3be95caf`.
