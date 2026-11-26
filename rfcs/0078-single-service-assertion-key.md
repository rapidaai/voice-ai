# RFC 0078: Single Service Assertion Key

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0077 decisions for per-service key distribution, multi-key verification, per-key attribution, revocation, and rotation only; RFC 0077 remains authoritative for the JWT envelope, audience, expiry, Principal, and route-policy contracts

## Context

RFC 0077 gives each internal service a private signing key and gives each receiver a list of trusted
caller public keys. Route policy intentionally treats every verified internal key as the same
`internal/1` Service caller. The repository owner requires one simpler platform credential and no
public-key registry.

## Decision

Use one platform-wide Ed25519 private-key file for every internal caller and receiver. The file
contains one canonical base64-encoded 64-byte Ed25519 private key. Callers sign destination-bound
assertions with that key. Receivers derive the one public key from the same validated private key and
verify assertions with it.

Configuration contains only:

```yaml
service_assertion:
  key_file: "/run/secrets/internal-service-assertion-key"
```

Remove `trusted_public_keys`, all service-specific assertion key paths, and every
`*_SERVICE_PUBLIC_KEY` environment variable. The JWT header, claims, lifetime, destination binding,
Principal construction, middleware order, and route policy remain unchanged. There is no legacy
configuration or multi-key receiver. Go and Python reject both old-only and mixed old/new
`service_assertion` configuration. The transport header is the fixed `x-internal-service-key`; it is
not configurable.

## Contracts and Ownership

- `config` and the Document API configuration own validation of the one key file.
- `pkg/authentication/jwt` owns signing and verification with one key pair. Its verifier accepts one
  public key, not a registry.
- Commands derive their inbound verifier from the same key used by outbound clients.
- Inbound verification uses the local service name as audience. Outbound clients continue creating
  destination-specific issuers.
- Deployment owns mounting the same secret file read-only into Web, Workflow, Endpoint, Integration,
  and Document services.
- Platform Infrastructure owns generation, distribution, access control, rotation, and emergency
  replacement of the shared private key.

## Alternatives Rejected

- One private key per service with receiver trust lists: rejected because the owner prioritizes one
  simple platform credential over independent attribution, revocation, and rotation.
- A central signing service: rejected because it adds a network dependency and availability boundary
  solely to avoid distributing the shared key.
- One public key with service-specific private keys: rejected because one Ed25519 public key verifies
  signatures from only its matching private key.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Configuration breaks atomically. All services must use `key_file`; old trust lists are rejected. The JWT wire format does not change. |
| Failure and recovery | Missing, malformed, inconsistent, or unreadable key files fail startup. Recovery restores or replaces the same key file across every service. |
| Security and privacy | Every service and receiver holds platform-wide signing authority. Compromise of any one process can forge structurally valid delegated User, Project credential, or Organization credential subjects, selected tenant, and effective permissions to every receiver. Recovery requires rotation everywhere. Per-service revocation and signing-key attribution are lost. Assertions remain destination-bound and expire within 60 seconds. Encrypted production transport remains required. |
| Data and migration | N/A. No database or persisted application data changes. |
| Concurrency and resources | Verification performs one Ed25519 signature check. No cache, goroutine, or mutable registry is introduced. |
| Operations and observability | Every assertion has the same signing-key fingerprint. Operators can correlate the platform key but cannot infer the calling service from it. Rotation is an atomic stack operation without an overlap window. |

## Rollout and Rollback

Generate one Ed25519 key file and provision it to every service before deployment. Deploy callers,
receivers, configuration, and secret mounts in one isolated green stack. Stop if any service lacks
the file, derives a different public key, accepts old multi-key configuration, or lacks encrypted
production transport. Validate all internal call paths, then switch traffic atomically.

Rollback traffic to the previous isolated stack, then restore RFC 0077 binaries and per-service key
configuration together. A compromised shared key requires generating a replacement and restarting
every service. There is no data rollback.

## Verification

Acceptance requires direct and delegated round trips, wrong-key rejection, malformed and inconsistent
key rejection, exact audience and expiry checks, Go and Python parity, all service command startup
coverage, Compose secret-mount coverage, and zero trusted-public-key configuration.

```bash
go test -count=1 ./pkg/authentication/jwt ./pkg/authentication/resolver ./pkg/middlewares/authentication ./config
go test -count=1 ./pkg/clients/... ./cmd/web ./cmd/assistant ./cmd/endpoint ./cmd/integration
go test -count=1 ./api/web-api/config ./api/assistant-api/config ./api/endpoint-api/config ./api/integration-api/config
go test -race -count=1 ./pkg/authentication/jwt ./pkg/authentication/resolver ./pkg/middlewares/authentication
python3 -m pytest api/document-api/tests/authentication api/document-api/tests/middlewares api/document-api/tests/bridges/test_auth_bridge.py api/document-api/tests/tasks
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml -f docker-compose.knowledge.yml config --quiet
! rg -n 'trusted_public_keys|TrustedPublicKeys|TrustedServicePublicKeys|SERVICE_PUBLIC_KEY|(web|workflow|endpoint|integration|document)-service-assertion-key' config pkg cmd api/document-api/app env docker docker-compose*.yml --glob '!**/*_test.go'
rg -n 'key_file: .*internal-service-assertion-key' env docker
git diff --check
```

## Review and Approval

- Challenger: Independent GPT-6.1 Sol and GPT-6 Astra design review
- Outcome: Approved
- Resolved findings: Required exact legacy-configuration rejection, fixed-header ownership, complete
  consumer and config-loader scope, local inbound versus destination outbound audiences, satisfiable
  cleanup checks, precise supersession, and explicit platform-wide delegated-impersonation impact.
- Open questions: None
- Approval reference: Repository owner selected one shared key in conversation on 2026-11-14
