import { PublicPreviewVoiceAgentPage } from '@/app/pages/preview-agent';
import { Navigate, type RouteObject, useParams } from 'react-router-dom';

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

export const previewRoute: RouteObject = {
  path: '/preview',
  children: [
    {
      path: 'call/:assistantId',
      element: <AssistantPlaygroundRedirect channel="phone" />,
    },
    {
      path: 'chat/:assistantId',
      element: <AssistantPlaygroundRedirect channel="chat" />,
    },
    {
      path: 'v2/chat/:assistantId',
      element: <AssistantPlaygroundRedirect channel="chat" />,
    },
    {
      path: 'public/assistant/:assistantId',
      element: <PublicPreviewVoiceAgentPage />,
    },
    { path: '*', element: null },
  ],
};
