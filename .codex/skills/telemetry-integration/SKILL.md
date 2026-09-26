---
name: telemetry-integration
description: Add or modify telemetry instrumentation across assistant-api and integration-api while preserving metric schema compatibility and audit behavior.
---

# Telemetry Integration Skill

## Mission

Extend observability without breaking existing metric consumers, dashboards, or external audit mappings.

## Inputs expected from user

1. Target surface: assistant-api, integration-api, or both.
2. Metric/event requirements.
3. Privacy/cardinality constraints.

If missing:
- Keep existing key conventions and use additive metrics.

## Hard boundaries

In scope:
- `api/integration-api/internal/caller/metrics/metrics_builder.go`
- provider-specific caller metrics hooks
- `api/integration-api/internal/entity/external_audit.go`
- assistant provider event/metric emission for touched path only

Out of scope:
- unrelated business logic not needed for instrumentation

## Compatibility rules

- Do not rename existing keys unless consumers are migrated.
- Emit success/failure symmetrically.
- Keep sensitive payloads out of metrics.
- Emit first-byte/first-token latency once per turn.

## Lifecycle integration

- `development-lifecycle` owns tier selection and the repository-level change contract.
- Use `change-analysis` when ownership, consumers, or blast radius is not already proven.
- This skill owns its domain evidence and boundaries; `developing-change` owns implementation discipline.
- Use `debugging` for unexplained failures and `writing-documentation` for documentation changes.
- Use `reviewing-change` for independent review and `responding-to-review` for its findings.
- Use `preparing-delivery` only when the user explicitly requests a delivery action.

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
- `./.codex/skills/telemetry-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/telemetry-checklist.md`
- `examples/sample.md`
