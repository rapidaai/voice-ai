# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files from root
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all necessary directories (from root context)
COPY api/web-api ./api/web-api
COPY bin/ ./bin/
COPY config/ ./config/
COPY cmd/ ./cmd/
COPY protos/ ./protos/
COPY pkg/ ./pkg/

# Build the application from cmd/web/web.go
RUN CGO_ENABLED=0 GOOS=linux go build -o web-api ./cmd/web/web.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget netcat-openbsd

# Create rapida-app user
RUN addgroup -g 1000 rapida-app && \
    adduser -D -u 1000 -G rapida-app rapida-app

WORKDIR /opt/apps

# Copy compiled binary from builder
COPY --from=builder /app/web-api .

# Copy migrations directory
COPY api/web-api/migrations/ ./api/web-api/migrations/

# Copy environment file from docker/web-api
COPY docker/web-api/.web.env ./env/.web.env


# Create directory for assets
RUN mkdir -p /opt/apps/assets
RUN mkdir -p /var/log/go-app

# Change ownership to rapida-app user
RUN chown -R rapida-app:rapida-app /opt/apps
RUN chown -R rapida-app:rapida-app /var/log/go-app

# Switch to rapida-app user
USER rapida-app

# Expose port
EXPOSE 9001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9001/healthz/ || exit 1

# Run the application
CMD ["./web-api"]