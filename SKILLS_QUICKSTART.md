# Skills Quickstart

This repository ships two skill systems:

- Claude skills: `.claude/skills/`
- Codex skills: `.codex/skills/`

Use the skill set that matches your agent runtime.

## Start here

- Claude: `.claude/skills/README.md`
- Codex: `.codex/skills/README.md`
- Security rules:
  - `.claude/skills/SECURITY_GUIDELINES.md`
  - `.codex/skills/SECURITY_GUIDELINES.md`

For features, fixes, and behavior changes, start with `development-lifecycle`. It selects the
required analysis, debugging, implementation, review, delivery, and integration skills for the
chosen Fast, Standard, or Governed tier.

## Local setup skill

If you need local environment setup instructions (Docker and non-Docker), use:

- Claude: `.claude/skills/local-setup-and-run/`
- Codex: `.codex/skills/local-setup-and-run/`

## Validation

Lifecycle skills:

```bash
just validate-agent-tooling
just agent-manifest-check
just agent-drift-check
just agent-style-check
just test-agent-hooks
bin/install-agent-tooling --check
```

Claude domain skills:

```bash
./.claude/skills/<skill>/scripts/validate.sh
```

Codex domain skills:

```bash
./.codex/skills/<skill>/scripts/validate.sh
```

For integration skills that touch providers, use strict mode with provider lock:

```bash
./.claude/skills/<skill>/scripts/validate.sh --check-diff --provider <provider>
./.codex/skills/<skill>/scripts/validate.sh --check-diff --provider <provider>
```

Path-specific instructions come from `agent-rules.json`. After changing the registry, regenerate
and verify its nested `AGENTS.md` and `CLAUDE.md` files:

```bash
python3 bin/render-agent-rules
just agent-drift-check
```

See `AGENT_TOOLING.md` for pull request gates, safety controls, Git hook installation, and
troubleshooting.
