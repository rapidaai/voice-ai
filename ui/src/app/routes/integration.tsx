import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
import {
  IntegrationModelPage,
  IntegrationProjectCredentialPage,
  IntegrationExternalToolPage,
} from '@/app/pages/external-integration';
import React from 'react';
import { Routes, Route, Outlet } from 'react-router-dom';
import { IntegrationPersonalCredentialPage } from '../pages/external-integration/index';

export function IntegrationRoute() {
  return (
    <Routes>
      <Route
        key="/integration/"
        path="/"
        element={
          <ProtectedBox>
            <MissionBox>
              <Outlet />
            </MissionBox>
          </ProtectedBox>
        }
      >
        <Route path="" element={<IntegrationModelPage />} />
        <Route path="models" element={<IntegrationModelPage />} />
        <Route
          key="project-credential"
          path="project-credential"
          element={<IntegrationProjectCredentialPage />}
        />
        <Route
          key="personal-credential"
          path="personal-credential"
          element={<IntegrationPersonalCredentialPage />}
        />
        <Route
          key="tools"
          path="tools"
          element={<IntegrationExternalToolPage />}
        />
      </Route>
    </Routes>
  );
}
