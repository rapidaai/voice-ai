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
COPY api/web-api ./api/web-api
COPY cmd/web/ ./cmd/web/

# Build with cache mounts for Go build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o web-api ./cmd/web/web.go

# Final stage
FROM rapidaai/rapida-alpine:3.21

# Copy compiled binary from builder
COPY --from=builder /app/web-api .

# Copy migrations directory
COPY api/web-api/migrations/ ./api/web-api/migrations/

# Copy default YAML config for standalone container runs
COPY docker/web-api/web.yml ./env/web.yml

# Switch to rapida-app user
USER rapida-app

# Expose port
EXPOSE 9001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9001/healthz/ || exit 1

# Run the application
CMD ["./web-api"]
