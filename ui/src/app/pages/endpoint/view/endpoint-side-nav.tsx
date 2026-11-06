import {
  Activity,
  Dashboard,
  Debug,
  Settings,
  SidePanelClose,
  SidePanelOpen,
  SourceControl,
} from '@carbon/icons-react';
import {
  SideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
} from '@carbon/react';
import { Fragment, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { cn } from '@/utils';
import { GhostButton } from '@/app/components/ui/primitives';

export function EndpointSideNav(props: { endpointId: string }) {
  const [expanded, setExpanded] = useState(true);
  const { pathname } = useLocation();
  const basePath = `/deployment/endpoint/${props.endpointId}`;

  const isActive = (path: string) => {
    if (path === 'versions' && pathname.endsWith('/create-endpoint-version')) {
      return true;
    }
    return pathname === `${basePath}/${path}`;
  };

  return (
    <div
      className={cn(
        'relative flex h-full shrink-0 flex-col',
        'border-r border-border-subtle bg-surface ',
        'transition-all duration-200',
        expanded ? 'w-56' : 'w-12',
      )}
    >
      <SideNav
        aria-label="Endpoint actions"
        expanded={expanded}
        isRail={!expanded}
        className="relative! inset-auto! z-0! h-auto! w-full! flex-1 border-none!"
      >
        <SideNavItems>
          <SideNavLink
            renderIcon={Dashboard}
            href={`${basePath}/overview`}
            isActive={isActive('overview')}
          >
            Overview
          </SideNavLink>
          <SideNavLink
            renderIcon={Activity}
            href={`${basePath}/logs`}
            isActive={isActive('logs')}
          >
            Logs
          </SideNavLink>
          <SideNavMenu
            title="Versions"
            renderIcon={SourceControl}
            isActive={isActive('versions')}
            defaultExpanded={isActive('versions')}
          >
            <SideNavMenuItem
              href={`${basePath}/versions`}
              isActive={pathname === `${basePath}/versions`}
            >
              View all
            </SideNavMenuItem>
            <SideNavMenuItem
              href={`${basePath}/create-endpoint-version`}
              isActive={pathname.endsWith('/create-endpoint-version')}
            >
              Add new version
            </SideNavMenuItem>
          </SideNavMenu>

          <Fragment>
            <li
              className={cn(
                'cds--switcher__item--divider transition-all duration-200',
                !expanded &&
                  'opacity-0 h-0 overflow-hidden !py-0 !my-0 !border-none',
              )}
            >
              <span className="uppercase!">Settings</span>
            </li>
            <SideNavLink
              renderIcon={Settings}
              href={`${basePath}/settings`}
              isActive={isActive('settings')}
            >
              General
            </SideNavLink>
          </Fragment>

          <Fragment>
            <li
              className={cn(
                'cds--switcher__item--divider transition-all duration-200',
                !expanded &&
                  'opacity-0 h-0 overflow-hidden !py-0 !my-0 !border-none',
              )}
            >
              <span className="uppercase!">Playground</span>
            </li>
            <SideNavLink
              renderIcon={Debug}
              href={`${basePath}/playground`}
              isActive={isActive('playground')}
            >
              Open playground
            </SideNavLink>
          </Fragment>
        </SideNavItems>
      </SideNav>

      <div className="shrink-0 border-t border-border-subtle ">
        <GhostButton
          size="sm"
          type="button"
          onClick={() => setExpanded(current => !current)}
          renderIcon={expanded ? SidePanelClose : SidePanelOpen}
          hasIconOnly={!expanded}
          iconDescription={expanded ? 'Collapse nav' : 'Expand nav'}
          className={cn(
            'h-10 w-full max-w-none!',
            !expanded && 'justify-center',
          )}
        >
          {expanded ? 'Collapse' : null}
        </GhostButton>
      </div>
    </div>
  );
}
