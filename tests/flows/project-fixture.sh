#!/bin/sh

cleanup_project_fixture() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" >/dev/null <<'SQL'
CREATE TEMP TABLE flow_assistants AS
SELECT id FROM assistants WHERE project_id = :'project_id';

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
DELETE FROM user_project_roles WHERE project_id = :'project_id';
DELETE FROM user_organization_roles WHERE organization_id = :'fixture_id';
DELETE FROM user_auth_tokens WHERE user_auth_id = :'fixture_id';
DELETE FROM projects WHERE id = :'project_id';
DELETE FROM organizations WHERE id = :'fixture_id';
DELETE FROM user_auths WHERE id = :'fixture_id';
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

verify_assistant_provider_fixture() {
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h postgres -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v fixture_id="$FLOW_FIXTURE_ID" <<'SQL'
SELECT
  (SELECT count(*) FROM assistants
   WHERE project_id = :'project_id'
     AND organization_id = :'fixture_id'
     AND created_actor_type = 'user'
     AND created_actor_id = :'fixture_id'),
  (SELECT count(*) FROM assistant_provider_models
   WHERE assistant_id IN (SELECT id FROM assistants WHERE project_id = :'project_id')
     AND model_provider_name IN ('openai', 'anthropic')),
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
  if [ "$counts" != '1|2|1|1|1' ]; then
    printf 'unexpected assistant provider counts: %s\n' "$counts" >&2
    return 1
  fi
}
