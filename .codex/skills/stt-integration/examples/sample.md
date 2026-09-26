# Sample Output: STT Integration

## Request classification

- Change type: new provider
- Provider: `acme_stt`
- Transport: websocket callback
- Interim transcripts: required

## Edit scope

- `api/assistant-api/internal/transformer/acme_stt/`
- `api/assistant-api/internal/transformer/transformer.go`
- `ui/src/providers/acme_stt/stt.json`
- `ui/src/providers/acme_stt/speech-to-text-models.json`
- `ui/src/providers/acme_stt/languages.json`

## Packet behavior

- Input: `UserAudioPacket`
- Emits interim/final `SpeechToTextPacket`
- Emits `InterruptionPacket{Source:"word"}` when supported
- Emits one `stt_latency_ms` per utterance

## Validation evidence

- `go test ./api/assistant-api/internal/transformer/... -run TestSTT`
- `go test ./api/assistant-api/internal/transformer/acme_stt/...`
- `./.codex/skills/stt-integration/scripts/validate.sh --check-diff --provider acme_stt`
