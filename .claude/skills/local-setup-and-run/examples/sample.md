# Sample Output: Local Setup And Run

## Environment choice

- Recommended path: Docker
- Why: repository Just recipes and compose files provide first-class startup flow.

## Prerequisites

- Docker workflow prerequisites: Docker Engine + Compose plugin, 16GB RAM.
- Non-docker workflow prerequisites: Go toolchain, Yarn, PostgreSQL, Redis, optional OpenSearch/document-api.

## Docker workflow

- Setup/build: `just setup-local build-all`
- Run: `just up-all`
- Knowledge mode: `just up-all-with-knowledge`
- Stop: `just down-all`

## Non-docker workflow

- Dependencies: `just deps` (or `just deps-knowledge`)
- APIs: `just run-web`, `just run-assistant`, `just run-endpoint`, `just run-integration`
- UI: `just run-ui`

## Health checks

- UI `http://localhost:3000`
- Web API `http://localhost:9001`
- Assistant API `http://localhost:9007`
- Endpoint API `http://localhost:9005`
- Integration API `http://localhost:9004`
- Document API `http://localhost:9010` (knowledge mode)

## Troubleshooting

- Port conflict: identify with `lsof -i :<port>` and stop conflicting process.
- Service crash: inspect `just logs-all` and `docker compose ps`.

## Verification evidence

- Commands executed: `just --list`, `just up-all`, `docker compose ps`
- Output summary: services reachable on expected ports.
