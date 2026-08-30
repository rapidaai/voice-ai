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
DELETE FROM product_usages WHERE organization_id = 1;
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

check_product_usage() {
  product-usage-smoke

  actor_counts=$(psql -v ON_ERROR_STOP=1 -h postgres -U rapida_user -d web_db -Atc \
    "SELECT created_actor_type || '|' || count(*) FROM product_usages WHERE organization_id = 1 AND project_id = 1 AND usage_type = 'llm_duration' AND usages = 1000000000 AND unit = 'nanosecond' GROUP BY created_actor_type ORDER BY created_actor_type")
  expected_actor_counts=$(printf 'project|1\nuser|1')
  if [ "$actor_counts" != "$expected_actor_counts" ]; then
    printf 'product usage persistence check returned %s\n' "$actor_counts" >&2
    return 1
  fi
  echo 'Product usage smoke passed for personal access token and project API key'
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

  echo 'Running product usage smoke tests with both authentication methods'
  check_product_usage

  echo 'All smoke tests passed'
}

case "$mode" in
  integration)
    run_integration_checks
    ;;
  smoke)
    run_smoke_tests
    ;;
  *)
    printf 'unsupported test mode: %s\n' "$mode" >&2
    exit 2
    ;;
esac
