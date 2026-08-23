SET lock_timeout = '5s';

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
