import { SidebarIconWrapper } from '@/app/components/layout/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/layout/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/layout/navigation/sidebar/sidebar-simple-list-item';
import { UserMultiple } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';

export function Team({ isLoading }: { isLoading?: boolean }) {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/organization/users';
  return (
    <li>
      <SidebarSimpleListItem
        navigate={currentPath}
        active={pathname.includes(currentPath)}
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <UserMultiple size={16} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Users and Teams</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
