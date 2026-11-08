#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../process-lifecycle.sh"
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100802'
export FLOW_PROJECT_ID='8101802'
export FLOW_EMAIL='ci-flow-assistant-components-react@example.invalid'
export FLOW_TOKEN='ci-flow-assistant-components-react-token'
export FLOW_ORGANIZATION='CI Assistant Components React Organization'
export FLOW_PROJECT='CI Assistant Components React Project'
export FLOW_ASSISTANT='CI React Components Assistant'
export FLOW_EXPECTED_ASSISTANTS='1'
export FLOW_EXPECTED_TOOLS='1'
export FLOW_EXPECTED_CONFIGURATIONS='5'

register_flow_cleanup cleanup_project_fixture
seed_project_fixture

run_flow_process timeout 60s node "$script_directory/assistant-components.js"
verify_assistant_components_fixture
