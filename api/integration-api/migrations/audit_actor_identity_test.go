package migrations

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestExpandAuditActorIdentityMigration(t *testing.T) {
	tables := []string{"external_audits", "external_audit_metadata"}
	up := readMigration(t, "000002_expand_audit_actor_identity.up.sql")
	down := readMigration(t, "000002_expand_audit_actor_identity.down.sql")
	validateLockTimeout(t, "up", up)
	validateLockTimeout(t, "down", down)

	for _, table := range tables {
		upBlock := fmt.Sprintf("ALTER TABLE public.%s\n    ADD COLUMN created_actor_type varchar(32),\n    ADD COLUMN created_actor_id bigint,\n    ADD COLUMN updated_actor_type varchar(32),\n    ADD COLUMN updated_actor_id bigint;", table)
		if strings.Count(up, upBlock) != 1 {
			t.Errorf("up migration must add bigint actor columns exactly once for %s", table)
		}

		downBlock := fmt.Sprintf("ALTER TABLE public.%s\n    DROP COLUMN created_actor_type,\n    DROP COLUMN created_actor_id,\n    DROP COLUMN updated_actor_type,\n    DROP COLUMN updated_actor_id;", table)
		if strings.Count(down, downBlock) != 1 {
			t.Errorf("down migration must remove actor columns exactly once for %s", table)
		}
	}

	if strings.Count(up, "ALTER TABLE public.") != len(tables) {
		t.Fatalf("up migration alters %d tables, want %d", strings.Count(up, "ALTER TABLE public."), len(tables))
	}
	if strings.Count(down, "ALTER TABLE public.") != len(tables) {
		t.Fatalf("down migration alters %d tables, want %d", strings.Count(down, "ALTER TABLE public."), len(tables))
	}

	lowerUp := strings.ToLower(up)
	for _, forbidden := range []string{" text", "drop column", "update ", "insert into", " default ", " using "} {
		if strings.Contains(lowerUp, forbidden) {
			t.Errorf("up migration contains forbidden expansion operation %q", forbidden)
		}
	}
}

func validateLockTimeout(t *testing.T, name, sql string) {
	t.Helper()
	if strings.Count(sql, "SET lock_timeout = '5s';") != 1 {
		t.Errorf("%s migration must set a five-second lock timeout exactly once", name)
	}
	if strings.Count(sql, "RESET lock_timeout;") != 1 {
		t.Errorf("%s migration must reset lock_timeout exactly once", name)
	}
}

func readMigration(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(contents)
}
