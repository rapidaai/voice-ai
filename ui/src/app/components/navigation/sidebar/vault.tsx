import { Disclosure } from '@/app/components/Disclosure';
import { ChevronDownIcon } from '@/app/components/Icon/ChevronDown';
import { ChevronUpIcon } from '@/app/components/Icon/ChevronUp';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { cn } from '@/styles/media';
import { FolderKey, Key, KeyIcon, KeySquare } from 'lucide-react';
import { useState } from 'react';
import { useLocation } from 'react-router-dom';

export function Vault() {
  const location = useLocation();
  const { pathname } = location;
  const [open, setOpen] = useState(
    false ||
      pathname.includes('/project-credential') ||
      pathname.includes('/personal-credential'),
  );

  return (
    <li>
      <SidebarSimpleListItem
        className={cn('justify-between')}
        active={open}
        onClick={() => {
          setOpen(!open);
        }}
        navigate="#"
      >
        <div className="flex items-center">
          <SidebarIconWrapper>
            <RapidaIcon />
          </SidebarIconWrapper>
          <SidebarLabel>Credentials</SidebarLabel>
        </div>
        <SidebarIconWrapper className="opacity-0 group-hover:opacity-100 transition-all duration-100">
          {open ? <ChevronUpIcon /> : <ChevronDownIcon />}
        </SidebarIconWrapper>
      </SidebarSimpleListItem>
      <Disclosure open={open}>
        <div className="ml-6 dark:border-gray-800 border-l hidden group-hover:block">
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/project-credential')}
            navigate="/integration/project-credential"
          >
            <SidebarIconWrapper>
              <FolderKey
                className={cn('w-4 h-4 opacity-75')}
                strokeWidth={1.5}
              />
            </SidebarIconWrapper>
            <SidebarLabel>Project Credential</SidebarLabel>
          </SidebarSimpleListItem>
          <SidebarSimpleListItem
            className="mx-0 mr-2"
            active={pathname.includes('/personal-credential')}
            navigate="/integration/personal-credential"
          >
            <SidebarIconWrapper>
              <Key className={cn('w-4 h-4 opacity-75')} strokeWidth={1.5} />
            </SidebarIconWrapper>
            <SidebarLabel>Personal Token</SidebarLabel>
          </SidebarSimpleListItem>
        </div>
      </Disclosure>
    </li>
  );
}
