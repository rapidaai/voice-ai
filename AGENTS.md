# AGENTS.md

## Scope

Repository-wide operating rules for Codex sessions.

## Development lifecycle

Classify work before creating agents or Orca tasks. Use the lightest tier that safely
fits the change; uncertainty defaults to Standard, not Governed.

### Fast

Use for documentation, comments, formatting, generated-file refreshes, test-only changes,
and isolated low-risk fixes that do not alter a public contract, schema, authentication,
authorization, concurrency, deployment, or data behavior.

Sequence: `understand -> implement -> targeted verification`.

### Standard

Use for normal features and bug fixes contained within one service or component.

Sequence: `understand -> concise plan -> implement -> targeted verification -> review`.

- Record acceptance criteria, allowed paths, risks, and exact verification commands in the task or PR.
- Use execution-focused workers only when parallelism materially helps; give each worker a disjoint write scope.
- Independent review is required before merge for production behavior changes, but it does not require an RFC or decision gate.

### Governed

Use only for public API or protocol changes, authentication or authorization changes,
database/schema/data migrations, cross-service contracts, irreversible operations,
high-risk production rollouts, or when an RFC is explicitly requested.

Sequence: `understand -> plan -> draft RFC -> challenge -> confirm -> implement -> verify -> review -> ship`.

- Reserve an unused `rfcs/NNNN-short-name.md` path before drafting.
- Start from `rfcs/TEMPLATE.md` and store all plan, amendment, challenge, confirmation, review, inventory, and operational JSON under `rfcs/NNNN-short-name/jsons/`.
- The challenger approves only final bytes whose sole metadata status line is `- Status: Accepted`.
- Implementation starts only after the exact-digest confirmation gate is approved.
- Limit plan/RFC and implementation/review correction cycles to two. After two unsuccessful cycles, stop and escalate the unresolved decision instead of retrying.

### Verification

- UI behavior changes in `ui/src/**` include focused UI tests using existing local patterns.
- Backend behavior changes in `api/**`, `pkg/**`, or `cmd/**` include corresponding `*_test.go` coverage in each changed package.
- Run validation explicitly with `make agent-finalize CHANGED_FILES="comma,separated,paths"`; completion hooks must not run tests or block agent exit.
- Ship only with passing required checks and no unresolved critical or major review findings.

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

## Code writing rules

These are the canonical repository-wide implementation rules. Agent-specific instructions
may add domain constraints but must not redefine or weaken this section.

### Ownership and scope

- Give each behavior, state transition, resource, and mutable value one clear owner.
- Do not split, bypass, or duplicate the established ownership model. Extend the existing owner unless the approved design explicitly changes ownership.
- Keep changes inside approved paths; avoid unrelated cleanup and opportunistic refactors.
- Reuse the existing source of truth; do not duplicate policy, state, configuration, or validation.
- Make dependencies explicit at construction or call boundaries; avoid hidden globals and implicit initialization.
- Keep resource creation, use, cancellation, cleanup, and shutdown responsibilities together.

### Simplicity and design

- Apply KISS: prefer direct control flow and the smallest design that fully satisfies the requirement.
- Apply YAGNI: add abstractions, interfaces, options, or configuration only for a current verified need.
- Do not create `helper` or `utils` packages, generic helper functions, or split logic across multiple functions unless reuse, isolation, or complexity makes the extraction clearly necessary.
- Keep related logic together when extraction would hide the execution path or force readers to jump between functions.
- Follow established local patterns before introducing a new package, layer, function, or dependency.
- Keep control flow linear, explicit, and easy to trace. Prefer guard clauses over deeply nested branches and avoid clever expressions.
- A developer should be able to understand the main execution path by reading the primary function from top to bottom.
- Preserve compatibility unless the accepted scope explicitly changes a contract.

### Naming

- Use domain language and one consistent term for each concept; do not introduce synonyms casually.
- Choose names that describe responsibility or intent, not implementation mechanics.
- Avoid vague names such as `util`, `helper`, `manager`, `data`, or `process` unless they are established domain terms.
- Do not introduce identifiers, comments, documentation, or generated prose containing the stem `normaliz`. Use the precise domain operation instead, such as `parse`, `validate`, `trim`, `map`, `canonicalize`, or `convert`.
- Match name length to scope: short conventional names are acceptable locally; exported and long-lived names must be descriptive.
- Name booleans as predicates where practical, using forms such as `is`, `has`, `can`, or `should`.
- Go names use idiomatic MixedCaps, preserve acronym casing, use consistent receiver names, and avoid redundant `Get` prefixes.
- UI components and types use `PascalCase`; functions, variables, and hooks use `camelCase`; hooks begin with `use`.

### Comments and documentation

- Developer comments must normally be one or two lines. Longer explanations belong in an RFC, package documentation, or a focused design document.
- Comments explain why, invariants, ownership, compatibility constraints, or non-obvious tradeoffs. They must not restate what the code already says.
- Do not use em dashes in comments, documentation, commit messages, or generated prose. Use a period, comma, colon, or parentheses instead.
- Keep comments synchronized with behavior; delete stale and commented-out code.
- Exported Go identifiers have useful doc comments when their contract is not already obvious from an established interface.
- TODOs state the remaining action and reference an owner, issue, or removal condition.

### Errors, context, and concurrency

- Return actionable errors with relevant context; preserve causes in Go with `%w` when callers may inspect them.
- Do not log and return the same error at the same ownership layer unless duplicate reporting is intentional.
- Propagate `context.Context` through blocking boundaries; do not store it in structs or replace caller cancellation.
- Every goroutine, channel, timer, stream, and connection has an explicit lifetime and shutdown path.
- Treat invalid input, partial failure, timeout, retry, and cleanup as designed behavior.

### Change quality

- Keep production and test changes focused, deterministic, and reviewable as one coherent diff.
- Add dependencies only when the standard library and existing repository dependencies are insufficient.
- Do not hand-edit generated files; update their source and run the repository generator.
- Run the configured formatter and the narrowest relevant tests before broader validation.
- Tests assert observable behavior and failure paths rather than internal implementation details.

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
