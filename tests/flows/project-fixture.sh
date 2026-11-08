#!/bin/sh

assert_project_fixture_ownership() {
  unexpected_records=$(psql -v ON_ERROR_STOP=1 -At -h "${FLOW_POSTGRES_HOST:-postgres}" \
    -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" \
    -v email="$FLOW_EMAIL" \
    -v organization="$FLOW_ORGANIZATION" \
    -v project="$FLOW_PROJECT" \
    -v organization_description="${FLOW_ORGANIZATION_DESCRIPTION:-CI SDK flow fixture}" \
    -v project_description="${FLOW_PROJECT_DESCRIPTION:-CI SDK flow fixture}" <<'SQL'
SELECT count(*) FROM (
  SELECT id FROM organizations
  WHERE id = :'fixture_id'
    AND (name IS DISTINCT FROM :'organization'
      OR description IS DISTINCT FROM :'organization_description'
      OR contact IS DISTINCT FROM :'email')
  UNION ALL
  SELECT id FROM projects
  WHERE id = :'project_id'
    AND (organization_id IS DISTINCT FROM :'fixture_id'
      OR name IS DISTINCT FROM :'project'
      OR description IS DISTINCT FROM :'project_description')
  UNION ALL
  SELECT id FROM user_auths
  WHERE id = :'fixture_id'
    AND email IS DISTINCT FROM :'email'
) AS unexpected;
SQL
)
  if [ "$unexpected_records" != '0' ]; then
    printf 'refusing to clean records not owned by this flow fixture\n' >&2
    return 1
  fi
}

cleanup_endpoint_fixture() {
  assert_project_fixture_ownership || return
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d endpoint_db \
    -v project_id="$FLOW_PROJECT_ID" >/dev/null <<'SQL'
CREATE TEMP TABLE flow_endpoints AS
SELECT id FROM endpoints WHERE project_id = :'project_id';

CREATE TEMP TABLE flow_endpoint_models AS
SELECT id FROM endpoint_provider_models
WHERE endpoint_id IN (SELECT id FROM flow_endpoints);

DELETE FROM endpoint_provider_model_options
WHERE endpoint_provider_model_id IN (SELECT id FROM flow_endpoint_models);
DELETE FROM endpoint_provider_models
WHERE id IN (SELECT id FROM flow_endpoint_models);
DELETE FROM endpoint_retries WHERE endpoint_id IN (SELECT id FROM flow_endpoints);
DELETE FROM endpoint_cachings WHERE endpoint_id IN (SELECT id FROM flow_endpoints);
DELETE FROM endpoint_tags WHERE endpoint_id IN (SELECT id FROM flow_endpoints);
DELETE FROM endpoints WHERE id IN (SELECT id FROM flow_endpoints);
SQL
}

cleanup_project_fixture() {
  assert_project_fixture_ownership || return
  project_cleanup_status=0
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" >/dev/null <<'SQL' || project_cleanup_status=$?
CREATE TEMP TABLE flow_assistants AS
SELECT id FROM assistants WHERE project_id = :'project_id';

CREATE TEMP TABLE flow_conversations AS
SELECT id FROM assistant_conversations WHERE project_id = :'project_id';

DELETE FROM call_contexts WHERE project_id = :'project_id';
DELETE FROM assistant_conversation_telephony_events
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_action_metrics
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_actions
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_arguments
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_contexts
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_message_metadata
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_message_metrics
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_messages
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_metadata
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_metrics
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_options
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversation_recordings
WHERE assistant_conversation_id IN (SELECT id FROM flow_conversations);
DELETE FROM assistant_conversations WHERE id IN (SELECT id FROM flow_conversations);

DELETE FROM assistant_deployment_audio_options
WHERE assistant_deployment_audio_id IN (
  SELECT id FROM assistant_deployment_audios
  WHERE assistant_deployment_id IN (
    SELECT id FROM assistant_phone_deployments
    WHERE assistant_id IN (SELECT id FROM flow_assistants)
  )
);
DELETE FROM assistant_deployment_audios
WHERE assistant_deployment_id IN (
  SELECT id FROM assistant_phone_deployments
  WHERE assistant_id IN (SELECT id FROM flow_assistants)
);
DELETE FROM assistant_deployment_telephony_options
WHERE assistant_deployment_telephony_id IN (
  SELECT id FROM assistant_phone_deployments
  WHERE assistant_id IN (SELECT id FROM flow_assistants)
);
DELETE FROM assistant_phone_deployments
WHERE assistant_id IN (SELECT id FROM flow_assistants);

DELETE FROM assistant_provider_model_options
WHERE assistant_provider_model_id IN (
  SELECT id FROM assistant_provider_models
  WHERE assistant_id IN (SELECT id FROM flow_assistants)
);
DELETE FROM assistant_provider_models
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistant_provider_agentkits
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistant_provider_websockets
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistant_provider_agentflows
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistant_tool_options
WHERE assistant_tool_id IN (
  SELECT id FROM assistant_tools
  WHERE assistant_id IN (SELECT id FROM flow_assistants)
);
DELETE FROM assistant_tools
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistant_configurations
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistant_tags
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistants WHERE id IN (SELECT id FROM flow_assistants);
SQL

  web_cleanup_status=0
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" >/dev/null <<'SQL' || web_cleanup_status=$?
DELETE FROM vaults WHERE project_id = :'project_id';
DELETE FROM project_credentials WHERE project_id = :'project_id';
DELETE FROM user_project_roles WHERE project_id = :'project_id';
DELETE FROM user_organization_roles WHERE organization_id = :'fixture_id';
DELETE FROM user_auth_tokens WHERE user_auth_id = :'fixture_id';
DELETE FROM projects WHERE id = :'project_id';
DELETE FROM organizations WHERE id = :'fixture_id';
DELETE FROM user_auths WHERE id = :'fixture_id';
SQL
  if [ "$project_cleanup_status" -ne 0 ]; then
    return "$project_cleanup_status"
  fi
  return "$web_cleanup_status"
}

cleanup_endpoint_project_fixture() {
  fixture_cleanup_status=0
  cleanup_endpoint_fixture || fixture_cleanup_status=$?
  cleanup_project_fixture || fixture_cleanup_status=$?
  return "$fixture_cleanup_status"
}

format_flow_ari_host() {
  case "$1" in
    \[*\]) printf '%s' "$1" ;;
    *:*) printf '[%s]' "$1" ;;
    *) printf '%s' "$1" ;;
  esac
}

seed_project_api_key_fixture() {
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" \
    -v api_key="$FLOW_API_KEY" >/dev/null <<'SQL'
INSERT INTO project_credentials (
  id, organization_id, project_id, name, key, status, created_actor_type
) VALUES (
  :'fixture_id', :'fixture_id', :'project_id', 'CI flow project API key', :'api_key', 'ACTIVE', 'unknown'
);
SQL
}

seed_asterisk_vault_fixture() {
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" \
    -v vault_id="$FLOW_VAULT_ID" \
    -v ari_url="$FLOW_ARI_URL" >/dev/null <<'SQL'
INSERT INTO vaults (
  id, organization_id, project_id, provider, name, value, status,
  created_actor_type, created_actor_id
) VALUES (
  :'vault_id', :'fixture_id', :'project_id', 'asterisk:Asterisk', 'CI Asterisk credential',
  json_build_object('ari_url', :'ari_url', 'ari_user', 'ci-flow', 'ari_password', 'ci-flow'),
  'ACTIVE', 'user', :'fixture_id'
);
SQL
}

seed_project_fixture() {
  cleanup_project_fixture

  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" \
    -v email="$FLOW_EMAIL" \
    -v token="$FLOW_TOKEN" \
    -v organization="$FLOW_ORGANIZATION" \
    -v project="$FLOW_PROJECT" \
    -v organization_description="${FLOW_ORGANIZATION_DESCRIPTION:-CI SDK flow fixture}" \
    -v project_description="${FLOW_PROJECT_DESCRIPTION:-CI SDK flow fixture}" >/dev/null <<'SQL'
INSERT INTO organizations (
  id, name, description, size, industry, contact, status, created_actor_type
) VALUES (
  :'fixture_id', :'organization', :'organization_description', '1', 'software', :'email', 'ACTIVE', 'unknown'
);
INSERT INTO projects (
  id, organization_id, name, description, status, created_actor_type
) VALUES (
  :'project_id', :'fixture_id', :'project', :'project_description', 'ACTIVE', 'unknown'
);
INSERT INTO user_auths (
  id, name, email, password, status, source, created_actor_type
) VALUES (
  :'fixture_id', 'CI SDK Flow', :'email', 'unused', 'ACTIVE', 'direct', 'unknown'
);
INSERT INTO user_auth_tokens (
  id, user_auth_id, token_type, token, expire_at, status, created_actor_type
) VALUES (
  :'fixture_id', :'fixture_id', 'auth-token', :'token', now() + interval '1 hour', 'ACTIVE', 'unknown'
);
INSERT INTO user_organization_roles (
  id, user_auth_id, organization_id, role, status, created_actor_type
) VALUES (
  :'fixture_id', :'fixture_id', :'fixture_id', 'owner', 'ACTIVE', 'unknown'
);
INSERT INTO user_project_roles (
  id, project_id, user_auth_id, role, status, created_actor_type
) VALUES (
  :'fixture_id', :'project_id', :'fixture_id', 'owner', 'ACTIVE', 'unknown'
);
SQL
}

verify_assistant_fixture() {
  expected_count=${FLOW_EXPECTED_COUNT:-2}
  count=$(psql -v ON_ERROR_STOP=1 -At -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v fixture_id="$FLOW_FIXTURE_ID" <<'SQL'
SELECT count(*) FROM assistants
WHERE project_id = :'project_id'
  AND organization_id = :'fixture_id';
SQL
)
  if [ "$count" != "$expected_count" ]; then
    printf 'unexpected assistant count: %s\n' "$count" >&2
    return 1
  fi
}

verify_assistant_lifecycle_fixture() {
  expected_count=${FLOW_EXPECTED_COUNT:-2}
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v fixture_id="$FLOW_FIXTURE_ID" <<'SQL'
SELECT
  (SELECT count(*) FROM assistants
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'),
  (SELECT count(*) FROM assistants
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND status = 'ARCHIEVE'),
  (SELECT count(*) FROM assistants
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND name LIKE '% Updated'),
  (SELECT count(*) FROM assistant_provider_models
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')),
  (SELECT count(*) FROM assistants AS assistant
   JOIN assistant_provider_models AS model
     ON model.id = assistant.assistant_provider_id
   WHERE assistant.project_id = :'project_id'
     AND assistant.organization_id = :'fixture_id'
     AND assistant.assistant_provider = 'MODEL'
     AND model.model_provider_name = 'anthropic');
SQL
)
  expected_providers=$((expected_count * 2))
  expected="$expected_count|$expected_count|$expected_count|$expected_providers|$expected_count"
  if [ "$counts" != "$expected" ]; then
    printf 'unexpected assistant lifecycle counts: %s\n' "$counts" >&2
    return 1
  fi
}

verify_assistant_components_fixture() {
  expected_assistants=${FLOW_EXPECTED_ASSISTANTS:-1}
  expected_tools=${FLOW_EXPECTED_TOOLS:-1}
  expected_configurations=${FLOW_EXPECTED_CONFIGURATIONS:-5}
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v fixture_id="$FLOW_FIXTURE_ID" <<'SQL'
SELECT
  (SELECT count(*) FROM assistants
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND status = 'ACTIVE'),
  (SELECT count(*) FROM assistant_tools
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND status = 'ARCHIEVE'),
  (SELECT count(*) FROM assistant_tools
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND status = 'ACTIVE'),
  (SELECT count(*) FROM assistant_tool_options
   WHERE assistant_tool_id IN (
     SELECT id FROM assistant_tools
     WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
   )
     AND key = 'tool.endpoint_id'
     AND value LIKE 'updated-%'
     AND status = 'ACTIVE'),
  (SELECT count(*) FROM assistant_configurations
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND status = 'ARCHIEVE'),
  (SELECT count(*) FROM assistant_configurations
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND status = 'ACTIVE'),
  (SELECT count(DISTINCT configuration_type) FROM assistant_configurations
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND status = 'ARCHIEVE'),
  (SELECT count(*) FROM assistant_configuration_options
   WHERE assistant_configuration_id IN (
     SELECT id FROM assistant_configurations
     WHERE project_id = :'project_id'
       AND organization_id = :'fixture_id'
   )
     AND value LIKE 'updated-%'
     AND status = 'ARCHIEVE');
SQL
)
  expected="$expected_assistants|$expected_tools|0|$expected_tools|$expected_configurations|0|$expected_configurations|$expected_configurations"
  if [ "$counts" != "$expected" ]; then
    printf 'unexpected assistant component counts: %s\n' "$counts" >&2
    return 1
  fi
}

verify_assistant_provider_fixture() {
  expected_assistants=${FLOW_EXPECTED_ASSISTANTS:-2}
  expected_providers=${FLOW_EXPECTED_PROVIDERS:-2}
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v fixture_id="$FLOW_FIXTURE_ID" <<'SQL'
SELECT
  (SELECT count(*) FROM assistants
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'),
  (SELECT count(*) FROM assistant_provider_models
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND model_provider_name = 'openai'),
  (SELECT count(*) FROM assistant_provider_models
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND model_provider_name = 'anthropic'),
  (SELECT count(*) FROM assistant_provider_agentkits
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND url = 'agentkit:50051'),
  (SELECT count(*) FROM assistant_provider_websockets
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND url = 'wss://example.invalid/agent'),
  (SELECT count(*) FROM assistant_provider_agentflows
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND schema_version = '1.0');
SQL
)
  expected="$expected_assistants|$expected_providers|$expected_providers|$expected_providers|$expected_providers|$expected_providers"
  if [ "$counts" != "$expected" ]; then
    printf 'unexpected assistant provider counts: %s\n' "$counts" >&2
    return 1
  fi
}

verify_endpoint_provider_fixture() {
  expected_count=${FLOW_EXPECTED_COUNT:-1}
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d endpoint_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v fixture_id="$FLOW_FIXTURE_ID" <<'SQL'
SELECT
  (SELECT count(*) FROM endpoints
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'),
  (SELECT count(*) FROM endpoint_provider_models
   WHERE endpoint_id IN (SELECT id FROM endpoints WHERE project_id = :'project_id')
     AND model_provider_name = 'openai'),
  (SELECT count(*) FROM endpoint_provider_models
   WHERE endpoint_id IN (SELECT id FROM endpoints WHERE project_id = :'project_id')
     AND model_provider_name = 'anthropic'),
  (SELECT count(*) FROM endpoints AS endpoint
   JOIN endpoint_provider_models AS model
     ON model.id = endpoint.endpoint_provider_model_id
   WHERE endpoint.project_id = :'project_id'
     AND model.model_provider_name = 'anthropic'),
  (SELECT count(*) FROM endpoint_provider_model_options
   WHERE endpoint_provider_model_id IN (
     SELECT id FROM endpoint_provider_models
     WHERE endpoint_id IN (SELECT id FROM endpoints WHERE project_id = :'project_id')
       AND model_provider_name = 'openai'
   )
     AND key = 'model'
     AND value = 'gpt-4o-mini'),
  (SELECT count(*) FROM endpoint_provider_model_options
   WHERE endpoint_provider_model_id IN (
     SELECT id FROM endpoint_provider_models
     WHERE endpoint_id IN (SELECT id FROM endpoints WHERE project_id = :'project_id')
       AND model_provider_name = 'anthropic'
   )
     AND key = 'model'
     AND value = 'claude-3-5-sonnet');
SQL
)
  expected="$expected_count|$expected_count|$expected_count|$expected_count|$expected_count|$expected_count"
  if [ "$counts" != "$expected" ]; then
    printf 'unexpected endpoint provider counts: %s\n' "$counts" >&2
    return 1
  fi
}

verify_assistant_phone_call_fixture() {
  expected_count=${FLOW_EXPECTED_COUNT:-2}
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v vault_id="$FLOW_VAULT_ID" <<'SQL'
SELECT
  (SELECT count(*) FROM assistants
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'),
  (SELECT count(*) FROM assistant_phone_deployments
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND telephony_provider = 'asterisk'
     AND status = 'ACTIVE'),
  (SELECT count(*) FROM assistant_deployment_telephony_options
   WHERE assistant_deployment_telephony_id IN (
     SELECT id FROM assistant_phone_deployments
     WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
   )
     AND key = 'rapida.credential_id'
     AND value = :'vault_id'),
  (SELECT count(*) FROM assistant_conversations
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND source = 'phone-call'
     AND direction = 'outbound'),
  (SELECT count(*) FROM call_contexts
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND provider = 'asterisk'
     AND direction = 'outbound'
     AND channel_uuid != '');
SQL
)
  expected="$expected_count|$expected_count|$expected_count|$expected_count|$expected_count"
  if [ "$counts" != "$expected" ]; then
    printf 'unexpected assistant deployment call counts: %s\n' "$counts" >&2
    return 1
  fi
}
