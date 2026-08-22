#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
postgres_bin="${POSTGRES_BIN:-/Applications/Postgres.app/Contents/Versions/16/bin}"
work_dir="$(mktemp -d "/tmp/voice-ai-rfc-0001-postgres.XXXXXX")"
data_dir="${work_dir}/data"
socket_dir="${work_dir}/socket"
log_path="${work_dir}/postgres.log"
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
"${postgres_bin}/pg_ctl" -D "${data_dir}" -l "${log_path}" -o "-F -p ${port} -k ${socket_dir}" start >/dev/null
base_url="postgres://postgres@127.0.0.1:${port}"

psql_exec() {
  local database="$1"
  shift
  "${postgres_bin}/psql" -v ON_ERROR_STOP=1 -h 127.0.0.1 -p "${port}" -U postgres -d "${database}" "$@"
}

create_database() {
  "${postgres_bin}/createdb" -h 127.0.0.1 -p "${port}" -U postgres "$1"
}

verify_history() {
  local service="$1"
  local final_version="$2"
  local database="phase3_${service//-/_}"
  local migrations="${repository_root}/api/${service}/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"

  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" up

  local version
  version="$(migrate -path "${migrations}" -database "${database_url}" version 2>&1)"
  if [[ "${version}" != "${final_version}" ]]; then
    echo "${service}: migration version ${version}, want ${final_version}" >&2
    exit 1
  fi

  local legacy_columns
  legacy_columns="$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND column_name IN ('created_by','updated_by');")"
  if [[ "${legacy_columns}" != "0" ]]; then
    echo "${service}: ${legacy_columns} legacy audit columns remain" >&2
    exit 1
  fi

	if [[ "${service}" == "web-api" ]]; then
		local registry_tables
		registry_tables="$(psql_exec "${database}" -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('service_identities','system_identities');")"
		if [[ "${registry_tables}" != "0" ]]; then
			echo "web-api: ${registry_tables} service identity registry tables remain" >&2
			exit 1
		fi
	fi

  local invalid_constraints
  invalid_constraints="$(psql_exec "${database}" -Atc "SELECT count(*) FROM pg_constraint WHERE connamespace='public'::regnamespace AND conname LIKE 'audit_%_actor_pair' AND NOT convalidated;")"
  if [[ "${invalid_constraints}" != "0" ]]; then
    echo "${service}: ${invalid_constraints} actor constraints remain unvalidated" >&2
    exit 1
  fi

  local metric_summary
  metric_summary="$(psql_exec "${database}" -Atc "SELECT count(*), coalesce(sum(failed_rows), 0), coalesce(sum(remaining_rows), 0) FROM public.audit_actor_migration_metrics;")"
  if [[ "${metric_summary}" == 0\|* || "${metric_summary}" != *"|0|0" ]]; then
    echo "${service}: invalid migration metric summary ${metric_summary}" >&2
    exit 1
  fi

  echo "${service}: full history reached version ${final_version}"
}

verify_interrupted_resume() {
  local database="phase3_interrupted_resume"
  local migrations="${repository_root}/api/integration-api/migrations"
  local database_url="${base_url}/${database}?sslmode=disable"
  create_database "${database}"
  migrate -path "${migrations}" -database "${database_url}" goto 3

  psql_exec "${database}" <<'SQL'
INSERT INTO public.external_audits (
  id, integration_name, asset_prefix, response_status, time_taken,
  credential_id, project_id, organization_id, status, metrics
)
SELECT id, 'resume-test', 'resume-test', 0, 0, 1, 1, 1, 'active', '[]'::json
FROM generate_series(1, 25000) AS id;

CREATE FUNCTION public.pause_second_audit_batch()
RETURNS trigger
LANGUAGE plpgsql
AS $function$
BEGIN
  IF NEW.id = 10001 THEN
    PERFORM pg_sleep(30);
  END IF;
  RETURN NEW;
END
$function$;

CREATE TRIGGER pause_second_audit_batch
BEFORE UPDATE ON public.external_audits
FOR EACH ROW EXECUTE FUNCTION public.pause_second_audit_batch();
SQL

  migrate -path "${migrations}" -database "${database_url}" up 1 >"${work_dir}/interrupted-migrate.log" 2>&1 &
  local migrate_pid=$!
  local converted=0
  for _ in $(seq 1 100); do
    converted="$(psql_exec "${database}" -Atc "SELECT count(*) FROM public.external_audits WHERE created_actor_type IS NOT NULL;")"
    if [[ "${converted}" == "10000" ]]; then
      break
    fi
    sleep 0.1
  done
  if [[ "${converted}" != "10000" ]]; then
    echo "interrupted backfill committed ${converted} rows before pause, want 10000" >&2
    kill "${migrate_pid}" >/dev/null 2>&1 || true
    wait "${migrate_pid}" >/dev/null 2>&1 || true
    exit 1
  fi
  kill "${migrate_pid}" >/dev/null 2>&1 || true
  wait "${migrate_pid}" >/dev/null 2>&1 || true

  psql_exec "${database}" <<'SQL'
DROP TRIGGER pause_second_audit_batch ON public.external_audits;
DROP FUNCTION public.pause_second_audit_batch();
SQL
  migrate -path "${migrations}" -database "${database_url}" force 3
  migrate -path "${migrations}" -database "${database_url}" up

  converted="$(psql_exec "${database}" -Atc "SELECT count(*) FROM public.external_audits WHERE created_actor_type = 'unknown' AND created_actor_id IS NULL;")"
  if [[ "${converted}" != "25000" ]]; then
    echo "resumed backfill converted ${converted} rows, want 25000" >&2
    exit 1
  fi
  local metric_summary
  metric_summary="$(psql_exec "${database}" -Atc "SELECT processed_rows, failed_rows, remaining_rows FROM public.audit_actor_migration_metrics WHERE table_name = 'external_audits';")"
  if [[ "${metric_summary}" != "25000|0|0" ]]; then
    echo "resumed backfill metric summary ${metric_summary}, want 25000|0|0" >&2
    exit 1
  fi
  if ! rg -q 'audit actor backfill complete table=external_audits processed=' "${log_path}"; then
    echo "PostgreSQL log does not contain the expected per-table backfill summary" >&2
    exit 1
  fi
  echo "integration-api: interrupted backfill preserved 10000 rows and resumed to 25000"
}

verify_history assistant-api 60
verify_history endpoint-api 6
verify_history web-api 12
verify_history integration-api 6
verify_interrupted_resume

echo "Phase 3 PostgreSQL migration verification passed."
