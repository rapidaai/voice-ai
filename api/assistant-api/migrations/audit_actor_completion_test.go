package migrations

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompleteAuditActorMigration(t *testing.T) {
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
	conversion := readMigration(t, "000057_convert_audit_actor_identity.up.sql")
	execution := readMigration(t, "000058_run_audit_actor_backfill.up.sql")
	finalization := readMigration(t, "000059_finalize_audit_actor_identity.up.sql")
	cleanup := readMigration(t, "000060_remove_legacy_audit_identity.up.sql")
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
	if strings.TrimSpace(execution) != "CALL public.backfill_assistant_audit_actor_identity();" {
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
		if !strings.Contains(cleanup, fmt.Sprintf("ALTER TABLE public.%s\n    DROP COLUMN created_by,", table)) {
			t.Errorf("cleanup migration missing legacy audit removal for %s", table)
		}
	}

	if strings.Count(finalization, "CREATE TRIGGER audit_created_actor_immutable") != len(tables) {
		t.Fatalf("finalization migration has %d triggers, want %d", strings.Count(finalization, "CREATE TRIGGER audit_created_actor_immutable"), len(tables))
	}
	if strings.Count(cleanup, "DROP COLUMN created_by") != len(tables) {
		t.Fatalf("cleanup migration removes created_by from %d tables, want %d", strings.Count(cleanup, "DROP COLUMN created_by"), len(tables))
	}
}

func TestAuditActorMigrationsAreExplicitlyIrreversible(t *testing.T) {
	for _, path := range []string{
		"000058_run_audit_actor_backfill.down.sql",
		"000059_finalize_audit_actor_identity.down.sql",
		"000060_remove_legacy_audit_identity.down.sql",
	} {
		migration := readMigration(t, path)
		if !strings.Contains(migration, "RAISE EXCEPTION") || !strings.Contains(migration, "backup set") {
			t.Errorf("%s must direct operators to full backup restoration", path)
		}
	}
}
