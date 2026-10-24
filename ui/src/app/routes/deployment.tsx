import { MissionBox } from '@/app/components/layout/container/mission-box';
import { ProtectedBox } from '@/app/components/layout/container/protected-box';
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
import { Navigate, Outlet, Route, Routes } from 'react-router-dom';

export function DeploymentRoute() {
  return (
    <Routes>
      <Route path="*" element={<StaticPageNotFoundPage />} />
      <Route
        key="/deployment/"
        path="/"
        element={
          <ProtectedBox>
            <MissionBox>
              <Outlet />
            </MissionBox>
          </ProtectedBox>
        }
      >
        {/* disvoer  */}
        {/* endpoint  */}
        <Route
          key={'/deployment/endpoint'}
          path={'endpoint'}
          element={<DeploymentEndpointPage />}
        />

        <Route index element={<Navigate to="assistant" replace />} />
        <Route
          path={'endpoint/create-endpoint'}
          element={<DeploymentCreateEndpointPage />}
        />
        {/*  */}
        <Route
          path={'endpoint/configure-endpoint'}
          element={<DeploymentCreateEndpointPage />}
        />
        <Route
          path={'endpoint/configure-endpoint/:endpointId'}
          element={<DeploymentConfigureEndpointPage />}
        />
        <Route path={'endpoint/:endpointId'} element={<EndpointViewLayout />}>
          <Route index element={<Navigate to="overview" replace />} />
          <Route
            path={'create-endpoint-version'}
            element={<DeploymentCreateVersionEndpointPage />}
          />
          <Route path={':tab'} element={<DeploymentViewEndpointPage />} />
        </Route>
        {/* assistant routes */}
        <Route
          key={'/deployment/assistant'}
          path={'assistant'}
          element={<DeploymentAssistantPage />}
        />

        <Route
          path={'assistant/:assistantId'}
          element={
            <AssistantViewLayout>
              <Outlet />
            </AssistantViewLayout>
          }
        >
          <Route index element={<Navigate to="overview" replace />} />
          <Route path={':tab'} element={<DeploymentViewAssistantPage />} />
          <Route
            path={'playground/chat'}
            element={<PreviewVoiceAgentV2Page embedded />}
          />
          <Route
            path={'playground/phone'}
            element={<PreviewPhoneAgentPage embedded />}
          />

          {/* Session detail */}
          <Route
            path={'sessions/:sessionId'}
            element={<DeploymentConversationDetailPage />}
          />

          {/* Create version */}
          <Route
            path={'create-new-version'}
            element={<DeploymentCreateVersionAssistantPage />}
          />
          <Route
            path={'create-websocket-version'}
            element={<DeploymentCreateWebsocketVersionAssistantPage />}
          />
          <Route
            path={'create-agentkit-version'}
            element={<DeploymentCreateAgentKitVersionAssistantPage />}
          />
          <Route
            path={'create-agentflow-version'}
            element={<DeploymentCreateAgentflowVersionAssistantPage />}
          />

          {/* Configure pages */}
          <Route
            path={'edit-assistant/'}
            element={<DeploymentEditAssistantPage />}
          />
          <Route
            path={'edit-agentflow/'}
            element={<DeploymentCreateAgentflowVersionAssistantPage />}
          />
          <Route
            path={'configure-analysis/'}
            element={<DeploymentConfigureAssistantAnalysisPage />}
          />
          <Route
            path={'configure-analysis/create'}
            element={<DeploymentCreateAssistantAnalysisPage />}
          />
          <Route
            path={'configure-analysis/:analysisId'}
            element={<DeploymentUpdateAssistantAnalysisPage />}
          />
          <Route
            path={'configure-tool'}
            element={<DeploymentConfigureAssistantToolPage />}
          />
          <Route
            path={'configure-tool/create'}
            element={<DeploymentCreateAssistantToolPage />}
          />
          <Route
            path={'configure-tool/:assistantToolId'}
            element={<DeploymentUpdateAssistantToolPage />}
          />
          <Route
            path={'configure-webhook/'}
            element={<DeploymentConfigureAssistantWebhookPage />}
          />
          <Route
            path={'configure-webhook/create'}
            element={<DeploymentCreateAssistantWebhookPage />}
          />
          <Route
            path={'configure-webhook/:webhookId'}
            element={<DeploymentUpdateAssistantWebhookPage />}
          />
          <Route
            path={'configure-authentication/'}
            element={<DeploymentConfigureAssistantAuthenticationPage />}
          />
          <Route
            path={'configure-authentication/create'}
            element={<DeploymentCreateAssistantAuthenticationPage />}
          />
          <Route
            path={'configure-authentication/edit'}
            element={<DeploymentUpdateAssistantAuthenticationPage />}
          />
          <Route
            path={'configure-telemetry/'}
            element={<DeploymentConfigureAssistantTelemetryPage />}
          />
          <Route
            path={'configure-telemetry/create'}
            element={<DeploymentCreateAssistantTelemetryPage />}
          />
          <Route
            path={'configure-telemetry/:telemetryId'}
            element={<DeploymentUpdateAssistantTelemetryPage />}
          />
          <Route
            path={'configure-storage/'}
            element={<DeploymentConfigureAssistantStoragePage />}
          />
          <Route
            path={'configure-storage/create'}
            element={<DeploymentCreateAssistantStoragePage />}
          />
          <Route
            path={'configure-storage/:storageId'}
            element={<DeploymentUpdateAssistantStoragePage />}
          />
          <Route
            path={'deployment/'}
            element={<DeploymentConfigureAssistantDeploymentPage />}
          />
          <Route
            path={'deployment/web/'}
            element={<DeploymentConfigureAssistantWebDeploymentPage />}
          />
          <Route
            path={'deployment/web/:deploymentId/'}
            element={<DeploymentEditAssistantWebDeploymentPage />}
          />
          <Route
            path={'deployment/call/'}
            element={<DeploymentConfigureAssistantCallDeploymentPage />}
          />
          <Route
            path={'deployment/call/:deploymentId/'}
            element={<DeploymentEditAssistantCallDeploymentPage />}
          />
          <Route
            path={'deployment/api/'}
            element={<DeploymentConfigureAssistantApiDeploymentPage />}
          />
          <Route
            path={'deployment/api/:deploymentId/'}
            element={<DeploymentEditAssistantApiDeploymentPage />}
          />
          <Route
            path={'deployment/debugger/'}
            element={<DeploymentConfigureAssistantDebuggerDeploymentPage />}
          />
          <Route
            path={'deployment/debugger/:deploymentId/'}
            element={<DeploymentEditAssistantDebuggerDeploymentPage />}
          />
        </Route>
        <Route
          key={'/deployment/assistant/create-assistant'}
          path={'assistant/create-assistant'}
          element={<DeploymentCreateAssistantPage />}
        />
        <Route
          key={'/deployment/assistant/connect-websocket'}
          path={'assistant/connect-websocket'}
          element={<DeploymentCreateWebsocketPage />}
        />
        <Route
          key={'/deployment/assistant/create-agentkit'}
          path={'assistant/connect-agentkit'}
          element={<DeploymentCreateAgentKitPage />}
        />
        <Route
          key={'/deployment/assistant/create-agentflow'}
          path={'assistant/create-agentflow'}
          element={<DeploymentCreateAgentflowPage />}
        />
      </Route>
    </Routes>
  );
}
