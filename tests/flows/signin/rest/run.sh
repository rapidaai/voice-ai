#!/bin/sh
set -eu

readonly flow_email='ci-flow-signin-rest@example.invalid'
readonly flow_password='ci-flow-signin-password'
readonly flow_token='ci-flow-signin-rest-token'
readonly web_endpoint="${WEB_API_REST_ENDPOINT:-http://web-api:9001}"
temporary_directory=$(mktemp -d)

cleanup_data() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
    -v flow_email="$flow_email" >/dev/null <<'SQL'
DELETE FROM user_auth_tokens
WHERE user_auth_id IN (SELECT id FROM user_auths WHERE email = :'flow_email');
DELETE FROM user_auths WHERE email = :'flow_email';
SQL
}

cleanup() {
  cleanup_data
  rm -rf "$temporary_directory"
}

trap cleanup EXIT HUP INT TERM
cleanup_data

psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db \
  -v flow_email="$flow_email" \
  -v flow_password="$flow_password" \
  -v flow_token="$flow_token" >/dev/null <<'SQL'
WITH created_user AS (
  INSERT INTO user_auths (
    name, email, password, status, source, created_actor_type
  ) VALUES (
    'CI REST Signin Flow', :'flow_email', md5(:'flow_password'), 'ACTIVE', 'direct', 'unknown'
  )
  RETURNING id
)
INSERT INTO user_auth_tokens (
  user_auth_id, token_type, token, expire_at, status, created_actor_type
)
SELECT id, 'auth-token', :'flow_token', now() + interval '1 hour', 'ACTIVE', 'unknown'
FROM created_user;
SQL

request_body=$(jq -n --arg email "$flow_email" --arg password "$flow_password" \
  '{email: $email, password: $password}')
response_file="$temporary_directory/response.json"
status=$(curl --silent --show-error --connect-timeout 2 --max-time 10 \
  --output "$response_file" --write-out '%{http_code}' \
  --request POST --header 'Content-Type: application/json' \
  --data "$request_body" "$web_endpoint/v1/auth/authenticate/")

if [ "$status" != '200' ]; then
  printf 'REST signin returned HTTP %s: ' "$status" >&2
  cat "$response_file" >&2
  exit 1
fi

jq -e --arg email "$flow_email" \
  '.code == 200 and .success == true and (.data.user.Email // .data.user.email) == $email and ((.data.token.Token // .data.token.token) | length > 0)' \
  "$response_file" >/dev/null

echo 'REST signin flow passed'
