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
  Button,
  SideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
} from '@carbon/react';
import { Fragment, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { cn } from '@/utils';

export function EndpointSideNav(props: { endpointId: string }) {
  const [expanded, setExpanded] = useState(
    () => !window.matchMedia?.('(max-width: 767px)').matches,
  );
  const { pathname } = useLocation();
  const basePath = `/deployment/endpoint/${props.endpointId}`;

  useEffect(() => {
    const media = window.matchMedia?.('(max-width: 767px)');
    if (!media) return undefined;

    const collapseOnMobile = (event: MediaQueryListEvent) => {
      if (event.matches) setExpanded(false);
    };

    if (media.matches) setExpanded(false);
    media.addEventListener('change', collapseOnMobile);
    return () => media.removeEventListener('change', collapseOnMobile);
  }, []);

  const isActive = (path: string) => {
    if (path === 'versions' && pathname.endsWith('/create-endpoint-version')) {
      return true;
    }
    return pathname === `${basePath}/${path}`;
  };

  return (
    <div
      className={cn(
        'relative shrink-0 flex flex-col h-full',
        'bg-shell text-foreground',
        'border-r border-border-subtle',
        'transition-[width] duration-200 ease-in-out',
        expanded ? 'w-64' : 'w-12',
      )}
    >
      <SideNav
        aria-label="Endpoint actions"
        expanded={expanded}
        isFixedNav
        isRail={!expanded}
        className="relative! inset-auto! h-auto! flex-1 w-full! transform-none! border-none! bg-transparent! z-0!"
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

      <div className="shrink-0 border-t border-border-subtle">
        <Button
          type="button"
          kind="ghost"
          size="sm"
          onClick={() => setExpanded(current => !current)}
          aria-label={
            expanded
              ? 'Collapse endpoint navigation'
              : 'Expand endpoint navigation'
          }
          className={cn(
            '!flex !h-10 !w-full !max-w-none !items-center !justify-start !px-4',
            '!text-muted',
            'hover:!bg-layer-hover hover:!text-foreground',
            '!transition-colors !duration-100',
          )}
        >
          <span className="shrink-0">
            {expanded ? (
              <SidePanelClose size={16} />
            ) : (
              <SidePanelOpen size={16} />
            )}
          </span>
          <span
            className={cn(
              'text-xs truncate transition-all duration-200 ml-3',
              expanded ? 'opacity-100' : 'opacity-0 w-0 ml-0 overflow-hidden',
            )}
          >
            {expanded ? 'Collapse' : 'Expand'}
          </span>
        </Button>
      </div>
    </div>
  );
}
