#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100101'
export FLOW_PROJECT_ID='8101101'
export FLOW_EMAIL='ci-flow-create-assistant-nodejs@example.invalid'
export FLOW_TOKEN='ci-flow-create-assistant-nodejs-token'
export FLOW_API_KEY='ci-flow-create-assistant-nodejs-api-key'
export FLOW_ORGANIZATION='CI Create Assistant Node Organization'
export FLOW_PROJECT='CI Create Assistant Node Project'
export FLOW_ASSISTANT='CI Node SDK Assistant'

trap cleanup_project_fixture EXIT HUP INT TERM
seed_project_fixture
seed_project_api_key_fixture

timeout 30s node "$script_directory/create-assistant.js"
verify_assistant_fixture
