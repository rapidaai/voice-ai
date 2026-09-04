#!/bin/sh
set -eu

readonly readiness_key='PSQL psql://postgres:5432'
readonly collection='/workspace/openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json'
readonly report_directory="${REPORT_DIRECTORY:-/reports}"
readonly ci_auth_token="${CI_STACK_AUTH_TOKEN:?CI_STACK_AUTH_TOKEN is required}"
readonly ci_project_api_key="${CI_STACK_PROJECT_API_KEY:?CI_STACK_PROJECT_API_KEY is required}"
readonly mode="${1:-smoke}"

temporary_directory=$(mktemp -d)
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM

retry_json() {
  retry_name=$1
  retry_url=$2
  retry_filter=$3
  retry_attempts=60

  while [ "$retry_attempts" -gt 0 ]; do
    if curl --fail --silent --show-error --connect-timeout 2 --max-time 5 "$retry_url" \
      | jq -e "$retry_filter" >/dev/null 2>&1; then
      printf '%s passed\n' "$retry_name"
      return
    fi
    retry_attempts=$((retry_attempts - 1))
    sleep 1
  done

  printf '%s failed: %s\n' "$retry_name" "$retry_url" >&2
  return 1
}

check_migration() {
  service=$1
  database=$2
  migration_directory=$3
  expected_version=$(find "$migration_directory" -type f -name '*.up.sql' \
    | sed -E 's#.*/0*([0-9]+)_.*#\1#' \
    | sort -n \
    | tail -n 1)
  state=$(psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d "$database" -Atc \
    "SELECT version || '|' || dirty FROM schema_migrations LIMIT 1")
  version=${state%%|*}
  dirty=${state##*|}

  if [ "$version" != "$expected_version" ] || [ "$dirty" != "false" ]; then
    printf '%s migration state is version=%s dirty=%s; expected version=%s dirty=false\n' \
      "$service" "$version" "$dirty" "$expected_version" >&2
    return 1
  fi
  printf '%s migration passed at version %s\n' "$service" "$version"
}

check_http_route() {
  path=$1
  expected_status=$2
  body_file="$temporary_directory/http-body"
  header_file="$temporary_directory/http-headers"
  status=$(curl --silent --show-error --output "$body_file" --dump-header "$header_file" \
    --write-out '%{http_code}' "http://nginx:8080$path")

  if [ "$status" != "$expected_status" ]; then
    printf 'nginx route %s returned HTTP %s; expected %s\n' "$path" "$status" "$expected_status" >&2
    return 1
  fi
  if grep -qi '<html' "$body_file"; then
    printf 'nginx route %s incorrectly returned the UI fallback\n' "$path" >&2
    return 1
  fi
  printf 'nginx route %s passed\n' "$path"
}

check_ui() {
  direct_ui="$temporary_directory/ui.html"
  nginx_ui="$temporary_directory/nginx.html"
  curl --fail --silent --show-error --retry 30 --retry-all-errors --retry-delay 1 \
    --max-time 5 http://ui:3000/ > "$direct_ui"
  curl --fail --silent --show-error --retry 30 --retry-all-errors --retry-delay 1 \
    --max-time 5 http://nginx:8080/ > "$nginx_ui"

  grep -qi '<div id="root"' "$nginx_ui"
  asset=$(grep -Eo "/static/[^\"']+\.(js|css)" "$nginx_ui" | head -n 1)
  if [ -z "$asset" ]; then
    echo 'nginx UI response contains no static asset' >&2
    return 1
  fi
  curl --fail --silent --show-error --max-time 10 "http://nginx:8080$asset" >/dev/null

  check_http_route /v1/__smoke__ 404
  check_http_route /oauth/__smoke__ 404
  retry_json 'nginx web-api health proxy' 'http://nginx:8080/healthz/' \
    '.code == 200 and .success == true and .data.healthy == true'
  printf 'ui and nginx smoke tests passed\n'
}

seed_assistant_smoke() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db >/dev/null <<SQL
DELETE FROM user_project_roles WHERE id = 1;
DELETE FROM user_organization_roles WHERE id = 1;
DELETE FROM user_auth_tokens WHERE id = 1;
DELETE FROM project_credentials WHERE id = 1;
DELETE FROM projects WHERE id = 1;
DELETE FROM organizations WHERE id = 1;
DELETE FROM user_auths WHERE id = 1;
INSERT INTO organizations (id,name,description,size,industry,contact,status,created_actor_type) VALUES (1,'CI','CI','1','CI','ci@example.invalid','ACTIVE','unknown');
INSERT INTO projects (id,organization_id,name,description,status,created_actor_type) VALUES (1,1,'CI','CI','ACTIVE','unknown');
INSERT INTO project_credentials (id,organization_id,project_id,name,key,status,created_actor_type) VALUES (1,1,1,'CI project API key','$ci_project_api_key','ACTIVE','unknown');
INSERT INTO user_auths (id,name,email,password,status,source,created_actor_type) VALUES (1,'CI','ci@example.invalid','unused','ACTIVE','direct','unknown');
INSERT INTO user_auth_tokens (id,user_auth_id,token_type,token,expire_at,status,created_actor_type) VALUES (1,1,'auth-token','$ci_auth_token',now()+interval '1 hour','ACTIVE','unknown');
INSERT INTO user_organization_roles (id,user_auth_id,organization_id,role,status,created_actor_type) VALUES (1,1,1,'owner','ACTIVE','unknown');
INSERT INTO user_project_roles (id,project_id,user_auth_id,role,status,created_actor_type) VALUES (1,1,1,'owner','ACTIVE','unknown');
SQL
}

seed_telephony_callback_contexts() {
  psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d assistant_db >/dev/null <<SQL
DELETE FROM call_contexts WHERE context_id LIKE 'ci-project-%' OR context_id LIKE 'ci-pat-%';
INSERT INTO call_contexts (id,context_id,assistant_id,conversation_id,project_id,organization_id,auth_type,auth_user_id,auth_actor_type,auth_actor_id,provider,direction) VALUES
  (7000001,'ci-project-twilio',1,1,1,1,'project',NULL,'project',1,'twilio','outbound'),
  (7000002,'ci-project-exotel',1,1,1,1,'project',NULL,'project',1,'exotel','outbound'),
  (7000003,'ci-project-vonage',1,1,1,1,'project',NULL,'project',1,'vonage','outbound'),
  (7000004,'ci-project-telnyx',1,1,1,1,'project',NULL,'project',1,'telnyx','outbound'),
  (7000005,'ci-project-asterisk',1,1,1,1,'project',NULL,'project',1,'asterisk','outbound'),
  (7000006,'ci-project-sip',1,1,1,1,'project',NULL,'project',1,'sip','outbound'),
  (7000007,'ci-project-vobiz',1,1,1,1,'project',NULL,'project',1,'vobiz','outbound'),
  (7000008,'ci-pat-twilio',1,1,1,1,'user',1,'user',1,'twilio','outbound'),
  (7000009,'ci-pat-exotel',1,1,1,1,'user',1,'user',1,'exotel','outbound'),
  (7000010,'ci-pat-vonage',1,1,1,1,'user',1,'user',1,'vonage','outbound'),
  (7000011,'ci-pat-telnyx',1,1,1,1,'user',1,'user',1,'telnyx','outbound'),
  (7000012,'ci-pat-asterisk',1,1,1,1,'user',1,'user',1,'asterisk','outbound'),
  (7000013,'ci-pat-sip',1,1,1,1,'user',1,'user',1,'sip','outbound'),
  (7000014,'ci-pat-vobiz',1,1,1,1,'user',1,'user',1,'vobiz','outbound');
SQL
}

check_telephony_callback() {
  callback_auth=$1
  callback_provider=$2
  callback_method=$3
  callback_payload=$4
  callback_content_type=$5
  callback_context_id="ci-$callback_auth-$callback_provider"
  callback_body_file="$temporary_directory/$callback_context_id.body"
  callback_url="http://assistant-api:9007/v1/talk/$callback_provider/ctx/$callback_context_id/event"

  if [ "$callback_method" = 'GET' ]; then
    callback_status=$(curl --silent --show-error --output "$callback_body_file" --write-out '%{http_code}' \
      "$callback_url?$callback_payload")
  else
    callback_status=$(curl --silent --show-error --output "$callback_body_file" --write-out '%{http_code}' \
      --request POST --header "Content-Type: $callback_content_type" --data "$callback_payload" "$callback_url")
  fi

  case "$callback_status" in
    2??) ;;
    *)
      printf '%s callback with %s auth returned HTTP %s: ' "$callback_provider" "$callback_auth" "$callback_status" >&2
      cat "$callback_body_file" >&2
      return 1
      ;;
  esac

  callback_call_status=$(psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d assistant_db -Atc \
    "SELECT call_status FROM call_contexts WHERE context_id = '$callback_context_id'")
  if [ "$callback_call_status" != 'completed' ]; then
    printf '%s callback with %s auth stored status %s; expected completed\n' \
      "$callback_provider" "$callback_auth" "$callback_call_status" >&2
    return 1
  fi
  printf '%s callback with %s auth passed\n' "$callback_provider" "$callback_auth"
}

run_telephony_callback_smoke() {
  seed_telephony_callback_contexts

  for callback_auth in project pat; do
    check_telephony_callback "$callback_auth" twilio POST \
      'CallSid=twilio-call&CallStatus=completed' 'application/x-www-form-urlencoded'
    check_telephony_callback "$callback_auth" exotel POST \
      'CallSid=exotel-call&Status=completed' 'application/x-www-form-urlencoded'
    check_telephony_callback "$callback_auth" vonage GET \
      'status=completed&uuid=vonage-call&duration=1' ''
    check_telephony_callback "$callback_auth" telnyx POST \
      '{"data":{"event_type":"call.hangup","payload":{"call_control_id":"telnyx-call"}}}' 'application/json'
    check_telephony_callback "$callback_auth" asterisk POST \
      '{"type":"ChannelDestroyed","channel":{"id":"asterisk-call"},"cause":16,"cause_txt":"NORMAL_CLEARING"}' 'application/json'
    check_telephony_callback "$callback_auth" sip POST \
      '{"event":"completed","call_id":"sip-call"}' 'application/json'
    check_telephony_callback "$callback_auth" vobiz POST \
      'Event=Hangup&CallUUID=vobiz-call&CallStatus=completed' 'application/x-www-form-urlencoded'
  done

  echo 'All authenticated telephony callback smoke tests passed'
}

run_integration_checks() {
  redis_response=$(redis-cli -h redis ping)
  [ "$redis_response" = 'PONG' ] || {
    printf 'redis integration check failed: %s\n' "$redis_response" >&2
    exit 1
  }
  pg_isready -h postgres -U rapida_user -d web_db

  check_migration web-api web_db /workspace/migrations/web-api
  check_migration integration-api integration_db /workspace/migrations/integration-api
  check_migration endpoint-api endpoint_db /workspace/migrations/endpoint-api
  check_migration assistant-api assistant_db /workspace/migrations/assistant-api

  for service in web-api:9001 integration-api:9004 endpoint-api:9005 assistant-api:9007; do
    service_name=${service%%:*}
    port=${service##*:}
    retry_json "$service_name health" "http://$service_name:$port/healthz/" \
      '.code == 200 and .success == true and .data.healthy == true'
    retry_json "$service_name readiness" "http://$service_name:$port/readiness/" \
      ".code == 200 and .success == true and .data[\"$readiness_key\"] == true"
  done

  check_ui
  echo 'All service integration checks passed'
}

run_smoke_tests() {
  mkdir -p "$report_directory"
  seed_assistant_smoke

  echo 'Running full assistant REST smoke flow with project API key authentication'
  ./node_modules/.bin/newman run "$collection" \
    --folder 'Smoke Flow' \
    --bail \
    --env-var baseUrl=http://assistant-api:9007 \
    --env-var apiKey="$ci_project_api_key" \
    --env-var authToken= \
    --env-var authId= \
    --env-var projectId= \
    --reporters cli,junit \
    --reporter-junit-export "$report_directory/assistant-smoke-api-key.xml"

  echo 'Running full assistant REST smoke flow with personal access token authentication'
  ./node_modules/.bin/newman run "$collection" \
    --folder 'Smoke Flow' \
    --bail \
    --env-var baseUrl=http://assistant-api:9007 \
    --env-var apiKey= \
    --env-var authToken="$ci_auth_token" \
    --env-var authId=1 \
    --env-var projectId=1 \
    --reporters cli,junit \
    --reporter-junit-export "$report_directory/assistant-smoke-pat.xml"

  echo 'Running released Node SDK smoke tests with both authentication methods'
  node /workspace/sdk-smoke.js

  echo 'Running telephony callback smoke tests with stored project and PAT authentication'
  run_telephony_callback_smoke

  echo 'All smoke tests passed'
}

case "$mode" in
  integration)
    run_integration_checks
    ;;
  smoke)
    run_smoke_tests
    ;;
  telephony-callbacks)
    /workspace/tests/integration/telephony/callbacks/run.sh
    ;;
  *)
    printf 'unsupported test mode: %s\n' "$mode" >&2
    exit 2
    ;;
esac
