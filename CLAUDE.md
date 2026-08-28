# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rapida is an open-source, Go-based microservices platform for voice AI orchestration (module: `github.com/rapidaai`, Go 1.25). It uses gRPC for inter-service communication and Gin for REST APIs.

## Commands

### Docker (primary development workflow)

```bash
just setup-local        # Create data directories (run once)
just build-all          # Build all Docker images
just up-all             # Start all services
just down-all           # Stop all services
just logs-all           # Tail logs for all services
just logs-assistant     # Tail logs for a specific service
just rebuild-assistant  # Rebuild and restart a specific service (no cache)
just status             # Show container status and ports
just clean              # Remove containers, volumes, and images
just deps               # Start only infrastructure (postgres and redis)
```

### Local (without Docker)

```bash
go run cmd/web/web.go                              # Run web-api
go run cmd/assistant/assistant.go                  # Run assistant-api
go run cmd/endpoint/endpoint.go                    # Run endpoint-api
go run cmd/integration/integration.go              # Run integration-api
PYTHONPATH=api/document-api uvicorn app.main:app --host 0.0.0.0 --port 9010  # Run document-api

cd ui && yarn start:dev   # Run UI (concurrent Tailwind watch + Craco dev server)
```

Requires PostgreSQL, Redis, and OpenSearch running separately when not using Docker.

### Testing

```bash
go test ./...                     # Run all Go tests
go test ./api/web-api/...         # Run tests for a specific service
go test -run TestName ./path/...  # Run a single test
pytest api/document-api/tests/    # Run Python tests
cd ui && yarn test                # Run UI tests
cd ui && yarn test:coverage       # Run UI tests with coverage
```

### Linting & Formatting

```bash
golangci-lint run          # Run Go linter (config in .golangci.yml)
cd ui && yarn lint         # Lint UI
cd ui && yarn lint:fix     # Lint and auto-fix UI
cd ui && yarn checkTs      # TypeScript type check
```

### Protocol Buffers

Proto sources live in `protos/artifacts/` (git submodule); generated files are in `protos/`. Use `buf` (config in `buf.yaml`) to regenerate after editing `.proto` files.

## Architecture

### Services

| Service             | Port             | Language | Role |
| ------------------- | ---------------- | -------- | ---- |
| **web-api**         | 9001             | Go       | Auth, OAuth, organizations, projects, vaults, proxy |
| **assistant-api**   | 9007, 4573 (SIP) | Go       | Voice orchestration, STT/TTS, telephony, knowledge |
| **integration-api** | 9004             | Go       | LLM provider integrations (OpenAI, Anthropic, Gemini, etc.) |
| **endpoint-api**    | 9005             | Go       | Endpoint management, invocation, caching, retry |
| **document-api**    | 9010             | Python   | Document processing, chunking, embedding, RAG (FastAPI + Celery) |
| **ui**              | 3000             | React/TS | Frontend dashboard |

### Infrastructure

- **PostgreSQL 15** (5432) — relational data with GORM ORM
- **Redis 7** (6379) — caching, second-level GORM cache, pub/sub
- **OpenSearch 2.11** (9200) — knowledge search with tenant-scoped indices
- **Nginx** (8080) — reverse proxy to all services + UI

### Communication

- **REST** via Gin (`/v1` prefix, external-facing)
- **gRPC** via protobuf (inter-service, 10MB max message size)
- **cmux** multiplexes HTTP/2 (gRPC), grpc-web, and HTTP on a single port per service
- **WebRTC** + **SIP** (port 4573) for real-time voice in assistant-api

### Environment Configuration

Each service reads from `env/config.yaml` + environment variables via Viper. Docker env files are at `docker/<service>/.<service>.env`. The global `AppConfig` struct in `config/config.go` holds shared fields; each service embeds it in a service-specific config struct.

## Key Patterns

### Go Service Bootstrap

All Go services follow an identical `AppRunner` struct in `cmd/*/`. Bootstrap sequence:
1. `ResolveConfig()` — Viper + `validator/v10`
2. `Logging()` — Zap SugaredLogger via `commons.NewApplicationLogger`
3. `AllConnectors()` — Postgres/Redis/OpenSearch
4. `Migrate()` — `golang-migrate/v4` with SQL files from `api/<service>/migrations/`
5. `grpc.NewServer()` — chained interceptors (logging, recovery, auth, project auth, service auth)
6. `AllRouters()` — registers REST routes on Gin + gRPC service implementations
7. cmux multiplexing — all protocols on single TCP port
8. Graceful shutdown via `os.Signal` + `Closeable` cleanup hooks

### Handler Structure

Each domain area has:
- Interface defined in `api/<service>/internal/service/<domain>.service.go`
- Implementation in `api/<service>/internal/service/<domain>/service.go` (unexported struct)
- REST handler struct + gRPC handler struct (gRPC embeds the base struct)
- Constructor returns the protobuf server interface

### Entity Pattern

Entities compose base GORM models: `Audited` (ID + timestamps), `Mutable` (status + created/updated by), `Organizational` (project + org scoping). IDs are auto-generated via Snowflake in `BeforeCreate` hooks.

### STT/TTS Transformer Pattern

Each provider under `api/assistant-api/internal/transformer/` implements `Transformers[AudioPacket]` (STT) and `Transformers[TextPacket]` (TTS) interfaces from `api/assistant-api/internal/type/`. Files follow the pattern: `<provider>.go`, `stt.go`, `tts.go`, `normalizer.go`.

### Inter-service Calls

Use typed gRPC clients from `pkg/clients/` for service-to-service calls (never direct HTTP). The `Communication` interface in `api/assistant-api/internal/type/` is the core voice pipeline contract tying together callbacks, LLM execution, auth, tracing, and conversation state.

## Conventions

- **Naming**: File naming follows `<domain>.<qualifier>.<entity>.go` (e.g., `knowledge.assistant.service.go`)
- **Migrations**: Sequential SQL files (`000001_initial_schema.up.sql`), one set per service
- **Import groups**: stdlib, external, then `github.com/rapidaai` (enforced by `goimports` in golangci-lint)
- **Linter exclusions**: `protos/` and `sdks/` are excluded from linting; test files are excluded from `dupl`, `gosec`, `gocyclo`, `gocognit`
- **Frontend state**: Custom hooks (`use-*-page-store.ts`) — no Redux
- **Provider metadata**: JSON files under `ui/src/providers/` define voices, languages, and models per AI provider

## Developer Docs

Detailed step-by-step guides for common development tasks live in `.claude/developer/`, `claude-skills/`, and `.claude/skills/`. Codex skill bundles live in `skills/`. Always check these before starting work on a new integration:

| Guide | File | Covers |
|-------|------|--------|
| Adding STT/TTS Provider | `claude-skills/adding-stt-tts-provider.md` | Full frontend-to-backend walkthrough: provider registry, UI config components, backend transformer, factory wiring, required events/metrics, checklist |
| Adding STT Provider | `claude-skills/adding-stt-provider.md` | Focused STT integration guide: provider metadata, STT option mapping (`listen.*`), transformer wiring, and validation checklist |
| Adding TTS Provider | `claude-skills/adding-tts-provider.md` | Focused TTS integration guide: voice/model metadata, `speak.*` mapping, transformer wiring, and output audio checks |
| Adding LLM Provider | `claude-skills/adding-llm-provider.md` | Full frontend-to-backend walkthrough: provider registry, model constants, UI config component, backend caller, gRPC API handler, router + integration client wiring, metrics, checklist |
| Adding Telephony Provider | `claude-skills/adding-telephony-provider.md` | Full frontend-to-backend walkthrough: provider registry, UI config component, WebSocket streamer, telephony webhook handler, factory wiring, audio format/resampling, required events, checklist |
| Adding Telemetry | `claude-skills/adding-telemetry-instrumentation.md` | Instrumentation guide for metrics/event additions, audit post-hook compatibility, and failure-path observability checks |
| Adding VAD Provider | `claude-skills/adding-vad-provider.md` | VAD provider integration guide: factory registration, `microphone.vad.*` options, callback behavior, and tests |
| Adding End-of-Speech | `claude-skills/adding-end-of-speech-provider.md` | EOS configuration guide: `microphone.eos.*` schema, VAD interplay, timeout tuning, and compatibility checks |
| Understanding System | `claude-skills/understanding-system.md` | Fast architecture ramp-up: service map, execution-path tracing, extension-point checklist, risk mapping, pre-coding test plan |

## Adding New Integrations

- **New STT/TTS provider**: Follow `claude-skills/adding-stt-tts-provider.md` for the complete guide
- **New LLM provider**: Follow `claude-skills/adding-llm-provider.md` for the complete guide
- **New telephony provider**: Follow `claude-skills/adding-telephony-provider.md` for the complete guide
- **Telemetry changes**: Follow `claude-skills/adding-telemetry-instrumentation.md`
- **VAD provider changes**: Follow `claude-skills/adding-vad-provider.md`
- **End-of-speech changes**: Follow `claude-skills/adding-end-of-speech-provider.md`
- **New telephony channel**: Add under `api/assistant-api/internal/channel/`
- **New OAuth connector**: Add under `api/web-api/internal/connect/`
