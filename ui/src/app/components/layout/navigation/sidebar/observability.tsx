import { memo } from 'react';
import { SidebarIconWrapper } from '@/app/components/layout/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/layout/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/layout/navigation/sidebar/sidebar-simple-list-item';
import { useLocation } from 'react-router-dom';
import {
  Activity,
  DataBase,
  Chat,
  DataCheck,
  EventSchedule,
  ToolKit,
} from '@carbon/icons-react';
import { useWorkspace } from '@/workspace';

export const Observability = memo(({ isLoading }: { isLoading?: boolean }) => {
  const location = useLocation();
  const { pathname } = location;
  const workspace = useWorkspace();
  const telemetryEnabled = workspace.features?.telemetry !== false;

  return (
    <li>
      {telemetryEnabled && (
        <SidebarSimpleListItem
          active={pathname.includes('/logs/traces')}
          navigate="/logs/traces"
          loading={isLoading}
        >
          <SidebarIconWrapper>
            <DataCheck size={16} />
          </SidebarIconWrapper>
          <SidebarLabel isLoading={isLoading}>Trace</SidebarLabel>
        </SidebarSimpleListItem>
      )}
      <SidebarSimpleListItem
        active={pathname.endsWith('/logs')}
        navigate="/logs"
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <Activity size={16} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>LLM logs</SidebarLabel>
      </SidebarSimpleListItem>

      <SidebarSimpleListItem
        active={pathname.includes('/logs/tool')}
        navigate="/logs/tool"
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <ToolKit size={16} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Tool logs</SidebarLabel>
      </SidebarSimpleListItem>
      <SidebarSimpleListItem
        active={pathname.includes('/logs/request')}
        navigate="/logs/request"
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <EventSchedule size={16} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Request logs</SidebarLabel>
      </SidebarSimpleListItem>
      {workspace.features?.knowledge !== false && (
        <SidebarSimpleListItem
          active={pathname.includes('/logs/knowledge')}
          navigate="/logs/knowledge"
          loading={isLoading}
        >
          <SidebarIconWrapper>
            <DataBase size={16} />
          </SidebarIconWrapper>
          <SidebarLabel isLoading={isLoading}>Knowledge logs</SidebarLabel>
        </SidebarSimpleListItem>
      )}
      <SidebarSimpleListItem
        active={pathname.includes('/logs/conversation')}
        navigate="/logs/conversation"
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <Chat size={16} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Conversation logs</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
});
