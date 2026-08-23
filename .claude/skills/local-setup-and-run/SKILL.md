---
name: local-setup-and-run
description: Explain and validate local setup paths for this repo with Docker and without Docker. Use when developers need exact prerequisites, startup commands, health checks, and troubleshooting steps.
---

# Local Setup And Run Skill

## Mission

Provide accurate, repo-grounded local run instructions for both Docker and non-Docker workflows.

## Scope (strict)

In scope:
- setup/run documentation and runbook updates
- command verification based on `README.md`, `Makefile`, `docker-compose.yml`, and `docker-compose.knowledge.yml`
- prerequisites and dependency matrix guidance

Out of scope:
- modifying runtime feature code for APIs/UI
- changing provider integrations unrelated to setup

## Source of truth files

- `README.md`
- `Makefile`
- `docker-compose.yml`
- `docker-compose.knowledge.yml`
- env templates under `docker/*/*.env` (when present)

## Required output sections

1. Prerequisites (Docker path + non-Docker path).
2. Docker quick start commands (recommended path).
3. Optional knowledge-stack startup path.
4. Non-Docker run path (`deps` + `run-*` commands).
5. Health-check URLs/ports.
6. Common failure modes and fixes.

## Defaults to minimize interruptions

- Prefer Docker flow as default recommendation.
- Use Makefile targets when available.
- If non-Docker dependencies are missing, call them out explicitly (PostgreSQL/Redis/OpenSearch).

## Governed lifecycle

- Classify work as Fast, Standard, or Governed using `DEVELOPMENT_PROCESS.md`; use the full gated lifecycle only for Governed work.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Governed repository edits start only from a coordinator-attested approved plan; Fast and Standard work follows its lighter documented lifecycle.
- Return documentation and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `make help`
- `make up-all`
- `make up-all-with-knowledge`
- `make deps && make run-web` (example non-Docker path)
- `docker compose ps`
