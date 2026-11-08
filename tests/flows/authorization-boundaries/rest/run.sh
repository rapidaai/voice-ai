#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../process-lifecycle.sh"
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"
# shellcheck disable=SC1091
. "$script_directory/../fixture.sh"

export FLOW_FIXTURE_ID='8100903'
export FLOW_PROJECT_ID='8101903'
export FLOW_EMAIL='ci-flow-authorization-owner@example.invalid'
export FLOW_TOKEN='ci-flow-authorization-owner-token'
export FLOW_ORGANIZATION='CI Authorization Owner Organization'
export FLOW_PROJECT='CI Authorization Owner Project'
export FLOW_OTHER_ORGANIZATION_ID='8100904'
export FLOW_OTHER_PROJECT_ID='8101904'
export FLOW_OTHER_EMAIL='ci-flow-authorization-other@example.invalid'
export FLOW_RESTRICTED_USER_ID='8100905'
export FLOW_RESTRICTED_EMAIL='ci-flow-authorization-restricted@example.invalid'
export FLOW_RESTRICTED_TOKEN='ci-flow-authorization-restricted-token'

assistant_endpoint=${ASSISTANT_API_REST_ENDPOINT:-http://assistant-api:9007}
temporary_directory=''

cleanup() {
  trap '' HUP INT TERM
  cleanup_status=0
  cleanup_security_data || cleanup_status=$?
  if [ -n "$temporary_directory" ]; then
    rm -rf "$temporary_directory" || cleanup_status=$?
  fi
  return "$cleanup_status"
}

register_flow_cleanup cleanup
temporary_directory=$(mktemp -d)

expect_denied() {
  label=$1
  output_file=$2
  shift 2
  status=$(curl --silent --show-error --connect-timeout 2 --max-time 15 \
    --output "$output_file" --write-out '%{http_code}' "$@")
  case "$status" in
    401 | 403) ;;
    *)
      printf '%s returned HTTP %s instead of 401 or 403: ' "$label" "$status" >&2
      cat "$output_file" >&2
      return 1
      ;;
  esac
  printf '%s was rejected with HTTP %s\n' "$label" "$status"
}

expect_invalid_identifier() {
  output_file=$1
  status=$(curl --silent --show-error --connect-timeout 2 --max-time 15 \
    --output "$output_file" --write-out '%{http_code}' \
    --header "authorization: $FLOW_TOKEN" \
    --header "x-auth-id: $FLOW_FIXTURE_ID" \
    --header "x-project-id: $FLOW_PROJECT_ID" \
    "$assistant_endpoint/v1/assistant/configurations/not-a-number")
  if [ "$status" != '400' ]; then
    printf 'invalid resource identifier returned HTTP %s: ' "$status" >&2
    cat "$output_file" >&2
    return 1
  fi
  printf 'invalid resource identifier was rejected with HTTP 400\n'
}

assistant_body=$(jq -n '{
  name: "Rejected authorization request",
  visibility: "private",
  language: "english",
  assistantProvider: {model: {modelProviderName: "openai"}}
}')

run_authorization_checks() {
  expect_denied 'invalid personal access token' "$temporary_directory/invalid-token.json" \
    --request POST --header 'Content-Type: application/json' \
    --header 'authorization: invalid-personal-access-token' \
    --header "x-auth-id: $FLOW_FIXTURE_ID" \
    --header "x-project-id: $FLOW_PROJECT_ID" \
    --data "$assistant_body" "$assistant_endpoint/v1/assistant/create-assistant"

  expect_denied 'invalid project API key' "$temporary_directory/invalid-api-key.json" \
    --request POST --header 'Content-Type: application/json' \
    --header 'x-api-key: invalid' \
    --data "$assistant_body" "$assistant_endpoint/v1/assistant/create-assistant"

  expect_denied 'cross-project access' "$temporary_directory/cross-project.json" \
    --request POST --header 'Content-Type: application/json' \
    --header "authorization: $FLOW_TOKEN" \
    --header "x-auth-id: $FLOW_FIXTURE_ID" \
    --header "x-project-id: $FLOW_OTHER_PROJECT_ID" \
    --data "$assistant_body" "$assistant_endpoint/v1/assistant/create-assistant"

  expect_denied 'user without project permission' "$temporary_directory/missing-permission.json" \
    --request POST --header 'Content-Type: application/json' \
    --header "authorization: $FLOW_RESTRICTED_TOKEN" \
    --header "x-auth-id: $FLOW_RESTRICTED_USER_ID" \
    --header "x-project-id: $FLOW_PROJECT_ID" \
    --data "$assistant_body" "$assistant_endpoint/v1/assistant/create-assistant"

  expect_invalid_identifier "$temporary_directory/invalid-identifier.json"
}

cleanup_security_data
seed_project_fixture
seed_security_fixtures
run_authorization_checks
verify_authorization_fixture
printf 'REST authorization boundary flow passed\n'
