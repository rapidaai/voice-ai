UPDATE public.assistant_api_deployments
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_configuration_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_configurations
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_action_metrics
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_arguments
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_message_metadata
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_message_metrics
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_messages
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_metadata
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_metrics
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversation_recordings
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_conversations
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_debugger_deployments
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_deployment_audio_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_deployment_audios
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_deployment_telephony_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_deployment_whatsapp_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_knowledge_logs
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_knowledge_reranker_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_knowledges
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_phone_deployments
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_provider_model_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_tags
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_tool_logs
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_tool_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_tools
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_web_plugin_deployments
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_http_logs
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_whatsapp_deployments
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistants
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.knowledge_document_process_rules
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.knowledge_documents
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.knowledge_embedding_model_options
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.knowledge_logs
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.knowledge_tags
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.knowledges
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END,
    updated_actor_type = CASE WHEN updated_by > 0 THEN 'user' ELSE NULL END,
    updated_actor_id = CASE WHEN updated_by > 0 THEN updated_by ELSE NULL END;

UPDATE public.assistant_provider_agentflows
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END;

UPDATE public.assistant_provider_agentkits
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END;

UPDATE public.assistant_provider_models
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END;

UPDATE public.assistant_provider_websockets
SET created_actor_type = CASE WHEN created_by > 0 THEN 'user' ELSE 'unknown' END,
    created_actor_id = CASE WHEN created_by > 0 THEN created_by ELSE NULL END;
