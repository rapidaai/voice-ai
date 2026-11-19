# RFC 0062: Strict Principal Context Occupancy

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0061 duplicate-attachment detection and error contract only

## Context

RFC 0061 describes principal attachment as write-once, but the implementation detects an existing
attachment through `Principal`. Public-route shadowing stores a zero principal, and `Principal`
intentionally reports that value as absent. The same request context can therefore accept another
principal after policy has hidden the original identity. Duplicate attachment also returns a
credential error even though the rejected state is an occupied principal context.

## Decision

Treat any `Principal` value stored under the private principal context key as occupied, including
the zero principal used for public-route shadowing. `ContextWithPrincipal` checks raw private-key
occupancy before storing a principal and never permits a second attachment.

Add `pkg/errors.AuthenticationPrincipalAlreadyAttachedError` for occupied-context rejection.
Middleware continues to map this internal error to the existing generic HTTP 401 or gRPC
`Unauthenticated` response.

Keep the two explicit subject-ID checks. `ContextWithPrincipal` validates attachment input, while
`Principal` validates whether a stored value is observable. Do not add a shared predicate or helper.

## Contracts and Ownership

- `pkg/middlewares/authentication` remains the sole owner of the private context key, attachment,
  lookup, and public-route shadowing.
- A request context can transition from absent to attached, then optionally to shadowed. Attached
  and shadowed states are both permanently occupied.
- `pkg/errors` owns the new internal duplicate-attachment error.
- Invalid principal input still returns `AuthenticationPrincipalInvalidError`.
- HTTP, gRPC, policy, credential, principal, controller, and persistence contracts are unchanged.

## Alternatives Rejected

- Detect duplicates through `Principal`: rejected because shadowed state is intentionally not
  observable as a principal.
- Let public handlers reattach a principal: rejected because it violates public-route identity
  shadowing and the write-once request invariant.
- Return `AuthenticationCredentialInvalidError`: rejected because no credential is invalidated by
  an internal context-state conflict.
- Extract a validity helper: rejected because input validation and observable-value validation are
  separate ownership boundaries despite sharing one predicate.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The internal duplicate error type changes. Transport clients continue to receive the same generic denial. |
| Failure and recovery | Reattachment after public shadowing now fails closed. Rollback restores the prior occupancy check and error. |
| Security and privacy | Public-route identity shadowing becomes irreversible for the request context. No identity or credential values are exposed. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | N/A. Context values remain immutable and request-scoped. |
| Operations and observability | N/A. Status codes, logs, metrics, and deployment configuration are unchanged. |

## Rollout and Rollback

Add the centralized error, change the private-key occupancy check, and add regression coverage in
one candidate. Stop on any changed transport response, successful reattachment after shadowing,
test failure, or unresolved Critical or Major finding. Rollback restores the prior check and error
together. No data or policy rollback is required.

## Verification

```bash
go test ./pkg/errors ./pkg/middlewares/authentication
go test -race ./pkg/errors ./pkg/middlewares/authentication
go vet ./pkg/errors ./pkg/middlewares/authentication
git diff --check
just agent-finalize "rfcs/0062-strict-principal-context-occupancy.md,pkg/errors/authentication_principal.go,pkg/errors/authentication_principal_test.go,pkg/middlewares/authentication/context.go,pkg/middlewares/authentication/context_test.go"
```

## Review and Approval

- Challenger: Independent gpt-6-astra and gpt-5.6-sol review with combined evidence panel
- Outcome: Approved
- Resolved findings: None
- Open questions: None
- Approval reference: Repository owner approved the strict occupancy correction on 2026-11-13;
  cross-model review approved digest
  `906e5718e9ad705b41da91a1a46c1be1cffdd6bd8dc82dd92add567a54c6bfa3`.
