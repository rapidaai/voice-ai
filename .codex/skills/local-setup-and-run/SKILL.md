---
name: local-setup-and-run
description: Explain and validate local setup paths for this repository with Docker and without Docker. Use when developers need exact prerequisites, startup commands, health checks, and troubleshooting.
---

# Local Setup And Run Skill

## Mission

Provide accurate local setup/run instructions based on this repository's actual build and run scripts.

## Scope

In scope:
- local run documentation and runbook changes
- command mapping from `README.md`, `Makefile`, and compose files
- prerequisites and troubleshooting guidance

Out of scope:
- application feature implementation changes
- provider integration behavior changes unrelated to setup

## Source-of-truth files

- `README.md`
- `Makefile`
- `docker-compose.yml`
- `docker-compose.knowledge.yml`

## Required output

1. Docker path (recommended) with commands.
2. Docker knowledge mode path.
3. Non-Docker path with dependencies and `run-*` commands.
4. Health-check URLs and expected ports.
5. Troubleshooting section.

## Governed lifecycle

- Non-trivial work follows `understand -> plan -> challenge -> approve -> implement -> verify -> independent review -> ship` from `DEVELOPMENT_PROCESS.md`.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Repository edits start only from a coordinator-attested approved plan with explicit allowed paths, owners, commands, and rollback.
- Return documentation and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `make help`
- `make up-all`
- `make up-all-with-knowledge`
- `make deps`
- `docker compose ps`

## References

- `references/checklist.md`
- `examples/sample.md`
