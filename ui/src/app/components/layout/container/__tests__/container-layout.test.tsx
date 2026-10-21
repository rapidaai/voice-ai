import React from 'react';
import { render, screen } from '@testing-library/react';
import { CenterBox } from '../center-box';
import { FlexBox } from '../flex-box';
import { MissionBox } from '../mission-box';
import { useRapidaStore } from '@/stores/app';

jest.mock('@/app/components/layout/navigation/header', () => ({
  Header: ({ className }: { className?: string }) => (
    <header className={className} data-testid="shell-header">
      Header
    </header>
  ),
}));

jest.mock('@/app/components/layout/footer/general-footer', () => ({
  GeneralFooter: () => <footer>Footer</footer>,
}));

jest.mock('@/app/components/layout/navigation/actionable-header', () => ({
  ActionableHeader: () => <header>Actionable header</header>,
}));

jest.mock('@/app/components/layout/navigation/sidebar', () => ({
  SidebarNavigation: () => <aside>Sidebar</aside>,
}));

jest.mock('@/app/components/ui/feedback', () => ({
  Loader: () => <div role="progressbar">Loading</div>,
  Toast: () => <div data-testid="toast" />,
}));

jest.mock('@/context/provider-context', () => ({
  ProviderContextProvider: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="provider-context">{children}</div>
  ),
}));

jest.mock('@/context/sidebar-context', () => ({
  SidebarProvider: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sidebar-context">{children}</div>
  ),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: jest.fn(),
}));

const mockedUseRapidaStore = useRapidaStore as unknown as jest.Mock;

describe('layout container structure', () => {
  beforeEach(() => {
    mockedUseRapidaStore.mockReturnValue({});
  });

  it('renders centered content with theme token classes and caller attributes', () => {
    render(
      <CenterBox
        className="custom-center"
        data-testid="center-box"
        mainLabel="Centered example"
      >
        <p>Centered content</p>
      </CenterBox>,
    );

    expect(screen.getByTestId('center-box')).toHaveClass(
      'custom-center',
      'bg-surface',
      'text-foreground',
      'grid-cols-[0_0_minmax(0,var(--container-2xl))_0_0]',
    );
    expect(screen.getByRole('main')).toHaveAccessibleName('Centered example');
    expect(screen.getByText('Centered content')).toBeInTheDocument();
  });

  it('renders the footer by default and supports a floating header', () => {
    render(
      <FlexBox className="custom-flex" data-testid="flex-box" isFloatingHeader>
        <p>Route content</p>
      </FlexBox>,
    );

    expect(screen.getByTestId('flex-box')).toHaveClass(
      'custom-flex',
      'bg-surface',
    );
    expect(screen.getByTestId('shell-header')).toHaveClass('sticky', 'top-0');
    expect(screen.getByText('Footer')).toBeInTheDocument();
    expect(screen.getByText('Route content')).toBeInTheDocument();
  });

  it('can hide the footer for auth-like routes', () => {
    render(
      <FlexBox showFooter={false}>
        <p>Auth content</p>
      </FlexBox>,
    );

    expect(screen.queryByText('Footer')).not.toBeInTheDocument();
    expect(screen.getByText('Auth content')).toBeInTheDocument();
  });

  it('renders the mission shell providers, chrome, loader, toast, and content', () => {
    render(
      <MissionBox className="custom-mission" data-testid="mission-box">
        <section>Workspace content</section>
      </MissionBox>,
    );

    expect(mockedUseRapidaStore).toHaveBeenCalled();
    expect(screen.getByTestId('mission-box')).toHaveClass('custom-mission');
    expect(screen.getByTestId('provider-context')).toBeInTheDocument();
    expect(screen.getByTestId('sidebar-context')).toBeInTheDocument();
    expect(screen.getByText('Sidebar')).toBeInTheDocument();
    expect(screen.getByText('Actionable header')).toBeInTheDocument();
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
    expect(screen.getByTestId('toast')).toBeInTheDocument();
    expect(screen.getByText('Workspace content')).toBeInTheDocument();
  });
});
