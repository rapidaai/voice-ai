# AGENTS.md

## Scope

Repository-wide operating rules for Codex sessions.

## Required development lifecycle

Every non-trivial change follows this sequence:

1. Understand:
- Map the affected contracts, dependencies, owners, existing tests, and production risks.
- Separate verified facts from assumptions and open questions.

2. Plan:
- Define acceptance criteria, allowed paths, out-of-scope paths, file ownership, required tests, required commands, risks, and rollback strategy.
- Record how the plan applies KISS, YAGNI, single ownership, explicit contracts, fail-safe behavior, observability, and least privilege.

3. Discuss:
- A plan challenger who is not the planner must test assumptions, identify a simpler option, and surface compatibility or operational risks.
- Implementation must not start until open questions are resolved and the plan decision is explicitly `approved`.

4. Implement:
- Use execution-focused worker(s) with disjoint ownership by file or module.
- Do not expand scope without returning to the plan and discussion gate.

5. Test and verify:
- UI changes in `ui/src/**` must include UI unit tests using existing local patterns.
- Backend changes in `api/**`, `pkg/**`, or `cmd/**` must include corresponding `*_test.go` updates in the same package.
- Run targeted tests, required validation commands, and relevant integration strict validators.

6. Code review:
- A dedicated code reviewer who did not implement the change must review the complete diff after verification.
- The reviewer is read-only: findings are routed to the implementation owner for correction.
- The reviewer must check correctness, acceptance criteria, scope, simplicity, ownership, contracts, failure behavior, security, observability, tests, and rollback safety.
- Critical or major findings block shipping. Approval must be explicit and include evidence.

7. Ship:
- Ship only after implementation, verification, and independent code review gates pass.
- Preserve the approved plan, verification commands, review report, and unresolved follow-ups in the PR.

See `DEVELOPMENT_PROCESS.md` for role responsibilities and the Orca workflow.

## Engineering principles

- KISS: choose the smallest complete solution and minimize moving parts.
- YAGNI: reject speculative abstractions, configuration, and generalization.
- Ownership: every file, state transition, goroutine, resource, and test has one explicit owner.
- Single source of truth: do not duplicate authoritative state or configuration.
- Explicit contracts: make API, schema, compatibility, timeout, and error behavior visible.
- Fail safely: invalid input, cancellation, timeout, partial failure, and cleanup are intentional paths.
- Observability: changed production behavior has actionable logs, metrics, or traces where appropriate.
- Least privilege: minimize permissions, secret exposure, trust, and data access.
- Reversibility: risky changes include a rollback, disablement, or migration strategy.
- Evidence over confidence: tests, commands, and review findings determine readiness.

## UI testing rules

- Reuse nearby `.test.tsx` / `.spec.tsx` / `__tests__` conventions.
- Include at least:
  - happy-path assertion
  - regression/edge assertion tied to the change
- For provider/config updates, add parity checks against `config-loader` and provider runtime parity test patterns.

## Backend testing rules

- Reuse existing package-level test style and helpers.
- Include at least:
  - success path
  - fallback/error path
  - factory/selection behavior where provider wiring changes
- If the target package already has benchmarks, add/update benchmark coverage for hot-path changes.

## Backend integration boundaries

- STT and TTS:
  - Primary scope: `api/assistant-api/internal/transformer/<provider>/` and `api/assistant-api/internal/transformer/transformer.go`
- VAD:
  - Primary scope: `api/assistant-api/internal/vad/internal/<provider>/` and `api/assistant-api/internal/vad/vad.go`
- End-of-speech:
  - Primary scope: `api/assistant-api/internal/end_of_speech/internal/<provider>/` and `api/assistant-api/internal/end_of_speech/end_of_speech.go`
- Noise reduction:
  - Primary scope: `api/assistant-api/internal/denoiser/internal/<provider>/` and `api/assistant-api/internal/denoiser/denoiser.go`
- Telephony:
  - Primary scope: `api/assistant-api/internal/channel/telephony/internal/<provider>/` and telephony factory files
- LLM:
  - Primary scope: `api/integration-api/internal/caller/<provider>/` and caller factory files

If work needs files outside the selected integration boundary, pause and ask before proceeding.

## Safety and boundaries

- Do not edit out-of-scope modules for the selected integration skill.
- Do not revert unrelated local changes.
- Keep edits minimal and behavior-focused.
