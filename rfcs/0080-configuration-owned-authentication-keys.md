# RFC 0080: Configuration-Owned Authentication Keys

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0079 ownership of authentication-root validation and key derivation only

## Context

RFC 0079 places root-secret validation and fixed key derivation in
`pkg/authentication/secret.go`. `config.AppConfig` immediately wraps those functions and is the sole
production source of the configured secret. The repository owner requires this logic and its tests
to live directly with configuration, without an extra authentication source file or forwarding
layer.

## Decision

Delete `pkg/authentication/secret.go` and `secret_test.go`. Move the exact canonical-base64
validation, fixed HMAC-SHA256 labels, service key derivation, Organization key derivation, and root
fingerprint calculation into the existing `config.AppConfig` methods in `config/config.go`. Inline
validation in each owning method. Move the deterministic vectors and failure coverage into
`config/config_test.go`.

No key bytes, labels, fingerprints, errors, configuration values, or authentication behavior
change. The exported package functions `authentication.ServiceAuthenticationKey` and
`authentication.OrganizationCredentialFingerprintKey` are intentionally removed. Repository search
finds no production caller; one Organization service test moves to the existing `AppConfig` method.

## Contracts and Ownership

- `config.AppConfig` owns the configured secret and its derived authentication keys.
- `pkg/authentication/jwt` continues to own assertion signing and verification.
- Existing callers continue using `ServiceAuthenticationKey`,
  `OrganizationCredentialFingerprintKey`, and `ServiceAssertionIssuer` as `AppConfig` methods with
  unchanged signatures and outputs.

## Alternatives Rejected

- Keep the forwarding package: rejected because configuration is already the sole production owner.
- Introduce a new helper or utility package: rejected because it recreates the unnecessary layer.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The two exported `pkg/authentication` derivation functions are intentionally removed. No production repository caller uses them. Existing `AppConfig` method signatures and outputs remain identical; one test caller migrates to that owner. |
| Failure and recovery | Invalid secrets return the same centralized error. Revert the ownership move to recover. |
| Security and privacy | Key validation, derivation labels, fingerprints, and redaction behavior remain identical. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | N/A. The functions remain stateless and return detached byte slices. |
| Operations and observability | N/A. Configuration and logged fingerprints remain identical. |

## Rollout and Rollback

Deploy with the existing RFC 0079 atomic authentication cutover. No independent rollout step or
compatibility window is introduced. Rollback restores the previous file ownership with identical
runtime behavior.

## Verification

```bash
test ! -e pkg/authentication/secret.go
test ! -e pkg/authentication/secret_test.go
! rg -n 'authentication\.(ServiceAuthenticationKey|OrganizationCredentialFingerprintKey)' --glob '*.go' .
go test -count=1 ./config ./pkg/authentication ./api/web-api/internal/service/organization
go test -race -count=1 ./config
git diff --check -- config/config.go config/config_test.go pkg/authentication/secret.go pkg/authentication/secret_test.go
just agent-finalize "config/config.go,config/config_test.go,pkg/authentication/secret.go,pkg/authentication/secret_test.go,api/web-api/internal/service/organization/authentication_test.go,rfcs/0080-configuration-owned-authentication-keys.md,plans/0080-configuration-owned-authentication-keys.md"
```

## Review and Approval

- Challenger: Independent GPT-6.1 Sol and GPT-6 Astra review
- Outcome: Approved
- Resolved findings: Added the intentional exported-function removal contract, the Organization test
  consumer migration, repository cleanup search, and downstream package verification.
- Open questions: None
- Approval reference: Repository owner explicitly directed removal of `secret.go` and movement into `config` on 2026-11-14
