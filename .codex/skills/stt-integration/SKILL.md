---
name: stt-integration
description: Add or modify speech-to-text providers in assistant-api with transport-aware ingestion (WS/SDK/HTTP), transcript packet correctness, and UI/provider wiring.
---

# STT Integration Skill

## Mission

Integrate STT providers that emit consistent interim/final transcripts, word interruption packets, and `stt_latency_ms` metrics.

## Inputs expected from user

1. Ingestion transport: WS, SDK callbacks, or HTTP chunks.
2. Interim transcript requirements.
3. Language/model constraints.

If missing:
- Use nearest transport baseline and preserve packet semantics.

## Hard boundaries

In scope:
- `api/assistant-api/internal/transformer/<provider>/stt.go` (+ helpers)
- `api/assistant-api/internal/transformer/transformer.go`
- optional contract updates: `api/assistant-api/internal/type/stt_transformer.go`, `packet.go`
- provider STT metadata and UI components

Out of scope:
- TTS-only internals
- telephony internals
- EOS/VAD algorithms

## Packet contract

Input:
- `UserAudioPacket`

Required outputs:
- `SpeechToTextPacket` interim/final
- `InterruptionPacket{Source:"word"}` when available
- `ConversationEventPacket{Name:"stt", ...}`
- `MessageMetricPacket{Name:"stt_latency_ms"}` once per utterance

## Implementation workflow

1. Pick baseline (`deepgram`, `assembly-ai`, `azure`, `google`, `aws`, `sarvam`).
2. Implement provider package under transformer.
3. Register in STT factory switch.
4. Wire UI provider model/language config.
5. Validate transcript ordering and context consistency.
6. Add provider + integration tests.

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/transformer/... -run TestSTT`
- `go test ./api/assistant-api/internal/transformer/<provider>/...`
- `cd ui && yarn test providers`
- `./skills/stt-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/stt-checklist.md`
- `examples/sample.md`
