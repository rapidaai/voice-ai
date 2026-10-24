import { PublicPreviewVoiceAgentPage } from '@/app/pages/preview-agent';
import { Navigate, Route, Routes, useParams } from 'react-router-dom';

function AssistantPlaygroundRedirect({
  channel,
}: {
  channel: 'chat' | 'phone';
}) {
  const { assistantId } = useParams();

  if (!assistantId) {
    return <Navigate to="/404" replace />;
  }

  return (
    <Navigate
      to={`/deployment/assistant/${assistantId}/playground/${channel}`}
      replace
    />
  );
}

/**
 *
 * @returns all the route which is needed for preview
 */
export function PreviewRoute() {
  return (
    <Routes>
      <Route
        path="call/:assistantId/"
        element={<AssistantPlaygroundRedirect channel="phone" />}
      />
      <Route
        path="chat/:assistantId"
        element={<AssistantPlaygroundRedirect channel="chat" />}
      />
      <Route
        path="v2/chat/:assistantId"
        element={<AssistantPlaygroundRedirect channel="chat" />}
      />
      <Route
        path="public/assistant/:assistantId"
        element={<PublicPreviewVoiceAgentPage />}
      />
    </Routes>
  );
}
