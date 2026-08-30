#!/usr/bin/env bash
set -Eeuo pipefail

bash just/require-docker.sh
./bin/check-go-version-consistency
./docker/assistant-api/scripts/verify-native-deps.sh docker/assistant-api/native-deps.lock

context=$(mktemp -d "${RUNNER_TEMP:-/tmp}/assistant-native-context.XXXXXX")
image="rapida/assistant-native-ci:${GITHUB_SHA:-local}"
trap 'rm -rf "$context"' EXIT

git ls-files -co --exclude-standard -z |
  while IFS= read -r -d '' path; do
    [[ -e "$path" ]] && printf '%s\0' "$path"
  done |
  tar --null -T - -cf - |
  tar -xf - -C "$context"
rm -f "$context/.dockerignore"

docker build \
  --file "$context/docker/assistant-api/Dockerfile" \
  --target ci \
  --platform linux/amd64 \
  --tag "$image" \
  "$context"
docker run --rm --platform linux/amd64 "$image"
