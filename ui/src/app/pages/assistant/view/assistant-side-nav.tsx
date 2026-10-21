import { FC } from 'react';
import { useLocation } from 'react-router-dom';
import { cn } from '@/utils';
import { SidePanelOpen, SidePanelClose } from '@carbon/icons-react';
import {
  SideNav,
  SideNavItems,
  SideNavLink,
  SideNavMenu,
  SideNavMenuItem,
  SkeletonText,
} from '@carbon/react';
import {
  assistantNavSections,
  AssistantNavSection,
  AssistantNavItem,
} from './assistant-nav-config';
import { Assistant } from '@rapidaai/react';

const getVisibleNavItems = (
  section: AssistantNavSection,
  assistant: Assistant | null,
  isLoading?: boolean,
): AssistantNavItem[] => {
  if (isLoading) return section.items;
  if (!assistant) return [];

  return section.items.filter(item => item.visible?.(assistant) ?? true);
};

// ─── Skeleton placeholder for a single nav item ─────────────────────────────

const NavItemSkeleton: FC<{ itemKey: string }> = ({ itemKey }) => (
  <li key={itemKey} className="flex items-center h-8 px-4 py-2">
    <SkeletonText className="mb-0! flex-1" width="70%" />
  </li>
);

// ─── Single nav item (link or menu with children) ───────────────────────────

const NavItem: FC<{
  item: AssistantNavItem;
  basePath: string;
  isPathActive: (path: string, exact?: boolean) => boolean;
  actions: Record<string, () => void>;
  isLoading?: boolean;
}> = ({ item, basePath, isPathActive, actions, isLoading }) => {
  if (isLoading) return <NavItemSkeleton itemKey={item.key} />;

  if (item.children && item.children.length > 0) {
    const isAnyChildActive = item.children.some(c => isPathActive(c.path));
    return (
      <SideNavMenu
        key={item.key}
        title={item.label}
        renderIcon={item.icon}
        isActive={isAnyChildActive}
        defaultExpanded={isAnyChildActive}
      >
        {item.children.map(child => {
          const childAction = child.action ? actions[child.action] : undefined;
          return (
            <SideNavMenuItem
              key={child.key}
              href={childAction ? undefined : `${basePath}/${child.path}`}
              onClick={childAction ? () => childAction() : undefined}
              isActive={isPathActive(child.path, true)}
            >
              {child.label}
            </SideNavMenuItem>
          );
        })}
      </SideNavMenu>
    );
  }

  const active = isPathActive(item.path, item.exact);
  const actionFn = item.action ? actions[item.action] : undefined;

  return (
    <SideNavLink
      key={item.key}
      renderIcon={item.icon}
      href={actionFn ? undefined : `${basePath}/${item.path}`}
      onClick={actionFn ? () => actionFn() : undefined}
      isActive={active}
    >
      {item.label}
    </SideNavLink>
  );
};

// ─── Section (label + items) ─────────────────────────────────────────────────

const NavSection: FC<{
  section: AssistantNavSection;
  assistant: Assistant | null;
  basePath: string;
  expanded: boolean;
  isPathActive: (path: string, exact?: boolean) => boolean;
  actions: Record<string, () => void>;
  isLoading?: boolean;
}> = ({
  section,
  assistant,
  basePath,
  expanded,
  isPathActive,
  actions,
  isLoading,
}) => {
  const visibleItems = getVisibleNavItems(section, assistant, isLoading);

  if (visibleItems.length === 0) return null;

  return (
    <>
      {section.label && (
        <li
          className={cn(
            'cds--switcher__item--divider transition-all duration-200',
            !expanded &&
              'opacity-0 h-0 overflow-hidden !py-0 !my-0 !border-none',
          )}
        >
          {isLoading ? (
            <SkeletonText className="!mb-0" width="50%" />
          ) : (
            <span className="uppercase!">{section.label}</span>
          )}
        </li>
      )}
      {visibleItems.map(item => (
        <NavItem
          key={item.key}
          item={item}
          basePath={basePath}
          isPathActive={isPathActive}
          actions={actions}
          isLoading={isLoading}
        />
      ))}
    </>
  );
};

// ─── Main component ──────────────────────────────────────────────────────────

interface AssistantSideNavProps {
  assistantId: string;
  assistant: Assistant | null;
  expanded: boolean;
  onToggle: () => void;
  actions?: Record<string, () => void>;
}

export const AssistantSideNav: FC<AssistantSideNavProps> = ({
  assistantId,
  assistant,
  expanded,
  onToggle,
  actions = {},
}) => {
  const { pathname } = useLocation();
  const basePath = `/deployment/assistant/${assistantId}`;

  const isPathActive = (path: string, exact?: boolean) => {
    const fullPath = `${basePath}/${path}`;
    return exact ? pathname === fullPath : pathname.startsWith(fullPath);
  };

  return (
    <div
      className={cn(
        'relative shrink-0 flex flex-col h-full',
        'bg-white dark:bg-gray-900',
        'border-r border-gray-200 dark:border-gray-800',
        'transition-all duration-200',
        expanded ? 'w-56' : 'w-12',
      )}
    >
      <SideNav
        aria-label="Assistant actions"
        expanded={expanded}
        isFixedNav
        isRail={!expanded}
        className="relative! inset-auto! h-auto! flex-1 w-full! border-none! z-0!"
      >
        <SideNavItems>
          {assistantNavSections.map((section, idx) => (
            <NavSection
              key={idx}
              section={section}
              assistant={assistant}
              basePath={basePath}
              expanded={expanded}
              isPathActive={isPathActive}
              actions={actions}
              isLoading={!assistant}
            />
          ))}
        </SideNavItems>
      </SideNav>

      <div className="shrink-0 border-t border-gray-200 dark:border-gray-800">
        <button
          type="button"
          onClick={onToggle}
          className={cn(
            'flex items-center h-10 w-full cursor-pointer px-4',
            'text-gray-700 dark:text-gray-300',
            'hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-700 dark:hover:text-gray-100',
            'transition-colors duration-100',
          )}
          aria-label={expanded ? 'Collapse nav' : 'Expand nav'}
        >
          <span className="shrink-0">
            {expanded ? (
              <SidePanelClose size={16} />
            ) : (
              <SidePanelOpen size={16} />
            )}
          </span>
          {expanded && <span className="text-xs truncate ml-3">Collapse</span>}
        </button>
      </div>
    </div>
  );
};
