import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { HelmetProvider } from 'react-helmet-async';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider, useTheme } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { Helmet } from './index';

const theme = developmentConfig.theme as unknown as ThemeManifest;
const tenantTheme: ThemeManifest = {
  ...theme,
  id: 'tenant-voice',
  brand: {
    ...theme.brand,
    name: 'Tenant Voice',
  },
};

const ModeToggle = () => {
  const { toggleMode } = useTheme();
  return (
    <button type="button" onClick={toggleMode}>
      Toggle mode
    </button>
  );
};

describe('Helmet', () => {
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

  afterEach(() => {
    document
      .querySelectorAll("link[rel~='icon']")
      .forEach(icon => icon.remove());
  });

  it('uses the configured brand in the page title', async () => {
    render(
      <HelmetProvider>
        <ThemeProvider theme={tenantTheme}>
          <Helmet title="Dashboard" />
          <ModeToggle />
        </ThemeProvider>
      </HelmetProvider>,
    );

    await waitFor(() => expect(document.title).toBe('Dashboard - Tenant Voice'));
    const icons =
      document.querySelectorAll<HTMLLinkElement>("link[rel~='icon']");
    expect(icons).toHaveLength(1);
    expect(icons[0].getAttribute('href')).toBe(tenantTheme.brand.favicon);

    fireEvent.click(screen.getByRole('button', { name: 'Toggle mode' }));
    await waitFor(() =>
      expect(document.title).toBe('Dashboard - Tenant Voice'),
    );
  });

  it('uses only the configured brand when no page title is provided', async () => {
    render(
      <HelmetProvider>
        <ThemeProvider theme={tenantTheme}>
          <Helmet />
        </ThemeProvider>
      </HelmetProvider>,
    );

    await waitFor(() => expect(document.title).toBe('Tenant Voice'));
  });
});
