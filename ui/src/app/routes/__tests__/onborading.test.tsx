import React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { OnboardingLayout } from '../onborading';

const theme = developmentConfig.theme as unknown as ThemeManifest;

const renderLayout = (configuredTheme: ThemeManifest) =>
  render(
    <MemoryRouter initialEntries={['/onboarding/organization']}>
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

  it('uses the inverse full logo and resolved mobile logo', () => {
    renderLayout(theme);

    const logos = screen.getAllByRole('img', { name: 'Rapida AI' });
    expect(logos[0]).toHaveAttribute('src', theme.brand.logos?.full.dark);
    expect(logos[1]).toHaveAttribute('src', theme.brand.logos?.compact.light);
  });

  it('falls back to the configured brand name without logos', () => {
    const brandOnlyTheme: ThemeManifest = {
      ...theme,
      brand: { name: 'Customer Voice', favicon: '/customer.ico' },
    };

    renderLayout(brandOnlyTheme);

    expect(screen.queryByRole('img', { name: 'Customer Voice' })).toBeNull();
    expect(screen.getAllByText('Customer Voice')).toHaveLength(2);
  });
});
