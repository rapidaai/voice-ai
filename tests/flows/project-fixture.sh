#!/bin/sh

cleanup_project_fixture() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" >/dev/null <<'SQL'
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
DELETE FROM assistant_configurations
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistant_tags
WHERE assistant_id IN (SELECT id FROM flow_assistants);
DELETE FROM assistants WHERE id IN (SELECT id FROM flow_assistants);
SQL

  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" >/dev/null <<'SQL'
DELETE FROM vaults WHERE project_id = :'project_id';
DELETE FROM project_credentials WHERE project_id = :'project_id';
DELETE FROM user_project_roles WHERE project_id = :'project_id';
DELETE FROM user_organization_roles WHERE organization_id = :'fixture_id';
DELETE FROM user_auth_tokens WHERE user_auth_id = :'fixture_id';
DELETE FROM projects WHERE id = :'project_id';
DELETE FROM organizations WHERE id = :'fixture_id';
DELETE FROM user_auths WHERE id = :'fixture_id';
SQL
}

seed_project_api_key_fixture() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
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
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
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

  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
    -v fixture_id="$FLOW_FIXTURE_ID" \
    -v project_id="$FLOW_PROJECT_ID" \
    -v email="$FLOW_EMAIL" \
    -v token="$FLOW_TOKEN" \
    -v organization="$FLOW_ORGANIZATION" \
    -v project="$FLOW_PROJECT" >/dev/null <<'SQL'
INSERT INTO organizations (
  id, name, description, size, industry, contact, status, created_actor_type
) VALUES (
  :'fixture_id', :'organization', 'CI SDK flow fixture', '1', 'software', :'email', 'ACTIVE', 'unknown'
);
INSERT INTO projects (
  id, organization_id, name, description, status, created_actor_type
) VALUES (
  :'project_id', :'fixture_id', :'project', 'CI SDK flow fixture', 'ACTIVE', 'unknown'
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
  count=$(psql -v ON_ERROR_STOP=1 -At -h postgres -U rapida_user -d assistant_db \
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

verify_assistant_provider_fixture() {
  expected_assistants=${FLOW_EXPECTED_ASSISTANTS:-2}
  expected_providers=${FLOW_EXPECTED_PROVIDERS:-2}
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h postgres -U rapida_user -d assistant_db \
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

verify_assistant_phone_call_fixture() {
  expected_count=${FLOW_EXPECTED_COUNT:-2}
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h postgres -U rapida_user -d assistant_db \
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
