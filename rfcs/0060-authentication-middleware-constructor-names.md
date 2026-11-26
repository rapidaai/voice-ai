# RFC 0060: Authentication Middleware Constructor Names

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0059 free-factory names only

## Context

RFC 0059 accepted native HTTP and gRPC middleware factories named `ProjectUnary`, `ProjectStream`,
`ProjectHTTP`, and equivalent functions for Organization, Service, User, and Policy. These
functions construct middleware closures, but their names do not follow the repository's established
`New...Middleware` constructor form.

## Decision

Keep RFC 0059's transport-specific middleware ownership and behavior unchanged. Rename the
factories consistently:

```go
NewProjectUnaryMiddleware(*resolver.ProjectPrincipalResolver, commons.Logger) grpc.UnaryServerInterceptor
NewProjectStreamMiddleware(*resolver.ProjectPrincipalResolver, commons.Logger) grpc.StreamServerInterceptor
NewProjectHTTPMiddleware(*resolver.ProjectPrincipalResolver, commons.Logger) gin.HandlerFunc
```

Apply the same `New<Family><Transport>Middleware` form to Organization, Service, User, and Policy.
Do not add a cross-transport middleware object, compatibility aliases, or forwarding wrappers.
HTTP and gRPC remain independently installable, and unsupported credential families remain absent
from command wiring.

## Contracts and Ownership

- `pkg/middlewares/authentication` owns the constructor names and returned native middleware types.
- RFC 0059 continues to own authentication behavior, ordering, failure mapping, principal
  attachment, policy enforcement, and transport independence.
- Every repository caller moves atomically to the new names. The old names are removed.
- No public HTTP, protobuf, credential, principal, JWT, policy, or controller contract changes.

## Alternatives Rejected

- Keep names without `New` or `Middleware`: rejected because they hide that the functions construct
  middleware closures.
- Add one `NewProjectMiddleware` object with HTTP, unary, and stream methods: rejected because it
  couples transport configuration and conflicts with RFC 0059's independent installation model.
- Retain aliases for the old names: rejected because the implementation is not released and aliases
  would create duplicate APIs without a compatibility requirement.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Internal Go callers must rename atomically. No released or network contract changes. |
| Failure and recovery | N/A. Middleware behavior and failure mapping are unchanged. Rollback restores the prior names with their callers. |
| Security and privacy | N/A. Authentication, authorization, secret handling, and trust boundaries are unchanged. |
| Data and migration | N/A. No persisted data or migration changes. |
| Concurrency and resources | N/A. Returned closures and resolver lifetimes are unchanged. |
| Operations and observability | N/A. Runtime logs, metrics, statuses, and rollout behavior are unchanged. |

## Rollout and Rollback

Rename definitions, tests, and all command callers in one change. Do not leave compatibility
wrappers. Stop on any stale caller, test failure, or unresolved Critical or Major review finding.
Rollback restores the prior definitions and callers together. No data, policy, or deployment
rollback is required.

## Verification

```bash
go test ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go test -race ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go vet ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
! rg -n -g '*.go' '\b(Project|Organization|Service|User|Policy)(Unary|Stream|HTTP)\b' .
just agent-finalize "rfcs/0060-authentication-middleware-constructor-names.md,pkg/middlewares/authentication/grpc.go,pkg/middlewares/authentication/grpc_test.go,pkg/middlewares/authentication/http.go,pkg/middlewares/authentication/http_test.go,cmd/web/web.go,cmd/web/authentication_middleware_test.go"
```

## Review and Approval

- Challenger: Independent gpt-6-astra and gpt-5.6-sol review with combined evidence panel
- Outcome: Approved
- Resolved findings: Expanded stale-name verification to all repository Go sources and reference forms.
- Open questions: None
- Approval reference: Repository owner approved the `New<Family><Transport>Middleware` naming on
  2026-11-13; cross-model review approved final digest
  `503b1dca6009912205f5379a005700bcdc03aca8bde940a6a645f25b914fa536`.
