/**
 *
 * App
 *
 * This component is the skeleton around the actual pages, and should only
 * contain code that should be seen on all pages. (e.g. navigation bar)
 */
import {
  createBrowserRouter,
  Navigate,
  Outlet,
  RouterProvider,
  type RouteObject,
} from 'react-router-dom';
import { ReactFlowProvider } from 'reactflow';
import 'reactflow/dist/style.css';
import '@/styles/global-styles';
import * as WebRoutes from '@/app/routes';
import { GA } from '@/app/components/app-shell/ga';
import React from 'react';
import { StaticPageNotFoundPage } from '@/app/pages/static-pages';
import { AuthProvider } from '@/context/auth-context';
import { Helmet } from '@/app/components/app-shell/helmet';
import { CONFIG } from '@/configs';

function RouterEffects() {
  return (
    <>
      <GA />
      <Outlet />
    </>
  );
}

const routes: RouteObject[] = [
  {
    element: <RouterEffects />,
    children: [
      WebRoutes.authRoute,
      ...(CONFIG.workspace.features?.knowledge !== false
        ? [WebRoutes.knowledgeRoute]
        : []),
      WebRoutes.onboardingRoute,
      WebRoutes.dashboardRoute,
      WebRoutes.deploymentRoute,
      WebRoutes.integrationRoute,
      WebRoutes.accountRoute,
      WebRoutes.observabilityRoute,
      WebRoutes.organizationRoute,
      WebRoutes.previewRoute,
      WebRoutes.commonConnectRoute,
      ...(CONFIG.workspace.features?.knowledge !== false
        ? [WebRoutes.connectKnowledgeRoute]
        : []),
      WebRoutes.connectCRMRoute,
      WebRoutes.connectActionRoute,
      { path: '/', element: <Navigate to="/auth/signin" replace /> },
      WebRoutes.staticRoute,
      { path: '*', element: <StaticPageNotFoundPage /> },
    ],
  },
];

const router = createBrowserRouter(routes);

/**
 * Main app containers
 * @returns
 */
export function App() {
  return (
    <React.Fragment>
      <Helmet title="Home" />
      <AuthProvider>
        <ReactFlowProvider>
          <RouterProvider
            router={router}
            future={{ v7_startTransition: true }}
          />
        </ReactFlowProvider>
      </AuthProvider>
    </React.Fragment>
  );
}
