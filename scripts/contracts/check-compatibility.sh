#!/usr/bin/env bash
set -Eeuo pipefail

baseline_directory=${1:?usage: check-compatibility.sh BASELINE_DIRECTORY}
current_openapi_directory=openapi/artifacts
baseline_openapi_directory=${baseline_directory}/openapi/artifacts
generator=$(pwd)/tests/system/cmd/protobuf-descriptor/main.go
working_directory=$(mktemp -d "${RUNNER_TEMP:-/tmp}/contract-compatibility.XXXXXX")
current_protobuf_image=${working_directory}/current-protos.binpb
baseline_protobuf_image=${working_directory}/baseline-protos.binpb
trap 'rm -rf "${working_directory}"' EXIT

for required_file in \
  go.mod \
  go.sum \
  "${current_openapi_directory}/assistant-api.yaml" \
  "${current_openapi_directory}/talk-api.yaml" \
  "${current_openapi_directory}/common.yaml" \
  "${generator}" \
  "${baseline_directory}/go.mod" \
  "${baseline_directory}/go.sum" \
  "${baseline_openapi_directory}/assistant-api.yaml" \
  "${baseline_openapi_directory}/talk-api.yaml" \
  "${baseline_openapi_directory}/common.yaml"; do
  test -f "${required_file}" || {
    echo "Missing required contract file: ${required_file}" >&2
    exit 1
  }
done

test -x ./tests/system/bin/buf
test -x ./tests/system/bin/oasdiff
compgen -G 'protos/*.go' >/dev/null || {
  echo "Missing current tracked protobuf Go files" >&2
  exit 1
}
compgen -G "${baseline_directory}/protos/*.go" >/dev/null || {
  echo "Missing baseline tracked protobuf Go files" >&2
  exit 1
}

go run ./tests/system/cmd/systemcheck openapi-parse "${current_openapi_directory}"
go run ./tests/system/cmd/systemcheck openapi-parse "${baseline_openapi_directory}"
python3 openapi/scripts/generate_assistant_postman_collection.py --check

./tests/system/bin/oasdiff breaking \
  "${baseline_openapi_directory}/assistant-api.yaml" \
  "${current_openapi_directory}/assistant-api.yaml"
./tests/system/bin/oasdiff breaking \
  "${baseline_openapi_directory}/talk-api.yaml" \
  "${current_openapi_directory}/talk-api.yaml"

go run "${generator}" --output "${current_protobuf_image}"
(
  cd "${baseline_directory}"
  go run "${generator}" --output "${baseline_protobuf_image}"
)

test -s "${current_protobuf_image}"
test -s "${baseline_protobuf_image}"
current_protobuf_sha=$(sha256sum "${current_protobuf_image}" | awk '{ print $1 }')
baseline_protobuf_sha=$(sha256sum "${baseline_protobuf_image}" | awk '{ print $1 }')
if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
  {
    echo
    echo "## Protobuf Compatibility"
    echo
    echo "- Current descriptor SHA-256: \`${current_protobuf_sha}\`"
    echo "- Baseline descriptor SHA-256: \`${baseline_protobuf_sha}\`"
  } >> "${GITHUB_STEP_SUMMARY}"
fi

./tests/system/bin/buf breaking "${current_protobuf_image}" \
  --against "${baseline_protobuf_image}"
