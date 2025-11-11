import { AccountIcon } from '@/app/components/Icon/Account';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import React from 'react';
import { useLocation } from 'react-router-dom';

export function Account() {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/account/personal-settings';

  return (
    <SidebarSimpleListItem
      navigate="/account/personal-settings"
      active={pathname.includes(currentPath)}
    >
      <SidebarIconWrapper>
        <AccountIcon />
      </SidebarIconWrapper>
      <SidebarLabel>Account Settings</SidebarLabel>
    </SidebarSimpleListItem>
  );
}
