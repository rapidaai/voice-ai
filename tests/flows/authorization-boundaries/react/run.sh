#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"
# shellcheck disable=SC1091
. "$script_directory/../fixture.sh"

export FLOW_FIXTURE_ID='8100902'
export FLOW_PROJECT_ID='8101902'
export FLOW_EMAIL='ci-flow-authorization-owner-react@example.invalid'
export FLOW_TOKEN='ci-flow-authorization-owner-react-token'
export FLOW_ORGANIZATION='CI React Authorization Owner Organization'
export FLOW_PROJECT='CI React Authorization Owner Project'
export FLOW_OTHER_ORGANIZATION_ID='8100912'
export FLOW_OTHER_PROJECT_ID='8101912'
export FLOW_OTHER_EMAIL='ci-flow-authorization-other-react@example.invalid'
export FLOW_RESTRICTED_USER_ID='8100922'
export FLOW_RESTRICTED_EMAIL='ci-flow-authorization-restricted-react@example.invalid'
export FLOW_RESTRICTED_TOKEN='ci-flow-authorization-restricted-react-token'

cleanup() {
  cleanup_security_data
}

trap cleanup EXIT
trap 'exit 1' HUP INT TERM
cleanup_security_data
seed_project_fixture
seed_security_fixtures
timeout 30s node "$script_directory/authorization-boundaries.js"
verify_authorization_fixture
