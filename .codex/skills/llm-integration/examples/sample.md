# Sample Output: LLM Integration

## Request classification

- Change type: new provider
- Provider: `acme_llm`
- Capabilities: chat, stream, verify, embedding

## Edit scope

- `api/integration-api/internal/caller/acme_llm/`
- `api/integration-api/internal/caller/caller.go`
- `api/integration-api/api/unified_provider.go`
- `pkg/clients/integration/integration_client.go`
- `ui/src/providers/acme_llm/text-models.json`

## Runtime behavior

- unified chat route resolves provider caller
- stream sends ordered deltas and completion
- verify credential returns explicit success/failure
- metrics use `TIME_TAKEN` and `STATUS` semantics

## Validation evidence

- `go test ./api/integration-api/internal/caller/...`
- `go test ./api/integration-api/api/...`
- `./.codex/skills/llm-integration/scripts/validate.sh --check-diff --provider acme_llm`
