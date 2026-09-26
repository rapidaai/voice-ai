# Codex Skills README

This directory contains Codex skills for the repository development lifecycle and voice integration work.

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

Domain skill folders include:

- `SKILL.md`: task instructions and scope
- `agents/openai.yaml`: agent metadata and default prompt
- `references/`: checklists and architecture notes
- `examples/sample.md`: expected output format
- `scripts/validate.sh`: local validator

Codex packaging intentionally uses `agents/openai.yaml` and `references/`. Semantic lifecycle parity with Claude is enforced by `just validate-agent-tooling`, not by requiring identical directory trees.

Lifecycle skill folders stay lean and include:

- `SKILL.md`: task instructions and scope
- `agents/openai.yaml`: discovery metadata and default prompt
- `examples/sample.md`: expected evidence shape

Lifecycle skills are validated centrally by `just validate-agent-tooling` so their policy checks have one owner.

## Available skills

- `development-lifecycle`
- `change-analysis`
- `designing-change`
- `debugging`
- `developing-change`
- `reviewing-change`
- `responding-to-review`
- `writing-documentation`
- `preparing-delivery`
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

1. Start with `development-lifecycle` for feature, fix, or behavior-change work.
2. Use the lifecycle skill it selects for analysis, debugging, implementation, review, or delivery.
3. Add `system-understanding` and the matching domain skill for voice integrations.
4. Implement changes only within the declared scope and validate before sharing results.

## Validation

Domain skill validation:

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

Every skill participates in the governed lifecycle and hands implementation evidence to an independent reviewer. Run `just validate-development-toolkit` before shipping toolkit changes.

## Security

Read and follow `.codex/skills/SECURITY_GUIDELINES.md` before editing or publishing skill changes.
