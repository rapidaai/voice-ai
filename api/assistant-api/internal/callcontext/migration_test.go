package internal_callcontext

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAuthenticationSnapshotMigrationContract(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve migration test path")
	}
	migrationDir := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	up, err := os.ReadFile(filepath.Join(migrationDir, "000056_add_call_context_authentication_snapshot.up.sql"))
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile(filepath.Join(migrationDir, "000056_add_call_context_authentication_snapshot.down.sql"))
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	for _, column := range []string{"auth_user_id bigint", "auth_actor_type varchar(50)", "auth_actor_id bigint"} {
		if !strings.Contains(string(up), column) {
			t.Errorf("up migration missing %q", column)
		}
	}
	if strings.Contains(string(up), "auth_token") || strings.Contains(string(down), "auth_token") {
		t.Fatal("migration must retain and not alter auth_token")
	}
	for _, column := range []string{"auth_user_id", "auth_actor_type", "auth_actor_id"} {
		if !strings.Contains(string(down), "DROP COLUMN "+column) {
			t.Errorf("down migration missing drop for %q", column)
		}
	}
}
