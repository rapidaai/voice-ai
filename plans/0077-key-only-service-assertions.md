# Key-Only Service Assertions Plan

## Change Contract

- Outcome: Replace issuer and actor-bearing internal JWTs with minimal Ed25519 assertions containing
  only destination, expiry, and request authentication facts.
- Acceptance: All configured public keys authenticate as the same generic internal Service caller;
  tenantless and tenant-bearing direct calls and delegated calls work; invalid, expired,
  excessive-lifetime, wrong-audience, malformed, and old-format tokens fail closed.
- Non-goals: Application mTLS, public user sessions, role resolution, database changes, or changes to
  non-JWT SIP domain identity.
- Lifecycle: Governed. RFC 0077 must be challenged and accepted before implementation.

## Current Behavior

- `pkg/authentication/jwt` signs `iss`, `aud`, `sub`, tenant, permissions, `iat`, `exp`, and `jti`,
  and selects verification keys through `iss` plus `kid`.
- `config` maps service name and actor ID to rotation keys.
- `pkg/authentication/resolver` reconstructs the request Principal from the verified assertion.
- `pkg/clients` puts the assertion in `x-internal-service-key` for gRPC and HTTP requests.
- Web route policy distinguishes Workflow, Endpoint, and Integration by issuer and actor ID.
- Go and Python development clients use plaintext internal transports. Production activation requires
  separate Platform Infrastructure evidence of encrypted internal transport.

## Allowed Paths

- `pkg/authentication/jwt/**`
- `pkg/authentication/resolver/**`
- `pkg/middlewares/authentication/**`
- `pkg/authentication/authorizer/**` and `pkg/authentication/policy/**` only for the generic internal
  caller contract
- `config/**`, `env/**`, and `docker/**` service-assertion configuration
- `pkg/clients/**` assertion construction and adjacent tests
- `cmd/{web,assistant,endpoint,integration}/**` constructor wiring and adjacent tests
- `api/{web,assistant,endpoint,integration}-api/config/**` configuration fixtures and tests
- `api/assistant-api/sip/sip.go`, `api/assistant-api/sip/middleware/**`,
  `api/assistant-api/sip/registration/**`, billing collector integration, and adjacent tests required
  for tenant-bearing direct Service assertions
- `api/document-api/app/authentication/**`, `api/document-api/app/configs/auth_config.py`,
  `api/document-api/app/bridges/**`, `api/document-api/app/main.py`, `celery_worker.py`,
  `core/indexing_runner.py`, `core/model_runtime/model_manager.py`, `routers/v1.py`,
  `tasks/index_knowledge_document.py`, and adjacent tests
- RFC and plan indexes or exact verification manifests required by repository workflow

## Exclusions

- Protobuf schemas and generated artifacts
- UI and React SDK
- Database migrations
- Access Management role behavior
- Assistant call-context persistence
- Listener, TLS, or service-mesh implementation

## Implementation

1. Replace the JWT claims and constructors with `aud`, `exp`, and authentication facts. Issue for 55
   seconds, compute the proof fingerprint from the signing public key, and verify one exact signature
   against the immutable trust list.
2. Preserve Tenant for direct Service calls. Carry Subject only for delegation and require direct
   Service permissions to be empty.
3. Make the resolver create exact generic values `ServiceActor{ID: 1, Issuer: "internal"}` or
   `Identity{Type: service, ID: 1, Issuer: "internal"}` after verification. Never accept Service
   identity from token claims.
4. Replace issuer-based configuration with one private-key file and a list of trusted public keys.
5. Update internal clients and command resolver construction without changing middleware ordering.
6. Change Web caller-specific policy entries to `{issuer: internal, actor_id: 1}`.
7. Apply the same envelope and configuration contract to Document Python code and every startup,
   worker, bridge, model, router, and task consumer.
8. Update SIP local assertion and billing paths without changing their non-JWT domain audit identity.
9. Delete old-format fields, structures, tests, and configuration without a fallback receiver.

## Verification and Review

- Run the exact RFC commands, tenantless resolver tests, SIP-to-Web billing tests, repeated-token
  tests, and scoped `just agent-finalize` for the final changed-file manifest.
- Challenge the revised draft with both independent model families before owner approval.
- After implementation and verification, run independent high-risk review against the fixed
  working-tree boundary. No Critical or Major finding may remain.
- Record Platform Infrastructure evidence that every production internal hop is encrypted. Without
  that external prerequisite the candidate is not approved for production activation.

## Rollout

- Distribute private keys and destination public-key lists first.
- Deploy all internal callers and receivers into one isolated green stack whose service discovery
  resolves only within green.
- Validate tenantless resolver calls, tenant-bearing SIP billing, delegated calls, and old-format
  rejection before the atomic traffic switch.
- Do not permit mixed old and new assertion traffic.
- Roll back traffic and service discovery first, then binaries and configuration together. There is
  no data rollback.
