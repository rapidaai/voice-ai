#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100402'
export FLOW_PROJECT_ID='8101402'
export FLOW_VAULT_ID='8102402'
export FLOW_ARI_PORT='18102'
FLOW_ARI_HOST=$(hostname -i | awk '{print $1}')
export FLOW_ARI_URL="http://$FLOW_ARI_HOST:$FLOW_ARI_PORT"
export FLOW_EMAIL='ci-flow-phone-call-react@example.invalid'
export FLOW_TOKEN='ci-flow-phone-call-react-token'
export FLOW_API_KEY='ci-flow-phone-call-react-api-key'
export FLOW_ORGANIZATION='CI Phone Call React Organization'
export FLOW_PROJECT='CI Phone Call React Project'
export FLOW_ASSISTANT='CI React Phone Assistant'
export FLOW_FROM_NUMBER='ci-react-caller'
export FLOW_TO_NUMBER='ci-react-destination'

trap cleanup_project_fixture EXIT HUP INT TERM
seed_project_fixture
seed_project_api_key_fixture
seed_asterisk_vault_fixture

timeout 30s node "$script_directory/create-assistant-deployment-phone-call.js"
verify_assistant_phone_call_fixture
