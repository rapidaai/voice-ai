import {
  IntegrationModelPage,
  IntegrationModelInformationPage,
  IntegrationProjectCredentialPage,
} from '@/app/pages/external-integration';
import type { RouteObject } from 'react-router-dom';
import { IntegrationPersonalCredentialPage } from '../pages/external-integration/index';
import { breadcrumb } from './breadcrumb-handle';
import { WorkspaceRouteLayout } from './workspace-route-layout';

export const integrationRoute: RouteObject = {
  path: '/integration',
  element: <WorkspaceRouteLayout />,
  handle: breadcrumb('Integrations'),
  children: [
    { index: true, element: <IntegrationModelPage /> },
    {
      path: 'models',
      handle: breadcrumb('Models'),
      children: [
        { index: true, element: <IntegrationModelPage /> },
        {
          path: ':provider',
          element: <IntegrationModelInformationPage />,
          handle: breadcrumb('Provider'),
        },
      ],
    },
    {
      path: 'project-credential',
      element: <IntegrationProjectCredentialPage />,
      handle: breadcrumb('Project credentials'),
    },
    {
      path: 'personal-credential',
      element: <IntegrationPersonalCredentialPage />,
      handle: breadcrumb('Personal credentials'),
    },
  ],
};
