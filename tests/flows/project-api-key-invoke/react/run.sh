#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../../process-lifecycle.sh"
# shellcheck disable=SC1091
. "$script_directory/../../project-fixture.sh"

export FLOW_FIXTURE_ID='8101002'
export FLOW_PROJECT_ID='8102002'
export FLOW_VAULT_ID='8103002'
export FLOW_MOCK_PORT='18162'
FLOW_MOCK_HOST=${FLOW_MOCK_HOST:-$(hostname -i | awk '{print $1}')}
FLOW_MOCK_AUTHORITY=$(format_flow_ari_host "$FLOW_MOCK_HOST")
export FLOW_MOCK_URL="http://$FLOW_MOCK_AUTHORITY:$FLOW_MOCK_PORT/v1"
export FLOW_EMAIL='ci-flow-project-key-invoke-react@example.invalid'
export FLOW_TOKEN='ci-flow-project-key-invoke-react-token'
export FLOW_API_KEY='ci-flow-project-key-invoke-react-api-key'
export FLOW_API_KEY_ROLE_KEY='automation_client'
export FLOW_ORGANIZATION='CI Project Key Invoke React Organization'
export FLOW_PROJECT='CI Project Key Invoke React Project'
export FLOW_ENDPOINT='CI Runtime Key Endpoint'
export FLOW_API_KEY_NAME='CI Runtime Invoker Key'

runtime_vault_cache_key="INTERNAL::GetCredential_${FLOW_FIXTURE_ID}__vlt__${FLOW_VAULT_ID}"

seed_runtime_vault_fixture() {
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" \
    -v vault_id="$FLOW_VAULT_ID" \
    -v mock_url="$FLOW_MOCK_URL" >/dev/null <<'SQL'
INSERT INTO vaults (
  id, organization_id, project_id, provider, name, value, status,
  created_actor_type
) VALUES (
  :'vault_id', :'fixture_id', :'project_id', 'custom-llm:Custom LLM',
  'CI Runtime Invoke Mock Provider',
  json_build_object(
    'api_compatibility', 'openai_chat_completions',
    'base_url', :'mock_url',
    'headers', json_build_object('Authorization', 'Bearer ci-flow-custom-llm')
  ),
  'ACTIVE', 'unknown'
);
SQL

  vault_cache=$(psql -v ON_ERROR_STOP=1 -At -h "${FLOW_POSTGRES_HOST:-postgres}" \
    -U rapida_user -d web_db -v vault_id="$FLOW_VAULT_ID" <<'SQL'
SELECT json_build_object(
  'id', id,
  'name', name,
  'value', value,
  'status', status,
  'provider', provider,
  'organizationId', organization_id
)::text
FROM vaults
WHERE id = :'vault_id';
SQL
)
  redis-cli -h "${FLOW_REDIS_HOST:-redis}" SET \
    "$runtime_vault_cache_key" "$vault_cache" >/dev/null
}

assert_runtime_api_key_fixture() {
  role_permissions=$(psql -v ON_ERROR_STOP=1 -At -F '|' \
    -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v project_id="$FLOW_PROJECT_ID" -v api_key="$FLOW_API_KEY" <<'SQL'
SELECT
  role_definition.key,
  count(*) FILTER (WHERE permission_definition.key = 'project.runtime.invoke'),
  count(*) FILTER (WHERE permission_definition.key = 'project.provider_credentials.use')
FROM project_credentials credential
JOIN role_definitions role_definition
  ON role_definition.id = credential.role_definition_id
JOIN role_permissions role_permission
  ON role_permission.role_definition_id = role_definition.id
 AND role_permission.status = 'ACTIVE'
JOIN permission_definitions permission_definition
  ON permission_definition.id = role_permission.permission_definition_id
 AND permission_definition.status = 'ACTIVE'
WHERE credential.project_id = :'project_id'
  AND credential.key = :'api_key'
  AND credential.status = 'ACTIVE'
GROUP BY role_definition.key;
SQL
)
  if [ "$role_permissions" != 'automation_client|1|1' ]; then
    printf 'unexpected runtime project API key role: %s\n' "$role_permissions" >&2
    return 1
  fi
}

cleanup() {
  cleanup_status=0
  redis-cli -h "${FLOW_REDIS_HOST:-redis}" DEL \
    "$runtime_vault_cache_key" >/dev/null || cleanup_status=$?
  cleanup_endpoint_project_fixture || cleanup_status=$?
  return "$cleanup_status"
}

register_flow_cleanup cleanup
cleanup_endpoint_fixture
seed_project_fixture
seed_project_api_key_fixture
seed_runtime_vault_fixture
assert_runtime_api_key_fixture

run_flow_process timeout 60s node "$script_directory/project-api-key-invoke.js"
verify_endpoint_invocation_fixture
