#!/bin/sh
set -eu

readonly flow_email='ci-flow-signup-react@example.invalid'
readonly flow_password='ci-flow-signup-password'
readonly flow_name='CI React Signup Flow'
readonly flow_organization='CI React SDK Organization'
readonly flow_project='CI React SDK Project'
script_directory=$(cd "$(dirname "$0")" && pwd)

cleanup() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
    -v flow_email="$flow_email" \
    -v flow_organization="$flow_organization" >/dev/null <<'SQL'
CREATE TEMP TABLE flow_users AS
SELECT id FROM user_auths WHERE email = :'flow_email';

CREATE TEMP TABLE flow_organizations AS
SELECT id FROM organizations WHERE name = :'flow_organization'
UNION
SELECT organization_id FROM user_organization_roles
WHERE user_auth_id IN (SELECT id FROM flow_users);

DELETE FROM project_credentials
WHERE project_id IN (SELECT id FROM projects WHERE organization_id IN (SELECT id FROM flow_organizations));
DELETE FROM user_project_roles
WHERE user_auth_id IN (SELECT id FROM flow_users)
   OR project_id IN (SELECT id FROM projects WHERE organization_id IN (SELECT id FROM flow_organizations));
DELETE FROM projects WHERE organization_id IN (SELECT id FROM flow_organizations);
DELETE FROM user_organization_roles
WHERE user_auth_id IN (SELECT id FROM flow_users)
   OR organization_id IN (SELECT id FROM flow_organizations);
DELETE FROM organization_credentials WHERE organization_id IN (SELECT id FROM flow_organizations);
DELETE FROM organizations WHERE id IN (SELECT id FROM flow_organizations);
DELETE FROM user_feature_permissions WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_auth_tokens WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_socials WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_roles WHERE user_auth_id IN (SELECT id FROM flow_users);
DELETE FROM user_auths WHERE id IN (SELECT id FROM flow_users);
SQL
}

trap cleanup EXIT HUP INT TERM
cleanup

FLOW_EMAIL="$flow_email" \
FLOW_PASSWORD="$flow_password" \
FLOW_NAME="$flow_name" \
FLOW_ORGANIZATION="$flow_organization" \
FLOW_PROJECT="$flow_project" \
  timeout 30s node "$script_directory/signup-organization-project.js"
