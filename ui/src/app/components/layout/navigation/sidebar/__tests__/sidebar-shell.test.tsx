import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SidebarNavigation } from '@/app/components/layout/navigation/sidebar';
import developmentConfig from '@/configs/config.development.json';
import { ThemeManifest } from '@/theme/types';

const theme = developmentConfig.theme as unknown as ThemeManifest;

let mockOpen = true;
let mockLocked = false;
let mockSetLocked = jest.fn();
let mockWorkspaceFeatures: Record<string, boolean> = { knowledge: false };
let mockRapidaLoading = false;
let mockRapidaLoadingType: string | undefined;

jest.mock('@/context/sidebar-context', () => ({
  useSidebar: () => ({
    open: mockOpen,
    locked: mockLocked,
    setOpen: jest.fn(),
    setLocked: mockSetLocked,
  }),
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children, kind, ...props }: any) => (
    <button data-design-system-button-kind={kind} {...props}>
      {children}
    </button>
  ),
}));

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => {
    const config = jest.requireActual('@/configs/config.development.json');
    return { resolvedMode: 'dark', theme: config.theme };
  },
}));

jest.mock('@/workspace', () => ({
  useWorkspace: () => ({ features: mockWorkspaceFeatures }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: mockRapidaLoading,
    loadingType: mockRapidaLoadingType,
  }),
}));

jest.mock('@/app/components/ui/primitives/text', () => ({
  Text: ({
    children,
    isLoading,
  }: {
    children: React.ReactNode;
    isLoading?: boolean;
  }) => <span data-loading={String(Boolean(isLoading))}>{children}</span>,
}));

jest.mock('@/app/components/layout/navigation/sidebar/dashboard', () => ({
  Dashboard: () => null,
}));
jest.mock('@/app/components/layout/navigation/sidebar/deployment', () => ({
  Deployment: () => null,
}));
jest.mock('@/app/components/layout/navigation/sidebar/knowledge', () => ({
  Knowledge: () => <li data-testid="knowledge-item" />,
}));
jest.mock('@/app/components/layout/navigation/sidebar/observability', () => ({
  Observability: () => null,
}));
jest.mock('@/app/components/layout/navigation/sidebar/external-tools', () => ({
  ExternalTool: () => null,
}));
jest.mock('@/app/components/layout/navigation/sidebar/vault', () => ({
  Vault: () => null,
}));
jest.mock('@/app/components/layout/navigation/sidebar/team', () => ({
  Team: () => null,
}));
jest.mock('@/app/components/layout/navigation/sidebar/project', () => ({
  Project: () => null,
}));

describe('sidebar shell', () => {
  beforeEach(() => {
    mockOpen = true;
    mockLocked = false;
    mockSetLocked = jest.fn();
    mockWorkspaceFeatures = { knowledge: false };
    mockRapidaLoading = false;
    mockRapidaLoadingType = undefined;
  });

  it('uses the shared shell surface and aligned full logo when expanded', () => {
    const { container } = render(<SidebarNavigation />);

    expect(container.firstChild).toHaveClass(
      'bg-shell',
      'border-border-subtle',
    );
    expect(screen.getByAltText('Rapida AI')).toHaveAttribute(
      'src',
      theme.brand.logos?.full.dark,
    );
    expect(screen.getByAltText('Rapida AI').parentElement).toHaveClass(
      'justify-start',
    );
  });

  it('uses the compact square logo centered in the collapsed rail', () => {
    mockOpen = false;

    render(<SidebarNavigation />);

    expect(screen.getByAltText('Rapida AI')).toHaveAttribute(
      'src',
      theme.brand.logos?.compact.dark,
    );
    expect(screen.getByAltText('Rapida AI')).toHaveClass('h-6', 'w-6');
    expect(screen.getByAltText('Rapida AI').parentElement).toHaveClass(
      'justify-center',
    );
  });

  it('uses the design-system ghost button to lock and unlock the sidebar', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<SidebarNavigation />);

    const toggle = screen.getByRole('button', { name: 'Expand sidebar' });
    expect(toggle).toHaveAttribute('data-design-system-button-kind', 'ghost');

    await user.click(toggle);
    expect(mockSetLocked).toHaveBeenCalledWith(true);
  });

  it('renders optional knowledge navigation and passes loading state to sections', () => {
    mockWorkspaceFeatures = {};
    mockRapidaLoading = true;
    mockRapidaLoadingType = 'block';

    render(<SidebarNavigation />);

    expect(screen.getByTestId('knowledge-item')).toBeInTheDocument();
    expect(screen.getByText('Observability')).toHaveAttribute(
      'data-loading',
      'true',
    );
  });

  it('uses the locked state for the collapse action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockLocked = true;

    render(<SidebarNavigation />);

    const toggle = screen.getByRole('button', { name: 'Collapse sidebar' });
    expect(toggle).toHaveTextContent('Collapse');

    await user.click(toggle);
    expect(mockSetLocked).toHaveBeenCalledWith(false);
  });
});
