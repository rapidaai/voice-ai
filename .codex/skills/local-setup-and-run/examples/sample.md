# Sample Output: Local Setup And Run

## Recommended path: Docker

- `just setup-local build-all`
- `just up-all`
- verify with `docker compose ps`

## Knowledge mode (optional)

- `just up-all-with-knowledge`

## Non-docker path

- dependencies: `just deps` (or `just deps-knowledge`)
- run APIs: `just run-web`, `just run-assistant`, `just run-endpoint`, `just run-integration`
- run UI: `just run-ui`

## Health checks

- UI: `http://localhost:3000`
- Web API: `http://localhost:9001`
- Assistant API: `http://localhost:9007`
- Endpoint API: `http://localhost:9005`
- Integration API: `http://localhost:9004`
- Document API: `http://localhost:9010` (knowledge mode)

## Troubleshooting

- Ports in use: `lsof -i :<port>`
- Service failures: `just logs-all`, `docker compose ps`
