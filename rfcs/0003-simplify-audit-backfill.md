# RFC 0003: Simplify Audit Actor Backfill Migrations

- Status: Accepted
- Date: 2026-08-22
- Supersedes: RFC 0001 Phase 3 batched backfill, migration metrics, and SQL-text test requirements only

## Summary

Replace the Phase 3 audit actor backfill procedures with direct SQL updates. Existing legacy audit IDs in Assistant, Endpoint, and Web are user IDs, so their historical rows map directly to `actor_type='user'` and the existing `created_by` or `updated_by` value. Integration audit tables have no legacy actor ID columns, so their historical creation actor remains `unknown` with a null ID. Remove Go tests that inspect migration SQL text, including the call-context migration-file test, and retain executable PostgreSQL migration verification.

## Motivation

The current migrations implement batching, resumability, progress metrics, dynamic table inspection, and per-table logging for a one-time unshipped conversion. Those mechanisms add substantial SQL and test maintenance without changing the intended data mapping. The user's requested contract is simpler:

- a populated legacy audit ID is a user ID;
- a null legacy update ID remains an absent update actor;
- Integration history has no legacy actor identity and therefore cannot be attributed to a user;
- invalid legacy IDs must fail migration rather than be silently reclassified.

## Data Contract

For every audited Assistant, Endpoint, and Web table:

- `created_actor_type` becomes `user`;
- `created_actor_id` becomes `created_by`;
- when `updated_by` is non-null, `updated_actor_type` becomes `user` and `updated_actor_id` becomes `updated_by`;
- when `updated_by` is null, both updated actor fields remain null;
- all affected migration versions run together while writes are fenced, so no actor-aware rows can exist before the backfill.

Some legacy columns are nullable at schema level, so each run migration first performs an explicit data preflight across every audited table. A null, zero, negative, or otherwise invalid legacy creation ID, or a non-positive non-null update ID, raises an exception before any backfill statement runs. The existing actor-pair constraints remain authoritative after conversion.

For Integration tables, which never had `created_by` or `updated_by`, historical rows receive `created_actor_type='unknown'` and a null actor ID. Updated actor fields remain null.

## Migration Design

- Keep the existing expand, convert, run, finalize, and cleanup migration version numbers.
- Remove `audit_actor_migration_metrics` creation and all persisted backfill procedures.
- Keep actor-pair constraints in the convert migrations.
- Replace each run migration's procedure call with a preflight followed by direct, explicit `UPDATE` statements for every audited table.
- Remove procedure drops from convert down and finalize up migrations.
- Keep final constraint validation, creation-actor immutability triggers, legacy-column cleanup, and rollback behavior unchanged.
- Do not add batching, dynamic SQL, progress tables, or resumability infrastructure.

Repository inspection on 2026-08-22 confirms the affected migration files are absent from `origin/main` and were introduced only by commit `e7765145` on `rfc/actor-aware-audit-identity`. Repository history cannot prove deployment state. Exact-digest approval of this RFC therefore also records the data owner's attestation that:

- populated legacy audit IDs are user IDs in every affected environment;
- all legacy creation IDs are non-null and positive;
- every non-null legacy update ID is positive; and
- no affected Phase 3 migration version has been applied in any environment.

If any statement is false, these migration files must not be rewritten; a new forward migration is required instead.

## Test Strategy

Delete Go tests under service migration directories because they assert SQL text rather than execute migrations. This includes the organization-credential and removed-registry migration tests at the user's direction. Delete `api/assistant-api/internal/callcontext/migration_test.go` for the same reason. Their schema and security contracts move to executable PostgreSQL verification; package behavior tests unrelated to migration text remain.

Update `bin/verify-phase3-migrations.sh` to execute full PostgreSQL histories and assert:

- legacy audit columns are removed;
- actor constraints are validated;
- every audited table is present in the backfill inventory and its direct update is executed by the migration;
- seeded legacy positive user IDs map to user actors with matching IDs;
- null legacy update IDs remain null actor pairs;
- Integration historical rows map to unknown creation actors because no legacy user ID exists;
- invalid legacy creation or update IDs fail before any backfill statement runs;
- the call-context authentication snapshot columns are added, `auth_token` remains present, and rollback removes only the snapshot columns;
- organization credentials store only credential fingerprints and the removed registry tables do not survive the final Web schema;
- no migration metrics table or backfill procedure remains;
- final Web schema still lacks service and system identity registry tables.

The validator combines full-history execution with an exhaustive expected-table inventory check, so omission of any direct update fails validation even when a table has no seed row. It also replaces the removed Go migration tests by checking:

- additive expansion and exact actor-column coverage for every audited table;
- five-second lock timeout and reset behavior;
- explicit irreversible-down guidance for destructive migrations;
- organization-credential fingerprint-only storage, actor constraints, uniqueness, and rejection of `unknown`;
- service/system registry creation constraints, ownership foreign keys, uniqueness, and dependency-safe removal ordering;
- call-context snapshot columns, preservation of `auth_token`, and snapshot-column rollback.

The interrupted-batch/resume scenario and migration-metrics assertions are removed because batching and metrics no longer exist.

## Scope

In scope:

- Phase 3 convert, run, and finalize migration files for Assistant, Endpoint, Integration, and Web.
- Phase 3 PostgreSQL migration verification script and verification documentation.
- A deployment operational-readiness receipt template.
- Removal of all Go tests located in the four service migration directories.
- Removal of the call-context migration SQL-text test.
- RFC index and lifecycle evidence for this change.

Out of scope:

- Runtime actor propagation or authorization behavior.
- Application schema columns, actor constraints, immutability triggers, and legacy-column cleanup semantics. Removal of migration-internal procedure and metrics objects is in scope.
- Protobuf, SDK, UI, or Document API changes.
- Unrelated local changes in Redis, docs, or examples.

## Risks and Mitigations

- Direct updates can hold locks longer than batched updates. Deployment therefore runs with writes fenced in the existing Phase 3 maintenance window after the release owner records per-table row counts, the production-sized rehearsal duration, the approved maximum duration, backup identity, and rollback owner in `rfcs/0003-simplify-audit-backfill.operational-readiness.json`. A pending or absent receipt blocks deployment, not implementation or commit.
- Rewriting an already-applied migration would be unsafe. Implementation is conditional on these migration versions remaining unshipped.
- Invalid legacy IDs now stop migration before updates begin instead of becoming `unknown`. This is intentional fail-closed behavior and the validator includes a rejection case.
- Integration history cannot be attributed to a user because no legacy user identity exists; retaining `unknown` avoids fabricating identity.

## Rollback

Rollback remains the same-point four-database backup restoration documented for Phase 3. Cleanup down migrations remain intentionally irreversible. Before migration, operators retain the previous binaries and complete database backup set; on failure they restore all four databases together before restarting the previous release.

## Acceptance Criteria

- No `audit_actor_migration_metrics` table or `backfill_*_audit_actor_identity` procedure is created or referenced by Phase 3 migrations.
- Assistant, Endpoint, and Web historical audit IDs are copied directly to user actor fields after an exhaustive preflight.
- Integration historical rows remain explicitly unknown without a fabricated actor ID.
- All migration-package Go tests and the call-context migration SQL-text test are removed.
- The PostgreSQL migration and rollback validators pass with exhaustive table and removed-contract coverage.
- Relevant Go tests and formatting checks pass.
- Exact-digest approval attests the legacy-ID mapping and that no affected migration version has been deployed.
- Deployment remains blocked until the operational-readiness receipt is approved by the release owner.
