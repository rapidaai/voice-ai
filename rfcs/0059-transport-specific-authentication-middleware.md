# RFC 0059: Transport-Specific Authentication Middleware

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0056

## Context

RFC 0056 introduced route preflight, service-selected authentication providers, and final route
authorization. That design rejects credential conflicts before resolver work and prevents
authentication on public, disabled, and unknown routes, but it adds three ingress stages around a
provider layer that only forwards typed credentials to the existing typed resolvers.

The committed middleware before RFC 0056 was easier to compose: HTTP and gRPC installed one
middleware per accepted credential family, followed by the final authorization middleware. The
new immutable `Principal`, typed resolvers, versioned route policy, and authoritative Service JWT
must remain, but the provider and route-policy context layers are not required to preserve those
contracts.

This decision deliberately favors an explicit transport middleware chain. It accepts that
authentication can occur before a later credential conflict or route-policy denial. A durable
decision is required because this changes the ordering and failure precedence accepted by RFC
0056.

## Decision

Install one policy-blind authentication middleware per supported credential family and transport,
followed by one policy middleware. The command wiring is the complete authentication trust
boundary for that deployable.

```text
HTTP or gRPC request
  -> Project authentication, when installed
  -> Organization authentication, when installed
  -> Service authentication, when installed
  -> User authentication, when installed
  -> route policy authorization
  -> controller
```

`pkg/middlewares/authentication` exposes free middleware factories that return the native
transport type:

```go
func ProjectUnary(*resolver.ProjectPrincipalResolver, commons.Logger) grpc.UnaryServerInterceptor
func ProjectStream(*resolver.ProjectPrincipalResolver, commons.Logger) grpc.StreamServerInterceptor
func ProjectHTTP(*resolver.ProjectPrincipalResolver, commons.Logger) gin.HandlerFunc

func OrganizationUnary(*resolver.OrganizationPrincipalResolver, commons.Logger) grpc.UnaryServerInterceptor
func OrganizationStream(*resolver.OrganizationPrincipalResolver, commons.Logger) grpc.StreamServerInterceptor
func OrganizationHTTP(*resolver.OrganizationPrincipalResolver, commons.Logger) gin.HandlerFunc

func ServiceUnary(*resolver.ServicePrincipalResolver, commons.Logger) grpc.UnaryServerInterceptor
func ServiceStream(*resolver.ServicePrincipalResolver, commons.Logger) grpc.StreamServerInterceptor
func ServiceHTTP(*resolver.ServicePrincipalResolver, commons.Logger) gin.HandlerFunc

func UserUnary(*resolver.UserPrincipalResolver, commons.Logger) grpc.UnaryServerInterceptor
func UserStream(*resolver.UserPrincipalResolver, commons.Logger) grpc.StreamServerInterceptor
func UserHTTP(*resolver.UserPrincipalResolver, commons.Logger) gin.HandlerFunc

func PolicyUnary(ServicePolicy, commons.Logger) grpc.UnaryServerInterceptor
func PolicyStream(ServicePolicy, commons.Logger) grpc.StreamServerInterceptor
func PolicyHTTP(ServicePolicy, commons.Logger) gin.HandlerFunc
```

The factories have no construction-error channel. A nil or typed-nil required resolver produces
middleware that rejects every intercepted request with the generic authentication failure without
calling another middleware or controller. A nil or typed-nil policy produces middleware that
rejects every request with the generic authorization failure. Logger may be nil. The same resolver
instance may be installed for HTTP, unary gRPC, and streaming gRPC and must produce equivalent
principals across transports.

Each credential-family middleware owns only its transport extraction and typed resolver call:

1. If its credential is absent, continue without resolver work.
2. If its credential is present and a `Principal` is already attached, reject with the generic
   unauthenticated response. Never replace or merge principals.
3. Validate the family-specific credential shape and call exactly its typed resolver.
4. Attach the returned immutable `Principal` and continue.

HTTP preserves RFC 0054 source precedence within each family. User `authorization` uses path,
header, then query. User `x-auth-id` and `x-project-id` use header, path, then query. Project,
Organization, and Service credentials use header, query, then path. A higher-priority nonblank value
selects the source and lower-priority copies are ignored, even when their values differ. More than
one value for the selected header or query key is ambiguous and returns HTTP 401. A path parameter
has one value by construction. gRPC accepts incoming metadata only and returns `Unauthenticated`
when a credential key has more than one value.

Resolvers remain transport-independent, read-only, idempotent, cancellation-aware, and safe for
concurrent HTTP, unary gRPC, and streaming gRPC use. Resolver operations must not perform audit,
authorization, or other externally visible state changes. Existing typed resolver names and
construction contracts remain unchanged, including RFC 0058's sole
`resolver.ServicePrincipalResolver` ownership.

The final policy middleware is the sole route-policy owner. It derives the canonical HTTP route or
gRPC full method, performs the policy lookup, and applies these outcomes:

- public shadows any attached principal and permits the controller without an observable
  principal;
- unknown or invalid policy denies without invoking the controller;
- disabled returns unavailable;
- a protected route without a principal returns unauthenticated;
- an attached principal that fails actor, tenant-context, caller, Project-token, or permission
  checks returns permission denied;
- successful authorization invokes the controller with the immutable principal attached.

Credential middleware never reads route policy. Consequently, a supplied malformed credential on
a public route returns unauthenticated. A valid credential may be resolved on a public route, but
the policy middleware shadows the principal before controller invocation so public controllers
cannot acquire optional-identity behavior. Unknown and disabled routes may perform authentication
before the policy middleware denies them. When multiple installed credential families are
presented, the first matching resolver may perform one invocation, potentially containing several
backend reads, before a later middleware rejects the conflict. Middleware order may affect work
and timing, but never selects an accepted identity or permits controller execution.

Transport failure mapping is exact:

- credential validation, non-context resolver rejection or failure, or credential conflict returns
  HTTP 401 or gRPC `Unauthenticated`;
- caller cancellation or deadline returns gRPC `Canceled` or `DeadlineExceeded`; HTTP aborts the
  Gin chain without writing an authentication response after the request context is canceled or
  its deadline is exceeded;
- a protected route without a principal returns HTTP 401 or gRPC `Unauthenticated`;
- unknown or invalid policy returns HTTP 403 or gRPC `PermissionDenied`;
- disabled policy returns HTTP 503 or gRPC `Unavailable`;
- an attached but unauthorized principal returns HTTP 403 or gRPC `PermissionDenied`.

Authentication failure terminates the chain before policy evaluation, including on public,
unknown, and disabled routes. Caller cancellation and deadline are never converted into an
authentication rejection. Assurance-dependent routes remain disabled under RFC 0054 until a
separate accepted contract adds typed assurance facts to the principal and policy.

A service omits an authentication family by not installing its HTTP or gRPC middleware. An
uninstalled family does not participate in authentication. Web API continues to omit Service
authentication until the accepted Ed25519 issuer and receiver cutover supplies trusted key
configuration. This decision does not restore any legacy token or receiver.

Remove the exported `Provider`, aggregate authentication `Middleware`, `RouteMiddleware`,
`AuthorizationMiddleware`, route-policy request context, provider constructors, and functional or
configuration options for authentication selection.

## Contracts and Ownership

- Each deployable command owns the installed and ordered HTTP, unary gRPC, and streaming gRPC
  middleware and therefore its accepted authentication trust boundary.
- Each credential-family middleware owns transport detection, extraction, validation, one typed
  resolver call, and principal attachment for exactly one credential family.
- Each typed resolver owns credential verification, backend access projection validation, and
  immutable principal construction without transport or policy knowledge.
- A private attachment operation validates the principal and returns an error when a principal is
  already attached. Credential middleware uses only this write-once operation. The existing
  `ContextWithPrincipal(context.Context, authentication.Principal) context.Context` signature
  remains only as a direct-controller test seam; it returns the original context on invalid or
  duplicate attachment and has no production caller. A second credential family denies the
  request.
- The final policy middleware solely owns route lookup and the authorization decision.
- The versioned service policy remains the sole route-authorization source of truth.
- Controllers consume the attached principal, validate typed requests and redundant tenant
  bindings, and pass the principal to services. They do not read route policy or select an
  authorization alternative.
- Services retain persistence, domain invariants, and transaction ownership.
- Public request and response schemas, protobuf, JWT, policy schema, principal shape, and
  controller authorization ownership do not change.
- Failure precedence intentionally changes from RFC 0056 as recorded in the Decision section.

## Alternatives Rejected

- Retain RFC 0056 providers and three ingress stages: rejected because the provider forwards to a
  resolver and forces readers across route, provider, resolver, and authorization layers for one
  request.
- Use one aggregate HTTP or gRPC middleware with a typed resolver set: rejected because it hides
  each service's credential order and accepted methods behind a configuration value.
- Use functional options or a common authentication interface: rejected because both recreate the
  provider abstraction under another name.
- Merge resolvers into transport middleware: rejected because HTTP and gRPC would duplicate
  backend projection validation and principal construction.
- Add policy preflight before authentication: rejected because it restores the split route-policy
  stage. The accepted design instead records strict optional-authentication behavior and
  authentication-before-policy failure precedence.
- Add shared credential preselection: rejected because it restores central credential ownership.
  Sequential conflicts remain fail-closed and resolvers are constrained to read-only behavior.
- Let controllers enforce static route policy: rejected because missing middleware or controller
  checks could expose a protected route.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Successful protected single-credential requests retain the same principal and authorization result. Malformed credentials on public routes now reject. Valid public credentials may perform authentication, but the principal is hidden from the controller. Invalid credentials can take precedence over unknown or disabled route responses. |
| Failure and recovery | Every authentication or authorization failure denies before controller execution. Rollback restores the RFC 0056 middleware stack as one binary change. |
| Security and privacy | Multiple credentials never merge or select an identity, but the first resolver may run before conflict denial. Generic responses and read-only resolvers limit timing and credential-oracle impact. Credentials and authority lists are never logged. |
| Data and migration | N/A. No persisted representation, migration, role assignment, or policy file format changes. |
| Concurrency and resources | Typed resolvers are immutable and concurrency-safe. They retain no request context or transport object and create no goroutine. Conflicting or denied requests may perform one otherwise unnecessary resolver invocation containing multiple backend reads. |
| Operations and observability | Middleware-specific logs identify the credential family and denial stage without credential values. Authentication traffic may reach backends for public, disabled, or unknown routes when credentials are supplied. |

## Rollout and Rollback

1. Add direct HTTP, unary gRPC, and streaming gRPC tests for every credential-family middleware and
   the final policy middleware.
2. Add the free middleware factories over the existing typed resolvers without changing command
   wiring.
3. Replace Web API provider wiring atomically with explicit Project, Organization, and User
   middleware followed by Policy middleware. Do not install Service middleware until its Ed25519
   trust configuration is accepted and available.
4. Remove provider, aggregate middleware, route-policy context, and redundant constructor code
   only after the explicit chain tests pass.
5. Apply the same explicit selection independently when another deployable adopts the new
   principal contract.

Stop rollout if a denied request reaches a controller, a second credential overwrites or merges a
principal, a resolver performs a state change, Web accepts an unconfigured Service credential, or
review retains a Critical or Major finding. Rollback deploys the prior binary and its RFC 0056
middleware wiring. Policy and data rollback are unnecessary because their formats do not change.

The existing `CreateProductUsage` limitation from RFC 0058 remains until Service authentication is
wired with the accepted Ed25519 trust configuration.

## Verification

- Every family has success, absent, malformed, rejected, and cancellation tests for its supported
  HTTP, unary gRPC, and streaming gRPC middleware. HTTP tests preserve the exact source precedence,
  ignore lower-priority copies, and reject multiple values within the selected header or query
  source. gRPC tests reject duplicate metadata values. Cancellation tests require gRPC `Canceled`
  or `DeadlineExceeded`; HTTP aborts without writing an authentication response or invoking the
  controller.
- Each resolver produces the same immutable principal through HTTP and gRPC authentication.
- Every pair of installed credential families in both orders returns unauthenticated, invokes no
  controller, never replaces the first principal, and invokes at most the first resolver.
- A public route without credentials succeeds. A valid credential may be resolved, but HTTP,
  unary gRPC, and streaming gRPC controllers observe no principal. A malformed installed
  credential returns HTTP 401 or gRPC `Unauthenticated` as the intentional strict
  optional-authentication contract.
- Unknown and disabled routes without credential failures return their policy response. Tests
  prove HTTP 401 and gRPC `Unauthenticated` authentication-error precedence when malformed
  credentials are supplied.
- Protected routes without an installed credential principal return unauthenticated.
- Omitted family middleware constructs no resolver and cannot attach a principal.
- Policy middleware alone reads route policy and tests public, disabled, unknown, authenticated,
  permission, and `any_of` decisions.
- Nil and typed-nil resolvers and policies deny every intercepted request without controller or
  downstream middleware invocation.
- Principal attachment rejects invalid principals and existing attachments. Public policy shadows
  an attached principal before controller invocation.
- Repository checks prove `ContextWithPrincipal` has no non-test production caller outside its
  declaration.
- Service middleware tests cover direct and delegated Ed25519 assertions, exact signing-key proof
  propagation, wrong audience, issuer, key, and expiry rejection, and rejection of legacy tokens.
- Command tests prove the exact HTTP, unary gRPC, and streaming gRPC order and the absence of Web
  Service authentication.

Exact verification commands:

```bash
go test ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go test -race ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go vet ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
test ! -e pkg/middlewares/authentication/provider.go
! rg -n -g '*.go' 'type (user|project|organization|service)Provider|\bprovider\b|New(User|Project|Organization|Service)Provider|NewRouteMiddleware|type RouteMiddleware|NewAuthorizationMiddleware|type AuthorizationMiddleware|NewMiddleware|type Middleware|routePolicyContextKey|contextWithRoutePolicy|policyFromContext' pkg/middlewares/authentication api/web-api/authenticator cmd/web
test "$(rg -o -g '*.go' -g '!**/*_test.go' 'ContextWithPrincipal\(' api cmd pkg | wc -l | tr -d ' ')" = "1"
rg -n -g '*.go' '^func ContextWithPrincipal\(' pkg/middlewares/authentication
just agent-finalize "rfcs/0059-transport-specific-authentication-middleware.md,pkg/authentication/resolver,pkg/middlewares/authentication,api/web-api/authenticator,cmd/web"
```

## Review and Approval

- Challenger: Independent gpt-6-astra and gpt-5.6-sol review with combined evidence panel
- Outcome: Approved
- Resolved findings: Preserved RFC 0054 HTTP source precedence and defined same-source duplicates;
  retained RFC 0058 Service resolver ownership; shadowed principals from public controllers;
  removed unsupported assurance and controller-selected authorization; defined exact transport,
  cancellation, nil-dependency, and write-once attachment behavior; expanded Service JWT,
  removal, and compatibility verification.
- Open questions: None
- Approval reference: Repository owner approved the transport-specific credential middleware and
  final policy middleware design on 2026-11-13. Independent cross-model reviewers and the combined
  panel approved the corrected final RFC at SHA-256
  `c949e9ed1c551fca20a0e11e048dff4f123fe17f7ebb2f184392d3f0e7bad62c`.
