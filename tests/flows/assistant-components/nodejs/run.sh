#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100801'
export FLOW_PROJECT_ID='8101801'
export FLOW_EMAIL='ci-flow-assistant-components-nodejs@example.invalid'
export FLOW_TOKEN='ci-flow-assistant-components-nodejs-token'
export FLOW_API_KEY='ci-flow-assistant-components-nodejs-api-key'
export FLOW_ORGANIZATION='CI Assistant Components Node Organization'
export FLOW_PROJECT='CI Assistant Components Node Project'
export FLOW_ASSISTANT='CI Node Components Assistant'
export FLOW_EXPECTED_ASSISTANTS='2'
export FLOW_EXPECTED_TOOLS='2'
export FLOW_EXPECTED_CONFIGURATIONS='5'

trap cleanup_project_fixture EXIT
trap 'exit 1' HUP INT TERM
seed_project_fixture
seed_project_api_key_fixture

timeout 60s node "$script_directory/assistant-components.js"
verify_assistant_components_fixture
