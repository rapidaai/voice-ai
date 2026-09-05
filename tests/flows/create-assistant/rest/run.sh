#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100103'
export FLOW_PROJECT_ID='8101103'
export FLOW_EMAIL='ci-flow-create-assistant-rest@example.invalid'
export FLOW_TOKEN='ci-flow-create-assistant-rest-token'
export FLOW_API_KEY='ci-flow-create-assistant-rest-api-key'
export FLOW_ORGANIZATION='CI Create Assistant REST Organization'
export FLOW_PROJECT='CI Create Assistant REST Project'
export FLOW_ASSISTANT='CI REST Assistant'

assistant_endpoint=${ASSISTANT_API_REST_ENDPOINT:-http://assistant-api:9007}
temporary_directory=$(mktemp -d)

cleanup() {
  cleanup_project_fixture
  rm -rf "$temporary_directory"
}

request() {
  auth_mode=$1
  body=$2
  response_file=$3
  set -- --header 'Content-Type: application/json'
  if [ "$auth_mode" = 'project-api-key' ]; then
    set -- "$@" --header "x-api-key: $FLOW_API_KEY"
  else
    set -- "$@" \
      --header "authorization: $FLOW_TOKEN" \
      --header "x-auth-id: $FLOW_FIXTURE_ID" \
      --header "x-project-id: $FLOW_PROJECT_ID"
  fi
  status=$(curl --silent --show-error --connect-timeout 2 --max-time 30 \
    --output "$response_file" --write-out '%{http_code}' \
    --request POST \
    "$@" \
    --data "$body" "$assistant_endpoint/v1/assistant/create-assistant")
  if [ "$status" != '200' ]; then
    printf 'create assistant returned HTTP %s: ' "$status" >&2
    cat "$response_file" >&2
    return 1
  fi
  jq -e '.code == 200 and .success == true and (.data.id | length > 0)' "$response_file" >/dev/null
}

run_flow() {
  auth_mode=$1
  auth_name=$2
  file_suffix=$3
  body=$(jq -n --arg name "$FLOW_ASSISTANT $auth_name" '{
    name: $name,
    description: "Created by the CI REST assistant flow",
    visibility: "private",
    language: "english",
    assistantProvider: {
      model: {modelProviderName: "openai"}
    }
  }')
  request "$auth_mode" "$body" "$temporary_directory/assistant-$file_suffix.json"
  printf 'REST create assistant flow passed with %s\n' "$auth_name"
}

trap cleanup EXIT HUP INT TERM
seed_project_fixture
seed_project_api_key_fixture

run_flow personal-access-token 'personal access token' pat
run_flow project-api-key 'project API key' api-key
verify_assistant_fixture
