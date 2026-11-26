#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/contract-baseline-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
fixture_directory="$temporary_directory/repository"
baseline_directory="$temporary_directory/baseline"

mkdir -p \
  "$fixture_directory/openapi/artifacts" \
  "$fixture_directory/protos/authentication"
git -C "$fixture_directory" init --quiet
git -C "$fixture_directory" config user.email test@example.invalid
git -C "$fixture_directory" config user.name test
printf 'module example.invalid/contracts\n\ngo 1.26\n' > "$fixture_directory/go.mod"
printf '' > "$fixture_directory/go.sum"
printf 'services: {}\n' > "$fixture_directory/docker-compose.yml"
printf 'openapi: 3.0.0\n' > "$fixture_directory/openapi/artifacts/assistant-api.yaml"
printf 'openapi: 3.0.0\n' > "$fixture_directory/openapi/artifacts/talk-api.yaml"
printf 'openapi: 3.0.0\n' > "$fixture_directory/openapi/artifacts/common.yaml"
printf 'package protos\n' > "$fixture_directory/protos/root.pb.go"
printf 'package authentication\n' > \
  "$fixture_directory/protos/authentication/authentication.pb.go"
git -C "$fixture_directory" add .
git -C "$fixture_directory" commit --quiet -m fixture
fixture_sha=$(git -C "$fixture_directory" rev-parse HEAD)

(
  cd "$fixture_directory"
  EVENT_NAME=pull_request PULL_REQUEST_BASE_SHA="$fixture_sha" \
    "$script_directory/materialize-baseline.sh" "$baseline_directory"
)
test -f "$baseline_directory/protos/root.pb.go"
test -f "$baseline_directory/protos/authentication/authentication.pb.go"

git -C "$fixture_directory" rm --quiet \
  protos/root.pb.go protos/authentication/authentication.pb.go
git -C "$fixture_directory" commit --quiet -m empty-protobuf
fixture_sha=$(git -C "$fixture_directory" rev-parse HEAD)
if (
  cd "$fixture_directory"
  EVENT_NAME=pull_request PULL_REQUEST_BASE_SHA="$fixture_sha" \
    "$script_directory/materialize-baseline.sh" "$baseline_directory"
) >"$temporary_directory/error.log" 2>&1; then
  printf 'baseline without tracked protobuf Go files unexpectedly passed\n' >&2
  exit 1
fi
grep -Fq 'Target commit has no tracked protos/*.go files' \
  "$temporary_directory/error.log"

printf 'Contract baseline materialization tests passed\n'
