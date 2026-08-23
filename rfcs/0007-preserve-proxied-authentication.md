# RFC 0007: Preserve Delegated Authentication

- Status: Accepted
- Owner: Platform and API teams
- Created: 2026-08-23
- Updated: 2026-08-23
- Reviewers: Authentication and service owners

## Summary

Restore actor-preserving authentication across every service assertion created by the
shared `InternalClient` implementation. `WithAuth`, `WithPlatform`, and `WithHttpAuth`
continue authenticating the caller with a short-lived signed service assertion, and every
new assertion also carries the already authenticated effective actor and tenant scope. Go
gRPC receivers reconstruct the originating `Authentication`, so authorization and audit
behavior remain consistent across those service boundaries.

This follows the repository's pre-2026-08-22 pattern, where service assertions carried
the originating user and tenant scope, while extending it to the current actor-aware
model. Raw browser authorization tokens and arbitrary request headers are not forwarded.

## Context

The UI sends its credential to Web API. Web API authenticates it and creates an
`Authentication` containing the actor, organization, and optional project context.

The shared downstream client methods currently replace request identity with a service
assertion containing only the calling service actor and tenant scope. Go gRPC receivers
consequently authorize and audit the call as the intermediary service rather than the
actor that initiated the request.

Before commit `e7765145` on 2026-08-22, the shared service assertion carried `userId`,
`organizationId`, and optional `projectId` across every service client. The actor-aware
rollout intentionally removed `userId`, causing originating actor attribution to stop at
the first service boundary.

## Goals

- Preserve the originating user, project credential, organization credential, service, or
  system actor in every service assertion created from an authenticated request.
- Continue authenticating each hop with the calling service's signed assertion.
- Preserve organization and optional project authorization scope.
- Keep raw public credentials inside the public boundary that received them.
- Read the immediate calling-service actor from the YAML `service_id` setting.
- Fail closed for malformed, unsupported, or internally inconsistent delegated identity.

## Non-Goals

- Forward browser authorization headers, API keys, cookies, or arbitrary metadata.
- Change public endpoint authentication contracts.
- Change protobuf contracts or database schemas.
- Add a service identity registry or network lookup.
- Support service or system authentication without organization scope.

## Scope and Ownership

### Allowed Paths

- `rfcs/0007-preserve-proxied-authentication.md` - RFC owner
- `rfcs/0007-preserve-proxied-authentication/jsons/**` - coordinator artifacts
- `pkg/types/jwt.go` - signed delegated-actor claims
- `pkg/types/jwt_test.go` - token contract tests
- `pkg/types/service_scope_principle.go` - validated service and delegated identity model
- `pkg/types/service_scope_principle_test.go` - identity reconstruction tests
- `pkg/types/rapida_auth.go` - separate immediate caller and effective actor ownership
- `config/config.go` - YAML-backed service actor identity
- `api/web-api/config/config_test.go` - Web YAML decoding coverage
- `api/assistant-api/config/config_test.go` - Assistant YAML decoding coverage
- `api/endpoint-api/config/config_test.go` - Endpoint YAML decoding coverage
- `api/integration-api/config/config_test.go` - Integration YAML decoding coverage
- `pkg/clients/internal_client.go` - shared service assertion creation
- `pkg/clients/internal_client_test.go` - shared caller tests
- `pkg/middlewares/service_authenticator_grpc_middleware.go` - gRPC reconstruction
- `pkg/middlewares/authentication_middleware_test.go` - authentication boundary tests
- `api/document-api/tests/middlewares/test_jwt_authoriation_middleware.py` - deferred receiver compatibility test
- `api/assistant-api/sip/sip.go` - inject the configured Assistant service actor into SIP routing
- `api/assistant-api/sip/sip_test.go` - verify Assistant service actor configuration wiring
- `api/assistant-api/sip/middleware/route_middleware.go` - construct service-authenticated SIP requests
- `api/assistant-api/sip/middleware/route_middleware_test.go` - SIP route authentication coverage
- `api/assistant-api/sip/registration/manager.go` - own SIP registration service authentication
- `api/assistant-api/sip/registration/owner.go` - use manager-owned service authentication
- `api/assistant-api/sip/registration/record.go` - use manager-owned service authentication
- `api/assistant-api/sip/registration/register.go` - use manager-owned service authentication
- `api/assistant-api/sip/registration/renewal.go` - use manager-owned service authentication
- `api/assistant-api/sip/registration/status.go` - use manager-owned service authentication
- `api/assistant-api/sip/registration/manager_test.go` - registration token-minting regression coverage
- `api/assistant-api/sip/registration/record_test.go` - registration manager service actor fixture
- `env/web.yml`, `env/assistant.yaml`, `env/endpoint.yml`, `env/integration.yml` - local service actor IDs
- `docker/web-api/web.yml`, `docker/assistant-api/assistant.yml`, `docker/assistant-api/assistant.knowledge.yml`, `docker/endpoint-api/endpoint.yml`, `docker/integration-api/integration.yml` - container service actor IDs

### Out-of-Scope Paths

- `ui/src/**`
- service-specific API handlers
- Python Document API authentication middleware
- `protos/**`
- database migrations
- unrelated client transport behavior

## Proposed Design

`InternalClient.createServiceScopeToken` remains the single owner of internal
authentication assertions. `WithAuth`, `WithPlatform`, and `WithHttpAuth` all use it to
create a short-lived JWT for every downstream service call.

The service assertion continues to identify the immediate caller with:

- `actor_type=service`
- `actor_id` from the YAML `service_id` setting
- `iss` from the configured service name
- `aud=rapida-internal`
- `iat` and `exp`

Every call to `createServiceScopeToken` adds the current effective actor using optional
claims whose names are unknown to RFC 0002 receivers:

- `delegated_auth_type`
- `delegated_actor_id`

The outer service actor always identifies the local caller configured by
`service_id`. The delegated actor identifies the effective authenticated actor,
including service and system actors. This keeps the immediate caller separate while
preserving the actor whose rights initiated the operation.

The configured IDs are positive and distinct: Web `9001`, Integration `9004`, Endpoint
`9005`, and Assistant `9007`. The field is independent of `port` even when current local
values match existing service ports.

The existing `organizationId` and optional `projectId` claims remain the delegated tenant
scope. No `userId` claim is emitted because RFC 0002 receivers explicitly reject it.

The exact claim matrix is:

| Effective authentication | Required delegated claims | Forbidden delegated state |
| --- | --- | --- |
| service | `delegated_auth_type=service`, positive `delegated_actor_id` | actor type other than service |
| system | `delegated_auth_type=system`, positive `delegated_actor_id` | actor type other than system |
| user | `delegated_auth_type=user`, positive `delegated_actor_id` | actor type other than user |
| project | `delegated_auth_type=project`, positive `delegated_actor_id`, `projectId` | missing project scope or non-project actor |
| organization | `delegated_auth_type=organization`, positive `delegated_actor_id` | project scope or non-organization actor |

The receiver first validates the calling service assertion. `ServiceScope` retains the
immediate service actor and exposes a separate effective authentication constructor. The
middleware stores the originating actor in `Authentication.ActorValue` and the immediate
service actor in a new `Authentication.CallerValue`. User delegation derives `UserValue`
from the delegated actor ID. Project delegation requires project scope. Organization
delegation rejects project scope. Missing, partial, mismatched, or unsupported delegated
claims are rejected.

When no delegated actor is present, the receiver constructs the existing service
authentication using the calling service actor for backward compatibility.

SIP route and registration work is initiated by Assistant API rather than by a project
credential. Those paths therefore construct service authentication with the Assistant
API `service_id` as the effective actor while retaining the resolved organization
and project scope. They must not label a project ID as a project credential actor ID.

Every caller of the shared `createServiceScopeToken` path, including `WithAuth`,
`WithPlatform`, and `WithHttpAuth`, emits the delegated actor and tenant details. Go gRPC
receivers reconstruct the actor in this change. Python Document API receiver reconstruction
is deferred, and its current compatibility behavior is retained.

Authentication errors used by this flow are package sentinels declared in
`pkg/types/rapida_auth.go`: `ErrInvalidServiceAssertion`, `ErrInvalidDelegatedIdentity`,
`ErrUnsupportedDelegatedAuthentication`, `ErrAuthenticationContextMismatch`,
`ErrServiceNameUnavailable`, `ErrServiceActorUnavailable`, and
`ErrServiceSecretUnavailable`. Callers may use `errors.Is` while wrapped errors retain
specific parsing or signing context.

## Contracts and Compatibility

- Public UI and API credential contracts remain unchanged.
- The internal service JWT gains optional `delegated_auth_type` and
  `delegated_actor_id` claims.
- Receivers accept assertions without delegated claims as ordinary service calls.
- RFC 0002 receivers ignore the new optional claim names and continue treating the call as
  service-authenticated, so mixed versions remain available but do not preserve actor
  attribution until the receiver is updated.
- `Authentication.ActorValue` remains the effective actor. `Authentication.CallerValue`
  identifies the immediate authenticated service when delegation is present.
- RFC 0001 Goal 4 and RFC 0002's prohibition on originating identity are superseded only
  for signed actor and tenant claims. Raw public credentials remain prohibited.

## Failure and Recovery

- Invalid signatures, caller identities, expiry, audience, tenant scope, or delegated
  identity fail closed.
- Exactly one of both delegated identity claims or neither must be present. Partial
  delegated identity fails closed on updated receivers.
- Delegated actor type must match delegated authentication type.
- Every newly created assertion includes a validated delegated actor.
- User ID must match the delegated user actor ID.
- Project and organization requirements are validated before handler execution.
- Context cancellation and deadlines continue through the existing context.

## Security and Privacy

- Only signed identity facts are propagated. Raw user tokens and API keys are not.
- The receiver authenticates the immediate calling service before trusting delegation.
- Delegated identity is limited to the already authenticated actor and tenant scope.
- Service assertion lifetime remains at most five minutes.
- Credentials and signed tokens are never logged.

## Observability

- Existing safe authentication rejection logs remain authoritative.
- Authentication failures distinguish invalid service assertions from invalid delegated
  identity without logging claim values.
- Existing request context and tracing behavior remain unchanged.

## Data and Migration

None.

## Rollout

1. Add a non-zero `service_id` to each Go service YAML configuration.
2. Deploy the updated binaries. During a rolling deployment, old receivers ignore the
   new optional claims and retain service attribution.
3. Verify all Go config loaders read the expected positive `service_id`.
4. Verify that `WithAuth`, `WithPlatform`, and `WithHttpAuth` assertions carry the
   effective actor.
5. Verify user and credential actor attribution across the Go gRPC Web, Assistant,
   Endpoint, and Integration boundaries.
6. Verify the current Python Document middleware accepts assertions containing the new
   optional claims and continues treating them as service-authenticated.

Mixed-version operation remains available in both directions. Actor preservation is
complete only after every receiver is updated.

## Rollback

Rollback the shared Go binaries normally. Older receivers ignore the optional claim names,
so no ordered rollback is required. No persistent data rollback is required. Existing
audit rows retain their recorded actors.

## Alternatives Considered

- Forward public credential headers through every service. Rejected because it spreads
  bearer credentials beyond their public ingress boundary.
- Preserve only user ID as in the legacy token. Rejected because project and organization
  credentials also require durable actor identity.
- Continue attributing downstream work to the intermediary service. Rejected because it
  loses the actor responsible for the original request.

## Testing and Verification

- JWT tests cover each delegated actor type, service-only assertions, and RFC 0002 receiver
  compatibility with unknown optional claims.
- JWT tests reject partial, mismatched, malformed, and unsupported delegated claims.
- Shared client tests prove `WithAuth` emits the originating actor and tenant scope.
- Unary and stream gRPC middleware tests prove reconstruction of the originating
  authentication and preservation of immediate caller identity.
- `WithPlatform` and `WithHttpAuth` tests prove they emit the same delegated actor claims.
- Go config tests prove each service loads a positive, distinct `service_id`.
- Python middleware tests prove optional delegated claims remain compatible while actor
  reconstruction is deferred.
- Multi-hop tests prove a reconstructed user, project, or organization actor is
  re-delegated unchanged across A to B to C.
- Multi-hop service tests prove A to B to C rotates the immediate caller to B while
  preserving A as the effective actor.
- SIP tests prove route and registration authentication uses the configured Assistant
  service actor and succeeds through the shared internal token-minting boundary.
- Forged and mismatched delegated actor claims fail closed.
- Run:
  - `go test ./pkg/types ./pkg/authenticators ./pkg/clients/... ./pkg/middlewares`
  - `go test ./api/web-api/... ./api/assistant-api/... ./api/endpoint-api/... ./api/integration-api/...`
  - `make agent-finalize CHANGED_FILES="pkg/types/jwt.go,pkg/types/jwt_test.go,pkg/types/service_scope_principle.go,pkg/types/service_scope_principle_test.go,pkg/types/rapida_auth.go,pkg/clients/internal_client.go,pkg/clients/internal_client_test.go,pkg/middlewares/service_authenticator_grpc_middleware.go,pkg/middlewares/authentication_middleware_test.go"`
  - `git diff --check`

## Acceptance Criteria

- [ ] User-authenticated work reaches every downstream service with the same user actor.
- [ ] Project-authenticated work reaches every downstream service with the same credential actor.
- [ ] Organization-authenticated work reaches every downstream service with the same credential actor.
- [ ] Service-originated work retains the originating service actor and the immediate caller.
- [ ] Delegated requests retain the immediate calling service separately from the effective actor.
- [ ] Organization and optional project scope are preserved.
- [ ] Raw public credentials are not forwarded.
- [ ] Invalid delegated identity fails closed.
- [ ] Old service assertions without delegated claims remain valid.
- [ ] `WithPlatform` and `WithHttpAuth` emit delegated actor claims without changing receiver code.
- [ ] SIP-originated downstream calls use the configured Assistant service actor and retain tenant scope.
- [ ] Required focused and package-level tests pass.

## Open Questions

None.

## Challenge Resolution

The first challenge approved a narrower raw-credential proxy design after revisions. The
user then clarified that actor propagation must apply to all services and follow the prior
shared service-assertion pattern. Amendment challenge 1 required backward-compatible claim
names, explicit caller/effective-actor separation, isolation to gRPC `WithAuth`, and fuller
mixed-version and middleware coverage. Amendment challenge 2 required an explicit rule
that service and system actors are never delegated, removal of Document API from the
verification scope, and service-only multi-hop actor rotation coverage. The user then
overrode that design and required every token creation path to include the effective
actor, including service and system actors, required the immediate service actor ID to
come from YAML, and required authentication errors to be declared in
`pkg/types/rapida_auth.go`. Amendment 4 records that decision. Implementation review then
found actorless synthetic project authentication in SIP route and registration paths.
Amendment 5 classifies those operations as Assistant service-authenticated while retaining
their organization and project scope.

## Artifact Index

- `jsons/plan.json` - Superseded Assistant-proxy plan.
- `jsons/challenge-round-01.json` - Findings on the superseded proxy design.
- `jsons/challenge.json` - Approval of the superseded proxy design.
- `jsons/confirmation.json` - Approval of the superseded proxy design.
- `jsons/amendment-01-plan.json` - Current all-service delegated-authentication plan.
- `jsons/amendment-01-challenge.json` - Findings on the first amendment draft.
- `jsons/amendment-02-plan.json` - Revised all-service delegated-authentication plan.
- `jsons/amendment-02-challenge.json` - Findings on the second amendment draft.
- `jsons/amendment-03-plan.json` - Final Go gRPC `WithAuth` delegated-authentication plan.
- `jsons/amendment-03-challenge.json` - Approval of the third amendment draft.
- `jsons/amendment-03-confirmation.json` - User approval of the third amendment draft.
- `jsons/amendment-04-plan.json` - Revised all-token-path delegated-authentication plan.
- `jsons/amendment-04-challenge.json` - Approval of the fourth amendment draft.
- `jsons/amendment-04-confirmation.json` - User approval of the fourth amendment draft.
- `jsons/amendment-04-implementation-review.json` - Blocking SIP compatibility finding.
- `jsons/amendment-05-plan.json` - SIP service-actor compatibility correction plan.
- `jsons/amendment-05-challenge.json` - Approval of the SIP compatibility plan.
- `jsons/amendment-05-confirmation.json` - User approval of the SIP compatibility plan.
- `jsons/amendment-05-implementation-review.json` - Test-scope inventory finding.
- `jsons/amendment-06-plan.json` - Test-path inventory correction plan.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-23 | Preserve public actor identity through the Web API Assistant proxy | Platform and API teams | `jsons/plan.json` |
| 2026-08-23 | Replace raw credential forwarding with actor-aware signed delegation across all services | Platform and API teams | `jsons/amendment-01-plan.json` |
| 2026-08-23 | Limit propagation to all Go gRPC calls using `InternalClient.WithAuth`; exclude HTTP, platform, and Python Document paths | User | `jsons/amendment-03-plan.json` |
| 2026-08-23 | Always emit delegated actor details, configure the service actor in YAML, and centralize authentication errors | User | `jsons/amendment-04-plan.json` |
| 2026-08-23 | Treat SIP-originated downstream work as Assistant service-authenticated while retaining tenant scope | Implementation reviewer | `jsons/amendment-04-implementation-review.json` |
| 2026-08-23 | Add the SIP package and registration fixture tests required by repository finalization | Implementation reviewer | `jsons/amendment-05-implementation-review.json` |
| 2026-08-23 | Isolate delegation to gRPC `WithAuth` and separate caller from effective actor | Plan challenger | `jsons/amendment-01-challenge.json` |
