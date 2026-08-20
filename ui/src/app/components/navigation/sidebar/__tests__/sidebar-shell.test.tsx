import React from 'react';
import { render, screen } from '@testing-library/react';
import { SidebarNavigation } from '@/app/components/navigation/sidebar';
import developmentConfig from '@/configs/config.development.json';
import { ThemeManifest } from '@/theme/types';

const theme = developmentConfig.theme as unknown as ThemeManifest;

let mockOpen = true;

jest.mock('@/context/sidebar-context', () => ({
  useSidebar: () => ({
    open: mockOpen,
    locked: false,
    setOpen: jest.fn(),
    setLocked: jest.fn(),
  }),
}));

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => {
    const config = jest.requireActual('@/configs/config.development.json');
    return { resolvedMode: 'dark', theme: config.theme };
  },
}));

jest.mock('@/workspace', () => ({
  useWorkspace: () => ({ features: { knowledge: false } }),
}));

jest.mock('@/hooks', () => ({
  useRapidaStore: () => ({ loading: false, loadingType: undefined }),
}));

jest.mock('@/app/components/carbon/text', () => ({
  Text: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/app/components/navigation/sidebar/dashboard', () => ({
  Dashboard: () => null,
}));
jest.mock('@/app/components/navigation/sidebar/deployment', () => ({
  Deployment: () => null,
}));
jest.mock('@/app/components/navigation/sidebar/knowledge', () => ({
  Knowledge: () => null,
}));
jest.mock('@/app/components/navigation/sidebar/observability', () => ({
  Observability: () => null,
}));
jest.mock('@/app/components/navigation/sidebar/external-tools', () => ({
  ExternalTool: () => null,
}));
jest.mock('@/app/components/navigation/sidebar/vault', () => ({
  Vault: () => null,
}));
jest.mock('@/app/components/navigation/sidebar/team', () => ({
  Team: () => null,
}));
jest.mock('@/app/components/navigation/sidebar/project', () => ({
  Project: () => null,
}));

describe('sidebar shell', () => {
  beforeEach(() => {
    mockOpen = true;
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
});
