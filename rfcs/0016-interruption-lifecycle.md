# RFC 0016: Interruption Lifecycle and Output Ownership

- Status: Accepted
- Owner: Assistant runtime coordinator
- Created: 2026-09-06
- Updated: 2026-09-07
- Reviewers: Independent runtime challenger and repository maintainer

## Summary

Separate speech detection from interruption commitment. While assistant audio is active,
VAD start opens one interruption candidate and the adapter asks the streamer output path
to pause. The adapter then has 500 milliseconds to choose one result:

- meaningful speech or speech still active at the deadline emits `TurnChangePacket`;
- speech that ended without meaningful text emits an internal continue control.

`TurnChangePacket` is the only committed interruption signal. Its handler flushes local
output, rotates the conversation context once, interrupts old TTS and LLM work, resets old
EOS work, and replays held meaningful STT into the new turn.

The existing `internal_type.Streamer` interface remains unchanged. Pause, continue, and
flush are internal stream values consumed by each concrete streamer's existing `Send` and
output loop. Streamers execute output controls but never classify speech, own timers, or
change conversation context.

## Context

Today `InterruptionDetectedPacket` can rotate context and interrupt TTS and LLM directly
from VAD start. This treats any detected activity, including a cough or short filler, as a
committed user turn. STT can also rotate context independently when text arrives. These
paths allow two different signals to commit the same turn transition.

Output ownership already belongs to concrete streamers. Their output loops and media
sessions buffer audio, pace frames, clear local queues, and send provider-specific clear
signals. The generic streamer interface only carries conversation stream values and must
not become a media-control API.

## Goals

- Pause assistant output immediately when VAD starts during active assistant output.
- Decide within 500 milliseconds whether to continue or commit a turn change.
- Keep VAD detection, interruption decision, and committed turn transition separate.
- Make `TurnChangePacket` the single authoritative interruption commit.
- Hold matching STT and EOS evidence while a candidate is pending.
- Keep the `internal_type.Streamer` interface unchanged.
- Give every concrete streamer the same internal pause, continue, and flush behavior.

## Non-Goals

- Acoustic intent classification.
- Recall of audio already handed to a remote provider, network, or device.
- Public protobuf or SDK changes.
- Provider-side LLM cancellation beyond current executor capabilities.
- Played-text reconstruction.
- Configurable decision timing in the first release.

## Scope and Ownership

### Allowed Paths

- `api/assistant-api/internal/type/packet.go`: adapter packet definitions.
- `api/assistant-api/internal/type/output_control.go`: internal stream control values.
- `api/assistant-api/internal/adapters/internal/requestor.go`: candidate state ownership.
- `api/assistant-api/internal/adapters/internal/stream.go`: internal output-control delivery with returned errors.
- `api/assistant-api/internal/adapters/internal/stream_test.go`: output-control delivery tests.
- `api/assistant-api/internal/adapters/internal/dispatch_handler.go`: candidate decisions and committed turn handling.
- `api/assistant-api/internal/adapters/internal/dispatch_handler_interruption_test.go`: adapter behavior tests.
- `api/assistant-api/internal/adapters/router/dispatch.go`: dispatch of an internal candidate deadline packet if required.
- `api/assistant-api/internal/adapters/router/router.go`: route declaration for that packet if required.
- `api/assistant-api/internal/adapters/router/*_test.go`: routing tests if routing changes.
- `api/assistant-api/internal/adapters/internal/interruption.go`: serialized candidate and transcript gate.
- `api/assistant-api/internal/adapters/internal/interruption_test.go`: serialized decision tests.
- `api/assistant-api/internal/end_of_speech/**`: synchronous local invalidation before replay.
- `api/assistant-api/internal/transformer/**`: local context fencing before replay where required.
- `api/assistant-api/internal/channel/output/`: shared buffered-output behavior and tests.
- `api/assistant-api/internal/channel/webrtc/`: WebRTC output-control consumption and tests.
- `api/assistant-api/internal/channel/grpc/`: gRPC output-control consumption and tests.
- `api/assistant-api/internal/channel/telephony/internal/base/`: shared telephony output-control consumption and tests.
- `api/assistant-api/internal/channel/telephony/internal/media/`: telephony media pause, continue, flush, and tests.
- `api/assistant-api/internal/channel/telephony/internal/twilio/`: provider clear behavior and tests.
- `api/assistant-api/internal/channel/telephony/internal/telnyx/`: provider clear behavior and tests.
- `api/assistant-api/internal/channel/telephony/internal/asterisk/`: output-control behavior and tests.
- `api/assistant-api/internal/channel/telephony/internal/vobiz/`: provider clear behavior and tests.
- `api/assistant-api/internal/channel/telephony/internal/vonage/`: provider clear behavior and tests.
- `api/assistant-api/internal/channel/telephony/internal/exotel/`: provider clear behavior and tests.
- `api/assistant-api/internal/channel/telephony/internal/sip/`: SIP output-control behavior and tests.
- `api/assistant-api/internal/adapters/lifecycle/`: conditional context transition and tests.
- `api/assistant-api/internal/llm/model/`: stale model-result rejection and tests.

### Out-of-Scope Paths

- `api/integration-api/**`
- `protos/**`
- `ui/**`
- provider VAD internals
- provider-specific recognition changes unrelated to existing transcript timing
- deployment, migrations, and infrastructure

## Proposed Design

### Internal Output Controls

Add three internal stream values:

```go
type PauseOutput struct{}
type ContinueOutput struct{}
type FlushOutput struct{}
```

They satisfy `internal_type.Stream` and are passed through a private
`sendOutputControl` path that calls the existing streamer `Send` owner and returns its
error. Ordinary public notifications continue through `Notify` with unchanged behavior.
The controls are not protobuf messages, are never sent to clients, and do not change the
`Streamer` interface.

Each concrete streamer handles the values before its public-message switch:

- `PauseOutput`: stop consuming locally buffered assistant audio and retain it in order.
- `ContinueOutput`: resume consumption of retained assistant audio.
- `FlushOutput`: discard retained assistant audio and invoke any existing provider clear.

The operations are synchronous, local, non-blocking, and idempotent. They update output
state without waiting for a network or device write. If no audio is active, they succeed
as no-ops. A streamer never starts a decision timer and never emits `TurnChangePacket`.

Each streamer uses its existing output lock, or one new output-state lock where none
exists, to serialize audio admission with pause and flush. An audio send either enters the
old queue before flush and is removed, or observes the flushed state and is rejected. The
same lock also changes paused output back to ready state for the next response. Public
`ConversationInterruption` becomes notification-only and never clears output after the
internal flush.

### Adapter Candidate

The adapter owns one serialized interruption loop. VAD start, VAD end, STT events,
decision deadlines, turn-change completion, and session close enter this loop in order.
The loop owns the sole candidate, transcript gate, held STT queue, and timer. No other
handler reads or writes candidate state.

The timer is selected inside the loop. There is no timer callback, callback wait group, or
second goroutine that can act after a candidate closes. The loop performs the local pause,
continue, or flush action before accepting another candidate. Blocking provider shutdown
and public notifications run outside the loop after the local decision is fixed.

STT events remain in provider order while the transcript gate is open. When provider
speech timestamps exist, the loop trims events ending before the current VAD overlap.
Arrival time is the fallback. The design intentionally adds no utterance identifier. The
maintainer accepts the narrow residual risk that a provider emitting an untimed, severely
late event during a later overlap may be treated as current. Such an event cannot reopen a
closed overlap, and a final transcript received while no gate is open falls back to normal
VAD handling for the current turn.

| Input | Pending candidate action |
| --- | --- |
| VAD start while assistant output is active | Create candidate, send `PauseOutput`, start STT if required, start 500 ms timer |
| Repeated VAD start | Keep the existing candidate and timer |
| Meaningful STT inside the open gate | Hold the text and commit `TurnChangePacket` immediately |
| Empty or filler interim STT | Hold nothing and keep waiting |
| VAD end | Mark speech inactive, send STT end, keep waiting until meaningful text or deadline |
| Deadline while VAD remains active | Commit `TurnChangePacket` |
| Deadline after VAD ended without meaningful text | Send `ContinueOutput` and close candidate |
| Session cancellation | Stop the loop timer, discard held evidence, continue locally paused output when possible, and exit |

For the first release, meaningful text is trimmed non-empty text that is not solely one
of the case-insensitive filler tokens `uh`, `um`, `hmm`, `mm`, `mhm`, `ah`, or `oh` after
surrounding punctuation is removed. Any other recognized token commits the interruption.
Interim text may commit early. Filler-only interim text never continues early because a
later result may add meaningful words.

### Unclear Input After Confirmed Interruption

The unclear-input watchdog is separate from the 500 millisecond interruption decision.
It starts only after meaningful interim STT commits `TurnChangePacket`. It does not start
for VAD-only activity, empty interim text, or filler-only text.

| Event after meaningful interim commit | Watchdog action |
| --- | --- |
| Additional meaningful interim | Extend the existing unclear-input deadline |
| Usable non-empty final STT | Stop the watchdog and continue normal EOS processing |
| Empty final STT | Keep the watchdog active because no usable final was produced |
| False interruption or filler-only overlap | Stop or never start the watchdog |
| Session close | Cancel the watchdog |
| Deadline expires before usable final STT | Inject the configured unclear-input message |

The old assistant response remains flushed after a confirmed interruption. An unclear
prompt is a new assistant response and must never resume the old buffered output.

Direct user text and configured word-trigger interruption do not require a VAD candidate.
They emit `TurnChangePacket` through the existing immediate path. Candidate validation is
required only when the turn-change source is the VAD/STT overlap decision.

### Committed Turn Change

Only `TurnChangePacket` commits interruption. Its handler performs one ordered operation:

1. The serialized loop claims the pending candidate and stops its timer.
2. The loop sends `FlushOutput` before it can accept another VAD start.
3. The loop emits one decision containing the previous context and held meaningful STT.
4. The adapter conditionally rotates the lifecycle context once.
5. EOS, TTS, and LLM owners synchronously invalidate local old-context state. Remote close
   and provider cleanup may continue asynchronously after local invalidation.
6. STT and TTS receive the new context.
7. The public conversation interruption notification is sent and cannot clear output.
8. Held meaningful STT is replayed once after local invalidation completes.
9. The adapter acknowledges completion to the serialized loop, which may then accept a
   new candidate.

An old deadline, VAD end, or STT result arriving after the candidate closes is ignored for
interruption decisions. Normal STT processing resumes for the current turn.

### Implementation Sequence

1. Add internal output-control values and the serialized adapter interruption loop with
   fake streamer tests that assert pause, continue, turn-change, and unclear-input ordering.
2. Add pause, continue, and flush consumption to shared and concrete streamer output paths.
3. Add conditional lifecycle transition and old model-result rejection.
4. Run full verification and independent review before enabling the behavior.

Step 1 may exist on the branch before step 2, but the feature must not be released or
enabled until all steps pass.

## Contracts and Compatibility

- `internal_type.Streamer` does not change.
- No protobuf, network, SDK, or stored configuration contract changes.
- Existing VAD packets remain the activity input during this implementation.
- Existing word-triggered text interruption remains supported.
- The 500 millisecond decision window is an internal constant for the first release.
- Only audio locally owned by the streamer can be paused or discarded.

## Failure and Recovery

- A pause error returned by `sendOutputControl` commits interruption once rather than
  risking continued stale output.
- The timer does not enqueue into a dispatch queue. It resolves only the still-current
  private utterance sequence and cannot leave a candidate paused behind a full queue.
- A flush failure is logged and does not undo an accepted turn change.
- STT errors while VAD remains active commit at the 500 millisecond deadline.
- STT errors after VAD ended continue the paused output at the deadline.
- Duplicate VAD, deadline, continue, flush, and turn-change events are harmless.
- Candidate state and the loop-owned timer are released during session finalization.
- An unclear-input expiry is ignored unless the current context still represents a
  committed meaningful interim without a usable final transcript.

## Security and Privacy

No new external calls, credentials, stored transcript fields, or tenant boundaries are
introduced. Held STT stays in the existing requestor memory and is released or discarded
within the bounded candidate lifetime.

## Observability

Record candidate start, pause request, decision reason, decision latency, continue, commit,
flush failure, and stale-event rejection. Do not log transcript text. Add counters for
candidate outcomes and a duration metric for the decision window.

## Data and Migration

None.

## Rollout

Land the implementation as one release unit. Do not enable the new adapter path until all
configured streamers consume the internal output controls. Stop rollout if output resumes
after a committed turn, held STT reaches the wrong context, timers survive session close,
or false interruptions increase materially.

## Rollback

Revert the complete interruption change set and restore direct legacy VAD handling. No
schema rollback or data repair is required. Active conversations are recycled through the
existing deployment procedure.

## Alternatives Considered

- Add pause methods to `Streamer`: rejected because media control belongs to concrete
  output implementations, not the transport-facing stream contract.
- Add pause identifiers, response identifiers, and output epochs: rejected for this first
  implementation because one requestor owns one active candidate and serializes decisions.
- Flush immediately on VAD: rejected because short noises and fillers would destroy valid
  assistant output.
- Wait for final STT without a fixed deadline: rejected because output could remain paused
  indefinitely.

## Testing and Verification

Required adapter tests use a fake clock and recording streamer:

- VAD start during assistant output sends one pause and does not rotate context.
- repeated VAD start does not create another pause or timer.
- meaningful STT before the deadline emits one turn change.
- filler-only STT followed by VAD end continues at the deadline.
- active VAD at the deadline emits one turn change without STT.
- meaningful interim starts the unclear-input watchdog only after turn change commits.
- additional meaningful interim extends the watchdog and usable final STT stops it.
- VAD-only, empty, and filler-only activity never starts the unclear-input watchdog.
- unclear-input expiry speaks the configured prompt without resuming old output.
- late deadline and late STT cannot affect the next turn.
- a late verdict cannot reopen a closed candidate; timed transcripts before the active overlap are trimmed.
- direct text and word-trigger interruption commit without a VAD candidate.
- committed turn change flushes before TTS and LLM interruption and replays held text once.
- delayed EOS or TTS reset cannot overtake held-text replay.
- audio racing flush is either removed or rejected, and public notification cannot clear new audio.
- pause error commits once and ordinary notification behavior remains unchanged.
- session cancellation stops candidate work.

Required commands after implementation:

```sh
env GOCACHE=/private/tmp/voice-ai-gocache go test -count=1 ./api/assistant-api/internal/type ./api/assistant-api/internal/adapters/internal ./api/assistant-api/internal/adapters/router ./api/assistant-api/internal/adapters/lifecycle ./api/assistant-api/internal/llm/model ./api/assistant-api/internal/transformer/deepgram/... ./api/assistant-api/internal/channel/output ./api/assistant-api/internal/channel/webrtc ./api/assistant-api/internal/channel/grpc ./api/assistant-api/internal/channel/telephony/...
env GOCACHE=/private/tmp/voice-ai-gocache go test -race -count=1 ./api/assistant-api/internal/adapters/internal ./api/assistant-api/internal/adapters/lifecycle ./api/assistant-api/internal/llm/model ./api/assistant-api/internal/transformer/deepgram/... ./api/assistant-api/internal/channel/output ./api/assistant-api/internal/channel/webrtc ./api/assistant-api/internal/channel/telephony/internal/media
git diff --check
```

Run `just agent-finalize` with the exact changed paths after formatting.

## Acceptance Criteria

- [ ] VAD start pauses local assistant output without rotating context or interrupting TTS or LLM.
- [ ] The adapter chooses continue or turn change within 500 milliseconds.
- [ ] Meaningful STT commits immediately; filler-only or empty ended speech continues at the deadline.
- [ ] Active speech at the deadline commits without requiring STT.
- [ ] The unclear-input watchdog starts only for a committed meaningful interim and stops
  only after usable final STT or completed user input.
- [ ] An unclear-input prompt never resumes the flushed assistant response.
- [ ] `TurnChangePacket` is the only interruption commit.
- [ ] Confirmed interruption flushes output and interrupts old TTS, LLM, and EOS work.
- [ ] Held meaningful STT is replayed once into the new context.
- [ ] Late candidate events cannot change a later turn.
- [ ] Delayed STT from a continued utterance cannot decide another candidate in the same context.
- [ ] Direct text and word-trigger interruptions do not require a VAD candidate.
- [ ] Old EOS and TTS state is reset before held STT replay.
- [ ] Audio admission and flush are serialized inside each streamer.
- [ ] `internal_type.Streamer` remains unchanged.
- [ ] Every concrete streamer consumes pause, continue, and flush internally before rollout.
- [ ] Required tests, race tests, finalization, and independent review pass.

## Open Questions

None.

## Challenge Resolution

Earlier drafts were rejected for unbounded final-STT waiting, excess response identity,
unclear repeated-speech behavior, incomplete output ownership, and unsafe lifecycle
completion. This revision applies the maintainer-selected simpler contract: one pending
candidate, one fixed deadline, no public control interface, no pause identifiers, and one
authoritative turn-change commit. Challenge cycle 4 accepted the simpler direction but
found six major gaps: observable pause failures, bounded deadline delivery, same-context
late STT, candidate-free text interruption, reset-before-replay ordering, and output
admission around flush. The maintainer approved alignment with the current design using a
private STT utterance sequence. This revision resolves the other findings with a private
error-returning control path, direct timer resolution with shutdown joining, preserved
direct-text commits, synchronous old-work reset, and streamer-local output admission.
A fresh exact-byte challenge is required.
Challenge cycle 5 rejected the revised bytes with four major findings: candidate actions
can cross after unlocking, joined timer callbacks can execute blocking provider work,
reset-before-replay lacks the required EOS and TTS owner scope, and the continuous STT
client cannot prove immutable utterance attribution. The governed correction limit is
reached. Implementation is blocked pending a maintainer choice between a VAD-only first
release and a broader provider/reset ownership design.
The maintainer selected the serialized recognition model after direct source review and
accepted its documented fallback for untimed late provider events. This revision removes
utterance identifiers, puts the timer and every candidate transition in one loop, keeps
local output actions ordered inside that loop, and broadens scope for local EOS/TTS
invalidation. A fresh exact-byte challenge is required before implementation.
On 2026-09-07 the maintainer added the existing unclear-input behavior to the first
dispatch slice. Meaningful interim STT starts or extends that watchdog after committing
the turn; usable final STT stops it; VAD-only, empty, and filler-only activity never starts
it. This amendment supersedes the prior confirmation gate and requires a new challenge.

## Artifact Index

- `jsons/plan.json` through `jsons/amendment-04-plan.json`: superseded design history.
- `jsons/challenge-01.json` through `jsons/challenge-03.json`: prior challenge evidence.
- `jsons/challenge-04.json`: rejected exact-byte challenge for amendment 05.
- `jsons/challenge-05.json`: rejected exact-byte challenge for amendment 06.
- `jsons/escalation-resolution-01.json`: pause-before-confirm decision.
- `jsons/escalation-resolution-02.json`: required output-control capability decision.
- `jsons/escalation-resolution-03.json`: simplified internal control and 500 ms decision.
- `jsons/escalation-resolution-04.json`: approval for internal STT utterance attribution.
- `jsons/amendment-05-plan.json`: superseded simplified implementation plan.
- `jsons/amendment-06-plan.json`: superseded utterance-sequence implementation plan.
- `jsons/escalation-resolution-05.json`: serialized transcript-gate decision and accepted residual risk.
- `jsons/amendment-07-plan.json`: current implementation plan.
- `jsons/escalation-resolution-06.json`: unclear-input trigger decision.
- `jsons/amendment-08-plan.json`: current dispatch-first implementation plan.
- `jsons/transport-inventory.json`: concrete output ownership inventory.
- `jsons/understanding.json`: current code evidence.
- `jsons/reservation.json`: RFC reservation.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-09-06 | Reserve RFC 0016 before implementation | Coordinator | `jsons/reservation.json` |
| 2026-09-06 | Pause output before deciding whether speech changes the turn | Repository maintainer | `jsons/escalation-resolution-01.json` |
| 2026-09-06 | Require pause, continue, and flush in streamer output paths | Repository maintainer | `jsons/escalation-resolution-02.json` |
| 2026-09-06 | Keep `Streamer` unchanged and use one 500 ms adapter candidate | Repository maintainer | `jsons/escalation-resolution-03.json` |
| 2026-09-06 | Withhold implementation after six blocking findings in the simplified contract | Independent challenger | `jsons/challenge-04.json` |
| 2026-09-06 | Permit a private STT utterance sequence while keeping output controls identifier-free | Repository maintainer | `jsons/escalation-resolution-04.json` |
| 2026-09-06 | Stop after the corrected design still has four blocking findings | Independent challenger | `jsons/challenge-05.json` |
| 2026-09-06 | Select one serialized transcript gate with filler classification and no utterance IDs | Repository maintainer | `jsons/escalation-resolution-05.json` |
| 2026-09-07 | Start unclear-input timeout only after meaningful interim commits and no usable final follows | Repository maintainer | `jsons/escalation-resolution-06.json` |
