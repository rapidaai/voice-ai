#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
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

trap cleanup_project_fixture EXIT HUP INT TERM
seed_project_fixture

timeout 30s node "$script_directory/assistant-lifecycle.js"
verify_assistant_lifecycle_fixture
