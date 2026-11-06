import { ProtectedBox } from '@/app/components/layout/container/protected-box';
import { Outlet, type RouteObject } from 'react-router-dom';
import { ConnectGmailAction, ConnectSlackAction } from '../pages/connect/index';

export const connectActionRoute: RouteObject = {
  path: '/connect-action',
  element: (
    <ProtectedBox>
      <Outlet />
    </ProtectedBox>
  ),
  children: [
    { path: 'gmail', element: <ConnectGmailAction /> },
    { path: 'slack', element: <ConnectSlackAction /> },
    { path: '*', element: null },
  ],
};
