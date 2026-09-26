# Sample Output: End Of Speech Integration

## Request classification

- Change type: new provider
- Provider: `acme_eos`
- Signal mode: audio-model
- Priority: lower latency

## Edit scope

- `api/assistant-api/internal/end_of_speech/internal/acme_eos/`
- `api/assistant-api/internal/end_of_speech/end_of_speech.go`
- `ui/src/providers/acme_eos/eos.json`
- `ui/src/providers/config-loader.ts`

Out of scope confirmed:
- `api/assistant-api/internal/vad/internal/*`

## Packet behavior

- Consumes: `UserAudioPacket`, `SpeechToTextPacket`, `InterruptionPacket`
- Emits: `InterimEndOfSpeechPacket`, `EndOfSpeechPacket`, `ConversationEventPacket{Name:"eos"}`
- Dedup guard: generation + fired flag

## Validation evidence

- `go test ./api/assistant-api/internal/end_of_speech/...`
- `go test ./api/assistant-api/internal/adapters/internal/...`
- `./.codex/skills/end-of-speech-integration/scripts/validate.sh --check-diff --provider acme_eos`
