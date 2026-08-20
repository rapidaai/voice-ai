# Claude Hooks Automation

This folder contains command hooks wired from `.claude/settings.json`.

## Installed hooks

- `PostToolUse`:
  - `track_changed_files.py` for Edit/Write/NotebookEdit events
  - `post_tool_test_hint.py` (non-blocking reminder)
- `SubagentStop`:
  - `enforce_completion.py` (sequenced blocking gate)
- `Stop`:
  - `enforce_completion.py` (sequenced blocking gate)

## Behavior

- If UI source changes under `ui/src/`, at least one UI unit test change is required.
- If backend Go source changes under `api/`, `pkg/`, or `cmd/`, at least one `*_test.go` change is required.
- On stop/subagent-stop, required tests are executed:
  - `cd ui && yarn test providers` for UI changes
  - `go test ./<changed-go-package-dir>` for backend changes

Hook scripts return `2` to block completion when checks fail.

## Scoped file mode (recommended for subagents)

To avoid checking unrelated worktree files, pass changed files explicitly:

```bash
HOOK_CHANGED_FILES="ui/src/providers/openai/stt.json,ui/src/providers/__tests__/config-loader.test.ts" \
python3 .claude/hooks/validate_changed_tests.py </dev/null
```

```bash
HOOK_CHANGED_FILES=$'api/assistant-api/internal/denoiser/denoiser.go\napi/assistant-api/internal/denoiser/denoiser_test.go' \
python3 .claude/hooks/run_required_tests.py </dev/null
```

Resolution order inside hooks:
1. `HOOK_CHANGED_FILES` env var
2. paths parsed from hook stdin JSON payload
3. session-scoped paths recorded by `track_changed_files.py`

Repository-wide `git diff` is intentionally not used because worktrees may contain unrelated local or parallel-agent changes.
