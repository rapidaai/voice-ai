#!/usr/bin/env bash
set -Eeuo pipefail

readonly results_root=${1:?usage: summarize-results.sh <results-root> <summary-file>}
readonly summary_file=${2:?usage: summarize-results.sh <results-root> <summary-file>}
script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
readonly manifest_file=${FLOW_MANIFEST_FILE:-$script_directory/manifest.txt}
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-flow-summary.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
combined_results="$temporary_directory/flows.tsv"
sorted_results="$temporary_directory/flows-sorted.tsv"
result_files="$temporary_directory/result-files"
status=0

if [[ -d "$results_root" ]]; then
  find "$results_root" -type f -name flows.tsv -print0 | sort -z > "$result_files"
else
  : > "$result_files"
fi
while IFS= read -r -d '' result_file; do
  cat "$result_file" >> "$combined_results"
done < "$result_files"
touch "$combined_results"

expected_rows=$(($(wc -l < "$manifest_file") * 3))
actual_rows=$(wc -l < "$combined_results")
duplicate_rows=$(cut -f1,2 "$combined_results" | sort | uniq -d)
invalid_rows=$(awk -F '\t' 'NF != 4 || ($3 != "passed" && $3 != "failed" && $3 != "skipped") || $4 !~ /^[0-9]+$/ {print NR}' "$combined_results")

if [[ -n "$duplicate_rows" || -n "$invalid_rows" || "$actual_rows" -ne "$expected_rows" ]]; then
  status=1
fi

sort -t $'\t' -k1,1 -k2,2 "$combined_results" > "$sorted_results"
passed=0
failed=0
skipped=0

{
  printf '## End-to-End Flow Results\n\n'
  if [[ "$actual_rows" -ne "$expected_rows" ]]; then
    printf '> ⚠️ Expected %d flow/client results but found %d.\n\n' "$expected_rows" "$actual_rows"
  fi
  if [[ -n "$duplicate_rows" ]]; then
    printf '> ⚠️ Duplicate flow/client results were found.\n\n'
  fi
  if [[ -n "$invalid_rows" ]]; then
    printf '> ⚠️ Invalid result rows were found at combined line(s): %s.\n\n' "$invalid_rows"
  fi
  printf '| Flow | Client | Status | Duration |\n'
  printf '| --- | --- | --- | ---: |\n'
  while IFS=$'\t' read -r flow client result duration_seconds; do
    [[ -n "$flow" ]] || continue
    case "$result" in
      passed) label='✅ Passed'; passed=$((passed + 1)) ;;
      failed) label='❌ Failed'; failed=$((failed + 1)) ;;
      skipped) label='⏭️ Not supported'; skipped=$((skipped + 1)) ;;
      *) label='⚠️ Invalid' ;;
    esac
    printf '| %s | %s | %s | %ss |\n' "$flow" "$client" "$label" "$duration_seconds"
  done < "$sorted_results"
  printf '\nPassed: %d, failed: %d, not supported: %d.\n' "$passed" "$failed" "$skipped"
} > "$summary_file"

exit "$status"
