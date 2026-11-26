# RFC 0082: Infallible Organization Credential Key

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0079, RFC 0080, and RFC 0081 Organization credential subkey input and accessor signature only

## Context

`OrganizationCredentialFingerprintKey` still validates and Base64-decodes configuration before
derivation, despite configuration loaders already owning the canonical secret contract.
`ServiceAssertionIssuer` also validates destination text before delegating to `jwt.NewIssuer`, which
owns the same JWT-domain validation. The repository owner requires these config methods to be
cleaned before the issuer convenience method is deleted separately.

## Decision

`OrganizationCredentialFingerprintKey() []byte` derives
`HMAC-SHA256([]byte(secret), "rapida/organization-credential-fingerprint/v1")`. It performs no
validation, decoding, or error handling. Web configuration loading remains the validity boundary.

`ServiceAssertionIssuer` remains temporarily source-compatible as `(*jwt.Issuer, error)`, but
performs no validation. It delegates directly to `jwt.NewIssuer`, which continues owning audience
and key validation. Deleting this convenience method is explicitly deferred.

## Contracts and Ownership

- Service config loaders remain the sole configuration-validity boundary.
- `AppConfig` key methods only derive detached key bytes from the configured text.
- `pkg/authentication/jwt.NewIssuer` owns issuer validation and its error contract.
- Web Organization authentication receives the derived key and does not validate configuration.

## Alternatives Rejected

- Keep Organization validation in the accessor: rejected because it duplicates loader ownership.
- Make JWT issuer construction infallible: rejected because this cleanup does not change the JWT
  package contract.
- Delete `ServiceAssertionIssuer` now: deferred at the repository owner's direction.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Organization credential fingerprints change because the configured Base64 text is used literally. Existing credentials do not match the new key. Service JWT behavior is unchanged. |
| Failure and recovery | Invalid configuration fails in loaders. Directly constructed `AppConfig` values are intentionally not validated. Rollback restores the decoded-root Organization derivation. |
| Security and privacy | Domain-separated HMAC derivation remains. No secret or derived key is logged. Loader validation remains mandatory before production construction. |
| Data and migration | Activation requires zero active Organization credentials because raw credentials are not stored and existing fingerprints cannot be recomputed. |
| Concurrency and resources | N/A. Derivation remains stateless and returns detached bytes. |
| Operations and observability | N/A. No log or metric behavior changes. |

## Rollout and Rollback

Before deployment, prove `SELECT COUNT(*) FROM organization_credentials WHERE status = 'ACTIVE'`
returns zero. A nonzero result blocks activation. Deploy Web with its existing canonical secret
loader. Rollback restores the previous Organization derivation; no data rollback is required under
the zero-active-credential gate.

## Verification

```bash
go test -count=1 ./config ./cmd/web ./api/web-api/config ./api/web-api/internal/service/organization ./pkg/authentication/jwt ./pkg/clients/...
go test -race -count=1 ./config ./cmd/web ./api/web-api/internal/service/organization
git diff --check
just agent-finalize "config/config.go,config/config_test.go,cmd/web/web.go,cmd/web/authentication_middleware_test.go,api/web-api/internal/service/organization/authentication_test.go,rfcs/0082-infallible-organization-key.md,plans/0082-infallible-organization-key.md"
```

## Review and Approval

- Challenger: Independent GPT-6.1 Sol and GPT-6 Astra review
- Outcome: Approved
- Resolved findings: Added direct rejection coverage for the immediately previous Organization
  fingerprint and invalid-destination parity coverage for the retained issuer convenience method.
- Open questions: None
- Approval reference: Repository owner directed matching cleanup for `OrganizationCredentialFingerprintKey` and temporary cleanup before deleting `ServiceAssertionIssuer` on 2026-11-14.
