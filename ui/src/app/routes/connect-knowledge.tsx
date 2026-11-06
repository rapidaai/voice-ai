import { ProtectedBox } from '@/app/components/layout/container/protected-box';
import {
  ConnectConfluenceKnowledge,
  ConnectGithubKnowledge,
  ConnectGoogleDriveKnowledge,
  ConnectHubspotCRM,
  ConnectNotionKnowledge,
  ConnectOneDriveKnowledge,
  ConnectSharePointKnowledge,
} from '@/app/pages/connect';
import { ProviderContextProvider } from '@/context/provider-context';
import { Outlet, type RouteObject } from 'react-router-dom';

function ConnectProviderLayout() {
  return (
    <ProtectedBox>
      <ProviderContextProvider>
        <Outlet />
      </ProviderContextProvider>
    </ProtectedBox>
  );
}

export const commonConnectRoute: RouteObject = {
  path: '/connect-common',
  element: <ConnectProviderLayout />,
  children: [
    { path: 'atlassian', element: <ConnectConfluenceKnowledge /> },
    { path: 'github', element: <ConnectGithubKnowledge /> },
    { path: '*', element: null },
  ],
};

export const connectCRMRoute: RouteObject = {
  path: '/connect-crm',
  element: <ConnectProviderLayout />,
  children: [
    { path: 'hubspot', element: <ConnectHubspotCRM /> },
    { path: '*', element: null },
  ],
};

export const connectKnowledgeRoute: RouteObject = {
  path: '/connect-knowledge',
  element: <ConnectProviderLayout />,
  children: [
    { path: 'google-drive', element: <ConnectGoogleDriveKnowledge /> },
    { path: 'one-drive', element: <ConnectOneDriveKnowledge /> },
    { path: 'confluence', element: <ConnectConfluenceKnowledge /> },
    { path: 'share-point', element: <ConnectSharePointKnowledge /> },
    { path: 'github', element: <ConnectGithubKnowledge /> },
    { path: 'notion', element: <ConnectNotionKnowledge /> },
    { path: '*', element: null },
  ],
};
