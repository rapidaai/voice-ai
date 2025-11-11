import { memo, useState } from 'react';
import { ChevronDownIcon } from '@/app/components/Icon/ChevronDown';
import { Disclosure } from '@/app/components/Disclosure';
import { ChevronUpIcon } from '@/app/components/Icon/ChevronUp';
import { cn } from '@/styles/media';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { useLocation } from 'react-router-dom';

import { ObservabilityIcon } from '@/app/components/Icon/Observability';
import { Activity, Database, MessageSquareIcon, Webhook } from 'lucide-react';
import { ToolIcon } from '@/app/components/Icon/tool';

export const Observability = memo(() => {
  const location = useLocation();
  const { pathname } = location;
  const [open, setOpen] = useState(false || pathname.includes('/logs'));

  return (
    <li>
      <SidebarSimpleListItem
        className={cn('justify-between')}
        active={pathname.includes('/logs')}
        onClick={() => {
          setOpen(!open);
        }}
        navigate="#"
      >
        <div className="flex items-center">
          <SidebarIconWrapper>
            <ObservabilityIcon />
          </SidebarIconWrapper>
          <SidebarLabel>Logs</SidebarLabel>
        </div>
        <SidebarIconWrapper className="opacity-0 group-hover:opacity-100 transition-all duration-100">
          {open ? <ChevronUpIcon /> : <ChevronDownIcon />}
        </SidebarIconWrapper>
      </SidebarSimpleListItem>
      <Disclosure open={open}>
        <div className="ml-6 dark:border-gray-800 border-l hidden group-hover:block">
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.endsWith('/logs')}
            navigate="/logs"
          >
            <SidebarIconWrapper>
              <Activity className="w-4 h-5" strokeWidth={1.5} />
            </SidebarIconWrapper>
            <SidebarLabel>LLM logs</SidebarLabel>
          </SidebarSimpleListItem>

          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/logs/tool')}
            navigate="/logs/tool"
          >
            <SidebarIconWrapper>
              <ToolIcon className="w-4 h-5" strokeWidth={1.5} />
            </SidebarIconWrapper>
            <SidebarLabel>Tool logs</SidebarLabel>
          </SidebarSimpleListItem>
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/logs/webhook')}
            navigate="/logs/webhook"
          >
            <SidebarIconWrapper>
              <Webhook className="w-4 h-5" strokeWidth={1.5} />
            </SidebarIconWrapper>
            <SidebarLabel>Webhook logs</SidebarLabel>
          </SidebarSimpleListItem>
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/logs/knowledge')}
            navigate="/logs/knowledge"
          >
            <SidebarIconWrapper>
              <Database className="w-4 h-5" strokeWidth={1.5} />
            </SidebarIconWrapper>
            <SidebarLabel>Knowledge logs</SidebarLabel>
          </SidebarSimpleListItem>
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/logs/conversation')}
            navigate="/logs/conversation"
          >
            <SidebarIconWrapper>
              <MessageSquareIcon className="w-4 h-5" strokeWidth={1.5} />
            </SidebarIconWrapper>
            <SidebarLabel>Conversation logs</SidebarLabel>
          </SidebarSimpleListItem>
        </div>
      </Disclosure>
    </li>
  );
});
