---
name: llm-integration
description: Add or modify LLM/integration providers in integration-api with caller factory wiring, unified provider routing, streaming behavior, and metric/audit compatibility.
---

# LLM Integration Skill

## Mission

Add provider callers that support chat/stream/verify (and optional embedding/reranking) with consistent metrics and unified routing.

## Inputs expected from user

1. Provider capabilities required: chat, stream, verify, embeddings, reranking.
2. Auth style and endpoint shape.
3. Any model parameter or tool-calling constraints.

If user does not answer:
- Implement chat + stream + verify first.
- Add embeddings/reranking only when provider supports them.

## Hard boundaries

In scope:
- `api/integration-api/internal/caller/<provider>/...`
- `api/integration-api/internal/caller/caller.go`
- `api/integration-api/internal/type/callers.go` when interface compatibility changes are required
- `api/integration-api/api/unified_provider.go`
- `pkg/clients/integration/integration_client.go` if routing/client behavior needs updates
- provider UI model metadata in `ui/src/providers/<provider>/...`

Out of scope:
- assistant-api telephony/STT/TTS/EOS/VAD internals

## Implementation workflow

1. Create provider caller package with `llm.go`, `verify-credential.go`, and optional embedding/reranking files.
2. Register provider in `caller.go` factory methods.
3. Ensure unified gRPC route usage supports provider name casing/validation.
4. Emit metrics via `MetricBuilder` semantics (`TIME_TAKEN`, `STATUS`, request ID).
5. Add tests for chat success/failure, streaming, verify credential, optional embedding/reranking.
6. Update UI provider catalogs (`text-models.json` / `models.json`) as needed.

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed implementation starts only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/integration-api/internal/caller/...`
- `go test ./api/integration-api/api/...`
- `go test ./pkg/clients/integration/...`
- `./.claude/skills/llm-integration/scripts/validate.sh --check-diff --provider <provider>`
