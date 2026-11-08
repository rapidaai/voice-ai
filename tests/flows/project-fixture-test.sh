#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/project-fixture.sh"

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

printf 'Project fixture regression tests passed\n'
