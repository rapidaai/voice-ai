#!/usr/bin/env bash
set -Eeuo pipefail

baseline_directory=${1:?usage: check-compatibility.sh BASELINE_DIRECTORY}
current_openapi_directory=openapi/artifacts
baseline_openapi_directory=${baseline_directory}/openapi/artifacts
baseline_protobuf_directory=${baseline_directory}/protos/artifacts

for required_file in \
  "${current_openapi_directory}/assistant-api.yaml" \
  "${current_openapi_directory}/talk-api.yaml" \
  "${current_openapi_directory}/common.yaml" \
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
test -d "${baseline_protobuf_directory}"

go run ./tests/system/cmd/systemcheck openapi-parse "${current_openapi_directory}"
go run ./tests/system/cmd/systemcheck openapi-parse "${baseline_openapi_directory}"
python3 openapi/scripts/generate_assistant_postman_collection.py --check

./tests/system/bin/oasdiff breaking \
  "${baseline_openapi_directory}/assistant-api.yaml" \
  "${current_openapi_directory}/assistant-api.yaml"
./tests/system/bin/oasdiff breaking \
  "${baseline_openapi_directory}/talk-api.yaml" \
  "${current_openapi_directory}/talk-api.yaml"

./tests/system/bin/buf breaking protos/artifacts \
  --against "${baseline_protobuf_directory}"
