# Claude Validation Utilities

This folder contains explicit validation utilities. `.claude/settings.json` intentionally
does not run edit-time or stop-time hooks because hidden tests can trap agents in repeated
completion attempts.

## Behavior

- If UI source changes under `ui/src/`, at least one UI unit test change is required.
- If backend Go source changes under `api/`, `pkg/`, or `cmd/`, at least one `*_test.go` change is required.
- When explicitly finalized, required tests are executed:
  - `cd ui && yarn test providers` for UI changes
  - `go test ./<changed-go-package-dir>` for backend changes

Run validation once with an explicit file scope:

```bash
just agent-finalize "api/example/service.go,api/example/service_test.go" .claude
```

Strict validation returns `2` when checks fail. It never runs automatically during agent exit.

## Scoped file mode

To avoid checking unrelated worktree files, pass changed files explicitly:

```bash
HOOK_CHANGED_FILES="ui/src/providers/openai/stt.json,ui/src/providers/__tests__/config-loader.test.ts" \
python3 .claude/hooks/validate_changed_tests.py </dev/null
```

```bash
HOOK_CHANGED_FILES=$'api/assistant-api/internal/denoiser/denoiser.go\napi/assistant-api/internal/denoiser/denoiser_test.go' \
python3 .claude/hooks/run_required_tests.py </dev/null
```

Resolution order inside validation utilities:
1. `HOOK_CHANGED_FILES` env var
2. paths parsed from hook stdin JSON payload

Repository-wide `git diff` is intentionally not used because worktrees may contain unrelated local or parallel-agent changes.
