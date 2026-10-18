import React from 'react';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { ViewEndpointPage } from '@/app/pages/endpoint/view';
import { EndpointViewLayout } from '@/app/pages/endpoint/view/endpoint-view.layout';
import { useEndpointPageStore } from '@/stores/endpoint';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockOnGetEndpoint = jest.fn();

jest.mock('@carbon/icons-react', () => ({
  Activity: ({ size }: any) => (
    <span aria-hidden="true" data-icon-size={size}>
      activity
    </span>
  ),
  Application: ({ size }: any) => (
    <span data-icon-size={size}>application</span>
  ),
  Checkmark: ({ size }: any) => <span data-icon-size={size}>checkmark</span>,
  Code: ({ size }: any) => <span data-icon-size={size}>code</span>,
  Copy: ({ size }: any) => <span data-icon-size={size}>copy</span>,
  Dashboard: ({ size }: any) => (
    <span aria-hidden="true" data-icon-size={size}>
      dashboard
    </span>
  ),
  Debug: ({ size }: any) => (
    <span aria-hidden="true" data-icon-size={size}>
      debug
    </span>
  ),
  Edit: ({ size }: any) => <span data-icon-size={size}>edit</span>,
  Globe: ({ size }: any) => <span data-icon-size={size}>globe</span>,
  Information: ({ size }: any) => <span data-icon-size={size}>info</span>,
  LogoPython: ({ size }: any) => <span data-icon-size={size}>python</span>,
  LogoReact: ({ size }: any) => <span data-icon-size={size}>react</span>,
  Phone: ({ size }: any) => <span data-icon-size={size}>phone</span>,
  Play: ({ size }: any) => (
    <span aria-hidden="true" data-icon-size={size}>
      play
    </span>
  ),
  SourceControl: ({ size }: any) => (
    <span aria-hidden="true" data-icon-size={size}>
      source-control
    </span>
  ),
  Tag: ({ size }: any) => <span data-icon-size={size}>tag</span>,
  SidePanelClose: () => <span>close panel</span>,
  SidePanelOpen: () => <span>open panel</span>,
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@carbon/react', () => ({
  Breadcrumb: ({ children }: any) => <nav>{children}</nav>,
  BreadcrumbItem: ({ children, href, isCurrentPage }: any) =>
    isCurrentPage ? <span>{children}</span> : <a href={href}>{children}</a>,
  HeaderGlobalBar: ({ children, ...props }: any) => (
    <div role="toolbar" {...props}>
      {children}
    </div>
  ),
  HeaderGlobalAction: ({ children, tooltipAlignment, ...props }: any) => (
    <button data-tooltip-alignment={tooltipAlignment} {...props}>
      {children}
    </button>
  ),
  Button: ({
    children,
    className,
    kind,
    renderIcon: Icon,
    size,
    ...props
  }: any) => (
    <button className={className} data-kind={kind} data-size={size} {...props}>
      {Icon ? <Icon size={16} /> : null}
      {children}
    </button>
  ),
  Tabs: ({ children, selectedIndex }: any) => (
    <div data-testid="endpoint-tabs" data-selected-index={selectedIndex}>
      {children}
    </div>
  ),
  TabList: ({ children, 'aria-label': ariaLabel, fullWidth }: any) => (
    <div
      role="tablist"
      aria-label={ariaLabel}
      data-full-width={String(Boolean(fullWidth))}
    >
      {children}
    </div>
  ),
  Tab: ({ children }: any) => <button role="tab">{children}</button>,
  TabPanels: ({ children }: any) => <div>{children}</div>,
  TabPanel: ({ children }: any) => <div role="tabpanel">{children}</div>,
  SideNav: ({ children, expanded, isRail, className, ...props }: any) => (
    <aside
      className={className}
      data-expanded={expanded}
      data-rail={isRail}
      {...props}
    >
      {children}
    </aside>
  ),
  SideNavItems: ({ children }: any) => <ul>{children}</ul>,
  SideNavLink: ({ children, isActive, href, renderIcon: Icon }: any) => (
    <li>
      <a
        className={isActive ? 'cds--side-nav__link--current' : undefined}
        data-active={isActive}
        href={href}
      >
        {Icon ? <Icon size={16} /> : null}
        {children}
      </a>
    </li>
  ),
  SideNavMenu: ({
    children,
    title,
    isActive,
    defaultExpanded,
    renderIcon: Icon,
  }: any) => (
    <li data-active={isActive} data-expanded={defaultExpanded}>
      {Icon ? <Icon size={16} /> : null}
      <span>{title}</span>
      {children}
    </li>
  ),
  SideNavMenuItem: ({ children, isActive, href }: any) => (
    <a
      className={isActive ? 'cds--side-nav__link--current' : undefined}
      data-active={isActive}
      href={href}
    >
      {children}
    </a>
  ),
  SkeletonText: () => <span>Loading nav item</span>,
}));

jest.mock('@/app/pages/endpoint/view/try-playground', () => ({
  Playground: () => <section>Endpoint playground</section>,
}));

jest.mock('@/app/pages/endpoint/view/pages/overview-page', () => ({
  EndpointOverviewPage: () => <section>Endpoint summary</section>,
}));

jest.mock('@/app/pages/endpoint/view/traces', () => ({
  EndpointTraces: () => <section>Endpoint logs</section>,
}));

jest.mock('@/app/pages/endpoint/view/version-list', () => ({
  Version: () => <section>Endpoint versions</section>,
}));

jest.mock('@/app/pages/endpoint/view/pages/settings-page', () => ({
  EndpointSettingsPage: () => <section>Endpoint settings</section>,
}));

jest.mock(
  '@/app/components/dialogs/endpoint/endpoint-instruction-modal',
  () => ({
    EndpointInstructionDialog: () => null,
  }),
);

jest.mock('@/app/components/dialogs/shared/create-tag-modal', () => ({
  CreateTagDialog: () => null,
}));

jest.mock('@/app/components/dialogs/shared/update-description-modal', () => ({
  UpdateDescriptionDialog: () => null,
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
}));

const makeTimestamp = (date: Date) => ({
  getSeconds: () => Math.floor(date.getTime() / 1000),
  getNanos: () => 0,
  toDate: () => date,
});

const makeEndpoint = () =>
  ({
    getId: () => 'endpoint-1',
    getName: () => 'Production endpoint',
    getDescription: () => 'Endpoint description',
    getEndpointtag: () => ({ getTagList: () => [] }),
    getEndpointprovidermodel: () => ({
      getId: () => 'epm-1',
      getCreateddate: () => makeTimestamp(new Date()),
    }),
  }) as any;

const renderEndpointDetailRoute = (
  initialEntry = '/deployment/endpoint/endpoint-1/overview',
) =>
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route
          path="/deployment/endpoint/:endpointId"
          element={<EndpointViewLayout />}
        >
          <Route
            path="create-endpoint-version"
            element={<section>Create endpoint version</section>}
          />
          <Route path=":tab" element={<ViewEndpointPage />} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );

describe('Endpoint detail layout', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useEndpointPageStore.setState({
      currentEndpoint: makeEndpoint(),
      currentEndpointProviderModel: {
        getId: () => 'epm-1',
      },
      instructionVisible: false,
      editTagVisible: false,
      updateDetailVisible: false,
      onGetEndpoint: mockOnGetEndpoint,
    } as any);
    mockOnGetEndpoint.mockImplementation(
      (
        _endpointId,
        _endpointProviderId,
        _projectId,
        _token,
        _userId,
        _onError,
        onSuccess,
      ) => {
        onSuccess(makeEndpoint());
      },
    );
  });

  it('uses the same collapsible Carbon side navigation as assistant details', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const overviewRender = renderEndpointDetailRoute(
      '/deployment/endpoint/endpoint-1/overview',
    );
    expect(screen.getByText('Endpoint summary')).toBeInTheDocument();
    const endpointNav = screen.getByRole('complementary', {
      name: 'Endpoint actions',
    });
    expect(
      Array.from(within(endpointNav).getByRole('list').children).every(
        child => child.tagName === 'LI',
      ),
    ).toBe(true);
    expect(endpointNav).toHaveAttribute('data-expanded', 'true');
    expect(
      within(endpointNav).getByRole('link', { name: 'Overview' }),
    ).toHaveClass('cds--side-nav__link--current');
    expect(within(endpointNav).getByText('Versions')).toBeInTheDocument();
    expect(
      within(endpointNav).getByRole('link', { name: 'View all' }),
    ).toHaveAttribute('href', '/deployment/endpoint/endpoint-1/versions');
    expect(
      within(endpointNav).getByRole('link', { name: 'Add new version' }),
    ).toHaveAttribute(
      'href',
      '/deployment/endpoint/endpoint-1/create-endpoint-version',
    );
    expect(
      within(endpointNav).getByRole('link', { name: 'Logs' }),
    ).toBeInTheDocument();
    expect(within(endpointNav).getByText('Settings')).toBeInTheDocument();
    expect(
      within(endpointNav).getByRole('link', { name: 'General' }),
    ).toHaveAttribute('href', '/deployment/endpoint/endpoint-1/settings');
    expect(within(endpointNav).getByText('Playground')).toBeInTheDocument();
    expect(
      within(endpointNav).getByRole('link', { name: 'Open playground' }),
    ).toHaveAttribute('href', '/deployment/endpoint/endpoint-1/playground');

    await user.click(screen.getByRole('button', { name: 'Collapse nav' }));
    expect(endpointNav).toHaveAttribute('data-expanded', 'false');
    expect(
      screen.getByRole('button', { name: 'Expand nav' }),
    ).toBeInTheDocument();

    overviewRender.unmount();
    renderEndpointDetailRoute('/deployment/endpoint/endpoint-1/logs');

    expect(screen.getByText('Endpoint logs')).toBeInTheDocument();
    expect(screen.queryByText('Endpoint playground')).not.toBeInTheDocument();
  });

  it('keeps endpoint navigation visible and marks versions on the create version URL', () => {
    renderEndpointDetailRoute(
      '/deployment/endpoint/endpoint-1/create-endpoint-version',
    );

    const endpointNav = screen.getByRole('complementary', {
      name: 'Endpoint actions',
    });
    expect(screen.getByText('Create endpoint version')).toBeInTheDocument();
    expect(
      within(endpointNav).getByRole('link', { name: 'Add new version' }),
    ).toHaveClass('cds--side-nav__link--current');
  });
});
