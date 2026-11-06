import { useState, useContext, useEffect, type HTMLAttributes } from 'react';
import { ProjectRole } from '@rapidaai/react';
import { cn } from '@/utils';
import { useMatches, type UIMatch } from 'react-router-dom';
import { CustomLink } from '@/app/components/ui/primitives';
import { AuthContext } from '@/context/auth-context';
import { Moon, Sun, UserAvatar } from '@carbon/icons-react';
import {
  HeaderGlobalAction,
  HeaderGlobalBar,
  HeaderPanel,
  Switcher,
  SwitcherItem,
} from '@carbon/react';
import { Breadcrumb } from '@/app/components/ui/primitives';
import { Dropdown } from '@/app/components/ui/primitives';
import { useRapidaStore } from '@/stores/app';
import { useTheme } from '@/theme/theme-provider';
import type { AppRouteHandle } from '@/app/routes/breadcrumb-handle';

export function ActionableHeader({
  className,
  ...attributes
}: HTMLAttributes<HTMLElement>) {
  const matches = useMatches() as UIMatch<unknown, AppRouteHandle>[];
  const { loading, loadingType } = useRapidaStore();
  const isLoading = loading && loadingType === 'block';
  const [maxVisibleBreadcrumbItems, setMaxVisibleBreadcrumbItems] = useState(
    () => (window.matchMedia?.('(max-width: 640px)').matches === true ? 2 : 3),
  );

  useEffect(() => {
    const media = window.matchMedia?.('(max-width: 640px)');
    if (!media) return;

    const updateVisibleItems = () =>
      setMaxVisibleBreadcrumbItems(media.matches ? 2 : 3);
    media.addEventListener('change', updateVisibleItems);
    return () => media.removeEventListener('change', updateVisibleItems);
  }, []);

  const breadcrumbMatches = matches.filter(match => match.handle?.breadcrumb);
  const breadcrumbs = breadcrumbMatches.map((match, index) => {
    const definition = match.handle.breadcrumb!;
    const label =
      'label' in definition ? (
        definition.label
      ) : (
        <definition.render params={match.params} />
      );
    const isCurrentPage = index === breadcrumbMatches.length - 1;

    return {
      label,
      href: match.pathname,
      isCurrentPage,
      render: isCurrentPage
        ? undefined
        : () => <CustomLink to={match.pathname}>{label}</CustomLink>,
    };
  });

  return (
    <header
      {...attributes}
      className={cn(
        'h-12 flex items-center justify-between',
        'bg-shell text-foreground',
        'border-b border-border-subtle',
        'shrink-0',
        className,
      )}
    >
      <Breadcrumb
        isLoading={isLoading}
        maxVisibleItems={maxVisibleBreadcrumbItems}
        className="global-breadcrumb min-w-0 overflow-hidden pl-4"
        items={breadcrumbs}
      />
      <CustomerOptions isLoading={isLoading} />
    </header>
  );
}

export function CustomerOptions({
  isLoading,
  showProjectSelector = true,
}: {
  isLoading?: boolean;
  showProjectSelector?: boolean;
}) {
  const { projectRoles, currentProjectRole, setCurrentProjectRole } =
    useContext(AuthContext);

  const [accountDropdownOpen, setAccountDropdownOpen] = useState(false);
  const { resolvedMode, theme, toggleMode } = useTheme();
  const isDarkMode = resolvedMode === 'dark';

  return (
    <HeaderGlobalBar>
      {showProjectSelector && projectRoles && setCurrentProjectRole && (
        <Dropdown
          id="project-selector"
          aria-label="Select project"
          titleText=""
          hideLabel
          label="Select a Project"
          size="sm"
          direction="bottom"
          items={projectRoles}
          selectedItem={currentProjectRole}
          itemToString={(item: ProjectRole.AsObject | null) =>
            item?.projectname || ''
          }
          onChange={({ selectedItem }) => {
            if (selectedItem) setCurrentProjectRole(selectedItem);
          }}
          className="project-selector-dropdown max-sm:w-12! max-sm:min-w-12!"
          isLoading={isLoading}
        />
      )}

      {theme.allowModeSelection && (
        <HeaderGlobalAction
          aria-label={`Switch to ${isDarkMode ? 'light' : 'dark'} mode`}
          onClick={toggleMode}
          tooltipAlignment="end"
        >
          {isDarkMode ? <Sun size={20} /> : <Moon size={20} />}
        </HeaderGlobalAction>
      )}

      <HeaderGlobalAction
        aria-label="Account"
        isActive={accountDropdownOpen}
        onClick={() => setAccountDropdownOpen(!accountDropdownOpen)}
        tooltipAlignment="end"
      >
        <UserAvatar size={20} />
      </HeaderGlobalAction>

      <HeaderPanel expanded={accountDropdownOpen}>
        <Switcher
          aria-label={`${theme.brand.name} account`}
          expanded={accountDropdownOpen}
        >
          <li className="cds--switcher__item--divider">
            <span className="uppercase!">Account</span>
          </li>
          <SwitcherItem aria-label="Account Settings" href="/account">
            Account Settings
          </SwitcherItem>
          <li className="cds--switcher__item--divider">
            <span className="uppercase!">Resources</span>
          </li>
          <SwitcherItem aria-label="Support" href={theme.links.support}>
            Support
          </SwitcherItem>
          <SwitcherItem aria-label="Terms" href={theme.links.terms}>
            Terms
          </SwitcherItem>
          <SwitcherItem aria-label="Privacy" href={theme.links.privacy}>
            Privacy
          </SwitcherItem>
          <li className="cds--switcher__item--divider">
            <span className="uppercase!">Session</span>
          </li>
          <SwitcherItem aria-label="Sign out" href="/auth/signin">
            Sign out
          </SwitcherItem>
        </Switcher>
      </HeaderPanel>
    </HeaderGlobalBar>
  );
}
