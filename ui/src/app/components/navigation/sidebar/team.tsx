import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { cn } from '@/utils';
import { Users } from 'lucide-react';
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
          <Users className={cn('w-5 h-5 opacity-75')} strokeWidth={1.5} />
        </SidebarIconWrapper>
        <SidebarLabel>Users and Teams</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
