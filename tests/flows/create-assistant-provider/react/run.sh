#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../process-lifecycle.sh"
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100202'
export FLOW_PROJECT_ID='8101202'
export FLOW_EMAIL='ci-flow-create-provider-react@example.invalid'
export FLOW_TOKEN='ci-flow-create-provider-react-token'
export FLOW_API_KEY='ci-flow-create-provider-react-api-key'
export FLOW_ORGANIZATION='CI Create Provider React Organization'
export FLOW_PROJECT='CI Create Provider React Project'
export FLOW_ASSISTANT='CI React Provider Assistant'

register_flow_cleanup cleanup_project_fixture
seed_project_fixture
seed_project_api_key_fixture

run_flow_process timeout 30s node "$script_directory/create-assistant-provider.js"
verify_assistant_provider_fixture
