#!/bin/sh
set -eu

readonly flow_email='ci-flow-signin-react@example.invalid'
readonly flow_password='ci-flow-signin-password'
readonly flow_token='ci-flow-signin-react-token'
readonly flow_id='8100002'
script_directory=$(cd "$(dirname "$0")" && pwd)

cleanup() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
    -v flow_email="$flow_email" >/dev/null <<'SQL'
DELETE FROM user_auth_tokens
WHERE user_auth_id IN (SELECT id FROM user_auths WHERE email = :'flow_email');
DELETE FROM user_auths WHERE email = :'flow_email';
SQL
}

trap cleanup EXIT HUP INT TERM
cleanup

psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
  -v flow_email="$flow_email" \
  -v flow_password="$flow_password" \
  -v flow_token="$flow_token" \
  -v flow_id="$flow_id" >/dev/null <<'SQL'
WITH created_user AS (
  INSERT INTO user_auths (
    id, name, email, password, status, source, created_actor_type
  ) VALUES (
    :'flow_id', 'CI React Signin Flow', :'flow_email', md5(:'flow_password'), 'ACTIVE', 'direct', 'unknown'
  )
  RETURNING id
)
INSERT INTO user_auth_tokens (
  id, user_auth_id, token_type, token, expire_at, status, created_actor_type
)
SELECT :'flow_id', id, 'auth-token', :'flow_token', now() + interval '1 hour', 'ACTIVE', 'unknown'
FROM created_user;
SQL

FLOW_EMAIL="$flow_email" FLOW_PASSWORD="$flow_password" \
  timeout 30s node "$script_directory/signin.js"
