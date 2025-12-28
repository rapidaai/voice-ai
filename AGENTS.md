# Rapida Voice AI - Agent Instructions

> This document provides context for AI agents working on this codebase.
> Follow the Research → Plan → Implement workflow for non-trivial tasks.

## Project Overview

**Rapida** is an open-source, end-to-end voice orchestration platform built in Go with a React TypeScript frontend.

**Core capabilities:**
- Real-time audio streaming via gRPC
- LLM-agnostic architecture (OpenAI, Anthropic, Google, Cohere, etc.)
- Multiple STT providers (Deepgram, Azure, Google, AssemblyAI, Speechmatics, etc.)
- Multiple TTS providers (ElevenLabs, Cartesia, Azure, Google, Sarvam, etc.)
- Telephony integrations (Twilio, Vonage, Exotel)
- Voice Activity Detection (Silero VAD, Ten VAD)
- Full observability and metrics

---

## Architecture

```
├── api/                    # Service APIs (each is a separate microservice)
│   ├── assistant-api/      # Core voice orchestration (STT → LLM → TTS pipelines)
│   ├── web-api/            # User management, auth, organization APIs
│   ├── integration-api/    # Third-party integrations management
│   ├── endpoint-api/       # Endpoint/deployment management
│   └── document-api/       # Document handling (Python/FastAPI)
├── cmd/                    # Service entry points (main.go equivalents)
├── pkg/                    # Shared Go packages
│   ├── commons/            # Logger (zap-based), constants, HTTP responses
│   ├── middlewares/        # gRPC/HTTP auth & logging middlewares
│   ├── models/gorm/        # Database models with embedded types
│   ├── types/              # Shared types, enums, JWT handling
│   ├── utils/              # Utility functions
│   ├── connectors/         # Database connectors (Postgres, Redis, OpenSearch)
│   └── clients/            # Internal service clients
├── protos/                 # Generated protobuf/gRPC code (DO NOT EDIT)
├── ui/                     # React TypeScript frontend
└── docker/                 # Dockerfiles per service
```

### Service Communication
- **Inter-service**: gRPC with protobuf
- **External APIs**: REST (Gin) + gRPC-Web
- **Real-time audio**: Bidirectional gRPC streams + WebSocket

---

## Code Conventions

### Go Services

#### Package Naming
```
internal_transformer_deepgram  # snake_case with prefix
internal_user_service          # domain + service suffix
gorm_models                    # shared package naming
type_enums                     # enums package
```

#### Struct Patterns
API handlers embed a base struct, then have RPC and gRPC variants:

```go
// Base struct with dependencies
type webAuthApi struct {
    cfg      *config.WebAppConfig
    logger   commons.Logger
    postgres connectors.PostgresConnector
    // ... domain services
}

// RPC (HTTP/REST) variant
type webAuthRPCApi struct {
    webAuthApi
}

// gRPC variant
type webAuthGRPCApi struct {
    webAuthApi
}
```

#### Constructor Pattern
Always inject dependencies via constructor:

```go
func NewUserService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.UserService {
    return &userService{
        logger:   logger,
        postgres: postgres,
    }
}
```

#### Database Models
Embed base types for consistent fields:

```go
type MyModel struct {
    gorm_models.Audited        // Id, CreatedDate, UpdatedDate
    gorm_models.Mutable        // Status, CreatedBy, UpdatedBy
    gorm_models.Organizational // ProjectId, OrganizationId

    Name        string `json:"name" gorm:"type:varchar(255);not null"`
    Description string `json:"description" gorm:"type:text"`
}
```

IDs use snowflake generation via `gorm_generator.ID()`.

#### Logging
Use `commons.Logger` interface (zap-based), never `fmt.Println`:

```go
logger.Errorf("failed to process request: %v", err)
logger.Infof("user %d authenticated", userId)
logger.Tracef(ctx, "processing for %s", endpoint)  // includes request ID
logger.Benchmark("FunctionName", duration)         // performance tracking
```

#### Error Handling
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Log errors at the handling point, not at every level
- Return appropriate gRPC status codes

#### Import Ordering
Imports should be grouped: stdlib → external → internal (github.com/rapidaai)

```go
import (
    "context"
    "fmt"

    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"

    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
)
```

### Frontend (ui/)

- **React 18** with TypeScript (strict mode)
- **State**: Redux Toolkit + Zustand
- **Styling**: Tailwind CSS 4.x
- **Components**: Headless UI, Material Tailwind
- **Backend**: gRPC-Web for typed communication

---

## Validation Commands

### Go
```bash
# Build all services
go build ./...

# Run linter (excludes cgo-dependent packages)
CGO_ENABLED=0 golangci-lint run ./pkg/...

# Run tests
go test -v ./...

# Format code
gofmt -w .
```

### Frontend
```bash
cd ui
yarn lint          # ESLint check
yarn lint:fix      # Auto-fix
yarn checkTs       # TypeScript type check
yarn test          # Jest tests
```

### Make Targets
```bash
make lint          # Run all linters
make test          # Run all tests
make check         # Lint + test (pre-commit)
make build-all     # Build all Docker images
make up-all        # Start all services
```

---

## Key Patterns

### Adding a New STT Provider

1. Create `api/assistant-api/internal/transformer/{provider}/stt.go`
2. Implement `transformer.SpeechToTextTransformer` interface:
   ```go
   type SpeechToTextTransformer interface {
       Name() string
       Initialize() error
       Transform(ctx context.Context, audio []byte, opts *SpeechToTextOption) error
       Close() error
   }
   ```
3. Create options file: `{provider}/options.go`
4. Register in `factory/transformer/audio_transformer_factory.go`
5. Add provider enum to `pkg/types/enums/`

### Adding a New TTS Provider

1. Create `api/assistant-api/internal/transformer/{provider}/tts.go`
2. Implement `transformer.TextToSpeechTransformer` interface
3. Register in factory
4. Add provider enum

### Adding a New Telephony Channel

1. Create `api/assistant-api/internal/telephony/{provider}/`
2. Implement `telephony.Telephony` interface
3. Create WebSocket handler: `{provider}_websocket.go`
4. Register in `factory/telephony/telephony_factory.go`

---

## Service Ports

| Service         | Port  | Description                        |
|-----------------|-------|------------------------------------|
| UI              | 3000  | React frontend                     |
| Web API         | 9001  | Auth, users, organizations         |
| Integration API | 9004  | Third-party integrations           |
| Endpoint API    | 9005  | Deployment management              |
| Assistant API   | 9007  | Voice orchestration (core)         |
| Document API    | 9010  | Document handling (Python)         |
| PostgreSQL      | 5432  | Primary database                   |
| Redis           | 6379  | Cache and session store            |
| OpenSearch      | 9200  | Search and analytics               |

---

## Testing Requirements

- New API endpoints MUST have corresponding `*_test.go` files
- Tests go in the same directory as the code
- Use table-driven tests for multiple scenarios
- Mock external dependencies (don't call real APIs)

---

## Common Issues

### CGO Dependencies
Some packages require native libraries (Azure Speech SDK, Silero VAD). These are excluded from linting but will build in Docker with proper dependencies.

### Protobuf Files
Files in `protos/` are generated. Do NOT edit manually. Regenerate from proto definitions if needed.

### Import Cycles
The package structure is designed to avoid cycles:
- `pkg/` → shared utilities (no internal imports)
- `api/{service}/internal/` → business logic
- `api/{service}/api/` → handlers (imports internal/)

---

## Do NOT

- Use `fmt.Println` for logging (use `commons.Logger`)
- Edit files in `protos/` (they are generated)
- Add dependencies without justification
- Bypass authentication middleware
- Store secrets in code (use environment variables)
- Create circular imports between packages
- Skip error checking on critical operations

---

## Research → Plan → Implement

For non-trivial tasks (multi-file changes, new features, architectural decisions):

### 1. Research Phase
- Understand existing patterns before proposing changes
- Find similar implementations in the codebase
- Note file paths and line numbers for reference
- Document what exists, not what should change

### 2. Plan Phase
- Create explicit step-by-step implementation plan
- Reference specific files and line numbers
- Define scope boundaries (what we're NOT doing)
- Include success criteria

### 3. Implement Phase
- Execute the plan systematically
- Run validation commands after changes
- Keep changes focused on the plan scope
