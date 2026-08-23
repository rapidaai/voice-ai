SET lock_timeout = '5s';

ALTER TABLE public.assistant_api_deployments
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_configuration_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_configurations
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_action_metrics
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_arguments
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_message_metadata
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_message_metrics
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_messages
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_metadata
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_metrics
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversation_recordings
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_conversations
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_debugger_deployments
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_deployment_audio_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_deployment_audios
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_deployment_telephony_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_deployment_whatsapp_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_knowledge_logs
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_knowledge_reranker_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_knowledges
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_phone_deployments
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_provider_agentflows
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_provider_agentkits
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_provider_model_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_provider_models
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_provider_websockets
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_tags
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_tool_logs
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_tool_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_tools
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_web_plugin_deployments
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_http_logs
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistant_whatsapp_deployments
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.assistants
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.knowledge_document_process_rules
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.knowledge_documents
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.knowledge_embedding_model_options
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.knowledge_logs
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.knowledge_tags
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

ALTER TABLE public.knowledges
    DROP COLUMN created_actor_type,
    DROP COLUMN created_actor_id,
    DROP COLUMN updated_actor_type,
    DROP COLUMN updated_actor_id;

RESET lock_timeout;
