#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
postgres_bin="${POSTGRES_BIN:-/Applications/Postgres.app/Contents/Versions/16/bin}"
work_dir="$(mktemp -d "/tmp/voice-ai-rfc-0003-postgres.XXXXXX")"
data_dir="${work_dir}/data"
socket_dir="${work_dir}/socket"
mkdir -p "${socket_dir}"

if [[ ! -x "${postgres_bin}/postgres" ]]; then
  postgres_bin="$(dirname "$(command -v postgres || true)")"
fi
for command in migrate "${postgres_bin}/initdb" "${postgres_bin}/pg_ctl" "${postgres_bin}/psql" "${postgres_bin}/createdb"; do
  if [[ "${command}" == */* ]]; then
    [[ -x "${command}" ]] || {
      echo "missing required command: ${command}" >&2
      exit 1
    }
  else
    command -v "${command}" >/dev/null || {
      echo "missing required command: ${command}" >&2
      exit 1
    }
  fi
done

port="$(python3 - <<'PY'
import socket
with socket.socket() as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
)"

cleanup() {
  "${postgres_bin}/pg_ctl" -D "${data_dir}" stop -m immediate >/dev/null 2>&1 || true
  rm -rf "${work_dir}"
}
trap cleanup EXIT

"${postgres_bin}/initdb" -D "${data_dir}" -A trust -U postgres --no-locale >/dev/null
"${postgres_bin}/pg_ctl" -D "${data_dir}" -l "${work_dir}/postgres.log" -o "-F -p ${port} -k ${socket_dir}" start >/dev/null
base_url="postgres://postgres@127.0.0.1:${port}"
migration_metrics_table="audit_actor_migration_"'metrics'
backfill_procedure_pattern="backfill_"'(assistant|endpoint|integration|web)'"_audit_actor_identity"

psql_exec() {
  local database="$1"
  shift
  "${postgres_bin}/psql" -v ON_ERROR_STOP=1 -h 127.0.0.1 -p "${port}" -U postgres -d "${database}" "$@"
}

create_database() {
  "${postgres_bin}/createdb" -h 127.0.0.1 -p "${port}" -U postgres "$1"
}

assert_equal() {
  local actual="$1"
  local expected="$2"
  local message="$3"
  if [[ "${actual}" != "${expected}" ]]; then
    echo "${message}: got ${actual}, want ${expected}" >&2
    exit 1
  fi
}

expect_psql_failure() {
  local database="$1"
  local sql="$2"
  if psql_exec "${database}" -c "${sql}" >/dev/null 2>&1; then
    echo "expected SQL failure in ${database}: ${sql}" >&2
    exit 1
  fi
}

assert_lock_timeout_pair() {
  local file="$1"
  assert_equal "$(grep -Fc "SET lock_timeout = '5s';" "${file}")" "1" "${file} lock timeout"
  assert_equal "$(grep -Fc 'RESET lock_timeout;' "${file}")" "1" "${file} lock timeout reset"
}

assert_irreversible() {
  local file="$1"
  rg -q 'RAISE EXCEPTION' "${file}"
  rg -qi 'backup set' "${file}"
}

assert_update_inventory() {
  local file="$1"
  shift
  local expected_count="$#"
  local actual_count
  actual_count="$(grep -Ec '^UPDATE public\.[a-z0-9_]+$' "${file}")"
  assert_equal "${actual_count}" "${expected_count}" "${file} update count"
  for table in "$@"; do
    assert_equal "$(grep -Fxc "UPDATE public.${table}" "${file}")" "1" "${file} update inventory for ${table}"
  done
}

assert_preflight_inventory() {
  local file="$1"
  shift
  local expected_count="$#"
  local actual_count
  actual_count="$(grep -Ec 'EXISTS \(SELECT 1 FROM public\.[a-z0-9_]+' "${file}")"
  assert_equal "${actual_count}" "${expected_count}" "${file} preflight count"
  for table in "$@"; do
    assert_equal "$(grep -Ec "EXISTS \\(SELECT 1 FROM public\\.${table} WHERE" "${file}")" "1" "${file} preflight inventory for ${table}"
  done
}

assistant_tables=(
  assistant_api_deployments assistant_configuration_options assistant_configurations
  assistant_conversation_action_metrics assistant_conversation_arguments
  assistant_conversation_message_metadata assistant_conversation_message_metrics
  assistant_conversation_messages assistant_conversation_metadata assistant_conversation_metrics
  assistant_conversation_options assistant_conversation_recordings assistant_conversations
  assistant_debugger_deployments assistant_deployment_audio_options assistant_deployment_audios
  assistant_deployment_telephony_options assistant_deployment_whatsapp_options
  assistant_knowledge_logs assistant_knowledge_reranker_options assistant_knowledges
  assistant_phone_deployments assistant_provider_agentflows assistant_provider_agentkits
  assistant_provider_model_options assistant_provider_models assistant_provider_websockets
  assistant_tags assistant_tool_logs assistant_tool_options assistant_tools
  assistant_web_plugin_deployments assistant_http_logs assistant_whatsapp_deployments assistants
  knowledge_document_process_rules knowledge_documents knowledge_embedding_model_options
  knowledge_logs knowledge_tags knowledges
)
endpoint_tables=(
  endpoint_cachings endpoint_log_arguments endpoint_log_metadata endpoint_log_metrics
  endpoint_log_options endpoint_provider_model_options endpoint_provider_models endpoint_retries
  endpoint_tags endpoints
)
integration_tables=(external_audits external_audit_metadata)
web_tables=(
  notification_settings organizations project_credentials projects user_auth_tokens user_auths
  user_feature_permissions user_organization_roles user_project_roles user_roles vaults
)

assert_static_contracts() {
  local migration_files
  migration_files="$(find "${repository_root}/api" -path '*/migrations/*.sql' -type f -print)"
  if rg -n "${migration_metrics_table}|${backfill_procedure_pattern}" ${migration_files}; then
    echo 'obsolete audit backfill infrastructure remains' >&2
    exit 1
  fi

  assert_update_inventory "${repository_root}/api/assistant-api/migrations/000058_run_audit_actor_backfill.up.sql" "${assistant_tables[@]}"
  assert_update_inventory "${repository_root}/api/endpoint-api/migrations/000004_run_audit_actor_backfill.up.sql" "${endpoint_tables[@]}"
  assert_update_inventory "${repository_root}/api/integration-api/migrations/000004_run_audit_actor_backfill.up.sql" "${integration_tables[@]}"
  assert_update_inventory "${repository_root}/api/web-api/migrations/000009_run_audit_actor_backfill.up.sql" "${web_tables[@]}"
  assert_preflight_inventory "${repository_root}/api/assistant-api/migrations/000058_run_audit_actor_backfill.up.sql" "${assistant_tables[@]}"
  assert_preflight_inventory "${repository_root}/api/endpoint-api/migrations/000004_run_audit_actor_backfill.up.sql" "${endpoint_tables[@]}"
  assert_preflight_inventory "${repository_root}/api/web-api/migrations/000009_run_audit_actor_backfill.up.sql" "${web_tables[@]}"

  for file in \
    "${repository_root}/api/assistant-api/migrations/000055_expand_audit_actor_identity.up.sql" \
    "${repository_root}/api/assistant-api/migrations/000055_expand_audit_actor_identity.down.sql" \
    "${repository_root}/api/assistant-api/migrations/000058_run_audit_actor_backfill.up.sql" \
    "${repository_root}/api/endpoint-api/migrations/000002_expand_audit_actor_identity.up.sql" \
    "${repository_root}/api/endpoint-api/migrations/000002_expand_audit_actor_identity.down.sql" \
    "${repository_root}/api/endpoint-api/migrations/000004_run_audit_actor_backfill.up.sql" \
    "${repository_root}/api/integration-api/migrations/000002_expand_audit_actor_identity.up.sql" \
    "${repository_root}/api/integration-api/migrations/000002_expand_audit_actor_identity.down.sql" \
    "${repository_root}/api/integration-api/migrations/000004_run_audit_actor_backfill.up.sql" \
    "${repository_root}/api/web-api/migrations/000005_expand_audit_actor_identity.up.sql" \
    "${repository_root}/api/web-api/migrations/000005_expand_audit_actor_identity.down.sql" \
    "${repository_root}/api/web-api/migrations/000009_run_audit_actor_backfill.up.sql" \
    "${repository_root}/api/web-api/migrations/000007_create_service_and_system_identities.up.sql" \
    "${repository_root}/api/web-api/migrations/000007_create_service_and_system_identities.down.sql"; do
    assert_lock_timeout_pair "${file}"
  done

  for file in \
    "${repository_root}/api/assistant-api/migrations/000058_run_audit_actor_backfill.down.sql" \
    "${repository_root}/api/assistant-api/migrations/000060_remove_legacy_audit_identity.down.sql" \
    "${repository_root}/api/endpoint-api/migrations/000004_run_audit_actor_backfill.down.sql" \
    "${repository_root}/api/endpoint-api/migrations/000006_remove_legacy_audit_identity.down.sql" \
    "${repository_root}/api/integration-api/migrations/000004_run_audit_actor_backfill.down.sql" \
    "${repository_root}/api/integration-api/migrations/000006_remove_legacy_audit_identity.down.sql" \
    "${repository_root}/api/web-api/migrations/000009_run_audit_actor_backfill.down.sql" \
    "${repository_root}/api/web-api/migrations/000011_remove_legacy_audit_identity.down.sql" \
    "${repository_root}/api/web-api/migrations/000012_remove_service_identity_registry.down.sql"; do
    assert_irreversible "${file}"
  done
}

table_list_sql() {
  local quoted=""
  local table
  for table in "$@"; do
    [[ -z "${quoted}" ]] || quoted+=","
    quoted+="'${table}'"
  done
  printf '%s' "${quoted}"
}

assert_actor_columns() {
  local database="$1"
  shift
  local quoted
  quoted="$(table_list_sql "$@")"
  local invalid
  invalid="$(psql_exec "${database}" -Atc "
    WITH expected(table_name) AS (SELECT unnest(ARRAY[${quoted}])),
    counts AS (
      SELECT expected.table_name, count(columns.column_name) AS actor_columns
      FROM expected
      LEFT JOIN information_schema.columns columns
        ON columns.table_schema = 'public'
       AND columns.table_name = expected.table_name
       AND columns.column_name IN ('created_actor_type','created_actor_id','updated_actor_type','updated_actor_id')
      GROUP BY expected.table_name
    )
    SELECT count(*) FROM counts WHERE actor_columns <> 4;")"
  assert_equal "${invalid}" "0" "${database} actor column coverage"
}

assert_final_actor_contracts() {
  local database="$1"
  shift
  local quoted
  quoted="$(table_list_sql "$@")"
  local invalid
  invalid="$(psql_exec "${database}" -Atc "
    WITH expected(table_name) AS (SELECT unnest(ARRAY[${quoted}]))
    SELECT count(*)
    FROM expected
    WHERE (
      SELECT count(*)
      FROM pg_constraint
      WHERE conrelid = to_regclass('public.' || expected.table_name)
        AND conname IN ('audit_created_actor_pair', 'audit_updated_actor_pair')
        AND convalidated
    ) <> 2
    OR (
      SELECT count(*)
      FROM pg_trigger
      WHERE tgrelid = to_regclass('public.' || expected.table_name)
        AND tgname = 'audit_created_actor_immutable'
        AND NOT tgisinternal
    ) <> 1;")"
  assert_equal "${invalid}" "0" "${database} actor constraints and triggers"
}

verify_expansion() {
  local service="$1"
  local pre_expand_version="$2"
  local seed_sql="$3"
  local data_sql="$4"
  local expected_data="$5"
  shift 5
  local database="phase3_expansion_${service//-/_}"
  local migrations="${repository_root}/api/${service}/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"
  local quoted
  quoted="$(table_list_sql "$@")"

  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" goto "${pre_expand_version}" >/dev/null
  psql_exec "${database}" -c "${seed_sql}" >/dev/null
  local before_columns
  before_columns="$(psql_exec "${database}" -Atc "SELECT table_name || '|' || column_name || '|' || data_type || '|' || is_nullable || '|' || coalesce(column_default, '') FROM information_schema.columns WHERE table_schema='public' AND table_name IN (${quoted}) ORDER BY table_name, ordinal_position;")"
  migrate -path "${migrations}" -database "${database_url}" up 1 >/dev/null
  local after_columns
  after_columns="$(psql_exec "${database}" -Atc "SELECT table_name || '|' || column_name || '|' || data_type || '|' || is_nullable || '|' || coalesce(column_default, '') FROM information_schema.columns WHERE table_schema='public' AND table_name IN (${quoted}) AND column_name NOT IN ('created_actor_type','created_actor_id','updated_actor_type','updated_actor_id') ORDER BY table_name, ordinal_position;")"
  assert_equal "${after_columns}" "${before_columns}" "${service} additive expansion columns"
  assert_actor_columns "${database}" "$@"
  assert_equal "$(psql_exec "${database}" -Atc "${data_sql}")" "${expected_data}" "${service} expansion data preservation"
}

verify_history() {
  local service="$1"
  local final_version="$2"
  shift 2
  local database="phase3_${service//-/_}"
  local migrations="${repository_root}/api/${service}/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"

  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" up >/dev/null

  assert_equal "$(migrate -path "${migrations}" -database "${database_url}" version 2>&1)" "${final_version}" "${service} migration version"
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND column_name IN ('created_by','updated_by');")" "0" "${service} legacy audit columns"
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM pg_constraint WHERE connamespace='public'::regnamespace AND conname LIKE 'audit_%_actor_pair' AND NOT convalidated;")" "0" "${service} unvalidated actor constraints"
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='${migration_metrics_table}';")" "0" "${service} migration metrics table"
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM pg_proc JOIN pg_namespace ON pg_namespace.oid=pg_proc.pronamespace WHERE pg_namespace.nspname='public' AND pg_proc.proname LIKE 'backfill_%_audit_actor_identity';")" "0" "${service} backfill procedures"
  assert_actor_columns "${database}" "$@"
  assert_final_actor_contracts "${database}" "$@"

  if [[ "${service}" == "assistant-api" ]]; then
    assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='call_contexts' AND column_name IN ('auth_token','auth_user_id','auth_actor_type','auth_actor_id');")" "4" "assistant-api call context authentication columns"
  fi
  if [[ "${service}" == "web-api" ]]; then
    assert_actor_columns "${database}" organization_credentials
    assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM pg_trigger WHERE tgrelid='public.organization_credentials'::regclass AND tgname='audit_created_actor_immutable' AND NOT tgisinternal;")" "1" "web-api organization credential immutability trigger"
    assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('service_identities','system_identities');")" "0" "web-api registry tables"
    assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='organization_credentials' AND column_name IN ('raw_key','plain_key','secret','private_key');")" "0" "web-api raw credential columns"
    assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM pg_constraint WHERE conrelid='public.organization_credentials'::regclass AND contype='u' AND pg_get_constraintdef(oid) LIKE '%(key)%';")" "1" "web-api organization credential key uniqueness"
  fi

  echo "${service}: full history reached version ${final_version}"
}

verify_expansions() {
  verify_expansion \
    assistant-api 54 \
    "INSERT INTO public.assistant_conversation_recordings (id, project_id, organization_id, assistant_id, assistant_conversation_id, assistant_recording_url, user_recording_url, created_by, updated_by) VALUES (901, 1, 1, 1, 1, 'assistant', 'user', 11, 12);" \
    "SELECT project_id || ':' || organization_id || ':' || created_by || ':' || updated_by FROM public.assistant_conversation_recordings WHERE id=901;" \
    "1:1:11:12" \
    "${assistant_tables[@]}"
  verify_expansion \
    endpoint-api 1 \
    "INSERT INTO public.endpoint_provider_models (id, created_by, updated_by) VALUES (901, 21, 22);" \
    "SELECT created_by || ':' || updated_by FROM public.endpoint_provider_models WHERE id=901;" \
    "21:22" \
    "${endpoint_tables[@]}"
  verify_expansion \
    integration-api 1 \
    "INSERT INTO public.external_audits (id, integration_name, asset_prefix, response_status, time_taken, credential_id, project_id, organization_id, status, metrics) VALUES (901, 'expand', 'expand', 0, 0, 1, 1, 1, 'active', '[]');" \
    "SELECT integration_name || ':' || project_id FROM public.external_audits WHERE id=901;" \
    "expand:1" \
    "${integration_tables[@]}"
  verify_expansion \
    web-api 4 \
    "INSERT INTO public.organizations (id, name, description, size, industry, contact, created_by, updated_by) VALUES (901, 'expand', 'expand', 'expand', 'expand', 'expand', 31, 32);" \
    "SELECT name || ':' || created_by || ':' || updated_by FROM public.organizations WHERE id=901;" \
    "expand:31:32" \
    "${web_tables[@]}"
}

verify_user_backfill() {
  local service="$1"
  local pre_run_version="$2"
  local run_steps="$3"
  local insert_sql="$4"
  local result_sql="$5"
  local expected="$6"
  local database="phase3_mapping_${service//-/_}"
  local migrations="${repository_root}/api/${service}/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"

  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" goto "${pre_run_version}" >/dev/null
  psql_exec "${database}" -c "${insert_sql}" >/dev/null
  migrate -path "${migrations}" -database "${database_url}" up "${run_steps}" >/dev/null
  assert_equal "$(psql_exec "${database}" -Atc "${result_sql}")" "${expected}" "${service} user actor backfill"
}

verify_backfill_data() {
  verify_user_backfill \
    assistant-api 57 1 \
    "INSERT INTO public.assistant_conversation_recordings (id, project_id, organization_id, assistant_id, assistant_conversation_id, assistant_recording_url, user_recording_url, created_by, updated_by) VALUES (1, 1, 1, 1, 1, 'one', 'one', 101, NULL), (2, 1, 1, 1, 2, 'two', 'two', 102, 103);" \
    "SELECT string_agg(created_actor_type || ':' || created_actor_id || ':' || coalesce(updated_actor_type, 'null') || ':' || coalesce(updated_actor_id::text, 'null'), ',' ORDER BY id) FROM public.assistant_conversation_recordings;" \
    "user:101:null:null,user:102:user:103"

  verify_user_backfill \
    endpoint-api 3 1 \
    "INSERT INTO public.endpoint_provider_models (id, created_by, updated_by) VALUES (1, 201, NULL), (2, 202, 203);" \
    "SELECT string_agg(created_actor_type || ':' || created_actor_id || ':' || coalesce(updated_actor_type, 'null') || ':' || coalesce(updated_actor_id::text, 'null'), ',' ORDER BY id) FROM public.endpoint_provider_models;" \
    "user:201:null:null,user:202:user:203"

  verify_user_backfill \
    web-api 8 1 \
    "INSERT INTO public.organizations (id, name, description, size, industry, contact, created_by, updated_by) VALUES (1, 'one', 'one', 'one', 'one', 'one', 301, 302), (2, 'two', 'two', 'two', 'two', 'two', 303, 304);" \
    "SELECT string_agg(created_actor_type || ':' || created_actor_id || ':' || updated_actor_type || ':' || updated_actor_id, ',' ORDER BY id) FROM public.organizations;" \
    "user:301:user:302,user:303:user:304"

  local database=phase3_mapping_integration_api
  local migrations="${repository_root}/api/integration-api/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"
  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" goto 3 >/dev/null
  psql_exec "${database}" -c "INSERT INTO public.external_audits (id, integration_name, asset_prefix, response_status, time_taken, credential_id, project_id, organization_id, status, metrics) VALUES (1, 'legacy', 'legacy', 0, 0, 1, 1, 1, 'active', '[]');" >/dev/null
  migrate -path "${migrations}" -database "${database_url}" up 1 >/dev/null
  assert_equal "$(psql_exec "${database}" -Atc "SELECT created_actor_type || ':' || coalesce(created_actor_id::text, 'null') FROM public.external_audits WHERE id=1;")" "unknown:null" "integration-api historical actor backfill"
}

verify_invalid_case() {
  local service="$1"
  local pre_run_version="$2"
  local insert_sql="$3"
  local untouched_sql="$4"
  local database="phase3_invalid_${service//-/_}"
  local migrations="${repository_root}/api/${service}/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"
  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" goto "${pre_run_version}" >/dev/null
  psql_exec "${database}" -c "${insert_sql}" >/dev/null
  if migrate -path "${migrations}" -database "${database_url}" up 1 >/dev/null 2>&1; then
    echo "${service} invalid legacy IDs unexpectedly migrated" >&2
    exit 1
  fi
  assert_equal "$(psql_exec "${database}" -Atc "${untouched_sql}")" "0" "${service} preflight partial update count"
}

verify_invalid_preflights() {
  verify_invalid_case assistant-api 57 \
    "INSERT INTO public.assistant_conversation_recordings (id, project_id, organization_id, assistant_id, assistant_conversation_id, assistant_recording_url, user_recording_url, created_by, updated_by) VALUES (1, 1, 1, 1, 1, 'valid', 'valid', 401, NULL), (2, 1, 1, 1, 2, 'null-created', 'null-created', NULL, NULL);" \
    "SELECT count(*) FROM public.assistant_conversation_recordings WHERE created_actor_type IS NOT NULL OR created_actor_id IS NOT NULL OR updated_actor_type IS NOT NULL OR updated_actor_id IS NOT NULL;"
  verify_invalid_case endpoint-api 3 \
    "INSERT INTO public.endpoint_provider_models (id, created_by, updated_by) VALUES (1, 501, NULL), (2, 0, NULL);" \
    "SELECT count(*) FROM public.endpoint_provider_models WHERE created_actor_type IS NOT NULL OR created_actor_id IS NOT NULL OR updated_actor_type IS NOT NULL OR updated_actor_id IS NOT NULL;"
  verify_invalid_case web-api 8 \
    "INSERT INTO public.organizations (id, name, description, size, industry, contact, created_by, updated_by) VALUES (1, 'valid', 'valid', 'valid', 'valid', 'valid', 601, 602), (2, 'invalid', 'invalid', 'invalid', 'invalid', 'invalid', 603, -1);" \
    "SELECT count(*) FROM public.organizations WHERE created_actor_type IS NOT NULL OR created_actor_id IS NOT NULL OR updated_actor_type IS NOT NULL OR updated_actor_id IS NOT NULL;"
}

verify_final_actor_behavior() {
  psql_exec phase3_assistant_api -c "INSERT INTO public.assistant_conversation_recordings (id, project_id, organization_id, assistant_id, assistant_conversation_id, assistant_recording_url, user_recording_url, created_actor_type, created_actor_id) VALUES (801, 1, 1, 1, 1, 'assistant', 'user', 'user', 1);" >/dev/null
  expect_psql_failure phase3_assistant_api "UPDATE public.assistant_conversation_recordings SET created_actor_id=2 WHERE id=801;"
  expect_psql_failure phase3_assistant_api "UPDATE public.assistant_conversation_recordings SET updated_actor_type='user', updated_actor_id=0 WHERE id=801;"

  psql_exec phase3_endpoint_api -c "INSERT INTO public.endpoint_provider_models (id, created_actor_type, created_actor_id) VALUES (801, 'user', 1);" >/dev/null
  expect_psql_failure phase3_endpoint_api "UPDATE public.endpoint_provider_models SET created_actor_id=2 WHERE id=801;"
  expect_psql_failure phase3_endpoint_api "UPDATE public.endpoint_provider_models SET updated_actor_type='user', updated_actor_id=0 WHERE id=801;"

  psql_exec phase3_web_api -c "INSERT INTO public.organizations (id, name, description, size, industry, contact, created_actor_type, created_actor_id) VALUES (801, 'runtime', 'runtime', 'runtime', 'runtime', 'runtime', 'user', 1);" >/dev/null
  expect_psql_failure phase3_web_api "UPDATE public.organizations SET created_actor_id=2 WHERE id=801;"
  expect_psql_failure phase3_web_api "UPDATE public.organizations SET updated_actor_type='user', updated_actor_id=0 WHERE id=801;"

  psql_exec phase3_integration_api -c "INSERT INTO public.external_audits (id, integration_name, asset_prefix, response_status, time_taken, credential_id, project_id, organization_id, status, metrics, created_actor_type, created_actor_id) VALUES (801, 'runtime', 'runtime', 0, 0, 1, 1, 1, 'active', '[]', 'service', 1);" >/dev/null
  expect_psql_failure phase3_integration_api "UPDATE public.external_audits SET created_actor_id=2 WHERE id=801;"
  expect_psql_failure phase3_integration_api "UPDATE public.external_audits SET updated_actor_type='user', updated_actor_id=0 WHERE id=801;"
}

verify_callcontext_rollback() {
  local database=phase3_callcontext_rollback
  local migrations="${repository_root}/api/assistant-api/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"
  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" goto 56 >/dev/null
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='call_contexts' AND column_name IN ('auth_token','auth_user_id','auth_actor_type','auth_actor_id');")" "4" "call context expanded columns"
  migrate -path "${migrations}" -database "${database_url}" down 1 >/dev/null
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='call_contexts' AND column_name='auth_token';")" "1" "call context auth_token after rollback"
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='call_contexts' AND column_name IN ('auth_user_id','auth_actor_type','auth_actor_id');")" "0" "call context snapshot columns after rollback"
}

verify_web_security_migrations() {
  local database=phase3_web_security
  local migrations="${repository_root}/api/web-api/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"
  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" goto 6 >/dev/null

  psql_exec "${database}" -c "INSERT INTO public.organization_credentials (id, organization_id, name, key, created_by, created_actor_type, created_actor_id) VALUES (1, 1, 'one', 'fingerprint', 1, 'user', 1);" >/dev/null
  expect_psql_failure "${database}" "INSERT INTO public.organization_credentials (id, organization_id, name, key, created_by, created_actor_type, created_actor_id) VALUES (0, 1, 'zero-id', 'zero-id', 1, 'user', 1);"
  expect_psql_failure "${database}" "INSERT INTO public.organization_credentials (id, organization_id, name, key, created_by, created_actor_type, created_actor_id) VALUES (2, 0, 'zero-org', 'zero-org', 1, 'user', 1);"
  expect_psql_failure "${database}" "INSERT INTO public.organization_credentials (id, organization_id, name, key, created_by, created_actor_type, created_actor_id) VALUES (3, 1, 'zero-actor', 'zero-actor', 1, 'user', 0);"
  expect_psql_failure "${database}" "INSERT INTO public.organization_credentials (id, organization_id, name, key, created_by, created_actor_type, created_actor_id) VALUES (2, 1, 'two', 'fingerprint', 1, 'user', 1);"
  expect_psql_failure "${database}" "INSERT INTO public.organization_credentials (id, organization_id, name, key, created_by, created_actor_type, created_actor_id) VALUES (4, 1, 'unknown', 'unknown', 1, 'unknown', 1);"

  migrate -path "${migrations}" -database "${database_url}" up 1 >/dev/null
  psql_exec "${database}" -c "INSERT INTO public.service_identities (id, name, signing_key_id, signing_public_key, created_actor_type, created_actor_id) VALUES (10, 'service', 'key', 'public', 'user', 1); INSERT INTO public.system_identities (id, name, owning_service_id, created_actor_type, created_actor_id) VALUES (11, 'system', 10, 'service', 10);" >/dev/null
  expect_psql_failure "${database}" "INSERT INTO public.service_identities (id, name, signing_key_id, signing_public_key, created_actor_type, created_actor_id) VALUES (0, 'zero-service', 'zero-key', 'public', 'user', 1);"
  expect_psql_failure "${database}" "INSERT INTO public.service_identities (id, name, signing_key_id, signing_public_key, created_actor_type, created_actor_id) VALUES (12, 'service', 'other-key', 'public', 'user', 1);"
  expect_psql_failure "${database}" "INSERT INTO public.service_identities (id, name, signing_key_id, signing_public_key, created_actor_type, created_actor_id) VALUES (13, 'missing-key', NULL, 'public', 'user', 1);"
  expect_psql_failure "${database}" "INSERT INTO public.service_identities (id, name, signing_key_id, signing_public_key, created_actor_type, created_actor_id) VALUES (14, 'zero-actor', 'zero-actor-key', 'public', 'user', 0);"
  expect_psql_failure "${database}" "INSERT INTO public.system_identities (id, name, owning_service_id, created_actor_type, created_actor_id) VALUES (12, 'orphan', 999, 'service', 10);"
  expect_psql_failure "${database}" "INSERT INTO public.system_identities (id, name, owning_service_id, created_actor_type, created_actor_id) VALUES (13, 'system', 10, 'service', 10);"
  expect_psql_failure "${database}" "INSERT INTO public.system_identities (id, name, owning_service_id, created_actor_type, created_actor_id) VALUES (14, 'zero-actor', 10, 'service', 0);"
  migrate -path "${migrations}" -database "${database_url}" down 1 >/dev/null
  assert_equal "$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('service_identities','system_identities');")" "0" "web-api registry rollback ordering"
}

assert_static_contracts
verify_expansions
verify_history assistant-api 60 "${assistant_tables[@]}"
verify_history endpoint-api 6 "${endpoint_tables[@]}"
verify_history web-api 12 "${web_tables[@]}"
verify_history integration-api 6 "${integration_tables[@]}"
verify_backfill_data
verify_invalid_preflights
verify_final_actor_behavior
verify_callcontext_rollback
verify_web_security_migrations

echo "Phase 3 PostgreSQL migration verification passed."
