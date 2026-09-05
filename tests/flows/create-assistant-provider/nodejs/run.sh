#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100201'
export FLOW_PROJECT_ID='8100201'
export FLOW_EMAIL='ci-flow-create-provider-nodejs@example.invalid'
export FLOW_TOKEN='ci-flow-create-provider-nodejs-token'
export FLOW_ORGANIZATION='CI Create Provider Node Organization'
export FLOW_PROJECT='CI Create Provider Node Project'
export FLOW_ASSISTANT='CI Node Provider Assistant'

trap cleanup_project_fixture EXIT HUP INT TERM
seed_project_fixture

timeout 30s node "$script_directory/create-assistant-provider.js"
