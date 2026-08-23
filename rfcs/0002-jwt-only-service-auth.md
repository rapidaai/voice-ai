# RFC 0002: JWT-Only Service Authentication

- Status: Accepted
- Date: 2026-08-22
- Supersedes: RFC 0001 Phase 3 service/system registry requirements only

## Summary

Internal service delegation uses a short-lived JWT signed and verified with the existing application secret. Only the existing internal client methods `WithAuth`, `WithPlatform`, and `WithHttpAuth` emit this credential. A receiver with the shared secret treats a valid token as trusted service authority for endpoints where its authentication boundary is installed; this RFC does not add a second route allowlist. No runtime identity-registry lookup, service-credential management API, service identity table, system identity table, or per-service public-key lifecycle is required.

## Motivation

The service credential is used only for selected internal API calls. A separate registry adds a network dependency, database ownership, key-management API, generated contracts, and failure modes that are unnecessary for this trust boundary. JWT signature and claim validation provide the required authentication and durable audit actor attribution.

## Contract

The service JWT contains:

- `actor_type=service`
- positive bigint `actor_id`
- non-empty `iss`
- `aud=rapida-internal`
- `iat` and `exp`, with a maximum five-minute lifetime
- positive bigint `organizationId`
- optional positive bigint `projectId`

The caller obtains `actor_id` from `RAPIDA_service_id` and signs with the existing application secret. The receiver verifies HS256, issuer presence, audience, issued-at, expiry, maximum lifetime, actor type, actor ID range, and delegated tenant scope before constructing authentication context. The signed actor and tenant claims are trusted because only the selected services receive that secret; compromise of any holder permits impersonation within this service-authentication boundary and is an explicitly accepted deployment risk.

The token must not contain or forward an originating user ID. Possession of the shared application secret is the service trust boundary. Rotation follows the existing application-secret deployment procedure.

## Scope

Remove:

- Web-owned service and system identity tables and management service.
- `ValidateServiceIdentity` and `ValidateSystemIdentity` RPC contracts and generated SDK output.
- Remote/local registry verifiers, registry response proofs, and startup registry validation.
- Per-service Ed25519 key and key-ID configuration.

Retain:

- Durable `service` actor attribution using `RAPIDA_service_id`.
- Short-lived service JWTs and delegated organization/project scope.
- Existing credential-selection rules that reject ambiguous credentials.
- Audit persistence, migrations, UI actor display, and non-service Phase 3 behavior.

## Failure Behavior

Missing secrets, malformed claims, invalid signatures, wrong algorithms, expired tokens, excessive lifetime, invalid actor IDs, or malformed tenant scope fail closed. No network or database lookup occurs during service authentication.

## Observability

Authentication rejection continues to use the existing safe authentication-failure logging. No registry lookup metrics remain because no registry lookup exists.

## Compatibility and Rollback

This intentionally removes the unshipped registry RPCs and runtime registry implementation. Migration history is not rewritten: `api/web-api/migrations/000012_remove_service_identity_registry.up.sql` drops the service/system identity tables with `IF EXISTS`, so it is safe whether the earlier additive migration ran or not. Its down migration refuses partial reconstruction and directs operators to the same-point database backup.

All callers and receivers must be deployed together while internal traffic is fenced because HS256 and EdDSA tokens are not mutually compatible. Before deployment, operators retain the previous binaries, the complete four-database backup set, every service's Ed25519 private key and key ID, configured service actor ID, optional system actor ID/name, and matching Web registry rows. Rollback restores that database backup and key configuration, starts the previous binaries, verifies registry-based startup validation for every configured service/system identity, and only then resumes traffic.

## Verification

- Go tests cover valid service JWTs, invalid signatures, wrong algorithms, expiry, excessive lifetime, invalid actor IDs, and prohibited forwarded user identity.
- Source scans prove no production reference to registry RPCs, registry verifiers, service/system identity tables, `RAPIDA_SERVICE_PRIVATE_KEY`, `RAPIDA_SERVICE_KEY_ID`, `RAPIDA_SYSTEM_ACTOR_ID`, or `RAPIDA_SYSTEM_ACTOR_NAME`.
- Protobuf and SDK generation/tests prove registry messages and RPCs are absent while audit actor fields remain.
- Existing Phase 3 migration, rollback, Document API, SDK, and UI verification remains required.
- Deployment verification proves all service-token producers and consumers use HS256 before internal traffic is unfenced.
- Full-history verification proves the final Web schema contains neither `service_identities` nor `system_identities`; rollback verification proves the same-point backup restores both tables and their key/ownership rows for the previous binaries.
