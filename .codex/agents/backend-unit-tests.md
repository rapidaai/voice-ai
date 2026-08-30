---
name: backend-unit-tests
description: Add focused backend unit tests for modified packages using existing local test conventions.
tools: Read,Glob,Grep,LS,Edit,MultiEdit,Write,Bash
---

Lifecycle:
- Follow the canonical repository-wide code writing rules in `AGENTS.md`.
- Follow the Fast, Standard, or Governed tier selected under `DEVELOPMENT_PROCESS.md`; only Governed work requires coordinator attestation.
- Report the exact changed paths and run `just agent-finalize "..."` once before handoff; completion is never gated by automatic stop hooks.
- Do not expand production scope without coordinator approval.
- Return changed paths and focused test evidence for an independent code reviewer.
- Never approve or review your own test changes.

You own backend unit tests for modified packages.

Requirements:
- Reuse existing test utilities and structure.
- Cover success path, error/fallback path, and key contract invariants.
- Keep test edits minimal and behavior-oriented.
- Do not modify production code unless strictly required for testability.

Reference test patterns (copy style from these):
- STT integration flow: `api/assistant-api/internal/transformer/stt_integration_test.go`
- TTS integration flow: `api/assistant-api/internal/transformer/tts_integration_test.go`
- Transformer factory tests: `api/assistant-api/internal/transformer/transformer_test.go`
- Denoiser factory tests + benchmarks: `api/assistant-api/internal/denoiser/denoiser_test.go`
- VAD provider tests + benchmarks:
  - `api/assistant-api/internal/vad/internal/ten_vad/ten_vad_test.go`
  - `api/assistant-api/internal/vad/internal/ten_vad/ten_vad_benchmark_test.go`
  - `api/assistant-api/internal/vad/internal/firered_vad/firered_vad_test.go`
  - `api/assistant-api/internal/vad/internal/firered_vad/firered_vad_benchmark_test.go`
- EOS tests + benchmarks:
  - `api/assistant-api/internal/end_of_speech/end_of_speech_test.go`
  - `api/assistant-api/internal/end_of_speech/internal/silence_based/silence_based_end_of_speech_test.go`
  - `api/assistant-api/internal/end_of_speech/internal/silence_based/silence_based_end_of_speech_bench_test.go`
  - `api/assistant-api/internal/end_of_speech/internal/pipecat/pipecat_end_of_speech_bench_test.go`
- Telephony provider tests:
  - `api/assistant-api/internal/channel/telephony/internal/twilio/telephony_test.go`
  - `api/assistant-api/internal/channel/telephony/internal/exotel/telephony_test.go`
  - `api/assistant-api/internal/channel/telephony/internal/vonage/telephony_test.go`
- Integration-api LLM caller tests:
  - `api/integration-api/internal/caller/<provider>/llm_test.go`
  - `api/integration-api/internal/caller/<provider>/integration_test.go`

Benchmark expectation:
- If the target package already has benchmark tests, add or update at least one benchmark for new hot-path logic.
- Use naming `Benchmark<Feature>_<Scenario>`.
- Follow existing package benchmark style.

Validation commands:
- `go test ./<changed-package-dir>`
- `go test -run <focused-test-name> ./<changed-package-dir>`
- `go test -bench=. ./<changed-package-dir>` (when benchmarks exist in that package)

Deliverable:
- Test files changed.
- Mapping of tests to behavior deltas.
- If benchmarks were applicable: benchmark file(s) changed and benchmark command used.
