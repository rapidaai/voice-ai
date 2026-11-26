import {
  OrganizationOverviewPage,
  OrganizationUserPage,
  OrganizationProjectPage,
  OrganizationAccessSecurityPage,
  OrganizationRoleCatalogPage,
  OrganizationRoleDetailPage,
} from '@/app/pages/workspace';
import type { RouteObject } from 'react-router-dom';
import { breadcrumb } from './breadcrumb-handle';
import { WorkspaceRouteLayout } from './workspace-route-layout';

export const organizationRoute: RouteObject = {
  path: '/organization',
  element: <WorkspaceRouteLayout />,
  handle: breadcrumb('Organization'),
  children: [
    { index: true, element: <OrganizationOverviewPage /> },
    {
      path: 'users',
      element: <OrganizationUserPage />,
      handle: breadcrumb('Users'),
    },
    {
      path: 'projects',
      element: <OrganizationProjectPage />,
      handle: breadcrumb('Projects'),
    },
    {
      path: 'roles',
      element: <OrganizationRoleCatalogPage />,
      handle: breadcrumb('Roles & Permissions'),
    },
    {
      path: 'roles/:roleDefinitionId',
      element: <OrganizationRoleDetailPage />,
      handle: breadcrumb('Role detail'),
    },
    {
      path: 'security',
      element: <OrganizationAccessSecurityPage />,
      handle: breadcrumb('Security'),
    },
  ],
};
