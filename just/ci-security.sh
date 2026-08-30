#!/usr/bin/env bash
set -Eeuo pipefail

bash just/require-docker.sh

docker run --rm \
  -v "$PWD:/workspace:ro" \
  -v rapida-trivy-cache:/root/.cache/ \
  aquasec/trivy:0.70.0 fs \
  --severity CRITICAL,HIGH \
  --ignore-unfixed \
  --skip-dirs /workspace/.git,/workspace/.cache,/workspace/ui/node_modules \
  --exit-code 0 \
  /workspace
