# syntax=docker/dockerfile:1
# Build stage
FROM rapidaai/rapida-golang:1.25.13-alpine AS builder

WORKDIR /app

# Copy go mod files from root
COPY go.mod go.sum ./

# Download dependencies (cached unless go.mod/go.sum change)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy only what this service needs
COPY config/ ./config/
COPY protos/ ./protos/
COPY pkg/ ./pkg/
COPY api/integration-api ./api/integration-api
COPY cmd/integration/ ./cmd/integration/

# Build with cache mounts for Go build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o integration-api ./cmd/integration/integration.go

# Final stage
FROM rapidaai/rapida-alpine:3.21

# Copy compiled binary from builder
COPY --from=builder /app/integration-api .

# Copy migrations directory
COPY api/integration-api/migrations/ ./api/integration-api/migrations/

# Copy default YAML config for standalone container runs
COPY docker/integration-api/integration.yml ./env/integration.yml

# Switch to rapida-app user
USER rapida-app

# Expose port
EXPOSE 9004

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9004/healthz/ || exit 1

# Run the application
CMD ["./integration-api"]
