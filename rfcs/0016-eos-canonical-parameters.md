# RFC 0016: Canonical EOS Parameters

- Status: Accepted
- Owner: Voice AI
- Created: 2026-09-10
- Updated: 2026-09-10
- Reviewers: plan-challenger, coordinator

## Summary

Use canonical-only LiveKit and Pipecat end-of-speech (EOS) parameters in the UI and
assistant-api runtime. Existing stored assistant deployment audio options are translated
once by a reversible migration before alias readers are removed from the UI and backend.

This RFC intentionally rejects ongoing legacy support. Legacy keys are handled only by the
one-time migration; after that, LiveKit and Pipecat read their canonical keys only.

## Context

EOS provider selection is currently an audio option named `microphone.eos.provider` in
`assistant_deployment_audio_options`. It is not stored in
`assistant_deployment_audios.audio_provider`, so migration scope must be discovered by
joining options to `assistant_deployment_audios` and reading the provider option row.

The UI provider JSON files already expose the intended canonical controls:

- Pipecat: `threshold`, `fallback_timeout`, `extended_timeout`.
- LiveKit: `threshold`, `quick_timeout`, `extended_timeout`, `max_history_turns`, `model`.

However, `ui/src/app/components/providers/end-of-speech/provider.tsx` still translates
legacy keys during hydration. LiveKit and Pipecat constructors also still read aliases in
`api/assistant-api/internal/end_of_speech/internal/livekit/` and
`api/assistant-api/internal/end_of_speech/internal/pipecat/`.

Migration `000060` removed `created_by` and `updated_by`; current audio option rows use
`created_actor_type`, `created_actor_id`, `updated_actor_type`, and `updated_actor_id`.
Audio option IDs are positive application-generated IDs, not SQL-generated IDs.

The application startup path logs migration failures and can force-clear dirty migration
state. That startup behavior is not a rollout gate for this change. The EOS option
migration must be applied and verified externally before deploying canonical-only backend
and UI code.

Prior model-quality and model-asset concerns are not addressed by this parameter migration.

## Goals

- Use canonical-only LiveKit and Pipecat EOS settings in the UI and backend.
- Translate existing stored legacy LiveKit and Pipecat EOS keys once through migration
  code.
- Preserve selected EOS provider, manual LiveKit model selection, valid configured
  values, backend model path overrides, and unrelated metadata.
- Keep missing canonical values absent in storage and let existing UI/backend defaults
  supply the value at read time.
- Remove obsolete provider-specific EOS keys for migrated LiveKit and Pipecat records.
- Provide reversible, idempotent, concurrent-edit-safe migration behavior.
- Keep the current audited schema compatible.
- Verify migration, UI, and backend behavior with provider-scoped tests.

## Non-Goals

- No live database rollout or environment-specific execution in this RFC.
- No VAD, dispatch, STT transport, telephony, OpenAPI, protobuf, or public API changes.
- No model-asset migration, model download, model regeneration, or model-quality approval.
- No blanket reset of custom EOS settings.
- No generic helper packages, shared migration utilities, or permanent compatibility layer.
- No inference of EOS provider from `assistant_deployment_audios.audio_provider`.
- No change to silence-based EOS. `microphone.eos.timeout` remains canonical for
  `silence_based_eos`.

## Scope and Ownership

### Allowed Paths

- `rfcs/0016-eos-canonical-parameters.md` - coordinator after the RFC author's completed draft.
- `rfcs/0016-eos-canonical-parameters/jsons/` - coordinator-owned lifecycle artifacts.
- `api/assistant-api/migrations/000061_eos_canonical_parameters.up.sql` - migration owner.
- `api/assistant-api/migrations/000061_eos_canonical_parameters.down.sql` - migration owner.
- `bin/verify-eos-options-migration.sh` - migration verification owner.
- `tests/integration/eos-options/` - migration verification owner.
- `bin/verify-phase3-migrations.sh` - migration verification owner, only to advance the
  assistant-api expected version from 60 to 61.
- `ui/src/app/components/providers/end-of-speech/provider.tsx` - UI EOS owner.
- `ui/src/app/components/providers/speech-to-text/provider.tsx` - UI microphone defaults owner.
- `ui/src/app/components/providers/__tests__/audio-input-advanced-defaults-parity.test.ts` - UI EOS owner.
- `ui/src/app/components/providers/end-of-speech/__tests__/provider-runtime-parity.test.tsx` - UI EOS owner.
- `ui/src/app/pages/assistant/actions/create-deployment/commons/configure-audio-input.tsx` - UI provider-switch owner.
- `ui/src/app/pages/assistant/actions/create-deployment/commons/__tests__/configure-audio-input.design.test.tsx` - UI audio configuration owner.
- `ui/src/providers/__tests__/provider-eos-config.test.ts` - UI provider config owner.
- `api/assistant-api/internal/end_of_speech/internal/livekit/constant.go` - backend EOS owner.
- `api/assistant-api/internal/end_of_speech/internal/livekit/livekit_end_of_speech.go` - backend EOS owner.
- `api/assistant-api/internal/end_of_speech/internal/livekit/livekit_end_of_speech_integration_test.go` - backend EOS owner.
- `api/assistant-api/internal/end_of_speech/internal/pipecat/constant.go` - backend EOS owner.
- `api/assistant-api/internal/end_of_speech/internal/pipecat/pipecat_end_of_speech.go` - backend EOS owner.
- `api/assistant-api/internal/end_of_speech/internal/pipecat/pipecat_end_of_speech_test.go` - backend EOS owner.
- `api/assistant-api/internal/end_of_speech/internal/pipecat/pipecat_end_of_speech_integration_test.go` - backend EOS owner.
- `api/assistant-api/internal/options/audio.go` - option registry owner.
- `api/assistant-api/internal/options/audio_test.go` - option registry owner.

### Out-of-Scope Paths

- `api/assistant-api/internal/vad/**`
- `api/assistant-api/internal/adapters/internal/dispatch_handler.go`
- `api/assistant-api/internal/adapters/internal/dispatch.go`
- `api/assistant-api/internal/transformer/**`
- `api/assistant-api/internal/channel/**`
- `api/assistant-api/internal/end_of_speech/internal/livekit/models/**`
- `api/assistant-api/internal/end_of_speech/internal/pipecat/models/**`
- `api/assistant-api/internal/end_of_speech/internal/livekit/testdata/benchmark/**`
- `api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/benchmark/**`
- `openapi/**`
- `protos/**`
- `api/integration-api/**`

## Proposed Design

### Parameter Contract

All keys below use the `microphone.eos.` prefix.

LiveKit canonical keys:

- `threshold`, default `0.0289`.
- `quick_timeout`, default `250` milliseconds.
- `extended_timeout`, default `3000` milliseconds.
- `max_history_turns`, default `6`.
- `model`, default `en`.

Pipecat canonical keys:

- `threshold`, default `0.5`.
- `fallback_timeout`, default `500` milliseconds.
- `extended_timeout`, default `3000` milliseconds.

Backend-only keys to preserve through UI hydration and migration:

- `microphone.eos.livekit.model_path`
- `microphone.eos.livekit.tokenizer_path`
- `microphone.eos.pipecat.model_path`

Manual LiveKit model values remain valid when nonblank, including custom names. The UI must
not infer a model from language.

### Migration Mapping

LiveKit `quick_timeout` precedence:

1. `quick_timeout`
2. `fallback_timeout`
3. `timeout`

LiveKit `extended_timeout` precedence:

1. `extended_timeout`
2. `silence_timeout`

LiveKit removes after migration:

- `fallback_timeout`
- `timeout`
- `silence_timeout`

Pipecat `fallback_timeout` precedence:

1. `fallback_timeout`
2. `timeout`

Pipecat `extended_timeout` precedence:

1. `extended_timeout`
2. `silence_timeout`

Pipecat removes after migration:

- `timeout`
- `silence_timeout`
- `quick_timeout`
- `max_history_turns`
- `model`

Canonical threshold values keep their existing row when present and valid. Missing canonical
threshold values are not persisted as duplicate default rows; existing defaults continue to
apply at read time.

### UI Changes

`GetDefaultEOSConfig` becomes canonical-only. It removes alias arrays, custom numeric
parsing, hex-float preservation, and legacy fallback retention. It uses provider JSON and
the existing config/default APIs for canonical keys.

EOS hydration must still preserve:

- unrelated non-EOS metadata,
- selected provider row,
- existing canonical provider values,
- manual LiveKit model and history values,
- backend-only EOS model path overrides.

Provider switching removes stale provider-specific controls for the newly selected provider
while preserving unrelated metadata and backend model path overrides.

The provider-switch handler in `configure-audio-input.tsx` must retain backend-only EOS
model paths before filtering EOS-prefixed metadata. Fixing hydration alone cannot recover
values that this caller has already discarded. Old provider-specific controls still clear
when the provider changes.

### Backend Changes

LiveKit and Pipecat constructors must read only canonical keys after migration. This means:

- Remove LiveKit reads for `fallback_timeout`, `timeout`, and `silence_timeout`.
- Remove Pipecat reads for `timeout` and `silence_timeout`.
- Remove private alias constants in both providers.
- Remove `MicrophoneEOSOptionLegacySilenceTimeout` from
  `api/assistant-api/internal/options/audio.go` if no references remain.
- Keep `MicrophoneEOSOptionTimeout` because it is canonical for silence-based EOS.

This exact removal is required to satisfy canonical-only behavior. Keeping backend alias
readers would preserve ongoing legacy support and would violate the user requirement.

## Contracts and Compatibility

- The one-time migration is the only compatibility bridge.
- Runtime code does not support LiveKit/Pipecat legacy aliases after migration.
- Missing canonical rows remain valid because existing UI/backend defaults supply absent
  values.
- Valid canonical rows are preserved and have priority over aliases.
- Valid zero timeout or threshold values are preserved because current runtime behavior does
  not add tuning bounds.
- Non-target providers and unknown provider values are left unchanged.
- `audio_provider` remains unrelated to EOS provider selection.
- Backend model path keys are configuration overrides, not user-facing EOS controls.

Intentional breaking change:

- A deployment that still stores only LiveKit or Pipecat legacy keys after migration and
  verification will no longer be configured by backend aliases.

## Failure and Recovery

Migration preflight validates candidate values before changing rows. Presence precedence is
canonical first, then aliases. If a present higher-priority candidate is invalid, the
migration must report and abort instead of silently choosing a lower-priority value.

Accepted numeric values:

- finite decimal or exponent text,
- Go-compatible digit separators,
- no outer whitespace,
- no PostgreSQL-only numeric coercion.

Rejected numeric values:

- non-finite values such as `NaN` and infinities,
- malformed values,
- hex-float values,
- out-of-range values,
- values requiring manual review.

`max_history_turns` must be a positive integer. `model` must be nonblank.

The migration runs atomically with bounded lock and statement timeouts. It must block
concurrent option/deployment writes while reading, journaling, and modifying the target set.

Rollback compares current rows and provider identity with journaled post-migration state
before changing anything. If any affected row was edited after migration, if provider
identity changed, if a destination conflict exists, or if parent deployment rows are
missing, rollback aborts the whole transaction and retains the journal.

## Security and Privacy

This change does not add permissions, secrets, credentials, or user data access. Backend
model path values are treated as stored configuration and are preserved without surfacing
new UI controls.

The migration must preserve audited actor fields exactly for renamed, deleted, and restored
rows. It must not invent a user identity. It must not use removed `created_by` or
`updated_by` columns.

## Observability

No new application runtime metrics are required.

Operator diagnostics are required for migration execution and rollback:

- target row counts by provider,
- changed row counts,
- deleted obsolete key counts,
- invalid candidate report,
- journal row counts,
- clean migration version 61,
- post-migration target alias count zero.

These diagnostics belong in `bin/verify-eos-options-migration.sh` and disposable database
verification, not in the application hot path.

## Data and Migration

Migration files:

- `api/assistant-api/migrations/000061_eos_canonical_parameters.up.sql`
- `api/assistant-api/migrations/000061_eos_canonical_parameters.down.sql`

Target set:

- All input-audio deployment rows whose stored `microphone.eos.provider` exactly equals
  `livekit_eos` or `pipecat_smart_turn_eos`.
- Include inactive target records so later activation does not restore old keys.
- Do not guess missing or unknown EOS providers.
- Do not target output audio.

Up migration algorithm:

1. Start a transaction with bounded lock and statement timeouts.
2. Lock the target audio option/deployment rows against concurrent writes.
3. Validate candidate values before changing any row.
4. Store original and resulting row images for changed or deleted rows in a
   migration-specific journal.
5. Preserve valid canonical rows.
6. When canonical is absent, rename the highest-priority valid alias row in place,
   preserving its ID and audit fields.
7. Remove lower-priority aliases and known obsolete provider-specific keys only after
   choosing the canonical survivor.
8. Keep unknown EOS keys, unrelated metadata, non-target providers, output audio, provider
   rows, and backend model path overrides unchanged.
9. Repeated application is a no-op and must not overwrite original journal images.

Down migration algorithm:

1. Start a transaction with the same write exclusion as the up migration.
2. Compare affected rows and provider identity with the journaled post-migration state.
3. Abort the whole rollback on later edits, provider switches, unique conflicts, or missing
   parent deployments.
4. Restore original row IDs, keys, values, status, and audit fields exactly.
5. Remove the migration journal only in the successful rollback transaction.

The migration must not contact or alter any live database as part of implementation review.
Environment-specific execution is a later operational step.

## Rollout

1. Ship migration files and verification scripts through the reviewed change.
2. In each environment, stop configuration writers and old application instances.
3. Run preflight inventory and disposable verification first.
4. Apply migration version 61 with the approved migration runner.
5. Verify clean migration version 61 and zero target aliases.
6. Deploy canonical-only backend and UI.
7. Stop rollout if migration preflight reports invalid values, dirty state, unexpected target
   counts, or nonzero target aliases after migration.

Startup migration logging is not a rollout approval signal.

## Rollback

Rollback before user edits:

1. Stop configuration writers and application instances.
2. Run the down migration.
3. Verify restored rows and clean version.
4. Redeploy the previous backend/UI that still supports aliases.

Rollback after user edits:

- The down migration aborts atomically and retains the journal.
- Operators must resolve conflicts explicitly instead of silently losing user edits.

Application rollback alone is insufficient after migration if canonical-only data has been
accepted by users. Database rollback and application rollback must be coordinated.

## Alternatives Considered

- Keep backend alias readers. Rejected because it preserves ongoing legacy support.
- Keep UI alias translation only. Rejected because runtime would still accept legacy rows
  and canonical-only behavior would be incomplete.
- Infer EOS provider from `audio_provider`. Rejected because EOS provider is stored as
  `microphone.eos.provider`.
- Persist defaults for every missing canonical value. Rejected because existing defaults
  already supply absent values and duplicated default rows add storage churn.
- Add a generic migration helper. Rejected because this is one focused provider migration.
- Clean every unknown `microphone.eos.*` key. Rejected because the migration only owns known
  obsolete LiveKit/Pipecat keys and must preserve unrelated metadata.

## Testing and Verification

Required test categories:

- Disposable PostgreSQL migration tests for both providers and every alias priority.
- Migration tests for valid canonical preservation, invalid candidate abort, missing values,
  inactive input rows, output audio exclusion, backend path preservation, repeated up, and
  rollback.
- Rollback tests for later edits, provider switches, unique conflicts, and transaction
  atomicity.
- UI tests for canonical hydration, ignored legacy values, provider switching, model path
  preservation, manual model/history preservation, and input ownership.
- Backend tests for canonical success, default/error paths, and ignored legacy aliases in
  real constructors.
- Option registry tests that keep provider JSON defaults in parity with backend defaults.

Exact commands:

```bash
bash bin/verify-eos-options-migration.sh
bash bin/verify-phase3-migrations.sh
env GOCACHE=/private/tmp/voice-ai-gocache go test -race -tags=integration -count=1 -timeout=120s ./api/assistant-api/internal/end_of_speech/... ./api/assistant-api/internal/options
yarn --cwd ui test --watchAll=false --watchman=false --runInBand src/app/components/providers/__tests__/audio-input-advanced-defaults-parity.test.ts src/app/components/providers/end-of-speech/__tests__/provider-runtime-parity.test.tsx src/providers/__tests__/provider-eos-config.test.ts src/app/pages/assistant/actions/create-deployment/commons/__tests__/configure-audio-input.design.test.tsx
just agent-finalize "$(git diff --name-only --diff-filter=ACMRT HEAD | paste -sd, -),$(git ls-files --others --exclude-standard | paste -sd, -)"
git diff --check
```

Environmental limitation:

- Live database execution is not part of this change. Verification must use disposable
  databases and repository test fixtures only.

## Acceptance Criteria

- [ ] UI and backend contain no active LiveKit/Pipecat alias read paths.
- [ ] Silence-based EOS behavior is unchanged.
- [ ] Migration preserves existing canonical values, manual models, custom finite values,
  zero, backend paths, non-target providers, and unrelated metadata.
- [ ] Migration tests cover both providers, every alias priority, invalid values, conflicts,
  missing values, inactive records, output audio, repeated execution, and rollback.
- [ ] Down migration restores original rows exactly and rejects later edits or conflicts
  atomically.
- [ ] UI tests cover canonical happy paths, ignored legacy values, backend model-path
  preservation, provider switching, and input ownership.
- [ ] Backend tests cover canonical values, default/error paths, and ignored legacy aliases.
- [ ] Full migration history reaches clean assistant-api version 61.
- [ ] No live database is used during implementation or verification.
- [ ] No model-quality approval is implied.
- [ ] Detached checkout is preserved through implementation.

## Open Questions

None.

## Challenge Resolution

The first challenge found that provider-switch model-path preservation required an omitted
production path, `configure-audio-input.tsx`. The plan and allowed paths now include that
caller and require preservation before its prefix filter. A second challenge is pending.

One correction cycle has been used; the limit is two.

## Artifact Index

- `jsons/plan.json` - coordinator-authored plan artifact, available.
- `jsons/challenge-01.json` - first challenge scope finding and requested correction.
- `jsons/challenge.json` - independent challenge receipt, pending.
- `jsons/confirmation.json` - exact-digest confirmation receipt, pending.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-09-10 | Use a one-time migration as the only compatibility bridge. | Voice AI | `jsons/plan.json` |
| 2026-09-10 | Remove LiveKit and Pipecat backend alias readers after migration. | Voice AI | `jsons/plan.json` |
| 2026-09-10 | Keep missing canonical values absent and use existing defaults at read time. | Voice AI | `jsons/plan.json` |
