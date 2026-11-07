#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
postgres_bin="${POSTGRES_BIN:-/Applications/Postgres.app/Contents/Versions/16/bin}"
work_dir="$(mktemp -d "/tmp/voice-ai-role-catalog-postgres.XXXXXX")"
data_dir="${work_dir}/data"
socket_dir="${work_dir}/socket"
mkdir -p "${socket_dir}"

if [[ ! -x "${postgres_bin}/postgres" ]]; then
  postgres_bin="$(dirname "$(command -v postgres || true)")"
fi
for command in migrate "${postgres_bin}/initdb" "${postgres_bin}/pg_ctl" "${postgres_bin}/psql" "${postgres_bin}/createdb"; do
  if [[ "${command}" == */* ]]; then
    [[ -x "${command}" ]] || { echo "missing required command: ${command}" >&2; exit 1; }
  else
    command -v "${command}" >/dev/null || { echo "missing required command: ${command}" >&2; exit 1; }
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
"${postgres_bin}/createdb" -h 127.0.0.1 -p "${port}" -U postgres role_catalog

database_url="postgres://postgres@127.0.0.1:${port}/role_catalog?sslmode=disable"
migrations="${repository_root}/api/web-api/migrations"

psql_exec() {
  "${postgres_bin}/psql" -v ON_ERROR_STOP=1 -h 127.0.0.1 -p "${port}" -U postgres -d role_catalog "$@"
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

expect_failure() {
  local sql="$1"
  if psql_exec -c "${sql}" >/dev/null 2>&1; then
    echo "expected SQL failure: ${sql}" >&2
    exit 1
  fi
}

migrate -path "${migrations}" -database "${database_url}" goto 13 >/dev/null
migrate -path "${migrations}" -database "${database_url}" up 1 >/dev/null

assert_equal "$(psql_exec -Atc "SELECT count(*) FROM permission_definitions")" "64" "permission count"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM role_definitions")" "16" "role count"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM role_permissions")" "186" "role-permission count"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM permission_definitions WHERE scope_type='ORGANIZATION'")" "33" "organization permission count"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM permission_definitions WHERE scope_type='PROJECT'")" "31" "project permission count"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM permission_definitions WHERE token_eligible")" "17" "token-eligible count"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM permission_definitions WHERE created_actor_type='unknown' AND created_actor_id IS NULL")" "64" "permission seed provenance"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM role_definitions WHERE created_actor_type='unknown' AND created_actor_id IS NULL")" "16" "role seed provenance"

permission_digest="$(psql_exec -Atc "
SELECT md5(string_agg(concat_ws('|', id::text, key, scope_type, name, description,
  token_eligible::text, status, created_actor_type, coalesce(created_actor_id::text, 'NULL')),
  E'\\n' ORDER BY id)) FROM permission_definitions;")"
role_digest="$(psql_exec -Atc "
SELECT md5(string_agg(concat_ws('|', id::text, coalesce(organization_id::text, 'NULL'), key,
  name, description, scope_type, subject_type, definition_type, status, created_actor_type,
  coalesce(created_actor_id::text, 'NULL')), E'\\n' ORDER BY id))
FROM role_definitions;")"
role_permission_digest="$(psql_exec -Atc "
SELECT md5(string_agg(concat_ws('|', rp.id::text, rd.key, pd.key, rp.scope_type, rp.subject_type,
  rp.token_eligible::text, rp.status, rp.created_actor_type,
  coalesce(rp.created_actor_id::text, 'NULL')), E'\\n' ORDER BY rp.id))
FROM role_permissions rp
JOIN role_definitions rd ON rd.id=rp.role_definition_id
JOIN permission_definitions pd ON pd.id=rp.permission_definition_id;")"
assert_equal "${permission_digest}" "0abc1d0bd98eb5b35f6359cfe873fa1e" "permission catalog digest"
assert_equal "${role_digest}" "4e86768db57f5f8f6e7d0b7eef24a526" "role catalog digest"
assert_equal "${role_permission_digest}" "6747f21febf8c6781a9d4f61619fd3e4" "role-permission catalog digest"

bundle_mismatches="$(psql_exec -Atc "
WITH expected(role_key, permission_count) AS (VALUES
  ('organization_owner',33),('organization_admin',26),('access_admin',11),
  ('security_admin',7),('credential_admin',11),('billing_admin',3),
  ('organization_auditor',9),('organization_member',2),('project_admin',31),
  ('project_developer',11),('project_operator',9),('project_viewer',5),
  ('project_auditor',8),('runtime_invoker',1),('automation_client',11),('observer_client',8)
), actual AS (
  SELECT rd.key role_key, count(*) permission_count
  FROM role_permissions rp JOIN role_definitions rd ON rd.id=rp.role_definition_id
  WHERE rp.status='ACTIVE' GROUP BY rd.key
)
SELECT count(*) FROM expected e FULL JOIN actual a USING(role_key)
WHERE e.permission_count IS DISTINCT FROM a.permission_count;")"
assert_equal "${bundle_mismatches}" "0" "role bundle counts"

expect_failure "INSERT INTO permission_definitions (id,key,scope_type,name,description,token_eligible,status,created_actor_type) VALUES (9001,'invalid.organization.token','ORGANIZATION','Invalid','Invalid',true,'ACTIVE','unknown')"
expect_failure "INSERT INTO role_definitions (id,key,name,description,scope_type,subject_type,definition_type,status,created_actor_type) VALUES (9002,'invalid_custom','Invalid','Invalid','PROJECT','USER','CUSTOM','ACTIVE','unknown')"
expect_failure "INSERT INTO role_definitions (id,key,name,description,scope_type,subject_type,definition_type,status,created_actor_type) VALUES (9003,'project_admin','Duplicate','Duplicate','PROJECT','USER','BUILT_IN','ACTIVE','unknown')"
expect_failure "INSERT INTO role_permissions (id,role_definition_id,permission_definition_id,scope_type,subject_type,token_eligible,status,created_actor_type) VALUES (9004,114,35,'PROJECT','PROJECT_TOKEN',false,'ACTIVE','unknown')"
expect_failure "INSERT INTO role_permissions (id,role_definition_id,permission_definition_id,scope_type,subject_type,token_eligible,status,created_actor_type) VALUES (9005,114,999999,'PROJECT','PROJECT_TOKEN',true,'ACTIVE','unknown')"
expect_failure "INSERT INTO role_permissions (id,role_definition_id,permission_definition_id,scope_type,subject_type,token_eligible,status,created_actor_type) VALUES (9006,114,46,'PROJECT','PROJECT_TOKEN',true,'ACTIVE','unknown')"
expect_failure "INSERT INTO permission_definitions (id,key,scope_type,name,description,token_eligible,status,created_actor_type,created_actor_id) VALUES (9007,'invalid.audit','PROJECT','Invalid','Invalid',false,'ACTIVE','unknown',1)"
expect_failure "INSERT INTO permission_definitions (id,key,scope_type,name,description,token_eligible,status,created_actor_type) VALUES (9010,'invalid.audit.user','PROJECT','Invalid','Invalid',false,'ACTIVE','user')"
expect_failure "INSERT INTO role_definitions (id,key,name,description,scope_type,subject_type,definition_type,status,created_actor_type) VALUES (9011,'invalid_audit_user','Invalid','Invalid','PROJECT','USER','BUILT_IN','ACTIVE','user')"
expect_failure "INSERT INTO role_permissions (id,role_definition_id,permission_definition_id,scope_type,subject_type,token_eligible,status,created_actor_type) VALUES (9012,114,46,'PROJECT','PROJECT_TOKEN',true,'ARCHIEVE','user')"

for table_and_id in "permission_definitions:1" "role_definitions:101" "role_permissions:1001"; do
  table="${table_and_id%%:*}"
  row_id="${table_and_id##*:}"
  expect_failure "UPDATE ${table} SET updated_actor_type='user', updated_actor_id=NULL WHERE id=${row_id}"
  expect_failure "UPDATE ${table} SET updated_actor_type=NULL, updated_actor_id=1 WHERE id=${row_id}"
done

psql_exec -c "UPDATE permission_definitions SET updated_actor_type='system', updated_actor_id=1 WHERE id=1" >/dev/null
psql_exec -c "UPDATE role_definitions SET updated_actor_type='system', updated_actor_id=1 WHERE id=101" >/dev/null
psql_exec -c "UPDATE role_permissions SET updated_actor_type='system', updated_actor_id=1 WHERE id=1001" >/dev/null

psql_exec -c "INSERT INTO organizations (id,name,description,size,industry,contact,status,created_actor_type,created_actor_id) VALUES (9008,'Catalog verifier','Verifier','test','test','test','ACTIVE','system',1)" >/dev/null
psql_exec -c "INSERT INTO role_definitions (id,organization_id,key,name,description,scope_type,subject_type,definition_type,status,created_actor_type,created_actor_id) VALUES (9009,9008,'custom_verifier','Custom Verifier','Verifier','PROJECT','USER','CUSTOM','ACTIVE','system',1)" >/dev/null
expect_failure "DELETE FROM organizations WHERE id=9008"

migrate -path "${migrations}" -database "${database_url}" down 1 >/dev/null
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename IN ('permission_definitions','role_definitions','role_permissions')")" "0" "catalog tables after down"
assert_equal "$(psql_exec -Atc "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename='organizations'")" "1" "existing tables after down"

"${postgres_bin}/createdb" -h 127.0.0.1 -p "${port}" -U postgres role_catalog_rollback
rollback_database_url="postgres://postgres@127.0.0.1:${port}/role_catalog_rollback?sslmode=disable"
rollback_migrations="${work_dir}/rollback-migrations"
cp -R "${migrations}" "${rollback_migrations}"
sed 's/^COMMIT;$/SELECT 1\/0;\
COMMIT;/' "${rollback_migrations}/000014_create_role_catalog.up.sql" > "${work_dir}/failed-migration.sql"
mv "${work_dir}/failed-migration.sql" "${rollback_migrations}/000014_create_role_catalog.up.sql"
migrate -path "${rollback_migrations}" -database "${rollback_database_url}" goto 13 >/dev/null
if migrate -path "${rollback_migrations}" -database "${rollback_database_url}" up 1 >/dev/null 2>&1; then
  echo "expected the injected migration failure" >&2
  exit 1
fi
rollback_table_count="$("${postgres_bin}/psql" -At -v ON_ERROR_STOP=1 -h 127.0.0.1 -p "${port}" -U postgres -d role_catalog_rollback \
  -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename IN ('permission_definitions','role_definitions','role_permissions')")"
assert_equal "${rollback_table_count}" "0" "catalog tables after failed migration"

echo "role catalog migration verification passed"
