#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
if [ "$#" -gt 1 ]; then
  echo "usage: $0 [native-deps.lock]" >&2
  exit 2
fi
NATIVE_DEPS_LOCK=${1:-${NATIVE_DEPS_LOCK:-$SCRIPT_DIR/../native-deps.lock}}
export NATIVE_DEPS_LOCK
# shellcheck disable=SC1091
. "$SCRIPT_DIR/load-native-deps.sh"

check_sha256() {
  name=$1
  value=$2
  if ! printf '%s\n' "$value" | grep -Eq '^[0-9a-f]{64}$'; then
    echo "$name must contain 64 hexadecimal characters" >&2
    exit 1
  fi
}

check_digest() {
  name=$1
  value=$2
  case "$value" in
    sha256:*) check_sha256 "$name" "${value#sha256:}" ;;
    *) echo "$name must use a sha256 digest" >&2; exit 1 ;;
  esac
}

check_digest BUILDER_DIGEST "$BUILDER_DIGEST"
check_digest RUNTIME_DIGEST "$RUNTIME_DIGEST"
check_sha256 ONNX_PYTHON_SHA256 "$ONNX_PYTHON_SHA256"
check_sha256 NUMPY_PYTHON_SHA256 "$NUMPY_PYTHON_SHA256"
check_sha256 PROTOBUF_PYTHON_SHA256 "$PROTOBUF_PYTHON_SHA256"
check_sha256 LIVEKIT_EN_MODEL_SHA256 "$LIVEKIT_EN_MODEL_SHA256"
check_sha256 LIVEKIT_TOKENIZER_SHA256 "$LIVEKIT_TOKENIZER_SHA256"
check_sha256 LIVEKIT_INTL_MODEL_SHA256 "$LIVEKIT_INTL_MODEL_SHA256"
check_sha256 PIPECAT_MODEL_SHA256 "$PIPECAT_MODEL_SHA256"
check_sha256 TEN_VAD_LIBRARY_SHA256 "$TEN_VAD_LIBRARY_SHA256"

if [ "$DEBIAN_ARCH" != amd64 ] || [ "$TEN_VAD_ARCH" != Linux/x64 ]; then
  echo "assistant native dependencies must be locked to Linux amd64" >&2
  exit 1
fi

dockerfile=$SCRIPT_DIR/../Dockerfile
if [ -f "$dockerfile" ]; then
  grep -Fq "${BUILDER_IMAGE}@${BUILDER_DIGEST} AS native-toolchain" "$dockerfile" || {
    echo "builder image does not match native dependency lock" >&2
    exit 1
  }
  grep -Fq "${RUNTIME_IMAGE}@${RUNTIME_DIGEST} AS runtime" "$dockerfile" || {
    echo "runtime image does not match native dependency lock" >&2
    exit 1
  }
fi

ten_library=/opt/ten_vad/lib/libten_vad.so
if [ -f "$ten_library" ]; then
  printf '%s  %s\n' "$TEN_VAD_LIBRARY_SHA256" "$ten_library" | sha256sum -c -
fi

echo "native dependency lock verified"
