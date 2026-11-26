# RFC 0077: Key-Only Service Assertions

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: Caller-specific Service decisions in RFC 0051, RFC 0052, and RFC 0054; RFC 0057; and the service-assertion contract in RFC 0076

## Context

RFC 0076 authenticates internal requests with Ed25519 JWTs, but its token and configuration
duplicate service identity through issuer names, actor IDs, and key IDs. The destination already
owns an explicit public-key trust list. The repository owner requires the assertion to contain only
the request authentication facts and expiry, with every trusted internal public key receiving equal
service-route access.

## Decision

Keep compact Ed25519-signed JWTs. Do not encrypt them. Encryption with a destination public key would
protect contents but would not authenticate the caller, so it would require an additional signature
and more keys. The signature provides authenticity and integrity, not confidentiality. Production
activation requires an encrypted internal transport owned and verified by Platform Infrastructure.
The repository's plaintext development connections are not production-ready transport evidence.

The protected JWT header contains only `alg: EdDSA` and `typ: JWT`. It has no `kid`. The payload
contains exactly:

- `aud`: the destination service name;
- `exp`: the expiration time; and
- `authentication`: an optional effective Subject, the selected Tenant, and sorted effective
  permission keys.

A delegated assertion carries its non-Service Subject. A direct internal-Service assertion omits the
Subject but preserves its selected Tenant, including the Project used by SIP billing. Permission keys
must be empty for a direct Service assertion. The issuer sets `exp` to its current time plus 55
seconds. The verifier tries every configured unique public key, accepts exactly one valid signature,
requires the exact local audience, requires `now < exp <= now + 60 seconds`, and rejects unknown or
additional claims. The five-second difference is the supported clock-skew budget. The verifier
derives the signing-key fingerprint as SHA-256 over the Ed25519 public key and records the full
unpadded base64url fingerprint as the nonsecret Proof reference. The token contains no issuer, actor,
key ID, issued-at time, or JWT ID.

All verified keys produce the same internal caller class. Its exact server-derived identity is
`ServiceActor{ID: 1, Issuer: "internal"}`. Direct calls use
`Identity{Type: service, ID: 1, Issuer: "internal"}`; delegated calls use the asserted User, Project
credential, or Organization credential Subject with that generic actor. Route policy uses the same
`internal` and `1` values. No policy or domain audit decision distinguishes which configured public
key signed the request. The fingerprint remains available only as Proof and security-log evidence.

Configuration becomes:

```yaml
service_assertion:
  private_key_file: "/run/secrets/service-assertion-key"
  trusted_public_keys:
    - "BASE64_ED25519_PUBLIC_KEY"
```

There is no compatibility receiver. All internal callers, Go destinations, and the Python Document
service change together.

## Contracts and Ownership

- `pkg/authentication/jwt` owns the minimal JWT envelope, Ed25519 signing, verification, expiry,
  audience validation, and signing-key fingerprint.
- `pkg/authentication/resolver` owns conversion of a verified envelope into a direct or delegated
  immutable Principal. The generic internal Service caller is server-derived and never read from the
  token.
- `pkg/middlewares/authentication` owns header extraction and request-context attachment. Its public
  transport order and failure mapping do not change.
- `config` owns the private-key file and destination trust list. It contains no service identity,
  actor, issuer, or key alias.
- Route policy treats every successfully verified internal key as the same Service caller. Existing
  caller-specific entries become `{issuer: internal, actor_id: 1}`.
- Commands continue to choose which Service middleware is installed. Internal clients continue to
  choose the exact destination audience.
- `service_id` may remain where non-JWT SIP or domain audit behavior already owns it. It is not part
  of service-assertion construction or verification.

## Alternatives Rejected

- Encrypt the JWT with the destination public key: rejected because anyone can encrypt with a public
  key and fabricate authorization facts. Sender authentication would still require signing.
- Keep issuer, actor ID, or key ID: rejected because the trusted public key is the authentication
  proof and the owner requires equal privilege for all configured keys.
- Use mTLS inside each application: rejected for this change because the commands multiplex HTTP,
  gRPC, and gRPC-Web on one listener. Application mTLS requires listener and client transport
  redesign rather than a small middleware contract.
- Carry the full Authentication aggregate: rejected because role graphs, organizations, billing,
  and user profiles are not needed for one route decision and can exceed practical header limits.
- Accept both assertion formats: rejected because it preserves duplicate trust contracts and makes
  downgrade behavior ambiguous.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The JWT and configuration formats break. Every internal caller and destination must deploy as one coordinated candidate. |
| Failure and recovery | Unknown claims, invalid signatures, zero or excessive lifetime, wrong audience, malformed Principal facts, or ambiguous key matches fail closed. Rollback restores all callers, receivers, and configuration together. |
| Security and privacy | Ed25519 protects authenticity and integrity but not confidentiality. Encrypted internal transport is a production activation prerequisite. Every trusted key may assert any structurally valid User, Project credential, Organization credential, Tenant, and permission set, and receives equal Service-route privilege. Compromise of any trusted private key therefore grants platform-wide delegated impersonation until the key is removed. Tokens remain valid for at most 60 seconds. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | Verification may try each configured public key. Trust lists are expected to remain small and immutable; no cache or background worker is added. |
| Operations and observability | Startup validates key sizes, uniqueness, and curve points. Rejections log route and key fingerprint only after signature verification, never token contents or private material. Valid assertions may be replayed during their bounded lifetime; request ID, not a removed JWT ID, correlates one request. |

## Rollout and Rollback

Provision each caller private key and each destination public-key list before activation. Build one
isolated green stack containing Go callers, Go receivers, Python caller and receiver parity,
configuration, and policy changes. Its internal service discovery must resolve only green
destinations. Platform Infrastructure must demonstrate encrypted transport for every production
internal hop before activation. Stop if any destination lacks a required public key, any old
assertion field or configuration key remains, transport encryption is unverified, or the green stack
accepts an old assertion. Validate direct tenantless resolver calls, tenant-bearing SIP billing, and
delegated calls before switching traffic atomically. Do not run mixed-version traffic.

Rollback first removes traffic from green and restores isolated service discovery to the previous
stack, then restores the previous binaries and issuer registry configuration together. No data
rollback is required. Remove a compromised public key from every destination trust list and restart
those destinations; previously issued tokens expire within 60 seconds.

## Verification

Acceptance requires tenantless and tenant-bearing direct round trips, delegated round trips, exact
audience binding, expiration, both clock-skew directions and maximum-lifetime rejection, malformed
Principal rejection, unknown-claim rejection, wrong-key rejection, multiple-key verification,
bounded replay behavior, configuration validation, SIP-to-Web billing, Go and Python parity,
middleware success and failure coverage, and zero old assertion fields.

```bash
go test -count=1 ./pkg/authentication/jwt ./pkg/authentication/resolver ./pkg/middlewares/authentication ./config
go test -count=1 ./pkg/clients/... ./cmd/web ./cmd/assistant ./cmd/endpoint ./cmd/integration
go test -count=1 ./api/web-api/api ./api/assistant-api/sip/... ./api/assistant-api/internal/observability/collectors/billing
go test -count=1 ./api/web-api/config ./api/assistant-api/config ./api/endpoint-api/config ./api/integration-api/config
go test -race -count=1 ./pkg/authentication/jwt ./pkg/authentication/resolver ./pkg/middlewares/authentication
python3 -m pytest api/document-api/tests/authentication api/document-api/tests/middlewares api/document-api/tests/bridges api/document-api/tests/tasks
! rg -n 'json:"(iss|iat|jti)"|Header\["kid"\]|TrustedIssuer|ActorID|KeyID|trusted_issuers|signing_key:' pkg/authentication/jwt config env docker api/document-api/app/configs api/document-api/app/authentication/service_assertion.py
git diff --check
```

## Review and Approval

- Challenger: Independent GPT-6.1 Sol and GPT-6 Astra design review
- Outcome: Approved
- Resolved findings: Added tenant-bearing direct assertions, exact generic caller values, full
  delegated-impersonation impact, affected superseded decisions, complete consumer scope, isolated
  green rollout, bounded replay behavior, clock-skew budget, transport-encryption prerequisite, and
  scoped cleanup checks.
- Open questions: None
- Approval reference: Repository owner approval recorded in conversation on 2026-11-14
