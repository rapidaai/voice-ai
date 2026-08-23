---
name: stt-integration
description: Add or modify speech-to-text providers in assistant-api with transport-aware ingestion (WS/SDK/HTTP), transcript packet correctness, and UI/provider wiring.
---

# STT Integration Skill

## Mission

Integrate STT providers that emit reliable interim/final transcripts, interruption signaling, and `stt_latency_ms` metrics.

## Inputs expected from user

1. Ingestion model: streaming WebSocket, SDK callback stream, or HTTP chunk/segment submit.
2. Interim transcript support required or final-only acceptable.
3. Required language/model controls.

If user does not answer:
- Follow nearest streaming baseline and preserve current transcript packet semantics.

## Hard boundaries

In scope:
- `api/assistant-api/internal/transformer/<provider>/stt.go` (+ provider option/callback helpers)
- `api/assistant-api/internal/transformer/transformer.go`
- optional contract updates: `api/assistant-api/internal/type/stt_transformer.go`, `packet.go`
- provider STT JSON + UI config files/components

Out of scope:
- TTS-only transport changes
- telephony channel implementation
- EOS/VAD algorithm changes

## Transport mapping

- WS callback baseline: `deepgram`, `assembly-ai`, `sarvam`
- SDK baseline: `azure`, `google`, `aws`
- HTTP/other baseline: provider-specific fallback keeping packet contract

## Packet contract

Input:
- `UserAudioPacket`

Required outputs:
- `SpeechToTextPacket` (interim/final)
- `InterruptionPacket{Source:"word"}` when provider exposes speech-onset word signal
- `ConversationEventPacket{Name:"stt", ...}`
- `MessageMetricPacket{Name:"stt_latency_ms"}` per utterance

## Implementation workflow

1. Select baseline provider by transport.
2. Implement STT in provider folder and callback/parser helpers.
3. Register provider in STT factory switch.
4. Wire UI model/language metadata and provider form.
5. Validate partial/final ordering and context IDs.
6. Add tests for init, interim/final, reconnect/retry, latency metric.

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/transformer/... -run TestSTT`
- `go test ./api/assistant-api/internal/transformer/<provider>/...`
- `cd ui && yarn test providers`
- `./.claude/skills/stt-integration/scripts/validate.sh --check-diff --provider <provider>`
