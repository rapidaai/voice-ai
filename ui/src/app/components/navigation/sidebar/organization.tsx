import { OrganizationIcon } from '@/app/components/Icon/Organization';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import React from 'react';
import { useLocation } from 'react-router-dom';

export function Organization() {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/organization';
  return (
    <SidebarSimpleListItem
      navigate={currentPath}
      active={pathname.endsWith(currentPath)}
    >
      <SidebarIconWrapper>
        <OrganizationIcon />
      </SidebarIconWrapper>
      <SidebarLabel>Overview</SidebarLabel>
    </SidebarSimpleListItem>
  );
}
