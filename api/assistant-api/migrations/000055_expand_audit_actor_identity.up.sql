SET lock_timeout = '5s';

ALTER TABLE public.assistant_api_deployments
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_configuration_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_configurations
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_action_metrics
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_arguments
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_message_metadata
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_message_metrics
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_messages
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_metadata
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_metrics
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversation_recordings
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_conversations
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_debugger_deployments
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_deployment_audio_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_deployment_audios
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_deployment_telephony_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_deployment_whatsapp_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_knowledge_logs
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_knowledge_reranker_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_knowledges
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_phone_deployments
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_provider_agentflows
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_provider_agentkits
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_provider_model_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_provider_models
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_provider_websockets
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_tags
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_tool_logs
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_tool_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_tools
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_web_plugin_deployments
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_http_logs
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistant_whatsapp_deployments
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.assistants
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.knowledge_document_process_rules
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.knowledge_documents
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.knowledge_embedding_model_options
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.knowledge_logs
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.knowledge_tags
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

ALTER TABLE public.knowledges
    ADD COLUMN created_actor_type varchar(32),
    ADD COLUMN created_actor_id text,
    ADD COLUMN updated_actor_type varchar(32),
    ADD COLUMN updated_actor_id text;

RESET lock_timeout;
