import { AccountSettingPage } from '@/app/pages/user';
import type { RouteObject } from 'react-router-dom';
import { breadcrumb } from './breadcrumb-handle';
import { WorkspaceRouteLayout } from './workspace-route-layout';

export const accountRoute: RouteObject = {
  path: '/account',
  element: <WorkspaceRouteLayout />,
  handle: breadcrumb('Account'),
  children: [{ index: true, element: <AccountSettingPage /> }],
};
