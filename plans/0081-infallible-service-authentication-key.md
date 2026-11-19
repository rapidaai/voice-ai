# Infallible Service Authentication Key Plan

## Change Contract

- Outcome: Make service-authentication key derivation infallible and independent from configuration
  validation and encoding.
- Acceptance: Go exposes `ServiceAuthenticationKey() []byte`; Go and Document API produce the same
  key from raw configured secret bytes; configuration loaders enforce canonical standard-Base64
  encoding of exactly 32 bytes without invoking derivation; middleware and JWT behavior otherwise
  remain unchanged.
- Allowed paths: shared config derivation and tests, centralized validator and tests, Go service
  composition callers and tests, all four Go service config loaders and tests, SIP and
  internal-client callers/tests, Go JWT parity tests, Document API service-assertion
  derivation/configuration/tests and shared fixtures, this plan, and RFC 0081.
- Non-goals: Organization credential derivation, Principal, middleware order, route policy,
  protobuf, schema, UI, JWT claims, lifetime, or audience behavior.
- Risks: Mixed deployments cannot authenticate. Deployment and rollback are atomic across internal
  services.

## Implementation

1. Replace the Go accessor with raw-secret HMAC derivation returning only detached key bytes.
2. Move the canonical 32-byte Base64 rule to Go and Python configuration boundaries, remove
   accessor-based validation, and simplify all Go callers.
3. Remove unsafe root-secret fingerprint logging and update wiring tests. Derived-key Proof
   fingerprints remain owned by JWT authentication.
4. Apply the identical derivation to Document API and move required-secret validation to its config
   model.
5. Record that direct `AppConfig` construction bypasses loader validation and update internal-client
   tests for that explicit contract.
6. Refresh both shared Go/Python JWT fixtures and Go parity tests, then run focused tests, race
   tests, scoped finalization, and independent authentication review.
