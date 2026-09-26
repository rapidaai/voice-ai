# Sample Output: VAD Integration

## Request classification

- Change type: new provider
- Provider: `acme_vad`
- Detector: ONNX
- Tradeoff: lower false positives

## Edit scope

- `api/assistant-api/internal/vad/internal/acme_vad/`
- `api/assistant-api/internal/vad/vad.go`
- `ui/src/providers/acme_vad/vad.json`

Out of scope confirmed:
- `api/assistant-api/internal/end_of_speech/internal/*`

## Packet behavior

- Input: `UserAudioPacket`
- Emits: `InterruptionPacket{Source:"vad"}`, `VadSpeechActivityPacket`
- Cleanup: detector/session closed on `Close` and context cancel

## Validation evidence

- `go test ./api/assistant-api/internal/vad/...`
- `go test -bench=. ./api/assistant-api/internal/vad/internal/acme_vad/...`
- `./.codex/skills/vad-integration/scripts/validate.sh --check-diff --provider acme_vad`
