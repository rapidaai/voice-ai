# RFC 0081: Infallible Service Authentication Key

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0079 and RFC 0080 service-authentication secret decoding, validation, accessor signature, and root-fingerprint logging decisions only

## Context

`config.AppConfig.ServiceAuthenticationKey` currently validates and Base64-decodes application
configuration, derives a service key, formats an operational fingerprint, and returns an error. The
configuration loaders also call this accessor only to validate configuration. This mixes
configuration validation, authentication-key derivation, and observability in one method. The
repository owner requires the accessor to derive a key only and return `[]byte` without validation
or encoding behavior.

Go and Document API must derive identical keys because they issue and verify the same internal JWT
contract. This is therefore an atomic cross-service authentication change.

## Decision

`ServiceAuthenticationKey() []byte` derives the service key as
`HMAC-SHA256([]byte(secret), "rapida/service-authentication/v1")`. It performs no validation,
Base64 decoding, fingerprint formatting, or error handling. Each service configuration boundary
continues requiring the canonical standard-Base64 encoding of exactly 32 bytes because the same
configured value also feeds unchanged Organization credential derivation. Service authentication
treats those 44 configured text bytes literally rather than decoding them. The runtime cannot prove
entropy. Production must encode 32 independently random bytes and must not use a tracked repository
value.

Go commands use the returned bytes directly. Document API applies the same raw-byte derivation.
Startup root-secret fingerprint logging is removed. JWT Proof references continue using the
nonsecret fingerprint of the derived service key.

The shared development parity vector is:

```text
secret = AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=
service authentication key = veUxBjUOPPj8M4ZdOuRNqMuIzamLmE4S2upg4QeQL40=
service Proof fingerprint = kT0c-JnsGE4x4pLFAL7pVVX4h2L42piDVcwmK2zPWbg
document-api assertion = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkb2N1bWVudC1hcGkiLCJleHAiOjE3OTQ1MjgwNTUsImF1dGhlbnRpY2F0aW9uIjp7InRlbmFudCI6eyJvcmdhbml6YXRpb25faWQiOjcsInByb2plY3RfaWQiOjExfSwicGVybWlzc2lvbnMiOlsib3JnYW5pemF0aW9uLm1lbWJlcnMucmVhZCIsInByb2plY3Qua25vd2xlZGdlLnJlYWQiXSwic3ViamVjdCI6eyJ0eXBlIjoidXNlciIsImlkIjo0Mn19fQ.WoHmCKOSMEraI__2IDKR2kzvc_MB3Wi0zv9RP15BppM
```

Organization credential key behavior is unchanged.

## Contracts and Ownership

- Service configuration loaders own canonical standard-Base64 validation for exactly 32 decoded
  bytes.
- `config.AppConfig.ServiceAuthenticationKey` owns only deterministic service-key derivation.
- Go and Document API use the exact UTF-8 bytes of the configured secret as the HMAC key.
- `pkg/authentication/jwt` continues to own JWT signing and verification validation.
- `pkg/authentication/jwt` continues to own the derived-key Proof fingerprint.
- Organization credential fingerprint derivation and validation remain unchanged.
- Production clients receive loader-validated configuration. Manually constructed `AppConfig`
  values do not gain implicit validation from authentication accessors.

## Alternatives Rejected

- Keep validation in `ServiceAuthenticationKey`: rejected because derivation must not own
  configuration validity.
- Move validation into another derivation helper: rejected because it preserves the same ownership
  problem and adds an abstraction.
- Return a key, fingerprint, and error: rejected because callers only require the key for
  authentication and root-secret fingerprint logging is removed.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Service JWT keys change because the configured Base64 text is now used directly. All Go and Document services must deploy atomically. Organization credentials are unchanged. |
| Failure and recovery | Missing, malformed, noncanonical, or wrong-length configured secrets fail configuration loading. Directly constructed test configuration is not validated by derivation. JWT constructor validation remains fail closed. Rollback restores RFC 0080 derivation across all services. |
| Security and privacy | Fixed HMAC domain separation remains. Loaders enforce format and length but cannot prove entropy. Production must encode 32 independently random bytes. No secret, effective HMAC key, or derived key is logged. Derived-key Proof fingerprints remain available for request correlation. |
| Data and migration | N/A. No persisted representation or schema changes. |
| Concurrency and resources | N/A. Derivation remains stateless and returns detached bytes. |
| Operations and observability | Root-secret startup fingerprint logs are removed because hashing unrestricted raw HMAC keys can expose HMAC's effective key when the input exceeds the hash block size. JWT Proof fingerprints remain unchanged. |

## Rollout and Rollback

Deploy every Go service and Document API in one isolated stack and exercise bidirectional internal
JWT authentication before switching traffic. Do not mix RFC 0080 and RFC 0081 processes because
their derived keys differ. Preflight must prove every service accepts the same canonical
standard-Base64 value, and deployment owners must attest that the production value encodes 32
independently random bytes and is not the tracked development value. Rollback atomically restores
the previous Go and Python derivation.

## Verification

```bash
go test -count=1 ./config ./pkg/validator ./cmd/web ./cmd/assistant ./cmd/endpoint ./cmd/integration ./api/web-api/config ./api/assistant-api/config ./api/endpoint-api/config ./api/integration-api/config ./api/assistant-api/sip ./pkg/authentication/jwt ./pkg/clients/...
go test -race -count=1 ./config ./cmd/web ./cmd/assistant ./cmd/endpoint ./cmd/integration
PIPENV_PIPFILE=api/document-api/Pipfile PYTHONPATH=api/document-api pipenv run pytest api/document-api/tests/authentication/test_service_assertion.py
git diff --check
just agent-finalize "config/config.go,config/config_test.go,pkg/validator/validator.go,pkg/validator/validator_test.go,cmd/web/web.go,cmd/web/authentication_middleware_test.go,cmd/assistant/assistant.go,cmd/assistant/authentication_middleware_test.go,cmd/endpoint/endpoint.go,cmd/endpoint/authentication_middleware_test.go,cmd/integration/integration.go,cmd/integration/authentication_middleware_test.go,api/web-api/config/config.go,api/web-api/config/config_test.go,api/assistant-api/config/config.go,api/assistant-api/config/config_test.go,api/endpoint-api/config/config.go,api/endpoint-api/config/config_test.go,api/integration-api/config/config.go,api/integration-api/config/config_test.go,api/assistant-api/sip/sip.go,api/document-api/app/authentication/service_assertion.py,api/document-api/app/config.py,api/document-api/app/main.py,api/document-api/app/celery_worker.py,api/document-api/tests/authentication/test_service_assertion.py,api/document-api/tests/fixtures/service_assertion.json,api/document-api/tests/fixtures/service_assertion_malformed.json,pkg/authentication/jwt/assertion_test.go,pkg/clients/document/indexer_client_test.go,pkg/clients/endpoint/endpoint_client_test.go,pkg/clients/integration/integration_client_test.go,pkg/clients/workflow/assistant_client_test.go,pkg/clients/workflow/dashboard_client_test.go,rfcs/0081-infallible-service-authentication-key.md,plans/0081-infallible-service-authentication-key.md"
```

## Review and Approval

- Challenger: Independent GPT-6.1 Sol and GPT-6 Astra review
- Outcome: Approved
- Resolved findings: Removed unsafe root-secret fingerprint logging; defined the canonical shared
  configuration boundary; scoped direct-client behavior, both shared JWT fixtures, and Go/Python
  parity verification; corrected Python test execution.
- Open questions: None
- Approval reference: Repository owner directed an infallible `[]byte` accessor with no encoding or validation on 2026-11-14.
