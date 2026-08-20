---
name: noise-reduction-integration
description: Add or modify noise-reduction providers in assistant-api with denoiser factory wiring, packet safety, and UI option compatibility.
---

# Noise Reduction Integration Skill

## Mission

Integrate denoisers that suppress background noise while preserving speech quality, low latency, and packet ordering.

## Inputs expected from user

1. Runtime type: native library, SDK, or service API.
2. Input audio assumptions: sample rate, channels, encoding.
3. Tradeoff target: stronger suppression vs natural voice.

If user does not answer:
- Assume 16kHz LINEAR16 mono internal input.
- Keep current chunk cadence and low-latency defaults.

## Hard boundaries

In scope:
- `api/assistant-api/internal/denoiser/internal/<provider>/...`
- `api/assistant-api/internal/denoiser/denoiser.go`
- optional compatibility updates in `api/assistant-api/internal/type/packet.go`
- `ui/src/providers/<provider>/noise.json`

Out of scope:
- STT/TTS provider internals
- VAD/EOS internal algorithms
- telephony transport internals

## Packet contract

Input:
- `DenoiseAudioPacket`

Required outputs:
- `DenoisedAudioPacket` with stable context and chunk ordering
- preserve downstream cadence expectations for VAD/STT/EOS
- on processing errors, fallback to original audio with `NoiseReduced=false`

Optional diagnostics:
- `ConversationEventPacket{Name:"denoise"}` for provider/debug context

## Implementation workflow

1. Pick baseline (`rn_noise`, `krisp`).
2. Implement provider under `internal/<provider>/`.
3. Register provider in `denoiser.go` factory switch.
4. Ensure lifecycle safety (`Close`, context cancellation, cleanup).
5. Wire provider config/metadata and `noise.json` where needed.
6. Add unit tests for selection/fallback/chunk integrity.

## Done criteria

- Provider selectable by `microphone.denoising.provider`.
- No edits under VAD/EOS provider internals.
- Denoised packet behavior validated by tests.

## Governed lifecycle

- Non-trivial work follows `understand -> plan -> challenge -> approve -> implement -> verify -> independent review -> ship` from `DEVELOPMENT_PROCESS.md`.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Implementation starts only from a coordinator-attested approved plan with explicit allowed paths, owners, tests, commands, and rollback.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/denoiser/...`
- `go test -run TestGetDenoiser ./api/assistant-api/internal/denoiser/...`
- `cd ui && yarn test providers`
- `./.claude/skills/noise-reduction-integration/scripts/validate.sh --check-diff --provider <provider>`
