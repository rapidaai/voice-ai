# Shared Authentication Root Secret Plan

## Change Contract

- Outcome: Replace the service-assertion key file and Organization fingerprint literal key with two
  subkeys derived from the existing flat application `secret`.
- Acceptance: One canonical 32-byte base64 root configures every service; HS256 assertions and
  Organization credential fingerprints use distinct fixed derivation labels; no key file or new
  config struct remains.
- Non-goals: JWT payload changes, Principal changes, route-policy changes, middleware reordering,
  user/project authentication, protobufs, database schema, UI, Organization credential issuance, or
  production secret-manager implementation.
- Lifecycle: Governed under RFC 0079 after acceptance.

## Allowed Paths

- `pkg/authentication/**`, `pkg/errors/**`, and adjacent tests
- `config/**`, `api/{web,assistant,endpoint,integration}-api/config/**`, and adjacent tests
- `pkg/clients/**`, `pkg/middlewares/authentication/**`, and constructor tests affected by HS256
- `cmd/{web,assistant,endpoint,integration}/**` and command wiring tests
- `api/web-api/authenticator/**`, `api/web-api/api/authentication_resolver*`,
  `api/web-api/router/web.go`, `api/web-api/internal/service/organization/**`, and adjacent tests
- `api/assistant-api/sip/**` only for service-authentication constructor changes and tests
- `api/document-api/app/config.py`, `app/configs/auth_config.py`, `app/authentication/**`, startup
  consumers, middleware consumers, and focused tests
- `api/document-api/tests/fixtures/document_test_ed25519.key` and
  `api/document-api/tests/fixtures/service_assertion*.json` for asymmetric-fixture removal
- `env/**`, `docker/**`, `docker-compose*.yml`, `.gitignore`, and `.dockerignore`
- `go.mod` and `go.sum` only for removing the obsolete Ed25519 point-validation dependency
- RFC and plan records

## Exclusions

- JWT authentication payload or Principal semantics
- Route authorization and permission resolution
- Public APIs, protobufs, database migrations, UI, and unrelated configuration cleanup

## Implementation

1. Add strict canonical base64 root decoding and fixed HMAC-SHA256 derivation vectors to
   `pkg/authentication`; mirror the service derivation in Python.
2. Convert Go and Python service assertion signing and verification from EdDSA to HS256 while
   preserving the closed JWT envelope, destination audience, expiry, and Principal facts.
3. Remove `ServiceAssertionConfig`; explicitly reject the top-level legacy `service_assertion` key
   before permissive Go or Python decoding, validate `secret` during every service configuration
   load, and cover old-only, mixed, and unknown legacy shapes.
4. Derive the Organization credential fingerprint key before constructing Web resolvers. Reject
   fingerprints produced with the old literal secret in regression coverage.
5. Put the public parity-vector root in every repository development config. Remove key-file
   configuration, Compose mounts, local generated key material, and its ignore rules.
6. Emit only the validated root fingerprint during Go and Python startup. Test that all development
   configs match it; production rollout evidence must match across services and differ from it.
7. Update all issuer, verifier, resolver, SIP, internal-client, and Document call sites and tests;
   replace asymmetric Python fixtures with the shared HS256 vector and remove
   `filippo.io/edwards25519` when no production import remains.
8. Record the zero-active-Organization-credential query as an activation gate. Do not add an
   unowned SQL reissue path or modify credential data.
9. Run the RFC commands, scoped finalization, and independent high-risk implementation review.

## Risks and Recovery

- Root compromise crosses service authentication and Organization credential verification. Fixed
  derivation labels prevent direct subkey reuse but do not reduce root compromise impact.
- Production activation is blocked unless active Organization credential count is zero. A nonzero
  count requires a separate accepted credential-transition design.
- There is no mixed algorithm or configuration window. Rollback restores the full RFC 0078 stack,
  old application secret, and requires no credential data restoration under the zero-row gate.
