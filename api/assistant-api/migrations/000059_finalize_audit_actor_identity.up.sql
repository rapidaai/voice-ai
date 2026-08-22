ALTER TABLE public.assistant_api_deployments VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_api_deployments VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_configuration_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_configuration_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_configurations VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_configurations VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_action_metrics VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_action_metrics VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_arguments VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_arguments VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_message_metadata VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_message_metadata VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_message_metrics VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_message_metrics VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_messages VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_messages VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_metadata VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_metadata VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_metrics VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_metrics VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversation_recordings VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversation_recordings VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_conversations VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_conversations VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_debugger_deployments VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_debugger_deployments VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_deployment_audio_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_deployment_audio_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_deployment_audios VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_deployment_audios VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_deployment_telephony_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_deployment_telephony_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_deployment_whatsapp_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_deployment_whatsapp_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_knowledge_logs VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_knowledge_logs VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_knowledge_reranker_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_knowledge_reranker_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_knowledges VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_knowledges VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_phone_deployments VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_phone_deployments VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_provider_agentflows VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_provider_agentflows VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_provider_agentkits VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_provider_agentkits VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_provider_model_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_provider_model_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_provider_models VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_provider_models VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_provider_websockets VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_provider_websockets VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_tags VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_tags VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_tool_logs VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_tool_logs VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_tool_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_tool_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_tools VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_tools VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_web_plugin_deployments VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_web_plugin_deployments VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_http_logs VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_http_logs VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistant_whatsapp_deployments VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistant_whatsapp_deployments VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.assistants VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.assistants VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.knowledge_document_process_rules VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.knowledge_document_process_rules VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.knowledge_documents VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.knowledge_documents VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.knowledge_embedding_model_options VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.knowledge_embedding_model_options VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.knowledge_logs VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.knowledge_logs VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.knowledge_tags VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.knowledge_tags VALIDATE CONSTRAINT audit_updated_actor_pair;
ALTER TABLE public.knowledges VALIDATE CONSTRAINT audit_created_actor_pair;
ALTER TABLE public.knowledges VALIDATE CONSTRAINT audit_updated_actor_pair;
CREATE OR REPLACE FUNCTION public.reject_created_actor_change()
RETURNS trigger
LANGUAGE plpgsql
AS $function$
BEGIN
    IF NEW.created_actor_type IS DISTINCT FROM OLD.created_actor_type
       OR NEW.created_actor_id IS DISTINCT FROM OLD.created_actor_id THEN
        RAISE EXCEPTION 'creation actor is immutable';
    END IF;
    RETURN NEW;
END
$function$;

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_api_deployments
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_configuration_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_configurations
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_action_metrics
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_arguments
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_message_metadata
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_message_metrics
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_messages
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_metadata
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_metrics
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversation_recordings
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_conversations
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_debugger_deployments
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_deployment_audio_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_deployment_audios
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_deployment_telephony_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_deployment_whatsapp_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_knowledge_logs
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_knowledge_reranker_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_knowledges
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_phone_deployments
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_provider_agentflows
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_provider_agentkits
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_provider_model_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_provider_models
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_provider_websockets
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_tags
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_tool_logs
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_tool_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_tools
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_web_plugin_deployments
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_http_logs
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistant_whatsapp_deployments
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.assistants
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.knowledge_document_process_rules
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.knowledge_documents
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.knowledge_embedding_model_options
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.knowledge_logs
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.knowledge_tags
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();

CREATE TRIGGER audit_created_actor_immutable
BEFORE UPDATE ON public.knowledges
FOR EACH ROW EXECUTE FUNCTION public.reject_created_actor_change();


DROP PROCEDURE public.backfill_assistant_audit_actor_identity();
