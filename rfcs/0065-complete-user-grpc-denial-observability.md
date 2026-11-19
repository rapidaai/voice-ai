# RFC 0065: Complete User gRPC Denial Observability

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: N/A

## Context

The accepted gRPC authentication middleware returns the correct generic `Unauthenticated` status
for malformed User project identifiers and principal-attachment conflicts, but those branches do
not emit the generic denial log used by the other User rejection paths. Unary policy coverage tests
all route outcomes, while streaming policy coverage tests only the public outcome. Review found no
Critical or Major defect, but these gaps weaken operational evidence and regression protection.

## Decision

Emit the existing generic User authentication denial log before every non-context rejection in the
User unary and streaming gRPC middleware. Cancellation and deadline returns remain unlogged as
authentication denials.

Add behavior tests that prove the previously silent User branches emit the generic denial event and
that streaming policy middleware preserves the accepted public, protected, disabled, unknown,
missing-principal, unauthorized-principal, and nil-policy outcomes without invoking a denied
handler.

## Contracts and Ownership

- `pkg/middlewares/authentication` continues to own transport validation, principal attachment,
  generic denial logging, status mapping, and route-policy enforcement.
- Credential values, identifiers, authorities, and resolver errors remain absent from logs.
- Resolver calls, policy decisions, gRPC status codes, controller invocation, middleware ordering,
  and public contracts remain unchanged.

## Alternatives Rejected

- Leave the gaps: rejected because equivalent User rejection paths would remain operationally
  indistinguishable from missing middleware execution.
- Refactor repeated logging into a helper: rejected because the direct control flow is small and
  repository rules prefer keeping ownership visible in the middleware.
- Expand logging to cancellation and deadline paths: rejected because those are caller lifecycle
  outcomes, not authentication denials.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Status codes, messages, principals, policies, and network contracts are unchanged. |
| Failure and recovery | Rejected requests terminate exactly as before. Rollback removes the new log calls and tests. |
| Security and privacy | Logs contain only the canonical gRPC method, never credential or identity values. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | The existing command-owned concurrency-safe logger is reused. No resource lifetime changes. |
| Operations and observability | Every non-context User gRPC rejection emits the established generic denial event. |

## Rollout and Rollback

Ship the direct log calls and focused tests atomically. Stop on changed status mapping, handler
invocation, resolver behavior, secret-bearing log output, test failure, or an unresolved Critical or
Major finding. Rollback restores the prior silent branches; no data rollback is required.

## Verification

```bash
go test -count=1 ./pkg/middlewares/authentication ./cmd/web
go test -race -count=1 ./pkg/middlewares/authentication ./cmd/web
go vet ./pkg/middlewares/authentication ./cmd/web
git diff --check
just agent-finalize "rfcs/0065-complete-user-grpc-denial-observability.md,pkg/middlewares/authentication/grpc.go,pkg/middlewares/authentication/grpc_test.go"
```

Acceptance requires exact generic log coverage for the eight previously silent User rejection
branches and direct streaming policy assertions for every accepted outcome listed in the Decision.

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol reviews with combined evidence panel
- Outcome: Approved
- Resolved findings: None
- Open questions: None
- Approval reference: Repository owner requested both verified gRPC middleware findings be fixed on
  2026-11-13. Independent reviews and the combined panel approved draft digest
  `e8983feb153700298938d0fef3eca1120e0a72e8cef035a45d3c4ec61e253975`.
