# syntax=docker/dockerfile:1
# ---- Build stage ----
FROM rapidaai/rapida-golang:1.25.8-bookworm AS builder

WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# ---- Copy only what this service needs ----
COPY config/ ./config/
COPY protos/ ./protos/
COPY pkg/ ./pkg/
COPY api/assistant-api ./api/assistant-api
COPY cmd/assistant/ ./cmd/assistant/

# ---- Install Python for ONNX model processing ----
RUN apt-get update && apt-get install -y --no-install-recommends python3 python3-pip && \
    rm -rf /var/lib/apt/lists/*

# ---- Download turn detector models ----
# Removed: LiveKit turn-detector model (license restricts usage to LiveKit Agents only)
# Removed: TEN VAD (Agora license has anti-competition clause)
RUN pip install --no-cache-dir --break-system-packages onnx && \
    mkdir -p ./api/assistant-api/internal/end_of_speech/internal/pipecat/models && \
    curl -fsSL -o ./api/assistant-api/internal/end_of_speech/internal/pipecat/models/smart-turn-v3.2-cpu.onnx \
    "https://huggingface.co/pipecat-ai/smart-turn-v3/resolve/main/smart-turn-v3.2-cpu.onnx" && \
    python3 -c "import onnx; m=onnx.load('api/assistant-api/internal/end_of_speech/internal/pipecat/models/smart-turn-v3.2-cpu.onnx'); \
    m.ir_version=9; used={n.domain for n in m.graph.node}; keep=[o for o in m.opset_import if o.domain=='' or o.domain in used]; \
    del m.opset_import[:]; m.opset_import.extend(keep); onnx.save(m, 'api/assistant-api/internal/end_of_speech/internal/pipecat/models/smart-turn-v3.2-cpu.onnx')" && \
    pip uninstall -y --break-system-packages onnx

# ---- Build Go binary with all CGO flags in one RUN command ----
# Removed: TEN VAD shared library (Agora license has anti-competition clause)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    export CGO_CFLAGS="-I/opt/onnxruntime/include -I/usr/local/include -I/opt/azure-speech-sdk/include/c_api" && \
    export CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime -L/usr/local/lib -lrnnoise -L/opt/azure-speech-sdk/lib/x64 -lMicrosoft.CognitiveServices.Speech.core" && \
    export LD_LIBRARY_PATH="/opt/onnxruntime/lib:/usr/local/lib:/opt/azure-speech-sdk/lib/x64:$LD_LIBRARY_PATH" && \
    CGO_ENABLED=1 go build -v -o assistant-api ./cmd/assistant/assistant.go

# ---- Runtime stage ----
FROM rapidaai/rapida-debian:bookworm-slim AS runtime

# Install assistant-api specific runtime deps
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    libopus0 libopusfile0 libc++1

# Copy binary and libraries
COPY --from=builder /app/assistant-api .
COPY --from=builder /opt/onnxruntime /opt/onnxruntime
COPY --from=builder /opt/azure-speech-sdk /opt/azure-speech-sdk
COPY --from=builder /usr/local/lib /usr/local/lib

# Copy ML model files (VAD + turn detector)
COPY --from=builder /app/api/assistant-api/internal/vad/internal/silero_vad/models/ ./models/silero_vad/
COPY --from=builder /app/api/assistant-api/internal/vad/internal/firered_vad/models/ ./models/firered_vad/
COPY --from=builder /app/api/assistant-api/internal/end_of_speech/internal/pipecat/models/ ./models/pipecat_turn/

# Copy migrations and env
COPY api/assistant-api/migrations/ ./api/assistant-api/migrations/
COPY docker/assistant-api/.assistant.env ./env/.assistant.env

USER rapida-app

ENV LD_LIBRARY_PATH="/usr/local/lib:/opt/onnxruntime/lib:/opt/azure-speech-sdk/lib/x64"
ENV SILERO_MODEL_PATH="./models/silero_vad/silero_vad_20251001.onnx"
ENV FIRERED_VAD_MODEL_PATH="./models/firered_vad/fireredvad_stream_vad_with_cache.onnx"
ENV PIPECAT_TURN_MODEL_PATH="./models/pipecat_turn/smart-turn-v3.2-cpu.onnx"

EXPOSE 9007

CMD ["./assistant-api"]
