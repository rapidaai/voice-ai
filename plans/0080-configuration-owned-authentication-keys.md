# Configuration-Owned Authentication Keys Plan

## Change Contract

- Outcome: Remove `pkg/authentication/secret.go` and place its unchanged logic in `config/config.go`.
- Acceptance: Existing derivation vectors and invalid-secret errors are unchanged; no forwarding
  authentication functions remain; focused and race tests pass.
- Allowed paths: `config/config.go`, `config/config_test.go`, the two deleted secret files,
  `api/web-api/internal/service/organization/authentication_test.go`, this plan, and RFC 0080.
- Non-goals: Cryptographic behavior, configuration values, callers, middleware, Principal, policy,
  protobuf, schema, UI, or deployment changes.

## Implementation

1. Move the two labels and exact inline validation/derivation into the existing `AppConfig` methods.
2. Move vector, invalid-value, domain-separation, and detached-result tests into `config/config_test.go`.
3. Move the sole non-config test consumer to the existing `AppConfig` method, then delete the
   redundant authentication source and test files.
4. Prove no direct authentication-package callers remain, then run focused tests, race tests,
   scoped finalization, and independent review.
