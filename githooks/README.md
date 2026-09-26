# Git Hooks

The repository owns `pre-commit`, `commit-msg`, and `pre-push` hooks in this directory. Install them
for the current clone with:

```bash
bin/install-agent-tooling
```

The installer validates the complete toolkit, renders path rules, and sets the local
`core.hooksPath` to `githooks`. This keeps linked worktrees on the same
versioned hooks without generating scripts under the shared Git directory.

- `pre-commit` runs the configured hygiene, style, and drift checks.
- `commit-msg` enforces the Conventional Commit contract from `AGENTS.md`.
- `pre-push` runs the configured push checks, including branch readiness.

Do not use `--no-verify`, skip variables, or an alternate hooks path to bypass a failure.
