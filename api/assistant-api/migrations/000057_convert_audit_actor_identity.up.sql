SET lock_timeout = '5s';

CREATE TABLE IF NOT EXISTS public.audit_actor_migration_metrics (
    table_name text PRIMARY KEY,
    processed_rows bigint NOT NULL DEFAULT 0,
    user_classified_rows bigint NOT NULL DEFAULT 0,
    unknown_classified_rows bigint NOT NULL DEFAULT 0,
    failed_rows bigint NOT NULL DEFAULT 0,
    remaining_rows bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE PROCEDURE public.backfill_assistant_audit_actor_identity()
LANGUAGE plpgsql
AS $migration$
<<backfill>>
DECLARE
    audited_table text;
    changed_rows integer;
    total_rows bigint;
    user_classified_rows bigint;
    unknown_classified_rows bigint;
    failed_rows bigint;
    remaining_rows bigint;
    has_updated_by boolean;
BEGIN
    PERFORM set_config('lock_timeout', '5s', false);
    FOREACH audited_table IN ARRAY ARRAY[
        'assistant_api_deployments',
        'assistant_configuration_options',
        'assistant_configurations',
        'assistant_conversation_action_metrics',
        'assistant_conversation_arguments',
        'assistant_conversation_message_metadata',
        'assistant_conversation_message_metrics',
        'assistant_conversation_messages',
        'assistant_conversation_metadata',
        'assistant_conversation_metrics',
        'assistant_conversation_options',
        'assistant_conversation_recordings',
        'assistant_conversations',
        'assistant_debugger_deployments',
        'assistant_deployment_audio_options',
        'assistant_deployment_audios',
        'assistant_deployment_telephony_options',
        'assistant_deployment_whatsapp_options',
        'assistant_knowledge_logs',
        'assistant_knowledge_reranker_options',
        'assistant_knowledges',
        'assistant_phone_deployments',
        'assistant_provider_agentflows',
        'assistant_provider_agentkits',
        'assistant_provider_model_options',
        'assistant_provider_models',
        'assistant_provider_websockets',
        'assistant_tags',
        'assistant_tool_logs',
        'assistant_tool_options',
        'assistant_tools',
        'assistant_web_plugin_deployments',
        'assistant_http_logs',
        'assistant_whatsapp_deployments',
        'assistants',
        'knowledge_document_process_rules',
        'knowledge_documents',
        'knowledge_embedding_model_options',
        'knowledge_logs',
        'knowledge_tags',
        'knowledges'
    ]
    LOOP
        total_rows := 0;
        SELECT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = audited_table
              AND column_name = 'updated_by'
        ) INTO has_updated_by;
        LOOP
            IF has_updated_by THEN
                EXECUTE format($sql$
                WITH batch AS (
                    SELECT ctid
                    FROM public.%I
                    WHERE created_actor_type IS NULL
                       OR (updated_by IS NOT NULL AND updated_actor_type IS NULL)
                    LIMIT 10000
                    FOR UPDATE SKIP LOCKED
                )
                UPDATE public.%I AS target
                SET created_actor_type = CASE
                        WHEN target.created_actor_type IS NOT NULL THEN target.created_actor_type
                        WHEN target.created_by > 0 THEN 'user'
                        ELSE 'unknown'
                    END,
                    created_actor_id = CASE
                        WHEN target.created_actor_type IS NOT NULL THEN target.created_actor_id
                        WHEN target.created_by > 0 THEN target.created_by
                        ELSE NULL
                    END,
                    updated_actor_type = CASE
                        WHEN target.updated_actor_type IS NOT NULL THEN target.updated_actor_type
                        WHEN target.updated_by > 0 THEN 'user'
                        WHEN target.updated_by IS NULL THEN NULL
                        ELSE 'unknown'
                    END,
                    updated_actor_id = CASE
                        WHEN target.updated_actor_type IS NOT NULL THEN target.updated_actor_id
                        WHEN target.updated_by > 0 THEN target.updated_by
                        ELSE NULL
                    END
                FROM batch
                WHERE target.ctid = batch.ctid
                $sql$, audited_table, audited_table);
            ELSE
                EXECUTE format($sql$
                    WITH batch AS (
                        SELECT ctid
                        FROM public.%I
                        WHERE created_actor_type IS NULL
                        LIMIT 10000
                        FOR UPDATE SKIP LOCKED
                    )
                    UPDATE public.%I AS target
                    SET created_actor_type = CASE
                            WHEN target.created_actor_type IS NOT NULL THEN target.created_actor_type
                            WHEN target.created_by > 0 THEN 'user'
                            ELSE 'unknown'
                        END,
                        created_actor_id = CASE
                            WHEN target.created_actor_type IS NOT NULL THEN target.created_actor_id
                            WHEN target.created_by > 0 THEN target.created_by
                            ELSE NULL
                        END
                    FROM batch
                    WHERE target.ctid = batch.ctid
                $sql$, audited_table, audited_table);
            END IF;
            GET DIAGNOSTICS changed_rows = ROW_COUNT;
            total_rows := total_rows + changed_rows;
            INSERT INTO public.audit_actor_migration_metrics (table_name, processed_rows, updated_at)
            VALUES (audited_table, changed_rows, CURRENT_TIMESTAMP)
            ON CONFLICT (table_name) DO UPDATE
            SET processed_rows = public.audit_actor_migration_metrics.processed_rows + EXCLUDED.processed_rows,
                updated_at = EXCLUDED.updated_at;
            RAISE LOG 'audit actor backfill table=% batch_rows=% cumulative_rows=%', audited_table, changed_rows, total_rows;
            COMMIT;
            EXIT WHEN changed_rows = 0;
        END LOOP;
        EXECUTE format(
            'SELECT count(*) FILTER (WHERE created_actor_type = ''user''), count(*) FILTER (WHERE created_actor_type = ''unknown''), count(*) FILTER (WHERE NOT ((created_actor_type = ''unknown'' AND created_actor_id IS NULL) OR (created_actor_type IN (''user'', ''project'', ''organization'', ''service'', ''system'') AND created_actor_id BETWEEN 1 AND 9223372036854775807)) OR NOT ((updated_actor_type IS NULL AND updated_actor_id IS NULL) OR (updated_actor_type = ''unknown'' AND updated_actor_id IS NULL) OR (updated_actor_type IN (''user'', ''project'', ''organization'', ''service'', ''system'') AND updated_actor_id BETWEEN 1 AND 9223372036854775807))), count(*) FILTER (WHERE created_actor_type IS NULL) FROM public.%I',
            audited_table
        ) INTO user_classified_rows, unknown_classified_rows, failed_rows, remaining_rows;
        UPDATE public.audit_actor_migration_metrics
        SET user_classified_rows = backfill.user_classified_rows,
            unknown_classified_rows = backfill.unknown_classified_rows,
            failed_rows = backfill.failed_rows,
            remaining_rows = backfill.remaining_rows,
            updated_at = CURRENT_TIMESTAMP
        WHERE table_name = audited_table;
        RAISE LOG 'audit actor backfill complete table=% processed=% user_classified=% unknown_classified=% failed=% remaining=%', audited_table, total_rows, user_classified_rows, unknown_classified_rows, failed_rows, remaining_rows;
    END LOOP;
END
$migration$;

ALTER TABLE public.assistant_api_deployments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_configuration_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_configurations
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_action_metrics
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_arguments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_message_metadata
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_message_metrics
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_messages
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_metadata
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_metrics
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversation_recordings
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_conversations
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_debugger_deployments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_deployment_audio_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_deployment_audios
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_deployment_telephony_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_deployment_whatsapp_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_knowledge_logs
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_knowledge_reranker_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_knowledges
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_phone_deployments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_provider_agentflows
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_provider_agentkits
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_provider_model_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_provider_models
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_provider_websockets
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_tags
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_tool_logs
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_tool_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_tools
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_web_plugin_deployments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_http_logs
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistant_whatsapp_deployments
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.assistants
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.knowledge_document_process_rules
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.knowledge_documents
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.knowledge_embedding_model_options
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.knowledge_logs
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.knowledge_tags
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;


ALTER TABLE public.knowledges
    ADD CONSTRAINT audit_created_actor_pair CHECK (
        (created_actor_type = 'unknown' AND created_actor_id IS NULL)
        OR (created_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND created_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID,
    ADD CONSTRAINT audit_updated_actor_pair CHECK (
        (updated_actor_type IS NULL AND updated_actor_id IS NULL)
        OR (updated_actor_type = 'unknown' AND updated_actor_id IS NULL)
        OR (updated_actor_type IN ('user', 'project', 'organization', 'service', 'system') AND updated_actor_id BETWEEN 1 AND 9223372036854775807)
    ) NOT VALID;
