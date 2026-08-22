package migrations

import (
	"strings"
	"testing"
)

func TestServiceAndSystemIdentityMigration(t *testing.T) {
	up := strings.ToLower(readMigration(t, "000007_create_service_and_system_identities.up.sql"))
	down := strings.ToLower(readMigration(t, "000007_create_service_and_system_identities.down.sql"))
	cleanup := strings.ToLower(readMigration(t, "000012_remove_service_identity_registry.up.sql"))
	for _, required := range []string{
		"create table public.service_identities",
		"name character varying(200) not null unique",
		"signing_key_id character varying(200) not null",
		"signing_public_key text not null",
		"create table public.system_identities",
		"owning_service_id bigint not null references public.service_identities(id)",
		"unique (owning_service_id, name)",
		"created_actor_id bigint not null check (created_actor_id > 0)",
	} {
		if !strings.Contains(up, required) {
			t.Errorf("identity migration missing %q", required)
		}
	}
	if !strings.Contains(down, "drop table public.system_identities") || !strings.Contains(down, "drop table public.service_identities") {
		t.Fatal("identity down migration must drop system identities before service identities")
	}
	systemDrop := strings.Index(cleanup, "drop table if exists public.system_identities")
	serviceDrop := strings.Index(cleanup, "drop table if exists public.service_identities")
	if systemDrop < 0 || serviceDrop < 0 {
		t.Fatal("registry cleanup must drop both identity tables")
	}
	if systemDrop > serviceDrop {
		t.Fatal("registry cleanup must drop system identities before service identities")
	}
}
