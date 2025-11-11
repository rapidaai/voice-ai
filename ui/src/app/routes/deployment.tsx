import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
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
  DeploymentConfigureAssistantCallDeploymentPage,
  DeploymentConfigureAssistantToolPage,
  DeploymentConfigureAssistantAnalysisPage,
  DeploymentConfigureAssistantApiDeploymentPage,
  DeploymentConfigureAssistantDebuggerDeploymentPage,
  DeploymentEditAssistantPage,
  DeploymentConfigureAssistantWebhookPage,
  DeploymentCreateAssistantWebhookPage,
  DeploymentUpdateAssistantWebhookPage,
  DeploymentConversationDetailPage,
  DeploymentCreateAssistantAnalysisPage,
  DeploymentUpdateAssistantAnalysisPage,
  DeploymentUpdateAssistantToolPage,
  DeploymentCreateAssistantToolPage,
  DeploymentConfigureAssistantDeploymentPage,
  DeploymentCreateWebsocketVersionAssistantPage,
  DeploymentCreateAgentKitVersionAssistantPage,
  DeploymentCreateAgentKitPage,
  DeploymentCreateWebsocketPage,
} from '@/app/pages/assistant';
import { AssistantManageLayout } from '@/app/pages/assistant/actions/assistant-manage.layout';
import { AssistantViewLayout } from '@/app/pages/assistant/view/assistant-view.layout';
import { StaticPageNotFoundPage } from '@/app/pages/static-pages';
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
          key={'/endpoint/'}
          path={''}
          element={<DeploymentEndpointPage />}
        />
        <Route
          key={'/deployment/endpoint'}
          path={'endpoint'}
          element={<DeploymentEndpointPage />}
        />

        <Route
          path={'endpoint/create-endpoint'}
          element={<DeploymentCreateEndpointPage />}
        />
        <Route
          path={'endpoint/:endpointId/create-endpoint-version'}
          element={<DeploymentCreateVersionEndpointPage />}
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
        <Route
          path={'endpoint/:endpointId'}
          element={<DeploymentViewEndpointPage />}
        />
        <Route
          path={'endpoint/:endpointId/:endpointProviderId'}
          element={<DeploymentViewEndpointPage />}
        />
        {/* assistant routes */}
        <Route
          key={'/deployment/assistant'}
          path={'assistant'}
          element={<DeploymentAssistantPage />}
        />

        <Route
          path={'assistant/:assistantId/create-new-version'}
          element={<DeploymentCreateVersionAssistantPage />}
        />

        <Route
          path={'assistant/:assistantId/create-websocket-version'}
          element={<DeploymentCreateWebsocketVersionAssistantPage />}
        />

        <Route
          path={'assistant/:assistantId/create-agentkit-version'}
          element={<DeploymentCreateAgentKitVersionAssistantPage />}
        />

        <Route
          key={'assistant/:assistantId/sessions/:sessionId'}
          path={'assistant/:assistantId/sessions/:sessionId'}
          element={<DeploymentConversationDetailPage />}
        />
        <Route
          path="assistant/:assistantId/manage"
          element={
            <AssistantManageLayout>
              <Outlet />
            </AssistantManageLayout>
          }
        >
          <Route index element={<Navigate to="deployment/" replace />} />
          <Route
            key={'/deployment/edit-assistant'}
            path={'edit-assistant/'}
            element={<DeploymentEditAssistantPage />}
          />

          <Route
            key={'/deployment/analysis'}
            path={'configure-analysis/'}
            element={<DeploymentConfigureAssistantAnalysisPage />}
          />

          <Route
            key={'/deployment/analysis'}
            path={'configure-analysis/create'}
            element={<DeploymentCreateAssistantAnalysisPage />}
          />

          <Route
            key={'/deployment/analysis'}
            path={'configure-analysis/:analysisId'}
            element={<DeploymentUpdateAssistantAnalysisPage />}
          />

          {/* <Route
            key={'/deployment/content-filter'}
            path={'configure-content-filter/'}
            element={<DeploymentConfigureAssistantContentFilterPage />}
          /> */}
          {/* <Route
            key={'/deployment/contextual-grounding'}
            path={'configure-contextual-grounding/'}
            element={<DeploymentConfigureAssistantContextualGroundingPage />}
          /> */}

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
          {/* <Route
            key={'/deployment/knowledge'}
            path={'configure-knowledge/'}
            element={<DeploymentConfigureAssistantKnowledgePage />}
          />
          <Route
            key={'/deployment/knowledge'}
            path={'configure-knowledge/create'}
            element={<DeploymentCreateAssistantKnowledgePage />}
          />
          <Route
            key={'/deployment/knowledge'}
            path={'configure-knowledge/:assistantKnowledgeId'}
            element={<DeploymentUpdateAssistantKnowledgePage />}
          /> */}

          <Route
            key={'/deployment/webhook'}
            path={'configure-webhook/'}
            element={<DeploymentConfigureAssistantWebhookPage />}
          />
          <Route
            key={'/deployment/webhook'}
            path={'configure-webhook/create'}
            element={<DeploymentCreateAssistantWebhookPage />}
          />
          <Route
            key={'/deployment/webhook'}
            path={'configure-webhook/:webhookId'}
            element={<DeploymentUpdateAssistantWebhookPage />}
          />

          <Route
            key={'/deployment/assistant/deployment/:assistantId'}
            path={'deployment/'}
            element={<DeploymentConfigureAssistantDeploymentPage />}
          />

          <Route
            key={'/deployment/assistant/deployment/web/:assistantId'}
            path={'deployment/web/'}
            element={<DeploymentConfigureAssistantWebDeploymentPage />}
          />

          <Route
            key={'/deployment/assistant/deployment/call/:assistantId'}
            path={'deployment/call/'}
            element={<DeploymentConfigureAssistantCallDeploymentPage />}
          />
          <Route
            key={'/deployment/assistant/deployment/api/:assistantId'}
            path={'deployment/api/'}
            element={<DeploymentConfigureAssistantApiDeploymentPage />}
          />
          <Route
            key={'/deployment/assistant/deployment/debugger/:assistantId'}
            path={'deployment/debugger/'}
            element={<DeploymentConfigureAssistantDebuggerDeploymentPage />}
          />
        </Route>

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
      </Route>
    </Routes>
  );
}
