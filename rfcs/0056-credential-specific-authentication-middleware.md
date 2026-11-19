# RFC 0056: Service-Specific Authentication Providers

- Status: Superseded
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-12
- Supersedes: RFC 0055 credential-specific middleware rejection; RFC 0054 middleware orchestration only
- Superseded by: [RFC 0059](0059-transport-specific-authentication-middleware.md)

## Context

The current authentication middleware extracts every supported credential family, selects one
family, dispatches through an aggregate principal resolver, and authorizes the route. This keeps
the final authorization decision deterministic, but gives one component ownership of User,
Project, Organization, and Service authentication.

Deployable services do not all accept the same credential families. Each service needs to install
only its supported authentication methods without allowing provider order to select an identity.
RFC 0054 also requires multiple credential families to be rejected before any resolver, database,
or API call.

RFC 0055 rejected sequential credential-specific middleware because middleware order could become
an identity-selection input. Independent challenge of this RFC also established that a sequential
chain may invoke one resolver before a later middleware discovers a credential conflict.

## Decision

Split ingress processing into three ordered stages:

```text
request
  -> route policy middleware
  -> authentication middleware with service-selected providers
  -> route authorization middleware
  -> controller
```

The route policy middleware performs the single route lookup. Unknown routes deny closed. Disabled
routes return unavailable without resolving credentials. Public routes bypass authentication.
For a protected route, it places the immutable route policy in private request-local state for the
remaining ingress stages.

Define one typed authentication provider for each supported credential family: User, Project,
Organization, and Service. A provider owns:

- detecting whether its credential is present for HTTP and gRPC;
- extracting and validating its transport credential;
- invoking its typed principal resolver;
- returning one immutable `Principal`.

Each deployable constructs one authentication middleware with only the providers it trusts. The
provider list is explicit in command wiring. Provider order is visible and deterministic but never
defines precedence.

For every protected request, the authentication middleware first asks every configured provider
whether its credential is present. No provider may invoke its resolver during this probe. The
middleware then applies exactly one result:

- zero presented providers returns unauthenticated;
- more than one presented provider returns credential conflict before any resolver call;
- exactly one presented provider performs extraction and resolution, then attaches its principal.

A malformed or rejected selected credential returns unauthenticated. A credential family whose
provider is not configured is not an accepted authentication method for that deployable and does
not participate in selection.

The final route authorization middleware receives exactly the route policy and attached principal.
It performs no credential parsing, resolver dispatch, database access, or service call.
Controllers continue to consume only the principal and retain request validation. Services retain
persistence, domain invariants, and transaction ownership.

The aggregate `PrincipalResolver`, generic presented-credential union, central credential-family
switch, and combined authentication-and-authorization middleware are removed after every consumer
uses provider-based authentication. The immutable principal, typed family resolvers,
per-deployable policy files, route-policy schema, and public API contracts do not change.

## Contracts and Ownership

- Each deployable command owns its configured provider list and therefore its accepted
  authentication trust boundary.
- The route policy middleware owns route lookup, unknown-route denial, disabled-route denial, and
  public-route bypass.
- Each provider owns transport detection, credential extraction, typed resolution, and principal
  construction for exactly one credential family.
- The authentication middleware owns provider probing, zero-or-one selection, pre-resolution
  conflict rejection, and principal attachment.
- Private request-local state carries only the selected route policy and authenticated principal.
- The final route authorization middleware owns the static decision from only policy and
  principal.
- The versioned service policy remains the sole route-authorization source of truth.
- Public HTTP, protobuf, credential, principal, and controller contracts do not change.

## Alternatives Rejected

- Keep the current combined middleware and aggregate resolver: rejected because one component owns
  every credential family and every deployable must construct dependencies it may not accept.
- Install one transport middleware per credential family: rejected because a resolver can run
  before a later middleware discovers a conflicting family.
- Use provider order as credential precedence: rejected because adding or reordering a provider
  could silently change the authenticated identity.
- Give every deployable every provider: rejected because it expands each service's accepted trust
  boundary without a current requirement.
- Add a separate credential-conflict middleware: rejected because provider probing already gives
  the authentication middleware one complete pre-resolution view without duplicating credential
  ownership.
- Let providers interpret route actor or permission policy: rejected because authorization policy
  evaluation belongs to the final route middleware.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Valid requests retain the same principal and policy decision. Public routes still bypass credentials. Multiple configured credential families remain invalid and perform no resolver call. |
| Failure and recovery | Unknown, disabled, malformed, rejected, missing, and conflicting authentication paths fail before controller execution. Rollback restores the prior combined middleware and aggregate resolver together. |
| Security and privacy | Each deployable accepts a smaller explicit credential set. Provider order cannot select an identity. Credentials and complete authority sets are never logged. |
| Data and migration | N/A. No persisted representation, migration, or role-assignment behavior changes. |
| Concurrency and resources | Providers use request-local immutable state and caller context cancellation. They create no goroutine or retained resource. |
| Operations and observability | Logs distinguish route-policy denial, authentication failure, credential conflict, and authorization denial without exposing credential values. A conflict performs zero resolver calls. |

## Rollout and Rollback

1. Add route-policy, provider, authentication-orchestration, and route-authorization tests without
   changing command wiring.
2. Add User, Project, Organization, and Service providers over the existing typed resolvers.
3. Wire the Web API with its required Project, Organization, Service, and User providers.
4. Prove provider-order permutations reject every credential conflict before resolver invocation.
5. Remove the aggregate resolver, generic presented-credential types, and combined middleware only
   after repository consumers are zero.

Stop rollout on an uncovered route, a protected route reaching a controller without an authorized
principal, any conflict that invokes a resolver, an order-dependent identity result, or an
unresolved Critical or Major finding. Rollback deploys the prior binary and matching policy. No
data rollback is required.

## Verification

- Each provider has success, missing, malformed, and rejected tests for HTTP, unary gRPC, and
  streaming gRPC.
- Public routes invoke no provider resolver even when credential headers are present.
- Disabled and unknown routes invoke no provider resolver or controller.
- Protected routes without a configured credential return unauthenticated.
- Every pair of configured credential families fails authentication in either provider order and
  invokes zero resolvers.
- A deployable can omit a provider without constructing its resolver.
- Duplicate or nil providers fail middleware construction.
- Route authorization reads only the immutable route policy and principal.
- Web API command tests prove the explicit provider list and absence of the aggregate resolver.

Exact verification commands:

```bash
go test ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go test -race ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go vet ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
! rg -n -g '*.go' '\bPrincipalResolver\b|\bNewPrincipalResolver\b|Presented(User|Project|Organization|Service)Credential|\bcredentialSelection\b' .
just agent-finalize "rfcs/0056-credential-specific-authentication-middleware.md,pkg/authentication/resolver,pkg/middlewares/authentication,api/web-api/authenticator,cmd/web"
```

## Review and Approval

- Challenger: Independent Codex gpt-5-6-sol and Astra gpt-6-astra review panel
- Outcome: Approved
- Resolved findings: Corrected the aggregate-resolver cleanup expression to preserve typed
  resolvers and search all repository Go sources. Replaced sequential credential middleware with
  provider probing so conflicts reject before any resolver call.
- Open questions: None
- Approval reference: Repository owner approved service-selected authentication providers and
  pre-resolution conflict rejection on 2026-11-12 after the independent challenge.
