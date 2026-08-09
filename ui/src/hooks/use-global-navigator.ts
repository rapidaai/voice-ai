import { useNavigate } from 'react-router-dom';

export const useGlobalNavigation = () => {
  const navigate = useNavigate();

  const goBack = () => navigate(-1);
  const goTo = (url: string) => navigate(url);

  const goToDashboard = () => {
    navigate(`/dashboard`);
  };
  const goToAssistant = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}`);

  const goToAssistantVersions = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/version-history`);

  const goToEditAssistant = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/edit-assistant`);

  const goToCreateAssistant = () =>
    navigate(`/deployment/assistant/create-assistant`);

  const goToCreateAssistantVersion = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/create-new-version`);

  const goToCreateAssistantWebsocketVersion = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/create-websocket-version`);

  const goToCreateAssistantAgentKitVersion = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/create-agentkit-version`);

  const goToCreateAssistantAgentflowVersion = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/create-agentflow-version`);

  const goToConfigureWeb = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/deployment/web`);
  const goToEditWeb = (assistantId: string, deploymentId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/deployment/web/${deploymentId}`,
    );

  const goToConfigureApp = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/deployment/app`);

  const goToConfigureWhatsapp = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/deployment/whatsapp`);

  const goToConfigureCall = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/deployment/call`);

  const goToEditCall = (assistantId: string, deploymentId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/deployment/call/${deploymentId}`,
    );

  const goToConfigureApi = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/deployment/api`);
  const goToEditApi = (assistantId: string, deploymentId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/deployment/api/${deploymentId}`,
    );

  const goToConfigureDebugger = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/deployment/debugger`);
  const goToEditDebugger = (assistantId: string, deploymentId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/deployment/debugger/${deploymentId}`,
    );
  const goToConfigureDebuggerExperience = (assistantId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/deployment/debugger?editMode=section&section=experience`,
    );
  const goToConfigureDebuggerSTT = (assistantId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/deployment/debugger?editMode=section&section=stt`,
    );
  const goToConfigureDebuggerTTS = (assistantId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/deployment/debugger?editMode=section&section=tts`,
    );

  const goToConfigureSlack = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/deployment/slack`);

  const goToManageAssistant = (assistantId: string) => {
    navigate(`/deployment/assistant/${assistantId}/deployment`);
  };

  const goToDeploymentAssistant = (assistantId: string) => {
    navigate(`/deployment/assistant/${assistantId}/deployment`);
  };

  const goToAssistantPreview = (assistantId: string) =>
    window.open(`/preview/chat/${assistantId}`);

  const goToAssistantPreviewCall = (assistantId: string) =>
    window.open(`/preview/call/${assistantId}`);

  const goToKnowledge = (knowledgeId: string) => {
    navigate(`/knowledge/${knowledgeId}`);
  };
  const goToKnowledgeAddManualFile = (knowledgeId: string) => {
    navigate(`/knowledge/${knowledgeId}/add-knowledge-file`);
  };
  const goToKnowledgeAddCloudFile = (knowledgeId: string) => {
    navigate(`/knowledge/${knowledgeId}/add-cloud-file`);
  };

  const goToKnowledgeAddStructureFile = (knowledgeId: string) => {
    navigate(`/knowledge/${knowledgeId}/add-structure-file`);
  };

  const goToEndpoint = (endpointId: string) => {
    navigate(`/deployment/endpoint/${endpointId}`);
  };

  const goToCreateAssistantWebhook = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-webhook/create`);

  const goToConfigureAssistantAnalysis = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-analysis`);
  const goToCreateAssistantAnalysis = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-analysis/create`);
  const goToEditAssistantAnalysis = (assistantId: string, analysisId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/configure-analysis/${analysisId}`,
    );

  const goToEditAssistantWebhook = (assistantId: string, webhookId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/configure-webhook/${webhookId}`,
    );

  const goToAssistantWebhook = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-webhook`);

  const goToAssistantTelemetry = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-telemetry`);

  const goToCreateAssistantTelemetry = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-telemetry/create`);

  const goToEditAssistantTelemetry = (
    assistantId: string,
    telemetryId: string,
  ) =>
    navigate(
      `/deployment/assistant/${assistantId}/configure-telemetry/${telemetryId}`,
    );

  const goToAssistantStorage = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-storage`);

  const goToCreateAssistantStorage = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-storage/create`);

  const goToEditAssistantStorage = (assistantId: string, storageId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/configure-storage/${storageId}`,
    );

  const goToAssistantSession = (assistantId: string, sessionId: string) =>
    navigate(`/deployment/assistant/${assistantId}/sessions/${sessionId}`);

  const goToAssistantSessionList = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/sessions`);

  const goToConversationTelemetry = (conversationId: string) => {
    const params = new URLSearchParams({
      query: `conversation:${conversationId}`,
    });
    navigate(`/logs/traces?${params.toString()}`);
  };

  const goToMessageTelemetry = (messageId: string) => {
    const telemetryMessageId = messageId.startsWith('assistant-')
      ? messageId.slice('assistant-'.length)
      : messageId.startsWith('user-')
        ? messageId.slice('user-'.length)
        : messageId;
    const params = new URLSearchParams({
      query: `message:${telemetryMessageId}`,
    });
    navigate(`/logs/traces?${params.toString()}`);
  };

  const goToCreateKnowledge = () => navigate('/knowledge/create-knowledge');
  const goToAssistantListing = () => navigate('/deployment/assistant/');

  const goToConfigureAssistantKnowledge = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-knowledge`);

  const goToEditAssistantKnowledge = (
    assistantId: string,
    knowledgeId: string,
  ) =>
    navigate(
      `/deployment/assistant/${assistantId}/configure-knowledge/${knowledgeId}`,
    );

  const goToCreateAssistantKnowledge = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-knowledge/create`);

  const goToConfigureAssistantTool = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-tool`);

  const goToEditAssistantTool = (assistantId: string, toolId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-tool/${toolId}`);

  const goToCreateAssistantTool = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/configure-tool/create`);

  const goToModelInformation = (provider: string) => {
    navigate(`/integration/models/${provider}`);
  };

  return {
    goBack,
    goTo,

    goToAssistant,
    goToDashboard,
    goToCreateAssistant,
    goToCreateKnowledge,
    goToCreateAssistantVersion,
    goToCreateAssistantWebsocketVersion,
    goToCreateAssistantAgentKitVersion,
    goToCreateAssistantAgentflowVersion,

    goToDeploymentAssistant,
    goToConfigureWeb,
    goToEditWeb,
    goToConfigureApp,
    goToConfigureSlack,
    goToAssistantPreview,
    goToAssistantPreviewCall,

    goToKnowledge,
    goToConfigureWhatsapp,
    goToKnowledgeAddManualFile,
    goToKnowledgeAddCloudFile,
    goToKnowledgeAddStructureFile,
    goToConfigureCall,
    goToEditCall,
    goToConfigureApi,
    goToEditApi,
    goToConfigureDebugger,
    goToEditDebugger,
    goToConfigureDebuggerExperience,
    goToConfigureDebuggerSTT,
    goToConfigureDebuggerTTS,
    goToEditAssistant,
    goToEndpoint,
    goToManageAssistant,

    //
    goToCreateAssistantWebhook,
    goToAssistantWebhook,
    goToEditAssistantWebhook,
    goToAssistantTelemetry,
    goToCreateAssistantTelemetry,
    goToEditAssistantTelemetry,
    goToAssistantStorage,
    goToCreateAssistantStorage,
    goToEditAssistantStorage,

    //
    goToAssistantSession,
    goToAssistantSessionList,
    goToConversationTelemetry,
    goToMessageTelemetry,
    goToAssistantListing,
    goToAssistantVersions,

    //
    goToCreateAssistantAnalysis,
    goToEditAssistantAnalysis,
    goToConfigureAssistantAnalysis,

    //
    goToCreateAssistantTool,
    goToEditAssistantTool,
    goToConfigureAssistantTool,

    goToCreateAssistantKnowledge,
    goToEditAssistantKnowledge,
    goToConfigureAssistantKnowledge,

    //
    goToModelInformation,
  };
};
