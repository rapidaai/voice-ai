# RFC 0057: Authoritative Internal Service Assertions

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0054 service-assertion revalidation and shared internal audience only

## Context

The current internal service assertion uses a shared HMAC audience and carries actor, delegated
identity, and tenant claims. Web API verifies the JWT, then queries User, Project, Organization,
assignment, and role data before constructing a principal. This makes a signed service assertion
an identity hint rather than an authoritative request snapshot and gives JWT verification database
dependencies.

Internal services will later migrate their issuers and clients together. The secure assertion
contract and verifier must exist first without changing active request behavior or accepting the
current assertion format.

## Decision

Add one new service assertion contract under `pkg/authentication/jwt`. Phase 1 is deliberately
unwired. It neither accepts nor generates the existing service-scope JWT.

Each assertion uses Ed25519 and the following closed JSON payload:

```json
{
  "iss": "assistant-api",
  "aud": "web-api",
  "sub": "user:42",
  "tenant": {
    "organization_id": 7,
    "project_id": 11
  },
  "permissions": ["organization.user.read", "project.assistant.read"],
  "iat": 1794528000,
  "exp": 1794528060,
  "jti": "EjRWeJCrze8SNFZ4kKvN7w"
}
```

The claims are:

- `iss`: the issuing service name;
- `aud`: one JSON string containing the exact destination service name. Arrays are rejected;
- `sub`: the effective subject encoded as `<type>:<positive-id>`, where type is `user`,
  `project`, `organization`, or `service` and ID is canonical base-10 in the range 1 through
  `math.MaxInt64`. Signs, leading zeroes, whitespace, extra separators, and unknown types are
  rejected;
- `tenant`: one required object containing both unsigned integer `organization_id` and
  `project_id` fields. Zero encodes an absent selection. Project without Organization is rejected;
- `permissions`: one required JSON string array. It is empty for Service and Organization
  subjects and is strictly increasing with no duplicates for other subjects;
- `iat` and `exp`: a lifetime greater than zero and no longer than 60 seconds;
- `jti`: 16 bytes from `crypto/rand`, encoded as canonical unpadded base64url;
- `kid`: the issuer-owned signing-key identifier in the JOSE header.

Every payload field above is required and unknown payload fields are rejected. Subject and tenant
combinations follow RFC 0054's subject-and-tenant matrix. The issuer constructs the canonical
permission snapshot through the existing principal constructors before signing.

The trusted issuer registry maps each issuer and globally unique key ID to an Ed25519 public key
and one durable Service actor ID. One issuer entry owns all current and rotation keys, so its actor
ID cannot differ between keys. Configuration rejects duplicate issuers, duplicate key IDs,
duplicate public-key material within or across issuers, malformed keys, and conflicting entries.
The registry and issuer copy key bytes during construction. The verifier derives the immediate
actor from this registry. Actor identity is never accepted from JWT claims. The issuer keeps the
corresponding private key; receivers never store issuer private keys or shared signing secrets.
Registry construction uses `filippo.io/edwards25519` because the Go standard library verifier does
not expose canonical prime-subgroup public-key validation.

Verification requires EdDSA, the exact configured destination audience, a registered issuer and
key ID, the exact JWT-ID encoding, and valid subject, tenant, and permission shapes. `iat` and
`exp` are integer Unix seconds. Expiry must be strictly after issued-at and no more than 60 seconds
later. Expiry at or before the verifier clock is rejected. Issued-at may be no more than five
seconds ahead of that clock. Constructors use the system clock and `crypto/rand`; same-package
tests replace those private sources for deterministic boundary and failure coverage.

The verifier is stateless. It validates JWT-ID syntax and length but cannot prove randomness,
uniqueness, or single use. Replaying the same valid assertion to its one destination within its
validity window is accepted. The short lifetime, destination binding, caller policy, and transport
security limit this residual risk.

Cryptographic verification returns immutable verified assertion facts containing the registry
actor, subject, tenant, canonical permissions, issuer, destination, key ID, JWT ID, issued-at, and
expiry. The destination-specific JWT audience remains separate from the principal proof category.
The typed Service principal resolver maps a verified assertion to `Proof.Audience =
AudienceRapidaInternal` and uses `NewDirectPrincipal` for a Service subject equal to the registry
actor or `NewDelegatedPrincipal` for User, ProjectCredential, or OrganizationCredential subjects.
Those constructors enforce the subject, tenant, permission-scope, actor, proof, and authority
matrix.

The receiver performs no database, service, role, assignment, Project, or Organization lookup.
The signed permissions are the immutable request snapshot. Expiry prevents later request admission
with that assertion; it does not terminate a request already admitted. Phase 2 must either refresh
the authoritative snapshot or ensure any forwarded assertion retains the original expiry rather
than extending it. Phase 2 must also require a matching `caller_services` policy for every
delegated internal caller before runtime wiring.

Phase 1 adds the new contract, issuer, verifier, and tests only. Existing runtime wiring remains
unchanged until a later accepted issuer-and-client cutover. The new implementation contains no
legacy parser, dual-format branch, compatibility adapter, or fallback.

## Contracts and Ownership

- `pkg/authentication/jwt` owns the service assertion claims, Ed25519 issuer and verifier, registry,
  structural validation, and immutable verified assertion facts.
- `pkg/authentication/resolver` retains RFC 0054 and RFC 0056 ownership of typed Service credential
  resolution and conversion from verified assertion facts to an immutable principal.
- The issuer registry owns the binding from issuer and key ID to Service actor and public key.
- The issuing service owns its private key and the subject, tenant, and permission snapshot it
  signs.
- Route policy owns allowed immediate callers through `caller_services` in Phase 2.
- Phase 1 changes no active HTTP or gRPC middleware, service configuration, or internal client.
- The existing assertion format is not part of the new contract and receives no compatibility
  support.

## Alternatives Rejected

- Retain HMAC signing: rejected because every verifier holding a shared secret can forge assertions
  for that issuer. Ed25519 separates signing authority from verification.
- Use one shared internal audience: rejected because a token stolen from one destination could be
  replayed to another service.
- Revalidate signed claims through persistence: rejected because the service assertion is the
  authoritative short-lived principal snapshot.
- Add a V1/V2 parser or fallback: rejected because the requested cutover explicitly forbids legacy
  support.
- Wire the receiver before issuers migrate: rejected because every current internal request would
  fail and leave the worktree with known broken production behavior.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Phase 1 is unwired and changes no runtime behavior. Phase 2 is an intentional breaking internal protocol cutover with no mixed-version window. |
| Failure and recovery | Unknown issuer, key, audience, algorithm, malformed claims, expired lifetime, or invalid principal shape denies before controller execution. Phase 1 rollback removes unused code. |
| Security and privacy | Private keys remain issuer-only. Destination audiences prevent cross-service replay. Assertions contain IDs and permission keys but no profile data, credentials, roles, or secrets. |
| Data and migration | N/A. No persisted data or database migration changes. |
| Concurrency and resources | Issuers and verifiers are immutable. Verification creates no goroutine, connection, or retained request state. |
| Operations and observability | Failures expose no token or permission list. JWT ID, issuer, audience, and key ID are nonsecret correlation fields. Phase 2 must monitor issuer and audience rejection counts. |

A trusted issuer can assert any structurally valid subject, tenant, and permission snapshot. Its
private-key compromise therefore grants that authority on routes which allow that issuer through
`caller_services`. Destination binding and route caller policy limit but do not remove this blast
radius. A receiver rejects outstanding assertions as soon as it adopts a registry without their
key. Receivers still using the previous registry accept them only until expiry. Neither path can
terminate admitted requests.

## Rollout and Rollback

1. Implement the unwired contract, issuer, and verifier with focused tests. Keep the existing
   `ServicePrincipalResolver` as the sole Service resolver.
2. Stop Phase 1 if cryptographic, claim-shape, principal-matrix, or immutability tests fail.
3. In a later RFC, inventory every issuer, client, destination, and `caller_services` route.
4. Migrate all issuers and receivers in one coordinated traffic drain, then remove the existing
   assertion implementation and revalidation dependencies.

Phase 1 rollback deletes the unwired implementation. Phase 2 rollback must restore all issuer and
receiver binaries together because no compatibility parser exists.

## Verification

- Ed25519 signing and verification succeeds for direct Service and delegated User, Project, and
  Organization principals.
- The registry-derived actor cannot be overridden by claims, and key material cannot be shared
  across issuers.
- Wrong destination, audience representation or cardinality, issuer, key ID, algorithm, signature,
  lifetime boundary, JWT-ID encoding, canonical subject, tenant shape, or permission scope denies.
- Entropy failure denies issuance. Repeated verification of one otherwise-valid token succeeds
  within its validity window and fails at expiry.
- Permission and tenant inputs and outputs are immutable snapshots.
- No new source imports database, connector, Web API service, or role packages.
- Existing middleware and internal-client tests remain unchanged and passing.

Exact verification commands:

```bash
go test -race ./pkg/authentication/...
go test ./pkg/middlewares/authentication ./pkg/clients/... ./api/web-api/authenticator ./cmd/web
go vet ./pkg/authentication/...
! rg -n 'gorm|connectors|internal/service|role_definition|role_permission' pkg/authentication/jwt/assertion*.go
! rg -n -g '*.go' -g '!**/*_test.go' 'authentication_jwt.New(Issuer|Verifier)|VerifyServiceAssertion|IssueServiceAssertion' api cmd pkg/clients pkg/middlewares
just agent-finalize "rfcs/0057-authoritative-internal-service-assertions.md,pkg/authentication/jwt/assertion.go,pkg/authentication/jwt/assertion_test.go,pkg/errors/authentication_principal.go,pkg/errors/authentication_principal_test.go,go.mod,go.sum"
```

## Review and Approval

- Challenger: Independent Codex gpt-6-astra and gpt-5.6-sol review panel
- Outcome: Approved
- Resolved findings: Preserved typed-resolver ownership; separated the destination JWT audience
  from `AudienceRapidaInternal`; closed the wire schema; replaced unverifiable JWT-ID uniqueness
  with exact random encoding and explicit replay semantics; preserved issuer-isolated registry
  invariants; defined clock boundaries, trust blast radius, expiry semantics, and compatibility
  verification; clarified immediate key-removal behavior after the combined evidence panel;
  rejected malformed and non-prime-order Ed25519 keys and ambiguous JSON member forms after the
  implementation review; retained one `ServicePrincipalResolver` instead of adding a parallel
  internal-assertion resolver.
- Open questions: None
- Approval reference: Repository owner approved the Phase 1 direction on 2026-11-13. Independent
  challenge corrections were incorporated before acceptance.
