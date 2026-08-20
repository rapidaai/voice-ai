---
name: noise-reduction-integration
description: Add or modify microphone noise-reduction providers in assistant-api with denoiser factory wiring, packet safety, and option/UI compatibility.
---

# Noise Reduction Integration Skill

## Mission

Integrate denoisers that reduce background noise while preserving speech intelligibility, timing stability, and packet ordering.

## Inputs expected from user

1. Provider/runtime type: native library, SDK, or cloud API.
2. Input audio assumptions: sample rate, channels, encoding.
3. Target tradeoff: stronger suppression vs speech naturalness.

If missing:
- Assume mono LINEAR16 internal audio path.
- Preserve existing frame/chunk cadence and low-latency defaults.

## Hard boundaries

In scope:
- `api/assistant-api/internal/denoiser/internal/<provider>/...`
- `api/assistant-api/internal/denoiser/denoiser.go`
- `api/assistant-api/internal/type/packet.go` only if packet compatibility requires it
- provider option wiring for `microphone.denoising.provider` in API/UI config paths

Out of scope:
- STT/TTS model internals
- EOS/VAD algorithms
- telephony transport adapters unless required by codec compatibility

## Packet contract

Input:
- `DenoiseAudioPacket`

Required behavior:
- emit denoised audio without changing packet ordering
- preserve timestamps/chunk cadence expected by downstream STT/VAD/EOS
- avoid generating duplicate or empty audio packets

Optional diagnostics:
- conversation/debug events with denoiser provider and processing stats

## Implementation workflow

1. Pick nearest baseline (`rn_noise`, `krisp`).
2. Implement provider package under `internal/<provider>/`.
3. Register provider in denoiser factory switch.
4. Handle init/close lifecycle and fallback behavior safely.
5. Wire provider option/config in API and UI provider config where needed.
6. Add provider tests for default selection, invalid provider fallback, and chunk integrity.

## Governed lifecycle

- Non-trivial work follows `understand -> plan -> challenge -> approve -> implement -> verify -> independent review -> ship` from `DEVELOPMENT_PROCESS.md`.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Implementation starts only from a coordinator-attested approved plan with explicit allowed paths, owners, tests, commands, and rollback.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/denoiser/...`
- `go test -run TestGetDenoiser ./api/assistant-api/internal/denoiser/...`
- `cd ui && yarn test providers`
- `./skills/noise-reduction-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/noise-reduction-checklist.md`
- `examples/sample.md`
