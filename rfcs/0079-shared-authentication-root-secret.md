# RFC 0079: Shared Authentication Root Secret

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0078; RFC 0077 decisions for Ed25519 signing, signing-key fingerprints, and service-assertion key configuration only

## Context

RFC 0078 distributes one Ed25519 private-key file to every internal service. The application
configuration already contains one shared `secret`, currently used as the HMAC key for Organization
credential fingerprints. A separate key file and configuration struct add operational state without
creating an independent trust boundary because every service already receives the same signing
authority. The repository owner requires one application secret for both concerns.

## Decision

Use the existing flat `secret` configuration value as the sole authentication root. Remove the
`service_assertion` configuration object, key files, secret mounts, and public/private-key handling.
The production root is a canonical standard-base64 encoding of exactly 32 random bytes. Values that
are blank, malformed, noncanonical, or not exactly 32 bytes fail application startup. Every Go and
Python configuration loader explicitly rejects any top-level `service_assertion` key before its
permissive decoder can discard that legacy configuration.

Base64-decode `secret` to the 32 root bytes, then derive subkeys using those bytes as the HMAC key
and the exact ASCII label bytes shown below as the message, without a terminator:

```text
service authentication key = HMAC-SHA256(root, "rapida/service-authentication/v1")
organization credential key = HMAC-SHA256(root, "rapida/organization-credential-fingerprint/v1")
```

The shared nonsecret parity vector is:

```text
secret = AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=
root fingerprint = Yw3NKWbEM2aRElRIu7JbT_QSpJxzLbLIq8G4WBvXEN0
service authentication key = wYx5CX7jmgJqjl/bJr/ci/AP+10mTp4Tv+lg36l6ZLQ=
service Proof fingerprint = 8oLKar0JrGQAFGH_RhnSupGKZhCBoaDajQciGKCrZrA
organization credential key = HW9KSmgntHXNmGZ3vOiwwBngZKZDQ4Q3gpkShme/KCs=
```

Service assertions use HS256 with the service-authentication subkey. The protected header remains
exactly `alg` and `typ`, with `alg: HS256` and `typ: JWT`. The payload, 55-second issuer lifetime,
60-second verifier bound, exact destination audience, Principal construction, generic internal
Service actor, middleware order, and route-policy behavior remain unchanged from RFC 0077.

The Proof reference remains type `signing_key`; its value is the unpadded base64url SHA-256 digest
of the derived service-authentication subkey. It identifies the active root generation without
exposing secret material.

Organization credential fingerprints continue using HMAC-SHA256 over the raw presented credential,
but use the organization-credential subkey instead of the literal configuration string. There is no
legacy verification fallback. Existing Organization credentials cannot be recomputed because their
raw values are not stored and the repository has no issuer or rotation workflow. Production
activation therefore requires this query to return zero:

```sql
SELECT COUNT(*) FROM organization_credentials WHERE status = 'ACTIVE';
```

If it is nonzero, rollout stops. Credential reissuance requires a separate accepted decision and is
not improvised through SQL in this change.

Repository development YAML contains the public parity-vector root shared by all services. It is not
a production credential. Production deployment replaces the repository YAML with a protected
deployment-owned configuration file at the existing `ENV_PATH` boundary (and Document's existing
mounted `env/document.yaml` boundary). Each process logs only the unpadded base64url SHA-256 root
fingerprint after validation. Activation requires every service fingerprint to match and to differ
from the public development fingerprint above.

## Contracts and Ownership

- `config.AppConfig.Secret` is the only configured authentication root. No additional struct or
  service-authentication configuration key exists.
- `pkg/authentication` owns strict root decoding and the two fixed derivation labels.
- `pkg/authentication/jwt` owns HS256 assertion signing, verification, lifetime, audience, envelope,
  and Proof fingerprint behavior.
- Web Organization authentication receives only the derived Organization credential key. It does
  not own or retain the root configuration string.
- Go commands and the Python Document service derive the service-authentication key at startup and
  keep it immutable in memory for the process lifetime. Only Web derives the Organization credential
  key. Subkeys are never configured, persisted, or logged.
- Platform Infrastructure owns production root generation, distribution, rotation, and emergency
  replacement. Rotation is atomic across every service and Organization credential.

## Alternatives Rejected

- Keep the Ed25519 file beside `secret`: rejected because it preserves duplicate configuration and
  every service already holds the same private key.
- Use the literal root bytes directly for both HMAC purposes: rejected because fixed derivation
  labels provide cross-protocol domain separation without adding configuration.
- Add `service_authentication_secret`: rejected because the owner selected the existing `secret` as
  the single source of truth.
- Accept the old key-file assertion during rollout: rejected because it preserves two trust
  contracts and creates downgrade ambiguity.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Configuration and JWT algorithms break atomically. Every caller and receiver must deploy together. Old EdDSA assertions and `service_assertion` configuration are rejected. |
| Failure and recovery | Invalid roots fail startup. Wrong roots, algorithms, signatures, audiences, claims, and lifetimes fail authentication. Recovery restores the same root to every service or rolls the entire isolated stack back. |
| Security and privacy | Compromise of the root grants platform-wide service assertion forgery and the Organization credential fingerprint key. Domain separation prevents direct subkey reuse but cannot contain root compromise. Assertions remain destination-bound, expire within 60 seconds, and require encrypted production transport. |
| Data and migration | Existing Organization credential fingerprints cannot be migrated without raw credentials. Activation is permitted only when the active-row count is zero, so this change performs no data or schema migration. |
| Concurrency and resources | Each process derives one 32-byte service key once; Web also derives one 32-byte Organization key. No mutable registry, cache, goroutine, or background work is added. |
| Operations and observability | All services share one root generation and Proof fingerprint. Startup logs only the nonsecret root fingerprint. Rotation is coordinated and is blocked while active Organization credentials exist. Secrets and derived keys are never logged. |

## Rollout and Rollback

Take a Web database backup and prove the active Organization credential count is zero. A nonzero
count stops this rollout and requires a separate credential-transition RFC. Generate one 32-byte
production root and place it in each deployment-owned protected service configuration, never in the
repository YAML. Provision those configurations to an isolated green stack. Deploy every caller,
receiver, and Web Organization credential resolver together. Confirm encrypted internal transport,
matching derived-key vectors, HS256-only assertion acceptance, explicit key-file configuration
rejection, and startup root fingerprints that all match but differ from the development fingerprint.
Stop if any condition fails.

Switch service discovery and traffic atomically. Rollback returns traffic to the isolated RFC 0078
stack and restores its key file and previous application secret. The zero-active-credential gate
means rollback has no Organization credential data transition. A compromised new root requires
replacement across every service; rotation remains blocked if active Organization credentials exist.
There is no database schema rollback.

## Verification

Acceptance requires deterministic Go and Python derivation vectors, strict root validation,
different subkeys for each label, HS256 direct and delegated round trips, EdDSA rejection, exact
audience and lifetime checks, Organization credential success with the derived key, old-fingerprint
rejection, explicit legacy-config rejection, public-development-root rejection in production
attestation, a zero-active-credential rollout query, configuration cleanup, and zero secret-file
mounts.

```bash
go test -count=1 ./pkg/authentication ./pkg/authentication/jwt ./pkg/authentication/resolver ./pkg/middlewares/authentication ./config
go test -count=1 ./pkg/clients/... ./cmd/web ./cmd/assistant ./cmd/endpoint ./cmd/integration
go test -count=1 ./api/web-api/config ./api/web-api/internal/service/organization ./api/assistant-api/config ./api/endpoint-api/config ./api/integration-api/config
go test -race -count=1 ./pkg/authentication ./pkg/authentication/jwt ./pkg/authentication/resolver ./pkg/middlewares/authentication
python3 -m pytest api/document-api/tests/authentication api/document-api/tests/middlewares api/document-api/tests/bridges/test_auth_bridge.py api/document-api/tests/tasks
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml -f docker-compose.knowledge.yml config --quiet
! rg -n 'service_assertion:|key_file:|INTERNAL_SERVICE_ASSERTION_KEY_FILE|internal-service-assertion-key|EdDSA|ed25519|TrustedPublicKeys' config cmd pkg/authentication api/document-api/app env docker docker-compose*.yml --glob '!**/*_test.go'
test ! -e api/document-api/tests/fixtures/document_test_ed25519.key
! rg -n '"(private_key|public_key)"' api/document-api/tests/fixtures/service_assertion*.json
! go list -m all | rg '^filippo.io/edwards25519 '
git diff --check
```

## Review and Approval

- Challenger: Independent GPT-6.1 Sol and GPT-6 Astra design review
- Outcome: Approved
- Resolved findings: Added explicit legacy-config rejection, a zero-active-Organization-credential
  activation gate, protected production configuration and nonsecret fingerprint attestation, exact
  KDF encoding and shared vectors, immutable subkey ownership, obsolete dependency cleanup, and
  asymmetric-fixture removal checks.
- Open questions: None
- Approval reference: Repository owner selected the existing `secret` as the single root in conversation on 2026-11-14
