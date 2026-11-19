# RFC 0086: Unified VerifyToken Credential Resolution

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-15
- Supersedes: RFC 0076 resolver-service protocol and RFC 0085 resolver transport only

## Context

RFC 0076 introduced `AuthenticationResolverService` with three methods even though Web already
owns `AuthenticationService.VerifyToken`. The old `VerifyToken` only looks up a User token and
returns its storage record. It cannot construct the canonical Authentication aggregate and remains
public in route policy and the React SDK.

The three resolver methods duplicate one trust boundary and one response type. Their only material
difference is the credential-specific request and persistence operation. The repository owner
requires one clean protocol with no compatibility service.

## Decision

Remove `AuthenticationResolverService` and all resolver request and response messages from
`authentication.proto`. Keep `authentication.proto` as the reusable aggregate contract only.

Replace the legacy `VerifyToken` request and response in `web-api.proto`:

```proto
message VerifyTokenRequest {
  reserved 1, 2;
  reserved "tokenType", "token";

  message UserCredential {
    string token = 1;
    uint64 userId = 2 [jstype = JS_STRING];
    uint64 selectedOrganizationId = 3 [jstype = JS_STRING];
    uint64 selectedProjectId = 4 [jstype = JS_STRING];
  }

  message ProjectCredential { string token = 1; }
  message OrganizationCredential { string token = 1; }

  oneof credential {
    UserCredential user = 3;
    ProjectCredential project = 4;
    OrganizationCredential organization = 5;
  }
}

message VerifyTokenResponse {
  reserved 1, 2, 3;
  reserved "code", "success", "data";
  authentication.Authentication authentication = 4;
}
```

`AuthenticationService.VerifyToken` becomes an internal-only service route. It requires a direct,
tenantless Service Principal admitted by the existing internal caller policy. The request oneof
selects exactly one credential family. The Web handler validates that family, delegates to the
existing User, Project, or Organization credential service, and returns the canonical
`authentication.Authentication`. Invalid credentials return `Unauthenticated`; cancellation and
deadline errors retain their gRPC status; unexpected persistence failures return `Internal`
without exposing credential material.

Internal Web clients use the existing `AuthenticationServiceClient` and keep implementing the
three credential-specific resolver interfaces consumed by authentication middleware. The
credential-specific interfaces remain because they make illegal cross-family calls impossible at
the middleware boundary. They no longer require separate wire methods.

Remove the public React `VerifyToken` convenience wrapper and the unused UI wrapper. Generated
clients still contain the service method, but browser calls cannot satisfy its Service policy.
`Authorize` remains the User session-refresh RPC and is not used for credential resolution.
Update the Web API and React SDK READMEs so neither describes VerifyToken as a public browser or
one-time-token API.

## Contracts and Ownership

- `web-api.proto` owns the sole credential-resolution RPC and its closed request variants.
- `authentication.proto` owns only reusable Authentication aggregate messages.
- Web `AuthenticationService.VerifyToken` owns credential-family dispatch and gRPC failure mapping.
- Existing User, Project, and Organization authentication services retain persistence verification
  and canonical aggregate construction.
- `pkg/clients/web` owns conversion from the three typed Go resolver calls to the oneof wire request.
- Web route policy owns service-only admission. The handler does not accept public callers.
- Commands continue to install credential-specific middleware and resolvers; only their Web client
  constructor changes.
- Generated Go, Python, JavaScript, and TypeScript artifacts are refreshed from authoritative proto
  sources. Stale resolver-service artifacts are deleted.

## Alternatives Rejected

- Keep three resolver RPCs: rejected because credential family is request data, not three service
  boundaries.
- Keep legacy `VerifyToken` beside the resolver service: rejected because it preserves two token
  verification contracts and a public storage-record response.
- Use free-form `credentialType` plus shared fields: rejected because irrelevant fields and unknown
  combinations become representable.
- Route resolution through `Authorize`: rejected because `Authorize` consumes an already attached
  User Principal and refreshes its session. Using it from authentication middleware would invert
  ownership and cannot resolve Project or Organization credentials.
- Retain the React wrapper for compatibility: rejected because the replacement route requires an
  internal Service Principal and has no browser use.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | This intentionally replaces both the public legacy `VerifyToken` payload and the internal resolver service. Removed tags and names remain reserved. All internal consumers and generated artifacts must deploy as one candidate. |
| Failure and recovery | Invalid, absent, or unsupported oneof variants fail closed. Existing credential services retain cancellation, deadline, revocation, and database error behavior. |
| Security and privacy | A formerly public RPC becomes service-only. Raw credentials remain request-scoped, are not logged or cached, and return only the canonical aggregate to an admitted internal caller. |
| Data and migration | No database or persisted-data change is required. |
| Concurrency and resources | N/A. Dispatch remains request-scoped and adds no goroutine, cache, or retained resource. |
| Operations and observability | Existing credential-resolution error logs remain credential-free. Deployment must be coordinated because old and new service descriptors are not compatible. |

The dirty worktree has a pre-existing `TestWebRPCAuthenticationContract` failure for proxy
handlers outside this RFC. Scoped verification can approve this implementation boundary, but the
repository candidate is not shippable until that separate Web API gate is dispositioned and full
finalization passes. Fixing unrelated proxy handlers is not authorized by this RFC.

## Rollout and Rollback

Regenerate all protocol artifacts, update Web registration and policy, then update every internal
client and command in the same candidate. Stop if the removed resolver service remains in any
descriptor, policy, generated source, or runtime call site; if `VerifyToken` remains public; or if
any credential family returns a different Authentication aggregate.

Deploy Web and its internal consumers atomically. Rollback restores the previous generated
artifacts, resolver service registration, policy entries, and internal clients together. No data
rollback is needed. External users of the old public `VerifyToken` are intentionally unsupported by
the clean cutover. Do not deploy while the pre-existing Web API package gate remains red.

## Verification

Acceptance requires:

- descriptor tests proving `authentication.proto` has no services and `VerifyToken` has the exact
  reserved fields, oneof variants, nested request fields, and Authentication response;
- handler tests for all three credential families, missing variants, malformed values, direct
  Service admission, cancellation, deadlines, invalid credentials, and internal failures;
- client tests proving every typed resolver call produces the correct oneof and returns the same
  aggregate;
- policy tests proving the single route is service-only and the three removed routes are absent;
- repository search proving no resolver-service symbol or public React/UI wrapper remains;
- regenerated root and React artifacts and matching authoritative proto sources.

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./protos ./pkg/authentication/... ./pkg/clients/web ./pkg/clients/rapida ./api/web-api/api ./api/web-api/router ./cmd/assistant ./cmd/endpoint ./cmd/integration
go test -race -count=1 ./pkg/clients/web ./pkg/clients/rapida ./api/web-api/api
(cd sdks/react && npm test -- --runInBand)
(cd ui && yarn test --watchAll=false --runInBand)
git diff --check
just agent-finalize "$(paste -sd, plans/0086-finalize-paths.txt)"
```

`just agent-finalize` must be run and its existing Web API package failure recorded. Delivery
requires a later clean run after that out-of-scope proxy contract is resolved.

## Review and Approval

- Challenger: Independent Codex protocol and lifecycle review
- Outcome: Approved
- Resolved findings: Added root and React Web proto parity. Recorded the pre-existing Web API
  contract-test failure as an out-of-scope delivery blocker that scoped verification cannot waive.
- Open questions: None
- Approval reference: Repository owner directed replacement of the resolver service with the
  existing `AuthenticationService.VerifyToken` RPC on 2026-11-15.
