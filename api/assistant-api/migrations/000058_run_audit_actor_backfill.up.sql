SET lock_timeout = '5s';

DO $migration$
BEGIN
    IF EXISTS (SELECT 1 FROM public.assistant_api_deployments WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_configuration_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_configurations WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_action_metrics WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_arguments WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_message_metadata WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_message_metrics WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_messages WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_metadata WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_metrics WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversation_recordings WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_conversations WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_debugger_deployments WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_deployment_audio_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_deployment_audios WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_deployment_telephony_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_deployment_whatsapp_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_knowledge_logs WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_knowledge_reranker_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_knowledges WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_phone_deployments WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_provider_model_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_tags WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_tool_logs WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_tool_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_tools WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_web_plugin_deployments WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_http_logs WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_whatsapp_deployments WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistants WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.knowledge_document_process_rules WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.knowledge_documents WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.knowledge_embedding_model_options WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.knowledge_logs WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.knowledge_tags WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.knowledges WHERE created_by IS NULL OR created_by <= 0 OR (updated_by IS NOT NULL AND updated_by <= 0))
        OR EXISTS (SELECT 1 FROM public.assistant_provider_agentflows WHERE created_by IS NULL OR created_by <= 0)
        OR EXISTS (SELECT 1 FROM public.assistant_provider_agentkits WHERE created_by IS NULL OR created_by <= 0)
        OR EXISTS (SELECT 1 FROM public.assistant_provider_models WHERE created_by IS NULL OR created_by <= 0)
        OR EXISTS (SELECT 1 FROM public.assistant_provider_websockets WHERE created_by IS NULL OR created_by <= 0)
    THEN
        RAISE EXCEPTION 'assistant-api audit actor backfill requires positive user IDs';
    END IF;
END
$migration$;

UPDATE public.assistant_api_deployments
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_configuration_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_configurations
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_action_metrics
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_arguments
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_message_metadata
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_message_metrics
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_messages
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_metadata
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_metrics
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversation_recordings
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_conversations
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_debugger_deployments
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_deployment_audio_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_deployment_audios
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_deployment_telephony_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_deployment_whatsapp_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_knowledge_logs
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_knowledge_reranker_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_knowledges
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_phone_deployments
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_provider_model_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_tags
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_tool_logs
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_tool_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_tools
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_web_plugin_deployments
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_http_logs
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_whatsapp_deployments
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistants
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.knowledge_document_process_rules
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.knowledge_documents
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.knowledge_embedding_model_options
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.knowledge_logs
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.knowledge_tags
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.knowledges
SET created_actor_type = 'user',
    created_actor_id = created_by,
    updated_actor_type = CASE WHEN updated_by IS NULL THEN NULL ELSE 'user' END,
    updated_actor_id = updated_by;

UPDATE public.assistant_provider_agentflows
SET created_actor_type = 'user',
    created_actor_id = created_by;

UPDATE public.assistant_provider_agentkits
SET created_actor_type = 'user',
    created_actor_id = created_by;

UPDATE public.assistant_provider_models
SET created_actor_type = 'user',
    created_actor_id = created_by;

UPDATE public.assistant_provider_websockets
SET created_actor_type = 'user',
    created_actor_id = created_by;

RESET lock_timeout;
