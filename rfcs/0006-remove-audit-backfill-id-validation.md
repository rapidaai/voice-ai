# RFC 0006: Simplify Audit Actor Migrations

- Status: Accepted
- Date: 2026-08-23
- Supersedes: RFC 0003 run-backfill validation and lock-timeout wrapper requirements only

## Summary

Reduce every Phase 3 audit run-backfill migration to its existing direct `UPDATE`
statements. Remove the positive-ID `DO` validation blocks from Assistant, Endpoint, and Web,
remove the migration-local `SET lock_timeout` and `RESET lock_timeout` statements from all
four services, and remove creation-actor immutability triggers from all four finalize
migrations.

No actor mapping, table inventory, actor-pair constraint, cleanup migration, or runtime API
changes. The PostgreSQL verifier is updated to reflect the removed wrappers and triggers while
preserving executable actor-pair validation.

## Decision

The implementation changes exactly these files:

- `api/assistant-api/migrations/000058_run_audit_actor_backfill.up.sql`
- `api/endpoint-api/migrations/000004_run_audit_actor_backfill.up.sql`
- `api/integration-api/migrations/000004_run_audit_actor_backfill.up.sql`
- `api/web-api/migrations/000009_run_audit_actor_backfill.up.sql`
- `api/assistant-api/migrations/000059_finalize_audit_actor_identity.up.sql`
- `api/endpoint-api/migrations/000005_finalize_audit_actor_identity.up.sql`
- `api/integration-api/migrations/000005_finalize_audit_actor_identity.up.sql`
- `api/web-api/migrations/000010_finalize_audit_actor_identity.up.sql`
- `bin/verify-phase3-migrations.sh`

Each run-backfill SQL file contains only the direct updates already present in that migration.
Update counts remain Assistant 41, Endpoint 10, Integration 2, and Web 11. The update bodies
remain byte-for-byte unchanged. The verifier changes only where required to validate this
contract.

The finalize migrations continue validating the actor-pair constraints but no longer create
`public.reject_created_actor_change()` or attach `audit_created_actor_immutable` triggers.
Creation actor fields therefore remain ordinary updateable columns after migration.

Assistant, Endpoint, and Web continue copying legacy IDs directly into user actor fields.
Null updater IDs continue producing null updater pairs. Nullable legacy creator IDs are
accepted as the direct result `created_actor_type='user', created_actor_id=NULL`. Existing
constraints may still reject zero or negative typed IDs. Integration continues assigning
historical creation actors as `unknown` with null IDs.

## Scope

One implementation owner owns all nine files listed above. No other production, test,
configuration, API, UI, or documentation file is part of implementation.

## Principles

- KISS: retain only direct data-copy statements.
- YAGNI: add no validation procedure, timeout wrapper, batching, metrics, repair, or fallback.
- Ownership: one worker owns all eight migration files and the verifier.
- Single source of truth: the existing legacy columns remain the only source for historical
  actor IDs.
- Explicit contracts: mappings and remaining constraint behavior are documented here.
- Fail-safe behavior: transactional migration execution and existing constraints remain
  unchanged.
- Observability: deployment and migration-runner errors remain the operational signal.
- Least privilege: no permissions or trust boundaries change.
- Reversibility: these historical migration files may be edited only while unapplied.

## Compatibility and Operations

The user confirmed on August 23, 2026 that the affected migration versions have not completed
in any environment and explicitly requested direct-update-only migrations. If any version has
actually been applied, stop rollout and use forward corrective migrations instead of rewriting
history.

Removing file-local lock timeouts means the deployment environment owns any desired session or
database timeout policy. Existing coordinated backup, maintenance-window, dirty-state checks,
and same-point restore procedures from RFC 0003 remain required.

## Alternatives Rejected

- Retaining the preflight: rejected because it blocks tolerated legacy rows before copying.
- Retaining lock-timeout wrappers: rejected by the direct-update-only requirement.
- Mapping missing IDs to another actor type: rejected because it fabricates identity.
- Rewriting constraints: rejected because it broadens the data contract.
- Retaining creation-actor triggers: rejected by the explicit request to remove
  `audit_created_actor_immutable` from every service migration.
- Keeping obsolete validator assertions: rejected because the checked-in verification contract
  must pass against the migration shape it validates.

## Verification

Verification must confirm:

- no target migration contains `DO`, the positive-ID exception, `SET lock_timeout`, or
  `RESET lock_timeout`;
- update counts remain 41, 10, 2, and 11;
- each direct-update body is byte-for-byte identical to its `HEAD` version after stripping
  only the removed wrapper;
- all four files parse and execute through disposable PostgreSQL migration histories;
- the checked-in PostgreSQL verifier covers nullable Assistant and Endpoint creators, retained
  zero/negative constraint failures, cross-table transactional rollback, and Integration's
  `unknown:null` mapping;
- no service migration creates `audit_created_actor_immutable` or
  `reject_created_actor_change`, and the verifier confirms both are absent from the migrated
  database catalogs;
- every non-target implementation file remains unchanged;
- `git diff --check` passes.

Required commands include `bash -n bin/verify-phase3-migrations.sh`,
`bash bin/verify-phase3-migrations.sh`, the Phase 3 rollback validator, static update inventory
checks, and `git diff --check`.

## Risks

- Zero or negative IDs can still fail existing actor constraints.
- Creation attribution can be changed after insert because database-level immutability is no
  longer enforced; this is the explicit requested contract.
- Removing migration-local lock timeouts can permit longer lock waits unless deployment policy
  supplies a timeout.
- Editing an applied migration creates divergent history; deployment-state confirmation is a
  hard prerequisite.
- Validator changes could accidentally weaken unrelated Phase 3 coverage; retain every check not
  specifically coupled to the removed validation and timeout wrappers.

## Rollback

Before deployment, revert all nine implementation files. After rollout begins, stop all four
services, restore the coordinated same-point backup, and restart the previous release. Do not
rely on destructive down migrations.

## Acceptance Criteria

- Only the eight listed SQL migrations and `bin/verify-phase3-migrations.sh` change in
  implementation.
- Each run-backfill migration contains only its original direct `UPDATE` statements.
- The three positive-ID validation blocks and all four lock-timeout wrappers are absent.
- All creation-actor immutability trigger functions and attachments are absent.
- Direct mappings and update inventories are unchanged.
- The checked-in PostgreSQL and rollback validators, static checks, and independent read-only
  review pass.
