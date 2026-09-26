# Sample Output: TTS Integration

## Request classification

- Change type: new provider
- Provider: `acme_tts`
- Transport: SSE/chunked HTTP
- Interruption behavior: reset pending synthesis

## Edit scope

- `api/assistant-api/internal/transformer/acme_tts/`
- `api/assistant-api/internal/transformer/transformer.go`
- `ui/src/providers/acme_tts/tts.json`
- `ui/src/providers/acme_tts/voices.json`
- `ui/src/providers/acme_tts/text-to-speech-models.json`

## Packet behavior

- Consumes delta/done/interruption packets
- Emits `TextToSpeechAudioPacket` chunks
- Emits `TextToSpeechEndPacket` on flush complete
- Emits one `tts_latency_ms` on first audio

## Validation evidence

- `go test ./api/assistant-api/internal/transformer/... -run TestTTS`
- `go test ./api/assistant-api/internal/transformer/acme_tts/...`
- `./.codex/skills/tts-integration/scripts/validate.sh --check-diff --provider acme_tts`
