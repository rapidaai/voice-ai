# Codex Skills README

This directory contains Codex skills for integration work in this repository.

## Install

Codex can load skills from:

- repo-local: `.codex/skills/`
- user-global: `~/.codex/skills/`

For this repository, skills are already installed at `.codex/skills/`.

Install into repo-local path in another repository:

```bash
mkdir -p /path/to/other-repo/.codex/skills
rsync -a /path/to/voice-ai/.codex/skills/ /path/to/other-repo/.codex/skills/
```

Install into user-global path:

```bash
mkdir -p ~/.codex/skills/voice-ai
rsync -a .codex/skills/ ~/.codex/skills/voice-ai/
```

Verify installation:

```bash
find .codex/skills -maxdepth 2 -type d | sort
```

## Skill layout

Each skill folder should include:

- `SKILL.md`: task instructions and scope
- `agents/openai.yaml`: agent metadata and default prompt
- `references/`: checklists and architecture notes
- `examples/sample.md`: expected output format
- `scripts/validate.sh`: local validator

Codex packaging intentionally uses `agents/openai.yaml` and `references/`. Semantic lifecycle parity with Claude is enforced by `make validate-agent-tooling`, not by requiring identical directory trees.

## Available skills

- `system-understanding`
- `telephony-integration`
- `stt-integration`
- `tts-integration`
- `noise-reduction-integration`
- `llm-integration`
- `telemetry-integration`
- `vad-integration`
- `end-of-speech-integration`
- `local-setup-and-run`

## How to use

1. Choose the skill folder matching the requested integration.
2. Read `SKILL.md` first, then the skill-specific `references/` checklist.
3. Implement changes only within the declared scope.
4. Validate before sharing results.

## Validation

Basic validation:

```bash
./.codex/skills/<skill>/scripts/validate.sh
```

Strict scope validation:

```bash
./.codex/skills/<skill>/scripts/validate.sh --check-diff --provider <provider>
```

Notes:
- `--provider` is required for integration skills (`telephony`, `stt`, `tts`, `llm`, `telemetry`, `vad`, `end-of-speech`, `noise-reduction`).
- strict mode enforces provider/factory/contract boundaries.
- strict mode should be run when working in an isolated/clean diff for best signal.
- `local-setup-and-run` uses `--check-diff` without `--provider`.

Every skill participates in the governed lifecycle and hands implementation evidence to an independent reviewer. Run `make validate-development-toolkit` before shipping toolkit changes.

## Security

Read and follow `.codex/skills/SECURITY_GUIDELINES.md` before editing or publishing skill changes.
