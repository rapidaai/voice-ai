import React from 'react';
import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { OnboardingLayout } from '../onborading';

const theme = developmentConfig.theme as unknown as ThemeManifest;

const renderLayout = (
  configuredTheme: ThemeManifest,
  initialEntry = '/onboarding/organization',
) =>
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <ThemeProvider theme={configuredTheme}>
        <OnboardingLayout />
      </ThemeProvider>
    </MemoryRouter>,
  );

describe('OnboardingLayout', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  it('uses the full logo resolved for the active theme', () => {
    renderLayout(theme);

    expect(screen.getByRole('img', { name: 'Rapida AI' })).toHaveAttribute(
      'src',
      theme.brand.logos?.full.light,
    );
  });

  it('falls back to the configured brand name without logos', () => {
    const brandOnlyTheme: ThemeManifest = {
      ...theme,
      brand: { name: 'Customer Voice', favicon: '/customer.ico' },
    };

    renderLayout(brandOnlyTheme);

    expect(screen.queryByRole('img', { name: 'Customer Voice' })).toBeNull();
    expect(screen.getByText('Customer Voice')).toBeInTheDocument();
  });

  it('uses the shared Carbon shell and a compact step context', () => {
    renderLayout(theme);

    const banner = screen.getByRole('banner');
    const footer = screen.getByRole('contentinfo');
    const workspaceSetup = screen.getByRole('region', {
      name: 'Workspace setup',
    });

    expect(banner).toHaveClass('cds--header', 'relative!', 'h-12!');
    expect(
      within(banner).getByRole('link', { name: theme.brand.name }),
    ).toHaveAttribute('href', '/');
    expect(footer).toHaveClass('h-12', 'bg-shell');
    expect(
      within(footer).getByRole('link', {
        name: `${theme.brand.name} Platform`,
      }),
    ).toBeInTheDocument();
    expect(workspaceSetup).toHaveClass(
      'max-w-[37rem]',
      'bg-surface',
      'border-0',
    );
    expect(screen.getByText('Workspace setup')).toHaveClass(
      'text-[0.75rem]',
      'leading-[1rem]',
      'tracking-[0.32px]',
    );
    expect(screen.getByText('Organization')).toHaveClass(
      'text-[0.875rem]',
      'font-[600]',
      'leading-[1.125rem]',
    );
    expect(screen.getByText('Step 1 of 2')).toHaveClass(
      'text-[0.875rem]',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
    );
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('reflects the project step without using a system progress bar', () => {
    renderLayout(theme, '/onboarding/project');

    expect(screen.getByText('Step 2 of 2')).toBeInTheDocument();
    expect(screen.getByText('Project')).toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
  });
});
