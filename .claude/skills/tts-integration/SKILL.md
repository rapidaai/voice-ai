---
name: tts-integration
description: Add or modify text-to-speech providers in assistant-api with transport-aware behavior (WS/SSE/SDK/HTTP), correct packet lifecycle, and UI/provider wiring.
---

# TTS Integration Skill

## Mission

Integrate TTS providers with correct streaming/flush/interruption semantics and stable `tts_latency_ms` metrics.

## Inputs expected from user

1. Transport model: bidirectional WebSocket, SSE/chunked HTTP, SDK callback, or request-response flush.
2. Voice/model parameter requirements.
3. Output audio format from provider.

If user does not answer:
- Select nearest baseline transport and preserve existing packet semantics.
- Keep internal output format compatibility with existing streamer path.

## Hard boundaries

In scope:
- `api/assistant-api/internal/transformer/<provider>/tts.go` (+ provider option/normalizer helpers)
- `api/assistant-api/internal/transformer/transformer.go` (factory case)
- optional contract updates: `api/assistant-api/internal/type/tts_transformer.go`, `packet.go`
- provider config JSON + UI component files for TTS

Out of scope:
- STT-only logic changes unrelated to shared provider setup
- telephony transport internals
- EOS/VAD internals

## Transport mapping

- WS streaming baseline: `deepgram`, `rime`, `sarvam`
- SSE/chunked baseline: `minimax`
- SDK/API callback baseline: `azure`, `google`
- HTTP flush baseline: `aws`, `resembleai`

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

1. Pick transport baseline and clone lifecycle shape.
2. Implement provider under `transformer/<provider>/`.
3. Add factory registration in `transformer.go`.
4. Wire UI metadata (`voices`, `languages`, `text-to-speech-models`) and config form.
5. Verify interruption behavior clears output state without deadlocks.
6. Add provider tests (init, streaming, completion, interruption, metric-on-first-audio).

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/transformer/... -run TestTTS`
- `go test ./api/assistant-api/internal/transformer/<provider>/...`
- `cd ui && yarn test providers`
- `./.claude/skills/tts-integration/scripts/validate.sh --check-diff --provider <provider>`
