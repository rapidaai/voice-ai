# RFC 0076: Clean Authentication Cutover

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0070, RFC 0074 runtime deferral, and RFC 0058 deferred internal-service cutover

## Context

The repository has two active authentication systems. Web API uses the immutable
`pkg/authentication.Principal` and transport middleware in `pkg/middlewares/authentication`, while
Assistant, Endpoint, and Integration still use `pkg/authenticators`, the root authentication
middleware, mutable `pkg/types.Authentication`, and legacy context accessors. Those services resolve
opaque credentials through the Web API `Authorize` and `ScopeAuthorize` RPCs.

The public protocol also exposes two representations. Web API produces both the deprecated
`web_api.Authentication` graph and the new `authentication.Authentication` aggregate. The React UI,
React SDK, and Document API still consume the deprecated response or scoped response. Removing only
the Go types or only the protobuf messages would leave an uncompilable or nonfunctional deployment.

The repository owner requires a clean migration with no legacy authentication type, middleware,
protobuf response, scoped discriminator, or compatibility adapter.

## Decision

Perform one coordinated cutover in two ordered implementation phases. The phases may be reviewed
separately but are one deployable candidate.

### Phase 1: Runtime Cutover

- Make `pkg/authentication.Principal` the only request authentication identity. Transport middleware
  attaches it once to `context.Context`; controllers and services read that value and pass the
  immutable Principal or explicit domain identifiers. They do not read a mutable compatibility
  authentication object.
- Use only `pkg/middlewares/authentication` for User, Project, Organization, and Service credentials.
  Replace Assistant, Endpoint, and Integration command wiring and delete the legacy root
  authentication middleware and `pkg/authenticators`.
- Add remote Web-backed credential sources for services that do not own the credential database.
  They call the common authentication resolver protocol described below. Each credential-specific
  source returns a validated `authentication.Authentication`; its resolver returns the aggregate's
  immutable Principal. Web uses the same source contract with database-backed implementations.
- Replace the aggregate constructor with `NewDirectAuthentication` and
  `NewDelegatedAuthentication`. Each constructor validates and copies the Role graph, derives the
  selected permissions exactly once, and constructs the Principal internally. Wire decoders and
  local producers use these constructors. No consumer independently derives authorities.
- Replace shared-secret internal service tokens with the destination-bound Ed25519 issuer and
  verifier in `pkg/authentication/jwt`. Issuers and verifier registries are explicit command
  dependencies. The Python Document API implements the same closed claims, issuer, verifier, and
  destination configuration. There is no dual-format receiver or fallback.
- Replace `service_assertion.current.secret` with one Go and Python configuration contract.
  `signing_key` contains `kid` and `private_key_file`; `trusted_issuers` contains issuer name,
  positive actor ID, and one or more `kid` plus `public_key` records. Private-key files contain one
  base64-encoded 64-byte Ed25519 private key. Public keys are base64-encoded 32-byte Ed25519 keys.
  The local issuer is `service_name`, its actor is `service_id`, and every assertion audience is the
  exact destination `service_name`. A receiver accepts multiple unique public keys per issuer for
  rotation. Startup rejects missing or unreadable private-key files, malformed key sizes or base64,
  duplicate issuers, actor conflicts, duplicate key IDs or material, and invalid curve points.
  Secrets never appear in logs or generated configuration output.
- Retain the existing controller-level authorization behavior in Assistant, Endpoint, and
  Integration during this cutover. Service-specific route-policy adoption is a later governed
  change. Web keeps its existing versioned route policy.
- Map legacy controller semantics explicitly. Calls that formerly used `Authorize` require the
  same selected Organization or Project in addition to a valid Principal. Calls that formerly used
  `AuthorizeUser` accept a tenantless onboarding User. Domain audit fields use `Principal.Subject`;
  `Principal.Caller` is reserved for immediate-Service policy and security audit.
- Persist every Principal fact needed by Assistant call contexts: Subject, optional Service Actor,
  selected Tenant, Proof, and sorted authorities. Reconstruction uses the closed Principal
  constructors and never fabricates authentication proof. Existing rows cannot be backfilled
  because they never stored proof or authorities. Before switching stacks, stop blue writers, drain
  all pending or active legacy call contexts, and require the migration precondition to prove none
  remain. A reversible migration then adds the new fields. Terminal legacy rows remain historical
  and unavailable for Principal reconstruction. Green code neither reads nor writes the retained
  legacy columns. Trusted local SIP work issues a direct Service assertion to the Assistant
  audience and verifies it through a command-owned local resolver built from the signing key's
  derived public key. The local resolver is not installed as network trust and no proof fact is
  fabricated.
- Move transport credential names and prefixes out of the legacy `pkg/types` owner into the
  authentication transport package. Delete legacy authentication structs, role projections,
  actor/delegation structs, JWT structs, context accessors, and related errors after their final
  consumers migrate.

### Phase 2: Protocol Cutover

Keep `authentication.proto` as the sole reusable authentication contract and add one explicit
resolver service:

```proto
service AuthenticationResolverService {
  rpc ResolveUserCredential(ResolveUserCredentialRequest)
      returns (ResolveAuthenticationResponse);
  rpc ResolveProjectCredential(ResolveProjectCredentialRequest)
      returns (ResolveAuthenticationResponse);
  rpc ResolveOrganizationCredential(ResolveOrganizationCredentialRequest)
      returns (ResolveAuthenticationResponse);
}

message ResolveUserCredentialRequest {
  string token = 1;
  uint64 userId = 2 [jstype = JS_STRING];
  uint64 selectedProjectId = 3 [jstype = JS_STRING];
}

message ResolveProjectCredentialRequest { string token = 1; }
message ResolveOrganizationCredentialRequest { string token = 1; }

message ResolveAuthenticationResponse {
  Authentication authentication = 1;
}
```

The resolver has no free-form scope selector and no Service method. It is internal-only. Each call
must carry a direct, tenantless Service assertion whose issuer and actor are allowlisted for that
exact resolver route. Web installs Service middleware before its policy middleware and adds a
`service` policy context that accepts only a direct Service subject with an empty tenant and no
delegated Actor. The resolver request's opaque credential is verified inside its credential-specific
handler and is never treated as transport authentication.

Service assertions are verified locally against destination-owned public keys. Each successful
credential response contains the canonical aggregate. Remote consumers call the appropriate direct
or delegated Authentication constructor, which validates the Role graph, derives authorities once,
and exposes the immutable Principal. Billing remains optional and is not required for credential
resolution. Credential-resolution responses are not cached across requests, so revocation and role
changes take effect at the next resolution.

In `web-api.proto`:

- Delete `OrganizationRole`, `ProjectRole`, `FeaturePermission`, `AuthBillingPlan`, the deprecated
  `Authentication`, `ScopedAuthentication`, `ScopeAuthorizeRequest`, and
  `ScopedAuthenticationResponse` messages.
- Delete the legacy `ScopeAuthorize` RPC after all remote resolvers use the common resolver service.
  Retain `Authorize` as the Web application session-refresh RPC with its existing request and route.
  It returns only `AuthenticateResponse.session`; it is not used by internal credential resolvers.
- Delete `AuthenticateResponse.data = 3` and reserve both tag `3` and name `data`. Keep
  `AuthenticateResponse.session = 5` and make it the sole successful authentication payload.
- Remove legacy role projection fields from Organization responses and reserve their removed tags
  and names. Membership and role data come from `authentication.Authentication`.
- Keep Web-owned login, registration, activation, social authentication, and token issuance RPCs.
  Raw credentials and issued tokens remain outside the reusable Authentication aggregate.

Update Web producers, the React SDK, UI state, and Document API to use
`session.authentication`. The UI persisted authentication store increments its version and drops
the deprecated state shape during rehydration. Regenerate every Go, Python, JavaScript, and
TypeScript artifact from the two authoritative proto source checkouts. Remove legacy authorization
caches instead of carrying their serialized values or revocation delay forward.

## Contracts and Ownership

- `pkg/authentication` owns immutable Principal and Authentication values, validation, permission
  derivation, and service assertion cryptography.
- `pkg/middlewares/authentication` owns credential extraction, one-time Principal attachment, and
  transport failure mapping. It does not own credential persistence or domain authorization.
- Web Access Management owns credential verification and the canonical Authentication aggregate
  returned by the resolver service.
- `authentication.proto` owns reusable authentication messages and remote credential resolution.
  `web-api.proto` owns Web application requests, token issuance, and response envelopes only.
- Commands own which credential middleware is installed and all issuer/verifier configuration.
- Each destination-specific internal client owns an issuer constructed with its target service name
  as audience. Generic client methods never choose or rewrite an audience at request time.
- The Python Document command owns its Ed25519 issuer, verifier registry, destination audience, and
  request-context mapping. Its wire contract is identical to the Go implementation and is covered
  by bidirectional fixtures.
- Assistant call-context persistence owns a complete, reconstructable Principal snapshot. The
  authentication package remains the only owner of Principal validation.
- Controllers own request validation and existing domain admission behavior. This RFC does not move
  validation into services and does not introduce helper or utility abstractions.
- There is no compatibility guarantee for removed legacy Go or protobuf authentication contracts.
  Removed protobuf tags and names are reserved to prevent unsafe reuse.

## Alternatives Rejected

- Delete legacy Go before adding remote resolution: rejected because three services would lose the
  only path that verifies opaque Web-owned credentials.
- Delete old protobuf messages first: rejected because generated clients, UI login, Document API,
  and internal credential resolution would fail.
- Keep deprecated response field 3 during a compatibility window: rejected because the owner
  requires one authentication representation and no legacy support.
- Return only wire Principal from the resolver: rejected because effective permissions are derived
  from assigned Roles in Authentication and are intentionally not duplicated on wire Principal.
- Keep shared-secret service assertions behind the new middleware: rejected because it preserves a
  second trust contract and receiver fallback.
- Retain the current cross-request authentication caches with a shorter TTL: rejected because any
  cache extends revoked sessions or role assignments beyond the request boundary.
- Add route policies for every service now: rejected because removing duplicate authentication is
  the current requirement. Policy migration changes authorization behavior and needs a separate
  decision.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Legacy Go APIs, protobuf messages, RPCs, generated accessors, and UI state intentionally break. All in-repository consumers move in the same candidate. Removed tags and names are reserved. |
| Failure and recovery | Any invalid remote aggregate, missing issuer key, ambiguous credential, or partial generated artifact fails closed. Deployment stops if any old symbol or consumer remains. |
| Security and privacy | One immutable Principal replaces mutable context projections. Opaque credentials are sent only to Web Access Management. Service assertions become short-lived, audience-bound Ed25519 assertions with destination-owned verification keys. Raw credentials are never stored in Principal or Authentication. |
| Data and migration | Assistant call contexts gain the complete Principal facts required for deterministic reconstruction. Existing snapshots cannot be truthfully backfilled. Blue writers stop and all nonterminal legacy contexts drain before the reversible additive migration. Terminal legacy rows remain historical and unavailable to green reconstruction. Authentication caches are removed rather than migrated. Other schemas are unchanged. |
| Concurrency and resources | Principals and aggregates are immutable value snapshots. gRPC clients retain existing command-owned connection lifetimes. The cutover adds no background worker. |
| Operations and observability | All five backend services must receive matching issuer and verifier configuration before activation. Authentication rejection logs continue to identify transport and route without logging credentials. Missing keys or resolver failures are deployment stop conditions. |

## Rollout and Rollback

Build and verify the complete candidate before deployment. Distribute destination public keys and
issuer private-key references before activation. Bring up an isolated green stack containing Web,
Assistant, Endpoint, Integration, Document, UI, and generated clients. Its internal service
discovery must resolve only green destinations. Stop blue call-context writers, drain all pending or
active legacy contexts, apply the additive Assistant migration only after its zero-row precondition
passes, validate the green stack, and switch traffic once. Do not deploy the final Web binary ahead
of old callers and do not route between blue and green services. Stop on missing key configuration,
resolver unavailability, migration validation failure, generated-source drift, UI session failures,
or any unresolved Critical or Major finding.

Rollback stops green writers, drains or explicitly terminates every nonterminal green call context,
and requires the down migration's zero-row precondition before dropping new snapshot columns. Green
terminal rows remain historical and unavailable to blue reconstruction. Switch traffic and internal
discovery away from green, run the Assistant down migration, and only then restart blue writers.
The retained legacy columns allow blue to resume without interpreting new snapshots. Restore the
preceding binaries, UI bundle, SDK, and key configuration together. There is no authentication cache
payload to translate.

## Verification

Acceptance requires:

- no production reference to the deleted legacy Go types, middleware, authenticators, JWT contract,
  protobuf messages, response field, or RPCs;
- all User, Project, Organization, and Service credential success and failure paths attach exactly
  one valid Principal in Web, Assistant, Endpoint, Integration, and Document;
- remote resolution rejects malformed, mismatched, canceled, and incomplete aggregates;
- direct tenantless Service admission is limited to exact resolver callers, and delegated callers
  are rejected on resolver routes;
- Go and Python commands reject incomplete, duplicate, malformed, or unreadable issuer/verifier
  configuration before opening request listeners;
- Assistant call-context migration and Principal reconstruction preserve Subject audit attribution,
  Service Actor, Proof, Tenant, and authorities, while the migration rejects any nonterminal legacy
  row rather than fabricating missing facts;
- login, registration, activation, social authentication, onboarding, and tenant selection use
  `session.authentication` in Web, UI, SDK, and Document API;
- resolver, middleware, controller, service, generated contract, UI, and race tests pass.

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/...
go test -count=1 ./protos/...
go test -count=1 ./api/web-api/... ./api/assistant-api/... ./api/endpoint-api/... ./api/integration-api/... ./cmd/...
go test -race -count=1 ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/...
(cd sdks/react && npm test -- --runInBand)
(cd ui && yarn test --watchAll=false --runInBand)
(cd ui && yarn checkTs)
(cd ui && yarn build)
python3 -m pytest api/document-api/tests
just ui-browser-test
! rg -n -g '*.go' '\b(PlainAuthPrinciple|AuthenticationPrinciple|SimplePrinciple|UserInfo|ProjectRole|OrganizaitonRole|FeaturePermission|ClaimAuthenticator)\b' api pkg cmd
! rg -n '\b(ScopedAuthentication|ScopeAuthorize|GetOrganizationRole|GetProjectRole|GetFeaturePermission)\b' api pkg cmd ui sdks/react protos
! rg -n 'HS256|CreateServiceScopeToken|ServiceScope|PlainClaimPrinciple' api/document-api/app api pkg cmd --glob '*.py' --glob '*.go' --glob '!**/*_test.go'
git diff --check
just agent-finalize "$(paste -sd, plans/0076-finalize-paths.txt)"
```

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-6.1 Sol contract, security, and rollout reviews
- Outcome: Approved
- Resolved findings: Added domain-owned aggregate construction and remote resolver ownership;
  retained `Authorize` as session refresh; included Python issuer and verifier migration; defined
  direct tenantless Service admission; preserved Subject audit semantics; replaced impossible call
  context backfill with symmetric drain gates; specified exact key configuration, whole-stack
  blue-green rollout, cache removal, generated and browser checks, and a task-owned finalization
  manifest.
- Open questions: None
- Approval reference: Repository owner directed implementation of the clean removal and new
  authentication protocol on 2026-11-14. Both challengers approved pre-metadata RFC digest
  `65727f823e48268f2554d22f6140983d516a1a5f6c5e0d7c6152c934b77bdf46` and plan digest
  `8c613329931831397f689e8e13da079366869e007ccfbc6660b37eaacaa9fe7e`.
