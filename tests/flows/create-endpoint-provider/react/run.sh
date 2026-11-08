#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../process-lifecycle.sh"
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100502'
export FLOW_PROJECT_ID='8101502'
export FLOW_EMAIL='ci-flow-create-endpoint-react@example.invalid'
export FLOW_TOKEN='ci-flow-create-endpoint-react-token'
export FLOW_ORGANIZATION='CI Create Endpoint React Organization'
export FLOW_PROJECT='CI Create Endpoint React Project'
export FLOW_ENDPOINT='CI React Endpoint'
export FLOW_EXPECTED_COUNT='20'

cleanup() {
  cleanup_endpoint_project_fixture
}

register_flow_cleanup cleanup
cleanup_endpoint_fixture
seed_project_fixture

run_flow_process timeout 120s node "$script_directory/create-endpoint-provider.js"
verify_endpoint_provider_fixture
