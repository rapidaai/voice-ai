# Claude Skills Setup

This repository includes Claude skills in `.claude/skills/`.

## Quick start (this repo)

Prerequisites:
- `python3`
- Python package `jsonschema`
- `go`
- `yarn` (for UI-related checks)
- `just`

Validate local setup:

```bash
find .claude/skills -maxdepth 2 -type d | sort
just validate-development-process
just validate-agent-tooling
just agent-finalize "api/example/service.go,api/example/service_test.go" .claude
```

## Install in this repository

```bash
find .claude/skills -maxdepth 2 -type d | sort
```

## Install in another repository

Copy the full `.claude` folder so hooks and subagents are included:

```bash
mkdir -p /path/to/target-repo/.claude
rsync -a /path/to/voice-ai/.claude/ /path/to/target-repo/.claude/
```

## Required skill structure

Each skill should contain:

- `SKILL.md`
- `template.md`
- `examples/sample.md`
- `scripts/validate.sh`

## Validate a skill

```bash
./.claude/skills/<skill>/scripts/validate.sh
./.claude/skills/<skill>/scripts/validate.sh --check-diff --provider <provider>
```

For integration skills (`stt`, `tts`, `telephony`, `llm`, `telemetry`, `vad`, `end-of-speech`, `noise-reduction`), include `--provider` in strict mode.

## Orchestrator Hooks

Lifecycle hook contracts, templates, and the runner are available at `.claude/orchestrator/`:

```bash
DEVELOPMENT_GATE_KEY="<coordinator-key>" python3 .claude/orchestrator/scripts/hook-run.py --stage pre-implementation --input .claude/orchestrator/examples/lifecycle-input.json --output /tmp/hook-out.json
```

Governed lifecycle:

`understand -> plan -> discuss -> approve -> implement -> verify -> independent code review -> ship`

The final `post-review` gate rejects self-review and unresolved critical or major findings. See `DEVELOPMENT_PROCESS.md`.

Render or open the Orca development panel:

```bash
just orca-panel "path/to/lifecycle-input.json"
just orca-panel-open "path/to/lifecycle-input.json"
```

Claude validation configuration is committed in:

- `.claude/settings.json` (automatic completion hooks intentionally disabled)
- `.claude/hooks/` (explicit validation commands)
- `.claude/agents/` (subagents for UI/backend implementation and tests)

Use `just validate-development-toolkit` to validate lifecycle gates, skill packaging, agent role contracts, non-blocking Claude settings, hook parity, and scoped validation together.

## References

- `.claude/skills/README.md`
- `.claude/skills/ENTERPRISE_POLICY.md`
- `.claude/skills/SECURITY_GUIDELINES.md`
