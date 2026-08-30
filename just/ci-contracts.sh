#!/usr/bin/env bash
set -Eeuo pipefail

baseline_directory=${1:-}
temporary_directory=''

if [[ -z "$baseline_directory" ]]; then
  temporary_directory=$(mktemp -d "${RUNNER_TEMP:-/tmp}/contract-baseline.XXXXXX")
  baseline_directory=$temporary_directory
  scripts/contracts/materialize-baseline.sh "$baseline_directory"
fi

trap 'if [[ -n "$temporary_directory" ]]; then rm -rf "$temporary_directory"; fi' EXIT

go test ./scripts/contracts/protobuf-descriptor
bash -n scripts/contracts/check-compatibility.sh scripts/contracts/materialize-baseline.sh
bash just/shellcheck.sh scripts/contracts/check-compatibility.sh scripts/contracts/materialize-baseline.sh \
  scripts/contracts/bin/buf scripts/contracts/bin/oasdiff
scripts/contracts/check-compatibility.sh "$baseline_directory"
