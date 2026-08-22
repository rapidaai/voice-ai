# RFC 0001 Phase 3 Cleanup Runbook

## Preconditions

- Record the exact Web, Endpoint, Assistant, and Integration release identifiers.
- Confirm actor-capable binaries and clients are deployed everywhere before cleanup.
- Confirm every database reports zero rows with a null `created_actor_type`.
- Confirm actor-pair constraints and `audit_created_actor_immutable` triggers are valid on all 64 inventoried tables.
- Confirm repository validators report no production use of `created_by`, `updated_by`, `CreatedBy`, `UpdatedBy`, `createdUser`, or `updatedUser`.
- Confirm all known external consumers have moved to the actor-only major contract.

## Operational Limits

- Set `lock_timeout` to five seconds for every migration session.
- Abort on lock timeout, replica lag above the deployment threshold, unexpected WAL growth, disk pressure, or any failed validation query.
- Do not continue to the next database after any failed cleanup or verification step.

## Backup And Rollback Rehearsal

1. Enable the global write fence and stop mutation workers.
2. Capture one named, same-point backup set for Web, Endpoint, Assistant, and Integration.
3. Restore all four databases into the rehearsal environment.
4. Deploy the previous complete release against the restored set.
5. Run legacy audit smoke tests and record the backup identifiers, release identifier, commands, timestamps, and results.
6. Treat a single-database restore or a lossy down migration as invalid rollback evidence.

## Cleanup Order

1. Keep the global write fence active and verify old writers are stopped.
2. Apply and verify the Web cleanup migration.
3. Apply and verify the Endpoint cleanup migration.
4. Apply and verify the Assistant cleanup migration.
5. Apply and verify the Integration no-op cleanup checkpoint.
6. Deploy actor-only binaries, generated protobuf artifacts, SDKs, and UI.
7. Run database, source, contract, SDK, UI, and smoke-test validators.
8. Resume traffic only after all four services pass and the release coordinator records approval.

## Database Verification

For every table listed in `rfcs/0001-phase-3-audit-contract-inventory.json`:

- `created_actor_type` is non-null and satisfies the actor-pair constraint.
- `created_actor_id` is null only for `unknown` history.
- `updated_actor_type` and `updated_actor_id` are either both null or a valid pair.
- Updating either creation actor column is rejected by `audit_created_actor_immutable`.
- Legacy columns are absent after cleanup.

## Failure Handling

- Leave the write fence active after any failure.
- Stop subsequent cleanup steps immediately.
- Fix forward only when the failed database remains compatible with the actor-only binaries and all validators can be rerun.
- Otherwise restore the complete named four-database backup set, deploy the previous complete release, run legacy smoke tests, and resume traffic only after cross-service approval.

## Evidence Record

Record the following in the release pull request:

- accepted RFC path and confirmed digest
- release and previous-release identifiers
- nested SDK starting SHAs
- migration commands and per-database results
- validation command output
- backup and restore identifiers
- write-fence enable/disable timestamps
- independent review report and unresolved follow-ups
