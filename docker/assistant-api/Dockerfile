# syntax=docker/dockerfile:1

FROM rapidaai/rapida-golang:1.25.13-bookworm@sha256:9199ed993f2b4d10f44d265d85436fbad0015031107f27e50c538a60df3ead8d AS native-toolchain

WORKDIR /app

COPY docker/assistant-api/native-deps.lock ./docker/assistant-api/native-deps.lock
COPY docker/assistant-api/scripts/ ./docker/assistant-api/scripts/
RUN ./docker/assistant-api/scripts/verify-native-deps.sh && \
    ./docker/assistant-api/scripts/install-system-deps.sh

COPY api/assistant-api/internal/vad/internal/ten_vad/lib/Linux/x64/libten_vad.so /opt/ten_vad/lib/
COPY api/assistant-api/internal/vad/internal/ten_vad/ten_vad.h /opt/ten_vad/include/
RUN ./docker/assistant-api/scripts/verify-native-deps.sh

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY config/ ./config/
COPY openapi/ ./openapi/
COPY protos/ ./protos/
COPY pkg/ ./pkg/
COPY api/assistant-api ./api/assistant-api
COPY cmd/assistant/ ./cmd/assistant/

RUN --mount=type=cache,target=/root/.cache/pip \
    ./docker/assistant-api/scripts/install-model-assets.sh

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_CFLAGS="-I/opt/onnxruntime/include -I/usr/local/include -I/opt/azure-speech-sdk/include/c_api -I/opt/ten_vad/include"
ENV CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime -L/usr/local/lib -lrnnoise -L/opt/azure-speech-sdk/lib/x64 -lMicrosoft.CognitiveServices.Speech.core -L/opt/ten_vad/lib -lten_vad -lc++"
ENV LD_LIBRARY_PATH="/opt/onnxruntime/lib:/usr/local/lib:/opt/azure-speech-sdk/lib/x64:/opt/ten_vad/lib"

FROM native-toolchain AS ci
RUN . ./docker/assistant-api/native-deps.lock && \
    go mod download && \
    go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"
ENTRYPOINT ["./docker/assistant-api/scripts/run-native-ci.sh"]

FROM native-toolchain AS builder
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -o assistant-api ./cmd/assistant/assistant.go

FROM rapidaai/rapida-debian:bookworm-slim@sha256:4ef57670941b638b4274d0b1079163ddf4a84acffd676d0b319e5054eb8b4832 AS runtime

COPY docker/assistant-api/native-deps.lock /tmp/native-deps.lock
COPY docker/assistant-api/scripts/load-native-deps.sh docker/assistant-api/scripts/install-system-deps.sh /tmp/native-scripts/
RUN NATIVE_DEPS_LOCK=/tmp/native-deps.lock /tmp/native-scripts/install-system-deps.sh runtime && \
    rm -rf /tmp/native-deps.lock /tmp/native-scripts

COPY --from=builder /app/assistant-api .
COPY --from=builder /opt/onnxruntime /opt/onnxruntime
COPY --from=builder /opt/azure-speech-sdk /opt/azure-speech-sdk
COPY --from=builder /opt/ten_vad /opt/ten_vad
COPY --from=builder /usr/local/lib /usr/local/lib

COPY --from=builder /app/api/assistant-api/internal/vad/internal/silero_vad/models/ ./models/silero_vad/
COPY --from=builder /app/api/assistant-api/internal/vad/internal/firered_vad/models/ ./models/firered_vad/
COPY --from=builder /app/api/assistant-api/internal/end_of_speech/internal/livekit/models/ ./models/livekit_turn/
COPY --from=builder /app/api/assistant-api/internal/end_of_speech/internal/pipecat/models/ ./models/pipecat_turn/

COPY api/assistant-api/migrations/ ./api/assistant-api/migrations/
COPY docker/assistant-api/assistant.yml ./env/assistant.yml

USER rapida-app

ENV LD_LIBRARY_PATH="/usr/local/lib:/opt/onnxruntime/lib:/opt/azure-speech-sdk/lib/x64:/opt/ten_vad/lib"
ENV SILERO_MODEL_PATH="./models/silero_vad/silero_vad_20251001.onnx"
ENV FIRERED_VAD_MODEL_PATH="./models/firered_vad/fireredvad_stream_vad_with_cache.onnx"
ENV LIVEKIT_TURN_MODEL_PATH="./models/livekit_turn/model_q8.onnx"
ENV LIVEKIT_TURN_MULTI_MODEL_PATH="./models/livekit_turn/model_q8_multilingual.onnx"
ENV LIVEKIT_TURN_TOKENIZER_PATH="./models/livekit_turn/tokenizer.json"
ENV PIPECAT_TURN_MODEL_PATH="./models/pipecat_turn/smart-turn-v3.2-cpu.onnx"

EXPOSE 9007

CMD ["./assistant-api"]
