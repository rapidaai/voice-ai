# RFC 0018: Playback-Driven Message Completion

- Status: Accepted
- Owner: Repository maintainer
- Created: 2026-09-10
- Updated: 2026-09-10
- Reviewers: Independent design challenger and implementation reviewer

## Summary

Complete an audio assistant message only after its producers have closed, all
admitted synthesis has ended, and the playback owner has acknowledged every audio
segment. The message lifecycle emits the existing completion metric and asks the
session lifecycle to start idle timing. Keep exactly those two lifecycle owners.

Add a sequence number to audio segments and their receipts. Do not add pause IDs,
streamer interface methods, helper packages, or a third lifecycle. Implementation
of this protocol change requires approval of this document's exact SHA-256.

## Context

RFC 0017 implemented protobuf stream messages and moved lifecycle ownership. Its
callback-authority follow-up remains pending. Review reproduced these issues:

- `dispatch_handler.go:1270` marks assistant idle at TTS end while audio is queued.
- `dispatch_handler.go:1320` records receipts without starting idle timing.
- `message_turn.go:241` emits `assistant_turn=complete` at generation completion.
- `session_lifecycle.go:299` admits old same-message idle expiry after restart.

The stale idle expiry and premature TTS-end idle transition were fixed as
independent Standard changes. They did not require this gate. Retaining active state at TTS end
preserves interruption eligibility, but does not establish actual playback drain.
Receipts remain diagnostic until the completion contract below is approved.

Native media emits completion per terminal-delimited segment. Several segments can
share the same message ID. `{id, time}` cannot distinguish a retransmitted receipt
with a refreshed timestamp from another segment's receipt. Requiring unique,
immutable timestamps would itself change the protocol and misuse time as identity.

Assistant generation also has intermediate boundaries. For example,
`llm/model/model_execute.go` executes tools before emitting the original
`LLMResponseDonePacket`. Generation end and an empty output queue do not establish
that no tool continuation or already-admitted synthesis will produce more audio.

## Goals

- Keep buffered audio interruptible until actual playback completion or discard.
- Emit successful message completion exactly once at the correct boundary.
- Start session idle timing only when the message is complete and no new activity
  has superseded that eligibility.
- Reject duplicate, stale, failed, and ambiguous receipts without manufacturing
  success. Preserve text-only completion without audio receipts.

## Non-Goals

- Changing the 500 ms interruption policy, filler vocabulary, or rollout default.
- Redesigning VAD, STT, end-of-speech, provider selection, or audio codecs.
- Inferring that the remote listener heard audio from a successful network write.
- Reconstructing a word-aligned heard transcript or migrating historical metrics.

## Scope and Ownership

### Allowed Paths

- `protos/artifacts/talk-api.proto` and artifacts from `bin/generate artifact`:
  protocol implementer; generated files are never hand-edited.
- `api/assistant-api/internal/type/packet.go` and corresponding tests: packet owner.
- `api/assistant-api/internal/adapters/lifecycle/`: coordinator, message state,
  pending work, acknowledgment admission, metrics, and session handoff.
- `api/assistant-api/internal/adapters/internal/` and `adapters/router/`:
  coordinator, packet routing and I/O outcome delivery only.
- `api/assistant-api/internal/llm/{model,agentflow,agentkit,websocket}/`:
  producer worker, response-closure signaling and tests only.
- `api/assistant-api/internal/transformer/`, the existing text processor matched by
  `api/assistant-api/internal/*/output/`, and `api/assistant-api/internal/watchdog/`:
  synthesis worker, propagating
  synthesis sequence and finality through existing packet paths only.
- `api/assistant-api/internal/channel/{base,grpc,webrtc}/` and
  `channel/telephony/internal/`: transport worker, sequence propagation,
  completion receipts, and contract tests only. No codec or SIP setup changes.
- `rfcs/0018-message-playback-completion.md` and its `jsons/`: coordinator.

Workers receive disjoint file scopes before implementation. An independent reviewer
does not edit production code. Expand no path without maintainer approval.

### Out-of-Scope Paths

UI, database schemas, authentication, integration-api providers, deployment files,
STT/VAD/EOS algorithms, commits, pushes, and deployments.

## Proposed Design

### Existing Owners

- Message lifecycle owns the active message, admitted response work, synthesis
  segments, pause/continue/flush decisions, and one terminal outcome.
- Session lifecycle owns session state, idle/max-session countdowns, countdown
  validity, prompt backoff, and shutdown policy.
- Dispatcher forwards packets, performs I/O, and reports outcomes to these owners.
  It does not inspect state to reconstruct completion or interruption decisions.

### Explicit Work Closure

Add internal packet fields/events for assistant production closure and synthesis
sequence. Register work in message lifecycle before handing it to an asynchronous
producer. Intermediate LLM responses and tool-injected speech may finish a synthesis
segment, but cannot close the response. Pending tool continuations remain admitted
work until they finish or fail. A response closes only when its executor has no
remaining continuation; tool-call results do not themselves establish closure.

The model executor reports closure on a no-more-tools completion, not the preceding
tool-bearing LLM completion. The workflow executor reports closure after the whole
workflow finishes, not after a message node. Remote executors translate their
response-complete boundary only after tracked continuations finish. Empty final
responses also report closure. Providers lacking such a boundary fail closed and
cannot opt into successful audio completion until their boundary is implemented.

Assign a monotonically increasing, nonzero synthesis/playback sequence within each
message. Text chunks, synthesis requests, audio chunks, ends, errors, and watchdog
fallbacks preserve that sequence from admission. Never assign a late provider
callback the sequence that merely happens to be current. Provider callbacks must
capture their originating request or stream generation. Where a transport permits
only one active synthesis, serialize that existing boundary; do not add parallel
requests that cannot preserve origin.

The same sequence is used on every audio chunk and terminal of that segment.
Retries retain it. Empty synthesis produces no required playback acknowledgment;
text that failed to produce audio is a failed segment, not successful empty speech.

### Completion Predicate

An audio message completes only when all of these hold atomically:

1. Its producer has closed and no admitted continuation can create output.
2. Every admitted synthesis segment ended successfully.
3. Every segment that produced audio has a matching playback receipt.
4. No pause decision, interruption commit, cancellation, or failure is pending.
5. No successful terminal outcome has already been emitted for this message.

Mark the message idle and emit `assistant_turn=complete` and
`StartIdleTimeoutPacket` from this single transition. Text-only responses use
producer closure and delivery completion without playback receipts. A new
standalone injection after closure is a new message, never a reopening of an
already-completed message ID. Tool injections before closure stay in their parent
message as separately admitted segments.

```text
producer closes + all synthesis ends + matching playback receipts
                              |
                              v
                     Message lifecycle
                              |
              +---------------+----------------+
              |                                |
      assistant_turn=complete          StartIdleTimeoutPacket
                                               |
                                               v
                                      Session lifecycle

new activity -> Message lifecycle -> StopIdleTimeoutPacket
```

TTS end alone only closes its synthesis segment. It never sets assistant idle.
Generation end may persist text and generation metrics but never successful audio
turn completion. Intermediate segment receipts cannot complete the whole message.

## Contracts and Compatibility

Add these fields using presently unused tags:

| Message | Field | Tag |
| --- | --- | --- |
| `ConversationAssistantMessage` | `uint64 playback_sequence` | 5 |
| `ConversationPlaybackComplete` | `uint64 playback_sequence` | 3 |

The message `id` remains unchanged. Pause, Continue, and Flush remain scoped by
that ID with no new fields. The Streamer interface remains Context/Recv/Send using
protobuf messages. Timestamps remain timestamps, not identities or ordering keys.

A receipt acknowledges exactly `(id, playback_sequence)` once. Receipt order is
irrelevant. Unknown, zero, unissued, or flushed sequences are not successful
completion evidence. A synchronous receipt during terminal Send may be retained,
but cannot commit success until that Send succeeds. Failed retries cannot erase an
earlier successful issuance of the same terminal.

Native streamers preserve the sequence through their output queues and echo it
after successful local output drain. Plain gRPC clients echo the sequence after
their playback boundary. A transport write is not a remote-hearing guarantee.

The additions are wire-compatible, but callback authority requires upgraded
playback clients. Legacy sequence-zero receipts remain diagnostic only. Do not
silently fall back to generation-based successful completion for those clients.
Client rollout is a prerequisite for enabling this behavior on external gRPC.

## Failure and Recovery

- Missing or malformed receipts cannot complete messages or start idle countdowns.
- After producer closure and terminal delivery, bound acknowledgment waiting by
  the total unacknowledged audio duration plus a five-second grace period. Pause
  suspends this countdown; Continue resumes its remaining budget. Expiry emits a
  message failure and asks session lifecycle to close with the existing error
  disconnection reason. It does not emit successful completion or an idle prompt.
- Missing synthesis completion uses the existing TTS watchdog, now carrying the
  originating sequence. A watchdog timeout is failure, not proof of playback.
- Confirmed interruption flushes output, invalidates all old message receipts,
  cancels pending producers/timers, and admits the next turn. Resumed speech keeps
  its message and segment identity and does not duplicate completion.
- Disconnect/mode switch invalidates pending completion and timer callbacks before
  resources are released. Close/join operations stay with their existing owner.
- Failed Send or partial output is failed/interrupted, never completed. Once a
  message has a terminal outcome, late packets cannot change that outcome.

## Security and Privacy

Receipts are scoped to the authenticated stream, its current message, and issued
sequences. Unknown sequences cannot allocate lifecycle entries. Retire sequence
state with the message. Do not log transcript/audio contents to diagnose receipts.

## Observability

Preserve metric names and existing persistence routes. Emit successful
`assistant_turn=complete` once from message lifecycle. Preserve generation metrics
as generation metrics. Record rejected receipts and timeout/failure reasons with
message ID and sequence; never use them to claim playback success.

## Data and Migration

No database schema change or historical backfill. Existing completion metric
timing changes for newly handled messages only. Regenerate protobuf artifacts with
`bin/generate artifact` and verify native/gRPC producer-consumer parity.

## Rollout

1. Implement and verify producer/synthesis identity, transport echo, and lifecycle
   transitions as one reviewable change after exact-digest confirmation.
2. Upgrade external playback clients before enabling callback authority for them.
3. Run real native and gRPC calls covering interruption, tools, delayed receipts,
   and shutdown. Stop rollout on premature completion, stuck output, missing idle
   prompts, or unexpected acknowledgment-timeout disconnects.
4. Keep the existing interruption rollout default unchanged; callback completion
   does not automatically enable the pause-confirm interruption path.

## Rollback

Restore the pre-authority adapter behavior while retaining additive protobuf fields
and sequence-aware clients. Do not remove or reuse field tags. Keep the independent
stale-idle-expiry fix. No persisted data rollback is required.

## Alternatives Considered

- Deduplicate by time: no uniqueness/retransmission guarantee; rejected.
- Count receipts: duplicate old receipts can complete future segments; rejected.
- Complete at TTS end: leaves buffered audio and inaccurate metrics; rejected.
- Treat the first same-ID receipt as final: tool/injected segments break it; rejected.
- Emit one final terminal only: still needs producer/synthesis closure and changes
  segment-drain semantics. Explicit sequence preserves existing segment boundaries.
- Add pause IDs or a new lifecycle owner: unnecessary; rejected.

## Testing and Verification

Required tests cover successful text/audio completion; same-ID multi-segment
output; delayed, duplicate, reordered, zero, and unknown receipts; callbacks during
successful/failed Send; final generation before final audio; tool continuations;
empty/no-audio failure; repeated terminals; pause/continue/flush; TTS/receipt
timeouts; stale idle expiry; mode switch and shutdown. Each changed backend package
must include corresponding tests. Do not claim remote playback from local drain.

```sh
bin/generate artifact
go test -race -count=1 -timeout=180s ./api/assistant-api/internal/adapters/... ./api/assistant-api/internal/watchdog/... ./api/assistant-api/internal/channel/...
go test -count=1 -timeout=180s ./api/assistant-api/internal/llm/... ./api/assistant-api/internal/transformer/...
go test -run '^$' ./api/assistant-api/...
git diff --check
```

Run `just agent-finalize "<exact comma-separated changed paths>"` after the final
file inventory is known. Independent review is mandatory before merge. Real-call
verification and unavailable provider credentials are recorded separately from
unit-test evidence.

## Acceptance Criteria

- [ ] The four reproduced review gaps are covered by regression tests.
- [ ] Generation/TTS end alone does not successfully complete an audio message.
- [ ] Same-ID segments, retries, and delayed receipts cannot complete later output.
- [ ] Message lifecycle alone decides successful/interrupted/failed completion.
- [ ] Session idle begins after accepted message completion and rejects stale work.
- [ ] Text-only, tool, timeout, pause, interruption, and shutdown paths are explicit.
- [ ] No pause IDs, third lifecycle, generic helpers, or streamer methods are added.
- [ ] Required tests and independent review pass; legacy-client limitations are explicit.

## Open Questions

Maintainer confirmation of the independently challenged document's exact digest
is required before implementation of the protocol-dependent changes. The metadata
status alone is not implementation authorization; the authoritative challenge and
confirmation receipts are stored in the JSON artifacts.

## Challenge Resolution

Initial read-only challenge rejected timestamp-based deduplication and receipt
counting. This proposal instead uses explicit segment identity and separate
producer/synthesis closure. Final-byte challenge and confirmation results are
stored separately. The earlier reviewer launches were blocked by an update prompt
and account credit. Review resumes against the final bytes in this document;
no approval or confirmation is implied until the separate receipts establish it.
Limit correction cycles to two.

## Artifact Index

- `jsons/reservation.json`: reserved path and Governed classification.
- `jsons/plan.json`: implementation scope, criteria, risks, and verification.
- `jsons/challenge.json`: independent final-byte decision and digest.
- `jsons/confirmation.json`: exact-digest approval, pending maintainer response.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-09-10 | Fix countdown validity independently; require a segment contract before callback authority | Coordinator | Review probes and design challenge |
