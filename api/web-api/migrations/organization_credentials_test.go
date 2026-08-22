package migrations

import (
	"strings"
	"testing"
)

func TestOrganizationCredentialsMigrationStoresOnlyFingerprint(t *testing.T) {
	up := readMigration(t, "000006_create_organization_credentials.up.sql")
	down := readMigration(t, "000006_create_organization_credentials.down.sql")
	lower := strings.ToLower(up)

	for _, required := range []string{
		"create table public.organization_credentials",
		"id bigint primary key check (id > 0)",
		"organization_id bigint not null check (organization_id > 0)",
		"key character varying(200) not null unique",
		"created_actor_type character varying(32) not null",
		"created_actor_id bigint not null check (created_actor_id > 0)",
		"updated_actor_type character varying(32)",
		"updated_actor_id bigint",
		"archived_date timestamp without time zone",
	} {
		if !strings.Contains(lower, required) {
			t.Errorf("migration missing %q", required)
		}
	}
	if strings.Contains(lower, "raw_key") || strings.Contains(lower, "plain_key") {
		t.Fatal("migration contains an explicit raw credential column")
	}
	for _, actorType := range []string{"'user'", "'project'", "'organization'", "'service'", "'system'"} {
		if !strings.Contains(lower, actorType) {
			t.Errorf("migration does not constrain actor type %s", actorType)
		}
	}
	if strings.Contains(lower, "'unknown'") {
		t.Fatal("new organization credentials must not accept unknown actors")
	}
	if !strings.Contains(strings.ToLower(down), "drop table public.organization_credentials") {
		t.Fatal("down migration does not drop organization_credentials")
	}
}
