#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
# shellcheck disable=SC1091
. "$SCRIPT_DIR/load-native-deps.sh"

download_verified() {
  url=$1
  sha256=$2
  destination=$3
  temporary="${destination}.download"
  mkdir -p "$(dirname "$destination")"
  curl -fsSL --retry 3 --retry-all-errors -o "$temporary" "$url"
  printf '%s  %s\n' "$sha256" "$temporary" | sha256sum -c -
  mv "$temporary" "$destination"
}

numpy_wheel=/tmp/${NUMPY_PYTHON_URL##*/}
protobuf_wheel=/tmp/${PROTOBUF_PYTHON_URL##*/}
onnx_wheel=/tmp/${ONNX_PYTHON_URL##*/}
download_verified "$NUMPY_PYTHON_URL" "$NUMPY_PYTHON_SHA256" "$numpy_wheel"
download_verified "$PROTOBUF_PYTHON_URL" "$PROTOBUF_PYTHON_SHA256" "$protobuf_wheel"
download_verified "$ONNX_PYTHON_URL" "$ONNX_PYTHON_SHA256" "$onnx_wheel"
pip install --no-cache-dir --break-system-packages --no-deps "$numpy_wheel" "$protobuf_wheel" "$onnx_wheel"
rm -f "$numpy_wheel" "$protobuf_wheel" "$onnx_wheel"

livekit_dir=api/assistant-api/internal/end_of_speech/internal/livekit/models
pipecat_dir=api/assistant-api/internal/end_of_speech/internal/pipecat/models

download_verified "https://huggingface.co/livekit/turn-detector/resolve/${LIVEKIT_EN_REVISION}/onnx/model_q8.onnx" "$LIVEKIT_EN_MODEL_SHA256" "$livekit_dir/model_q8.onnx"
download_verified "https://huggingface.co/livekit/turn-detector/resolve/${LIVEKIT_INTL_REVISION}/onnx/model_q8.onnx" "$LIVEKIT_INTL_MODEL_SHA256" "$livekit_dir/model_q8_multilingual.onnx"
download_verified "https://huggingface.co/livekit/turn-detector/resolve/${LIVEKIT_EN_REVISION}/tokenizer.json" "$LIVEKIT_TOKENIZER_SHA256" "$livekit_dir/tokenizer.json"
download_verified "https://huggingface.co/pipecat-ai/smart-turn-v3/resolve/${PIPECAT_REVISION}/smart-turn-v3.2-cpu.onnx" "$PIPECAT_MODEL_SHA256" "$pipecat_dir/smart-turn-v3.2-cpu.onnx"

python3 - <<'PY'
import onnx

paths = (
    "api/assistant-api/internal/end_of_speech/internal/livekit/models/model_q8.onnx",
    "api/assistant-api/internal/end_of_speech/internal/livekit/models/model_q8_multilingual.onnx",
)
for path in paths:
    model = onnx.load(path)
    used_domains = {node.domain for node in model.graph.node}
    imports = [item for item in model.opset_import if not item.domain or item.domain in used_domains]
    del model.opset_import[:]
    model.opset_import.extend(imports)
    onnx.save(model, path)

path = "api/assistant-api/internal/end_of_speech/internal/pipecat/models/smart-turn-v3.2-cpu.onnx"
model = onnx.load(path)
model.ir_version = 9
used_domains = {node.domain for node in model.graph.node}
imports = [item for item in model.opset_import if not item.domain or item.domain in used_domains]
del model.opset_import[:]
model.opset_import.extend(imports)
onnx.save(model, path)
PY

pip uninstall -y --break-system-packages onnx numpy protobuf
