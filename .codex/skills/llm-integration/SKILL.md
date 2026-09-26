---
name: llm-integration
description: Add or modify integration-api LLM providers with caller factory wiring, unified provider routing, streaming behavior, and metric/audit compatibility.
---

# LLM Integration Skill

## Mission

Add provider callers for chat/stream/verify (plus optional embedding/reranking) with consistent metrics and routing.

## Inputs expected from user

1. Required capabilities: chat, stream, verify, embedding, reranking.
2. Auth and endpoint model.
3. Model parameter/tool-calling constraints.

If missing:
- Implement chat + stream + verify first.
- Add embedding/reranking only if provider supports them.

## Hard boundaries

In scope:
- `api/integration-api/internal/caller/<provider>/...`
- `api/integration-api/internal/caller/caller.go`
- `api/integration-api/internal/type/callers.go` if interfaces change
- `api/integration-api/api/unified_provider.go`
- `pkg/clients/integration/integration_client.go` if routing updates are needed
- provider model metadata in `ui/src/providers/<provider>/...`

Out of scope:
- assistant-api telephony/STT/TTS/EOS/VAD internals

## Implementation workflow

1. Build provider caller package (`llm`, `verify-credential`, optional embedding/reranking).
2. Register in caller factory switches.
3. Ensure unified provider endpoint behavior is consistent.
4. Keep MetricBuilder lifecycle consistent (`TIME_TAKEN`, `STATUS`).
5. Add tests for success/failure and stream completion.
6. Update provider model metadata where required.

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

- `go test ./api/integration-api/internal/caller/...`
- `go test ./api/integration-api/api/...`
- `go test ./pkg/clients/integration/...`
- `./.codex/skills/llm-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/llm-checklist.md`
- `examples/sample.md`
