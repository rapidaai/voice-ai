# RFC 0068: Infallible Principal Resolver Constructors

- Status: Accepted
- Owner: Authentication
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0059 for User, Project, and Organization resolver constructor signatures and dependency validation only

## Context

User, Project, and Organization principal resolver constructors accept dependencies that are
assembled directly by deployable wiring. They currently use `validator.NonNil` and return an error
solely for missing constructor dependencies. This adds impossible error branches to the Web API
authenticator factories and `cmd/web`, while runtime `Resolve` already fails closed when a resolver
or dependency is malformed.

The Service principal resolver is different: its constructor parses and validates a cryptographic
audience and trusted issuer/key registry and therefore has a real configuration failure path.

## Decision

Make `NewUserPrincipalResolver`, `NewProjectPrincipalResolver`, and
`NewOrganizationPrincipalResolver` return only their typed resolver pointer. Remove constructor
dependency validation and construct the resolver directly from the provided services.

Make Web API `NewUserResolver`, `NewProjectResolver`, and `NewOrganizationResolver` return only the
corresponding resolver pointer. Remove the three impossible error branches from `cmd/web`.

Keep every runtime `Resolve` signature as `(authentication.Principal, error)` and preserve its
existing validation, cancellation, credential, dependency, and principal-construction failure
behavior. Keep `NewServicePrincipalResolver` returning an error because cryptographic verifier
configuration remains fallible.

## Contracts and Ownership

- Package `pkg/authentication/resolver` owns the typed resolver constructor signatures and runtime
  fail-closed behavior.
- Package `api/web-api/authenticator` owns construction of concrete Web API authentication-service
  dependencies and exposes infallible typed resolver factories.
- `cmd/web` owns resolver composition order and no longer handles impossible dependency-constructor
  failures.
- RFC 0059 remains authoritative for transport-specific middleware, resolver types, credential
  precedence, runtime resolution, and Service authentication. This RFC supersedes only its User,
  Project, and Organization constructor result and dependency-validation contract.
- Middleware constructor types, credential formats, principal contents, route policy, and runtime
  authentication errors remain unchanged.

## Alternatives Rejected

- Retain constructor errors for defensive nil checks: rejected because deployable wiring constructs
  the dependencies directly and runtime `Resolve` already owns malformed-resolver rejection.
- Remove errors from runtime `Resolve`: rejected because credential verification, cancellation,
  tenant lookup, and principal validation are legitimate request-time failures.
- Make the Service resolver constructor infallible: rejected because trusted issuer and key
  configuration must be validated before accepting assertions.
- Panic on missing dependencies: rejected because it adds a second failure mechanism without a
  production state that the typed deployable wiring can produce.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Go constructor signatures change for User, Project, and Organization resolver callers; all repository callers migrate atomically. Runtime authentication behavior is unchanged. |
| Failure and recovery | Impossible startup error branches disappear. A manually malformed resolver still fails closed from `Resolve` with `AuthenticationResolverInvalidError`. |
| Security and privacy | Credential verification and principal validation remain unchanged; Service verifier configuration remains fail-closed. No credential data is added. |
| Data and migration | N/A. No persisted data, schema, or generated contract changes. |
| Concurrency and resources | N/A. No goroutine, connection, or resource lifetime changes. |
| Operations and observability | Three impossible Web API startup panic branches disappear. Runtime denial logging and errors remain unchanged. |

## Rollout and Rollback

Change the three resolver constructors, Web API factories, command wiring, and all compile-time
callers in one candidate. Stop on any changed runtime resolver test, changed Service constructor,
middleware behavior failure, command wiring failure, or unresolved Critical or Major finding.
Rollback restores the constructor result tuples and command error branches together. No data
rollback is required.

## Verification

```bash
go test -count=1 ./pkg/authentication/resolver ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go test -race -count=1 ./pkg/authentication/resolver ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
go vet ./pkg/authentication/resolver ./pkg/middlewares/authentication ./api/web-api/authenticator ./cmd/web
! rg -n 'func New(User|Project|Organization)PrincipalResolver\([^)]*\) \(\*[^,]+, error\)' pkg/authentication/resolver
! rg -n 'func New(User|Project|Organization)Resolver\([^)]*\) \(\*[^,]+, error\)' api/web-api/authenticator
git diff --check
just agent-finalize "rfcs/0068-infallible-principal-resolver-constructors.md,pkg/authentication/resolver/user_principal_resolver.go,pkg/authentication/resolver/user_principal_resolver_test.go,pkg/authentication/resolver/project_principal_resolver.go,pkg/authentication/resolver/project_principal_resolver_test.go,pkg/authentication/resolver/organization_principal_resolver.go,pkg/authentication/resolver/organization_principal_resolver_test.go,pkg/middlewares/authentication/grpc_test.go,pkg/middlewares/authentication/http_test.go,api/web-api/authenticator/authenticator.go,api/web-api/authenticator/principal_authentication_test.go,cmd/web/web.go,cmd/web/authentication_middleware_test.go"
```

Acceptance requires infallible typed constructors for User, Project, and Organization resolvers,
no constructor dependency validation, no Web API startup error branches for those constructors,
unchanged runtime `Resolve` error behavior, unchanged Service resolver construction, passing caller
compilation and behavior tests, explicit runtime rejection tests for nil and typed-nil dependencies,
and no unresolved Critical or Major finding.

## Review and Approval

- Challenger: GPT-6 Astra and GPT-5.6 Sol independent reviews with combined evidence panel
- Outcome: Approved
- Resolved findings: Narrowed supersession of RFC 0059 and required nil plus typed-nil runtime evidence
- Open questions: None
- Approval reference: Repository owner requested infallible principal resolver construction on
  2026-11-13. Reviewers approved digest
  `3b51042de9133d374d6759a26870761a56f3a783f24d18718f9500a7eb6e64aa`.
