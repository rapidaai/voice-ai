---
name: telemetry-integration
description: Add or modify telemetry/metrics instrumentation for assistant-api and integration-api flows while preserving metric schema compatibility and audit behavior.
---

# Telemetry Integration Skill

## Mission

Extend observability without breaking existing metric consumers, dashboards, or external audit mapping.

## Inputs expected from user

1. Target surface: assistant pipeline, integration callers, or both.
2. New metric/event names and semantic definitions.
3. Cardinality and privacy constraints.

If user does not answer:
- Reuse existing key conventions (`TIME_TAKEN`, `STATUS`, `stt_latency_ms`, `tts_latency_ms`).
- Prefer additive metrics over key redefinition.

## Hard boundaries

In scope:
- `api/integration-api/internal/caller/metrics/metrics_builder.go`
- provider-specific caller metrics hooks under `api/integration-api/internal/caller/<provider>/`
- audit mapping compatibility in `api/integration-api/internal/entity/external_audit.go`
- assistant packet event/metric emissions in touched provider flow only

Out of scope:
- unrelated business logic changes not needed for instrumentation

## Compatibility rules

- Do not rename existing metric keys unless all readers are updated.
- Emit success/failure status symmetrically.
- Keep sensitive payloads out of metric values.
- Ensure first-byte/first-token latency metrics are emitted once per turn.

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/integration-api/internal/caller/metrics/...`
- `go test ./api/integration-api/internal/caller/<provider>/...`
- `go test ./api/assistant-api/internal/transformer/<provider>/...`
- `rg -n "TIME_TAKEN|STATUS|stt_latency_ms|tts_latency_ms" api`
- `./.claude/skills/telemetry-integration/scripts/validate.sh --check-diff --provider <provider>`
