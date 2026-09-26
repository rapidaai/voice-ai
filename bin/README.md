# `/bins`

Scripts to perform various build, install, analysis, etc operations.
These scripts back the modular recipes under `just/`.

Development governance helpers:

- `agent-finalize` runs explicit, scoped test-presence and targeted-test checks once.
- `agent-egress` allowlists repository paths and redacts secret-looking content before external transfer.
- `agent-pr-ready` runs branch policy and scoped verification before pull request publication.
- `agent-review` runs configured independent reviewers and writes one validated decision panel.
- `check-agent-manifest` validates capability phase status and coverage contracts.
- `check-agent-docs` validates agent documentation links, repository paths, Just recipes, and skill-name parity.
- `check-agent-drift` verifies generated path rules and paired agent surfaces.
- `check-agent-style` validates added prose, comments, TODO ownership, and identifier terminology.
- `check-pr-body` validates pull request titles and required template sections.
- `render-agent-rules` renders nested agent instructions from `agent-rules.json`.
- `install-agent-tooling` configures and validates a fresh repository checkout.
- `orca-development-run` reserves an RFC path and creates the planning/RFC/challenge DAG.
- `orca-rfc-release` removes an abandoned empty RFC reservation lock.
- `orca-confirm-rfc` creates or collects the exact-digest RFC confirmation gate.
- `validate-rfc-layout` enforces the canonical RFC template and `rfcs/<rfc-stem>/jsons/` artifact layout.
- `sign-approved-plan` attests an approved plan for lifecycle hooks.
- `.claude/hooks/bash_guard.py` blocks destructive shell, Git, infrastructure, and release commands before Claude executes them.

- artifacts-generate.sh Generating artifacts from protos and OpenAPI specs
- git-commit-hook-setup.sh Configures versioned pre-commit, commit-msg, and pre-push hooks.
- check-go-version-consistency Verifies Go, CI, and Docker base-image versions remain aligned.
- pre-commit-go Go formatting, tidy, lint, vet, test, and build checks used by pre-commit.
- pre-commit-hygiene Lightweight repository hygiene checks used by pre-commit.
- go-fmt.sh Legacy formatting check.
