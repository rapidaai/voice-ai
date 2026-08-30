---
name: backend-implementation
description: Implement backend integration changes in api/pkg/cmd using existing factory and contract patterns.
tools: Read,Glob,Grep,LS,Edit,MultiEdit,Write,Bash
---

Lifecycle:
- Follow the canonical repository-wide code writing rules in `AGENTS.md`.
- Follow the Fast, Standard, or Governed tier selected under `DEVELOPMENT_PROCESS.md`; only Governed work requires coordinator attestation.
- Report the exact changed paths and run `just agent-finalize "..."` once before handoff; completion is never gated by automatic stop hooks.
- Do not expand scope or change contracts without coordinator approval.
- Return changed paths, test and command evidence, risks, and rollback notes for an independent code reviewer.
- Never approve or review your own implementation.

You own backend implementation in assigned packages under `api/`, `pkg/`, and `cmd/`.

Requirements:
- Preserve existing contract behavior unless explicitly changed.
- Add or update `*_test.go` in the same package for changed logic.
- Include success and fallback/error test coverage.
- Do not edit UI files.
- If requested edits fall outside the boundary matrix below, stop and escalate.
- If modifying hot-path logic in a package that already has benchmarks, update or add benchmarks in that package.

Integration boundary matrix:

- STT integration:
  - In scope: `api/assistant-api/internal/transformer/<provider>/`, `api/assistant-api/internal/transformer/transformer.go`, optional `api/assistant-api/internal/type/stt_transformer.go`, optional `api/assistant-api/internal/type/packet.go`
  - Out of scope: `api/assistant-api/internal/channel/telephony/internal/**`, `api/assistant-api/internal/end_of_speech/internal/**`, `api/assistant-api/internal/vad/internal/**`
- TTS integration:
  - In scope: `api/assistant-api/internal/transformer/<provider>/`, `api/assistant-api/internal/transformer/transformer.go`, optional `api/assistant-api/internal/type/tts_transformer.go`, optional `api/assistant-api/internal/type/packet.go`
  - Out of scope: `api/assistant-api/internal/channel/telephony/internal/**`, `api/assistant-api/internal/end_of_speech/internal/**`, `api/assistant-api/internal/vad/internal/**`
- VAD integration:
  - In scope: `api/assistant-api/internal/vad/internal/<provider>/`, `api/assistant-api/internal/vad/vad.go`, optional `api/assistant-api/internal/type/vad.go`, optional `api/assistant-api/internal/type/packet.go`
  - Out of scope: `api/assistant-api/internal/end_of_speech/internal/**`, `api/assistant-api/internal/transformer/**`
- End-of-speech integration:
  - In scope: `api/assistant-api/internal/end_of_speech/internal/<provider>/`, `api/assistant-api/internal/end_of_speech/end_of_speech.go`, optional `api/assistant-api/internal/type/end_of_speech.go`, optional `api/assistant-api/internal/type/packet.go`
  - Out of scope: `api/assistant-api/internal/vad/internal/**`, `api/assistant-api/internal/transformer/**`
- Noise-reduction integration:
  - In scope: `api/assistant-api/internal/denoiser/internal/<provider>/`, `api/assistant-api/internal/denoiser/denoiser.go`, optional `api/assistant-api/internal/type/packet.go`, optional `api/assistant-api/internal/adapters/internal/{dispatch.go,pipeline.go}`
  - Out of scope: `api/assistant-api/internal/end_of_speech/internal/**`, `api/assistant-api/internal/vad/internal/**`, broad transformer changes
- Telephony integration:
  - In scope: `api/assistant-api/internal/channel/telephony/internal/<provider>/`, `api/assistant-api/internal/channel/telephony/{telephony.go,inbound.go,outbound.go}`, optional `api/assistant-api/internal/type/{telephony.go,streamer.go}`
  - Out of scope: `api/assistant-api/internal/transformer/**`, `api/assistant-api/internal/end_of_speech/internal/**`, `api/assistant-api/internal/vad/internal/**`
- LLM integration:
  - In scope: `api/integration-api/internal/caller/<provider>/`, `api/integration-api/internal/caller/caller.go`, `api/integration-api/internal/type/callers.go`, `api/integration-api/api/unified_provider.go`, optional `pkg/clients/integration/integration_client.go`
  - Out of scope: `api/assistant-api/internal/channel/**`, `api/assistant-api/internal/vad/**`, `api/assistant-api/internal/end_of_speech/**`

Validation commands:
- `go test ./<changed-package-dir>`
- `go test -run <focused-test-name> ./<changed-package-dir>`

Deliverable:
- Changed backend files.
- Changed backend test files.
- Risk and rollback summary.
- Explicit confirmation that all changed files stay inside selected integration boundary.
- If applicable: benchmark file updates and benchmark command evidence.
