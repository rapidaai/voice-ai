import { CenterBox } from '@/app/components/container/center-box';
import { FlexBox } from '@/app/components/container/flex-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
import {
  OnboardingCreateOrganizationPage,
  OnboardingCreateProjectPage,
} from '@/app/pages/user-onboarding';
import React from 'react';
import { Outlet, Route, Routes } from 'react-router-dom';

export function OnbaordingRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <FlexBox>
              <CenterBox>
                <Outlet />
              </CenterBox>
            </FlexBox>
          </ProtectedBox>
        }
      >
        <Route
          key="organization"
          path="organization"
          element={<OnboardingCreateOrganizationPage />}
        />
        <Route
          key="project"
          path="project"
          element={<OnboardingCreateProjectPage />}
        />
      </Route>
    </Routes>
  );
}
