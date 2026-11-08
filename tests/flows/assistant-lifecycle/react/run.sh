#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../process-lifecycle.sh"
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100702'
export FLOW_PROJECT_ID='8101702'
export FLOW_EMAIL='ci-flow-assistant-lifecycle-react@example.invalid'
export FLOW_TOKEN='ci-flow-assistant-lifecycle-react-token'
export FLOW_ORGANIZATION='CI Assistant Lifecycle React Organization'
export FLOW_PROJECT='CI Assistant Lifecycle React Project'
export FLOW_ASSISTANT='CI React Lifecycle Assistant'
export FLOW_EXPECTED_COUNT='1'

cleanup() {
  cleanup_project_fixture
}

register_flow_cleanup cleanup
seed_project_fixture

run_flow_process timeout 30s node "$script_directory/assistant-lifecycle.js"
verify_assistant_lifecycle_fixture
