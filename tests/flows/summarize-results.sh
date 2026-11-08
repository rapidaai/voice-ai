#!/usr/bin/env bash
set -Eeuo pipefail

readonly results_root=${1:?usage: summarize-results.sh <results-root> <summary-file>}
readonly summary_file=${2:?usage: summarize-results.sh <results-root> <summary-file>}
script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
readonly manifest_file=${FLOW_MANIFEST_FILE:-$script_directory/manifest.txt}
readonly flow_definition_root=${FLOW_DEFINITION_ROOT:-$(dirname "$manifest_file")}
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-flow-summary.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
combined_results="$temporary_directory/flows.tsv"
expected_results="$temporary_directory/flows-expected.tsv"
merged_results="$temporary_directory/flows-merged.tsv"
result_files="$temporary_directory/result-files"
all_result_files="$temporary_directory/all-result-files"
artifact_candidates="$temporary_directory/artifact-candidates"
status=0

if [[ -d "$results_root" ]]; then
  find "$results_root" -type f -name flows.tsv -print0 | sort -z > "$all_result_files"
else
  : > "$all_result_files"
fi
: > "$result_files"
: > "$artifact_candidates"
while IFS= read -r -d '' result_file; do
  relative_path=${result_file#"$results_root"/}
  artifact_name=${relative_path%%/*}
  if [[ "$artifact_name" =~ ^flow-results-(node|react|rest)-([0-9]+)-([0-9]+)-([0-9]+)$ ]]; then
    printf '%s-%s\t%s\t%s\n' \
      "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[4]}" "$result_file" \
      >> "$artifact_candidates"
  else
    printf '%s\n' "$result_file" >> "$result_files"
  fi
done < "$all_result_files"
if [[ -s "$artifact_candidates" ]]; then
  sort -t $'\t' -k1,1 -k2,2nr "$artifact_candidates" | \
    awk -F '\t' '!selected[$1]++ {print $3}' >> "$result_files"
fi
sort -o "$result_files" "$result_files"
while IFS= read -r -d '' result_file; do
  cat "$result_file" >> "$combined_results"
done < <(while IFS= read -r result_file; do printf '%s\0' "$result_file"; done < "$result_files")
touch "$combined_results"

while IFS= read -r journey || [[ -n "$journey" ]]; do
  declaration=$(grep -E "^run_flow_clients \"\\\$script_directory\" '[^']+'( (nodejs|react|rest))+$" \
    "$flow_definition_root/$journey/run.sh")
  flow_name=$(sed -E "s/^run_flow_clients \"\\\$script_directory\" '([^']+)'.*/\1/" <<< "$declaration")
  for client in nodejs react rest; do
    printf '%s\t%s\n' "$flow_name" "$client" >> "$expected_results"
  done
done < "$manifest_file"

expected_rows=$(wc -l < "$expected_results")
actual_rows=$(wc -l < "$combined_results")
duplicate_rows=$(cut -f1,2 "$combined_results" | sort | uniq -d)
invalid_rows=$(awk -F '\t' 'NF != 4 || ($3 != "passed" && $3 != "failed" && $3 != "skipped") || $4 !~ /^[0-9]+$/ {print NR}' "$combined_results")
unexpected_rows=$(awk -F '\t' '
  phase != 1 { expected[$1 SUBSEP $2] = 1; next }
  !(($1 SUBSEP $2) in expected) { print $1 " / " $2 }
' "$expected_results" phase=1 "$combined_results")

awk -F '\t' -v OFS='\t' '
  phase != 1 { result[$1 SUBSEP $2] = $3 OFS $4; next }
  {
    key = $1 SUBSEP $2
    if (key in result) {
      print $1, $2, result[key]
    } else {
      print $1, $2, "missing", "-"
    }
  }
' "$combined_results" phase=1 "$expected_results" > "$merged_results"
missing_rows=$(awk -F '\t' '$3 == "missing" {count++} END {print count + 0}' "$merged_results")

if [[ -n "$duplicate_rows" || -n "$invalid_rows" || -n "$unexpected_rows" || \
  "$actual_rows" -ne "$expected_rows" || "$missing_rows" -ne 0 ]]; then
  status=1
fi

passed=0
failed=0
skipped=0
missing=0

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
  if [[ -n "$unexpected_rows" ]]; then
    printf '> ⚠️ Unexpected flow/client results were found: %s.\n\n' "$unexpected_rows"
  fi
  if [[ "$missing_rows" -ne 0 ]]; then
    printf '> ⚠️ %d flow/client result(s) did not run or report a status.\n\n' "$missing_rows"
  fi
  printf '| Flow | Client | Status | Duration |\n'
  printf '| --- | --- | --- | ---: |\n'
  while IFS=$'\t' read -r flow client result duration_seconds; do
    [[ -n "$flow" ]] || continue
    case "$result" in
      passed) label='✅ Passed'; passed=$((passed + 1)) ;;
      failed) label='❌ Failed'; failed=$((failed + 1)) ;;
      skipped) label='⏭️ Not supported'; skipped=$((skipped + 1)) ;;
      missing) label='⚠️ Not run'; missing=$((missing + 1)) ;;
      *) label='⚠️ Invalid' ;;
    esac
    if [[ "$duration_seconds" == '-' ]]; then
      printf '| %s | %s | %s | - |\n' "$flow" "$client" "$label"
    else
      printf '| %s | %s | %s | %ss |\n' "$flow" "$client" "$label" "$duration_seconds"
    fi
  done < "$merged_results"
  printf '\nPassed: %d, failed: %d, not supported: %d, not run: %d.\n' \
    "$passed" "$failed" "$skipped" "$missing"
} > "$summary_file"

exit "$status"
