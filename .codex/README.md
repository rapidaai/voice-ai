# Codex Skills Setup

This repository keeps Codex skills at `.codex/skills/`.

## Quick start (this repo)

Prerequisites:
- `python3`
- Python package `jsonschema`
- `go`
- `yarn` (for UI-related checks)
- `just`

Validate local setup:

```bash
find .codex/skills -maxdepth 2 -type d | sort
just validate-development-process
just validate-agent-tooling
just agent-finalize "api/example/service.go,api/example/service_test.go"
```

## Install in another repository

Copy the full `.codex` folder (skills + hooks + agents + orchestrator):

```bash
mkdir -p /path/to/other-repo/.codex
rsync -a /path/to/voice-ai/.codex/ /path/to/other-repo/.codex/
```

## Install globally (optional)

Install only skills globally for all repos on this machine:

```bash
mkdir -p ~/.codex/skills/voice-ai
rsync -a .codex/skills/ ~/.codex/skills/voice-ai/
```

## Validate a skill

```bash
./.codex/skills/<skill>/scripts/validate.sh
./.codex/skills/<skill>/scripts/validate.sh --check-diff --provider <provider>
```

For integration skills (`stt`, `tts`, `telephony`, `llm`, `telemetry`, `vad`, `end-of-speech`, `noise-reduction`), include `--provider` in strict mode.

## Orchestrator Hooks

Lifecycle hook contracts, templates, and the runner are available at `.codex/orchestrator/`:

```bash
DEVELOPMENT_GATE_KEY="<coordinator-key>" python3 .codex/orchestrator/scripts/hook-run.py --stage pre-implementation --input .codex/orchestrator/examples/lifecycle-input.json --output /tmp/hook-out.json
```

Governed lifecycle:

`understand -> plan -> discuss -> approve -> implement -> verify -> independent code review -> ship`

The final `post-review` gate rejects self-review and unresolved critical or major findings. See `DEVELOPMENT_PROCESS.md`.

Render or open the Orca development panel:

```bash
just orca-panel "path/to/lifecycle-input.json"
just orca-panel-open "path/to/lifecycle-input.json"
```

Parity assets for subagent/hook workflow are available in:

- `.codex/agents/`
- `.codex/hooks/`

Use `just validate-development-toolkit` to validate lifecycle gates, skill packaging, agent role contracts, non-blocking Claude settings, hook parity, and scoped validation together.

Codex-standard repo guidance is defined in root `AGENTS.md`.
Custom project subagent profiles are defined in `.codex/agents/*.md`.

## References

- `.codex/skills/README.md`
- `.codex/skills/SECURITY_GUIDELINES.md`
