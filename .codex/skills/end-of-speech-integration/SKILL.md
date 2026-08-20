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
- `api/assistant-api/internal/end_of_speech/end_of_speech.go`
- `api/assistant-api/internal/type/end_of_speech.go` and packet compatibility only if required
- `ui/src/providers/<provider>/eos.json` and optional `model-options.json`
- `ui/src/app/components/providers/end-of-speech/`

Out of scope:
- `api/assistant-api/internal/vad/internal/...`
- VAD factory/provider behavior
- STT/TTS/telephony provider internals

## Inputs expected from user

1. EOS signal mode: transcript-only, audio-model, or history-aware.
2. Priority: lower latency or lower false-finalization.
3. Deployment/model constraints.

If missing:
- Default to transcript-only (`silence_based_eos`) for text/STT flows.
- Keep current threshold/timeout defaults.

## Packet contract

Consumed packets vary by provider:
- transcript flow: `SpeechToTextPacket`, `UserTextPacket`
- timer reset flow: `InterruptionPacket`, `VadSpeechActivityPacket`
- model-aware flow: optional `UserAudioPacket`, `LLMResponseDonePacket`

Required outputs:
- `InterimEndOfSpeechPacket`
- `EndOfSpeechPacket` (exactly once per utterance)
- `ConversationEventPacket{Name:"eos", ...}`

## Implementation workflow

1. Choose baseline (`silence_based`, `pipecat`, `livekit`).
2. Implement provider in `internal/<provider>/`.
3. Register provider in EOS factory switch.
4. Enforce deterministic interim/final packet order.
5. Wire UI provider config and component mapping.
6. Add tests for timeout/interruption/dedup-finalization.

## Governed lifecycle

- Non-trivial work follows `understand -> plan -> challenge -> approve -> implement -> verify -> independent review -> ship` from `DEVELOPMENT_PROCESS.md`.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Implementation starts only from a coordinator-attested approved plan with explicit allowed paths, owners, tests, commands, and rollback.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/end_of_speech/...`
- `go test ./api/assistant-api/internal/adapters/internal/...`
- `cd ui && yarn test providers`
- `./skills/end-of-speech-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/eos-checklist.md`
- `examples/sample.md`
