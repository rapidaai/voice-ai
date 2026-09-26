# Sample Output: Telemetry Integration

## Request classification

- Change type: additive metric and event consistency
- Target provider: `acme_llm`
- Surface: integration-api + assistant provider path

## Edit scope

- `api/integration-api/internal/caller/metrics/metrics_builder.go`
- `api/integration-api/internal/entity/external_audit.go`
- `api/integration-api/internal/caller/acme_llm/llm.go`

## Metric behavior

- retained keys: `TIME_TAKEN`, `STATUS`
- added key: `first_token_latency_ms`
- success and failure both emit terminal status
- sensitive prompt/audio payload not logged as metrics

## Validation evidence

- `go test ./api/integration-api/internal/caller/metrics/...`
- `go test ./api/integration-api/internal/caller/acme_llm/...`
- `./.codex/skills/telemetry-integration/scripts/validate.sh --check-diff --provider acme_llm`
