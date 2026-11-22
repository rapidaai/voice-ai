import { Disclosure } from '@/app/components/ui/primitives';
import { SidebarIconWrapper } from '@/app/components/layout/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/layout/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/layout/navigation/sidebar/sidebar-simple-list-item';
import { useSidebar } from '@/context/sidebar-context';
import { cn } from '@/utils';
import { Locked, Key, ChevronDown } from '@carbon/icons-react';
import { useState } from 'react';
import { useLocation } from 'react-router-dom';

export function Vault({ isLoading }: { isLoading?: boolean }) {
  const location = useLocation();
  const { open } = useSidebar();
  const { pathname } = location;
  const [opt, setOpt] = useState(
    false ||
      pathname.includes('/project-credential') ||
      pathname.includes('/personal-credential'),
  );

  return (
    <li>
      <SidebarSimpleListItem
        className={cn('justify-between')}
        active={opt}
        onClick={() => {
          setOpt(!opt);
        }}
        navigate="#"
        loading={isLoading}
      >
        <div className="flex items-center">
          <SidebarIconWrapper>
            <Locked size={16} />
          </SidebarIconWrapper>
          <SidebarLabel isLoading={isLoading}>Credentials</SidebarLabel>
        </div>
        <SidebarIconWrapper className="transition-all duration-100">
          <ChevronDown
            size={16}
            className={cn('transition-all duration-200', opt && 'rotate-180')}
          />
        </SidebarIconWrapper>
      </SidebarSimpleListItem>
      <Disclosure open={opt}>
        <div
          className={cn(
            'ml-6 border-l border-border-subtle',
            open ? 'block' : 'hidden',
          )}
        >
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/project-credential')}
            navigate="/integration/project-credential"
            loading={isLoading}
          >
            <SidebarIconWrapper>
              <Locked size={16} />
            </SidebarIconWrapper>
            <SidebarLabel isLoading={isLoading}>
              Project Credential
            </SidebarLabel>
          </SidebarSimpleListItem>
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/personal-credential')}
            navigate="/integration/personal-credential"
            loading={isLoading}
          >
            <SidebarIconWrapper>
              <Key size={16} />
            </SidebarIconWrapper>
            <SidebarLabel isLoading={isLoading}>Personal Token</SidebarLabel>
          </SidebarSimpleListItem>
        </div>
      </Disclosure>
    </li>
  );
}
