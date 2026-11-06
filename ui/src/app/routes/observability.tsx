import {
  ConversationActivityListingPage,
  KnowledgeActivityListingPage,
  LLMActivityListingPage,
  RequestActivityListingPage,
  TraceExplorerPage,
  ToolActivityListingPage,
} from '@/app/pages/activities';
import type { RouteObject } from 'react-router-dom';
import { CONFIG } from '@/configs';
import { breadcrumb } from './breadcrumb-handle';
import { WorkspaceRouteLayout } from './workspace-route-layout';

export const observabilityRoute: RouteObject = {
  path: '/logs',
  element: <WorkspaceRouteLayout />,
  handle: breadcrumb('Logs'),
  children: [
    { index: true, element: <LLMActivityListingPage /> },
    {
      path: 'request',
      element: <RequestActivityListingPage />,
      handle: breadcrumb('Requests'),
    },
    ...(CONFIG.workspace.features?.knowledge !== false
      ? [
          {
            path: 'knowledge',
            element: <KnowledgeActivityListingPage />,
            handle: breadcrumb('Knowledge'),
          },
        ]
      : []),
    {
      path: 'tool',
      element: <ToolActivityListingPage />,
      handle: breadcrumb('Tools'),
    },
    {
      path: 'conversation',
      element: <ConversationActivityListingPage />,
      handle: breadcrumb('Conversations'),
    },
    ...(CONFIG.workspace.features?.telemetry !== false
      ? [
          {
            path: 'traces',
            element: <TraceExplorerPage />,
            handle: breadcrumb('Traces'),
          },
        ]
      : []),
  ],
};
