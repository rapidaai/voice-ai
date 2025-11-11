import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';
import {
  OrganizationOverviewPage,
  OrganizationUserPage,
  OrganizationProjectPage,
  OrganizationBillingPage,
  OrganizationAccessSecurityPage,
} from '@/app/pages/workspace';
import { Routes, Route, Outlet } from 'react-router-dom';

export function OrganizationRoute() {
  return (
    <Routes>
      <Route
        key="/organization"
        path="/"
        element={
          <ProtectedBox>
            <MissionBox>
              <Outlet />
            </MissionBox>
          </ProtectedBox>
        }
      >
        <Route
          key="/organization/overview"
          path=""
          element={<OrganizationOverviewPage />}
        />
        <Route
          key="organization-users"
          path="users"
          element={<OrganizationUserPage />}
        />
        <Route
          key="organization-projects"
          path="projects"
          element={<OrganizationProjectPage />}
        />
        <Route
          key="organization-billing-and-invoices"
          path="billing-and-invoices"
          element={<OrganizationBillingPage />}
        />
        <Route
          key="organization-security"
          path="security"
          element={<OrganizationAccessSecurityPage />}
        />
      </Route>
    </Routes>
  );
}
