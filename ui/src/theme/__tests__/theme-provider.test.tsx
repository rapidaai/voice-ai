import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { ThemeProvider, useTheme } from '@/theme/theme-provider';
import { DEFAULT_THEME, THEME_STORAGE_KEY } from '@/theme/theme-loader';
import { ThemeManifest } from '@/theme/types';

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
  });

  it('applies brand tokens and toggles the resolved color mode', () => {
    render(
      <ThemeProvider theme={DEFAULT_THEME}>
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
    ).toBe(DEFAULT_THEME.colors.light.primary);
    expect(
      document.documentElement.style.getPropertyValue('--brand-on-primary'),
    ).toBe(DEFAULT_THEME.colors.light.onPrimary);
    expect(document.querySelector("meta[name='theme-color']")).toHaveAttribute(
      'content',
      DEFAULT_THEME.colors.light.primary,
    );

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
    expect(document.documentElement).toHaveAttribute('data-color-mode', 'dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
    expect(
      document.documentElement.style.getPropertyValue('--brand-primary'),
    ).toBe(DEFAULT_THEME.colors.dark.primary);
  });

  it('does not change a locked whitelabel theme', () => {
    const lockedTheme: ThemeManifest = {
      ...DEFAULT_THEME,
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
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBeNull();
  });

  it('synchronizes mode changes from another browser tab', () => {
    render(
      <ThemeProvider theme={DEFAULT_THEME}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    fireEvent(
      window,
      new StorageEvent('storage', {
        key: THEME_STORAGE_KEY,
        newValue: 'dark',
      }),
    );

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
      <ThemeProvider theme={DEFAULT_THEME}>
        <ThemeConsumer />
      </ThemeProvider>,
    );

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByRole('button')).toHaveTextContent('Rapida AI:dark');
  });
});
