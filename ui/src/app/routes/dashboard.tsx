import { DashboardHomePage } from '@/app/pages/main';
import type { RouteObject } from 'react-router-dom';
import { breadcrumb } from './breadcrumb-handle';
import { WorkspaceRouteLayout } from './workspace-route-layout';

export const dashboardRoute: RouteObject = {
  path: '/dashboard',
  element: <WorkspaceRouteLayout />,
  handle: breadcrumb('Dashboard'),
  children: [{ index: true, element: <DashboardHomePage /> }],
};
