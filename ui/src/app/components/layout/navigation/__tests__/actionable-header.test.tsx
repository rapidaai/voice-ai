import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import {
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from 'react-router-dom';

import {
  ActionableHeader,
  CustomerOptions,
} from '@/app/components/layout/navigation/actionable-header';
import { AuthContext } from '@/context/auth-context';

const mockToggleMode = jest.fn();

const mockTheme = {
  allowModeSelection: true,
  brand: {
    name: 'Acme Voice',
    logos: {
      full: {
        light: '/brand/full-light.svg',
        dark: '/brand/full-dark.svg',
      },
      compact: {
        light: '/brand/compact-light.svg',
        dark: '/brand/compact-dark.svg',
      },
    },
  },
  links: {
    documentation: 'https://example.com/docs',
    source: 'https://example.com/source',
    support: 'https://example.com/support',
    terms: 'https://example.com/terms',
    privacy: 'https://example.com/privacy',
  },
};

let mockResolvedMode: 'light' | 'dark' = 'light';

let mockRapidaLoading = false;
let mockRapidaLoadingType: string | undefined;

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({
    resolvedMode: mockResolvedMode,
    toggleMode: mockToggleMode,
    theme: mockTheme,
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: mockRapidaLoading,
    loadingType: mockRapidaLoadingType,
  }),
}));

jest.mock('@/app/components/ui/primitives/custom-link', () => ({
  CustomLink: ({ to, children }: any) => <a href={to}>{children}</a>,
}));

jest.mock('@carbon/icons-react', () => ({
  Moon: () => <span>moon</span>,
  Sun: () => <span>sun</span>,
  UserAvatar: () => <span>avatar</span>,
}));

jest.mock('@carbon/react', () => ({
  Breadcrumb: ({ children }: any) => <ol>{children}</ol>,
  BreadcrumbItem: ({
    children,
    className,
    'aria-current': ariaCurrent,
  }: any) => (
    <li aria-current={ariaCurrent} className={className}>
      {children}
    </li>
  ),
  BreadcrumbSkeleton: ({ className }: any) => (
    <div className={className} data-testid="breadcrumb-skeleton" />
  ),
  OverflowMenu: ({ children, iconDescription }: any) => (
    <div>
      <button type="button" aria-label={iconDescription} />
      {children}
    </div>
  ),
  OverflowMenuItem: ({ href, itemText }: any) => <a href={href}>{itemText}</a>,
  HeaderGlobalBar: ({ children }: any) => <div>{children}</div>,
  HeaderGlobalAction: ({
    children,
    tooltipAlignment,
    isActive,
    ...props
  }: any) => <button {...props}>{children}</button>,
  HeaderPanel: ({ children, expanded }: any) =>
    expanded ? <div>{children}</div> : null,
  Switcher: ({ children }: any) => <ul>{children}</ul>,
  SwitcherItem: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
  DropdownSkeleton: ({ hideLabel, size }: any) => (
    <div
      data-hide-label={String(Boolean(hideLabel))}
      data-size={size}
      data-testid="dropdown-skeleton"
    />
  ),
  Dropdown: ({
    'aria-label': ariaLabel,
    id,
    label,
    items,
    selectedItem,
    itemToString,
    onChange,
    isLoading,
    className,
  }: any) => {
    const selectedIndex = Math.max(
      items.findIndex((x: any) => x === selectedItem),
      0,
    );
    return (
      <select
        id={id}
        aria-label={ariaLabel ?? label}
        data-testid={id}
        data-loading={String(Boolean(isLoading))}
        className={className}
        value={String(selectedIndex)}
        onChange={e => {
          const idx = Number(e.target.value);
          onChange({ selectedItem: items[idx] ?? null });
        }}
      >
        <option value="-1">{itemToString(null) || 'No project'}</option>
        {items.map((item: any, idx: number) => (
          <option key={item.id || idx} value={String(idx)}>
            {itemToString(item)}
          </option>
        ))}
      </select>
    );
  },
}));

const projectRoles = [
  { id: 'r1', projectid: 'p1', projectname: 'Alpha' },
  { id: 'r2', projectid: 'p2', projectname: 'Beta' },
] as any;

function renderHeader(
  routes: RouteObject[],
  initialEntry: string,
  value: React.ContextType<typeof AuthContext> = {},
) {
  const router = createMemoryRouter(routes, { initialEntries: [initialEntry] });
  return render(
    <AuthContext.Provider value={value}>
      <RouterProvider router={router} />
    </AuthContext.Provider>,
  );
}

const breadcrumb = (label: string) => ({ breadcrumb: { label } });

describe('Actionable header project switcher', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockResolvedMode = 'light';
    mockTheme.allowModeSelection = true;
    mockRapidaLoading = false;
    mockRapidaLoadingType = undefined;
  });

  it('renders ancestor links, current location, and project selection', async () => {
    const user = userEvent.setup();
    const setCurrentProjectRole = jest.fn();

    renderHeader(
      [
        {
          path: 'dashboard',
          handle: breadcrumb('Dashboard'),
          children: [
            {
              path: 'assistant',
              handle: breadcrumb('Assistants'),
              children: [
                {
                  path: 'list',
                  handle: breadcrumb('List'),
                  element: (
                    <ActionableHeader
                      className="custom-header"
                      data-testid="header"
                    />
                  ),
                },
              ],
            },
          ],
        },
      ],
      '/dashboard/assistant/list',
      {
        projectRoles,
        currentProjectRole: projectRoles[0],
        setCurrentProjectRole,
      },
    );

    expect(screen.getByTestId('header')).toHaveClass(
      'custom-header',
      'bg-shell',
      'border-border-subtle',
    );
    expect(screen.getByRole('link', { name: 'Dashboard' })).toHaveAttribute(
      'href',
      '/dashboard',
    );
    expect(screen.getByRole('link', { name: 'Assistants' })).toHaveAttribute(
      'href',
      '/dashboard/assistant',
    );
    expect(screen.getByText('List').closest('li')).toHaveAttribute(
      'aria-current',
      'page',
    );
    expect(
      screen.queryByRole('link', { name: 'List' }),
    ).not.toBeInTheDocument();

    const projectSelector = screen.getByLabelText('Select project');
    expect(projectSelector).toBeInTheDocument();
    expect(projectSelector).toHaveClass('max-sm:w-12!', 'max-sm:min-w-12!');
    expect(screen.getByRole('option', { name: 'Alpha' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Beta' })).toBeInTheDocument();

    await user.selectOptions(projectSelector, String('1'));
    expect(setCurrentProjectRole).toHaveBeenCalledWith(projectRoles[1]);
  });

  it('renders the dashboard as a non-interactive current location', () => {
    renderHeader(
      [
        {
          path: 'dashboard',
          handle: breadcrumb('Dashboard'),
          element: <ActionableHeader />,
        },
      ],
      '/dashboard',
    );

    expect(screen.getByText('Dashboard')).toHaveAttribute(
      'aria-current',
      'page',
    );
    expect(
      screen.queryByRole('link', { name: 'Dashboard' }),
    ).not.toBeInTheDocument();
  });

  it('uses Carbon overflow and renders dynamic route metadata', () => {
    const AssistantName = () => <>Support bot</>;

    renderHeader(
      [
        {
          path: 'deployment',
          handle: breadcrumb('Deployment'),
          children: [
            {
              path: 'assistant',
              handle: breadcrumb('Assistants'),
              children: [
                {
                  path: ':assistantId',
                  handle: { breadcrumb: { render: AssistantName } },
                  children: [
                    {
                      path: 'create-new-version',
                      handle: breadcrumb('Create version'),
                      element: <ActionableHeader />,
                    },
                  ],
                },
              ],
            },
          ],
        },
      ],
      '/deployment/assistant/2301664620831571968/create-new-version',
    );

    expect(
      screen.getByRole('button', { name: 'Show hidden breadcrumb items' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Assistants' })).toHaveAttribute(
      'href',
      '/deployment/assistant',
    );
    expect(screen.getByRole('link', { name: 'Support bot' })).toHaveAttribute(
      'href',
      '/deployment/assistant/2301664620831571968',
    );
    expect(screen.getByText('Create version').closest('li')).toHaveAttribute(
      'aria-current',
      'page',
    );
    expect(
      screen.queryByRole('link', { name: 'Create version' }),
    ).not.toBeInTheDocument();
  });

  it('renders loading skeletons while a blocking request is active', () => {
    mockRapidaLoading = true;
    mockRapidaLoadingType = 'block';

    renderHeader(
      [
        {
          path: 'dashboard',
          handle: breadcrumb('Dashboard'),
          element: <ActionableHeader />,
        },
      ],
      '/dashboard',
      {
        projectRoles,
        currentProjectRole: projectRoles[0],
        setCurrentProjectRole: jest.fn(),
      },
    );

    expect(screen.getByTestId('breadcrumb-skeleton')).toHaveClass('pl-4');
    expect(screen.getByTestId('dropdown-skeleton')).toHaveAttribute(
      'data-size',
      'sm',
    );
  });

  it('does not render project dropdown when setter is unavailable', () => {
    render(
      <AuthContext.Provider
        value={{
          projectRoles,
          currentProjectRole: projectRoles[0],
        }}
      >
        <CustomerOptions />
      </AuthContext.Provider>,
    );

    expect(screen.queryByLabelText('Select project')).not.toBeInTheDocument();
  });

  it('does not render project dropdown when it is disabled', () => {
    render(
      <AuthContext.Provider
        value={{
          projectRoles,
          currentProjectRole: projectRoles[0],
          setCurrentProjectRole: jest.fn(),
        }}
      >
        <CustomerOptions showProjectSelector={false} />
      </AuthContext.Provider>,
    );

    expect(screen.queryByLabelText('Select project')).not.toBeInTheDocument();
  });

  it('ignores an empty project selection from the dropdown', async () => {
    const user = userEvent.setup();
    const setCurrentProjectRole = jest.fn();

    render(
      <AuthContext.Provider
        value={{
          projectRoles,
          currentProjectRole: projectRoles[0],
          setCurrentProjectRole,
        }}
      >
        <CustomerOptions />
      </AuthContext.Provider>,
    );

    await user.selectOptions(
      screen.getByLabelText('Select project'),
      String('-1'),
    );

    expect(setCurrentProjectRole).not.toHaveBeenCalled();
  });

  it('renders configured account links without documentation or source', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <AuthContext.Provider value={{}}>
        <CustomerOptions />
      </AuthContext.Provider>,
    );

    await user.click(screen.getByRole('button', { name: 'Account' }));

    expect(
      screen.queryByRole('img', { name: 'Acme Voice' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Documentation')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Source')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Support')).toHaveAttribute(
      'href',
      mockTheme.links.support,
    );
    expect(screen.getByLabelText('Terms')).toHaveAttribute(
      'href',
      mockTheme.links.terms,
    );
    expect(screen.getByLabelText('Privacy')).toHaveAttribute(
      'href',
      mockTheme.links.privacy,
    );
  });

  it('hides the mode action when the theme disables mode selection', () => {
    mockTheme.allowModeSelection = false;

    render(
      <AuthContext.Provider value={{}}>
        <CustomerOptions />
      </AuthContext.Provider>,
    );

    expect(
      screen.queryByRole('button', { name: 'Switch to dark mode' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Account' })).toBeInTheDocument();
  });

  it('toggles theme mode from the global action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <AuthContext.Provider value={{}}>
        <CustomerOptions />
      </AuthContext.Provider>,
    );

    await user.click(
      screen.getByRole('button', { name: 'Switch to dark mode' }),
    );

    expect(mockToggleMode).toHaveBeenCalledTimes(1);
  });

  it('shows the light mode action while dark mode is active', () => {
    mockResolvedMode = 'dark';

    render(
      <AuthContext.Provider value={{}}>
        <CustomerOptions />
      </AuthContext.Provider>,
    );

    expect(
      screen.getByRole('button', { name: 'Switch to light mode' }),
    ).toBeInTheDocument();
  });
});
