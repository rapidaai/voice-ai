import { MissionBox } from '@/app/components/layout/container/mission-box';
import { ProtectedBox } from '@/app/components/layout/container/protected-box';
import { Outlet } from 'react-router-dom';

export function WorkspaceRouteLayout() {
  return (
    <ProtectedBox>
      <MissionBox>
        <Outlet />
      </MissionBox>
    </ProtectedBox>
  );
}
