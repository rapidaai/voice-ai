# RFC 0001: Actor-Aware Audit Identity

- Status: Accepted
- Date: 2026-08-20
- Owners: Platform and API teams
- Reviewers: Authentication, data, SDK, UI, and service owners

## Summary

Replace the assumption that every `created_by` or `updated_by` value is a user ID
with an additive actor-aware audit identity.

Every externally exposed mutation of a persistent business resource must record the
durable identity of the actor that performed it. Supported actors include users,
project credentials, organization credentials, internal services, and system jobs.

The migration preserves existing `created_by`, `updated_by`, `createdUser`, and
`updatedUser` contracts while introducing explicit actor type and actor identifier
fields. The rollout uses expand, dual-write, backfill, read-preference, and eventual
contract migration phases.

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

The initial repository analysis found audited columns across approximately:

- 48 assistant-api tables
- 11 web-api tables
- 10 endpoint-api tables

The exact table inventory must be regenerated and reviewed before implementation.

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
5. Preserve existing public fields and successful behavior during migration.
6. Support incremental, independently reversible service rollouts.
7. Make missing or unresolved actor enrichment observable without failing mutations.

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
- Existing `createdBy`, `updatedBy`, `createdUser`, and `updatedUser` fields remain
  available through the compatibility window.
- New protobuf fields use new field numbers.
- Existing response wrappers, success codes, error codes, and cardinality remain stable.
- Direct service and gateway behavior remain compatible unless explicitly versioned.
- SDK regeneration is coordinated across Go, Node.js, Python, React, and widget clients.

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
    ID   string
}
```

Actor identifiers are strings. User and credential IDs can be encoded as decimal strings,
while service identities can use stable names or UUIDs without another schema migration.

Actor type names describe the principal class, not the authorization scope identifier or
the credential implementation. Their identifiers are defined as follows:

| Actor type | Actor ID | Must not use |
| --- | --- | --- |
| `user` | User ID | Session or access-token ID |
| `project` | Project credential ID | Project ID or raw project key |
| `organization` | Organization credential ID | Organization ID or raw organization key |
| `service` | Provisioned stable service ID | Shared secret, token, user ID, or project ID |
| `system` | Registered system-job ID | Empty or implicit identity |
| `unknown` | Empty | Guessed user, credential, scope, or service identity |

For example, `{type: "project", id: "123"}` means project credential `123` performed the
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

Do not add new methods directly to `SimplePrinciple`. That would immediately break every
implementation and test mock.

Add a companion contract:

```go
type ActorIdentityProvider interface {
    AuditActor() (ActorIdentity, bool)
}
```

Audit-writing code uses a centralized, fail-closed resolver:

```go
func ResolveAuditActor(auth SimplePrinciple) (ActorIdentity, error)
```

The resolver rejects authenticated mutation paths that claim to support actor-aware audit
but do not provide a stable identity. Compatibility-only paths may explicitly use
`unknown` while rollout is in progress.

### Project Credentials

Add project credential identity to the project authentication path:

1. Add a credential ID field to `ProjectScope`.
2. Select and map `project_credentials.id` during authorization.
3. Add optional actor type and actor ID fields to `ScopedAuthentication`.
4. Preserve the fields through the web authentication client and middleware.

The raw project credential remains secret and is never persisted as audit metadata.

### Organization Credentials

An organization scope currently identifies only the organization. Before organization-key
mutations become actor-aware, the platform must define a durable organization credential
entity or equivalent stable identity.

The organization ID alone is not a valid credential actor ID.

### Service Identity

Service assertions must include a signed stable identity such as `service_id` or `sub`.
A shared signing secret and tenant scope are insufficient to identify which service acted.

Internal delegation creates a new short-lived service assertion containing tenant scope
and actor provenance. It must not forward the originating user or API-key credential.

### System Identity

Background workers use named system identities registered in configuration or code, for
example `assistant-indexer` or `conversation-retention-job`. An empty actor is not treated
as a system actor implicitly.

## Persistence Design

Add four nullable columns to every audited persistent resource table:

```text
created_actor_type varchar(32)
created_actor_id   text
updated_actor_type varchar(32)
updated_actor_id   text
```

Keep the legacy columns unchanged:

```text
created_by bigint
updated_by bigint
```

### Write Behavior

| Actor | Legacy field | Actor type | Actor ID |
| --- | --- | --- | --- |
| User | User ID | `user` | User ID |
| Project credential | `0` or unchanged compatibility value | `project` | Credential ID |
| Organization credential | `0` or unchanged compatibility value | `organization` | Credential ID |
| Service | `0` or configured compatibility user | `service` | Service ID |
| System | `0` | `system` | System ID |

Using a configured compatibility user for service writes is permitted only as a temporary,
documented rollout mechanism. It must not be presented as the true actor.

### Read Behavior

Readers prefer actor-aware fields. If they are absent, readers derive a legacy projection:

- Non-zero `created_by` or `updated_by` becomes a legacy user candidate.
- Zero or missing legacy values become `unknown`.
- Readers do not guess project, organization, or service identities.

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
  string id = 2;
  optional string displayName = 3;
}
```

Audited resources may add:

```proto
optional AuditActor createdActor = <new-field-number>;
optional AuditActor updatedActor = <new-field-number>;
```

Legacy fields remain available:

```text
createdBy
updatedBy
createdUser
updatedUser
```

### Projection Rules

- `createdUser.id` must correspond to a user `createdActor` or legacy `createdBy`.
- `updatedUser.id` must correspond to a user `updatedActor` or legacy `updatedBy`.
- Non-user actors do not produce synthetic users.
- Deleted or unavailable actors leave display fields absent.
- Projection lookup failure is logged and measured but does not fail the resource operation.
- Actor projections are current metadata, not immutable historical snapshots.

## Migration Plan

### Phase 0: Contract Inventory

- Identify every externally exposed persistent create, update, archive, and delete path.
- Record the canonical public edge for each path.
- Identify the owning service, table, response message, and current audit behavior.
- Add golden compatibility tests before changing contracts.

The initial review identified approximately 46 create and update RPC paths, but this must
be regenerated from the implementation branch.

### Phase 1: Authentication Foundation

Phase 1 is delivered in independently reviewed slices. Phase 1A establishes actor
identity primitives, project credential propagation, and safe versioned scope-auth cache
behavior. Later Phase 1 slices add organization credential entities and stable signed
service identities before those actors may perform actor-aware mutations.

- Add actor identity types and resolver.
- Propagate project credential ID through scope authorization.
- Introduce stable organization credential and service identities.
- Add versioned authentication cache entries containing actor identity.
- Replace raw-token cache keys with domain-separated HMAC fingerprints.
- Add bounded cache TTL and credential invalidation behavior.

### Phase 2: Schema Expansion

- Add nullable actor columns without defaults.
- Use short lock timeouts and independently deployable service migrations.
- Do not backfill in the schema migration transaction.
- Deploy actor-aware entity fields while keeping legacy fields intact.

### Phase 3: Dual Write

- User mutations write both legacy and actor-aware fields.
- Non-user mutations write actor-aware fields and compatibility-safe legacy values.
- Readers prefer actor-aware fields and fall back to legacy fields.
- Add mismatch metrics when legacy and actor-aware user IDs disagree.

### Phase 4: Backfill

- Process bounded primary-key ranges.
- Backfill verified historical user rows as `user`.
- Mark ambiguous records as `unknown`.
- Do not attempt to identify historical API keys from secrets, logs, or current credentials.
- Make backfill idempotent, resumable, rate-limited, and observable.

### Phase 5: Public Contract Expansion

- Add `createdActor` and `updatedActor` to resource messages.
- Regenerate and release SDKs.
- Update UI components to render users, credentials, services, and system actors.
- Keep nested actor display resolution best-effort and authorization-scoped.

### Phase 6: Domain Rollout

Recommended sequence:

1. Assistant creation and update pilot.
2. Remaining assistant resources.
3. Endpoint resources.
4. Web resources and credentials.
5. Organization credentials and service/system operations.

Each domain requires an independently approved plan, migration, tests, rollout switch,
and rollback evidence.

### Phase 7: Contract Cleanup

After a full SDK and client compatibility window:

- Mark legacy numeric audit fields as deprecated.
- Stop adding new dependencies on `createdUser` and `updatedUser`.
- Evaluate removal only through a separate RFC.

This RFC does not approve removing legacy fields.

## Failure Behavior

- Missing actor identity fails closed for endpoints declared actor-aware.
- During compatibility rollout, explicitly listed endpoints may write `unknown` and emit a
  metric rather than fail.
- Actor display resolution never fails a successful mutation or read.
- Database actor-column write failure follows normal transaction rollback behavior.
- Backfill failures stop the affected batch and preserve its checkpoint.
- Cache entries missing actor identity are treated as misses when identity is required.

## Observability

Add metrics for:

- Audit writes by actor type and service
- Missing actor identity
- Unknown actor writes
- Legacy/actor field mismatches
- Actor projection lookup failures
- Authentication actor cache hits and misses
- Backfill rows processed, skipped, failed, and remaining

Logs must include actor type and a non-secret actor identifier. Logs must never contain
raw credentials or originating tokens.

## Testing Strategy

### Shared Packages

- Actor resolver success for every supported actor type
- Missing identity and unsupported principal behavior
- Compatibility behavior for principals without `ActorIdentityProvider`
- No raw-token serialization or logging

### Authentication

- Project credential ID reaches `ProjectScope` and `ScopedAuthentication`
- Organization credential and service IDs are stable and signed
- Archived or rotated credentials invalidate cached identity
- Old cache entries without actor identity are rejected where required
- Tenant scope remains unchanged

### Persistence

- User dual-write success
- Project credential dual-write success
- Organization, service, and system writes
- Update actor does not overwrite creation actor
- Transaction rollback preserves both legacy and actor fields
- Legacy read fallback
- Unknown historical record behavior

### API Compatibility

- Existing protobuf and JSON golden responses remain compatible
- New actor fields use new field numbers
- Non-user actors do not populate `createdUser` or `updatedUser`
- Projection failure does not change resource success
- Direct service and gateway responses have declared parity

### Migration

- Up and down migrations for every service
- Backfill idempotency and resume behavior
- Mixed-version application compatibility
- Rollback with actor columns present but unused
- Representative production-size migration timing

## Rollback

Rollback occurs in stages:

1. Disable actor-aware reads and return to legacy projections.
2. Disable actor-aware writes while leaving nullable columns in place.
3. Roll back authentication propagation if no actor-aware endpoint requires it.
4. Stop backfill workers and preserve checkpoints.
5. Drop actor columns only after the application rollback and retention window complete.

Columns must not be dropped during an emergency application rollback.

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

- Broad schema scope across independently deployed services
- Partial rollout producing mixed legacy and actor-aware records
- Public SDK churn from contract additions
- Incorrect actor attribution from fallback logic
- N+1 lookups and latency from actor projections
- Cross-tenant or PII exposure from actor display metadata
- Long-running backfills affecting database performance
- Service assertions that identify scope but not the calling service

These risks are mitigated through additive fields, phased service ownership, server-derived
identity, best-effort projections, bounded backfills, metrics, and rollback switches.

## Resolved Design Decisions

1. Actor IDs are strings in shared, protobuf, and public contracts. Numeric database
   identifiers are formatted in base 10 at the contract boundary.
2. Organization credentials will be owned by a durable `organization_credentials`
   entity. Organization ID remains authorization scope and is never the credential actor
   ID.
3. Internal service identity is a signed stable `service_id` string from an allow-listed
   service registry. Shared secrets and tenant scope are not identities.
4. The service that owns the persistent resource is the canonical audit-writing edge.
   Gateways and callers propagate identity but do not write duplicate audit state.
5. Historical rows with a verified non-zero legacy user ID may be classified as `user`.
   All ambiguous rows become `unknown`.
6. Legacy fields remain supported through at least two minor SDK release cycles after all
   public actor fields are generally available.
7. Actor display metadata exposes ID and display name by default. Email requires a
   separately authorized projection and is not part of the base actor contract.
8. `unknown` is allowed only for legacy fallback, historical backfill ambiguity, and
   explicitly allow-listed compatibility paths. Newly actor-aware authenticated mutation
   paths fail closed when durable identity is unavailable.
9. Archive and soft-delete attribution is stored in `updatedActor`. Hard deletion and
   immutable lifecycle history require a separate audit-event design.
10. Assistant creation and update are the first dual-write pilot after the authentication
    foundation, contract inventory, and assistant schema expansion are approved.

## Acceptance Criteria

This RFC may move to `Accepted` when:

- Actor taxonomy and identifier format are approved.
- Project, organization, service, and system identity ownership is defined.
- Backward-compatible API and SDK strategy is approved.
- Historical backfill policy is approved.
- Pilot domain, rollout sequence, observability, and rollback are approved.
- A plan challenger finds no unresolved critical or major issue.

The externally exposed mutation inventory remains a mandatory Phase 0 deliverable before
schema expansion or dual write begins; it does not block the additive authentication
foundation in Phase 1A.

Implementation must not begin while this RFC remains `Draft`.

## Decision Log

| Date | Decision | Status |
| --- | --- | --- |
| 2026-08-20 | Actor type `project` stores the project credential ID; project ID remains authorization scope. | Approved |
| 2026-08-20 | Actor IDs use strings at contract boundaries; organization and service identities use dedicated durable owners. | Approved |
| 2026-08-20 | Resource-owning services are canonical audit writers; assistant create/update is the first dual-write pilot. | Approved |
| 2026-08-20 | Phase 1A includes versioned HMAC-fingerprinted scope-auth cache entries with bounded TTL. | Approved |
