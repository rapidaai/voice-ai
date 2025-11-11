# ---- Build stage ----
FROM golang:1.24-bullseye AS builder

WORKDIR /app

# Install system dependencies for CGO
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc g++ make autoconf automake libtool pkg-config \
    curl git wget unzip ca-certificates tar \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# ---- Install ONNX Runtime ----
RUN mkdir -p /tmp/build && cd /tmp/build && \
    wget -q https://github.com/microsoft/onnxruntime/releases/download/v1.16.0/onnxruntime-linux-x64-1.16.0.tgz && \
    mkdir -p /opt/onnxruntime && \
    tar -xzf onnxruntime-linux-x64-1.16.0.tgz -C /opt/onnxruntime --strip-components=1 && \
    rm onnxruntime-linux-x64-1.16.0.tgz

# ---- Build RNNoise ----
# RUN cd /tmp/build && \
#     git clone https://github.com/rapidaai/rnnoise.git && \
#     cd rnnoise && \
#     ./autogen.sh && \
#     ./configure --prefix=/usr/local && \
#     make -j$(nproc) && \
#     make install && \
#     cd .. && rm -rf rnnoise

RUN cd /tmp/build && \
    curl -fsSL https://github.com/rapidaai/rnnoise/archive/refs/heads/main.tar.gz -o rnnoise.tar.gz && \
    tar -xzf rnnoise.tar.gz && \
    cd rnnoise-main && \
    ./autogen.sh && \
    ./configure --prefix=/usr/local && \
    make -j$(nproc) && \
    make install && \
    cd /tmp/build && \
    rm -rf rnnoise.tar.gz rnnoise-main


# ---- Install Azure Speech SDK ----
RUN cd /tmp/build && \
    wget -q -O SpeechSDK-Linux.tar.gz https://aka.ms/csspeech/linuxbinary && \
    mkdir -p /opt/azure-speech-sdk && \
    tar --strip 1 -xzf SpeechSDK-Linux.tar.gz -C /opt/azure-speech-sdk && \
    rm SpeechSDK-Linux.tar.gz && \
    sed -i '/Speech_SegmentationMaximumTimeMs = 9004/d' /opt/azure-speech-sdk/include/c_api/speechapi_c_property_bag.h && \
    rm -rf /tmp/build

# ---- Copy project files ----
COPY api/assistant-api ./api/assistant-api
COPY bin/ ./bin/
COPY config/ ./config/
COPY cmd/ ./cmd/
COPY protos/ ./protos/
COPY pkg/ ./pkg/

# ---- Build Go binary with all CGO flags in one RUN command ----
RUN export CGO_CFLAGS="-I/opt/onnxruntime/include -I/usr/local/include -I/opt/azure-speech-sdk/include/c_api" && \
    export CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime -L/usr/local/lib -lrnnoise -L/opt/azure-speech-sdk/lib/x64 -lMicrosoft.CognitiveServices.Speech.core" && \
    export LD_LIBRARY_PATH="/opt/onnxruntime/lib:/usr/local/lib:/opt/azure-speech-sdk/lib/x64:$LD_LIBRARY_PATH" && \
    CGO_ENABLED=1 go build -v -o assistant-api ./cmd/assistant/assistant.go

# ---- Runtime stage ----
FROM golang:1.24-bullseye AS runtime

WORKDIR /opt/apps

# Install minimal runtime deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates wget netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

# Create user
RUN groupadd -g 1000 rapida-app && useradd -m -u 1000 -g rapida-app rapida-app

# Copy binary and libraries
COPY --from=builder /app/assistant-api .
COPY --from=builder /opt/onnxruntime /opt/onnxruntime
COPY --from=builder /opt/azure-speech-sdk /opt/azure-speech-sdk
COPY --from=builder /usr/local/lib /usr/local/lib

# Copy migrations and env
COPY api/assistant-api/migrations/ ./api/assistant-api/migrations/
COPY docker/assistant-api/.assistant.env ./env/.assistant.env

# Create required directories
RUN mkdir -p /opt/apps/assets /var/log/go-app && \
    chown -R rapida-app:rapida-app /opt/apps /var/log/go-app

USER rapida-app

ENV LD_LIBRARY_PATH="/usr/local/lib:/opt/onnxruntime/lib:/opt/azure-speech-sdk/lib/x64"

EXPOSE 9007

CMD ["./assistant-api"]