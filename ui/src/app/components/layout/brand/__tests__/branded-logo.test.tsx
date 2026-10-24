import React from 'react';
import { render, screen } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { BrandedLogo } from '../branded-logo';

const baseTheme = developmentConfig.theme as unknown as ThemeManifest;

const renderLogo = (
  logo: React.ReactElement,
  theme: ThemeManifest = {
    ...baseTheme,
    defaultMode: 'dark',
    allowModeSelection: false,
  },
) => render(<ThemeProvider theme={theme}>{logo}</ThemeProvider>);

describe('BrandedLogo', () => {
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

  it('renders the themed logo for the active color mode and variant', () => {
    renderLogo(
      <BrandedLogo
        variant="compact"
        className="h-6 w-6"
        data-testid="brand-logo"
      />,
    );

    expect(screen.getByAltText('Rapida AI')).toHaveAttribute(
      'src',
      baseTheme.brand.logos?.compact.dark,
    );
    expect(screen.getByTestId('brand-logo')).toBe(
      screen.getByAltText('Rapida AI'),
    );
    expect(screen.getByAltText('Rapida AI')).toHaveClass('h-6', 'w-6');
  });

  it('uses an explicit color mode when one is provided', () => {
    renderLogo(<BrandedLogo variant="full" colorMode="light" />);

    expect(screen.getByAltText('Rapida AI')).toHaveAttribute(
      'src',
      baseTheme.brand.logos?.full.light,
    );
    expect(screen.getByLabelText('Environment: (dev)')).toHaveClass(
      'text-[0.75rem]',
      'font-normal',
      'leading-[1rem]',
      'tracking-[0.32px]',
    );
  });

  it('falls back to brand text when no logo assets are configured', () => {
    renderLogo(
      <BrandedLogo
        className="max-w-32"
        data-testid="brand-text"
        textClassName="text-lg"
        title="Tenant brand"
      />,
      {
        ...baseTheme,
        brand: {
          ...baseTheme.brand,
          name: 'Tenant Voice',
          logos: undefined,
        },
        defaultMode: 'light',
        allowModeSelection: false,
      },
    );

    expect(screen.getByTestId('brand-text')).toHaveTextContent('Tenant Voice');
    expect(screen.getByTestId('brand-text')).toHaveAttribute(
      'title',
      'Tenant brand',
    );
    expect(screen.getByTestId('brand-text')).toHaveClass('max-w-32', 'text-lg');
    expect(screen.queryByAltText('Tenant Voice')).not.toBeInTheDocument();
  });
});
