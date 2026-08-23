---
name: tts-integration
description: Add or modify text-to-speech providers in assistant-api with transport-aware behavior (WS/SSE/SDK/HTTP), packet lifecycle correctness, and UI/provider wiring.
---

# TTS Integration Skill

## Mission

Integrate TTS providers with correct stream/flush/interruption lifecycle and stable `tts_latency_ms` metrics.

## Inputs expected from user

1. Output transport: WS, SSE/chunked HTTP, SDK callback, or HTTP flush.
2. Voice/model requirements.
3. Provider output format constraints.

If missing:
- Choose closest transport baseline and preserve existing packet behavior.

## Hard boundaries

In scope:
- `api/assistant-api/internal/transformer/<provider>/tts.go` (+ option/normalizer helpers)
- `api/assistant-api/internal/transformer/transformer.go`
- optional shared contracts: `tts_transformer.go`, `packet.go`
- provider TTS metadata and UI components

Out of scope:
- STT-only internals
- telephony internals
- EOS/VAD internals

## Packet contract

Input packets:
- `LLMResponseDeltaPacket`
- `LLMResponseDonePacket`
- `InterruptionPacket`

Required outputs:
- `TextToSpeechAudioPacket`
- `TextToSpeechEndPacket`
- `ConversationEventPacket{Name:"tts", ...}`
- `MessageMetricPacket{Name:"tts_latency_ms"}` once per utterance

## Implementation workflow

1. Pick transport baseline (`deepgram`/`rime`/`sarvam`, `minimax`, `azure`/`google`, `aws`).
2. Implement provider package.
3. Register in TTS factory switch.
4. Wire UI provider voice/model config.
5. Verify interruption clears pending state correctly.
6. Add provider + integration tests.

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/transformer/... -run TestTTS`
- `go test ./api/assistant-api/internal/transformer/<provider>/...`
- `cd ui && yarn test providers`
- `./skills/tts-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/tts-checklist.md`
- `examples/sample.md`
