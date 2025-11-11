import { ProtectedBox } from '@/app/components/container/protected-box';
import { Outlet, Route, Routes } from 'react-router-dom';
import { ConnectGmailAction, ConnectSlackAction } from '../pages/connect/index';

/**
 *
 * @returns all the route which is needed for preview
 */
export function ConnectActionRoute() {
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
          key="/connect-action/gmail"
          path="gmail"
          element={<ConnectGmailAction />}
        />
        <Route
          key="/connect-action/slack"
          path="slack"
          element={<ConnectSlackAction />}
        />
      </Route>
    </Routes>
  );
}
