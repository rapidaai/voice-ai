# RFC 0058: Remove Legacy Service Assertion Receiver

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0057 deferred legacy-receiver removal and runtime-unchanged rollout only

## Context

RFC 0057 added the authoritative Ed25519 service assertion contract but retained the active HMAC
receiver and added a second unwired resolver. The repository owner requires no legacy receiver and
one resolver for each authentication family.

## Decision

Delete the HMAC and database-backed legacy assertion verifier, its legacy access projection, and
its tests. Keep `ServicePrincipalResolver` as the only Service resolver and change it to consume
`authentication/jwt.Verifier` facts directly. Do not add a parallel internal
assertion resolver.

Web API has no accepted Ed25519 public-key configuration yet. Remove its legacy Service provider
construction, trusted-service receiver secrets, and Service provider from `cmd/web` until the
coordinated issuer, configuration, and client cutover. A service header alone on a protected route
therefore fails authentication rather than falling back to HMAC or database revalidation. Per RFC
0056, an unsupported Service header is ignored when a configured User, Project, or Organization
credential is also present, and public routes bypass credential probing.

This receiver-only change does not remove the existing outbound internal token issuer or the
Assistant, Integration, and Endpoint HMAC receivers. Those issuers and receivers remain explicitly
deferred to the coordinated internal-service cutover requested by the repository owner.

## Contracts and Ownership

- `pkg/authentication/jwt` owns Ed25519 assertion verification and immutable verified facts.
- `pkg/authentication/resolver.ServicePrincipalResolver` solely owns Service principal creation.
- Web command wiring owns the temporary absence of Service authentication.
- No legacy assertion parser, HMAC verifier, compatibility adapter, receiver secret, or database
  revalidation remains in the Web Access Management receiver path.

## Alternatives Rejected

- Retain the HMAC receiver during migration: rejected because it preserves the shared-secret trust
  path that this decision removes.
- Add a second internal assertion resolver: rejected because one resolver must own each
  authentication family.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Current internal service tokens are intentionally rejected by Web API until the Ed25519 cutover. `CreateProductUsage` is unavailable because its policy requires the `workflow-api` Service caller and no direct credential can substitute. Do not deploy this intermediate revision where that billing path must remain available. |
| Failure and recovery | Service-only protected requests fail authentication until the coordinated issuer and receiver cutover. Rollback restores the prior receiver and matching Web provider wiring together. |
| Security and privacy | Removal fails closed and eliminates shared-secret receiver trust and assertion-driven database reads. No credential value is added to logs or responses. |
| Data and migration | N/A because the receiver removal changes no database schema or persisted data. |
| Concurrency and resources | N/A. Removal creates no new state, goroutine, or resource. |
| Operations and observability | Service-only protected requests return the existing unauthenticated response. The later cutover must restore `CreateProductUsage` before production rollout requiring billing ingestion. |

## Rollout and Rollback

The later cutover must add destination public-key configuration, construct the new verifier, wire
the existing Service resolver, and migrate all issuers and clients without a dual-format window.
Do not deploy this intermediate revision where `CreateProductUsage` must remain available. Rollback
restores the prior receiver and matching Web provider wiring together.

## Verification

```bash
go test -race ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/... ./api/web-api/authenticator ./api/web-api/config ./api/web-api/internal/service/user ./api/web-api/internal/service/project ./api/web-api/internal/service/organization ./cmd/web
go vet ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/... ./api/web-api/authenticator ./api/web-api/config ./api/web-api/internal/service/user ./api/web-api/internal/service/project ./api/web-api/internal/service/organization ./cmd/web
test ! -e pkg/authentication/resolver/service_assertion.go
! rg -n -g '*.go' '\b(ServiceAuthenticationService|TrustedService|TrustedServiceKey|VerifiedServiceAccess|VerifyServiceAssertion|DelegatedUserAuthenticationService|DelegatedProjectAuthenticationService|DelegatedOrganizationAuthenticationService|VerifyDelegatedUserAccess|VerifyDelegatedProjectAccess|VerifyDelegatedOrganizationAccess)\b' .
! rg -n -g '*.go' 'InternalAssertion|internalAssertion|internal_assertion' .
! rg -n 'trusted_services|TrustedServices|TrustedServiceConfig|TrustedServiceKeyConfig|AuthenticationConfig' api/web-api/config env/web.yml docker/web-api/web.yml
just agent-finalize "rfcs/0057-authoritative-internal-service-assertions.md,rfcs/0058-remove-legacy-service-assertion-receiver.md,pkg/authentication/jwt/assertion.go,pkg/authentication/jwt/assertion_test.go,pkg/authentication/resolver/service_principal_resolver.go,pkg/authentication/resolver/service_principal_resolver_test.go,pkg/errors/authentication_principal.go,pkg/errors/authentication_principal_test.go,pkg/middlewares/authentication/middleware_test.go,pkg/clients/internal_client_test.go,api/web-api/authenticator/authenticator.go,api/web-api/authenticator/principal_authentication_test.go,api/web-api/config/config.go,api/web-api/config/config_test.go,api/web-api/internal/service/user/authentication.go,api/web-api/internal/service/user/authentication_test.go,api/web-api/internal/service/project/authentication.go,api/web-api/internal/service/project/authentication_test.go,api/web-api/internal/service/organization/authentication.go,api/web-api/internal/service/organization/authentication_test.go,cmd/web/web.go,cmd/web/authentication_middleware_test.go,env/web.yml,docker/web-api/web.yml"
```

## Review and Approval

- Challenger: Independent Codex gpt-6-astra and gpt-5.6-sol review panel
- Outcome: Approved
- Resolved findings: Scoped the no-legacy guarantee to Web Access Management; documented mixed
  credential semantics and the `CreateProductUsage` outage; added removal of Web receiver secrets;
  included every resolver, middleware, client-test, config, and command consumer in verification.
- Open questions: None
- Approval reference: Repository owner reiterated the no-legacy and single-resolver requirements on
  2026-11-13.
