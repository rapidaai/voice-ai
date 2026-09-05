#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100302'
export FLOW_PROJECT_ID='8101302'
export FLOW_EMAIL='ci-flow-create-vault-react@example.invalid'
export FLOW_TOKEN='ci-flow-create-vault-react-token'
export FLOW_ORGANIZATION='CI Create Vault React Organization'
export FLOW_PROJECT='CI Create Vault React Project'
export FLOW_VAULT_NAME='CI React SDK OpenAI Key'

trap cleanup_project_fixture EXIT HUP INT TERM
seed_project_fixture

timeout 30s node "$script_directory/create-vault.js"
