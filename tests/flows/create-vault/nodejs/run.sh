#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100301'
export FLOW_PROJECT_ID='8101301'
export FLOW_EMAIL='ci-flow-create-vault-nodejs@example.invalid'
export FLOW_TOKEN='ci-flow-create-vault-nodejs-token'
export FLOW_ORGANIZATION='CI Create Vault Node Organization'
export FLOW_PROJECT='CI Create Vault Node Project'
export FLOW_VAULT_NAME='CI Node SDK OpenAI Key'

trap cleanup_project_fixture EXIT HUP INT TERM
seed_project_fixture

timeout 30s node "$script_directory/create-vault.js"
