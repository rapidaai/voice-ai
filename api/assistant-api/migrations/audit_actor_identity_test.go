package migrations

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestExpandAuditActorIdentityMigration(t *testing.T) {
	tables := []string{
		"assistant_api_deployments",
		"assistant_configuration_options",
		"assistant_configurations",
		"assistant_conversation_action_metrics",
		"assistant_conversation_arguments",
		"assistant_conversation_message_metadata",
		"assistant_conversation_message_metrics",
		"assistant_conversation_messages",
		"assistant_conversation_metadata",
		"assistant_conversation_metrics",
		"assistant_conversation_options",
		"assistant_conversation_recordings",
		"assistant_conversations",
		"assistant_debugger_deployments",
		"assistant_deployment_audio_options",
		"assistant_deployment_audios",
		"assistant_deployment_telephony_options",
		"assistant_deployment_whatsapp_options",
		"assistant_knowledge_logs",
		"assistant_knowledge_reranker_options",
		"assistant_knowledges",
		"assistant_phone_deployments",
		"assistant_provider_agentflows",
		"assistant_provider_agentkits",
		"assistant_provider_model_options",
		"assistant_provider_models",
		"assistant_provider_websockets",
		"assistant_tags",
		"assistant_tool_logs",
		"assistant_tool_options",
		"assistant_tools",
		"assistant_web_plugin_deployments",
		"assistant_http_logs",
		"assistant_whatsapp_deployments",
		"assistants",
		"knowledge_document_process_rules",
		"knowledge_documents",
		"knowledge_embedding_model_options",
		"knowledge_logs",
		"knowledge_tags",
		"knowledges",
	}

	validateActorExpansionMigration(t, "000055_expand_audit_actor_identity.up.sql", "000055_expand_audit_actor_identity.down.sql", tables)
}

func validateActorExpansionMigration(t *testing.T, upPath, downPath string, tables []string) {
	t.Helper()

	up := readMigration(t, upPath)
	down := readMigration(t, downPath)
	for _, migration := range []struct {
		name string
		sql  string
	}{{"up", up}, {"down", down}} {
		if strings.Count(migration.sql, "SET lock_timeout = '5s';") != 1 {
			t.Errorf("%s migration must set a five-second lock timeout exactly once", migration.name)
		}
		if strings.Count(migration.sql, "RESET lock_timeout;") != 1 {
			t.Errorf("%s migration must reset lock_timeout exactly once", migration.name)
		}
	}

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
	for _, forbidden := range []string{
		"created_actor_id text",
		"updated_actor_id text",
		"drop column",
		"alter column created_by",
		"alter column updated_by",
		"update public.",
		"insert into",
		" default ",
		" using ",
	} {
		if strings.Contains(lowerUp, forbidden) {
			t.Errorf("up migration contains forbidden expansion operation %q", forbidden)
		}
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
