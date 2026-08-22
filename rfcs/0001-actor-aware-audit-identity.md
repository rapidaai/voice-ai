# RFC 0001: Actor-Aware Audit Identity

- Status: Accepted
- Approved pilot: Phase 1C may implement the Endpoint-only authentication boundary; its recorded authorization follow-ups block expansion to other binaries
- Approved cleanup direction: Endpoint middleware creates one request `Authentication` object; controllers stop at `Authorize` and `Scope`; Endpoint services consume its context methods directly without `optionalUserID`, `Require*`, type assertions, or type switches
- Approved repository direction: migrate shared clients, Integration, Web, and Assistant to the same request `Authentication` contract, then delete legacy accessors and capability helpers after repository-wide zero-caller evidence
- Approved amendment direction: use bigint actor IDs and complete schema expansion, conversion, contract rollout, all audited domains, and legacy removal inside one Phase 3 with ordered safety checkpoints
- Date: 2026-08-20
- Owners: Platform and API teams
- Reviewers: Authentication, data, SDK, UI, and service owners

## Summary

Replace the assumption that every `created_by` or `updated_by` value is a user ID
with an actor-aware audit identity.

Every externally exposed mutation of a persistent business resource must record the
durable identity of the actor that performed it. Supported actors include users,
project credentials, organization credentials, internal services, and system jobs.

The migration converts existing numeric user attribution into bigint actor identifiers,
updates every writer, reader, protobuf, SDK, and UI projection, and removes `created_by`,
`updated_by`, `createdUser`, and `updatedUser` before Phase 3 completes. Phase 3 is one
approved execution scope with ordered checkpoints; it is not split into pilot, backfill,
domain-rollout, contract, or cleanup RFC phases.

## Motivation

The shared mutable model currently stores only numeric audit identifiers:

```go
type Mutable struct {
    Status    RecordState
    CreatedBy uint64
    UpdatedBy uint64
}
```

These identifiers are treated throughout the system as user IDs. That model is not
valid for API-key and service-authenticated operations:

- A user-authenticated operation has a user ID.
- A project-key operation has a project credential ID, but no user ID.
- An organization-key operation requires an organization credential ID.
- An internal-service operation requires a stable service identity.
- A background operation requires an explicit system identity.

Using project or organization scope IDs as actor IDs would be incorrect. Scope tells
us where an actor was authorized to operate; it does not identify which credential or
service performed the operation.

## Current State

### Persistence

The shared audit fields are defined in `pkg/models/gorm/audited.go`. They are embedded
across persistent entities in the web, assistant, and endpoint services.

The reviewed migration inventory contains:

- 41 assistant-api tables
- 11 web-api tables
- 10 endpoint-api tables

The reviewed inventory contains 62 legacy-audited tables plus Integration API's
`external_audits` and `external_audit_metadata`, which receive actor identity without
legacy conversion. The 64-table inventory artifact is authoritative for the rollout.

### Authentication

`pkg/types/rapida_auth.go` exposes authentication types for:

- `user`
- `project`
- `organization`
- `service`

However, the common `SimplePrinciple` contract exposes tenant scope and raw token
information, but no durable actor identity.

Project credentials already have generated database IDs in
`api/web-api/internal/entity/organization.go`. That ID is currently discarded when a
project credential is converted into `ProjectScope`.

`ScopedAuthentication` in `protos/artifacts/web-api.proto` returns user, organization,
project, and status information, but does not return the credential that authenticated
the request.

Organization scopes currently have no durable credential identity. Service assertions
carry user, project, and organization scope but no stable calling-service identity.

### Public Contracts

Several API messages expose numeric `createdBy` and `updatedBy` fields. Some also expose
`createdUser` and `updatedUser` projections.

Current behavior is inconsistent:

- Some read and list paths hydrate `createdUser`.
- Many create and update paths return downstream responses without hydration.
- `updatedUser` may be defined but not populated.
- A user lookup represents the current user record, not an immutable event-time snapshot.

## Goals

1. Record the actual durable actor for every externally exposed persistent mutation.
2. Distinguish actor identity from authorization scope and authentication transport.
3. Support user, project credential, organization credential, service, and system actors.
4. Avoid storing or forwarding raw API keys and originating authentication tokens.
5. Remove legacy audit persistence and public contracts before Phase 3 completes.
6. Complete all audited domains in one approved phase rather than leaving permanent mixed audit models.
7. Make missing durable actor identity fail closed before persistence.

## Non-Goals

- Replacing the existing external audit-log subsystem.
- Adding audit fields to health checks, transient invocations, metrics submissions, or
  response-only objects that do not represent persistent business resources.
- Reconstructing historical API-key identity when reliable evidence does not exist.
- Storing an immutable snapshot of complete user or credential metadata on every row.
- Performing a single all-services cutover.

## Terminology

### Actor

The durable principal that performed an operation.

### Scope

The organization and project within which the actor was authorized to operate.

### Authentication Type

The mechanism used to authenticate a request. Authentication type may help derive an
actor but is not itself the actor identity.

### Actor Projection

Optional display information resolved from the durable actor identity, such as user name
or credential name. Projections are best-effort and may change over time.

## Requirements

### Functional Requirements

- User operations record the user ID.
- Project-key operations record the project credential ID.
- Organization-key operations record the organization credential ID.
- Service operations record a provisioned stable service ID.
- Background operations record an explicit system actor.
- Creation and update identities are maintained independently.
- Actor identity is derived on the server and cannot be supplied by the caller.
- Pre-write durable actor resolution is part of mutation authorization and fails closed
  before persistence. Post-write actor display or projection lookup failure is best-effort
  and does not convert an already successful mutation into a failure.

### Compatibility Requirements

- Existing protobuf field numbers and JSON names are never reused or changed.
- Existing `createdBy`, `updatedBy`, `createdUser`, and `updatedUser` remain available only
  during Phase 3 migration checkpoints and are removed in the final versioned contract cutover.
- Removed protobuf field numbers and names are reserved and never reused.
- `createdActor` and `updatedActor` use new field numbers.
- Existing response wrappers, success codes, error codes, and cardinality remain stable.
- Actor-capable protobufs, generated artifacts, SDKs, UI, gateways, and services deploy
  before the final breaking contract cleanup.
- Old application binaries must not run after the final legacy-column removal checkpoint.

### Security Requirements

- Raw API keys are never stored as actor identifiers.
- Raw tokens are not used in database indexes or cache keys.
- Actor display information is tenant-authorized and minimizes personally identifiable
  information.
- Service identity is cryptographically bound to a signed assertion.
- Organization and project IDs are not substituted for missing credential identity.

## Proposed Design

### Actor Contract

Introduce a shared actor identity contract:

```go
type ActorType string

const (
    ActorTypeUser         ActorType = "user"
    ActorTypeProject      ActorType = "project"
    ActorTypeOrganization ActorType = "organization"
    ActorTypeService      ActorType = "service"
    ActorTypeSystem       ActorType = "system"
    ActorTypeUnknown      ActorType = "unknown"
)

type ActorIdentity struct {
    Type ActorType
    ID   uint64
}
```

Actor identifiers are unsigned 64-bit contract values restricted to `1..9223372036854775807`
and persisted as positive PostgreSQL `bigint` values.
Every durable user, credential, service, and system actor therefore has a numeric owned
record. Names, keys, tokens, tenant IDs, UUID strings, and scope IDs are not actor IDs.

Actor type names describe the principal class, not the authorization scope identifier or
the credential implementation. Their identifiers are defined as follows:

| Actor type | Actor ID | Must not use |
| --- | --- | --- |
| `user` | User ID | Session or access-token ID |
| `project` | Project credential ID | Project ID or raw project key |
| `organization` | Organization credential ID | Organization ID or raw organization key |
| `service` | Provisioned numeric service identity ID | Shared secret, token, service name, user ID, or project ID |
| `system` | Registered numeric system identity ID | Empty, job name, or implicit identity |
| `unknown` | `0` | Guessed user, credential, scope, or service identity |

For example, `{type: "project", id: 123}` means project credential `123` performed the
operation. It does not mean project `123` performed the operation.

### Approved Project Actor Decision

For actor type `project`, `ActorIdentity.ID` is the durable project credential ID.
The project ID remains a separate authorization-scope field on the resource or request
context and is never used as the actor ID.

This preserves the ability to distinguish multiple credentials belonging to the same
project, investigate credential-specific activity, and retain attribution across key
rotation or archival. Project credentials must therefore be archived rather than
hard-deleted when historical actor resolution is required.

### Principle Integration

The legacy authentication boundary spreads one authentication decision across multiple
optional middleware and multiple context accessors. Project, user, organization, and
service middleware independently inspect a request, may silently continue, and may write
different values into the same context slot. Controllers then repeat authentication and
scope probing with `GetSimplePrincipleGRPC`, `GetAuthPrinciple`, `RequireUser`, and
`RequireProject`.

Replace that flow with one authentication decision at the transport boundary and one
small controller contract. The public contract must remain readable to open-source
contributors and must not hide context access or scope checks behind utility layers.

```go
type AuthenticationPrinciple interface {
    IsAuthenticated() bool
    Type() AuthType
    Scope(allowed ...AuthType) (AuthenticationPrinciple, error)
}
```

`Scope` accepts one or more authentication types. It returns the same authenticated
principle when its concrete type is allowed and returns an error otherwise. It validates
the caller class; it does not perform resource authorization and does not mutate user,
organization, or project context.

An empty allowed list, an unknown authentication type, or an unauthenticated principle
returns an error. Duplicate allowed types are harmless and do not change the result.

Authentication is read directly from the request context:

```go
func Authorize(ctx context.Context) (AuthenticationPrinciple, error) {
    auth, ok := ctx.Value(CTX_).(AuthenticationPrinciple)
    if !ok || auth == nil || !auth.IsAuthenticated() {
        return nil, ErrUnauthenticated
    }
    return auth, nil
}
```

Do not introduce `authenticationFromContext`, `WithAuthentication`, generic context
helpers, scope utility functions, or wrapper abstractions around this contract. The
authentication middleware writes the value with `context.WithValue`; `Authorize` reads it
directly; controllers call `Scope` directly.

Controllers begin with the same two operations:

```go
auth, err := types.Authorize(ctx)
if err != nil {
    return nil, unauthenticatedError
}

scopedAuth, err := auth.Scope(
    types.AuthTypeUser,
    types.AuthTypeProject,
    types.AuthTypeOrg,
)
if err != nil {
    return nil, permissionDeniedError
}
```

An API lists every authentication type allowed to call it. User-only APIs pass only
`AuthTypeUser`. APIs available to user, project, organization, or service callers list
those types explicitly. Authentication-type checks must not be inferred from incidental
availability of user, organization, or project IDs.

Authentication and resource authorization are separate. After `Scope` succeeds, the
controller or service authorizes the requested organization, project, assistant, or other
resource. The target design does not call `SwitchProject` or otherwise mutate identity
from request parameters during authentication.

The Endpoint pilot temporarily preserves the existing user `x-project-id` and
`SwitchProject` behavior because Endpoint services currently derive project authorization
from that selected context. This is explicit compatibility debt, not the target contract.
It must be replaced by an approved resource-authorization design before the authentication
boundary expands beyond the Endpoint pilot.

The existing narrow `UserIdentityProvider`, `OrganizationContextProvider`,
`ProjectContextProvider`, and `DelegatedContextProvider` contracts remain available to
resource-authorization code during migration. They are not replaced with a new large
authentication interface, fake identity methods, reflection, or generic capability
utilities. Their removal or replacement requires a separately approved authorization
design.

The transport boundary uses one coordinating middleware for each transport shape: gRPC
unary, gRPC stream, and Gin. Each middleware delegates credential verification to the
existing user, project, organization, and service authenticators, but it alone owns:

1. extracting presented credential classes;
2. rejecting missing, malformed, or conflicting credentials for protected routes;
3. selecting exactly one concrete authenticator;
4. attaching exactly one authenticated principle to context; and
5. returning a consistent unauthenticated response on failure.

Health and other intentionally public routes are registered outside protected middleware
groups or are listed explicitly as public methods. Missing credentials must never pass
silently through a protected route.

Each binary must publish and test its public-route inventory before adopting fail-closed
middleware. Phase 1C pilots only Endpoint gRPC and migrates all Endpoint gRPC controllers
as one binary-level unit. Endpoint Gin exposes only `/readiness/` and `/healthz/`, which
remain outside the protected gRPC boundary. The registered `EndpointService` and
`Deployment` gRPC methods are protected. gRPC reflection is not registered by the
Endpoint binary.

Actor identity remains distinct from authentication scope. Once authentication is
established, audit-writing code derives the durable actor from the concrete authenticated
principle. Project IDs and organization IDs remain authorization scope and must not replace
credential actor IDs.

### Project Credentials

Add project credential identity to the project authentication path:

1. Add a credential ID field to `ProjectScope`.
2. Select and map `project_credentials.id` during authorization.
3. Add optional actor type and actor ID fields to `ScopedAuthentication`.
4. Preserve the fields through the web authentication client and middleware.

The raw project credential remains secret and is never persisted as audit metadata.

### Organization Credentials

Web API owns a durable `organization_credentials` entity with a positive bigint primary
key, organization scope, non-secret name, HMAC fingerprint, status, creation/update actor,
and archived timestamp. Credential IDs are allocated by the existing positive ID generator,
must not exceed PostgreSQL bigint range, are never reused, and survive key rotation and
archival. Organization credential authentication returns that entity ID as the actor ID.

The organization ID alone is not a valid credential actor ID.

### Service Identity

Web API owns a durable `service_identities` entity with a positive bigint primary key,
unique non-secret service name, status, signing-key identifier, creation/update actor, and
archived timestamp. IDs use the existing positive generator, are never reused, and remain
stable across signing-key rotation. Service assertions include `actor_type=service`, the
numeric actor ID, issuer, audience, issued-at, expiry, and signing-key ID; receivers verify
signature, issuer, audience, expiry, and positive bigint range before accepting the actor.

Internal delegation creates a new short-lived service assertion containing tenant scope
and actor provenance. It must not forward the originating user or API-key credential.

### System Identity

Web API owns a durable `system_identities` entity with a positive bigint primary key,
unique non-secret job name, owning service, status, creation/update actor, and archived
timestamp. IDs use the existing positive generator and are never reused. Human-readable
names such as `assistant-indexer` remain display metadata and are not persisted as actor
IDs. Worker configuration references the numeric ID, startup verifies its registered
owner, and an empty actor is never treated as a system actor implicitly.

## Persistence Design

The persistence contract uses **actor**, not **scope**. Actor identifies the durable user,
credential, service, or system that performed the operation. Scope identifies the
organization or project where that actor was authorized. A project credential ID must be
stored as `created_actor_id`; it must never be stored in a `scope_id` column or confused
with `project_id`.

Add actor columns to every audited persistent resource table:

```text
created_actor_type varchar(32)
created_actor_id   bigint
updated_actor_type varchar(32)
updated_actor_id   bigint
```

The unexecuted expansion migrations use bigint actor IDs, preserve the legacy columns, and
add database checks for valid type/ID pairs once conversion has populated the rows:

```text
actor_type in (user, project, organization, service, system, unknown)
durable actor type -> actor_id between 1 and 9223372036854775807
unknown actor type -> actor_id is null
null updated actor type -> updated_actor_id is null
```

After conversion validation, each table receives explicit named CHECK constraints for
created and updated actor pairs. One shared PostgreSQL trigger function rejects any update
that changes `created_actor_type` or `created_actor_id`; the migration attaches a named
trigger to each of the 64 actor-audited tables. Application models also mark creation actor
fields create-only, and update statements use explicit column allowlists.

Legacy columns remain during the internal Phase 3 checkpoints. A separate final cleanup
migration drops `created_by` and `updated_by` only after actor backfill is complete, all
writers and readers use actor fields, external contract migration is complete, and source
and database validators prove zero legacy dependency.

### Historical Conversion

Historical non-zero legacy values are converted only as verified legacy user candidates:

```text
created_by > 0 -> created_actor_type = user, created_actor_id = created_by
updated_by > 0 -> updated_actor_type = user, updated_actor_id = updated_by
```

Null, zero, or negative creation values become `unknown` with a null actor ID. A null
legacy update remains a null update actor; zero or negative update values become `unknown`
with a null actor ID. The conversion never infers project credentials,
organization credentials, services, or system jobs from current secrets, tenant IDs,
logs, or present-day ownership.

Conversion runs in bounded, resumable primary-key ranges inside Phase 3 rather than the
schema-expansion transaction. Each table records processed range, converted user rows,
unknown rows, failures, duration, and remaining count. Conversion is idempotent and updates
only rows with null actor fields.

### Write Behavior

| Actor | Actor type | Actor ID |
| --- | --- | --- |
| User | `user` | User ID |
| Project credential | `project` | Project credential ID |
| Organization credential | `organization` | Organization credential ID |
| Service | `service` | Numeric service identity ID |
| System | `system` | Numeric system identity ID |

During the writer checkpoint, user mutations write matching legacy and actor values while
non-user mutations write the actor fields and zero compatibility legacy values. Every new
authenticated mutation resolves a non-zero durable actor before persistence. No scope ID,
credential secret, configured compatibility user, or guessed identity may be written.

### Read Behavior

Actor-capable readers prefer actor fields and temporarily fall back to positive legacy user
IDs only until conversion validation completes. The fallback is removed before legacy
columns are dropped. `unknown` is valid only for converted ambiguous history; newly
actor-aware writes must never produce `unknown`.

### Indexes

Do not index every actor column by default. Add composite indexes only for demonstrated
query patterns, such as:

```text
(organization_id, project_id, created_actor_type, created_actor_id)
```

Backfill and operational queries should use primary-key ranges instead of full-table actor
indexes.

## Public API Design

Add an actor message with new protobuf field numbers:

```proto
message AuditActor {
  string type = 1;
  uint64 id = 2 [jstype = JS_STRING];
  optional string displayName = 3;
}
```

Audited resources may add:

```proto
optional AuditActor createdActor = <new-field-number>;
optional AuditActor updatedActor = <new-field-number>;
```

Actor-capable contracts add `createdActor` and `updatedActor` before legacy fields are
removed. The final Phase 3 contract release removes `createdBy`, `updatedBy`, `createdUser`,
and `updatedUser`; their protobuf field numbers and names are reserved permanently. The
breaking removal is published as the next major API/SDK version, and the previous major
version is supported only until the announced Phase 3 maintenance boundary.

### Projection Rules

- User actors may resolve a current user display projection.
- Credential, service, system, and unknown actors do not produce synthetic users.
- Deleted or unavailable actors leave display fields absent.
- Projection lookup failure is logged and measured but does not fail the resource operation.
- Actor projections are current metadata, not immutable historical snapshots.

## Migration Plan

### Phase 0: Contract Inventory

- Identify every externally exposed persistent create, update, archive, and delete path.
- Record the canonical public edge for each path.
- Identify the owning service, table, response message, and current audit behavior.
- Add golden compatibility tests before changing contracts.

The accepted inventory is `rfcs/0001-phase-3-audit-contract-inventory.json`. It records
83 administrative public mutation edges, six Assistant runtime persistence handlers, one
Integration lifecycle persistence handler, their canonical
writers and derived writes, the 64 actor-audited tables, six canonical protobuf sources, five
SDK delivery roots, known UI references, and the document/indexer contract-test boundary.

### Phase 1: Authentication Foundation

Phase 1 is delivered in independently reviewed slices. Phase 1A establishes actor
identity primitives, project credential propagation, and safe versioned scope-auth cache
behavior. Later Phase 1 slices add organization credential entities and stable signed
service identities before those actors may perform actor-aware mutations.

- Establish one fail-closed authentication decision at each protected transport boundary.
- Expose only `Authorize(ctx)` and `auth.Scope(allowed...)` to controllers.
- Keep concrete user, project, organization, and service credential verification separate
  behind the coordinating middleware.
- Remove silent protected-route pass-through and multi-middleware context overwrites.
- Remove legacy context accessors and capability probing only after all consumers migrate.
- Keep resource authorization and audit actor resolution separate from authentication.
- Propagate project credential ID through scope authorization.
- Introduce stable organization credential and service identities before actor-aware writes.
- Preserve versioned HMAC-fingerprinted authentication cache entries and bounded TTL.

### Phase 2: Bigint Schema and Identity Prerequisites

- Change the three unexecuted legacy-audited expansion migrations so actor ID columns use
  bigint and add an Integration migration for `external_audits` and `external_audit_metadata`.
- Add owned numeric organization credential, service identity, and system identity entities.
- Add actor-capable shared models and public contracts without removing legacy fields.
- Add type/ID pair constraints after historical conversion has populated each table.

### Phase 3: Complete Actor Rollout and Legacy Removal

The former dual-write, backfill, public-contract, domain-rollout, and cleanup phases are
merged into one approved Phase 3. Phase 3 may use ordered operational checkpoints, but it
is not split into independently scoped pilot or domain RFCs and is not complete until all
64 actor-audited tables and all public consumers are actor-only.

1. Publish the exact mutation, table, protobuf, SDK, UI, and external-consumer inventory.
2. Deploy bigint actor columns, numeric identity owners, and actor-capable contracts.
3. Deploy every canonical writer with actor resolution, temporary dual-write, creation
   actor immutability, missing-actor failure, and metrics.
4. Convert all historical tables in bounded resumable ranges and validate counts.
5. Deploy actor-first readers and projections, then remove legacy fallback after validation.
6. Complete Assistant, Endpoint, Web, organization credential, service, and system paths.
7. Release the next major protobuf, SDK, and UI contract without legacy audit fields.
8. Enter the final maintenance window: enforce a platform write fence, verify backups and
   replica health, stop old binaries, run service cleanup migrations in Web -> Endpoint ->
   Assistant order, verify each database, deploy only actor-only binaries, and lift the
   write fence after cross-service health and audit smoke tests pass.

If any cleanup migration or verification fails, the coordinator keeps the global write
fence active, stops subsequent database cleanup, and either fixes forward before traffic
resumes or restores all four databases to the same pre-maintenance backup set. No database
is restored independently while another remains on the actor-only contract.

## Failure Behavior

- Missing durable actor identity fails closed for every authenticated persistent mutation.
- Newly actor-aware mutations never write `unknown`.
- Actor display resolution never fails a successful mutation or read.
- Database actor-column write failure follows normal transaction rollback behavior.
- Historical conversion failure stops the affected range and preserves its checkpoint.
- Cleanup migration failure leaves the platform write fence active and blocks later service cleanup.
- Cache entries missing actor identity are treated as misses when identity is required.

## Observability

Add metrics for:

- Audit writes by actor type and service
- Missing actor identity
- Unknown actor writes
- Actor projection lookup failures
- Authentication actor cache hits and misses
- Conversion rows processed, user-classified, unknown-classified, failed, and remaining
- Migration duration, lock wait, WAL growth, disk headroom, and replica lag

Logs must include actor type and a non-secret actor identifier. Logs must never contain
raw credentials or originating tokens.

## Testing Strategy

### Shared Packages

- Actor resolver success for every supported actor type
- Missing identity and unsupported principal behavior
- Rejection of zero actor IDs and principals without durable identity
- No raw-token serialization or logging

### Authentication

- Project credential ID reaches `ProjectScope` and `ScopedAuthentication`
- Organization credential, service, and system actor IDs are non-zero, numeric, stable,
  owned, and signed where transported between services
- Archived or rotated credentials invalidate cached identity
- Old cache entries without actor identity are rejected where required
- Tenant scope remains unchanged

### Persistence

- User actor-only write success
- Project credential actor-only write success
- Organization, service, and system writes
- Update actor does not overwrite creation actor
- Transaction rollback preserves actor fields
- Unknown historical record behavior

### API Contract Cutover

- Removed protobuf field numbers and names are reserved
- Actor fields use new field numbers and `uint64` IDs
- Generated artifacts, SDKs, and UI contain no legacy audit fields
- Projection failure does not change resource success
- Direct service and gateway responses have declared parity

### Migration

- Up and down migrations for every service
- Exact historical conversion for positive, zero, and null legacy values
- Constraint validation for allowed types, positive bigint range, and type/ID pairing
- Legacy columns are absent after final cleanup migration
- Old binaries are stopped before the actor-only maintenance checkpoint
- Representative production-size migration timing

## Rollback

Before final legacy removal, rollback disables actor-first reads and writers while leaving
additive actor columns and converted data in place. Conversion checkpoints remain
resumable.

After final legacy removal begins, rollback requires the global write fence to remain
active. The migration coordinator either fixes forward before traffic resumes or restores
the same verified pre-maintenance backup set for Web, Integration, Endpoint, and Assistant, deploys the
previous binaries and public artifacts, validates all four databases, and only then lifts
the write fence. Down migrations are development-validation aids and are not treated as a
lossless production rollback because non-user actors cannot be represented in user-only
legacy columns.

## Alternatives Considered

### Repurpose `created_by` and `updated_by`

Store either a user ID or credential ID in the existing fields and add only actor-type
columns.

Rejected because existing clients, GORM relationships, and user hydration assume these
values are user IDs. Numeric namespaces may also collide.

### Create a Global Actor Table

Create a shared actor registry and reference it from all services.

Deferred because it introduces cross-service lifecycle ownership, availability coupling,
replication, and cleanup complexity. It may be reconsidered if actor metadata queries
become a core platform capability.

### Store Full Actor Snapshots Per Row

Persist actor name, email, and credential metadata with every resource.

Rejected because it duplicates personally identifiable information, complicates erasure
and correction requirements, and significantly increases schema and write complexity.

### Store Only Authentication Type

Record that a request used a user token or API key without a stable actor ID.

Rejected because it cannot identify which user, credential, or service performed the
operation.

### Reconstruct Actor Identity From Tokens

Persist or replay API keys and service tokens to derive historical identity.

Rejected for security, rotation, expiration, and reliability reasons.

## Risks

- Broad schema and contract scope across independently deployed services
- Old binaries failing against removed columns during final cleanup
- Breaking major protobuf, SDK, and UI contract changes for untracked external consumers
- Long-running conversion, cleanup locks, WAL growth, disk pressure, and replica lag
- Irreversible loss of non-user actor detail if rollback uses down migrations instead of backups
- N+1 lookups and latency from actor projections
- Cross-tenant or PII exposure from actor display metadata
- Missing durable numeric organization, service, or system identity owners

These risks are mitigated through additive prerequisites, bounded conversion, exact
inventory validation, numeric identity ownership, database constraints, a versioned major
contract release, a global write fence, explicit Web -> Endpoint -> Assistant cleanup
ordering, verified same-point backups, representative-size timing, and cross-service
post-migration verification.

## Resolved Design Decisions

1. Actor IDs are `uint64` contract values restricted to `1..9223372036854775807` and
   positive PostgreSQL `bigint` values in persistence.
2. Organization credentials will be owned by a durable `organization_credentials`
   entity. Organization ID remains authorization scope and is never the credential actor
   ID.
3. Internal service and system identities are durable numeric records. Signed assertions
   carry their numeric IDs; shared secrets, names, and tenant scope are not identities.
4. The service that owns the persistent resource is the canonical audit-writing edge.
   Gateways and callers propagate identity but do not write duplicate audit state.
5. Historical rows with a verified non-zero legacy user ID may be classified as `user`.
   All ambiguous rows become `unknown`.
6. Legacy persistence and public audit fields are removed before the complete Phase 3
   finishes; removed protobuf field numbers and names remain reserved permanently.
7. Actor display metadata exposes ID and display name by default. Email requires a
   separately authorized projection and is not part of the base actor contract.
8. `unknown` is allowed only for ambiguous historical values converted by the migration.
   New authenticated mutation paths fail closed when durable identity is unavailable.
9. Archive and soft-delete attribution is stored in `updatedActor`. Hard deletion and
   immutable lifecycle history require a separate audit-event design.
10. All 64 actor-audited tables, writers, readers, public contracts, SDKs, and UI consumers move
    to actor-only audit identity inside one complete Phase 3 with ordered checkpoints and
    no separately approved pilot, domain rollout, backfill, contract, or cleanup phase.

## Acceptance Criteria

This RFC may move to `Accepted` when:

- Actor taxonomy and identifier format are approved.
- Project, organization, service, and system identity ownership is defined.
- Breaking API, SDK, and UI cutover strategy is approved.
- In-migration historical conversion policy is approved.
- Complete-domain maintenance window, observability, backup rollback, and validation are approved.
- A plan challenger finds no unresolved critical or major issue.

The externally exposed mutation inventory remains a mandatory Phase 0 deliverable before
the actor-only cutover begins; it does not block the authentication foundation.

Implementation must not begin while this RFC remains `Draft`.

## Decision Log

| Date | Decision | Status |
| --- | --- | --- |
| 2026-08-20 | Actor type `project` stores the project credential ID; project ID remains authorization scope. | Approved |
| 2026-08-20 | Actor IDs use strings at contract boundaries; organization and service identities use dedicated durable owners. | Approved |
| 2026-08-20 | Resource-owning services are canonical audit writers; assistant create/update is the first dual-write pilot. | Approved |
| 2026-08-20 | Phase 1A includes versioned HMAC-fingerprinted scope-auth cache entries with bounded TTL. | Approved |
| 2026-08-21 | Reopen RFC 0001 for an authentication-boundary amendment: one protected-request authentication decision, `Authorize(ctx)`, `auth.Scope(allowed...)`, and no context utility layers. | Draft |
| 2026-08-21 | Persist audit identity as `created_actor_type`, `created_actor_id`, `updated_actor_type`, and `updated_actor_id`; actor and scope remain separate concepts. | Approved |
| 2026-08-21 | Remove `created_by` and `updated_by` only in a later per-service contract phase after dual write, backfill, read cutover, compatibility evidence, and rollback validation. | Approved |
| 2026-08-21 | Endpoint middleware converts credential-specific principles into one request `Authentication` object containing actor and available user, organization, and project contexts; no normalization naming is used. | Approved |
| 2026-08-21 | Apply the request `Authentication` contract to every API and shared client without compatibility type checks; preserve existing per-RPC authorization semantics during migration. | Approved |
| 2026-08-22 | Supersede string actor IDs with positive bigint-compatible uint64 contract IDs and PostgreSQL bigint actor columns. | Accepted |
| 2026-08-22 | Merge dual-write, conversion, contract rollout, all audited domains, and legacy cleanup into one complete Phase 3 with ordered safety checkpoints. | Accepted |
