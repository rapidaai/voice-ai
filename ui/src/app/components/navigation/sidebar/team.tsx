import { TeamIcon } from '@/app/components/Icon/Team';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { useLocation } from 'react-router-dom';

export function Team() {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/organization/users';
  return (
    <li>
      <SidebarSimpleListItem
        navigate={currentPath}
        active={pathname.includes(currentPath)}
      >
        <SidebarIconWrapper>
          <TeamIcon />
        </SidebarIconWrapper>
        <SidebarLabel>Users and Teams</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
