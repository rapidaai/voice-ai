#!/usr/bin/env bash
set -Eeuo pipefail

if CGO_ENABLED=0 go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 \
  run --allow-serial-runners --timeout=5m --build-tags=nocgo "$@"; then
  exit 0
fi

echo 'Go lint reported existing issues. This lane is non-blocking to match GitHub CI.' >&2
