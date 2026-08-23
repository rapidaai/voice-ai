# `/bins`

Scripts to perform various build, install, analysis, etc operations.
These scripts keep the root level Makefile small and simple.

Development governance helpers:

- `agent-finalize` runs explicit, scoped test-presence and targeted-test checks once.
- `orca-development-run` reserves an RFC path and creates the planning/RFC/challenge DAG.
- `orca-rfc-release` removes an abandoned empty RFC reservation lock.
- `orca-confirm-rfc` creates or collects the exact-digest RFC confirmation gate.
- `validate-rfc-layout` enforces the canonical RFC template and `rfcs/<rfc-stem>/jsons/` artifact layout.
- `sign-approved-plan` attests an approved plan for lifecycle hooks.

- artifacts-generate.sh Generating artifacts from protos and OpenAPI specs
- git-commit-hook-setup.sh Setup pre-commit, commit-msg, and pre-push hooks.
- check-go-version-consistency Verifies Go, CI, and Docker base-image versions remain aligned.
- pre-commit-go Go formatting, tidy, lint, vet, test, and build checks used by pre-commit.
- pre-commit-hygiene Lightweight repository hygiene checks used by pre-commit.
- go-fmt.sh Legacy formatting check.
