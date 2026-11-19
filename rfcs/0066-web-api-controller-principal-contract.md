# RFC 0066: Web API Controller Principal Contract

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: N/A

## Context

Generated unary gRPC handlers allocate a concrete protobuf request before invoking a Web API
controller. Eleven controllers nevertheless replace a nil request with an empty protobuf message.
Those branches cannot execute through the registered transport and obscure transport ownership.

The Web API authentication-contract inventory tracks 116 controllers but the current route policy
contains 121 protected gRPC methods. It includes disabled controllers, omits the Access Management,
Dashboard, Product Usage, scoped-authentication, and billing-management additions, and currently
misses three valid principal consumers because one file uses a different middleware import alias.
The omitted `UpdateBillingInformation` controller is the only protected Web API controller that
does not read the new immutable principal.

## Decision

Treat the generated gRPC transport as the sole owner of request allocation. Remove every controller
branch that substitutes an empty protobuf request for nil. Direct Go invocation with a nil request
is unsupported; registered unary transport invocation remains nonnil by construction.

Make the versioned Web API route policy the protected-controller inventory source of truth. Update
the controller matrix to contain exactly every non-public, non-disabled gRPC method and add a test
that rejects policy/matrix drift. Every inventoried controller must immediately fail closed when
the new middleware principal is absent. Standardize the middleware import alias and make
`UpdateBillingInformation` consume and validate the new principal before its existing placeholder
result.

Do not migrate Assistant API, Endpoint API, Integration API, or internal runtime authentication in
this change. Their accepted legacy state and later cutovers remain separate decisions.

## Contracts and Ownership

- Generated gRPC handlers own construction and decoding of nonnil request messages.
- `pkg/authentication/policy/v1/web-api.yaml` remains the sole source of truth for protected Web API
  routes and actor authorization.
- `api/web-api/api` controllers consume only `pkg/authentication.Principal` through
  `pkg/middlewares/authentication.Principal`; legacy `types.Authentication`, `types.Principle`, and
  `types.CTX_` are forbidden in production controller files.
- Public and disabled routes do not need a controller principal contract because policy either
  shadows identity or prevents controller invocation.
- Protobuf, status, policy decision, resolver, service, persistence, and network contracts remain
  unchanged for registered transport calls.

## Alternatives Rejected

- Keep nil substitution for direct unit calls: rejected because it duplicates generated transport
  ownership for a state the registered server cannot produce.
- Audit controllers only with grep: rejected because it does not prove that the controller inventory
  remains synchronized with route policy.
- Convert all deployables in one change: rejected because each service owns a separate trust
  boundary, middleware selection, policy, and rollout.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Registered gRPC calls are unchanged. Direct Go calls with nil requests become explicitly unsupported. |
| Failure and recovery | Missing principals fail closed before controller work. Rollback restores the substitutions and prior inventory. |
| Security and privacy | Every protected Web API controller is proven to consume the immutable principal; no credential data is added. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | N/A. No goroutine, connection, or resource ownership changes. |
| Operations and observability | N/A. Status mapping and logging remain unchanged. |

## Rollout and Rollback

Remove the eleven substitutions, synchronize the protected-controller matrix, repair the principal
contract, and add the drift assertion in one candidate. Stop on any changed registered status,
service invocation, policy entry, generated file, test failure, or unresolved Critical or Major
finding. Rollback restores the controller and test changes together; no data rollback is required.

## Verification

```bash
go test -count=1 ./api/web-api/api/... ./cmd/web
go test -race -count=1 ./api/web-api/api/... ./cmd/web
go vet ./api/web-api/api/... ./cmd/web
! rg -n -U 'if\s+[A-Za-z_][A-Za-z0-9_]*\s*==\s*nil\s*\{\s*\n\s*[A-Za-z_][A-Za-z0-9_]*\s*=\s*&protos\.[A-Za-z0-9_]+\{\}\s*\n\s*\}' api/web-api/api --glob '*.go'
! rg -n 'types\.(Authentication|Principle)|types\.CTX_' api/web-api/api --glob '*.go' --glob '!*_test.go'
go test -count=1 ./api/web-api/api -run TestWebRPCAuthenticationContract
git diff --check
just agent-finalize "rfcs/0066-web-api-controller-principal-contract.md,api/web-api/api/access_management.go,api/web-api/api/authentication_contract_test.go,api/web-api/api/testdata/authentication-policy-matrix.json,api/web-api/api/dashboard/dashboard.go,api/web-api/api/organization.go,api/web-api/api/organization_membership.go,api/web-api/api/project.go,api/web-api/api/vault.go"
```

Acceptance requires zero empty-request substitutions, exact policy/matrix parity for all 121
protected gRPC methods, successful fail-closed principal checks in every inventoried controller,
and no legacy authentication object in production Web API controllers.

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol reviews with combined evidence panel
- Outcome: Approved
- Resolved findings: The controller inventory now compares all 121 canonical gRPC methods and maps
  each to one controller. The contract checker proves the missing-principal guard terminates with
  `Unauthenticated`, includes a negative checker test, and directly tests billing principal and
  tenant rejection.
- Open questions: None
- Approval reference: Repository owner requested removal of empty-request substitution and complete
  Web API principal verification on 2026-11-13. Independent reviews and the combined panel approved
  draft digest `74f8ca5c4ad0d40ff719010226deb2ac0e810796e33bf5a5c4c09b020fa5eb67`.
