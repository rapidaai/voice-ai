#!/bin/sh

cleanup_restricted_user() {
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v user_id="$FLOW_RESTRICTED_USER_ID" >/dev/null <<'SQL'
DELETE FROM user_auth_tokens WHERE user_auth_id = :'user_id';
DELETE FROM user_project_roles WHERE user_auth_id = :'user_id';
DELETE FROM user_organization_roles WHERE user_auth_id = :'user_id';
DELETE FROM user_auths WHERE id = :'user_id';
SQL
}

cleanup_security_data() {
  cleanup_project_fixture
  original_fixture_id=$FLOW_FIXTURE_ID
  original_project_id=$FLOW_PROJECT_ID
  FLOW_FIXTURE_ID=$FLOW_OTHER_ORGANIZATION_ID
  FLOW_PROJECT_ID=$FLOW_OTHER_PROJECT_ID
  cleanup_project_fixture
  FLOW_FIXTURE_ID=$original_fixture_id
  FLOW_PROJECT_ID=$original_project_id
  cleanup_restricted_user
}

seed_security_fixtures() {
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v organization_id="$FLOW_OTHER_ORGANIZATION_ID" \
    -v project_id="$FLOW_OTHER_PROJECT_ID" \
    -v organization_email="$FLOW_OTHER_EMAIL" \
    -v user_id="$FLOW_RESTRICTED_USER_ID" \
    -v user_email="$FLOW_RESTRICTED_EMAIL" \
    -v user_token="$FLOW_RESTRICTED_TOKEN" >/dev/null <<'SQL'
INSERT INTO organizations (
  id, name, description, size, industry, contact, status, created_actor_type
) VALUES (
  :'organization_id', 'CI Authorization Other Organization',
  'Cross-project authorization fixture', '1', 'software',
  :'organization_email', 'ACTIVE', 'unknown'
);
INSERT INTO projects (
  id, organization_id, name, description, status, created_actor_type
) VALUES (
  :'project_id', :'organization_id', 'CI Authorization Other Project',
  'Cross-project authorization fixture', 'ACTIVE', 'unknown'
);
INSERT INTO user_auths (
  id, name, email, password, status, source, created_actor_type
) VALUES (
  :'user_id', 'CI Restricted User', :'user_email', 'unused',
  'ACTIVE', 'direct', 'unknown'
);
INSERT INTO user_auth_tokens (
  id, user_auth_id, token_type, token, expire_at, status, created_actor_type
) VALUES (
  :'user_id', :'user_id', 'auth-token', :'user_token',
  now() + interval '1 hour', 'ACTIVE', 'unknown'
);
SQL
}

verify_authorization_fixture() {
  count=$(psql -v ON_ERROR_STOP=1 -At -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d assistant_db \
    -v project_id="$FLOW_PROJECT_ID" \
    -v other_project_id="$FLOW_OTHER_PROJECT_ID" <<'SQL'
SELECT count(*) FROM assistants
WHERE project_id IN (:'project_id', :'other_project_id');
SQL
)
  if [ "$count" != '0' ]; then
    printf 'authorization failures created %s assistant records\n' "$count" >&2
    return 1
  fi
}
