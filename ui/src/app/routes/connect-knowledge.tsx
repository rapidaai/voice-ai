import { ProtectedBox } from '@/app/components/container/protected-box';
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
import { Outlet, Route, Routes } from 'react-router-dom';

/**
 *
 * @returns all the route which is needed for preview
 */

export const CommonConnectRoute = () => {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <ProviderContextProvider>
              <Outlet />
            </ProviderContextProvider>
          </ProtectedBox>
        }
      >
        <Route
          key="/connect-common/confluence"
          path="atlassian"
          element={<ConnectConfluenceKnowledge />}
        />
        <Route
          key="/connect-common/github"
          path="github"
          element={<ConnectGithubKnowledge />}
        />
      </Route>
    </Routes>
  );
};

export const ConnectCRMRoute = () => {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <ProviderContextProvider>
              <Outlet />
            </ProviderContextProvider>
          </ProtectedBox>
        }
      >
        <Route
          key="/connect-crm/hubspot"
          path="hubspot"
          element={<ConnectHubspotCRM />}
        />
      </Route>
    </Routes>
  );
};

export function ConnectKnowledgeRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <ProviderContextProvider>
              <Outlet />
            </ProviderContextProvider>
          </ProtectedBox>
        }
      >
        <Route
          key="/connect-knowledge/google-drive"
          path="google-drive"
          element={<ConnectGoogleDriveKnowledge />}
        />
        <Route
          key="/connect-knowledge/one-drive"
          path="one-drive"
          element={<ConnectOneDriveKnowledge />}
        />
        <Route
          key="/connect-knowledge/confluence"
          path="confluence"
          element={<ConnectConfluenceKnowledge />}
        />
        <Route
          key="/connect-knowledge/share-point"
          path="share-point"
          element={<ConnectSharePointKnowledge />}
        />
        <Route
          key="/connect-knowledge/github"
          path="github"
          element={<ConnectGithubKnowledge />}
        />
        <Route
          key="/connect-knowledge/notion"
          path="notion"
          element={<ConnectNotionKnowledge />}
        />
      </Route>
    </Routes>
  );
}
