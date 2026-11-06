import { SkeletonText } from '@carbon/react';
import { Navigate, type RouteObject } from 'react-router-dom';
import {
  DeploymentEndpointPage,
  DeploymentCreateEndpointPage,
  DeploymentViewEndpointPage,
  DeploymentCreateVersionEndpointPage,
  DeploymentConfigureEndpointPage,
} from '@/app/pages/endpoint';
import {
  DeploymentCreateAssistantPage,
  DeploymentAssistantPage,
  DeploymentViewAssistantPage,
  DeploymentCreateVersionAssistantPage,
  DeploymentConfigureAssistantWebDeploymentPage,
  DeploymentEditAssistantWebDeploymentPage,
  DeploymentConfigureAssistantCallDeploymentPage,
  DeploymentEditAssistantCallDeploymentPage,
  DeploymentConfigureAssistantToolPage,
  DeploymentConfigureAssistantAnalysisPage,
  DeploymentConfigureAssistantApiDeploymentPage,
  DeploymentEditAssistantApiDeploymentPage,
  DeploymentConfigureAssistantDebuggerDeploymentPage,
  DeploymentEditAssistantDebuggerDeploymentPage,
  DeploymentEditAssistantPage,
  DeploymentConfigureAssistantWebhookPage,
  DeploymentCreateAssistantWebhookPage,
  DeploymentUpdateAssistantWebhookPage,
  DeploymentConfigureAssistantTelemetryPage,
  DeploymentCreateAssistantTelemetryPage,
  DeploymentUpdateAssistantTelemetryPage,
  DeploymentConfigureAssistantStoragePage,
  DeploymentCreateAssistantStoragePage,
  DeploymentUpdateAssistantStoragePage,
  DeploymentConfigureAssistantAuthenticationPage,
  DeploymentCreateAssistantAuthenticationPage,
  DeploymentUpdateAssistantAuthenticationPage,
  DeploymentConversationDetailPage,
  DeploymentCreateAssistantAnalysisPage,
  DeploymentUpdateAssistantAnalysisPage,
  DeploymentUpdateAssistantToolPage,
  DeploymentCreateAssistantToolPage,
  DeploymentConfigureAssistantDeploymentPage,
  DeploymentCreateWebsocketVersionAssistantPage,
  DeploymentCreateAgentKitVersionAssistantPage,
  DeploymentCreateAgentflowVersionAssistantPage,
  DeploymentCreateAgentKitPage,
  DeploymentCreateWebsocketPage,
  DeploymentCreateAgentflowPage,
} from '@/app/pages/assistant';
import { AssistantViewLayout } from '@/app/pages/assistant/view/assistant-view.layout';
import { EndpointViewLayout } from '@/app/pages/endpoint/view/endpoint-view.layout';
import { StaticPageNotFoundPage } from '@/app/pages/static-pages';
import {
  PreviewPhoneAgentPage,
  PreviewVoiceAgentV2Page,
} from '@/app/pages/preview-agent';
import { useAssistantPageStore } from '@/stores/assistant/assistant.store';
import { useEndpointPageStore } from '@/stores/endpoint';
import { breadcrumb, type BreadcrumbLabelProps } from './breadcrumb-handle';
import { WorkspaceRouteLayout } from './workspace-route-layout';

function AssistantBreadcrumbLabel({ params }: BreadcrumbLabelProps) {
  const assistant = useAssistantPageStore(state => state.currentAssistant);
  if (!assistant || assistant.getId() !== params.assistantId) {
    return <SkeletonText className="mb-0!" width="8rem" />;
  }

  return assistant.getName();
}

function EndpointBreadcrumbLabel({ params }: BreadcrumbLabelProps) {
  const endpoint = useEndpointPageStore(state => state.currentEndpoint);
  if (!endpoint || endpoint.getId() !== params.endpointId) {
    return <SkeletonText className="mb-0!" width="8rem" />;
  }

  return endpoint.getName();
}

export const deploymentRoute: RouteObject = {
  path: '/deployment',
  element: <WorkspaceRouteLayout />,
  handle: breadcrumb('Deployment'),
  children: [
    { index: true, element: <Navigate to="assistant" replace /> },
    {
      path: 'endpoint',
      handle: breadcrumb('Endpoints'),
      children: [
        { index: true, element: <DeploymentEndpointPage /> },
        {
          path: 'create-endpoint',
          element: <DeploymentCreateEndpointPage />,
          handle: breadcrumb('Create endpoint'),
        },
        {
          path: 'configure-endpoint',
          handle: breadcrumb('Configure endpoint'),
          children: [
            { index: true, element: <DeploymentCreateEndpointPage /> },
            {
              path: ':endpointId',
              element: <DeploymentConfigureEndpointPage />,
              handle: breadcrumb('Edit endpoint'),
            },
          ],
        },
        {
          path: ':endpointId',
          element: <EndpointViewLayout />,
          handle: { breadcrumb: { render: EndpointBreadcrumbLabel } },
          children: [
            { index: true, element: <Navigate to="overview" replace /> },
            {
              path: 'overview',
              element: <DeploymentViewEndpointPage tab="overview" />,
              handle: breadcrumb('Overview'),
            },
            {
              path: 'logs',
              element: <DeploymentViewEndpointPage tab="logs" />,
              handle: breadcrumb('Logs'),
            },
            {
              path: 'versions',
              element: <DeploymentViewEndpointPage tab="versions" />,
              handle: breadcrumb('Versions'),
            },
            {
              path: 'settings',
              element: <DeploymentViewEndpointPage tab="settings" />,
              handle: breadcrumb('Settings'),
            },
            {
              path: 'playground',
              element: <DeploymentViewEndpointPage tab="playground" />,
              handle: breadcrumb('Playground'),
            },
            {
              path: 'create-endpoint-version',
              element: <DeploymentCreateVersionEndpointPage />,
              handle: breadcrumb('Create version'),
            },
          ],
        },
      ],
    },
    {
      path: 'assistant',
      handle: breadcrumb('Assistants'),
      children: [
        { index: true, element: <DeploymentAssistantPage /> },
        {
          path: 'create-assistant',
          element: <DeploymentCreateAssistantPage />,
          handle: breadcrumb('Create assistant'),
        },
        {
          path: 'connect-websocket',
          element: <DeploymentCreateWebsocketPage />,
          handle: breadcrumb('Connect WebSocket'),
        },
        {
          path: 'connect-agentkit',
          element: <DeploymentCreateAgentKitPage />,
          handle: breadcrumb('Connect AgentKit'),
        },
        {
          path: 'create-agentflow',
          element: <DeploymentCreateAgentflowPage />,
          handle: breadcrumb('Create agentflow'),
        },
        {
          path: ':assistantId',
          element: <AssistantViewLayout />,
          handle: { breadcrumb: { render: AssistantBreadcrumbLabel } },
          children: [
            { index: true, element: <Navigate to="overview" replace /> },
            {
              path: 'overview',
              element: <DeploymentViewAssistantPage tab="overview" />,
              handle: breadcrumb('Overview'),
            },
            {
              path: 'version-history',
              element: <DeploymentViewAssistantPage tab="version-history" />,
              handle: breadcrumb('Version history'),
            },
            {
              path: 'sessions',
              handle: breadcrumb('Sessions'),
              children: [
                {
                  index: true,
                  element: <DeploymentViewAssistantPage tab="sessions" />,
                },
                {
                  path: ':sessionId',
                  element: <DeploymentConversationDetailPage />,
                  handle: breadcrumb('Session details'),
                },
              ],
            },
            {
              path: 'playground',
              handle: breadcrumb('Playground'),
              children: [
                {
                  path: 'chat',
                  element: <PreviewVoiceAgentV2Page embedded />,
                  handle: breadcrumb('Chat'),
                },
                {
                  path: 'phone',
                  element: <PreviewPhoneAgentPage embedded />,
                  handle: breadcrumb('Phone'),
                },
              ],
            },
            {
              path: 'create-new-version',
              element: <DeploymentCreateVersionAssistantPage />,
              handle: breadcrumb('Create version'),
            },
            {
              path: 'create-websocket-version',
              element: <DeploymentCreateWebsocketVersionAssistantPage />,
              handle: breadcrumb('Create WebSocket version'),
            },
            {
              path: 'create-agentkit-version',
              element: <DeploymentCreateAgentKitVersionAssistantPage />,
              handle: breadcrumb('Create AgentKit version'),
            },
            {
              path: 'create-agentflow-version',
              element: <DeploymentCreateAgentflowVersionAssistantPage />,
              handle: breadcrumb('Create agentflow version'),
            },
            {
              path: 'edit-assistant',
              element: <DeploymentEditAssistantPage />,
              handle: breadcrumb('Edit assistant'),
            },
            {
              path: 'edit-agentflow',
              element: <DeploymentCreateAgentflowVersionAssistantPage />,
              handle: breadcrumb('Edit agentflow'),
            },
            {
              path: 'configure-analysis',
              handle: breadcrumb('Analysis'),
              children: [
                {
                  index: true,
                  element: <DeploymentConfigureAssistantAnalysisPage />,
                },
                {
                  path: 'create',
                  element: <DeploymentCreateAssistantAnalysisPage />,
                  handle: breadcrumb('Create'),
                },
                {
                  path: ':analysisId',
                  element: <DeploymentUpdateAssistantAnalysisPage />,
                  handle: breadcrumb('Edit'),
                },
              ],
            },
            {
              path: 'configure-tool',
              handle: breadcrumb('Tools'),
              children: [
                {
                  index: true,
                  element: <DeploymentConfigureAssistantToolPage />,
                },
                {
                  path: 'create',
                  element: <DeploymentCreateAssistantToolPage />,
                  handle: breadcrumb('Create'),
                },
                {
                  path: ':assistantToolId',
                  element: <DeploymentUpdateAssistantToolPage />,
                  handle: breadcrumb('Edit'),
                },
              ],
            },
            {
              path: 'configure-webhook',
              handle: breadcrumb('Webhooks'),
              children: [
                {
                  index: true,
                  element: <DeploymentConfigureAssistantWebhookPage />,
                },
                {
                  path: 'create',
                  element: <DeploymentCreateAssistantWebhookPage />,
                  handle: breadcrumb('Create'),
                },
                {
                  path: ':webhookId',
                  element: <DeploymentUpdateAssistantWebhookPage />,
                  handle: breadcrumb('Edit'),
                },
              ],
            },
            {
              path: 'configure-authentication',
              handle: breadcrumb('Authentication'),
              children: [
                {
                  index: true,
                  element: <DeploymentConfigureAssistantAuthenticationPage />,
                },
                {
                  path: 'create',
                  element: <DeploymentCreateAssistantAuthenticationPage />,
                  handle: breadcrumb('Create'),
                },
                {
                  path: 'edit',
                  element: <DeploymentUpdateAssistantAuthenticationPage />,
                  handle: breadcrumb('Edit'),
                },
              ],
            },
            {
              path: 'configure-telemetry',
              handle: breadcrumb('Telemetry'),
              children: [
                {
                  index: true,
                  element: <DeploymentConfigureAssistantTelemetryPage />,
                },
                {
                  path: 'create',
                  element: <DeploymentCreateAssistantTelemetryPage />,
                  handle: breadcrumb('Create'),
                },
                {
                  path: ':telemetryId',
                  element: <DeploymentUpdateAssistantTelemetryPage />,
                  handle: breadcrumb('Edit'),
                },
              ],
            },
            {
              path: 'configure-storage',
              handle: breadcrumb('Storage'),
              children: [
                {
                  index: true,
                  element: <DeploymentConfigureAssistantStoragePage />,
                },
                {
                  path: 'create',
                  element: <DeploymentCreateAssistantStoragePage />,
                  handle: breadcrumb('Create'),
                },
                {
                  path: ':storageId',
                  element: <DeploymentUpdateAssistantStoragePage />,
                  handle: breadcrumb('Edit'),
                },
              ],
            },
            {
              path: 'deployment',
              handle: breadcrumb('Deployments'),
              children: [
                {
                  index: true,
                  element: <DeploymentConfigureAssistantDeploymentPage />,
                },
                {
                  path: 'web',
                  handle: breadcrumb('Web'),
                  children: [
                    {
                      index: true,
                      element: (
                        <DeploymentConfigureAssistantWebDeploymentPage />
                      ),
                    },
                    {
                      path: ':deploymentId',
                      element: <DeploymentEditAssistantWebDeploymentPage />,
                      handle: breadcrumb('Edit'),
                    },
                  ],
                },
                {
                  path: 'call',
                  handle: breadcrumb('Call'),
                  children: [
                    {
                      index: true,
                      element: (
                        <DeploymentConfigureAssistantCallDeploymentPage />
                      ),
                    },
                    {
                      path: ':deploymentId',
                      element: <DeploymentEditAssistantCallDeploymentPage />,
                      handle: breadcrumb('Edit'),
                    },
                  ],
                },
                {
                  path: 'api',
                  handle: breadcrumb('API'),
                  children: [
                    {
                      index: true,
                      element: (
                        <DeploymentConfigureAssistantApiDeploymentPage />
                      ),
                    },
                    {
                      path: ':deploymentId',
                      element: <DeploymentEditAssistantApiDeploymentPage />,
                      handle: breadcrumb('Edit'),
                    },
                  ],
                },
                {
                  path: 'debugger',
                  handle: breadcrumb('Debugger'),
                  children: [
                    {
                      index: true,
                      element: (
                        <DeploymentConfigureAssistantDebuggerDeploymentPage />
                      ),
                    },
                    {
                      path: ':deploymentId',
                      element: (
                        <DeploymentEditAssistantDebuggerDeploymentPage />
                      ),
                      handle: breadcrumb('Edit'),
                    },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
    {
      path: '*',
      element: <StaticPageNotFoundPage />,
      handle: breadcrumb('Not found'),
    },
  ],
};
