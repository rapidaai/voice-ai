# Claude Skills Index

Project skills live in `.claude/skills/<skill-name>/`.

## Install

For this repository, skills are already installed at `.claude/skills/`.

To install these skills into another repository:

```bash
mkdir -p /path/to/other-repo/.claude/skills
rsync -a /path/to/voice-ai/.claude/skills/ /path/to/other-repo/.claude/skills/
```

To verify installation:

```bash
find .claude/skills -maxdepth 2 -type d | sort
```

Each skill contains:

- `SKILL.md` (required instructions + metadata)
- `template.md` (structured implementation output)
- `examples/sample.md` (expected output shape)
- `scripts/validate.sh` (local skill self-check)

Claude packaging intentionally uses `template.md` and enterprise policy files. Semantic lifecycle parity with Codex is enforced by `make validate-agent-tooling`, not by requiring identical directory trees.

## Enterprise standards

- Follow `.claude/skills/ENTERPRISE_POLICY.md` for risk review and approval checks.
- Follow `.claude/skills/SECURITY_GUIDELINES.md` for public-safe contribution rules.
- Keep `SKILL.md` scope strict and tied to real factory/provider paths.
- Keep transport and packet contracts explicit for integration skills.

## Validation usage

Basic structure checks:

```bash
./.claude/skills/<skill>/scripts/validate.sh
```

Strict diff scope checks:

```bash
./.claude/skills/<skill>/scripts/validate.sh --check-diff --provider <provider>
```

Notes:
- `--provider` is required for integration skills (`telephony`, `stt`, `tts`, `llm`, `telemetry`, `vad`, `end-of-speech`, `noise-reduction`).
- `local-setup-and-run` uses `--check-diff` without `--provider`.
- strict mode allows only skill files, factory files, contract files, and the specified provider folder scope.
- EOS strict mode rejects edits under VAD internals; VAD strict mode rejects edits under EOS internals.

Every skill participates in the governed lifecycle and hands implementation evidence to an independent reviewer. Run `make validate-development-toolkit` before shipping toolkit changes.

## Skill list

- system-understanding
- telephony-integration
- stt-integration
- tts-integration
- llm-integration
- telemetry-integration
- vad-integration
- end-of-speech-integration
- noise-reduction-integration
- local-setup-and-run
