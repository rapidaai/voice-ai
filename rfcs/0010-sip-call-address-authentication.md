# RFC 0010: Telephony Call Address in Assistant Authentication

- Status: Draft
- Owner: Assistant API Telephony and Authentication
- Created: 2026-08-24
- Updated: 2026-08-24
- Reviewers: Telephony, Authentication, UI, Security

## Summary

Replace telephony-owned `client.*` metadata and authentication sources with one provider-neutral
`telephony.*` contract. Every provider publishes actual call-direction From and To phone values.
Native SIP additionally publishes full party URIs. SIP headers remain internal to `CallAddress`.

This is an intentional breaking change. Removed telephony `client.*` keys have no runtime aliases.

## Context

All telephony streamers create a `ConversationInitialization` through the shared base streamer, and
assistant authentication resolves its metadata through the default variable registry. Today that
path publishes role-based keys such as `client.phone` and `client.assistant_phone`. Those names do
not represent wire-level From and To consistently across inbound and outbound calls.

Provider webhooks and outbound APIs already supply From and To values. Native SIP has the richer
`CallAddress` snapshot with parsed users, full URIs, and filtered headers. Authentication should
use one namespace while retaining provider-specific optional detail.

## Goals

- Publish `telephony.from_phone` and `telephony.to_phone` for every provider when available.
- Publish provider, direction, call ID, context ID, codec, and sample rate under `telephony.*`.
- Publish `telephony.from_uri` and `telephony.to_uri` for native SIP.
- Remove telephony fields from the authentication UI's `client` source.
- Remove runtime compatibility aliases for the replaced telephony `client.*` keys.
- Keep genuine client-device metadata such as timezone and user agent under `client.*`.

## Non-Goals

- Converting phone values to E.164.
- Resolving an agent SIP URI through an external DID lookup.
- Fabricating URI or header values for non-SIP providers.
- Changing authentication HTTP execution semantics.
- Changing protobuf definitions or database schemas.

## Scope and Ownership

### Allowed Paths

- `rfcs/0010-sip-call-address-authentication.md` and JSON artifacts
- `api/assistant-api/internal/type/telephony.go`
- `api/assistant-api/internal/channel/telephony/**`
- `api/assistant-api/sip/pipeline/runtime.go`
- `api/assistant-api/internal/observability/conversation.go`
- `api/assistant-api/internal/services/assistant/conversaction.impl.service.go`
- `api/assistant-api/internal/variable/namespace/registry.go`
- focused tests in the changed backend packages
- `ui/src/app/pages/assistant/actions/configure-assistant-authentication/**`
- `ui/src/app/pages/assistant/view/conversations/**`
- `ui/src/utils/prompt-reserved-variables.ts`

### Out-of-Scope Paths

- `protos/**`
- database migrations
- `api/assistant-api/internal/authentication/http/**`
- STT, TTS, VAD, EOS, and LLM implementations

## Proposed Design

The canonical runtime keys are:

| Runtime source | Availability |
| --- | --- |
| `telephony.from_phone` | Every provider when supplied |
| `telephony.to_phone` | Every provider when supplied |
| `telephony.direction` | Every provider |
| `telephony.provider` | Every provider |
| `telephony.provider_call_id` | When supplied |
| `telephony.context_id` | When supplied |
| `telephony.codec` | When known |
| `telephony.sample_rate` | When known |
| `telephony.from_uri` | Native SIP |
| `telephony.to_uri` | Native SIP |

`CallInfo` gains explicit `FromPhone` and `ToPhone` fields owned by each provider. Providers fill
them directly from webhook values or outbound arguments. Authentication never reconstructs wire
direction from the role-based `CallerNumber` and `FromNumber` fields.

For persisted call context data, From and To are assigned as follows:

- inbound: From is the caller number and To is the assistant/provider number
- outbound: From is the assistant/provider number and To is the called number

Native SIP uses its immutable `CallAddress` directly instead of re-deriving party values. The SIP
streamer adds URI fields to the base initialization metadata. Headers are not projected into
authentication metadata.

The default variable registry registers `telephony` over the `telephony.` metadata prefix. The
authentication executor remains unchanged because `registry.Apply` already maps runtime sources
to configured outgoing JSON keys.

The authentication UI replaces its telephony-oriented `Client` choices with a `Telephony` group.
The default body becomes:

```json
{
  "assistant.id": "assistantId",
  "telephony.from_phone": "fromPhone",
  "telephony.to_phone": "toPhone"
}
```

## Provider Mapping

- Exotel inbound: `CallFrom` to From and `CallTo` to To.
- Twilio inbound: `From` to From and `To` to To.
- Other inbound providers map their existing caller and destination fields when present.
- Every outbound provider maps its existing `fromPhone` and `toPhone` arguments directly.
- Native SIP maps phone and URI fields from `CallAddress`.
- Missing provider values are omitted, not replaced with empty strings or role guesses.

## Contracts and Compatibility

- `client.phone`, `client.assistant_phone`, `client.direction`, `client.channel`,
  `client.provider_call_id`, `client.context_id`, `client.codec`, and `client.sample_rate` are
  removed from telephony production, resolution, UI selection, prompt suggestions, and display.
- Existing saved mappings using those keys stop resolving until changed to `telephony.*`.
- No legacy aliases or dual writes are added.
- Non-telephony `client.*` metadata remains supported.

## Failure and Recovery

- Missing optional fields are omitted by existing registry behavior.
- A provider that lacks a reliable destination does not invent `telephony.to_phone`.
- SIP URI metadata conversion failure does not block media startup.
- Existing authentication timeout and failure handling remain unchanged.

## Security and Privacy

- SIP headers are not exposed to assistant authentication.
- Raw addresses are not added to metric labels or logs.
- Provider signaling values are claims, not proof of caller ownership.

## Observability

Rename telephony metadata constants and emitted keys to `telephony.*`. Update backend conversation
query translation, conversation display, and search fields to use the new keys. Do not dual write
old values.

## Data and Migration

No schema migration. Previously persisted conversations retain old metadata keys and are
intentionally unsupported by the new telephony filters and channel display. Saved authentication
mappings and prompts are not rewritten automatically; removed telephony `client.*` sources resolve
absent until manually edited.

## Rollout

Deploy backend and UI together. Validate inbound and outbound calls for native SIP and at least one
webhook provider. Stop rollout if From and To are reversed, credential headers appear, or a
non-telephony `client.*` variable regresses.

## Rollback

Revert the backend and UI change together. New `telephony.*` mappings created during rollout will
not resolve after rollback.

## Alternatives Considered

- Keep `client.*` aliases: rejected by product direction because there must be one contract.
- Use `sip.*` for every field: rejected because From and To are available across providers.
- Use `client.sip.*`: rejected because signaling direction is not client identity.
- Change protobufs: rejected because initialization metadata already carries the values.

## Testing and Verification

- Provider tests cover explicit From/To capture for Exotel, Twilio, Telnyx, Vonage, Vobiz,
  Asterisk, and native SIP where each direction is supported.
- Base streamer tests cover explicit From/To projection and omitted values.
- Native SIP tests cover phone and URI projection while proving headers are not exposed.
- Registry and authentication dispatch tests cover `telephony.*` mapping.
- Provider tests cover existing Exotel and Twilio inbound mappings and outbound direction.
- UI tests cover the Telephony group, removed telephony Client choices, default mapping, and saved JSON.
- Conversation UI tests cover `telephony.provider` display and search fields.
- Run focused Go and UI tests, `git diff --check`, and `make agent-finalize CHANGED_FILES="..."`.

## Acceptance Criteria

- [ ] All telephony providers use the shared `telephony.*` initialization path.
- [ ] From and To retain wire direction for inbound and outbound calls.
- [ ] Native SIP authentication can map phone and URI values, but not SIP headers.
- [ ] Removed telephony `client.*` keys are neither emitted nor selectable.
- [ ] Genuine non-telephony `client.*` metadata remains resolvable.
- [ ] Existing HTTP authentication execution remains unchanged.
- [ ] Required tests and repository finalization pass.

## Open Questions

None.

## Challenge Resolution

Pending independent challenge.

## Artifact Index

- `jsons/plan.json`: implementation plan, draft

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-24 | Replace telephony `client.*` with `telephony.*` without aliases | Product | User direction |
