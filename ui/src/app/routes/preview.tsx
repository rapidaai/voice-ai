import { ProtectedBox } from '@/app/components/layout/container/protected-box';
import { Outlet, Route, Routes } from 'react-router-dom';
import {
  PreviewPhoneAgentPage,
  PreviewVoiceAgentPage,
  PreviewVoiceAgentV2Page,
  PublicPreviewVoiceAgentPage,
} from '@/app/pages/preview-agent';

/**
 *
 * @returns all the route which is needed for preview
 */
export function PreviewRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <Outlet />
          </ProtectedBox>
        }
      >
        <Route
          key="/assistant/call"
          path="call/:assistantId/"
          element={<PreviewPhoneAgentPage />}
        />
        <Route
          key="/assistant/chat"
          path="chat/:assistantId"
          element={<PreviewVoiceAgentPage />}
        />
        <Route
          key="/assistant/v2/chat"
          path="v2/chat/:assistantId"
          element={<PreviewVoiceAgentV2Page />}
        />
      </Route>
      <Route
        path="public/assistant/:assistantId"
        element={<PublicPreviewVoiceAgentPage />}
      />
    </Routes>
  );
}
