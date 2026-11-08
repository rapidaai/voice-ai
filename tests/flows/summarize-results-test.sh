#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-flow-summary-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
results_directory="$temporary_directory/results"
definition_directory="$temporary_directory/definitions"
manifest_file="$definition_directory/manifest.txt"
summary_file="$temporary_directory/summary.md"
mkdir -p "$results_directory" "$definition_directory/alpha" "$definition_directory/beta"
printf 'alpha\nbeta' > "$manifest_file"
printf '%s\n' "run_flow_clients \"\$script_directory\" 'Alpha' nodejs react rest" \
  > "$definition_directory/alpha/run.sh"
printf '%s\n' "run_flow_clients \"\$script_directory\" 'Beta' nodejs react rest" \
  > "$definition_directory/beta/run.sh"

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
grep -Fq 'Passed: 4, failed: 1, not supported: 1, not run: 0.' "$summary_file"

mkdir -p "$results_directory/duplicate"
cp "$results_directory/node-1/flows.tsv" "$results_directory/duplicate/flows.tsv"
if FLOW_MANIFEST_FILE="$manifest_file" \
  "$script_directory/summarize-results.sh" "$results_directory" "$summary_file"; then
  printf 'duplicate flow result unexpectedly passed\n' >&2
  exit 1
fi
grep -Fq 'Duplicate flow/client results were found.' "$summary_file"

rm -rf "$results_directory/duplicate" "$results_directory/rest-2"
if FLOW_MANIFEST_FILE="$manifest_file" \
  "$script_directory/summarize-results.sh" "$results_directory" "$summary_file"; then
  printf 'missing flow result unexpectedly passed\n' >&2
  exit 1
fi
grep -Fq '| Beta | rest | ⚠️ Not run | - |' "$summary_file"
grep -Fq 'Passed: 4, failed: 1, not supported: 0, not run: 1.' "$summary_file"

rm -rf "$results_directory"
mkdir -p "$results_directory"
if FLOW_MANIFEST_FILE="$manifest_file" \
  "$script_directory/summarize-results.sh" "$results_directory" "$summary_file"; then
  printf 'empty flow results unexpectedly passed\n' >&2
  exit 1
fi
grep -Fq '| Alpha | nodejs | ⚠️ Not run | - |' "$summary_file"
grep -Fq '| Beta | rest | ⚠️ Not run | - |' "$summary_file"
grep -Fq 'Passed: 0, failed: 0, not supported: 0, not run: 6.' "$summary_file"

rerun_results_directory="$temporary_directory/rerun-results"
write_rerun_result() {
  local artifact=$1
  shift
  mkdir -p "$rerun_results_directory/$artifact"
  printf '%s\n' "$@" > "$rerun_results_directory/$artifact/flows.tsv"
}
write_rerun_result flow-results-node-1-42-1 \
  $'Alpha\tnodejs\tfailed\t1'
write_rerun_result flow-results-node-1-42-2 \
  $'Alpha\tnodejs\tpassed\t2' $'Beta\tnodejs\tpassed\t2'
write_rerun_result flow-results-react-1-42-1 \
  $'Alpha\treact\tpassed\t1' $'Beta\treact\tpassed\t1'
write_rerun_result flow-results-rest-1-42-1 \
  $'Alpha\trest\tpassed\t1' $'Beta\trest\tpassed\t1'
FLOW_MANIFEST_FILE="$manifest_file" \
  "$script_directory/summarize-results.sh" "$rerun_results_directory" "$summary_file"
grep -Fq '| Alpha | nodejs | ✅ Passed | 2s |' "$summary_file"
grep -Fq 'Passed: 6, failed: 0, not supported: 0, not run: 0.' "$summary_file"

printf 'Flow summary regression tests passed\n'
