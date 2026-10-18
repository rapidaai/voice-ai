import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

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

let mockPathname = '/dashboard/assistant/list';
let mockRapidaLoading = false;
let mockRapidaLoadingType: string | undefined;

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useLocation: () => ({ pathname: mockPathname }),
}));

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
  BreadcrumbItem: ({ children }: any) => <li>{children}</li>,
  BreadcrumbSkeleton: ({ className }: any) => (
    <div className={className} data-testid="breadcrumb-skeleton" />
  ),
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

describe('Actionable header project switcher', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPathname = '/dashboard/assistant/list';
    mockResolvedMode = 'light';
    mockTheme.allowModeSelection = true;
    mockRapidaLoading = false;
    mockRapidaLoadingType = undefined;
  });

  it('renders project dropdown in header and switches project on selection', async () => {
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
        <ActionableHeader className="custom-header" data-testid="header" />
      </AuthContext.Provider>,
    );

    expect(screen.getByTestId('header')).toHaveClass(
      'custom-header',
      'bg-shell',
      'border-border-subtle',
    );
    expect(screen.getByRole('link', { name: 'dashboard' })).toHaveAttribute(
      'href',
      '/dashboard',
    );
    expect(screen.getByRole('link', { name: 'assistant' })).toHaveAttribute(
      'href',
      '/dashboard/assistant',
    );
    expect(screen.getByRole('link', { name: 'list' })).toHaveAttribute(
      'href',
      '/dashboard/assistant/list',
    );

    const projectSelector = screen.getByLabelText('Select project');
    expect(projectSelector).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Alpha' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Beta' })).toBeInTheDocument();

    await user.selectOptions(projectSelector, String('1'));
    expect(setCurrentProjectRole).toHaveBeenCalledWith(projectRoles[1]);
  });

  it('renders loading skeletons while a blocking request is active', () => {
    mockRapidaLoading = true;
    mockRapidaLoadingType = 'block';

    render(
      <AuthContext.Provider
        value={{
          projectRoles,
          currentProjectRole: projectRoles[0],
          setCurrentProjectRole: jest.fn(),
        }}
      >
        <ActionableHeader />
      </AuthContext.Provider>,
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
