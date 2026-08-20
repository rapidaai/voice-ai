import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { ThemeProvider, useTheme } from '@/theme/theme-provider';
import { THEME_STORAGE_KEY } from '@/theme/theme-config';
import { ThemeManifest } from '@/theme/types';
import developmentConfig from '@/configs/config.development.json';

const configuredTheme = developmentConfig.theme as unknown as ThemeManifest;

const matchMedia = (matches: boolean) =>
  ({
    matches,
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
  }) as unknown as MediaQueryList;

const ThemeConsumer = () => {
  const { resolvedMode, theme, toggleMode } = useTheme();
  return (
    <button onClick={toggleMode} type="button">
      {theme.brand.name}:{resolvedMode}
    </button>
  );
};

describe('ThemeProvider', () => {
  beforeEach(() => {
    localStorage.clear();
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue(matchMedia(false)),
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
    document.documentElement.removeAttribute('data-brand');
    document.documentElement.removeAttribute('data-color-mode');
    document.documentElement.removeAttribute('data-theme-ready');
    document.documentElement.removeAttribute('style');
    document.querySelector("meta[name='theme-color']")?.remove();
    document
      .querySelectorAll("link[rel~='icon']")
      .forEach(icon => icon.remove());
  });

  it('applies brand tokens and toggles the resolved color mode', () => {
    render(
      <ThemeProvider theme={configuredTheme}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:light');
    expect(document.documentElement).toHaveAttribute('data-brand', 'rapida');
    expect(document.documentElement).toHaveAttribute(
      'data-color-mode',
      'light',
    );
    expect(document.documentElement).toHaveAttribute(
      'data-theme-ready',
      'true',
    );
    expect(
      document.documentElement.style.getPropertyValue('--brand-primary'),
    ).toBe(configuredTheme.colors.light.primary);
    expect(
      document.documentElement.style.getPropertyValue('--brand-on-primary'),
    ).toBe(configuredTheme.colors.light.onPrimary);
    expect(document.querySelector("meta[name='theme-color']")).toHaveAttribute(
      'content',
      configuredTheme.colors.light.primary,
    );

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
    expect(document.documentElement).toHaveAttribute('data-color-mode', 'dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
    expect(
      document.documentElement.style.getPropertyValue('--brand-primary'),
    ).toBe(configuredTheme.colors.dark.primary);
  });

  it('does not change a locked whitelabel theme', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'light');
    const lockedTheme: ThemeManifest = {
      ...configuredTheme,
      defaultMode: 'dark',
      allowModeSelection: false,
    };

    render(
      <ThemeProvider theme={lockedTheme}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');
  });

  it('synchronizes mode changes from another browser tab', () => {
    render(
      <ThemeProvider theme={configuredTheme}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    const storageEvent = new Event('storage');
    Object.defineProperty(storageEvent, 'key', { value: THEME_STORAGE_KEY });
    Object.defineProperty(storageEvent, 'newValue', { value: 'dark' });
    fireEvent(window, storageEvent);

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
    expect(document.documentElement).toHaveAttribute('data-color-mode', 'dark');
  });

  it('renders and changes mode when browser storage is unavailable', () => {
    jest.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('Denied', 'SecurityError');
    });
    jest.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('Denied', 'SecurityError');
    });

    render(
      <ThemeProvider theme={configuredTheme}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
  });

  it('keeps a selected mode when a selectable theme object refreshes', () => {
    const { rerender } = render(
      <ThemeProvider theme={configuredTheme}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    fireEvent.click(screen.getByRole('button'));
    rerender(
      <ThemeProvider theme={{ ...configuredTheme, id: 'refreshed' }}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
  });

  it('follows system changes when selection is locked to system', () => {
    let mediaListener: ((event: MediaQueryListEvent) => void) | undefined;
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(
          (_event: string, listener: (event: MediaQueryListEvent) => void) => {
            mediaListener = listener;
          },
        ),
        removeEventListener: jest.fn(),
      }),
    });
    const lockedSystemTheme: ThemeManifest = {
      ...configuredTheme,
      defaultMode: 'system',
      allowModeSelection: false,
    };

    render(
      <ThemeProvider theme={lockedSystemTheme}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    act(() => mediaListener?.({ matches: true } as MediaQueryListEvent));

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
  });
});
