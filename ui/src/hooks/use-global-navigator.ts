import { useNavigate } from 'react-router-dom';

export const useGlobalNavigation = () => {
  const navigate = useNavigate();

  const goBack = () => navigate(-1);

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

  const goToConfigureWeb = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/deployment/web`);

  const goToConfigureApp = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/deployment/app`);

  const goToConfigureWhatsapp = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/deployment/whatsapp`);

  const goToConfigureCall = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/deployment/call`);

  const goToConfigureApi = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/deployment/api`);

  const goToConfigureDebugger = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/deployment/debugger`);

  const goToConfigureSlack = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/deployment/slack`);

  const goToManageAssistant = (assistantId: string) => {
    navigate(`/deployment/assistant/${assistantId}/manage`);
  };
  const goToDeploymentAssistant = (assistantId: string) => {
    navigate(`/deployment/assistant/${assistantId}/manage/deployment`);
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
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-webhook/create`,
    );

  const goToConfigureAssistantAnalysis = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/configure-analysis`);
  const goToCreateAssistantAnalysis = (assistantId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-analysis/create`,
    );
  const goToEditAssistantAnalysis = (assistantId: string, analysisId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-analysis/${analysisId}`,
    );

  const goToEditAssistantWebhook = (assistantId: string, webhookId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-webhook/${webhookId}`,
    );

  const goToAssistantWebhook = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/configure-webhook`);

  const goToAssistantSession = (assistantId: string, sessionId: string) =>
    navigate(`/deployment/assistant/${assistantId}/sessions/${sessionId}`);

  const goToAssistantSessionList = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/sessions`);

  const goToCreateKnowledge = () => navigate('/knowledge/create-knowledge');
  const goToAssistantListing = () => navigate('/deployment/assistant/');

  const goToConfigureAssistantKnowledge = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/configure-knowledge`);

  const goToEditAssistantKnowledge = (
    assistantId: string,
    knowledgeId: string,
  ) =>
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-knowledge/${knowledgeId}`,
    );

  const goToCreateAssistantKnowledge = (assistantId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-knowledge/create`,
    );

  const goToConfigureAssistantTool = (assistantId: string) =>
    navigate(`/deployment/assistant/${assistantId}/manage/configure-tool`);

  const goToEditAssistantTool = (assistantId: string, toolId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-tool/${toolId}`,
    );

  const goToCreateAssistantTool = (assistantId: string) =>
    navigate(
      `/deployment/assistant/${assistantId}/manage/configure-tool/create`,
    );

  return {
    goBack,
    goToAssistant,
    goToDashboard,
    goToCreateAssistant,
    goToCreateKnowledge,
    goToCreateAssistantVersion,
    goToCreateAssistantWebsocketVersion,
    goToCreateAssistantAgentKitVersion,

    goToDeploymentAssistant,
    goToConfigureWeb,
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
    goToConfigureApi,
    goToConfigureDebugger,
    goToEditAssistant,
    goToEndpoint,
    goToManageAssistant,

    //
    goToCreateAssistantWebhook,
    goToAssistantWebhook,
    goToEditAssistantWebhook,

    //
    goToAssistantSession,
    goToAssistantSessionList,
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
  };
};
