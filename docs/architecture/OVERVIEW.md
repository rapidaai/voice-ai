# Architecture Overview

Understand how Rapida works before contributing.

---

## High-Level Architecture

```
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
```

---

## Services

Rapida runs as multiple microservices, each with a specific responsibility.

| Service | Port | Responsibility |
|---------|------|----------------|
| **Web API** | 9001 | Authentication, users, organizations, projects |
| **Assistant API** | 9007 | Voice orchestration, STT/TTS/LLM processing, real-time streaming |
| **Integration API** | 9004 | External integrations (Twilio, Vonage, webhooks) |
| **Endpoint API** | 9005 | LLM endpoint management, versioning |
| **Document API** | 9010 | Knowledge base processing, document chunking, embeddings |
| **UI** | 3000 | React frontend |
| **NGINX** | 8080 | Reverse proxy, gRPC-Web translation |
| **PostgreSQL** | 5432 | Persistent data storage |
| **Redis** | 6379 | Caching, session state |
| **OpenSearch** | 9200 | Knowledge base search, document indexing |

### Service Dependencies

```
                    ┌─────────────────────┐
                    │        UI           │
                    │   (React, :3000)    │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │       NGINX         │
                    │      (:8080)        │
                    └──────────┬──────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
        ▼                      ▼                      ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   Web API     │    │ Assistant API │    │Integration API│
│   (:9001)     │    │   (:9007)     │    │   (:9004)     │
└───────┬───────┘    └───────┬───────┘    └───────┬───────┘
        │                    │                    │
        └──────────┬─────────┴────────────────────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   PostgreSQL    Redis    OpenSearch
    (:5432)     (:6379)    (:9200)
```

---

## Directory Structure

```
voice-ai/
├── cmd/                        # Application entrypoints
│   ├── assistant/              # Assistant API main
│   ├── web/                    # Web API main
│   ├── endpoint/               # Endpoint API main
│   └── integration/            # Integration API main
│
├── api/                        # Service implementations
│   ├── assistant-api/
│   │   ├── api/                # gRPC handlers
│   │   ├── internal/
│   │   │   ├── transformer/    # STT/TTS/LLM providers ← Key for contributors
│   │   │   ├── telephony/      # Phone channel integrations
│   │   │   └── ...
│   │   └── router/             # HTTP routes
│   │
│   ├── web-api/                # Auth, users, projects
│   ├── integration-api/        # External integrations
│   ├── endpoint-api/           # LLM endpoint management
│   └── document-api/           # Python service for document processing
│
├── pkg/                        # Shared packages
│   ├── authenticators/         # Auth middleware
│   ├── clients/                # Internal service clients
│   ├── connectors/             # Database connections
│   ├── models/                 # Shared data models
│   └── utils/                  # Utilities
│
├── protos/                     # gRPC protocol definitions
│   ├── assistant-api.pb.go
│   ├── web-api.pb.go
│   └── ...
│
├── ui/                         # React frontend
│   └── src/
│       ├── app/
│       │   ├── pages/          # Page components
│       │   └── components/     # Reusable components
│       └── hooks/              # React hooks
│
├── docker/                     # Dockerfiles for each service
├── docker-compose.yml          # Local development setup
└── Makefile                    # Build and run commands
```

---

## Key Abstractions

### Transformers

Transformers are the abstraction for STT, TTS, and LLM providers.

**Location:** `api/assistant-api/internal/transformer/`

```
transformer/
├── stt.go                      # Speech-to-Text interface
├── tts.go                      # Text-to-Speech interface
├── llm.go                      # Language model interface
├── deepgram/
│   ├── stt.go                  # Deepgram STT implementation
│   └── tts.go                  # Deepgram TTS implementation
├── openai/
│   ├── stt.go                  # OpenAI Whisper implementation
│   ├── tts.go                  # OpenAI TTS implementation
│   └── llm.go                  # OpenAI GPT implementation
└── ...
```

To add a new provider:
1. Implement the relevant interface (`SpeechToTextTransformer`, `TextToSpeechTransformer`, etc.)
2. Register it in the factory
3. Add UI configuration in `ui/src/app/components/providers/`

### Channels

Channels handle communication with external telephony providers.

**Location:** `api/assistant-api/internal/telephony/`

Examples: Twilio, Vonage, Exotel, WebRTC

### Tools

Tools are actions the LLM can invoke during a conversation (API calls, database lookups, etc.).

**Location:** `api/assistant-api/internal/tools/`

---

## Data Flow: Voice Conversation

1. **User speaks** → Audio stream arrives via WebSocket
2. **VAD (Voice Activity Detection)** → Detect speech vs silence
3. **STT** → Transcribe audio to text
4. **LLM** → Generate response (may invoke tools)
5. **TTS** → Convert response to audio
6. **Stream back** → Audio sent to user in real-time

```
User Audio ──▶ WebSocket ──▶ VAD ──▶ STT ──▶ LLM ──▶ TTS ──▶ WebSocket ──▶ User
                              │              │
                              │              ├── Tool calls (optional)
                              │              └── Knowledge retrieval (optional)
                              │
                              └── End-of-speech detection
```

---

## Communication Protocols

| Layer | Protocol |
|-------|----------|
| UI ↔ Backend | gRPC-Web (via NGINX proxy) |
| Service ↔ Service | gRPC |
| Real-time voice | WebSocket + gRPC streaming |
| External APIs | REST/HTTP |

### gRPC Definitions

All service contracts are defined in `.proto` files and compiled to Go.

**Location:** `protos/`

Example:
- `assistant-api.proto` → `assistant-api.pb.go`, `assistant-api_grpc.pb.go`

---

## Database Schema

Each service has its own database:

| Service | Database |
|---------|----------|
| Web API | `web_db` |
| Assistant API | `assistant_db` |
| Integration API | `integration_db` |
| Endpoint API | `endpoint_db` |

Migrations are in each service's `migrations/` folder:
- `api/assistant-api/migrations/*.sql`
- `api/web-api/migrations/*.sql`
- etc.

---

## Configuration

### Environment Variables

Services are configured via environment variables. In Docker, these come from `.env` files:
- `docker/web-api/.web.env`
- `docker/assistant-api/.assistant.env`
- etc.

### Config Files

Some services also use YAML config:
- `env/config.yaml` (Document API)
- `docker/document-api/config.yaml`

---

## Local Development

### With Docker (recommended)

```bash
make build-all
make up-all
```

### Without Docker

Start dependencies first:
```bash
make up-db up-redis up-opensearch
```

Run services individually:
```bash
make run-web          # Web API (Go)
make run-assistant    # Assistant API (Go)
make run-document     # Document API (Python)
make run-ui           # Frontend (React)
```

Requires:
- Go 1.21+
- Node.js 18+
- Python 3.11+
- PostgreSQL, Redis, OpenSearch running

---

## Contributing a New Provider

To add a new STT, TTS, or LLM provider:

### 1. Implement the Interface

Look at existing implementations:
- STT: `api/assistant-api/internal/transformer/deepgram/stt.go`
- TTS: `api/assistant-api/internal/transformer/deepgram/tts.go`
- LLM: `api/assistant-api/internal/transformer/openai/llm.go`

### 2. Register in Factory

Add your provider to the factory that instantiates transformers.

### 3. Add UI Configuration

Create configuration components in:
- `ui/src/app/components/providers/speech-to-text/` (for STT)
- `ui/src/app/components/providers/text-to-speech/` (for TTS)
- `ui/src/app/components/providers/text/` (for LLM)

### 4. Test

- Run locally and test end-to-end
- Add unit tests for your transformer

Detailed contribution guides coming soon.

---

## Further Reading

- [Contributing Guide](../../CONTRIBUTING.md)
- [Product Documentation](https://doc.rapida.ai)
- [SDK Documentation](https://doc.rapida.ai/api-reference/installation)

