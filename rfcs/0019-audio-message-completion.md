# RFC 0019: One Audio Completion Marker Per Message

- Status: Accepted
- Owner: Repository maintainer
- Created: 2026-09-10
- Updated: 2026-09-11
- Reviewers: Independent design challenger and implementation reviewer

## Summary

Amend RFC 0018 to use the existing `ConversationAssistantMessage.completed`
field on explicit audio messages. Send one final audio marker and accept one
`ConversationPlaybackComplete` per unique message ID. Do not track playback
sequences or collect segment receipts. Remove the sequence fields from protobuf
and generated APIs, reserving their names and numbers. The accepted RFC 0018 bytes
remain intact.

This document's metadata identifies the final candidate for independent challenge.
Implementation still requires an approved exact-digest confirmation receipt.

## Context

The maintainer requested reuse of the existing completion flag for audio only.
The adapter already emits an explicit empty audio payload with `completed: true`
in `adapters/internal/dispatch_handler.go:1270`. Text completion is separately
emitted by `HandleTextToSpeechDone`. Some streamer paths also accept a missing
payload as an audio terminal, and native queues permit multiple terminal-delimited
segments under the same ID.

RFC 0018 chose segment identity to retain that behavior. This amendment instead
changes the boundary to one final audio marker per message. The existing
`playback_sequence` fields are removed at the maintainer's September 11 request.
Reserve their tags and names so unrelated data cannot reuse them. No new protobuf
field or stream message is needed.

Producer finality still matters. `llm/model/model_execute.go:onCompletion` can
execute tools before emitting `LLMResponseDonePacket`, and workflow nodes can
emit intermediate done packets. A provider flush or a model invocation ending is
not automatically the end of the whole assistant message.

## Goals

- Complete successful audio playback once per message, without segment receipts.
- Preserve audio FIFO order and interruption control until the final audio drains.
- Keep completion metrics in message lifecycle and idle timing in session lifecycle.
- Reuse existing protobuf messages and leave dispatcher responsible for routing/I/O.

## Non-Goals

New stream methods, pause IDs, public messages, sequence tracking, provider choice,
codecs, call setup, filler policy, word-aligned heard text, or proof of remote hearing.

## Scope and Ownership

### Allowed Paths

- `api/assistant-api/internal/adapters/lifecycle/`: coordinator, finality, receipt
  admission, interruption outcome, completion metric, and idle handoff.
- `api/assistant-api/internal/adapters/internal/` and `adapters/router/`:
  coordinator, packet routing and I/O outcome reporting only.
- `api/assistant-api/internal/type/packet.go` and its tests: coordinator,
  remove unused playback-sequence plumbing without changing unrelated packets.
- `api/assistant-api/internal/llm/{model,agentflow,agentkit,websocket}/`:
  producer worker, correct final message boundary after admitted continuations.
- `api/assistant-api/internal/transformer/`, the existing text processor under
  `api/assistant-api/internal/*/output/`, and `internal/watchdog/`:
  synthesis worker, final audio ordering and failure handling only, not STT.
- `api/assistant-api/internal/channel/{base,grpc,webrtc}/` and
  `channel/telephony/internal/`: transport worker, audio-only marker admission,
  one message-level receipt, and removal of sequence-dependent queue state.
- `protos/artifacts/talk-api.proto`, `protos/talk-api.pb.go`, and
  `api/document-api/app/bridges/artifacts/protos/talk_api_pb2.{py,pyi}`:
  protocol worker, remove sequence fields, reserve their names/numbers, and
  regenerate artifacts with `bash bin/artifacts-generate.sh`. Do not hand-edit
  generated files or include unrelated generator churn.
- `rfcs/0019-audio-message-completion.md` and its `jsons/`: coordinator.

Each changed Go package includes corresponding tests. Implementation workers get
disjoint file inventories before starting. Reviewers do not edit implementation.

### Out-of-Scope Paths

Accepted RFC 0018 and its signed plan, unrelated protobuf/generated artifacts, UI,
database, authentication, deployment, integration-api, VAD, STT, and EOS algorithms.
Do not commit, push, or deploy without a separate request.

The separately authorized Standard fixes make expiry resume without STT
confirmation and make repeated telephony flush release pause. Preserve those
changes and their tests. They do not authorize this protocol implementation or
change the interruption rollout default.

## Proposed Design

```text
All admitted response generation finishes
    |
    v
TTS finishes the remaining text and enqueues its last audio
    |
    v
Audio FIFO: [audio M1] [audio M1] [audio M1, completed=true]
    |
    v
Streamer drains preceding audio and its own codec/resampler tail
    |
    v
ConversationPlaybackComplete(id=M1)
    |
    v
Message lifecycle completes once -> session lifecycle may start idle timing
```

### Producer and Synthesis Boundary

The outer response executor owns generation closure. Its final done packet is
emitted only after all admitted tool continuations, workflow nodes, and speech
injections have finished producing text for that message. Intermediate model/node
completion and sentence flushing must not emit a final message done signal.

TTS owns its pending provider work. `TextToSpeechDonePacket` closes message text;
`TextToSpeechEndPacket` follows the final audio callback after that text and all
pending synthesis have drained. Provider callbacks retain their originating
message ID. Serial provider requests must finish before the next request can reuse
their connection state. No elapsed-time estimate may fabricate successful end.

Message lifecycle tracks generation closure and final synthesis outcome, not a
collection of playback segments. It authorizes exactly one final audio marker
after both have succeeded. The dispatcher sends the existing protobuf and reports
its result; it does not own finality flags or decide turn completion.

### Audio-Only Marker

Playback terminal admission requires the protobuf audio oneof to be selected and
`completed: true`. The final chunk may carry audio or explicit empty audio. A text
payload or missing oneof never closes audio playback. Text completion remains
available for transcript delivery and text-only responses.

The marker travels in the same FIFO as its audio. It must not use a priority path
that overtakes audio. Streamers finish partial frames and codec/resampler tails
before emitting their one receipt for the message. A queue gap is not completion.

After a final marker is admitted, further audio for that ID is rejected. Duplicate
markers do not emit extra receipts. Later standalone speech uses a new message ID;
closed IDs are never reopened within a session. Queue state retains closed/flushed
identity long enough to reject stale output without relabeling it as current.

### Lifecycle Admission and Controls

Pause holds both audio and its marker. Continue resumes the same FIFO. Flush
discards remaining audio and the marker, invalidates any in-flight receipt, and
cannot produce successful completion. Failed writes and dropped audio also fail
playback rather than claiming success.

A receipt is admissible only for the active audio message with a successfully
issued final marker, closed generation, ended synthesis, no pending interruption
decision, and no previous terminal outcome. A receipt arriving during marker Send
is held by message lifecycle until that Send succeeds; failed Send discards it.
Paused receipt admission is deferred until continue or discarded on flush.

The successful transition emits the assistant completion metric once and asks
session lifecycle to start idle timing. Generation end alone does neither for
audio messages. Text-only completion uses generation and text delivery, not an
audio marker. Mode switch, cancellation, shutdown, or timeout cannot become success.

## Contracts and Compatibility

- Keep existing `id`, audio oneof, `completed`, and playback-complete messages.
- Remove `playback_sequence` from queue identity, receipt admission, packets,
  diagnostics, protobuf definitions, and generated APIs. In
  `ConversationAssistantMessage`, reserve field number `5` and name
  `playback_sequence`; in `ConversationPlaybackComplete`, reserve number `3` and
  the same name. Preserve every other field number and type.
- This removes generated source accessors and is not source-compatible for clients
  using them. Such clients must rebuild and adopt the one-final-marker contract.
  Old binary sequence fields may decode as unknown fields; that does not prove
  semantic compatibility. ProtoJSON clients still sending the removed property
  may be rejected by strict parsers. Do not enable authoritative callbacks for
  segment-based or otherwise unverified clients.
- Native, gRPC, and WebRTC consumers must agree on the audio-only final marker.
  Clients completing on text, empty-queue gaps, or payload-less terminals are not
  compatible with callback-authoritative completion until updated.
- Do not infer compatibility from an unkeyed event or a timestamp. Receipt matching
  uses the unique message ID and the final-marker issuance state.
- Preserve RFC 0018's bounded, pause-aware missing-receipt failure handling. Timeout
  is failure, never successful completion. Keep callback authority disabled for
  unverified clients; do not silently fall back to generation-end success.

## Failure Modes and Observability

Log stale/duplicate receipts, audio after finality, marker send failure, provider
failure, and missing playback receipts with message ID and outcome. Native drain
only describes the local observable output boundary. Empty successful output needs
no playback receipt; nonempty text that fails to produce audio is a synthesis failure.

## Data and Migration

No database or historical metric migration. Update the protobuf source and
regenerate its Go and document-api Python artifacts together with runtime removal.
Verify reserved descriptors, unchanged remaining tags, binary unknown-field
decoding, and the intended ProtoJSON rejection of the removed property. Do not
activate the new lifecycle boundary until producer, synthesis, transport, and
client paths are verified together. Preserve all unrelated dirty worktree changes.

## Rollout

After exact-digest approval, implement producer/synthesis finality, audio-only
streamer markers, and lifecycle receipt admission as one verified contract. Test
native and external client paths, including tool continuations and interruption.
Keep unverified clients diagnostic-only. Do not change interruption rollout defaults.

## Rollback

Disable callback authority and return receipts to diagnostics if rollout fails.
Do not restore sequence-based runtime code against the new generated API. A full
rollback must restore the matching pre-removal protobuf source, generated APIs,
and runtime together, without reassigning retired tags to different data. Preserve
independent queue/shutdown, control, and stale-idle fixes. No stored-data rollback
is required.

## Alternatives Considered

- Per-segment sequence receipts: superseded by the maintainer's one-marker choice.
- First TTS end or LLM end: unsafe with pending synthesis or tool continuations.
- Text `completed`, empty queues, or duration estimates: not audio finality.
- New end message: unnecessary because the existing audio oneof and flag suffice.

## Testing and Verification

Cover text and payload-less completed messages not closing audio; explicit empty
and nonempty final audio; tail drain; one receipt per ID; duplicate/stale markers;
late audio rejection; multiple synthesis requests and tool continuations; callback
during successful/failed Send; pause/continue/flush; write failure; mode switch;
empty output; synthesis/receipt timeout; exactly-once metrics and idle startup.
Verify the two sequence field names/numbers are reserved, generated accessors and
runtime uses are absent, old binary unknown fields remain decodable, and obsolete
ProtoJSON sequence properties cannot silently authorize playback completion.

```sh
go test -race -count=1 -timeout=180s ./api/assistant-api/internal/adapters/... ./api/assistant-api/internal/watchdog/... ./api/assistant-api/internal/channel/...
go test -count=1 -timeout=180s ./api/assistant-api/internal/llm/... ./api/assistant-api/internal/transformer/...
go test -p 1 -run '^$' ./api/assistant-api/...
go test -count=1 ./api/assistant-api/internal/type
python3 -m compileall -q api/document-api/app/bridges/artifacts/protos
git diff --check
```

Run focused tests in every changed package and `just agent-finalize` with the exact
changed-file inventory. Record real-call/provider-credential limitations separately.

## Acceptance Criteria

- One audio-only final marker and one message-ID receipt; no sequence tracking.
- Sequence fields and generated accessors are removed; retired names/tags remain reserved.
- All admitted generation and synthesis finish before the final marker.
- Failed, interrupted, duplicate, stale, and ambiguous output cannot complete a message.
- Message/session lifecycle own metrics/idle transitions; dispatcher only routes.
- Existing text-only behavior, queue policies, and unrelated changes are preserved.
- Required tests and independent review pass before merge.

## Open Questions

Only exact-digest confirmation of this amended contract remains before implementation.

## Decision Log

| Date | Decision | Owner |
| --- | --- | --- |
| 2026-09-10 | Reuse audio `completed` and message ID instead of segment receipts | Repository maintainer |
| 2026-09-11 | Remove sequence fields and generated APIs, reserving names and numbers; supersede the prior pending digest | Repository maintainer |
