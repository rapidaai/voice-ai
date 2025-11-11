import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const DeploymentAssistantPage = lazyLoad(
  () => import('./listing'),
  module => module.AssistantPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateAssistantPage = lazyLoad(
  () => import('./actions/create-assistant'),
  module => module.CreateAssistantPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateAgentKitPage = lazyLoad(
  () => import('./actions/create-assistant/create-agentkit'),
  module => module.CreateAgentKit,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateWebsocketPage = lazyLoad(
  () => import('./actions/create-assistant/create-websocket'),
  module => module.CreateWebsocket,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentViewAssistantPage = lazyLoad(
  () => import('./view'),
  module => module.ViewAssistantPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantKnowledgePage = lazyLoad(
  () => import('./actions/configure-assistant-knowledge'),
  module => module.ConfigureAssistantKnowledgePage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateAssistantKnowledgePage = lazyLoad(
  () => import('./actions/configure-assistant-knowledge'),
  module => module.CreateAssistantKnowledgePage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentUpdateAssistantKnowledgePage = lazyLoad(
  () => import('./actions/configure-assistant-knowledge'),
  module => module.UpdateAssistantKnowledgePage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantWebhookPage = lazyLoad(
  () => import('./actions/configure-assistant-webhook'),
  module => module.ConfigureAssistantWebhookPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateAssistantWebhookPage = lazyLoad(
  () => import('./actions/configure-assistant-webhook'),
  module => module.CreateAssistantWebhookPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentUpdateAssistantWebhookPage = lazyLoad(
  () => import('./actions/configure-assistant-webhook'),
  module => module.UpdateAssistantWebhookPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantWebDeploymentPage = lazyLoad(
  () => import('./actions/create-deployment/web-plugin'),
  module => module.ConfigureAssistantWebDeploymentPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantCallDeploymentPage = lazyLoad(
  () => import('./actions/create-deployment/phone'),
  module => module.ConfigureAssistantCallDeploymentPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantDeploymentPage = lazyLoad(
  () => import('./actions/create-deployment'),
  module => module.ConfigureAssistantDeploymentPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantApiDeploymentPage = lazyLoad(
  () => import('./actions/create-deployment/api'),
  module => module.ConfigureAssistantApiDeploymentPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantDebuggerDeploymentPage = lazyLoad(
  () => import('./actions/create-deployment/debugger'),
  module => module.ConfigureAssistantDebuggerDeploymentPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateVersionAssistantPage = lazyLoad(
  () => import('./actions/create-assistant-version'),
  module => module.CreateVersionAssistantPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateWebsocketVersionAssistantPage = lazyLoad(
  () => import('./actions/create-assistant-version/create-websocket-version'),
  module => module.CreateWebsocketVersion,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateAgentKitVersionAssistantPage = lazyLoad(
  () => import('./actions/create-assistant-version/create-agent-kit-version'),
  module => module.CreateAgentKitVersion,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantToolPage = lazyLoad(
  () => import('./actions/configure-assistant-tool'),
  module => module.ConfigureAssistantToolPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateAssistantToolPage = lazyLoad(
  () => import('./actions/configure-assistant-tool'),
  module => module.CreateAssistantToolPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentUpdateAssistantToolPage = lazyLoad(
  () => import('./actions/configure-assistant-tool'),
  module => module.UpdateAssistantToolPage,
  {
    fallback: <PageLoader />,
  },
);

// quality
export const DeploymentConfigureAssistantContentFilterPage = lazyLoad(
  () => import('./actions/quality/configure-assistant-content-filter'),
  module => module.ConfigureAssistantContentFilterPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantContextualGroundingPage = lazyLoad(
  () => import('./actions/quality/configure-assistant-contextual-grounding'),
  module => module.ConfigureAssistantContextualGroundingPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureAssistantAnalysisPage = lazyLoad(
  () => import('./actions/configure-assistant-analysis'),
  module => module.ConfigureAssistantAnalysisPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateAssistantAnalysisPage = lazyLoad(
  () => import('./actions/configure-assistant-analysis'),
  module => module.CreateAssistantAnalysisPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentUpdateAssistantAnalysisPage = lazyLoad(
  () => import('./actions/configure-assistant-analysis'),
  module => module.UpdateAssistantAnalysisPage,
  {
    fallback: <PageLoader />,
  },
);

// end of quality

export const DeploymentEditAssistantPage = lazyLoad(
  () => import('./actions/edit-assistant'),
  module => module.EditAssistantPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConversationDetailPage = lazyLoad(
  () => import('./view/conversations/conversation-detail'),
  module => module.ConversationDetailPage,
  {
    fallback: <PageLoader />,
  },
);
