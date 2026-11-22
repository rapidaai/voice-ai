import { Observability } from '@/app/components/layout/navigation/sidebar/observability';
import { Deployment } from '@/app/components/layout/navigation/sidebar/deployment';
import { Dashboard } from '@/app/components/layout/navigation/sidebar/dashboard';
import { Team } from '@/app/components/layout/navigation/sidebar/team';
import { Project } from '@/app/components/layout/navigation/sidebar/project';
import { Roles } from '@/app/components/layout/navigation/sidebar/roles';
import { Vault } from '@/app/components/layout/navigation/sidebar/vault';
import { Knowledge } from '@/app/components/layout/navigation/sidebar/knowledge';
import { Aside } from '@/app/components/layout/aside';
import { ExternalTool } from '@/app/components/layout/navigation/sidebar/external-tools';
import { BrandedLogo } from '@/app/components/layout/brand/branded-logo';
import { useSidebar } from '@/context/sidebar-context';
import { cn } from '@/utils';
import { useRapidaStore } from '@/stores/app';
import { Text, typographyClassNames } from '@/app/components/ui/primitives';
import { useWorkspace } from '@/workspace';
import { useRef } from 'react';
import type { DashboardInterfaceTour } from '@/app/components/layout/navigation/dashboard-interface-tour';
import { Coachmark } from '@/app/components/ui/composites/coachmark';
import {
  SideNavigationCollapseControl,
  SideNavigationSectionLabel,
} from '@/app/components/layout/navigation/side-navigation-patterns';

export function SidebarNavigation({
  interfaceTour,
}: {
  interfaceTour?: DashboardInterfaceTour;
}) {
  const workspace = useWorkspace();
  const { locked, setLocked, open } = useSidebar();
  const { loading, loadingType } = useRapidaStore();
  const isLoading = loading && loadingType === 'block';
  const sidebarButtonRef = useRef<HTMLButtonElement>(null);

  return (
    <Aside
      className="shrink-0 flex flex-col"
      style={
        interfaceTour?.step === 'navigation' ||
        interfaceTour?.step === 'sidebar-toggle'
          ? { overflow: 'visible' }
          : undefined
      }
    >
      <div
        className={cn(
          'h-12 flex shrink-0 items-center border-b border-border-subtle px-3',
          open ? 'justify-start' : 'justify-center',
        )}
      >
        <BrandedLogo
          variant={open ? 'full' : 'compact'}
          className={cn(
            open ? 'h-6 w-auto max-w-[12rem] object-left' : 'h-6 w-6',
          )}
          textClassName={cn(
            'transition-all duration-200',
            open
              ? 'opacity-100'
              : [typographyClassNames['label-01'], 'w-6 text-center'],
          )}
        />
      </div>

      <div className="relative min-h-0 flex-1">
        <nav className="h-full overflow-y-auto no-scrollbar py-2">
          <ul>
            <Dashboard isLoading={isLoading} />
            <Deployment isLoading={isLoading} />
            {workspace.features?.knowledge !== false && (
              <Knowledge isLoading={isLoading} />
            )}
          </ul>

          <div>
            <SideNavigationSectionLabel as="div" expanded={open}>
              <Text isLoading={isLoading} skeletonWidth="60%">
                Observability
              </Text>
            </SideNavigationSectionLabel>
            <ul>
              <Observability isLoading={isLoading} />
            </ul>
          </div>

          <div>
            <SideNavigationSectionLabel as="div" expanded={open}>
              <Text isLoading={isLoading} skeletonWidth="60%">
                Integrations
              </Text>
            </SideNavigationSectionLabel>
            <ul>
              <ExternalTool isLoading={isLoading} />
              <Vault isLoading={isLoading} />
            </ul>
          </div>

          <div>
            <SideNavigationSectionLabel as="div" expanded={open}>
              <Text isLoading={isLoading} skeletonWidth="60%">
                Organizations
              </Text>
            </SideNavigationSectionLabel>
            <ul>
              <Team isLoading={isLoading} />
              <Project isLoading={isLoading} />
              <Roles isLoading={isLoading} />
            </ul>
          </div>
        </nav>

        {interfaceTour?.step === 'navigation' && (
          <div className="pointer-events-none absolute inset-x-0 top-0 h-12">
            <Coachmark
              actionLabel="Next"
              align="right-start"
              description="Use these menu items to move between dashboard, deployments, observability, integrations, and organization settings."
              launcherButtonRef={sidebarButtonRef}
              onAction={interfaceTour.next}
              onClose={interfaceTour.dismiss}
              open
              targetClassName="h-full w-full [&>.cds--popover-container]:h-full [&>.cds--popover-container]:w-full"
              title="Explore the side navigation"
            >
              <span
                aria-hidden
                className="block h-full w-full"
                data-coachmark-target="navigation"
              />
            </Coachmark>
          </div>
        )}
      </div>

      <div className="shrink-0 border-t border-border-subtle">
        <div className="relative">
          <SideNavigationCollapseControl
            ref={sidebarButtonRef}
            onClick={() => setLocked(!locked)}
            aria-label={locked ? 'Collapse sidebar' : 'Expand sidebar'}
            expanded={locked}
            showLabel={open}
          />
          <div className="pointer-events-none absolute inset-y-0 left-px w-12">
            <Coachmark
              actionLabel="Done"
              align="top-start"
              description="Collapse the navigation for more workspace, then expand it whenever you need it."
              launcherButtonRef={sidebarButtonRef}
              onAction={interfaceTour?.next ?? (() => undefined)}
              onClose={interfaceTour?.dismiss ?? (() => undefined)}
              open={interfaceTour?.step === 'sidebar-toggle'}
              targetClassName="h-full w-full"
              title="Collapse or expand the side navigation"
            >
              <span
                aria-hidden
                className="block h-8 w-12"
                data-coachmark-target="sidebar-toggle"
              />
            </Coachmark>
          </div>
        </div>
      </div>
    </Aside>
  );
}
