# Sample Output: Noise Reduction Integration

## Request classification

- Change type: new provider
- Provider: `acme_denoise`
- Runtime: native library
- Tradeoff: stronger suppression, keep speech naturalness

## Edit scope

- `api/assistant-api/internal/denoiser/internal/acme_denoise/`
- `api/assistant-api/internal/denoiser/denoiser.go`
- `ui/src/providers/acme_denoise/noise.json`

Out of scope confirmed:
- `api/assistant-api/internal/end_of_speech/internal/*`
- `api/assistant-api/internal/vad/internal/*`

## Packet behavior

- Input: `DenoiseAudioPacket`
- Emits: `DenoisedAudioPacket`
- Guarantees: preserves context IDs, ordering, and chunk cadence
- Fallback: emits original audio with `NoiseReduced=false` on provider error

## Validation evidence

- `go test ./api/assistant-api/internal/denoiser/...`
- `go test -run TestGetDenoiser ./api/assistant-api/internal/denoiser/...`
- `./.codex/skills/noise-reduction-integration/scripts/validate.sh --check-diff --provider acme_denoise`
