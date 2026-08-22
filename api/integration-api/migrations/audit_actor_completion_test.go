package migrations

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompleteAuditActorMigration(t *testing.T) {
	tables := []string{
		"external_audits",
		"external_audit_metadata",
	}
	conversion := readMigration(t, "000003_convert_audit_actor_identity.up.sql")
	execution := readMigration(t, "000004_run_audit_actor_backfill.up.sql")
	finalization := readMigration(t, "000005_finalize_audit_actor_identity.up.sql")
	cleanup := readMigration(t, "000006_remove_legacy_audit_identity.up.sql")
	schema := conversion + finalization

	for _, required := range []string{
		"LIMIT 10000",
		"FOR UPDATE SKIP LOCKED",
		"GET DIAGNOSTICS changed_rows = ROW_COUNT",
		"COMMIT;",
		"audit actor backfill complete table=% processed=% user_classified=% unknown_classified=% failed=% remaining=%",
		"BETWEEN 1 AND 9223372036854775807",
		"'user', 'project', 'organization', 'service', 'system'",
		"created_actor_type = 'unknown' AND created_actor_id IS NULL",
	} {
		if !strings.Contains(conversion, required) {
			t.Errorf("conversion migration missing %q", required)
		}
	}
	if strings.TrimSpace(execution) != "CALL public.backfill_integration_audit_actor_identity();" {
		t.Fatalf("backfill migration must contain one top-level CALL, got %q", execution)
	}
	for _, required := range []string{"CREATE OR REPLACE FUNCTION public.reject_created_actor_change()", "RAISE EXCEPTION 'creation actor is immutable'"} {
		if !strings.Contains(finalization, required) {
			t.Errorf("finalization migration missing %q", required)
		}
	}

	for _, table := range tables {
		if !strings.Contains(schema, fmt.Sprintf("ALTER TABLE public.%s\n    ADD CONSTRAINT audit_created_actor_pair", table)) {
			t.Errorf("conversion migration missing actor constraints for %s", table)
		}
		if !strings.Contains(schema, fmt.Sprintf("BEFORE UPDATE ON public.%s", table)) {
			t.Errorf("conversion migration missing creation actor trigger for %s", table)
		}
	}

	if strings.Count(finalization, "CREATE TRIGGER audit_created_actor_immutable") != len(tables) {
		t.Fatalf("finalization migration has %d triggers, want %d", strings.Count(finalization, "CREATE TRIGGER audit_created_actor_immutable"), len(tables))
	}
	if strings.Contains(cleanup, "DROP COLUMN") {
		t.Fatal("integration cleanup must remain a no-op because legacy audit columns never existed")
	}
}

func TestAuditActorMigrationsAreExplicitlyIrreversible(t *testing.T) {
	for _, path := range []string{
		"000004_run_audit_actor_backfill.down.sql",
		"000005_finalize_audit_actor_identity.down.sql",
		"000006_remove_legacy_audit_identity.down.sql",
	} {
		migration := readMigration(t, path)
		if !strings.Contains(migration, "RAISE EXCEPTION") || !strings.Contains(migration, "backup set") {
			t.Errorf("%s must direct operators to full backup restoration", path)
		}
	}
}
