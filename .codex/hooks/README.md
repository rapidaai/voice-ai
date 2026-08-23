# Codex Validation Utilities

Codex does not run these checks automatically. They are explicit validation utilities,
mirrored with Claude for parity.

Included scripts:

- `validate_changed_tests.py`
- `run_required_tests.py`

Preferred usage:

```bash
make agent-finalize CHANGED_FILES="api/example/service.go,api/example/service_test.go"
```

## Scoped file mode

To avoid checking unrelated worktree files, pass changed files explicitly:

```bash
HOOK_CHANGED_FILES="ui/src/providers/openai/stt.json,ui/src/providers/__tests__/config-loader.test.ts" \
python3 .codex/hooks/validate_changed_tests.py </dev/null
```

```bash
HOOK_CHANGED_FILES=$'api/assistant-api/internal/denoiser/denoiser.go\napi/assistant-api/internal/denoiser/denoiser_test.go' \
python3 .codex/hooks/run_required_tests.py </dev/null
```

Resolution order inside validation utilities:
1. `HOOK_CHANGED_FILES` env var
2. paths parsed from hook stdin JSON payload

Repository-wide `git diff` is intentionally not used because worktrees may contain unrelated local or parallel-agent changes.
