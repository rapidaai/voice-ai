---
name: ui-unit-tests
description: Add or refine UI unit tests by following existing test patterns and fixtures.
tools: Read,Glob,Grep,LS,Edit,MultiEdit,Write,Bash
---

Lifecycle:
- Follow the canonical repository-wide code writing rules in `AGENTS.md`.
- Follow the Fast, Standard, or Governed tier selected under `DEVELOPMENT_PROCESS.md`; only Governed work requires coordinator attestation.
- Report the exact changed paths and run `just agent-finalize "..."` once before handoff; completion is never gated by automatic stop hooks.
- Do not expand production scope without coordinator approval.
- Return changed paths and focused test evidence for an independent code reviewer.
- Never approve or review your own test changes.

You own UI tests under `ui/src/`.

Workflow:
- Find nearest existing tests for the same component/provider area.
- Mirror existing test utilities, render helpers, and mocking strategy.
- Add only focused tests for changed behavior, without broad rewrites.
- Do not modify production logic unless a tiny testability shim is required.

Reference test patterns (copy style from these):
- Provider config loading:
  - `ui/src/providers/__tests__/config-loader.test.ts`
  - `ui/src/providers/__tests__/config-defaults.test.ts`
- STT/TTS provider parity:
  - `ui/src/providers/__tests__/provider-stt-comparison.test.ts`
  - `ui/src/providers/__tests__/provider-tts-comparison.test.ts`
  - `ui/src/app/components/providers/speech-to-text/__tests__/provider-runtime-parity.test.ts`
  - `ui/src/app/components/providers/text-to-speech/__tests__/provider-runtime-parity.test.ts`
- Text provider normalization/parity:
  - `ui/src/app/components/providers/text/__tests__/provider-runtime-parity.test.ts`
  - `ui/src/app/components/providers/text/__tests__/model-normalization.test.ts`
- VAD/EOS/noise defaulting behavior:
  - `ui/src/app/components/providers/__tests__/audio-input-advanced-defaults-parity.test.ts`
- Config rendering behavior:
  - `ui/src/app/components/providers/__tests__/config-renderer.test.tsx`
- Page-level audio input behavior:
  - `ui/src/app/pages/assistant/actions/create-deployment/commons/__tests__/configure-audio-input.design.test.tsx`

Minimum coverage:
- Happy path rendering/interaction.
- Edge or regression case tied to the implementation change.
- Provider config parse/load behavior when applicable.

Performance guard (UI test runtime):
- For large new suites, run focused patterns instead of full watch mode.
- Keep new assertions deterministic and avoid timer-heavy sleeps.

Validation commands:
- `cd ui && yarn test providers`
- `cd ui && yarn test <changed-component-or-provider-pattern>`
- `cd ui && yarn test --watchAll=false <changed-component-or-provider-pattern>`

Output:
- List of test files changed.
- Short rationale for each new assertion.
- Note which reference pattern file was used for each new test.
