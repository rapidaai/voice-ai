import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
import { DashboardHomePage } from '@/app/pages/main';
import React from 'react';
import { Route, Routes } from 'react-router-dom';

export function DashboardRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <MissionBox>
              <DashboardHomePage />
            </MissionBox>
          </ProtectedBox>
        }
      />
    </Routes>
  );
}
