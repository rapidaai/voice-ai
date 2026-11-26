# Client-Owned Service Issuer Plan

## Change Contract

- Outcome: Remove the config forwarding method and place issuer construction with the outbound
  internal client.
- Acceptance: No `AppConfig.ServiceAssertionIssuer` method or call remains; issuer validation
  errors and successful assertions remain unchanged.
- Allowed paths: shared config and tests, internal client and tests, this plan, and RFC 0083.
- Non-goals: JWT constructor, key derivation, claims, middleware, Principal, policy, schema, UI, or
  deployment behavior.
- Risk: Missing error propagation could defer a nil issuer failure. The moved invalid-destination
  regression must prove the client returns the original JWT error before issuing.

## Implementation

1. Move invalid-destination coverage from config tests to `NewInternalClient` behavior. Prove the
   original JWT error is returned by both `WithService` and delegated `WithAuth` paths.
2. Construct the issuer directly in `NewInternalClient` and preserve `issuerErr` storage.
3. Delete the config method and its now-unused JWT import.
4. Prove the symbol is absent, then run focused tests, race tests, finalization, and independent
   review.
