#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/project-fixture.sh"
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-project-fixture-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
database_calls="$temporary_directory/database-calls"
psql_calls="$temporary_directory/psql-calls"

[[ $(format_flow_ari_host '127.0.0.1') == '127.0.0.1' ]]
[[ $(format_flow_ari_host '2001:db8::1') == '[2001:db8::1]' ]]
[[ $(format_flow_ari_host '[2001:db8::1]') == '[2001:db8::1]' ]]

endpoint_cleanup=$(declare -f cleanup_endpoint_fixture)
project_cleanup=$(declare -f cleanup_project_fixture)
grep -Fq -- '-d endpoint_db' <<< "$endpoint_cleanup"
if grep -Fq -- '-d endpoint_db' <<< "$project_cleanup"; then
  printf 'project cleanup unexpectedly depends on endpoint_db\n' >&2
  exit 1
fi

psql() {
  local database=''
  local sql
  printf '%s\n' "$*" >> "$psql_calls"
  while [[ "$#" -gt 0 ]]; do
    if [[ "$1" == '-d' ]]; then
      database=$2
      shift 2
    else
      shift
    fi
  done
  sql=$(</dev/stdin)
  printf '%s\n' "$sql" >> "$psql_calls"
  if [[ "$sql" == *'AS unexpected;'* ]]; then
    printf '%s\n' "$FLOW_OWNERSHIP_COUNT"
    return
  fi
  printf '%s\n' "$database" >> "$database_calls"
  if [[ "$database" == 'assistant_db' && "$FLOW_FAIL_ASSISTANT" == true ]] || \
    [[ "$database" == 'endpoint_db' && "$FLOW_FAIL_ENDPOINT" == true ]]; then
    return 1
  fi
}

export FLOW_FIXTURE_ID='9900001'
export FLOW_PROJECT_ID='9901001'
export FLOW_EMAIL='fixture-test@example.invalid'
export FLOW_ORGANIZATION='Fixture Test Organization'
export FLOW_PROJECT='Fixture Test Project'
FLOW_FAIL_ASSISTANT=false
FLOW_FAIL_ENDPOINT=false
FLOW_OWNERSHIP_COUNT=1
if cleanup_project_fixture 2>/dev/null; then
  printf 'cleanup unexpectedly accepted records owned by another project\n' >&2
  exit 1
fi
[[ ! -s "$database_calls" ]]

FLOW_OWNERSHIP_COUNT=0
FLOW_FAIL_ASSISTANT=true
if cleanup_project_fixture; then
  printf 'cleanup unexpectedly ignored an assistant database failure\n' >&2
  exit 1
fi
grep -Fqx 'assistant_db' "$database_calls"
grep -Fqx 'web_db' "$database_calls"

: > "$database_calls"
FLOW_FAIL_ASSISTANT=false
FLOW_FAIL_ENDPOINT=true
if cleanup_endpoint_project_fixture; then
  printf 'cleanup unexpectedly ignored an endpoint database failure\n' >&2
  exit 1
fi
grep -Fqx 'endpoint_db' "$database_calls"
grep -Fqx 'assistant_db' "$database_calls"
grep -Fqx 'web_db' "$database_calls"

: > "$database_calls"
: > "$psql_calls"
FLOW_FAIL_ENDPOINT=false
export FLOW_TOKEN='fixture-test-token'
export FLOW_ORGANIZATION_DESCRIPTION='Custom organization fixture'
export FLOW_PROJECT_DESCRIPTION='Custom project fixture'
seed_project_fixture
[[ $(grep -Fc 'organization_description=Custom organization fixture' "$psql_calls") -eq 2 ]]
[[ $(grep -Fc 'project_description=Custom project fixture' "$psql_calls") -eq 2 ]]
[[ $(grep -Fc ":'organization_description'" "$psql_calls") -eq 2 ]]
[[ $(grep -Fc ":'project_description'" "$psql_calls") -eq 2 ]]

printf 'Project fixture regression tests passed\n'
