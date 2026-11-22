import {
  useState,
  useContext,
  useEffect,
  useRef,
  type HTMLAttributes,
  type MouseEvent,
} from 'react';
import { cn } from '@/utils';
import { useMatches, useNavigate, type UIMatch } from 'react-router-dom';
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
import { useRapidaStore } from '@/stores/app';
import { useTheme } from '@/theme/theme-provider';
import type { AppRouteHandle } from '@/app/routes/breadcrumb-handle';
import { OrganizationProjectSelector } from '@/app/components/ui/composites';
import type { DashboardInterfaceTour } from '@/app/components/layout/navigation/dashboard-interface-tour';
import { Coachmark } from '@/app/components/ui/composites/coachmark';

export function ActionableHeader({
  className,
  interfaceTour,
  ...attributes
}: HTMLAttributes<HTMLElement> & {
  interfaceTour?: DashboardInterfaceTour;
}) {
  const matches = useMatches() as UIMatch<unknown, AppRouteHandle>[];
  const navigate = useNavigate();
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
    if (typeof media.addEventListener === 'function') {
      media.addEventListener('change', updateVisibleItems);
      return () => media.removeEventListener('change', updateVisibleItems);
    }
    media.addListener(updateVisibleItems);
    return () => media.removeListener(updateVisibleItems);
  }, []);

  const breadcrumbMatches = matches.filter(match => match.handle?.breadcrumb);
  const breadcrumbs = breadcrumbMatches.map((match, index) => {
    const definition = match.handle.breadcrumb!;
    const label =
      'label' in definition ? (
        definition.label
      ) : (
        <span>
          <definition.render params={match.params} />
        </span>
      );
    const isCurrentPage = index === breadcrumbMatches.length - 1;

    return {
      label,
      href: match.pathname,
      isCurrentPage,
      onClick: (event: MouseEvent<HTMLElement>) => {
        if (
          event.defaultPrevented ||
          event.button !== 0 ||
          event.metaKey ||
          event.ctrlKey ||
          event.shiftKey ||
          event.altKey
        ) {
          return;
        }
        event.preventDefault();
        navigate(match.pathname);
      },
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
      <CustomerOptions
        interfaceTour={interfaceTour}
        isCompact={maxVisibleBreadcrumbItems === 2}
        isLoading={isLoading}
      />
    </header>
  );
}

export function CustomerOptions({
  isCompact = false,
  interfaceTour,
  isLoading,
  showContextSwitcher = true,
}: {
  isCompact?: boolean;
  interfaceTour?: DashboardInterfaceTour;
  isLoading?: boolean;
  showContextSwitcher?: boolean;
}) {
  const {
    getOrganizations,
    getCurrentOrganization,
    getCurrentProject,
    setCurrentContext,
  } = useContext(AuthContext);
  const organizations = getOrganizations?.() || [];
  const currentOrganization = getCurrentOrganization?.();
  const currentProject = getCurrentProject?.();
  const navigate = useNavigate();

  const [accountDropdownOpen, setAccountDropdownOpen] = useState(false);
  const contextSwitcherButtonRef = useRef<HTMLButtonElement>(null);
  const themeButtonRef = useRef<HTMLButtonElement>(null);
  const profileButtonRef = useRef<HTMLButtonElement>(null);
  const { resolvedMode, theme, toggleMode } = useTheme();
  const isDarkMode = resolvedMode === 'dark';

  return (
    <HeaderGlobalBar>
      {showContextSwitcher && setCurrentContext && (
        <Coachmark
          actionLabel="Next"
          align="bottom-end"
          description="Switch organizations or projects from this menu."
          launcherButtonRef={contextSwitcherButtonRef}
          onAction={interfaceTour?.next ?? (() => undefined)}
          onClose={interfaceTour?.dismiss ?? (() => undefined)}
          open={interfaceTour?.step === 'context'}
          targetClassName="inline-flex h-full"
          title="Switch organization or project"
        >
          <span className="inline-flex h-full">
            <OrganizationProjectSelector
              organizations={organizations.map(organization => ({
                id: organization.id,
                name: organization.name,
                projects: organization.projectsList.map(project => ({
                  id: project.id,
                  name: project.name,
                })),
              }))}
              selectedOrganizationId={currentOrganization?.id}
              selectedProjectId={currentProject?.id}
              isCompact={isCompact}
              isLoading={isLoading}
              triggerRef={contextSwitcherButtonRef}
              onChange={(organization, project) => {
                const selectedOrganization = organizations.find(
                  candidate => candidate.id === organization.id,
                );
                if (!selectedOrganization) return;
                const selectedProject = project
                  ? selectedOrganization.projectsList.find(
                      candidate => candidate.id === project.id,
                    )
                  : undefined;
                if (project && !selectedProject) return;
                if (
                  selectedOrganization.id === currentOrganization?.id &&
                  selectedProject?.id === currentProject?.id
                )
                  return;
                if (setCurrentContext(selectedOrganization, selectedProject))
                  navigate('/dashboard');
              }}
            />
          </span>
        </Coachmark>
      )}

      {theme.allowModeSelection && (
        <Coachmark
          actionLabel="Next"
          align="bottom-end"
          description="Switch between light and dark mode whenever you want."
          launcherButtonRef={themeButtonRef}
          onAction={interfaceTour?.next ?? (() => undefined)}
          onClose={interfaceTour?.dismiss ?? (() => undefined)}
          open={interfaceTour?.step === 'theme'}
          targetClassName="inline-flex h-full"
          title="Choose your appearance"
        >
          <HeaderGlobalAction
            ref={themeButtonRef}
            aria-label={`Switch to ${isDarkMode ? 'light' : 'dark'} mode`}
            onClick={toggleMode}
            tooltipAlignment="end"
          >
            {isDarkMode ? <Sun size={20} /> : <Moon size={20} />}
          </HeaderGlobalAction>
        </Coachmark>
      )}

      <Coachmark
        actionLabel="Next"
        align="bottom-end"
        description="Open account settings, resources, and session controls here."
        launcherButtonRef={profileButtonRef}
        onAction={interfaceTour?.next ?? (() => undefined)}
        onClose={interfaceTour?.dismiss ?? (() => undefined)}
        open={interfaceTour?.step === 'account'}
        targetClassName="inline-flex h-full"
        title="Open your account"
      >
        <HeaderGlobalAction
          ref={profileButtonRef}
          aria-label="Account"
          isActive={accountDropdownOpen}
          onClick={() => setAccountDropdownOpen(!accountDropdownOpen)}
          tooltipAlignment="end"
        >
          <UserAvatar size={20} />
        </HeaderGlobalAction>
      </Coachmark>

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
