---
name: vad-integration
description: Add or modify VAD providers and tuning in assistant-api with strict separation from EOS internals. Use for speech activity detection engines, provider registration, and VAD config wiring.
---

# VAD Integration Skill

## Mission

Implement VAD providers that emit stable speech activity and interruption signals with low false positives.

## Hard boundaries

In scope:
- `api/assistant-api/internal/vad/internal/<provider>/...`
- `api/assistant-api/internal/vad/vad.go`
- `api/assistant-api/internal/type/vad.go` and packet compatibility if needed
- `ui/src/providers/<provider>/vad.json`

Out of scope:
- `api/assistant-api/internal/end_of_speech/internal/...`
- EOS factory behavior
- telephony/STT/TTS internals

## Inputs expected from user

1. Detector type: ONNX/native/sdk.
2. Input audio assumptions.
3. Latency vs false-positive tradeoff.

If missing:
- Assume 16kHz LINEAR16 mono internal input.
- Use existing threshold/frame defaults.

## Packet contract

Input:
- `UserAudioPacket`

Required outputs:
- `InterruptionPacket{Source:"vad"}` on speech onset
- `VadSpeechActivityPacket` heartbeat during active speech
- optional `ConversationEventPacket{Name:"vad"}` diagnostics

## Implementation workflow

1. Pick baseline (`silero_vad`, `ten_vad`, `firered_vad`).
2. Implement provider under `internal/<provider>/`.
3. Register provider in `vad.go`.
4. Ensure resource lifecycle and `Close()` safety.
5. Wire VAD UI config.
6. Add unit/benchmark coverage.

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/vad/...`
- `go test -bench=. ./api/assistant-api/internal/vad/internal/<provider>/...`
- `cd ui && yarn test providers`
- `./skills/vad-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/vad-checklist.md`
- `examples/sample.md`
