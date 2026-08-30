#!/usr/bin/env bash
set -Eeuo pipefail

if (($#)); then
  files=("$@")
else
  files=(
    just/ci-assistant-native.sh
    just/ci-commitlint.sh
    just/ci-contracts.sh
    just/ci-go-lint.sh
    just/ci-python.sh
    just/ci-security.sh
    just/ci-service-boundaries.sh
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
fi

if command -v shellcheck >/dev/null 2>&1; then
  shellcheck "${files[@]}"
  exit
fi

docker run --rm -v "$PWD:/mnt:ro" -w /mnt koalaman/shellcheck-alpine@sha256:9955be09ea7f0dbf7ae942ac1f2094355bb30d96fffba0ec09f5432207544002 \
  shellcheck "${files[@]}"
