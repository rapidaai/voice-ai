# Codex Hooks Automation (Parity)

Codex does not use Claude hook config directly, but this folder mirrors the same automation checks for parity.

Included scripts:

- `track_changed_files.py` (session-scoped edit tracking)
- `post_tool_test_hint.py`
- `validate_changed_tests.py`
- `run_required_tests.py`
- `enforce_completion.py` (sequenced blocking completion gate)

Usage example:

```bash
python3 .codex/hooks/validate_changed_tests.py </dev/null
python3 .codex/hooks/run_required_tests.py </dev/null
```

## Scoped file mode (recommended for subagents)

To avoid checking unrelated worktree files, pass changed files explicitly:

```bash
HOOK_CHANGED_FILES="ui/src/providers/openai/stt.json,ui/src/providers/__tests__/config-loader.test.ts" \
python3 .codex/hooks/validate_changed_tests.py </dev/null
```

```bash
HOOK_CHANGED_FILES=$'api/assistant-api/internal/denoiser/denoiser.go\napi/assistant-api/internal/denoiser/denoiser_test.go' \
python3 .codex/hooks/run_required_tests.py </dev/null
```

Resolution order inside hooks:
1. `HOOK_CHANGED_FILES` env var
2. paths parsed from hook stdin JSON payload
3. session-scoped paths recorded by `track_changed_files.py`

Repository-wide `git diff` is intentionally not used because worktrees may contain unrelated local or parallel-agent changes.
