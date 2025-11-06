# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files from root
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all necessary directories (from root context)
COPY api/integration-api ./api/integration-api
COPY bin/ ./bin/
COPY config/ ./config/
COPY cmd/ ./cmd/
COPY protos/ ./protos/
COPY pkg/ ./pkg/

# Build the application from cmd/integration/integration.go
RUN CGO_ENABLED=0 GOOS=linux go build -o integration-api ./cmd/integration/integration.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget netcat-openbsd

# Create rapida-app user
RUN addgroup -g 1000 rapida-app && \
    adduser -D -u 1000 -G rapida-app rapida-app

WORKDIR /opt/apps

# Copy compiled binary from builder
COPY --from=builder /app/integration-api .

# Copy migrations directory
COPY api/integration-api/migrations/ ./api/integration-api/migrations/

# Copy environment file from docker/integration-api
COPY docker/integration-api/.integration.env ./env/.integration.env


# Create directory for assets
RUN mkdir -p /opt/apps/assets
RUN mkdir -p /var/log/go-app

# Change ownership to rapida-app user
RUN chown -R rapida-app:rapida-app /opt/apps
RUN chown -R rapida-app:rapida-app /var/log/go-app

# Switch to rapida-app user
USER rapida-app

# Expose port
EXPOSE 9004

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9004/healthz/ || exit 1

# Run the application
CMD ["./integration-api"]