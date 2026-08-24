# RFC 0010: Remove Call Status Event From Webhooks

- Status: Accepted
- Owner: Assistant API
- Date: 2026-08-24

## Summary

Remove `status_event` from version 1 call webhook payloads. The typed `status` field is
the single source of truth for the call lifecycle state. Provider status reporter ringing
webhooks also stop assigning `source` to `provider_status_reporter`.

## Problem

Call ringing, outbound dispatched, and failed webhook payloads expose both `status` and
`status_event`. Producers derive `status_event` from the internal telephony event, while
`status` already carries the supported public lifecycle value. During development this
duplicate field adds an unnecessary contract and can give consumers two values to
interpret for the same lifecycle transition.

## Verified Context

- `CallRingingWebhookPayload`, `CallOutboundDispatchedWebhookPayload`, and
  `CallFailedWebhookPayload` declare `StatusEvent`.
- Provider callback, outbound pipeline, and telephony reporter producers populate it.
- Webhook serialization and collector tests assert or construct the field.
- The payload version is still version 1 and the project is in development, so no
  compatibility bridge or new payload version is required by the contract owner.

## Goals

- Remove `status_event` from call webhook JSON.
- Keep typed `status` as the sole public lifecycle field.
- Stop assigning `provider_status_reporter` to the ringing webhook `source` field.
- Preserve internal telephony event mapping and metric labels.
- Keep all other webhook fields and event names unchanged.

## Non-Goals

- Do not change provider callback parsing or `TelephonyEvent` values.
- Do not change call metrics or their labels.
- Do not change conversation webhook payloads.
- Do not introduce a version 2 webhook payload.
- Do not refactor webhook construction beyond removing the field.

## Ownership and Scope

The Assistant API webhook contract owner owns all writable files for this change.

### Allowed Paths

- `api/assistant-api/internal/observability/webhook.go`
- `api/assistant-api/internal/observability/webhook_test.go`
- `api/assistant-api/internal/observability/recorder_test.go`
- `api/assistant-api/internal/observability/collectors/webhook/collector_test.go`
- `api/assistant-api/api/talk/callback.go`
- `api/assistant-api/api/talk/callback_webhook_test.go`
- `api/assistant-api/internal/channel/pipeline/outbound.go`
- `api/assistant-api/internal/channel/pipeline/webhook_test.go`
- `api/assistant-api/internal/channel/telephony/reporter.go`
- `api/assistant-api/internal/channel/telephony/reporter_test.go`
- `rfcs/0010-remove-call-status-event/`
- `rfcs/0010-remove-call-status-event.md`

### Out-of-Scope Paths

- Telephony provider implementations and callback parsers.
- UI provider configuration.
- Database models and migrations.
- Generated files and external SDKs.

## Proposed Design

Delete `StatusEvent` from the three call webhook payload structs and remove every
assignment at their construction sites. Remove the provider status reporter's
`Source: "provider_status_reporter"` assignment without changing the payload type. Update
focused tests to assert that serialized call webhook payloads do not contain
`status_event` and that reporter ringing payloads do not carry that source value.

No replacement field, conversion function, compatibility alias, or feature flag is
introduced. Internal telephony events remain available to routing and metrics before the
public webhook payload is constructed.

## Contracts and Compatibility

The version 1 JSON payload stops emitting `status_event` for `call.ringing`,
`call.outbound_dispatched`, and `call.failed`. The `status` field and its typed values do
not change. Ringing webhooks created by the provider status reporter stop carrying the
`provider_status_reporter` source value. This is intentionally accepted as a
development-phase contract change.

## Failure and Recovery

Compilation identifies any remaining payload construction using `StatusEvent`. JSON
tests verify the removed key is absent while `status` remains present.

## Security and Privacy

Removing the field reduces externally exposed provider and internal event detail. No new
data is collected or transmitted.

## Observability

Call status metrics continue using internal telephony events as labels. Webhook delivery
logging and delivery metrics remain unchanged.

## Data and Migration

None.

## Rollout

Deploy the Assistant API normally during development. Consumers must use `status` rather
than `status_event`.

## Rollback

Restore the three struct fields, producer assignments, and serialization assertions in
one revert. No persistent-data rollback is required.

## Alternatives Considered

- Keep both fields. Rejected because they duplicate lifecycle state for webhook consumers.
- Add a version 2 payload. Rejected because the contract owner confirmed the project is
  still in development.
- Retain `status_event` only on failures. Rejected because one canonical lifecycle field
  is simpler and provider error detail remains in `error` and `extra`.

## Testing and Verification

- Test call webhook serialization keeps `status` and omits `status_event`.
- Run focused tests for observability, callback, pipeline, and telephony packages.
- Run `make agent-finalize CHANGED_FILES="comma,separated,paths"` with the final changed
  file list.

## Acceptance Criteria

- [ ] No call webhook payload type exposes `StatusEvent`.
- [ ] No call webhook JSON contains `status_event`.
- [ ] Internal telephony event parsing and call metric labels remain unchanged.
- [ ] Existing call webhook `status` values remain unchanged.
- [ ] Provider status reporter ringing webhooks do not set `source` to
  `provider_status_reporter`.
- [ ] Focused tests and repository finalization pass.

## Challenge Resolution

The webhook contract owner approved removal during development. The review confirmed that
`status` remains the sole public lifecycle field, internal telephony events and metric
labels remain unchanged, and no compatibility bridge or version 2 payload is required.

## Artifact Index

- `jsons/plan.json`: approved implementation plan.
- `jsons/challenge.json`: approved contract-owner challenge receipt.
- `jsons/confirmation.json`: exact-digest implementation gate pending.
- `jsons/amendment-01-plan.json`: source-removal amendment plan.
- `jsons/amendment-01-challenge.json`: approved contract-owner amendment challenge.
- `jsons/amendment-01-confirmation.json`: amendment exact-digest gate pending.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-24 | Remove `status_event` during development | Webhook contract owner | User instruction |
| 2026-08-24 | Remove the provider status reporter source value | Webhook contract owner | User instruction |
