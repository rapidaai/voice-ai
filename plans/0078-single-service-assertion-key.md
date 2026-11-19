# Single Service Assertion Key Plan

## Change Contract

- Outcome: Replace per-service signing keys and receiver public-key registries with one shared
  platform Ed25519 key file.
- Acceptance: Every service signs and verifies through the same validated key file; old trust-list
  configuration fails; JWT and Principal behavior remain unchanged.
- Non-goals: JWT claim changes, middleware reordering, route-policy changes, user authentication,
  mTLS, protobufs, database changes, or UI changes.
- Lifecycle: Governed under accepted RFC 0078.

## Allowed Paths

- `pkg/authentication/jwt/**`, `pkg/authentication/resolver/**`, and adjacent tests
- `pkg/middlewares/authentication/**` constructor consumers and tests
- `config/**`, `cmd/{web,assistant,endpoint,integration}/**`, and adjacent tests
- `api/{web,assistant,endpoint,integration}-api/config/**` exact configuration decoding and tests
- `pkg/clients/**` only where constructor behavior or tests require the new key configuration
- `api/assistant-api/sip/**` only for its local Service resolver construction and adjacent tests
- `api/document-api/app/authentication/**`, `app/configs/auth_config.py`, startup consumers, and
  `app/middlewares/**` configuration consumers and adjacent tests
- `env/**`, `docker/**`, and `docker-compose*.yml` service-assertion configuration
- RFC and plan records

## Exclusions

- JWT header or payload changes
- Authentication Principal, route policy, permissions, or controller changes
- Protobuf, database, UI, and external transport changes

## Implementation

1. Make Go and Python service-assertion configuration contain only `key_file`. Reject old-only,
   mixed old/new, and unknown service-assertion keys before Go's permissive Viper decoding can
   discard them. Keep `x-internal-service-key` as a fixed middleware transport header.
2. Validate the canonical 64-byte private key and derive its public key.
3. Make the verifier accept exactly one public key and remove registry iteration and ambiguity paths.
4. Load the shared key for each command, build its inbound verifier for the local service audience,
   and keep outbound issuers destination-specific. Cover at least two outbound destinations.
5. Mount one shared secret file into all Go and Python services and remove public-key environment
   variables and service-specific key paths.
6. Replace multi-key tests with shared-key round trips and wrong-key rejection.
7. Run RFC verification, independent high-risk review, and scoped finalization.

## Risks and Recovery

- Any service compromise exposes the shared signing key. Recovery rotates the key across every
  service in one isolated stack.
- There is no mixed configuration or rotation overlap. Rollback restores the complete RFC 0077 stack.
- Production activation remains blocked without encrypted internal transport evidence.
