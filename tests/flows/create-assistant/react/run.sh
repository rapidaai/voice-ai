#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100102'
export FLOW_PROJECT_ID='8101102'
export FLOW_EMAIL='ci-flow-create-assistant-react@example.invalid'
export FLOW_TOKEN='ci-flow-create-assistant-react-token'
export FLOW_API_KEY='ci-flow-create-assistant-react-api-key'
export FLOW_ORGANIZATION='CI Create Assistant React Organization'
export FLOW_PROJECT='CI Create Assistant React Project'
export FLOW_ASSISTANT='CI React SDK Assistant'

trap cleanup_project_fixture EXIT HUP INT TERM
seed_project_fixture
seed_project_api_key_fixture

timeout 30s node "$script_directory/create-assistant.js"
verify_assistant_fixture
