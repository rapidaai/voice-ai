import { memo } from 'react';
import { SidebarIconWrapper } from '@/app/components/layout/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/layout/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/layout/navigation/sidebar/sidebar-simple-list-item';
import { useLocation } from 'react-router-dom';
import { ChatBot, Connect } from '@carbon/icons-react';

export const Deployment = memo(({ isLoading }: { isLoading?: boolean }) => {
  const location = useLocation();
  const { pathname } = location;

  return (
    <li>
      <SidebarSimpleListItem
        active={pathname.includes('/deployment/assistant')}
        navigate="/deployment/assistant"
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <ChatBot size={16} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Assistants</SidebarLabel>
      </SidebarSimpleListItem>
      <SidebarSimpleListItem
        active={pathname.includes('/deployment/endpoint')}
        navigate="/deployment/endpoint"
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <Connect size={16} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Endpoints</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
});
