---
name: end-of-speech-integration
description: Add or modify end-of-speech integrations in assistant-api with strict separation from VAD internals. Use for transcript/audio/history-aware turn-finalization logic, provider wiring, and EOS UI config.
---

# End Of Speech Integration Skill

## Mission

Implement EOS that finalizes each user turn exactly once, at low latency, without mutating VAD behavior.

## Hard boundaries

In scope:
- `api/assistant-api/internal/end_of_speech/internal/<provider>/...`
- `api/assistant-api/internal/end_of_speech/end_of_speech.go` (factory registration)
- `api/assistant-api/internal/type/end_of_speech.go` and packet compatibility only if required
- EOS config in `ui/src/providers/<provider>/eos.json` (plus optional `model-options.json`)
- EOS UI rendering under `ui/src/app/components/providers/end-of-speech/`

Out of scope:
- `api/assistant-api/internal/vad/internal/...` implementation changes
- `api/assistant-api/internal/vad/vad.go` provider logic changes
- STT/TTS/telephony provider implementation changes

## Inputs expected from user

1. EOS signal mode: transcript-only, audio-model, or history-aware model.
2. Priority: lower latency or lower false-finalization.
3. Any deployment/model constraints.

If user does not answer:
- Use transcript-only (`silence_based_eos`) for text/STT driven flows.
- Keep defaults for `threshold`, `quick_timeout`, `silence_timeout`.

## Packet contract

Accepted packet inputs depend on provider strategy:
- transcript flow: `SpeechToTextPacket`, `UserTextPacket`
- timing/reset flow: `InterruptionPacket`, `VadSpeechActivityPacket`
- model-aware flow: optional `UserAudioPacket`, `LLMResponseDonePacket`

Required outputs:
- `InterimEndOfSpeechPacket`
- `EndOfSpeechPacket` (once per utterance)
- `ConversationEventPacket{Name:"eos", ...}`

## Implementation workflow

1. Choose baseline provider to clone:
- transcript timer: `internal/silence_based/...`
- audio smart-turn: `internal/pipecat/...`
- history/model-aware: `internal/livekit/...`
2. Add new provider package under `internal/<provider>/`.
3. Register provider constant and switch case in `end_of_speech.go`.
4. Ensure interim/final packet ordering is stable and deduplicated.
5. Add/update EOS UI config JSON and component mapping.
6. Add tests for interruption, interim reset, timeout, and duplicate-final guard.

## Done criteria

- Provider selectable via `microphone.eos.provider`.
- No edits under `api/assistant-api/internal/vad/internal/`.
- Final EOS emitted once per turn in tests.
- Config loads via `ui/src/providers/config-loader.ts` path resolution.

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/end_of_speech/...`
- `go test ./api/assistant-api/internal/adapters/internal/...`
- `cd ui && yarn test providers`
- `./.claude/skills/end-of-speech-integration/scripts/validate.sh --check-diff --provider <provider>`
