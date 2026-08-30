#!/usr/bin/env bash
set -Eeuo pipefail

files=(
  .github/workflows/ci.yml
  .github/workflows/package.yml
  .github/workflows/reusable-assistant-native-ci.yml
  .github/workflows/reusable-docker-ci.yml
  .github/workflows/reusable-end-to-end-ci.yml
  .github/workflows/reusable-go-ci.yml
  .github/workflows/reusable-repository-ci.yml
  .github/workflows/reusable-ui-ci.yml
  .github/workflows/tag-and-package-services.yml
  docker-compose.ci.yml
  just/ci.just
  just/ci-stack.sh
  tests/smoke
)

if grep -R -n -F -- 'document-api' "${files[@]}"; then
  echo 'document-api must remain outside CI, packaging, and release flows.' >&2
  exit 1
fi

echo 'CI service boundary excludes document-api.'
