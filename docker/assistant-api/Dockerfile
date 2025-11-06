# Build stage
FROM --platform=linux/amd64 golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add gcc make libtool autoconf automake curl git wget bash tar

# Install necessary Alpine build dependencies for CGO
RUN apk add build-base g++ libc6-compat musl-dev


# Copy go mod files from root
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install Protobuf and related tools
RUN wget -q https://github.com/protocolbuffers/protobuf/releases/download/v3.20.0/protoc-3.20.0-linux-x86_64.zip && \
    unzip protoc-3.20.0-linux-x86_64.zip -d /usr/local && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Install ONNX Runtime
RUN wget https://github.com/microsoft/onnxruntime/releases/download/v1.16.0/onnxruntime-linux-x64-1.16.0.tgz && \
    mkdir -p /opt/onnxruntime && \
    tar -xzf onnxruntime-linux-x64-1.16.0.tgz -C /opt/onnxruntime --strip-components=1

# Install RNNoise
RUN git clone https://github.com/xiph/rnnoise.git && \
    cd rnnoise && \
    ./autogen.sh && ./configure --prefix=/usr/local && make && make install

# Install Azure Speech SDK
RUN wget -O SpeechSDK-Linux.tar.gz https://aka.ms/csspeech/linuxbinary && \
    mkdir -p /opt/azure-speech-sdk && \
    tar --strip 1 -xzf SpeechSDK-Linux.tar.gz -C /opt/azure-speech-sdk && \
    sed -i '/Speech_SegmentationMaximumTimeMs = 9004/d' /opt/azure-speech-sdk/include/c_api/speechapi_c_property_bag.h

# Export CGO_CFLAGS and CGO_LDFLAGS to include the correct paths
ENV CGO_CFLAGS="-I/opt/onnxruntime/include -I/usr/local/include -I/opt/azure-speech-sdk/include/c_api"
ENV CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime -L/usr/local/lib -lrnnoise -L/opt/azure-speech-sdk/lib/x64 -lMicrosoft.CognitiveServices.Speech.core"
ENV LD_LIBRARY_PATH="/opt/onnxruntime/lib:/usr/local/lib:/opt/azure-speech-sdk/lib/x64"


# Copy all necessary directories (from root context)
COPY api/assistant-api ./api/assistant-api
COPY bin/ ./bin/
COPY config/ ./config/
COPY cmd/ ./cmd/
COPY protos/ ./protos/
COPY pkg/ ./pkg/

# Build the application from cmd/assistant/assistant.go
RUN GOARCH=amd64 CGO_ENABLED=1 GOOS=linux go build -o assistant-api ./cmd/assistant/assistant.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget netcat-openbsd

# Create rapida-app user
RUN addgroup -g 1000 rapida-app && \
    adduser -D -u 1000 -G rapida-app rapida-app

WORKDIR /opt/apps

# Copy compiled application binary from builder
COPY --from=builder /app/assistant-api .

# Copy necessary runtime dependencies (if applicable, i.e., custom RNNoise/ONNX/Azure binaries)
COPY --from=builder /usr/local/lib/* /usr/local/lib/
COPY --from=builder /opt/onnxruntime /opt/onnxruntime
COPY --from=builder /opt/azure-speech-sdk /opt/azure-speech-sdk

# Copy migrations directory
COPY api/assistant-api/migrations/ ./api/assistant-api/migrations/

# Copy environment file from docker/assistant-api
COPY docker/assistant-api/.assistant.env ./env/.assistant.env

# Create directory for assets
RUN mkdir -p /opt/apps/assets
RUN mkdir -p /var/log/go-app

# Change ownership to rapida-app user
RUN chown -R rapida-app:rapida-app /opt/apps
RUN chown -R rapida-app:rapida-app /var/log/go-app

# Switch to rapida-app user
USER rapida-app

# Set environment variable for library paths
ENV LD_LIBRARY_PATH="/usr/local/lib:/opt/onnxruntime/lib:/opt/azure-speech-sdk/lib/x64"


# Expose port
EXPOSE 9001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9001/healthz/ || exit 1

# Run the application
CMD ["./assistant-api"]