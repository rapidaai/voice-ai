import { UserRole } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';
import { SidebarIconWrapper } from './sidebar-icon-wrapper';
import { SidebarLabel } from './sidebar-label';
import { SidebarSimpleListItem } from './sidebar-simple-list-item';

export function Roles({ isLoading }: { isLoading?: boolean }) {
  const { pathname } = useLocation();
  const currentPath = '/organization/roles';
  return (
    <li>
      <SidebarSimpleListItem
        navigate={currentPath}
        active={pathname.includes(currentPath)}
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <UserRole size={20} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Roles & Permissions</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
