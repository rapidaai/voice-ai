# RFC 0016: Canonical EOS Parameters

- Status: Accepted
- Owner: Voice AI
- Created: 2026-09-10
- Updated: 2026-09-10
- Reviewers: plan-challenger, coordinator

## Summary

Use canonical-only LiveKit and Pipecat EOS parameters in the UI and backend.
Migration 000061 renames stored keys in place and deletes known obsolete keys.
It is forward-only; recovery requires a pre-migration backup.

Amendment 01 replaces the original journal-based migration following the user's
explicit approval to simplify it. The original RFC and receipts remain historical
evidence in commit 43a713f3 and the JSON directory. UI/backend behavior is unchanged.

## Context

EOS selection is the `microphone.eos.provider` option, not the audio provider column.
Options have application-generated IDs and unique (key, deployment audio ID) pairs.
The pending migration has only run in disposable test databases, not live databases.
Exact row-image journaling and conflict-aware down migration are unnecessary for the
requested one-time key conversion once backup-based recovery is accepted.

## Goals

- Keep existing canonical values and manually selected models.
- Rename the highest-priority alias only when its canonical key is absent.
- Preserve option values, IDs, audit fields, backend paths, and unrelated settings.
- Keep the SQL direct, atomic, and idempotent without a journal or temporary tables.

## Non-Goals

- No runtime legacy support, new defaults in storage, or automatic model selection.
- No VAD, dispatch, STT transport, telephony, model-asset, or public API changes.
- No live migration, push, or resolution of the existing model-quality hold.

## Scope and Ownership

### Allowed Paths

The coordinator owns this amendment's changes:
- `api/assistant-api/migrations/000061_eos_canonical_parameters.up.sql`
- `api/assistant-api/migrations/000061_eos_canonical_parameters.down.sql`
- `bin/verify-eos-options-migration.sh`
- `tests/integration/eos-options/`
- `rfcs/0016-eos-canonical-parameters.md`
- `rfcs/0016-eos-canonical-parameters/jsons/`

The UI/backend implementation already present remains under the original plan's
ownership and is not changed by this amendment.

### Out-of-Scope Paths

- UI, backend runtime, other service migrations, and startup behavior.
- `bin/verify-phase3-migrations.sh`: retain the existing assistant-api 60-to-61
  change; no additional web-api expectation change is authorized.

## Proposed Design

Use one transaction with bounded lock and statement timeouts. Exclude concurrent
writes to the deployment and option tables while executing two statements:

1. UPDATE selected aliases in place, only when no canonical or higher-priority
   alias exists for that input deployment and its selected EOS provider.
2. DELETE remaining known obsolete keys for those same provider-scoped inputs.

All matching input records are included, including inactive records. Missing and
unknown providers, silence-based EOS, and output audio are excluded.

| Provider | Destination | Presence precedence |
| --- | --- | --- |
| LiveKit | quick_timeout | quick_timeout, fallback_timeout, timeout |
| LiveKit | extended_timeout | extended_timeout, silence_timeout |
| Pipecat | fallback_timeout | fallback_timeout, timeout |
| Pipecat | extended_timeout | extended_timeout, silence_timeout |

All keys use the `microphone.eos.` prefix. Delete LiveKit fallback_timeout, timeout,
and silence_timeout after promotion. Delete Pipecat timeout, silence_timeout,
quick_timeout, max_history_turns, and model after promotion.

## Contracts and Compatibility

LiveKit retains threshold, quick_timeout, extended_timeout, max_history_turns, and
model. Pipecat retains threshold, fallback_timeout, and extended_timeout. Existing
UI/backend defaults and canonical-only readers remain unchanged. Silence-based EOS
continues to use timeout. All three backend model-path override keys are preserved.

This migration operates on key presence, not numeric values. It never parses,
casts, resets, or rewrites stored values, including unusual or malformed text.
There is no invalid-value fallback to a lower-priority alias. Operators must audit
selected values before rollout; that manual audit is not enforced by migration SQL.

## Failure and Recovery

Lock timeout, statement timeout, or statement failure rolls back the transaction.
Successful repeat up is a no-op. The down file always raises an actionable error
requiring backup restoration; it does not attempt a lossy reverse rename.

## Security and Privacy

Only known keys on selected input deployments are changed. Diagnostics contain
counts and IDs, not configuration contents, credentials, or audio.

## Observability

The disposable verifier reports target counts by provider, changed and deleted row
counts from test snapshots, clean schema version 61, and zero target obsolete keys.
No recovery table is created solely for diagnostics.

## Data and Migration

Update and delete existing option rows only. No new configuration rows or persistent
tables are created. Tests compare complete row images so audit-field and unrelated
setting preservation are verified without production recovery infrastructure.

## Rollout

Stop configuration writers and old instances. Back up the relevant deployment and
option tables, audit selected values and key conflicts, apply 000061 externally,
verify clean version 61 and zero obsolete target keys, then deploy canonical-only
UI/backend code. Do not rely on startup to block a failed migration.

The known full-history verifier mismatch (web-api expects 12 but reaches 13) still
blocks shipping until separately authorized and corrected. No live run is approved.

## Rollback

Keep writers stopped, restore the pre-migration backup and matching migration
version metadata, and redeploy compatible application code. Deleted conflicting
aliases cannot be reconstructed from the surviving canonical values. Do not invoke
down or force-clear dirty metadata as a substitute for backup restoration.

## Alternatives Considered

- Exact row-image journal and guarded automatic rollback: rejected as unnecessary
  complexity for this one-time conversion after the user accepted backup recovery.
- Reimplement numeric parsing in SQL: rejected because this is a key-only migration.
- Silent no-op down: rejected because it would falsely report restoration.

## Testing and Verification

- `bash bin/verify-eos-options-migration.sh`: provider scope, all precedence cases,
  exact values/audit preservation, no persisted defaults, no extra tables, repeat up,
  concurrent-writer timeout, statement failure atomicity, and explicit down refusal.
- `bash -n bin/verify-eos-options-migration.sh`
- `just agent-finalize "api/assistant-api/migrations/000061_eos_canonical_parameters.up.sql,api/assistant-api/migrations/000061_eos_canonical_parameters.down.sql,bin/verify-eos-options-migration.sh,tests/integration/eos-options/seed.sql,tests/integration/eos-options/test_migration.py"`
- `git diff --check`

The existing full-change verification requirement remains in force before shipping.
No live database is used for this amendment's tests.

## Acceptance Criteria

- [ ] One transaction with direct UPDATE and DELETE and no recovery infrastructure.
- [ ] Canonical precedence, manual model selection, values, IDs, and audits preserved.
- [ ] Unrelated providers, output audio, unknown keys, and backend paths untouched.
- [ ] Repeat up makes no changes; failures are atomic; down requires a backup.
- [ ] Focused verification and independent review completed before shipping.

## Open Questions

None for the amendment. The separately identified full-history check remains blocked.

## Challenge Resolution

Amendment 01 has its own challenge and confirmation evidence. The prior review's
journal-specific diagnostics and rollback requirements no longer apply to this design.
The two-cycle limit applies to this user-requested amendment.

## Artifact Index

- `jsons/plan.json`, `jsons/approved-plan.json`, `jsons/challenge.json`, and
  `jsons/confirmation.json`: original design evidence, superseded for migration design.
- `jsons/review-implementation.json`: original implementation review.
- `jsons/amendment-01-plan.json`: forward-only migration plan.
- `jsons/amendment-01-challenge.json`: amendment's independent challenge.
- `jsons/amendment-01-confirmation.json`: amendment's exact-digest confirmation.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-09-10 | Canonical-only UI/backend with one-time migration | Voice AI | `jsons/plan.json` |
| 2026-09-10 | Simplify migration and require backup recovery | User | `jsons/amendment-01-plan.json` |
