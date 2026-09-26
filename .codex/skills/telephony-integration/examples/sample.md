# Sample Output: Telephony Integration

## Request classification

- Change type: new provider
- Provider: `acme_tel`
- Transport: websocket media
- Direction: inbound + outbound

## Edit scope

- `api/assistant-api/internal/channel/telephony/internal/acme_tel/`
- `api/assistant-api/internal/channel/telephony/telephony.go`
- `ui/src/providers/provider.development.json`
- `ui/src/providers/provider.production.json`
- `ui/src/app/components/providers/telephony/acme-tel/`

## Runtime behavior

- `ReceiveCall` parses caller/context payload
- `InboundCall` returns connect response with context stream URL
- Streamer handles decode/resample/send lifecycle
- Status callback maps provider events to status info

## Validation evidence

- `go test ./api/assistant-api/internal/channel/telephony/...`
- `go test ./api/assistant-api/internal/adapters/internal/...`
- `./.codex/skills/telephony-integration/scripts/validate.sh --check-diff --provider acme_tel`
