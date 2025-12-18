<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://github.com/rapidaai/voice-ai/blob/807a2b596c45326e9db5eb7051b133722e646de8/.github/banner-01.png">
  <source media="(prefers-color-scheme: light)" srcset="https://github.com/rapidaai/voice-ai/blob/807a2b596c45326e9db5eb7051b133722e646de8/.github/banner-01.png">
  <img style="width:100%;" src="https://github.com/rapidaai/voice-ai/blob/807a2b596c45326e9db5eb7051b133722e646de8/.github/banner-01.png" alt="Banner">
</picture>

# Rapida: End-to-End Voice Orchestration Platform

[Rapida](https://rapida.ai) is an open-source platform for designing, building, and deploying voice agents at scale.  
It’s built around three core principles:

- **Reliable** — designed for production workloads, real-time audio, and fault-tolerant execution  
- **Observable** — deep visibility into calls, latency, metrics, and tool usage  
- **Customizable** — flexible architecture that adapts to any LLM, workflow, or enterprise stack  

Rapida provides both a **platform** and a **framework** for building real-world voice agents—from low-latency audio streaming to orchestration, monitoring, and integrations.

Rapida is written in **Go**, using the highly optimized [gRPC](https://github.com/grpc/grpc-go) protocol for fast, efficient, bidirectional communication.

---
## Features

- **Real-time Voice Orchestration**  
  Stream and process audio with low latency using GRPC.

- **LLM-Agnostic Architecture**  
  Bring your own model—OpenAI, Anthropic, open-source models, or custom inference.

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

## Architecture Overview


    ┌──────────────────────────────────────────────────────────────┐
    │                           CHANNELS                           │
    │         Phone • Web • WhatsApp • SIP • WebRTC • Others       │
    └──────────────────────────────────┬───────────────────────────┘
                                    │
                                    ▼
    ┌─────────────────────────────────────────────────────────────┐
    │                       RAPIDA ORCHESTRATOR                   │
    │   Routing • State • Parallelism • Tools • Observability     │
    └───────────────┬──────────────────────────────┬──────────────┘
                    │                              │
                    ▼                              ▼
        ┌──────────────────────┐        ┌────────────────────────┐
        │   Audio Preprocess   │        │          STT           │
        │  • VAD               │ <----> │   Speech-to-Text       │
        │  • Noise Reduction   │        │   (ASR Engine)         │
        │  • End-of-Speech     │        └───────────┬────────────┘
        └───────────┬──────────┘                    │
                    │                               ▼
                    │                    ┌────────────────────────┐
                    │                    │           LLM          │
                    │                    │ Reasoning • Tools •    │
                    │                    │  Memory • Policies     │
                    │                    └───────────┬────────────┘
                    │                                │
                    │                                ▼
                    │                    ┌────────────────────────┐
                    └──────────────────▶ │           TTS          │
                                         │    Text-to-Speech      │
                                         └───────────┬────────────┘
                                                     │
                                                     ▼
                                    ┌────────────────────────────────────┐
                                    │              USER OUTPUT           │
                                    │         Audio Stream Response      │
                                    └────────────────────────────────────┘


## Documentation & Guides

https://doc.rapida.ai

---

## Services

| Service         | Description                          | Port           |
|------------------|--------------------------------------|----------------|
| PostgreSQL       | Database for persistent storage      | `5432`         |
| Redis            | In-memory caching                   | `6379`         |
| OpenSearch       | Search engine for document indexing | `9200`, `9600` |
| Web API          | Backend service                     | `9001`         |
| Assistant API    | Intelligence and assistance API      | `9007`         |
| Integration API  | Third-party integrations API         | `9004`         |
| Endpoint API     | Endpoint management API             | `9005`         |
| Document API     | Document handling API               | `9010`         |
| UI               | React front-end                     | `3000`         |
| NGINX            | Reverse proxy and static server     | `8080`         |

---

## Prerequisites

- **Docker**: [Install Docker](https://www.docker.com/).
- **Docker Compose**: Ensure Docker Compose is included with your Docker installation.

---

## Setup

### 1. Clone the Repository

```bash
git clone https://github.com/rapidaai/voice-ai.git
cd voice-ai
```

### 2. Create Necessary Directories

Ensure the following directories exist for containerized services to mount their data:

```bash
mkdir -p ${HOME}/rapida-data/
```
For more about how the data is structured for services https://doc.rapida.ai

### 3. Set Permissions for Docker Access

Grant `docker` group access to the created directories to ensure proper mounting:

```bash
sudo setfacl -m g:docker:rwx ${HOME}/rapida-data/
```

### 4. Build the Services

```bash
make build-all
```

### 5. Start the Services

Start all services:

```bash
make up-all
```

Alternatively, start specific services (e.g., just PostgreSQL):

```bash
make up-db
```

### 6. Stop the Services

Stop all running services:

```bash
make down-all
```

Stop specific services:

```bash
make down-web
```

---

## Accessing Services

| Service            | URL                                |
|---------------------|------------------------------------|
| UI                 | [http://localhost:3000](http://localhost:3000) |
| Web-API            | [http://localhost:9001](http://localhost:9001) |
| Assistant-API      | [http://localhost:9007](http://localhost:9007) |
| Integration-API    | [http://localhost:9004](http://localhost:9004) |
| Endpoint-API       | [http://localhost:9005](http://localhost:9005) |
| Document-API       | [http://localhost:9010](http://localhost:9010) |
| OpenSearch         | [http://localhost:9200](http://localhost:9200) |

---

## Makefile Usage

The `Makefile` simplifies operations using Docker Compose:

### Common Commands

- **Build all images:**
    ```bash
    make build-all
    ```

- **Start all services:**
    ```bash
    make up-all
    ```

- **Stop all services:**
    ```bash
    make down-all
    ```

- **Check service logs (e.g., Web API):**
    ```bash
    make logs-web
    ```

- **Restart specific services (e.g., Redis):**
    ```bash
    make restart-redis
    ```

### View All Commands

Run `make help` to see a full list of available `Makefile` commands.

---

## Notes

- Ensure to create the necessary directories (`rapida-data/assets/...`) and apply permissions before starting the services.
- Custom configurations for NGINX and other services are mounted and should be adjusted as per your requirements.

---
## SDKs & Tools

### Client SDKs

Client SDKs enable your frontend to include interactive, multi-user experiences.

| Language                | Repo                                                                                    | Docs                                                        |
| :---------------------- | :-------------------------------------------------------------------------------------- | :---------------------------------------------------------- |
| Web (React)                     | [rapida-react](https://github.com/rapidaai/rapida-react)                               | [docs](https://doc.rapida.ai/api-reference/installation) |
| Web Widget (react)                    | [react-widget](https://github.com/rapidaai/react-widget)                           |                                                             |

### Server SDKs

Server SDKs enable your backend to build and manage agents.

| Language                | Repo                                                                                    | Docs                                                        |
| :---------------------- | :-------------------------------------------------------------------------------------- | :---------------------------------------------------------- |
| Go                      | [rapida-go](https://github.com/rapidaai/rapida-go)                               | [docs](https://doc.rapida.ai/api-reference/installation) |
| Python| [rapida-python](https://github.com/rapidaai/rapida-python)                               | [docs](https://doc.rapida.ai/api-reference/installation/)              |

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

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=rapidaai/voice-ai&type=Date)](https://star-history.com/#rapidaai/voice-ai&Date)