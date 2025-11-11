/**
 *
 * App
 *
 * This component is the skeleton around the actual pages, and should only
 * contain code that should be seen on all pages. (e.g. navigation bar)
 */
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { GlobalStyle } from '@/styles/global-styles';
import * as WebRoutes from '@/app/routes';
import { GA } from '@/app/components/ga';
import React from 'react';
import { StaticPageNotFoundPage } from '@/app/pages/static-pages';
import { AuthProvider } from '@/context/auth-context';
import { useWorkspace } from '@/context/workplace-context';
import { Helmet } from '@/app/components/Helmet';

/**
 * Main app containers
 * @returns
 */
export function App() {
  const workspace = useWorkspace();
  return (
    <React.Fragment>
      <Helmet title="Home" />
      <AuthProvider>
        <BrowserRouter future={{ v7_startTransition: true }}>
          <GA />
          <Routes>
            <Route index path="/auth/*" element={<WebRoutes.AuthRoute />} />
            <Route path="/knowledge/*" element={<WebRoutes.KnowledgeRoute />} />
            <Route
              path="/onboarding/*"
              element={<WebRoutes.OnbaordingRoute />}
            />
            <Route path="/dashboard/*" element={<WebRoutes.DashboardRoute />} />
            <Route
              path="/deployment/*"
              element={<WebRoutes.DeploymentRoute />}
            />

            <Route
              path="/integration/*"
              element={<WebRoutes.IntegrationRoute />}
            />
            <Route path="/account/*" element={<WebRoutes.AccountRoute />} />
            <Route path="/logs/*" element={<WebRoutes.ObservabilityRoute />} />
            <Route
              path="/organization/*"
              element={<WebRoutes.OrganizationRoute />}
            />
            <Route path="/preview/*" element={<WebRoutes.PreviewRoute />} />
            <Route
              path="/connect-common/*"
              element={<WebRoutes.CommonConnectRoute />}
            />
            <Route
              path="/connect-knowledge/*"
              element={<WebRoutes.ConnectKnowledgeRoute />}
            />

            <Route
              path="/connect-crm/*"
              element={<WebRoutes.ConnectCRMRoute />}
            />

            <Route
              path="/connect-action/*"
              element={<WebRoutes.ConnectActionRoute />}
            />

            <Route
              key="/"
              path="/"
              element={<Navigate to={'/auth/signin'} replace />}
            />
            <Route path="/static/*" element={<WebRoutes.StaticRoute />} />
            <Route path="*" element={<StaticPageNotFoundPage />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
      <GlobalStyle />
    </React.Fragment>
  );
}
