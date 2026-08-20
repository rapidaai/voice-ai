import {
  DEFAULT_THEME,
  getContrastRatio,
  getBootstrapTheme,
  getThemeManifestUrl,
  loadThemeManifest,
  THEME_CACHE_KEY,
} from '@/theme/theme-loader';
import { ThemeManifest } from '@/theme/types';

const customTheme: ThemeManifest = {
  ...DEFAULT_THEME,
  id: 'customer-a',
  brand: { name: 'Customer A' },
};

describe('theme loader', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('loads a valid runtime theme manifest', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => customTheme,
    } as Response);

    await expect(loadThemeManifest()).resolves.toEqual(customTheme);
    expect(fetch).toHaveBeenCalledWith(getThemeManifestUrl(), {
      cache: 'no-store',
      signal: expect.any(AbortSignal),
    });
  });

  it('falls back when the runtime manifest is invalid', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'invalid' }),
    } as Response);

    await expect(loadThemeManifest()).resolves.toEqual(DEFAULT_THEME);
  });

  it('rejects incomplete whitelabel logo sets', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({
        ...customTheme,
        brand: {
          name: 'Customer A',
          logos: { full: { light: '/logo.svg', dark: '/logo-dark.svg' } },
        },
      }),
    } as Response);

    await expect(loadThemeManifest()).resolves.toEqual(DEFAULT_THEME);
  });

  it('rejects unsafe colors and link protocols', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({
        ...customTheme,
        colors: {
          ...customTheme.colors,
          light: {
            ...customTheme.colors.light,
            primary: 'red; background: url(https://example.com)',
          },
        },
        links: {
          ...customTheme.links,
          documentation: 'ftp://example.com/docs',
        },
      }),
    } as Response);

    await expect(loadThemeManifest()).resolves.toEqual(DEFAULT_THEME);
  });

  it('rejects protocol-relative assets and inaccessible interaction colors', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({
        ...customTheme,
        brand: {
          ...customTheme.brand,
          favicon: '//attacker.example/favicon.ico',
        },
        colors: {
          ...customTheme.colors,
          dark: {
            ...customTheme.colors.dark,
            onPrimary: '#ffffff',
          },
        },
      }),
    } as Response);

    await expect(loadThemeManifest()).resolves.toEqual(DEFAULT_THEME);
  });

  it('uses the last-known-good theme during startup and request failures', async () => {
    localStorage.setItem(THEME_CACHE_KEY, JSON.stringify(customTheme));
    jest.spyOn(global, 'fetch').mockRejectedValue(new Error('offline'));

    expect(getBootstrapTheme()).toEqual(customTheme);
    await expect(loadThemeManifest()).resolves.toEqual(customTheme);
  });

  it('keeps default interaction states at WCAG AA contrast', () => {
    for (const colors of Object.values(DEFAULT_THEME.colors)) {
      expect(
        getContrastRatio(colors.primary, colors.onPrimary),
      ).toBeGreaterThanOrEqual(4.5);
      expect(
        getContrastRatio(colors.primaryHover, colors.onPrimary),
      ).toBeGreaterThanOrEqual(4.5);
      expect(
        getContrastRatio(colors.primaryActive, colors.onPrimary),
      ).toBeGreaterThanOrEqual(4.5);
    }
  });

  it('aborts a stalled manifest request and falls back', async () => {
    jest.useFakeTimers();
    jest.spyOn(global, 'fetch').mockImplementation((_, init) => {
      return new Promise((_, reject) => {
        init?.signal?.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'));
        });
      });
    });

    const result = loadThemeManifest(25);
    jest.advanceTimersByTime(25);

    await expect(result).resolves.toEqual(DEFAULT_THEME);
    jest.useRealTimers();
  });
});
