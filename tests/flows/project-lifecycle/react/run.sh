#!/bin/sh
set -eu

readonly flow_owner_id='8100602'
readonly flow_member_id='8100603'
readonly flow_organization='CI React Project Lifecycle Organization'
readonly flow_project='CI React Project Lifecycle'
readonly flow_updated_project='CI React Project Lifecycle Updated'
readonly flow_owner_email='ci-flow-project-owner-react@example.invalid'
readonly flow_member_email='ci-flow-project-member-react@example.invalid'
script_directory=$(cd "$(dirname "$0")" && pwd)

cleanup() {
  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v owner_id="$flow_owner_id" \
    -v member_id="$flow_member_id" \
    -v organization="$flow_organization" >/dev/null <<'SQL'
CREATE TEMP TABLE flow_users AS
SELECT id FROM user_auths WHERE id IN (:'owner_id', :'member_id');

CREATE TEMP TABLE flow_organizations AS
SELECT id FROM organizations WHERE name = :'organization';

CREATE TEMP TABLE flow_projects AS
SELECT id FROM projects
WHERE organization_id IN (SELECT id FROM flow_organizations);

DELETE FROM project_credentials
WHERE project_id IN (SELECT id FROM flow_projects);
DELETE FROM user_project_roles
WHERE user_auth_id IN (SELECT id FROM flow_users)
   OR project_id IN (SELECT id FROM flow_projects);
DELETE FROM projects WHERE id IN (SELECT id FROM flow_projects);
DELETE FROM user_organization_roles
WHERE user_auth_id IN (SELECT id FROM flow_users)
   OR organization_id IN (SELECT id FROM flow_organizations);
DELETE FROM organization_credentials
WHERE organization_id IN (SELECT id FROM flow_organizations);
DELETE FROM organizations WHERE id IN (SELECT id FROM flow_organizations);
DELETE FROM user_feature_permissions
WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_auth_tokens WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_socials WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_roles WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_auths WHERE id IN (SELECT id FROM flow_users);
SQL
}

seed() {
  cleanup

  psql -v ON_ERROR_STOP=1 -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v owner_id="$flow_owner_id" \
    -v member_id="$flow_member_id" \
    -v organization="$flow_organization" \
    -v owner_email="$flow_owner_email" \
    -v member_email="$flow_member_email" \
    -v token="$FLOW_TOKEN" >/dev/null <<'SQL'
INSERT INTO organizations (
  id, name, description, size, industry, contact, status, created_actor_type
) VALUES (
  :'owner_id', :'organization', 'CI React project lifecycle fixture', '1',
  'software', :'owner_email', 'ACTIVE', 'unknown'
);
INSERT INTO user_auths (
  id, name, email, password, status, source, created_actor_type
) VALUES
  (:'owner_id', 'CI Project Owner', :'owner_email', 'unused', 'ACTIVE', 'direct', 'unknown'),
  (:'member_id', 'CI Project Member', :'member_email', 'unused', 'ACTIVE', 'direct', 'unknown');
INSERT INTO user_auth_tokens (
  id, user_auth_id, token_type, token, expire_at, status, created_actor_type
) VALUES (
  :'owner_id', :'owner_id', 'auth-token', :'token', now() + interval '1 hour',
  'ACTIVE', 'unknown'
);
INSERT INTO user_organization_roles (
  id, user_auth_id, organization_id, role, status, created_actor_type
) VALUES
  (:'owner_id', :'owner_id', :'owner_id', 'owner', 'ACTIVE', 'unknown'),
  (:'member_id', :'member_id', :'owner_id', 'member', 'ACTIVE', 'unknown');
SQL
}

verify() {
  counts=$(psql -v ON_ERROR_STOP=1 -At -F '|' -h "${FLOW_POSTGRES_HOST:-postgres}" -U rapida_user -d web_db \
    -v owner_id="$flow_owner_id" \
    -v member_id="$flow_member_id" \
    -v project="$flow_updated_project" <<'SQL'
SELECT
  (SELECT count(*) FROM projects
   WHERE organization_id = :'owner_id'
     AND name = :'project'
     AND description = 'Updated by the CI React project lifecycle flow'
     AND status = 'ACTIVE'),
  (SELECT count(*) FROM user_project_roles
   WHERE user_auth_id = :'owner_id'
     AND project_id IN (SELECT id FROM projects WHERE organization_id = :'owner_id')
     AND role = 'admin'
     AND status = 'ACTIVE'),
  (SELECT count(*) FROM user_project_roles
   WHERE user_auth_id = :'member_id'
     AND project_id IN (SELECT id FROM projects WHERE organization_id = :'owner_id')
     AND role = 'writer'
     AND status = 'ACTIVE');
SQL
)
  if [ "$counts" != '1|1|1' ]; then
    printf 'unexpected project lifecycle counts: %s\n' "$counts" >&2
    return 1
  fi
}

export FLOW_OWNER_ID="$flow_owner_id"
export FLOW_MEMBER_ID="$flow_member_id"
export FLOW_TOKEN='ci-flow-project-lifecycle-react-token'
export FLOW_PROJECT="$flow_project"
export FLOW_UPDATED_PROJECT="$flow_updated_project"
export FLOW_MEMBER_EMAIL="$flow_member_email"

trap cleanup EXIT HUP INT TERM
seed
timeout 30s node "$script_directory/project-lifecycle.js"
verify
