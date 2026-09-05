#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100203'
export FLOW_PROJECT_ID='8101203'
export FLOW_EMAIL='ci-flow-create-provider-rest@example.invalid'
export FLOW_TOKEN='ci-flow-create-provider-rest-token'
export FLOW_API_KEY='ci-flow-create-provider-rest-api-key'
export FLOW_ORGANIZATION='CI Create Provider REST Organization'
export FLOW_PROJECT='CI Create Provider REST Project'
export FLOW_ASSISTANT='CI REST Provider Assistant'
export FLOW_EXPECTED_ASSISTANTS='10'
export FLOW_EXPECTED_PROVIDERS='2'

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
    printf 'create assistant provider returned HTTP %s: ' "$status" >&2
    cat "$response_file" >&2
    return 1
  fi
  jq -e '.code == 200 and .success == true and (.data.id | length > 0)' "$response_file" >/dev/null
}

create_provider_assistant() {
  auth_mode=$1
  auth_name=$2
  provider_name=$3
  provider=$4
  file_suffix=$5
  body=$(jq -n \
    --arg name "$FLOW_ASSISTANT $auth_name $provider_name" \
    --argjson provider "$provider" '{
      name: $name,
      description: "Created by the CI REST assistant provider flow",
      visibility: "private",
      language: "english",
      assistantProvider: $provider
    }')
  request "$auth_mode" "$body" "$temporary_directory/$file_suffix.json"
}

run_flow() {
  auth_mode=$1
  auth_name=$2
  file_suffix=$3

  create_provider_assistant "$auth_mode" "$auth_name" openai \
    '{"model":{"modelProviderName":"openai"}}' "$file_suffix-openai"
  create_provider_assistant "$auth_mode" "$auth_name" anthropic \
    '{"model":{"modelProviderName":"anthropic"}}' "$file_suffix-anthropic"
  create_provider_assistant "$auth_mode" "$auth_name" AgentKit \
    '{"agentkit":{"agentkitUrl":"agentkit:50051","transportSecurity":"PLAINTEXT"}}' "$file_suffix-agentkit"
  create_provider_assistant "$auth_mode" "$auth_name" WebSocket \
    '{"websocket":{"websocketUrl":"wss://example.invalid/agent"}}' "$file_suffix-websocket"
  create_provider_assistant "$auth_mode" "$auth_name" AgentFlow \
    '{"agentflow":{"schemaVersion":"1.0","definition":{"entryNodeId":"start","nodes":[{"id":"start"}],"edges":[]}}}' "$file_suffix-agentflow"

  printf 'REST create assistant provider flow passed with %s\n' "$auth_name"
}

trap cleanup EXIT HUP INT TERM
seed_project_fixture
seed_project_api_key_fixture

run_flow personal-access-token 'personal access token' pat
run_flow project-api-key 'project API key' api-key
verify_assistant_provider_fixture
