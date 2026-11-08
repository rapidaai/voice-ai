#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-flow-summary-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
results_directory="$temporary_directory/results"
manifest_file="$temporary_directory/manifest.txt"
summary_file="$temporary_directory/summary.md"
mkdir -p "$results_directory"
printf '%s\n' alpha beta > "$manifest_file"

write_result() {
  local artifact=$1
  local flow=$2
  local client=$3
  local result=$4
  local duration=$5
  mkdir -p "$results_directory/$artifact"
  printf '%s\t%s\t%s\t%s\n' "$flow" "$client" "$result" "$duration" \
    > "$results_directory/$artifact/flows.tsv"
}

write_result node-1 Alpha nodejs passed 1
write_result node-2 Beta nodejs failed 2
write_result react-1 Alpha react passed 3
write_result react-2 Beta react passed 4
write_result rest-1 Alpha rest passed 5
write_result rest-2 Beta rest skipped 0

FLOW_MANIFEST_FILE="$manifest_file" \
  "$script_directory/summarize-results.sh" "$results_directory" "$summary_file"
grep -Fq '| Alpha | react | ✅ Passed | 3s |' "$summary_file"
grep -Fq '| Beta | nodejs | ❌ Failed | 2s |' "$summary_file"
grep -Fq '| Beta | rest | ⏭️ Not supported | 0s |' "$summary_file"
grep -Fq 'Passed: 4, failed: 1, not supported: 1.' "$summary_file"

mkdir -p "$results_directory/duplicate"
cp "$results_directory/node-1/flows.tsv" "$results_directory/duplicate/flows.tsv"
if FLOW_MANIFEST_FILE="$manifest_file" \
  "$script_directory/summarize-results.sh" "$results_directory" "$summary_file"; then
  printf 'duplicate flow result unexpectedly passed\n' >&2
  exit 1
fi
grep -Fq 'Duplicate flow/client results were found.' "$summary_file"

printf 'Flow summary regression tests passed\n'
