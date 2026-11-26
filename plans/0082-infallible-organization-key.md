# Infallible Organization Credential Key Plan

## Change Contract

- Outcome: Make Organization key derivation infallible and reduce the temporary issuer config
  method to direct JWT delegation.
- Acceptance: Organization derivation returns detached `[]byte` from raw configured text; Web has
  no accessor error branch; JWT issuer behavior and validation remain unchanged.
- Allowed paths: shared config and tests, Web composition and test, Organization authentication
  test, this plan, and RFC 0082.
- Non-goals: Deleting `ServiceAssertionIssuer`, changing `jwt.NewIssuer`, credential migration,
  middleware, Principal, policy, protobuf, schema, or UI behavior.
- Risk: Existing Organization credential fingerprints become invalid. Activation requires zero
  active credentials.

## Implementation

1. Replace Organization derivation with raw-secret HMAC returning only `[]byte`.
2. Remove Web and test error plumbing, preserve detached-domain-key coverage, and add an opaque
   input vector proving the accessor does not validate configuration. Prove the resolver rejects a
   fingerprint created with RFC 0081's decoded-root Organization subkey before accepting the new
   raw-text-derived fingerprint.
3. Remove duplicate destination validation from `ServiceAssertionIssuer` while retaining direct JWT
   errors. Test blank, whitespace-padded, and NUL destinations through the convenience method.
4. Run focused and race tests, scoped finalization, and independent authentication review.
