#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8100403'
export FLOW_PROJECT_ID='8101403'
export FLOW_VAULT_ID='8102403'
export FLOW_ARI_PORT='18103'
FLOW_ARI_HOST=$(hostname -i | awk '{print $1}')
export FLOW_ARI_URL="http://$FLOW_ARI_HOST:$FLOW_ARI_PORT"
export FLOW_EMAIL='ci-flow-phone-call-rest@example.invalid'
export FLOW_TOKEN='ci-flow-phone-call-rest-token'
export FLOW_API_KEY='ci-flow-phone-call-rest-api-key'
export FLOW_ORGANIZATION='CI Phone Call REST Organization'
export FLOW_PROJECT='CI Phone Call REST Project'
export FLOW_ASSISTANT='CI REST Phone Assistant'
export FLOW_FROM_NUMBER='ci-rest-caller'
export FLOW_TO_NUMBER='ci-rest-destination'

assistant_endpoint=${ASSISTANT_API_REST_ENDPOINT:-http://assistant-api:9007}
temporary_directory=$(mktemp -d)
mock_pid=''

cleanup() {
  if [ -n "$mock_pid" ]; then
    kill "$mock_pid" 2>/dev/null || true
    wait "$mock_pid" 2>/dev/null || true
  fi
  cleanup_project_fixture
  rm -rf "$temporary_directory"
}

request() {
  auth_mode=$1
  method=$2
  path=$3
  body=$4
  response_file=$5
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
    --request "$method" \
    "$@" \
    --data "$body" "$assistant_endpoint$path")
  if [ "$status" != '200' ]; then
    printf '%s returned HTTP %s: ' "$path" "$status" >&2
    cat "$response_file" >&2
    return 1
  fi
  jq -e '.code == 200 and .success == true' "$response_file" >/dev/null
}

run_flow() {
  auth_mode=$1
  auth_name=$2
  file_suffix=$3

  assistant_response="$temporary_directory/assistant-$file_suffix.json"
  assistant_body=$(jq -n --arg name "$FLOW_ASSISTANT $auth_name" '{
  name: $name,
  description: "Created by the CI REST phone flow",
  visibility: "private",
  language: "english",
  assistantProvider: {
    model: {modelProviderName: "openai"}
  }
}')
  request "$auth_mode" POST '/v1/assistant/create-assistant' "$assistant_body" "$assistant_response"
  assistant_id=$(jq -er '.data.id' "$assistant_response")

  deployment_response="$temporary_directory/deployment-$file_suffix.json"
  deployment_body=$(jq -n \
  --arg assistant_id "$assistant_id" \
  --arg vault_id "$FLOW_VAULT_ID" '{
    assistantId: $assistant_id,
    greeting: "Hello from the CI phone flow",
    idealTimeout: 30,
    idealTimeoutBackoff: 2,
    maxSessionDuration: 180,
    phoneProviderName: "asterisk",
    phoneOptions: [{key: "rapida.credential_id", value: $vault_id}]
  }')
  request "$auth_mode" POST '/v1/assistant-deployment/create-phone-deployment' "$deployment_body" "$deployment_response"
  jq -e --arg assistant_id "$assistant_id" '.data.assistantId == $assistant_id and .data.phoneProviderName == "asterisk"' \
    "$deployment_response" >/dev/null

  call_response="$temporary_directory/call-$file_suffix.json"
  call_body=$(jq -n \
  --arg assistant_id "$assistant_id" \
  --arg from_number "$FLOW_FROM_NUMBER" \
  --arg to_number "$FLOW_TO_NUMBER" '{
    assistant: {assistantId: $assistant_id, version: "latest"},
    fromNumber: $from_number,
    toNumber: $to_number
  }')
  request "$auth_mode" POST '/v1/talk/create-phone-call' "$call_body" "$call_response"
  jq -e '.data.id | length > 0' "$call_response" >/dev/null

  printf 'REST assistant deployment phone call flow passed with %s\n' "$auth_name"
}

trap cleanup EXIT HUP INT TERM
seed_project_fixture
seed_project_api_key_fixture
seed_asterisk_vault_fixture
node "$script_directory/../mock-ari-server.js" "$FLOW_ARI_PORT" &
mock_pid=$!
sleep 1

run_flow personal-access-token 'personal access token' pat
run_flow project-api-key 'project API key' api-key

verify_assistant_phone_call_fixture
