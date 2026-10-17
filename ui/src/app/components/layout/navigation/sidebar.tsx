import { Observability } from '@/app/components/layout/navigation/sidebar/observability';
import { Deployment } from '@/app/components/layout/navigation/sidebar/deployment';
import { Dashboard } from '@/app/components/layout/navigation/sidebar/dashboard';
import { Team } from '@/app/components/layout/navigation/sidebar/team';
import { Project } from '@/app/components/layout/navigation/sidebar/project';
import { Vault } from '@/app/components/layout/navigation/sidebar/vault';
import { Knowledge } from '@/app/components/layout/navigation/sidebar/knowledge';
import { Aside } from '@/app/components/layout/aside';
import { ExternalTool } from '@/app/components/layout/navigation/sidebar/external-tools';
import { BrandedLogo } from '@/app/components/layout/brand/branded-logo';
import { SidePanelClose, SidePanelOpen } from '@carbon/icons-react';
import { Button } from '@carbon/react';
import { useSidebar } from '@/context/sidebar-context';
import { cn } from '@/utils';
import { useRapidaStore } from '@/stores/app';
import { Text } from '@/app/components/ui/primitives';
import { useWorkspace } from '@/workspace';

export function SidebarNavigation() {
  const workspace = useWorkspace();
  const { locked, setLocked, open } = useSidebar();
  const { loading, loadingType } = useRapidaStore();
  const isLoading = loading && loadingType === 'block';

  return (
    <Aside className="shrink-0 flex flex-col">
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
            open ? 'text-base opacity-100' : 'w-6 text-center text-xs',
          )}
        />
      </div>

      <nav className="flex-1 overflow-y-auto no-scrollbar py-2">
        <ul>
          <Dashboard isLoading={isLoading} />
          <Deployment isLoading={isLoading} />
          {workspace.features?.knowledge !== false && (
            <Knowledge isLoading={isLoading} />
          )}
        </ul>

        <div className="mt-2">
          <div
            className={cn(
              'flex items-center px-4 py-2 border-b border-border-subtle',
              'text-[10px] font-medium capitalize tracking-[0.1em]',
              'text-muted',
              'transition-all duration-200',
              open
                ? 'opacity-100'
                : 'opacity-0 h-0 py-0 overflow-hidden border-none',
            )}
          >
            <Text
              isLoading={isLoading}
              skeletonWidth="60%"
              className="uppercase"
            >
              Observability
            </Text>
          </div>
          <ul>
            <Observability isLoading={isLoading} />
          </ul>
        </div>

        <div className="mt-2">
          <div
            className={cn(
              'flex items-center px-4 py-2 border-b border-border-subtle',
              'text-[10px] font-medium capitalize tracking-[0.1em]',
              'text-muted',
              'transition-all duration-200',
              open
                ? 'opacity-100'
                : 'opacity-0 h-0 py-0 overflow-hidden border-none',
            )}
          >
            <Text
              isLoading={isLoading}
              skeletonWidth="60%"
              className="uppercase"
            >
              Integrations
            </Text>
          </div>
          <ul>
            <ExternalTool isLoading={isLoading} />
            <Vault isLoading={isLoading} />
          </ul>
        </div>

        <div className="mt-2">
          <div
            className={cn(
              'flex items-center px-4 py-2 border-b border-border-subtle',
              'text-[10px] font-medium capitalize tracking-[0.1em]',
              'text-muted',
              'transition-all duration-200',
              open
                ? 'opacity-100'
                : 'opacity-0 h-0 py-0 overflow-hidden border-none',
            )}
          >
            <Text
              isLoading={isLoading}
              skeletonWidth="60%"
              className="uppercase"
            >
              Organizations
            </Text>
          </div>
          <ul>
            <Team isLoading={isLoading} />
            <Project isLoading={isLoading} />
          </ul>
        </div>
      </nav>

      <div className="shrink-0 border-t border-border-subtle">
        <Button
          type="button"
          kind="ghost"
          size="sm"
          onClick={() => setLocked(!locked)}
          aria-label={locked ? 'Collapse sidebar' : 'Expand sidebar'}
          className={cn(
            '!flex !h-10 !w-full !max-w-none !items-center !justify-start !px-4',
            '!text-muted',
            'hover:!bg-layer-hover hover:!text-foreground',
            '!transition-colors !duration-100',
          )}
        >
          <span className="shrink-0">
            {locked ? (
              <SidePanelClose size={16} />
            ) : (
              <SidePanelOpen size={16} />
            )}
          </span>
          <span
            className={cn(
              'text-xs truncate transition-all duration-200 ml-3',
              open ? 'opacity-100' : 'opacity-0 w-0 ml-0 overflow-hidden',
            )}
          >
            {locked ? 'Collapse' : 'Expand'}
          </span>
        </Button>
      </div>
    </Aside>
  );
}
