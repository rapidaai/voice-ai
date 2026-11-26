# RFC 0070: Retain Authentication Response

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0069

## Context

RFC 0069 replaced the existing public `Authentication` response while the Access Management
cutover was still in progress. The console already depends on that response for the complete set of
Organization and Project role assignments and their effective permissions. Changing the public
session shape is not required to introduce the new request-scoped authentication Principal.

## Decision

Retain the existing public `Authentication` protobuf response and its established fields. Continue
returning all Organization and Project assigned roles and effective permissions required by the UI.
Keep the request-scoped `authentication.Principal` internal to middleware, route authorization, and
controllers. The Principal is not serialized and does not replace the client session response.

This decision cancels the `UserSession`, `AuthenticatedUser`, `OrganizationMembership`, and
`ProjectMembership` public protocol migration from RFC 0069. Authentication request messages and
RPC methods remain unchanged.

## Contracts and Ownership

- The Web API User service owns the canonical role and effective-permission projection.
- Authentication controllers map that projection to the existing public `Authentication` message.
- Middleware owns the internal request Principal. Public response compatibility remains unchanged.
- Proto sources remain the source of truth and generated artifacts are regenerated, not edited.

## Alternatives Rejected

- Replace `Authentication` with `UserSession`: rejected because it creates a public migration with
  no requirement for the Principal cutover.
- Serialize the request Principal: rejected because one selected request scope is not the complete
  UI session projection.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Existing generated clients and UI response accessors remain compatible. |
| Failure and recovery | Regeneration must restore identical proto sources and consumers. A partial migration must not ship. |
| Security and privacy | The Principal remains private. The response retains only its established role and permission projection. |
| Data and migration | N/A. No schema or persisted-session migration is introduced. |
| Concurrency and resources | N/A. No resource ownership changes. |
| Operations and observability | Existing authentication response behavior and monitoring remain unchanged. |

## Rollout and Rollback

Restore the existing proto source, regenerate all artifacts, and remove partial `UserSession`
consumer changes before continuing the Principal cutover. Stop if descriptor or consumer tests show
either response shape is only partially present. Rollback is the prior coordinated candidate; no
data rollback is required.

## Verification

```bash
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./api/web-api/api ./api/web-api/internal/service/user ./pkg/clients/web ./pkg/authenticators ./protos
git diff --check
```

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol contract and security reviews, followed by a combined evidence panel.
- Outcome: Approved
- Resolved findings: Removed weaker duplicate assignment loaders. Public Authentication producers
  now use the same canonical Organization and Project loaders as Principal resolution and propagate
  loader failures. Added canonical-filter and fail-closed query regressions.
- Open questions: None
- Approval reference: Repository owner directed that the existing response remain unchanged on
  2026-11-13.
