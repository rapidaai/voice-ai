import {
  Activity,
  Dashboard,
  Debug,
  Settings,
  SourceControl,
} from '@carbon/icons-react';
import {
  SideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
} from '@carbon/react';
import { Fragment, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import {
  getSidenavExpanded,
  setSidenavExpanded,
} from '@/context/sidenav-storage';
import {
  SideNavigationCollapseControl,
  SideNavigationPanel,
  SideNavigationSectionLabel,
} from '@/app/components/layout/navigation/side-navigation-patterns';
import { typographyClassNames } from '@/app/components/ui/primitives/text';
import { cn } from '@/utils';

export function EndpointSideNav(props: { endpointId: string }) {
  const [expanded, setExpanded] = useState(
    () =>
      getSidenavExpanded('endpoint') &&
      !window.matchMedia?.('(max-width: 767px)').matches,
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

  const toggleExpanded = () => {
    const nextExpanded = !expanded;
    setSidenavExpanded('endpoint', nextExpanded);
    setExpanded(nextExpanded);
  };

  return (
    <SideNavigationPanel expanded={expanded}>
      <SideNav
        aria-label="Endpoint actions"
        expanded={expanded}
        isFixedNav
        isRail={!expanded}
        className={cn(
          typographyClassNames['body-compact-01'],
          'app-side-nav relative! inset-auto! h-auto! flex-1 w-full! transform-none! border-none! bg-transparent! z-0!',
        )}
      >
        <SideNavItems className="pt-2">
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
            <SideNavigationSectionLabel expanded={expanded}>
              Settings
            </SideNavigationSectionLabel>
            <SideNavLink
              renderIcon={Settings}
              href={`${basePath}/settings`}
              isActive={isActive('settings')}
            >
              General
            </SideNavLink>
          </Fragment>

          <Fragment>
            <SideNavigationSectionLabel expanded={expanded}>
              Playground
            </SideNavigationSectionLabel>
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
        <SideNavigationCollapseControl
          onClick={toggleExpanded}
          aria-label={
            expanded
              ? 'Collapse endpoint navigation'
              : 'Expand endpoint navigation'
          }
          expanded={expanded}
        />
      </div>
    </SideNavigationPanel>
  );
}
