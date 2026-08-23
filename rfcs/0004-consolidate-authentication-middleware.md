# RFC 0004: Separate Authentication Middleware by Credential Class

- Status: Accepted
- Date: 2026-08-23
- Owner: Platform API

## Summary

Remove the parallel `NewAuthenticationBoundary*` constructor family and retain one explicit
middleware family for each credential class: user, project, organization, and service. Every
service selects the exact authentication mechanisms it supports by installing only those
middleware constructors in its Gin or gRPC chain.

All four credential classes have Gin, unary gRPC, and streaming gRPC variants. Each exported
constructor contains its authentication behavior directly. There is no dependency/options
container, private wrapper, shared credential classifier, conflict middleware, or shared
response/log helper.

## Motivation

The Boundary middleware combines unrelated credential dependencies behind one constructor and
makes a service's accepted authentication mechanisms less obvious. It also duplicates the
pre-existing credential-specific middleware APIs. Small private wrappers and credential-reading
helpers add indirection without owning a meaningful contract.

Separate middleware keeps policy visible in command wiring: installing user middleware enables
user authentication, installing project middleware enables project-key authentication, and so
on. A service that supports several credential classes composes those explicit middleware in
one chain.

The callback path additionally logs a reconstruction error that can include invalid persisted
or request-derived authentication values. That error payload must not reach logs.

## Design

### Middleware Families

The supported constructors are:

| Credential | Gin | Unary gRPC | Streaming gRPC |
| --- | --- | --- | --- |
| User | `NewAuthenticationMiddleware` | `NewAuthenticationUnaryServerMiddleware` | `NewAuthenticationStreamServerMiddleware` |
| Project | `NewProjectAuthenticatorMiddleware` | `NewProjectAuthenticatorUnaryServerMiddleware` | `NewProjectAuthenticatorStreamServerMiddleware` |
| Organization | `NewOrganizationAuthenticatorMiddleware` | `NewOrganizationAuthenticatorUnaryServerMiddleware` | `NewOrganizationAuthenticatorStreamServerMiddleware` |
| Service | `NewServiceAuthenticatorMiddleware` | `NewServiceAuthenticatorUnaryServerMiddleware` | `NewServiceAuthenticatorStreamServerMiddleware` |

Existing constructor signatures remain unchanged. The organization stream and service Gin
constructors are added for transport parity. The `NewAuthenticationBoundary*` and
`NewCredentialConflict*` constructors are deleted.

### Direct Implementation

Each exported constructor implements its own extraction, validation, authentication, context
assignment, logging, and transport response directly. The implementation does not delegate to a
private middleware constructor or common authentication function. The following helpers are not
present:

- `resolveAuthenticationDependencies`
- `grpcAuthenticationError`
- `logAuthenticationFailure`
- `abortGinAuthentication`
- credential-presence structs or count/class methods
- per-credential Gin reader helpers

Shared string constants remain the single source of truth for generic transport errors and safe,
capitalized log messages.

### Selection and Conflicts

Each middleware reads only its own credential keys. If its credential is absent, it passes
through. If its credential is malformed, unsupported, rejected, or cannot produce its required
audit actor, it fails closed before the handler runs.

After successful authentication, middleware writes one `*types.Authentication` to `types.CTX_`.
Before authenticating a presented credential, it checks whether `types.CTX_` already contains an
authenticated request. If so, it rejects the request as a credential conflict rather than
overwriting the existing identity. This makes conflict rejection independent of middleware
ordering for all enabled credential classes.

Credentials for middleware not installed by a service are ignored at the authentication layer.
Protected handlers still reject requests that have no accepted authenticated identity. This is
the service owner's explicit authentication policy.

### Service Wiring

Middleware order is user, project, organization, then service. The authoritative matrix is:

| Service | Gin | Unary gRPC | Streaming gRPC |
| --- | --- | --- | --- |
| Assistant | user, project, organization, service | user, project, organization, service | user, project, organization, service |
| Web | user, project, organization, service | user, project, organization, service | user, project, organization, service |
| Endpoint | none | user, project, organization, service | user, project, organization, service |
| Integration | none | user, project, organization, service | user, project, organization, service |

Command wiring tests own this matrix. Changing an enabled class or its order requires a separate
reviewed policy change.

### Gin Source Precedence

Gin source precedence remains:

- user authorization token: path, then header, then query;
- user auth ID and selected project ID: header, then path, then query;
- project, organization, and service credentials: header, then query, then path.

Conflicting nonempty values within one credential class follow this precedence and are not
cross-class conflicts.

### Failure and Logging

HTTP failures return status 401 with `Invalid authentication credentials`. gRPC failures return
`codes.Unauthenticated` with the same message. Middleware log messages begin with a capital
letter and contain only the credential class and failure category, never credential values.

`CallbackByContext` and the catch-all callback log a static `Failed to reconstruct call
authentication` event without attaching the reconstruction error.

## Compatibility

Established constructor signatures remain source compatible. Existing credential headers,
query/path precedence, successful authentication context, public-route pass-through, HTTP status,
and gRPC status remain stable. Organization compatibility middleware changes from fail-open to
fail-closed when a presented credential is rejected; this is an intentional security correction.

Command services migrate atomically from the Boundary constructors to an explicit ordered list
of the credential middleware they support. No schema, protobuf, SDK, or UI change is included.

## Verification

- Test every Gin, unary, and stream constructor for absent credentials, successful credentials,
  malformed or rejected credentials, audit-actor failure, and an existing-authentication
  conflict.
- Test composed Gin, unary, and stream chains with two presented credential classes in both
  normal and reversed middleware order; both orders must reject before the handler runs.
- Test exact Gin source precedence with conflicting nonempty path, header, and query values.
- Capture Gin, unary, and stream logs and verify malicious credential values are absent.
- Test command wiring against the authoritative service matrix and order, and reject Boundary or
  standalone conflict middleware.
- Test Assistant callback logs with malicious authentication metadata.
- Run focused middleware, command wiring, and callback tests.
- Run broad backend tests, scoped vet, and `git diff --check`.
- Require GitHub CodeQL and aggregate `CI Success` to pass on the pushed commit before shipping.
- Obtain an independent read-only review of the complete diff before committing.

## Rollback

Revert the implementation commit to restore the previous middleware wiring and behavior. The
change is code-only and requires no schema, data, configuration, or protocol rollback.
