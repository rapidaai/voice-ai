# RFC 0009: Typed Call Webhook Status and Disconnect Reason

- Status: Accepted
- Owner: Assistant API team
- Created: 2026-08-23
- Updated: 2026-08-23
- Reviewers: Webhook contract owner

## Summary

Use typed values for call webhook status, direction, and disconnect reason. Remove the
duplicate `reason` field from terminal call webhook payloads and emit only the approved
`disconnect_reason` values.

## Context

Call webhook payloads currently expose `status`, `direction`, and `disconnect_reason` as
unrestricted strings. Terminal payloads also contain both `reason` and
`disconnect_reason`, allowing conflicting values. Several producers currently send raw
provider reasons or error text through these fields.

The webhook package already contains draft `WebhookCallStatus` and
`WebhookCallDisconnectReason` types. All call webhook producers must use these types and
assign values inline without introducing mapping helpers or additional logic.

## Goals

- Type every call webhook `status` field as `WebhookCallStatus`.
- Type every call webhook `direction` field as `WebhookCallDirection`.
- Type terminal call webhook `disconnect_reason` fields as
  `WebhookCallDisconnectReason`.
- Remove `reason` from `CallFailedWebhookPayload` and `CallEndedWebhookPayload`.
- Prevent arbitrary provider strings and error messages from being emitted as typed values.
- Preserve existing webhook event names and payload field names.

## Non-Goals

- Do not change conversation webhook payloads.
- Do not add conversion helpers, lookup tables, or provider-specific abstractions.
- Do not change telephony provider callback parsing.
- Do not change persisted call-context fields.

## Scope and Ownership

### Allowed Paths

- `api/assistant-api/internal/observability/webhook.go` - webhook enum and payload ownership.
- `api/assistant-api/internal/observability/webhook_test.go` - webhook contract tests.
- `api/assistant-api/api/talk/callback.go` - provider callback webhook construction.
- `api/assistant-api/internal/channel/pipeline/` - call pipeline webhook construction.
- `api/assistant-api/internal/channel/telephony/reporter.go` - provider status webhook construction.
- `api/assistant-api/sip/pipeline/` - SIP call webhook construction.
- Focused tests in each changed package.
- `rfcs/0009-call-webhook-status-and-disconnect-reason/` - governed workflow evidence.

### Out-of-Scope Paths

- Conversation webhook payloads and producers.
- Provider callback request types.
- Database models and migrations.
- Generated protobuf and SDK files.

## Proposed Design

Add `WebhookCallDirection` with `inbound` and `outbound` values. Use the existing
`WebhookCallStatus` and `WebhookCallDisconnectReason` types for all call webhook payload
fields.

Call event status assignments are:

- Received, answered, and started: `in_progress`.
- Ringing: `ringing`.
- Outbound requested: `pending`.
- Outbound dispatched: `in_progress`.
- Ended: `completed`.
- Failed: `failed`.
- Cancelled failure events: `cancelled`.

Disconnect reason assignments are:

- Provider callback failure: `provider_failed`.
- Raw pipeline or media error: `media_failed` when media-owned, otherwise `internal_error`.
- Successful talk completion: `assistant_ended`.
- Transfer completion: `transferred`.
- Provider completed callback or remote endpoint termination: `remote_hangup`.
- Existing busy, no-answer, rejected, cancellation, authentication, configuration,
  network, capacity, timeout, tool, and internal failure paths use their corresponding
  enum values.
- Missing or unrecognized reasons use `unknown`.

Assignments remain inline at each payload construction site. No new conversion function
or shared mapping layer is introduced.

## Contracts and Compatibility

- JSON keys remain `status`, `direction`, and `disconnect_reason`.
- `reason` is removed from `call.failed` and `call.ended` payloads.
- Status values change to the lowercase enum values in this RFC.
- Provider-specific details remain available through `error`, `status_event`, or `extra`.
- Existing webhook consumers relying on `reason` or uppercase status values require an
  coordinated update.

## Failure and Recovery

- Every call webhook construction must compile against the typed fields.
- Unknown external values must use `unknown`, not a cast from an arbitrary string.
- Compilation failures identify any missed call webhook producer.

## Security and Privacy

Raw provider errors must not be copied into `disconnect_reason`. Existing error-detail
fields remain unchanged and must not gain new sensitive information.

## Observability

Existing webhook request logging and delivery metrics remain unchanged.

## Data and Migration

None.

## Rollout

Deploy the assistant API and update webhook consumer documentation together. Stop rollout
if contract tests show an unapproved status, direction, or disconnect reason.

## Rollback

Revert the enum field types, restore the two `reason` fields, and restore the prior call
webhook assignments. No persistent-data rollback is required.

## Alternatives Considered

- Keep unrestricted strings. Rejected because producers can emit incompatible values.
- Add mapping helpers. Rejected because the requested implementation requires direct,
  inline assignments.
- Keep both `reason` and `disconnect_reason`. Rejected because they duplicate terminal
  cause ownership.

## Testing and Verification

- Test every enum wire value.
- Test JSON serialization for all call webhook payloads.
- Test that terminal call payloads omit `reason` and include typed `disconnect_reason`.
- Run focused tests for every changed package.
- Run `make agent-finalize CHANGED_FILES="comma,separated,paths"`.

## Acceptance Criteria

- [ ] All call webhook status fields use `WebhookCallStatus`.
- [ ] All call webhook direction fields use `WebhookCallDirection`.
- [ ] Terminal call webhook disconnect fields use `WebhookCallDisconnectReason`.
- [ ] `CallFailedWebhookPayload` and `CallEndedWebhookPayload` have no `reason` field.
- [ ] No call webhook producer casts an arbitrary string into a typed field.
- [ ] All approved checks pass.

## Open Questions

None.

## Challenge Resolution

The webhook contract owner approved the inline mappings and requested no mapping helpers
or additional logic. The design preserves raw provider details outside the typed fields.

## Artifact Index

- `jsons/plan.json` - approved implementation plan.
- `jsons/challenge.json` - webhook contract owner challenge decision.
- `jsons/confirmation.json` - pending exact-digest confirmation receipt.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-23 | Use prefixed call webhook enums and inline assignments | Webhook contract owner | `jsons/challenge.json` |
