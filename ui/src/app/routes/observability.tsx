import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
import {
  ConversationActivityListingPage,
  KnowledgeActivityListingPage,
  LLMActivityListingPage,
  ToolActivityListingPage,
  WebhookActivityListingPage,
} from '@/app/pages/activities';
import { Routes, Route, Outlet } from 'react-router-dom';

export function ObservabilityRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <MissionBox>
              <Outlet />
            </MissionBox>
          </ProtectedBox>
        }
      >
        <Route index key="logs" path="/" element={<LLMActivityListingPage />} />
        <Route
          key="webhook-logs"
          path="/webhook"
          element={<WebhookActivityListingPage />}
        />
        <Route
          key="knowledge-logs"
          path="/knowledge"
          element={<KnowledgeActivityListingPage />}
        />
        <Route
          key="tool-logs"
          path="/tool"
          element={<ToolActivityListingPage />}
        />
        <Route
          key="conversation-logs"
          path="/conversation"
          element={<ConversationActivityListingPage />}
        />
      </Route>
    </Routes>
  );
}
