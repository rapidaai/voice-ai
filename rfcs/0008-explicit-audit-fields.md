# RFC 0008: Explicit Audit Fields

- Status: Accepted
- Owner: Platform and API teams
- Created: 2026-08-23
- Updated: 2026-08-23
- Reviewers: Authentication, data, and service owners

## Summary

Remove hidden GORM audit callbacks, actor hydration hooks, and audit mutation helper
functions. Every create and update operation will use the actor from the explicit
authentication argument and populate only the persisted columns owned by that operation.

## Context

`pkg/models/gorm` currently has multiple owners for audit state. Models can set actor
fields directly, helper functions can merge actor columns into update maps, and global
GORM callbacks can derive the actor from `context.Context` and overwrite the same
fields. Read hooks also construct transient actor objects after queries.

This hidden context dependency caused direct SIP conversation creation to fail even
though the SIP stage carried valid authentication. The conversation service received
the authentication as an argument, but the GORM callback ignored that argument and
looked for a separate context value.

## Goals

- Make every persistent audit write explicit at its service write site.
- Set `CreatedActorType` and `CreatedActorID` directly for every create.
- Set `UpdatedActorType` and `UpdatedActorID` directly for every update.
- Make `Authentication.Actor()` a direct accessor. Actor validity is enforced by
  `IsAuthenticated()` and `Scope(...)`.
- Remove GORM audit callbacks and their registration.
- Remove actor hydration hooks and transient actor fields from persistence models.
- Remove audit mutation constructors and map-merging helpers.
- Preserve existing database columns and public actor response values.

## Non-Goals

- No database column additions, removals, or type changes.
- No authentication type or token format changes.
- No change to tenant authorization or routing rules.
- No unrelated service or model refactoring.

## Scope and Ownership

### Allowed Paths

- `pkg/models/gorm/` - simplify the shared persistence model and remove callbacks.
- `pkg/connectors/` - remove callback registration.
- `cmd/assistant/assistant.go` - remove callback registration.
- `cmd/endpoint/endpoint.go` - remove callback registration.
- `cmd/integration/integration.go` - remove callback registration.
- `cmd/web/web.go` - remove callback registration.
- `api/assistant-api/internal/services/` - explicitly assign audit columns at writes.
- `api/endpoint-api/internal/service/` - explicitly assign audit columns at writes.
- `api/web-api/internal/service/` - explicitly assign audit columns at writes.
- API entity-to-protobuf conversion files that currently consume hydrated actor fields.
- Focused tests in each changed package.
- `rfcs/0008-explicit-audit-fields/` - governed workflow evidence.

### Out-of-Scope Paths

- Database migrations.
- Generated protobuf and SDK files.
- Unrelated authentication middleware.
- UI code.

## Proposed Design

`Mutable` remains a plain persisted structure containing status and the four actor
columns. It has no actor constructor, actor setters, update-map helpers, transient actor
objects, or GORM read hooks.

At each create boundary, the service uses the actor from its explicit
`*types.Authentication` argument and assigns `CreatedActorType` and `CreatedActorID`
before calling GORM. Create paths do not populate update audit columns.

At each update boundary, the service assigns `UpdatedActorType` and `UpdatedActorID` to
the explicit update operation. Audit-owned columns must not be accepted from external
request payloads.

`Authentication.IsAuthenticated()` owns actor validation for every authentication type.
`Authentication.Scope(...)` continues to reject unauthenticated or disallowed callers.
After those checks, `Authentication.Actor()` returns the actor value directly without an
error result, allowing write sites to use `auth.Actor().Type.String()` and
`auth.Actor().ID`.

Response conversion constructs `types.ActorIdentity` or protobuf audit actors directly
from the persisted type and ID columns. Persistence models will not retain hydrated
actor fields.

Global create and update callbacks are removed. Database connectors no longer register
audit callbacks. Consequently, database writes have no hidden authentication dependency.

## Contracts and Compatibility

- Existing database column names and types remain unchanged.
- Existing API actor response fields remain unchanged.
- Existing authenticated write behavior remains compatible when the caller provides a
  valid durable actor.
- `IsAuthenticated()` and `Scope(...)` reject authentication without a valid actor.

## Failure and Recovery

- Authentication and scope validation abort the operation before database mutation.
- Multi-record writes read the actor once and assign the same actor to every record.
- Transactions retain their current rollback behavior.
- Partial migration is not deployable after callbacks are removed. All audited write
  sites must be migrated in the same release.

## Security and Privacy

- Services use only the durable actor identity from the validated authentication object.
- Tenant scope IDs are not substituted for actor IDs.
- Request payloads cannot set or override audit-owned columns.
- No credentials or tokens are persisted.

## Observability

- Existing service error logging remains unchanged.
- Actor resolution errors remain actionable and occur before database execution.
- Callback-specific audit metrics are removed with the callbacks.

## Data and Migration

No schema or backfill is required. Existing actor columns remain authoritative. The
application write path changes atomically from callback-populated fields to explicitly
populated fields.

## Rollout

1. Inventory every create and update of a model embedding `Mutable`.
2. Add explicit actor resolution and column assignment at every inventoried write.
3. Update response conversion to read persisted actor columns directly.
4. Remove callbacks, callback registration, hydration hooks, and helper functions.
5. Run focused tests for every changed package and repository finalization.
6. Stop rollout if any audited write can reach GORM without all required actor columns.

## Rollback

Revert the implementation commit as one unit. No database rollback is required because
the schema and stored representation do not change.

## Alternatives Considered

- Keep callback ownership and propagate authentication through every context. Rejected
  because audit behavior remains hidden and duplicates explicit authentication inputs.
- Keep constructors or map-merging helpers. Rejected because each write site should show
  the complete persisted audit values directly.
- Migrate one service at a time while retaining callbacks. Rejected because two writers
  would continue to own the same columns and obscure incomplete migrations.

## Testing and Verification

- Add or update focused service tests covering successful creates and invalid actors in
  every changed package.
- Add or update focused update tests covering explicit updated-actor columns.
- Verify response conversion from persisted actor columns.
- Run `go test` for every changed package.
- Run `make agent-finalize CHANGED_FILES="comma,separated,paths"`.
- Run repository review with no unresolved critical or major findings.

## Acceptance Criteria

- [ ] No production code calls `NewMutable`, `SetCreatedActor`, `SetUpdatedActor`,
  `ActorUpdateColumns`, or `MergeActorUpdateColumns`.
- [ ] No GORM audit callback is registered.
- [ ] No `AfterFind` hook hydrates actor identities.
- [ ] Every audited create explicitly assigns created actor type and ID before GORM.
- [ ] Every audited update explicitly assigns both updated-actor columns.
- [ ] API responses preserve existing created and updated actor values.
- [ ] `IsAuthenticated()` and `Scope(...)` reject missing or invalid actors.
- [ ] Required tests and finalization pass.
- [ ] Independent review has no unresolved critical or major findings.

## Open Questions

None.

## Challenge Resolution

The user rejected hidden callbacks, hydration, and mutation helpers and required direct
assignment of the four persisted actor columns at every create. Callback registration
must be removed from every service binary. No unresolved design questions remain.

## Artifact Index

- `jsons/plan.json` - proposed implementation and verification plan.
- `jsons/challenge.json` - approved design challenge.
- `jsons/confirmation.json` - pending exact-digest confirmation.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-23 | Replace hidden audit behavior with explicit persisted fields | User | Conversation review feedback |
| 2026-08-23 | Remove audit callbacks and observability from every service | User | Conversation review feedback |
| 2026-08-23 | Creates set only created fields, updates set only updated fields, and `Actor()` becomes a direct accessor | User | Conversation review feedback |
