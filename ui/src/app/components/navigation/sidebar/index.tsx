import { Observability } from '@/app/components/navigation/sidebar/observability';
import { Deployment } from '@/app/components/navigation/sidebar/deployment';
import { Dashboard } from '@/app/components/navigation/sidebar/dashboard';
import { Team } from '@/app/components/navigation/sidebar/team';
import { Project } from '@/app/components/navigation/sidebar/project';
import { Vault } from '@/app/components/navigation/sidebar/vault';
import { Knowledge } from '@/app/components/navigation/sidebar/knowledge';
import { Aside } from '@/app/components/aside';
import { ExternalTool } from '@/app/components/navigation/sidebar/external-tools';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { RapidaTextIcon } from '@/app/components/Icon/RapidaText';
import { SidePanelClose, SidePanelOpen } from '@carbon/icons-react';
import { useSidebar } from '@/context/sidebar-context';
import { cn } from '@/utils/index';
import { useRapidaStore } from '@/hooks';
import { Text } from '@/app/components/carbon/text';
import { useTheme } from '@/theme/theme-provider';
import { useWorkspace } from '@/workspace';

/**
 * Carbon UI Shell — Side Navigation
 * Spec: h-8 nav items, 4px left accent on active, 48px logo header,
 *       label-01 group headers, lock/collapse button in footer.
 */
export function SidebarNavigation(props: {}) {
  const workspace = useWorkspace();
  const { resolvedMode, theme } = useTheme();
  const { locked, setLocked, open } = useSidebar();
  const { loading, loadingType } = useRapidaStore();
  const isLoading = loading && loadingType === 'block';

  return (
    <Aside className="relative shrink-0 flex flex-col">
      {/* ── Logo row — Carbon UI Shell header: h-12, border-b ── */}
      <div
        className={cn(
          'h-12 flex shrink-0 items-center border-b border-border-subtle px-3',
          open ? 'justify-start' : 'justify-center',
        )}
      >
        {theme.brand.logos ? (
          <img
            src={
              open
                ? theme.brand.logos.full[resolvedMode]
                : theme.brand.logos.compact[resolvedMode]
            }
            alt={theme.brand.name}
            className={cn(
              'block object-contain',
              open ? 'h-6 w-auto max-w-[12rem] object-left' : 'h-6 w-6',
            )}
          />
        ) : (
          <div className="flex items-center gap-2 text-primary">
            <RapidaIcon className="h-6 w-6 shrink-0" />
            <RapidaTextIcon
              className={cn(
                'h-[18px] transition-all duration-200',
                open ? 'opacity-100' : 'opacity-0 w-0',
              )}
            />
          </div>
        )}
      </div>

      {/* ── Nav groups — scrollable ── */}
      <nav className="flex-1 overflow-y-auto no-scrollbar py-2">
        {/* Group 1 — primary nav */}
        <ul>
          <Dashboard isLoading={isLoading} />
          <Deployment isLoading={isLoading} />
          {workspace.features?.knowledge !== false && (
            <Knowledge isLoading={isLoading} />
          )}
        </ul>

        {/* Group 2 — Observability */}
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

        {/* Group 3 — Integrations */}
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

        {/* Group 4 — Organizations */}
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

      {/* ── Footer — collapse/expand button ── */}
      <div className="shrink-0 border-t border-border-subtle">
        <button
          type="button"
          onClick={() => setLocked(!locked)}
          aria-label={locked ? 'Collapse sidebar' : 'Expand sidebar'}
          className={cn(
            'flex items-center h-10 w-full cursor-pointer px-4',
            'text-muted',
            'hover:bg-layer-hover hover:text-foreground',
            'transition-colors duration-100',
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
        </button>
      </div>
    </Aside>
  );
}
