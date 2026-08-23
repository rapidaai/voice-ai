#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
previous_release_ref="${PREVIOUS_RELEASE_REF:?PREVIOUS_RELEASE_REF is required}"
postgres_bin="${POSTGRES_BIN:-/Applications/Postgres.app/Contents/Versions/16/bin}"
work_dir="$(mktemp -d "/tmp/voice-ai-rfc-0001-rollback.XXXXXX")"
data_dir="${work_dir}/data"
socket_dir="${work_dir}/socket"
log_path="${work_dir}/postgres.log"
source_dir="${work_dir}/previous-release"
go_tmp_dir="${work_dir}/go-tmp"
mkdir -p "${socket_dir}" "${source_dir}" "${go_tmp_dir}"
redis_pid=""
web_pid=""

if [[ ! -x "${postgres_bin}/postgres" ]]; then
  postgres_bin="$(dirname "$(command -v postgres || true)")"
fi
for command in curl git migrate redis-server "${postgres_bin}/initdb" "${postgres_bin}/pg_ctl" "${postgres_bin}/psql" "${postgres_bin}/createdb" "${postgres_bin}/dropdb" "${postgres_bin}/pg_dump" "${postgres_bin}/pg_restore"; do
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
git -C "${repository_root}" rev-parse --verify "${previous_release_ref}^{commit}" >/dev/null

free_port() {
python3 - <<'PY'
import socket
with socket.socket() as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
}

port="$(free_port)"

cleanup() {
	if [[ -n "${web_pid}" ]]; then
		kill "${web_pid}" >/dev/null 2>&1 || true
		wait "${web_pid}" >/dev/null 2>&1 || true
	fi
	if [[ -n "${redis_pid}" ]]; then
		kill "${redis_pid}" >/dev/null 2>&1 || true
		wait "${redis_pid}" >/dev/null 2>&1 || true
	fi
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

drop_database() {
  "${postgres_bin}/dropdb" --force -h 127.0.0.1 -p "${port}" -U postgres "$1"
}

declare -A pre_cleanup_versions=(
  [assistant-api]=59
  [endpoint-api]=5
  [web-api]=10
  [integration-api]=5
)

for service in assistant-api endpoint-api web-api integration-api; do
  database="rollback_${service//-/_}"
  create_database "${database}"
  migrate -path "${repository_root}/api/${service}/migrations" -database "${base_url}/${database}?sslmode=disable" goto "${pre_cleanup_versions[$service]}"
done

psql_exec rollback_assistant_api -c "INSERT INTO public.assistant_tags (id, assistant_id, tag, created_by, updated_by) VALUES (9001, 9002, 'legacy', 71, 72);"
psql_exec rollback_endpoint_api -c "INSERT INTO public.endpoint_tags (id, endpoint_id, tag, created_by, updated_by) VALUES (9001, 9002, 'legacy', 71, 72);"
psql_exec rollback_web_api -c "INSERT INTO public.projects (id, organization_id, name, description, created_by, updated_by) VALUES (9001, 9002, 'legacy', 'legacy smoke', 71, 72);"
psql_exec rollback_web_api -c "INSERT INTO public.service_identities (id, name, signing_key_id, signing_public_key, status, created_actor_type, created_actor_id) VALUES (9101, 'rollback-service', 'rollback-key', E'-----BEGIN PUBLIC KEY-----\\nMCowBQYDK2VwAyEAnEJlElWL5qGQuJoUqZyAGs7qnCWMWJeVeRS+JcIPOMQ=\\n-----END PUBLIC KEY-----\\n', 'ACTIVE', 'system', 1);"
psql_exec rollback_web_api -c "INSERT INTO public.system_identities (id, name, owning_service_id, status, created_actor_type, created_actor_id) VALUES (9201, 'rollback-system', 9101, 'ACTIVE', 'service', 9101);"
psql_exec rollback_integration_api -c "INSERT INTO public.external_audits (id, integration_name, asset_prefix, response_status, time_taken, credential_id, project_id, organization_id, status, metrics, created_actor_type, created_actor_id) VALUES (9001, 'legacy', 'legacy', 0, 0, 1, 1, 1, 'active', '[]', 'unknown', NULL);"

for service in assistant-api endpoint-api web-api integration-api; do
  database="rollback_${service//-/_}"
  "${postgres_bin}/pg_dump" -Fc -h 127.0.0.1 -p "${port}" -U postgres -d "${database}" -f "${work_dir}/${database}.dump"
  steps=1
  if [[ "${service}" == "web-api" ]]; then
    steps=2
  fi
  migrate -path "${repository_root}/api/${service}/migrations" -database "${base_url}/${database}?sslmode=disable" up "${steps}"
done

for database in rollback_assistant_api rollback_endpoint_api rollback_web_api rollback_integration_api; do
  drop_database "${database}"
  create_database "${database}"
  "${postgres_bin}/pg_restore" --exit-on-error -h 127.0.0.1 -p "${port}" -U postgres -d "${database}" "${work_dir}/${database}.dump"
done

[[ "$(psql_exec rollback_assistant_api -Atc "SELECT created_by || ':' || updated_by FROM public.assistant_tags WHERE id=9001;")" == "71:72" ]]
[[ "$(psql_exec rollback_endpoint_api -Atc "SELECT created_by || ':' || updated_by FROM public.endpoint_tags WHERE id=9001;")" == "71:72" ]]
[[ "$(psql_exec rollback_web_api -Atc "SELECT created_by || ':' || updated_by FROM public.projects WHERE id=9001;")" == "71:72" ]]
[[ "$(psql_exec rollback_web_api -Atc "SELECT name || ':' || signing_key_id FROM public.service_identities WHERE id=9101;")" == "rollback-service:rollback-key" ]]
[[ "$(psql_exec rollback_web_api -Atc "SELECT name || ':' || owning_service_id FROM public.system_identities WHERE id=9201;")" == "rollback-system:9101" ]]
[[ "$(psql_exec rollback_web_api -Atc "SELECT count(*) FROM public.service_identities service JOIN public.system_identities system ON system.owning_service_id = service.id WHERE service.id=9101 AND service.status='ACTIVE' AND service.archived_date IS NULL AND system.id=9201 AND system.status='ACTIVE' AND system.archived_date IS NULL;")" == "1" ]]
[[ "$(psql_exec rollback_integration_api -Atc "SELECT count(*) FROM public.external_audits WHERE id=9001;")" == "1" ]]

git -C "${repository_root}" archive "${previous_release_ref}" | tar -x -C "${source_dir}"
mkdir -p "${source_dir}/cmd/rollback-registry-validator"
cat >"${source_dir}/cmd/rollback-registry-validator/main.go" <<'GO'
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

func requiredUint64(name string) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(os.Getenv(name)), 10, 64)
	if err != nil || value == 0 {
		panic(fmt.Sprintf("%s must be a positive bigint", name))
	}
	return value
}

func main() {
	database, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer database.Close()

	serviceID := requiredUint64("RAPIDA_SERVICE_ACTOR_ID")
	serviceName := strings.TrimSpace(os.Getenv("RAPIDA_SERVICE_ACTOR_NAME"))
	keyID := strings.TrimSpace(os.Getenv("RAPIDA_SERVICE_KEY_ID"))
	privateKeyPEM := strings.TrimSpace(os.Getenv("RAPIDA_SERVICE_PRIVATE_KEY"))
	if serviceName == "" || keyID == "" || privateKeyPEM == "" {
		panic("service identity name, key id, and private key are required")
	}

	var publicKeyPEM string
	err = database.QueryRowContext(context.Background(), `
		SELECT signing_public_key
		FROM public.service_identities
		WHERE id = $1 AND name = $2 AND signing_key_id = $3
		  AND status = 'ACTIVE' AND archived_date IS NULL
	`, serviceID, serviceName, keyID).Scan(&publicKeyPEM)
	if err != nil {
		panic(fmt.Errorf("validate configured service identity: %w", err))
	}

	privateBlock, _ := pem.Decode([]byte(privateKeyPEM))
	publicBlock, _ := pem.Decode([]byte(publicKeyPEM))
	if privateBlock == nil || publicBlock == nil {
		panic("configured service signing keys must be PEM encoded")
	}
	parsedPrivateKey, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
	if err != nil {
		panic(err)
	}
	privateKey, ok := parsedPrivateKey.(ed25519.PrivateKey)
	if !ok {
		panic("configured service private key must be Ed25519")
	}
	parsedPublicKey, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		panic(err)
	}
	publicKey, ok := parsedPublicKey.(ed25519.PublicKey)
	if !ok || !bytes.Equal(privateKey.Public().(ed25519.PublicKey), publicKey) {
		panic("configured service private key does not match registry public key")
	}

	systemID := requiredUint64("RAPIDA_SYSTEM_ACTOR_ID")
	systemName := strings.TrimSpace(os.Getenv("RAPIDA_SYSTEM_ACTOR_NAME"))
	if systemName == "" {
		panic("system identity name is required")
	}
	var systemCount int
	err = database.QueryRowContext(context.Background(), `
		SELECT count(*)
		FROM public.system_identities
		WHERE id = $1 AND name = $2 AND owning_service_id = $3
		  AND status = 'ACTIVE' AND archived_date IS NULL
	`, systemID, systemName, serviceID).Scan(&systemCount)
	if err != nil || systemCount != 1 {
		panic(fmt.Errorf("validate configured system identity: count=%d err=%w", systemCount, err))
	}
}
GO
(
  cd "${source_dir}"
  for service in assistant endpoint web integration; do
    GOTMPDIR="${go_tmp_dir}" go build -ldflags='-s -w' -o "${work_dir}/${service}" "./cmd/${service}"
  done
  GOTMPDIR="${go_tmp_dir}" go build -ldflags='-s -w' -o "${work_dir}/registry-startup-validator" ./cmd/rollback-registry-validator
)

DATABASE_URL="${base_url}/rollback_web_api?sslmode=disable" \
RAPIDA_SERVICE_ACTOR_ID=9101 \
RAPIDA_SERVICE_ACTOR_NAME=rollback-service \
RAPIDA_SERVICE_KEY_ID=rollback-key \
RAPIDA_SERVICE_PRIVATE_KEY=$'-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIHx6UXaatRC9vlJO71owzGZ08Jc/vbHevsTRnEN5N40B\n-----END PRIVATE KEY-----' \
RAPIDA_SYSTEM_ACTOR_ID=9201 \
RAPIDA_SYSTEM_ACTOR_NAME=rollback-system \
"${work_dir}/registry-startup-validator"

redis_port="$(free_port)"
web_port="$(free_port)"
redis-server --bind 127.0.0.1 --port "${redis_port}" --save "" --appendonly no >"${work_dir}/redis.log" 2>&1 &
redis_pid=$!

cat >"${work_dir}/web.yml" <<YAML
service_name: web-api
host: 127.0.0.1
port: ${web_port}
log_level: error
secret: rollback-secret
env: development
postgres:
  host: 127.0.0.1
  port: ${port}
  db_name: rollback_web_api
  auth:
    user: postgres
    password: ""
  max_open_connection: 2
  max_ideal_connection: 1
  ssl_mode: disable
redis:
  host: 127.0.0.1
  port: ${redis_port}
  db: 0
  max_connection: 2
  auth:
    user: ""
    password: ""
asset_store:
  storage_type: local
  storage_path_prefix: ${work_dir}/assets
integration:
  host: 127.0.0.1:1
endpoint:
  host: 127.0.0.1:1
assistant:
  host: 127.0.0.1:1
web:
  host: 127.0.0.1:${web_port}
document:
  host: http://127.0.0.1:1
ui:
  host: http://127.0.0.1:1
YAML

ENV_PATH="${work_dir}/web.yml" "${work_dir}/web" -skip-migration >"${work_dir}/web.log" 2>&1 &
web_pid=$!
ready=false
for _ in $(seq 1 100); do
	if curl -fsS "http://127.0.0.1:${web_port}/healthz/" >/dev/null 2>&1; then
		ready=true
		break
	fi
	if ! kill -0 "${web_pid}" >/dev/null 2>&1; then
		cat "${work_dir}/web.log" >&2
		echo "previous Web binary exited before startup validation" >&2
		exit 1
	fi
	sleep 0.1
done
if [[ "${ready}" != "true" ]]; then
	cat "${work_dir}/web.log" >&2
	echo "previous Web binary did not become healthy after registry restoration" >&2
	exit 1
fi

kill "${web_pid}" >/dev/null 2>&1 || true
wait "${web_pid}" >/dev/null 2>&1 || true
web_pid=""

echo "Phase 3 rollback verification restored all four backups, passed registry-backed startup validation, started the previous Web binary successfully, passed legacy read/write smoke checks, and built all ${previous_release_ref} service binaries."
