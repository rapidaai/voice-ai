import React from 'react';
import { render, screen } from '@testing-library/react';
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

  it('uses non-interactive Carbon progress for the two-step flow', () => {
    renderLayout(theme);

    expect(screen.getByText('Step 1 of 2')).toBeInTheDocument();
    expect(
      screen.getByRole('progressbar', { name: 'Onboarding progress' }),
    ).toHaveAttribute('aria-valuenow', '1');
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('reflects the project step in the visible and semantic progress', () => {
    renderLayout(theme, '/onboarding/project');

    expect(screen.getByText('Step 2 of 2')).toBeInTheDocument();
    expect(
      screen.getByRole('progressbar', { name: 'Onboarding progress' }),
    ).toHaveAttribute('aria-valuenow', '2');
  });
});
