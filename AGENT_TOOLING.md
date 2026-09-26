# Agent Tooling

This repository applies the same lifecycle, path, review, and delivery policy across supported
agent runtimes. `AGENTS.md` is the canonical engineering policy. This document explains the
executable controls.

`agent-tooling-manifest.json` is the capability and coverage registry. Its state values distinguish
implemented controls from partial or designed work, and every capability records evidence,
verification, and its remaining limitation.

## Controls

| Control | Source of truth | Enforcement |
| --- | --- | --- |
| Permission deny rules | `.claude/settings.json` | Reject sensitive reads, generated-file edits, destructive commands, hook bypasses, and release mutations. |
| Command safety | `.claude/hooks/bash_guard.py` | Reject destructive shell and Git commands plus unfiltered outbound transfers. |
| Plan reminder | `.claude/hooks/plan_reminder.py` | Remind the first source edit in a session to establish the selected lifecycle contract. |
| Pull request policy | `.github/PULL_REQUEST_TEMPLATE.md` | `bin/check-pr-body`, `.claude/hooks/pr_gate.py`, `bin/agent-pr-ready`, and the `Agent Policy` workflow. |
| Path-specific rules | `agent-rules.json` | Generated nested instructions for both runtimes and Claude pre-edit context injection. |
| Drift detection | `agent-rules.json` and paired agent assets | `just agent-drift-check` and `just validate-development-toolkit`. |
| Style | `AGENTS.md` | `just agent-style-check`, the pre-commit hook, and the pull request policy workflow. |
| Hook behavior | `tests/agent_tooling/` | `just test-agent-hooks`. |
| External content | `bin/agent-egress` | Allowlisted paths, sensitive-path refusal, and secret-looking line redaction. |
| Git lifecycle | `githooks/` and `.pre-commit-config.yaml` | Versioned pre-commit, commit-message, and pre-push gates. |
| Independent review | `bin/agent-review` | Run configured read-only reviewers and aggregate a validated Standard or high-risk panel. |
| Browser quality | `ui/playwright.config.ts` | Run Chromium accessibility checks and a reviewed screenshot contract for a stable critical route. |
| Capability status | `agent-tooling-manifest.json` | Declare built, partial, and designed controls with executable evidence. |

## Commands

```bash
python3 bin/render-agent-rules
just agent-manifest-check
just agent-drift-check
just agent-style-check
just test-agent-hooks
just validate-development-toolkit
just agent-pr-ready main
just ui-browser-test
bin/install-agent-tooling
```

Before any repository-owned script sends content to an external service, validate each source path:

```bash
bin/agent-egress paths AGENTS.md AGENT_TOOLING.md
bin/agent-egress text AGENT_TOOLING.md /tmp/agent-tooling.filtered.md
bin/agent-egress diff /tmp/change.diff /tmp/change.filtered.diff
```

The filter is defense in depth. It does not replace secret management, explicit authorization, or
careful review of the filtered output and its adjacent manifest.

## Independent Review Panel

Copy `agent-reviewers.example.json` to the ignored `.agent-reviewers.json`, enable the available
review commands, and keep each command read-only. Commands either receive the prompt on standard
input or receive a `{prompt_file}` argument.

```bash
bin/agent-review --working-tree --risk standard
bin/agent-review --base main --risk high
```

Standard review requires one configured reviewer. High-risk review requires at least two distinct
model families. External transports do not run unless `--allow-external` is provided. Every review
uses the egress filter, rejects invalid result schemas, detects candidate mutation, and writes the
combined JSON and Markdown panel under the ignored `.agent-reviews/` directory by default.

## Browser Quality

`just ui-browser-test` starts the UI when `PLAYWRIGHT_BASE_URL` is absent and runs Chromium checks.
The initial contract covers the sign-in route for serious accessibility violations and a stable
authentication-card screenshot. Update a screenshot only after reviewing the rendered difference:

```bash
cd ui
yarn test:e2e:update
```

## Lessons

`AGENT_LESSONS.md` records concrete workflow defects and the guard that prevents each recurrence.
Add entries only when evidence establishes a reusable invariant.

## Pull Request Protection

The workflow publishes the `Agent Policy / Agent Policy` status check. Repository administrators
must require that check and `05 CI Complete` in branch protection. Repository files cannot enforce
the hosting service's branch-protection setting by themselves.

## Troubleshooting

| Symptom | Likely cause | Resolution |
| --- | --- | --- |
| A safe shell command is blocked | The command matches a deny rule or a compound command contains a blocked segment. | Split the command, inspect each segment, and use the repository-owned safe command. Do not bypass the guard. |
| A pull request command is blocked | The title or body is incomplete, the body is inline, or readiness checks fail. | Complete `.github/PULL_REQUEST_TEMPLATE.md` in a repository-local file, run `just agent-pr-ready <base>`, then retry with `--body-file`. |
| Path guidance does not appear | The path matches no rule, the hook is not configured, or the registry is invalid. | Run `python3 .claude/hooks/path_rules.py` with a test payload through `just test-agent-hooks`, then run `just agent-drift-check`. |
| A source edit receives a plan reminder | The session has reached its first source edit. | Confirm the selected tier and its change contract. Governed work must stop until its exact-digest gate is approved. |
| Generated rule files differ | `agent-rules.json` changed without regeneration or a generated file was edited. | Run `python3 bin/render-agent-rules`, inspect the diff, and rerun `just agent-drift-check`. |
| Style validation fails | A changed line violates the prose, comment, TODO, or identifier rules in `AGENTS.md`. | Fix the reported line. Do not suppress or exclude it. |
| Egress validation refuses a file | The path is outside the repository, sensitive, binary, too large, or outside the allowlist. | Do not send it. Select the minimum safe source files or produce an approved redacted artifact. |
| Git hooks do not run | `core.hooksPath` is not configured for this clone. | Run `bin/install-agent-tooling`, then confirm `git config --get core.hooksPath` returns `githooks`. |
| Review configuration is rejected | No reviewer is enabled, a command is unavailable, or high-risk reviewers share one model family. | Fix `.agent-reviewers.json`, run `bin/agent-review --check`, and retry without weakening the review policy. |
| Browser checks fail | Chromium is missing, the UI did not start, accessibility regressed, or the screenshot changed. | Run `cd ui && yarn playwright install chromium`, inspect the failure artifact, and update a screenshot only for an accepted UI change. |
| A required tool is unavailable | Python, Just, or a language toolchain is missing. | Install the documented prerequisite and rerun the exact failed command. Report the limitation if installation is outside scope. |
