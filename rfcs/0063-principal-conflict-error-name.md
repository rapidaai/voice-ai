# RFC 0063: Principal Conflict Error Name

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0062 duplicate-attachment error identifier only

## Context

RFC 0062 names occupied principal-context rejection
`AuthenticationPrincipalAlreadyAttachedError`. The state is a conflict between a requested
attachment and an already occupied context, including a context occupied by a public-route shadow.
`AuthenticationPrincipalConflictError` expresses that domain condition more concisely without
implying that an observable principal remains attached.

## Decision

Rename the centralized error type to `AuthenticationPrincipalConflictError` and use the message
`authentication principal conflicts with request context`. Update every repository caller and test
atomically. Do not retain an alias or compatibility wrapper.

## Contracts and Ownership

- `pkg/errors` continues to own the centralized internal error.
- `pkg/middlewares/authentication` continues to return it for any occupied principal context.
- Raw-key occupancy, invalid-input precedence, public shadowing, and generic transport mappings are
  unchanged.
- No HTTP, gRPC, credential, policy, principal, controller, or persistence contract changes.

## Alternatives Rejected

- Keep `AuthenticationPrincipalAlreadyAttachedError`: rejected because the context may be occupied
  by a hidden public-route shadow rather than an observable attached principal.
- Use `DuplicateAuthenticationPrincipalError`: rejected because the attempted principal may differ
  and a shadowed context exposes no principal to duplicate.
- Retain an alias: rejected because the API is unreleased and duplicate names add no compatibility
  value.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Internal Go callers rename atomically. No released or network contract changes. |
| Failure and recovery | N/A. Error conditions and transport mappings are unchanged. Rollback restores the prior identifier and callers together. |
| Security and privacy | N/A. Context occupancy and failure behavior are unchanged, and the message contains no identity data. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | N/A. Context lifetime and immutability are unchanged. |
| Operations and observability | The internal error text changes. Client responses, status codes, logs, and metrics are unchanged. |

## Rollout and Rollback

Rename the definition, context return, tests, and RFC reference in one candidate. Stop on any stale
old identifier, changed transport behavior, test failure, or unresolved Critical or Major finding.
Rollback restores the prior identifier and callers together.

## Verification

```bash
go test ./pkg/errors ./pkg/middlewares/authentication
go test -race ./pkg/errors ./pkg/middlewares/authentication
go vet ./pkg/errors ./pkg/middlewares/authentication
! rg -n -g '*.go' '\bAuthenticationPrincipalAlreadyAttachedError\b' .
git diff --check
just agent-finalize "rfcs/0063-principal-conflict-error-name.md,pkg/errors/authentication_principal.go,pkg/errors/authentication_principal_test.go,pkg/middlewares/authentication/context.go,pkg/middlewares/authentication/context_test.go"
```

## Review and Approval

- Challenger: Independent gpt-6-astra and gpt-5.6-sol review with combined evidence panel
- Outcome: Approved
- Resolved findings: None
- Open questions: None
- Approval reference: Repository owner approved `AuthenticationPrincipalConflictError` on
  2026-11-13; cross-model review approved digest
  `5fad2be17b9682372b29c4afadec5c3d86be2c86b9ab2bafc98e99cc93d080e1`.
