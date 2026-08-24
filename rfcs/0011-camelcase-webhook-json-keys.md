# RFC 0011: Use lowerCamelCase for Rapida-Owned Assistant API Webhook JSON Keys

- Status: Accepted

## Summary

Change only Rapida-owned Assistant API webhook payload field names from snake_case to
lowerCamelCase before first release. Keep webhook event names, enum values, payload
version, configuration option keys, REST contracts, database schema, and unrelated
provider payloads unchanged.

## Context

The public Assistant API webhook payload structs live in
`api/assistant-api/internal/observability/webhook.go`. Those structs still expose
snake_case JSON keys such as `call_id`, `context_id`, `duration_ms`,
`disconnect_reason`, `message_count`, `session_id`, `media_session_id`,
`ice_latency_ms`, `peer_connection_state`, `restart_attempt`, and `restart_limit`.

The webhook collector in
`api/assistant-api/internal/observability/collectors/webhook/collector.go` forwards the
typed payload directly under the top-level `data` field and does not rename keys. That
makes the JSON tags in `webhook.go` the single source of truth for the public webhook
field-name contract.

Call, conversation, and WebRTC webhook producers already build typed payload structs with
Go field names such as `CallID`, `ContextID`, `DurationMs`, `DisconnectReason`,
`SessionID`, and `MediaSessionID`. Those producers do not depend on the serialized JSON
key spelling. This project is still before first release, so the contract owner accepts a
direct field-name change without a compatibility bridge, version bump, or dual-field
payload.

## Goals

- Emit lowerCamelCase JSON keys for every Rapida-owned Assistant API webhook payload
  field defined in `api/assistant-api/internal/observability/webhook.go`.
- Preserve existing webhook event names, enum values, payload version, top-level
  envelope fields, and `omitempty` behavior.
- Prove the new field names through focused serialization and collector tests.
- Keep the implementation to the smallest complete contract change.

## Non-Goals

- Do not change webhook event names such as `call.ringing` or `call.failed`.
- Do not change enum values such as `in_progress`, `provider_failed`, or
  `remote_hangup`.
- Do not change configuration option keys such as `http_url`, `http_headers`,
  `max_retry_count`, `retry_status_codes`, or `timeout_seconds`.
- Do not change REST request or response payloads, OpenAPI files, protobuf contracts, or
  database schema.
- Do not change provider callback payloads, SIP payloads, telephony provider request
  bodies, or other non-webhook snake_case contracts.
- Do not add a version 2 payload, feature flag, translation helper, custom marshaler, or
  compatibility alias.

## Scope and Ownership

This RFC covers every Rapida-owned webhook payload struct in
`api/assistant-api/internal/observability/webhook.go`, including call, conversation, and
WebRTC webhook payloads. That resolves the planning-stage scope question in favor of one
consistent outbound contract across all Rapida-owned Assistant API webhook payloads.

### Allowed Paths

- `api/assistant-api/internal/observability/webhook.go`: authoritative webhook payload
  JSON tags.
- `api/assistant-api/internal/observability/webhook_test.go`: focused webhook
  serialization coverage.
- `api/assistant-api/internal/observability/collectors/webhook/collector_test.go`:
  end-to-end collector request-body assertions.
- `rfcs/0011-camelcase-webhook-json-keys/`: governed workflow artifacts.
- `rfcs/0011-camelcase-webhook-json-keys.md`: this RFC.

### Out-of-Scope Paths

- `api/assistant-api/internal/observability/collectors/webhook/collector.go`
- `api/assistant-api/api/talk/callback.go`
- `api/assistant-api/internal/channel/pipeline/`
- `api/assistant-api/internal/channel/telephony/`
- `api/assistant-api/sip/pipeline/`
- `api/assistant-api/internal/channel/telephony/internal/**`
- `api/assistant-api/sip/internal/**`
- `openapi/**`
- `protos/**`
- `ui/**`
- Database migrations and entity schemas

## Proposed Design

Update only the JSON tags on Rapida-owned webhook payload structs in
`api/assistant-api/internal/observability/webhook.go`. The collector already serializes
the typed payload directly, so no rekeying layer, adapter, or producer refactor is
required.

The field-name mapping is:

- `call_id` -> `callId`
- `context_id` -> `contextId`
- `duration_ms` -> `durationMs`
- `disconnect_reason` -> `disconnectReason`
- `message_count` -> `messageCount`
- `session_id` -> `sessionId`
- `media_session_id` -> `mediaSessionId`
- `ice_latency_ms` -> `iceLatencyMs`
- `peer_connection_state` -> `peerConnectionState`
- `restart_attempt` -> `restartAttempt`
- `restart_limit` -> `restartLimit`

Top-level webhook envelope fields remain `assistant`, `conversation`, `data`, and
`event`. Payload contents remain typed Go structs owned by the webhook package. Producers
continue constructing those structs inline with the same field ownership and event timing
they use today.

## Contracts and Compatibility

- The only intentional contract change is the spelling of Rapida-owned webhook payload
  JSON field names from snake_case to lowerCamelCase.
- Webhook event names remain unchanged.
- Enum values remain unchanged.
- The payload version remains unchanged.
- Top-level envelope field names remain unchanged.
- `omitempty` behavior remains unchanged.
- REST contracts, database schema, provider-specific payloads, and configuration option
  keys remain unchanged.

This is intentionally a pre-release breaking contract change for any early webhook
consumer already reading snake_case payload keys. No compatibility alias is provided
because the contract owner chose a clean pre-release cut over carrying two key styles.

## Failure and Recovery

The main implementation risk is a partial tag conversion that leaves mixed key styles
across webhook payload families. Focused serialization tests must assert both the
presence of the new lowerCamelCase keys and the absence of the old snake_case keys so
regressions fail loudly.

If a consumer still expects snake_case keys, delivery succeeds but that consumer may fail
to read the renamed fields. Recovery is operationally simple during development: revert
the tag changes and the focused test expectations in one change.

## Security and Privacy

This is a naming-only contract change on fields that are already emitted. No new data is
collected, persisted, or exposed. Authentication, authorization, tenant isolation,
secrets handling, and privacy boundaries remain unchanged.

## Observability

Webhook delivery logging, metrics, and tracing remain unchanged. Internal observability
attributes and metric labels keep their current naming because they are not part of the
outbound webhook contract. Focused tests provide the main evidence that the public JSON
contract changed only where intended.

## Data and Migration

None. No persistent-data or schema change exists.

Operational migration is consumer-facing only: any early webhook consumer must switch its
field reads from snake_case to lowerCamelCase before relying on the pre-release payload
contract.

## Rollout

Deploy the Assistant API change normally during development. Update any owned webhook
examples or external consumer guidance to use lowerCamelCase payload keys at the same
time, even though those artifacts are outside this repository's writable scope.

Stop rollout if:

- a focused serialization or collector test still shows snake_case keys,
- a non-webhook or provider payload contract changes unintentionally, or
- a reviewer finds scope expansion outside `webhook.go` and the focused tests.

## Rollback

Revert the JSON tag updates in `api/assistant-api/internal/observability/webhook.go` and
restore the old snake_case expectations in
`api/assistant-api/internal/observability/webhook_test.go` and
`api/assistant-api/internal/observability/collectors/webhook/collector_test.go`.

No data rollback, schema rollback, or staged migration rollback is required.

## Alternatives Considered

- Keep snake_case until after first release. Rejected because the request is explicitly
  to make the contract correction before release and avoid carrying the wrong field names
  forward.
- Add a collector-side map rekeying layer. Rejected because `webhook.go` already owns
  the contract and a second mapping layer would duplicate ownership.
- Emit both snake_case and lowerCamelCase fields. Rejected because the project is still
  pre-release and dual fields would create ambiguity with no current need.
- Broadly replace snake_case tags across Assistant API packages. Rejected because many of
  those tags belong to protected REST, provider, transport, config, or persistence
  contracts that must remain unchanged.

## Testing and Verification

Required coverage:

- webhook payload serialization tests for call, conversation, and WebRTC payload structs
- webhook collector request-body assertions for serialized `data` payload JSON
- package-level regression coverage for payload producers that compile against the shared
  payload types

Exact commands:

```bash
go test ./api/assistant-api/internal/observability
go test ./api/assistant-api/internal/observability/collectors/webhook
go test ./api/assistant-api/api/talk ./api/assistant-api/internal/adapters/internal ./api/assistant-api/internal/channel/pipeline ./api/assistant-api/internal/channel/telephony ./api/assistant-api/internal/channel/webrtc/... ./api/assistant-api/sip/pipeline
make agent-finalize CHANGED_FILES="api/assistant-api/internal/observability/webhook.go,api/assistant-api/internal/observability/webhook_test.go,api/assistant-api/internal/observability/collectors/webhook/collector_test.go"
```

Expected evidence:

- serialization tests show lowerCamelCase keys and confirm the old snake_case keys are
  absent
- collector tests show lowerCamelCase keys under `data` and no top-level `contextId`
  field
- targeted producer-package tests still compile and pass without relying on JSON tag
  spellings
- `make agent-finalize` passes for the final changed-file list

## Acceptance Criteria

- [ ] Every Rapida-owned Assistant API webhook payload field defined in
      `api/assistant-api/internal/observability/webhook.go` uses lowerCamelCase JSON
      tags.
- [ ] Webhook event names, enum values, configuration option keys, REST contracts,
      database schema, and unrelated provider payloads remain unchanged.
- [ ] The webhook collector continues to wrap the typed payload under `assistant`,
      `conversation`, `data`, and `event` without introducing a remapping layer.
- [ ] Focused serialization tests assert the new lowerCamelCase keys and prove the old
      snake_case keys are absent.
- [ ] Focused collector tests assert lowerCamelCase keys inside `data` and keep the
      top-level context omission behavior unchanged.
- [ ] The targeted test commands and `make agent-finalize` pass for the final changed
      file list.

## Open Questions

None.

## Challenge Resolution

The planning-stage scope question is resolved by including all Rapida-owned webhook
payload structs in `webhook.go`, not only telephony call payloads. The design keeps the
contract change at the single ownership point for outbound webhook keys and rejects
collector-side translation, dual-key compatibility, and broader snake_case cleanup
outside webhook payload ownership. The first challenge identified that producer-package
verification omitted conversation and WebRTC producers. The verification command now
explicitly covers `internal/adapters/internal` and `internal/channel/webrtc/...`.

## Artifact Index

- `jsons/plan.json`: approved planning artifact for this RFC.
- `jsons/challenge.json`: challenge receipt for the exact RFC bytes in this file.
- `jsons/confirmation.json`: exact-digest confirmation receipt required before
  implementation.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-24 | Change only Rapida-owned webhook payload field names to lowerCamelCase before first release | Webhook contract owner | `jsons/plan.json` |
| 2026-08-24 | Keep event names, enums, envelope fields, config keys, REST contracts, and provider payloads unchanged | Webhook contract owner | `jsons/plan.json` |
