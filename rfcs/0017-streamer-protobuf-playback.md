# RFC 0017: Protobuf Stream Controls and Playback Completion

- Status: Draft
- Owner: Assistant runtime coordinator
- Created: 2026-09-09
- Updated: 2026-09-10
- Reviewers: Independent runtime challenger and repository maintainer

## Summary

Use `proto.Message` directly in the three-method `Streamer` contract, without a
custom `Stream` interface or alias. Define generated pause, continue, flush, and
playback-complete messages in `protos/artifacts/talk-api.proto`.

Streamers execute output commands and report completion to the adapter through
their existing input path. The adapter continues to own interruption decisions.
The maintainer requested implementation while retaining this document as a working
design record. No exact-digest gate approval is claimed by this draft.

## Context

The previous implementation uses handwritten `PauseOutput`, `ContinueOutput`, and
`FlushOutput` structs with empty `ProtoMessage()` methods. They cannot travel as
generated `AssistantTalkResponse` variants. The current TTS end notification also
has no audio oneof, so telephony streamers can miss its terminal boundary.

`MediaSession` already owns telephony output. WebRTC owns peer-specific buffering
and pacing. A plain gRPC client owns its own playback. TTS generation end is not
proof that any of these playback queues have drained.

This change supersedes RFC 0016 only for control representation and completion
reporting, not for interruption timing or classification.

## Goals

- Use `proto.Message` directly throughout the streamer contract and its consumers.
- Send generated playback commands over the existing AssistantTalk response stream.
- Deliver response-scoped playback completion back to the adapter.
- Preserve pause, continue, flush, stale-output rejection, and transport ownership.

## Non-Goals

- A universal concrete streamer, new helper packages, or extra Streamer methods.
- Pause IDs, new response IDs, or moving interruption timers into streamers.
- AgentKit or WebRTC schema changes and unrelated SDK/client changes.
- A new capability negotiation system or replacing legacy TTS lifecycle fallback.
- Proof that a remote listener heard the audio.

## Scope and Ownership

### Allowed Paths

- Coordinator: `protos/artifacts/talk-api.proto`, artifacts produced by
  `bin/artifacts-generate.sh`, shared `internal/type/streamer.go`, SIP contract
  relocation, `internal/type/output_control.go` removal, and channel/base migration.
- Adapter worker: `api/assistant-api/internal/adapters/` and
  `api/assistant-api/internal/type/packet.go`, with corresponding tests.
- Telephony worker: `api/assistant-api/internal/channel/telephony/internal/`,
  existing channel/output types only as needed, and SIP pipeline test signatures.
- gRPC worker: `api/assistant-api/internal/channel/grpc/` and its tests.
- WebRTC worker: `api/assistant-api/internal/channel/webrtc/` and its tests.
- Coordinator: this document and `rfcs/0017-streamer-protobuf-playback/jsons/`.

Workers have disjoint write scopes. Generated files are never edited manually.
Only generator-produced changes required by the authoritative schema are retained.

### Out-of-Scope Paths

STT, VAD, EOS, LLM integrations, SIP signaling, database migrations, AgentKit, and
SDK implementations. Existing source files outside the scopes above are unchanged.

## Proposed Design

### Streamer Contract

```go
type Streamer interface {
    Context() context.Context
    Recv() (proto.Message, error)
    Send(proto.Message) error
}
```

SIP-specific runtime interfaces retain their capabilities in a separate contract
file. `streamer.go` contains only the shared contract, imports, and documentation.

### Protobuf Messages

Definitions belong in `talk-api.proto`, not AgentKit:

```proto
message ConversationPlaybackPause {
  string id = 1;
}
message ConversationPlaybackContinue {
  string id = 1;
}
message ConversationPlaybackFlush {
  string id = 1;
}

message ConversationPlaybackComplete {
  string id = 1;
  google.protobuf.Timestamp time = 2;
}
```

| Envelope | Field | Tag |
| --- | --- | --- |
| `AssistantTalkResponse.data` | `playbackPause` | 21 |
| `AssistantTalkResponse.data` | `playbackContinue` | 22 |
| `AssistantTalkResponse.data` | `playbackFlush` | 23 |
| `AssistantTalkRequest.request` | `playbackComplete` | 10 |

Existing tags are unchanged. Each playback message's ID is the existing assistant
response context ID, not a separate pause ID. Pause and continue carry the interrupted
response ID; flush carries the previous response ID after context rotation. Controls
still apply to active output on the ordered stream, including legacy empty-ID controls.

```text
Adapter -- Pause / Continue / Flush --> Streamer --> gRPC peer, when applicable
Adapter -- audio and terminal -------> Streamer --> paced transport output
Adapter <-- PlaybackComplete -------- Streamer <-- local drain or gRPC client report
```

### Completion Ownership

The terminal audio marker closes a response segment. A completed text message is
not an audio terminator. TTS end sends an explicit empty audio oneof with
`completed=true`; existing payload-less terminal messages remain supported where
required for compatibility.

Native completion means the existing output owner has accepted the terminal,
drained conversion tails and padding, and successfully written its response frames.
It is a local transport boundary, not carrier or device playout confirmation.
An empty queue during generation is not completion. Neither pause nor ambient,
ringback, pre-answer buffering, transfer discard, flush, or write failure is success.

The response identity must remain attached to its completion boundary. Do not emit
the newest ID when older frames finish. Emit outside output locks. Keep response
state with the existing media owner and keep the pacer transport-independent.
Multiple terminal-delimited TTS segments may share a response ID. Completing one
segment must not reject later audio for that ID. Only flushed output stays blocked.
Provider flow control is not a drain: buffered frames must remain pending until
output resumes, even when the provider cannot currently supply a frame.

Plain gRPC does not infer playback from `server.Send`. Its client sends
`ConversationPlaybackComplete` in `AssistantTalkRequest`; `Recv` returns that
generated message to the adapter. Explicit controls are also sent as response oneofs
while preserving existing local queue control and send-error behavior.

### Adapter Delivery

The existing Talk loop routes completion into its typed packet/dispatch architecture.
Reject missing, stale, duplicate, and pre-terminal completion IDs. Lifecycle owns
the terminal-issued flag and response dedupe, both reset on context rotation. Set
eligibility before terminal send because a receipt can arrive before send returns.
On send failure, restore prior eligibility so a failed retry cannot invalidate an
earlier terminal. This is a diagnostic issuance boundary, not delivery proof; an
already-recorded observation is not retracted on send failure.
Record completion separately from generation end. This patch preserves current TTS
lifecycle fallback instead of making legacy clients wait for an unsupported receipt.

Completion is control traffic: the BaseStreamer input path must use cancellable
delivery rather than the replace-oldest audio queue. The adapter route must not
silently replace completion either. Do not emit while holding output locks or block
the same dispatch worker waiting for itself.

## Contracts and Compatibility

Schema additions are additive. Older clients keep their existing wire behavior and
lifecycle fallback; new clients may consume explicit playback controls and report
completion. A client handling explicit controls must not also interpret an
interruption observation as a second playback command.

There is no claim that older clients implement remote pause/continue. The new
messages make this behavior representable over gRPC; client adoption is separate.
The server does not require acknowledgements to preserve current lifecycle behavior.
The Go source migration intentionally has no `Stream` alias compatibility layer.

## Failure and Recovery

Flush discards pending audio and terminal boundaries without successful completion.
Shutdown, conversion failure, and output-write failure cannot report played output.
Nil-transport no-op paths do not establish successful delivery. Completion must
survive input saturation or terminate on stream cancellation, without deadlock.
Stale and duplicate completion events cannot mutate a later conversation turn.

Known resampler-tail and startup-order defects are fixed only to the extent required
by the verified completion implementation; unrelated cleanup is excluded.

## Security and Privacy

Use existing stream authentication and conversation scope. Validate completion IDs
against the current connection and adapter response. Client timestamps are diagnostic
only. Messages contain no audio, transcript, credentials, or new customer identifiers.

## Observability

Playback completion and generation completion are separate response-scoped events.
Preserve existing transport write-error metrics. Do not label a local drain event as
proof of remote audibility or log audio contents.
The delivery guarantee ends at adapter dispatch. Recording retains the existing
best-effort observability behavior; no durable event store or retry protocol is added.

## Data and Migration

No persistent-data migration. The schema lives in the `protos/artifacts` submodule;
publish its source commit and matching generated artifacts together before shipping.
The generator is `bash bin/artifacts-generate.sh`, as requested by the maintainer.

## Rollout

Implement schema and shared contract first, then transport and adapter consumers.
Keep legacy lifecycle fallback. Validate generated oneof round trips, real buffered
audio paths, transport ordering, and completion delivery before enabling consumers.
No automatic SDK deployment or remote push is part of this implementation request.

## Rollback

Revert the runtime integration and source-signature migration together if required.
Do not reuse added protobuf tags. Preserve unrelated parent SIP fixes and keep schema
source and generated artifacts synchronized.

## Alternatives Considered

- `Stream` alias or fake protobuf marker methods: rejected by the maintainer.
- New Streamer control/callback methods: rejected; keep the three-method contract.
- Generation end or RPC send success as playback proof: incorrect boundaries.
- New client negotiation and lifecycle wait policy: deferred to keep this fix scoped.

## Testing and Verification

Run package tests and race checks for changed contracts, adapter dispatch, BaseStreamer,
gRPC, WebRTC, telephony media/providers, and SIP pipeline test consumers. Include
generated marshal/unmarshal round trips and wire envelope mappings. Run
`just agent-finalize` with the exact changed-file inventory, including new tests.

Required cases: terminal audio, streaming gap, delayed resampler tails, two response
IDs, pause/continue, flush before completion, stale/duplicate completion, output
failure, cancellation, pre-answer replay, transfer discard, and queue saturation.
Generation and native dependency limitations must be reported, not hidden.

### Verification Evidence

- Generated Go and Python artifacts with `bash bin/artifacts-generate.sh`.
- Compiled all assistant-api packages with `go test -run '^$'`.
- Passed the combined race suite for shared types, adapter dispatch/lifecycle,
  BaseStreamer, gRPC, WebRTC, telephony providers, and the SIP pipeline.
- Passed 20 repeated race runs for shared types, BaseStreamer, and gRPC.
- Added SIP drain-state benchmarks for empty, buffered, and transfer output; all
  three cases reported zero allocations across three runs.
- Independent review verified shutdown cancellation, ID-less output, same-ID
  segments, flow-controlled drain, and terminal-send retry handling.
- Final `just agent-finalize` passed with no missing-test errors and zero failing
  packages; the inventory included generated artifacts and the source proto path.

Initial parallel native-converter fixtures reproduced a `soxr_create` crash.
Serialized native fixtures passed repeated cold-process and race runs while explicit
streamer concurrency tests remained enabled. Native resampler internals are unchanged;
concurrent cold initialization remains a separate dependency risk. No live carrier
or external SDK smoke test was performed. Python syntax validation passed, but full
application import validation lacked the local `fastapi` dependency.

## Acceptance Criteria

- [x] No custom Stream type/alias or handwritten output-control marker remains.
- [x] Four real messages exist in talk-api.proto with generated artifacts.
- [x] gRPC transmits controls and receives client completion through its oneofs.
- [x] Native output reports response-scoped completion without false success on discard.
- [x] Adapter receives completion through the existing packet path without silent loss.
- [x] Legacy lifecycle fallback remains; no new acknowledgement wait is introduced.
- [x] Focused tests, race checks, and independent implementation review pass.

## Open Questions

The current adapter records the first eligible completion for a response ID. This
does not assert that no later TTS segment can use that ID. A future completion-driven
lifecycle requires response-wide boundaries, capability handling, and missing-ack
recovery; it is not enabled by this fix.

## Lifecycle Ownership Follow-Up

The maintainer requested callback-driven idle timing and assistant message completion,
then clarified that there must be exactly two lifecycle owners. The ownership
refactor is implemented; the callback-authority change remains pending. Earlier
streamer and completion-delivery changes remain intact.

| Owner | Responsibility |
| --- | --- |
| Session lifecycle | Connection transitions, idle and maximum-session countdowns, expiry acceptance, and shutdown. |
| Message lifecycle | Message identity, user and assistant turns, interruption decisions and timer, held input, playback acceptance, and completion metric decisions. |
| Dispatcher | Route packets, execute provider and streamer I/O, persist observations, and return I/O outcomes to the owning lifecycle. |

Moving fields alone is insufficient. Message lifecycle must atomically validate an
event, update its state, and decide which existing packets and protobuf controls to
emit. The dispatcher must not reconstruct that decision using state getters or
mutable flags. No third turn, interruption, or playback lifecycle is introduced.

```text
Streamer playback-completed callback
    |
    v
Dispatcher: deliver PlaybackCompletedPacket
    |
    v
Message lifecycle: accept completion for active message
    |
    +--> assistant_turn=complete metric packet --> existing persistence route
    |
    +--> StartIdleTimeoutPacket --> Session lifecycle: start countdown

New user or assistant activity
    |
    v
Message lifecycle --> StopIdleTimeoutPacket --> Session lifecycle: stop countdown
```

Interruption flags, held input, decision timers, transcript admission, context changes,
unclear-input prompts, and message completion packet decisions now reside in message
lifecycle. Session lifecycle owns idle/max-session watchdogs, prompt backoff, timeout
admission, and tool-action cancellation. Dispatch handlers execute the returned
packets and controls without reading message state or allocating replacement IDs.

Playback control I/O is serialized by message lifecycle separately from its state
lock. Speech can be admitted while Pause is in flight, but its confirmed Flush cannot
overtake Pause. A superseded pause is rejected before transport execution. Context
replacement invalidates held input and old interruption callbacks.

Ownership verification passed the adapter/lifecycle/router/watchdog race suites and
the full assistant-api compile check. Independent read-only review found no critical
or major issues. The staged interruption rollout remains disabled by default; its
unchanged default is now defined by message lifecycle.

The next behavior change moves audio completion from generation-end to accepted
playback completion, with text-only completion remaining independent of playback.
Current generation/TTS completion timing is deliberately unchanged by this refactor.
The same-message multi-segment ambiguity above must be resolved before treating its
first receipt as proof that the entire assistant message has completed.

Detailed scope and validation commands are recorded in
`0017-streamer-protobuf-playback/jsons/amendment-01-lifecycle-ownership.json`.

## Challenge Resolution

The initial draft review requested precise terminal correlation, reliable delivery,
and bounded missing-ack recovery. The implementation is narrowed to completion
delivery with legacy lifecycle fallback, eliminating the new acknowledgement wait.
Correlation and queue guarantees remain mandatory implementation review checks.

## Artifact Index

- `jsons/plan.json`: implementation scope and the maintainer's latest direction.
- `jsons/challenge.json`: initial draft findings, retained as design review history.
- `jsons/implementation-review.json`: implementation review and verification evidence.
- `jsons/amendment-01-lifecycle-ownership.json`: planned ownership refactor and callback behavior.
- `jsons/amendment-01-lifecycle-ownership-review.json`: ownership review and verification evidence.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-09-09 | Use talk-api.proto, proto.Message directly, and reverse completion | Maintainer | User request |
| 2026-09-09 | Keep RFC and start implementation using repository artifact generator | Maintainer | Latest user instruction |
| 2026-09-09 | Preserve lifecycle fallback; implement message transport and completion observation | Coordinator | Scoped implementation plan |
| 2026-09-10 | Add response ID at field 1 to pause, continue, and flush | Maintainer | User request |
| 2026-09-10 | Keep session and message as the only lifecycle owners; remove turn and interruption decisions from dispatcher | Maintainer | User direction; implementation pending |
| 2026-09-10 | Implement the two-owner refactor first while retaining existing completion timing | Maintainer and coordinator | User approval; ownership implementation and regression checks |
