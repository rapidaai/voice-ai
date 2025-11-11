import { ModelIcon } from '@/app/components/Icon/Model';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import React from 'react';
import { useLocation } from 'react-router-dom';

export function Model() {
  const location = useLocation();
  const { pathname } = location;
  return (
    <li>
      <SidebarSimpleListItem
        navigate="/integration/models"
        active={pathname.includes('/integration/models')}
      >
        <SidebarIconWrapper>
          <ModelIcon />
        </SidebarIconWrapper>
        <SidebarLabel>Models</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
