<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/rapidaai/voice-ai/main/.github/banner-02.jpg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/rapidaai/voice-ai/main/.github/banner-02.jpg">
  <img style="width:100%;" src="https://raw.githubusercontent.com/rapidaai/voice-ai/main/.github/banner-02.jpg" alt="Banner">
</picture>

# Rapida: End-to-End Voice Orchestration Platform

[Rapida](https://rapida.ai) is an open-source platform for designing, building, and deploying voice agents at scale.  
It’s built around three core principles:

- **Reliable** — designed for production workloads, real-time audio, and fault-tolerant execution
- **Observable** — deep visibility into calls, latency, metrics, and tool usage
- **Customizable** — flexible architecture that adapts to any LLM, workflow, or enterprise stack

Rapida provides both a **platform** and a **framework** for building real-world voice agents—from low-latency audio streaming to orchestration, monitoring, and integrations.

Rapida is written in **Go**, using the highly optimized [gRPC](https://github.com/grpc/grpc-go) protocol for fast, efficient, bidirectional communication.

[![GitHub stars](https://img.shields.io/github/stars/rapidaai/voice-ai?style=social&label=Star&maxAge=2592000)](https://github.com/rapidaai/voice-ai/stargazers/)
[![Twitter Follow](https://img.shields.io/twitter/follow/rapidaai)](https://twitter.com/rapidaai)
[![Discord](https://img.shields.io/badge/Discord-Join%20Us-5865F2?logo=discord&logoColor=white)](https://discord.gg/qKNzjBkh)
[![Book a Meeting](https://img.shields.io/badge/Book%20a%20Meeting-Cal.com-blue)](https://cal.com/prashant-srivastav-u8duzh/30min)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/rapidaai/voice-ai)
[![CodeQL](https://github.com/rapidaai/voice-ai/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/rapidaai/voice-ai/actions/workflows/github-code-scanning/codeql)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/rapidaai/voice-ai)](https://github.com/rapidaai/voice-ai/releases/latest)

---

## Architecture

![Rapida Platform Architecture](.github/rapida-platform-architecture.svg)

---

## Features

- **Real-time Voice Orchestration**  
  Stream and process audio with low latency using GRPC.

- **LLM-Agnostic Architecture**
  Bring your own model—OpenAI, Anthropic, MiniMax, open-source models, or custom inference.

- **Production-grade Reliability**  
  Built-in retries, error handling, call lifecycle management, and health checks.

- **Full Observability**  
  Call logs, streaming events, tool traces, latency breakdowns, metrics, and dashboards.

- **Flexible Tooling System**  
  Build custom tools and actions for your agents, or integrate with any backend.

- **Developer-friendly**  
  Clear APIs, modular components, and simple configuration.

- **Enterprise-ready**  
  Scalable design, efficient protocol, and predictable performance.

## Documentation & Guides

https://doc.rapida.ai

## Prerequisites

- **Docker** & **Docker Compose** ([Install](https://www.docker.com/))
- **16GB+ RAM** (for all services)

---

## Quick Start

Get all services running in 4 commands:

```bash
# Clone repo
git clone https://github.com/rapidaai/voice-ai.git && cd voice-ai

# Setup & build
make setup-local && make build-all

# Start all services
make up-all

# View running services
docker compose ps
```

**Services Ready:**

- UI: http://localhost:3000
- Web API: http://localhost:9001
- Assistant API: http://localhost:9007
- Endpoint API: http://localhost:9005
- Integration API: http://localhost:9004
- Document API: http://localhost:9010

**Stop services:**

```bash
make down-all
```

---

## Development

### Work on Specific Services

```bash
# Start only database
make up-db

# Start only UI
make up-ui

# Start only Assistant API
make up-assistant

# List all start commands
make help
```

### View Logs

```bash
# All services
make logs-all

# Specific service
make logs-web
make logs-assistant
```

### Rebuild After Code Changes

```bash
# Rebuild and restart one service
make rebuild-assistant

# Rebuild all
make rebuild-all
```

### Configure Services

Edit environment files before starting:

- `docker/web-api/.web.env` - Web API (port 9001)
- `docker/assistant-api/.assistant.env` - Assistant API (port 9007)
- `docker/endpoint-api/.endpoint.env` - Endpoint API (port 9005)
- `docker/integration-api/.integration.env` - Integration API (port 9004)
- `docker/document-api/config.yaml` - Document API (port 9010)

Add your API keys (OpenAI, Anthropic, MiniMax, Deepgram, Twilio, etc.) in these files.

---

## Local Development (Without Docker)

### Go Services

```bash
# Install dependencies
go mod download

# Build service
go build -o bin/web ./cmd/web

# Run service
./bin/web
```

Requires PostgreSQL, Redis, OpenSearch running separately.

### React UI

```bash
cd ui

# Install & run
yarn install
yarn start:dev

# Build for production
yarn build
```

---

## Troubleshooting

**Port already in use:**

```bash
lsof -i :3000    # Find process
kill -9 <PID>    # Kill it
```

**Services won't start:**

```bash
make logs-all    # Check logs
docker compose ps  # Verify status
```

**Database issues:**

```bash
# Test connection
docker compose exec postgres psql -U rapida -d web_db -c "SELECT 1"

# Reset everything
make clean
make setup-local
make build-all
make up-all
```

---

## All Commands

```bash
make help          # Show all available commands
make setup-local   # Create data directories
make build-all     # Build all Docker images
make up-all        # Start all services
make down-all      # Stop all services
make logs-all      # View all logs
make clean         # Remove containers & volumes
make restart-all   # Restart all services
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Want to add:

- New STT/TTS provider? Check `api/assistant-api/internal/transformer/`
- New telephony channel? Check `api/assistant-api/internal/telephony/`

---

## SDKs & Tools

### Client SDKs

Client SDKs enable your frontend to include interactive, multi-user experiences.

| Language           | Repo                                                     | Docs                                                     |
| :----------------- | :------------------------------------------------------- | :------------------------------------------------------- |
| Web (React)        | [rapida-react](https://github.com/rapidaai/rapida-react) | [docs](https://doc.rapida.ai/api-reference/installation) |
| Web Widget (react) | [react-widget](https://github.com/rapidaai/react-widget) |                                                          |

### Server SDKs

Server SDKs enable your backend to build and manage agents.

| Language | Repo                                                       | Docs                                                      |
| :------- | :--------------------------------------------------------- | :-------------------------------------------------------- |
| Go       | [rapida-go](https://github.com/rapidaai/rapida-go)         | [docs](https://doc.rapida.ai/api-reference/installation)  |
| Python   | [rapida-python](https://github.com/rapidaai/rapida-python) | [docs](https://doc.rapida.ai/api-reference/installation/) |

## Contributing

For those who'd like to contribute code, see our [Contribution Guide](https://github.com/rapidaai/voice-ai/blob/main/CONTRIBUTING.md).
At the same time, please consider supporting RapidaAi by sharing it on social media and at events and conferences.

## Security disclosure

To protect your privacy, please avoid posting security issues on GitHub. Instead, report issues to contact@rapida.ai, and our team will respond with detailed answer.

## License

Rapida is open-source under the GPL-2.0 license, with additional conditions:

- Open-source users must keep the Rapida logo visible in UI components.
- Future license terms may change; this does not affect released versions.

A commercial license is available for enterprise use, which allows:

- Removal of branding
- Closed-source usage
- Private modifications
  Contact sales@rapida.ai for details.
