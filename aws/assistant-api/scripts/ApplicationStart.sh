#!/bin/bash
export ENV_PATH="/opt/app/backend-app/artifacts/assistant-api/env.production"
export CGO_CFLAGS="-Il"
export CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime"
export LD_LIBRARY_PATH="/opt/onnxruntime/lib:$LD_LIBRARY_PATH"
export CGO_CFLAGS="$CGO_CFLAGS -I/usr/local/include"  # Append RNNoise include path
export CGO_LDFLAGS="$CGO_LDFLAGS -L/usr/local/lib -lrnnoise"  # Append RNNoise library path
export LD_LIBRARY_PATH="/usr/local/lib:$LD_LIBRARY_PATH"  # Append RNNoise library path
export SILERO_MODEL_PATH="/opt/silero/silero-vad/src/silero_vad/data/silero_vad_16k_op15.onnx"
# for azure sdk
export CGO_CFLAGS="$CGO_CFLAGS -I/opt/azure-speech-sdk/include/c_api"
export CGO_LDFLAGS="$CGO_LDFLAGS -L/opt/azure-speech-sdk/lib/x64 -lMicrosoft.CognitiveServices.Speech.core"
export LD_LIBRARY_PATH="/opt/azure-speech-sdk/lib/x64:$LD_LIBRARY_PATH"

/opt/app/backend-app/artifacts/assistant-api/assistant-api.0.0.1 > /dev/null 2> /dev/null < /dev/null &
