import React from 'react';
import { render, screen } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { PreviewAgentHeader } from '../preview-agent-header';

const theme = developmentConfig.theme as unknown as ThemeManifest;

const tenantTheme: ThemeManifest = {
  ...theme,
  id: 'tenant-voice',
  brand: {
    ...theme.brand,
    name: 'Tenant Voice',
    logos: {
      full: {
        light: '/tenant/full-light.svg',
        dark: '/tenant/full-dark.svg',
      },
      compact: {
        light: '/tenant/compact-light.svg',
        dark: '/tenant/compact-dark.svg',
      },
    },
  },
  defaultMode: 'light',
};

jest.mock('@/app/components/navigation/actionable-header', () => ({
  CustomerOptions: () => <div>Customer options</div>,
}));

describe('PreviewAgentHeader', () => {
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

  it('renders the configured full logo', () => {
    render(
      <ThemeProvider theme={tenantTheme}>
        <PreviewAgentHeader />
      </ThemeProvider>,
    );

    expect(screen.getByRole('img', { name: 'Tenant Voice' })).toHaveAttribute(
      'src',
      '/tenant/full-light.svg',
    );
    expect(screen.queryByAltText('Rapida AI')).not.toBeInTheDocument();
  });

  it('falls back to the configured brand name without logos', () => {
    render(
      <ThemeProvider
        theme={{
          ...tenantTheme,
          brand: {
            ...tenantTheme.brand,
            logos: undefined,
          },
        }}
      >
        <PreviewAgentHeader />
      </ThemeProvider>,
    );

    expect(screen.getByText('Tenant Voice')).toBeInTheDocument();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });
});
