import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { HelmetProvider } from 'react-helmet-async';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider, useTheme } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { Helmet } from './index';

const theme = developmentConfig.theme as unknown as ThemeManifest;

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

  it('keeps the configured favicon owned by ThemeProvider', async () => {
    render(
      <HelmetProvider>
        <ThemeProvider theme={theme}>
          <Helmet title="Dashboard" />
          <ModeToggle />
        </ThemeProvider>
      </HelmetProvider>,
    );

    await waitFor(() => expect(document.title).toBe('Dashboard - Rapida AI'));
    const icons =
      document.querySelectorAll<HTMLLinkElement>("link[rel~='icon']");
    expect(icons).toHaveLength(1);
    expect(icons[0].getAttribute('href')).toBe(theme.brand.favicon);

    fireEvent.click(screen.getByRole('button', { name: 'Toggle mode' }));
    await waitFor(() => expect(document.title).toBe('Dashboard - Rapida AI'));
  });
});
