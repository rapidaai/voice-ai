import { DashboardIcon } from '@/app/components/Icon/Dashboard';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { useLocation } from 'react-router-dom';

export function Dashboard() {
  /**
   *
   */
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/dashboard';
  return (
    <SidebarSimpleListItem
      navigate={currentPath}
      active={pathname.includes(currentPath)}
    >
      <SidebarIconWrapper>
        <DashboardIcon />
      </SidebarIconWrapper>
      <SidebarLabel>Dashboard</SidebarLabel>
    </SidebarSimpleListItem>
  );
}
