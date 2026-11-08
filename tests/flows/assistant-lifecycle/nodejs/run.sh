#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../process-lifecycle.sh"
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100701'
export FLOW_PROJECT_ID='8101701'
export FLOW_EMAIL='ci-flow-assistant-lifecycle-nodejs@example.invalid'
export FLOW_TOKEN='ci-flow-assistant-lifecycle-nodejs-token'
export FLOW_API_KEY='ci-flow-assistant-lifecycle-nodejs-api-key'
export FLOW_ORGANIZATION='CI Assistant Lifecycle Node Organization'
export FLOW_PROJECT='CI Assistant Lifecycle Node Project'
export FLOW_ASSISTANT='CI Node Lifecycle Assistant'

cleanup() {
  cleanup_project_fixture
}

register_flow_cleanup cleanup
seed_project_fixture
seed_project_api_key_fixture

run_flow_process timeout 30s node "$script_directory/assistant-lifecycle.js"
verify_assistant_lifecycle_fixture
