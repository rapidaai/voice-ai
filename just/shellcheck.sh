#!/usr/bin/env bash
set -Eeuo pipefail

files=(
  just/ci-assistant-native.sh
  just/ci-commitlint.sh
  just/ci-contracts.sh
  just/ci-go-lint.sh
  just/ci-security.sh
  just/ci-stack.sh
  just/doctor.sh
  just/require-docker.sh
  just/shellcheck.sh
  tests/smoke/run.sh
  scripts/contracts/check-compatibility.sh
  scripts/contracts/materialize-baseline.sh
  scripts/contracts/bin/buf
  scripts/contracts/bin/oasdiff
)

if command -v shellcheck >/dev/null 2>&1; then
  shellcheck "${files[@]}"
  exit
fi

docker run --rm -v "$PWD:/mnt" -w /mnt koalaman/shellcheck-alpine:v0.11.0 \
  shellcheck "${files[@]}"
