import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
import { AccountPersonalSettingPage } from '@/app/pages/Account';
import React from 'react';
import { Routes, Route } from 'react-router-dom';

export function AccountRoute() {
  return (
    <Routes>
      <Route
        key="personal-settings"
        path="personal-settings"
        element={
          <ProtectedBox>
            <MissionBox>
              <AccountPersonalSettingPage />
            </MissionBox>
          </ProtectedBox>
        }
      />
    </Routes>
  );
}
