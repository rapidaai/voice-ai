import { ThemeManifest, ThemeMode } from './types';
import { safeStorageGet, safeStorageSet } from './theme-storage';

export const THEME_STORAGE_KEY = 'ui-theme-mode';
export const THEME_CACHE_KEY = 'ui-theme-manifest';
export const THEME_LOAD_TIMEOUT_MS = 3000;

export const getThemeManifestUrl = () =>
  new URL(
    `${process.env.PUBLIC_URL?.replace(/\/$/, '') ?? ''}/theme.json`,
    window.location.origin,
  ).toString();

export const DEFAULT_THEME: ThemeManifest = {
  schemaVersion: 1,
  id: 'rapida',
  brand: {
    name: 'Rapida AI',
    logos: {
      full: {
        light: '/logos/rapida-db.svg',
        dark: '/logos/rapida-wh.svg',
      },
      compact: {
        light: '/favicon_io/original-icon.png',
        dark: '/favicon_io/original-icon.png',
      },
    },
    favicon: '/favicon_io/favicon.ico',
  },
  links: {
    documentation: 'https://doc.rapida.ai',
    source: 'https://github.com/rapidaai/voice-ai',
    support: 'mailto:prashant@rapida.ai',
    terms: '/static/terms-and-conditions',
    privacy: '/static/privacy-policy',
  },
  defaultMode: 'system',
  allowModeSelection: true,
  colors: {
    light: {
      primary: '#0353e9',
      primaryHover: '#002d9c',
      primaryActive: '#001d6c',
      onPrimary: '#ffffff',
    },
    dark: {
      primary: '#4589ff',
      primaryHover: '#78a9ff',
      primaryActive: '#a6c8ff',
      onPrimary: '#161616',
    },
  },
};

const isThemeMode = (value: unknown): value is ThemeMode =>
  value === 'light' || value === 'dark' || value === 'system';

const isNonEmptyString = (value: unknown): value is string =>
  typeof value === 'string' && value.trim().length > 0;

const HEX_COLOR_PATTERN = /^#[0-9a-f]{6}$/i;

const isThemeColor = (value: unknown): value is string =>
  isNonEmptyString(value) && HEX_COLOR_PATTERN.test(value);

const getRelativeLuminance = (hexColor: string) => {
  const channels = [1, 3, 5].map(offset =>
    Number.parseInt(hexColor.slice(offset, offset + 2), 16),
  );
  const [red, green, blue] = channels.map(channel => {
    const normalized = channel / 255;
    return normalized <= 0.04045
      ? normalized / 12.92
      : ((normalized + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue;
};

export const getContrastRatio = (first: string, second: string) => {
  const firstLuminance = getRelativeLuminance(first);
  const secondLuminance = getRelativeLuminance(second);
  return (
    (Math.max(firstLuminance, secondLuminance) + 0.05) /
    (Math.min(firstLuminance, secondLuminance) + 0.05)
  );
};

const hasAccessibleInteractionContrast = (
  colors: ThemeManifest['colors']['light'],
) =>
  [colors.primary, colors.primaryHover, colors.primaryActive].every(
    color => getContrastRatio(color, colors.onPrimary) >= 4.5,
  );

const isAppLink = (value: unknown): value is string => {
  if (!isNonEmptyString(value)) return false;
  if (value.startsWith('/') && !value.startsWith('//')) return true;

  try {
    const url = new URL(value);
    return ['https:', 'mailto:'].includes(url.protocol);
  } catch {
    return false;
  }
};

const isAssetLink = (value: unknown): value is string => {
  if (!isNonEmptyString(value)) return false;
  if (value.startsWith('/') && !value.startsWith('//')) return true;

  try {
    const url = new URL(value);
    return url.protocol === 'https:';
  } catch {
    return false;
  }
};

export const isThemeManifest = (value: unknown): value is ThemeManifest => {
  if (!value || typeof value !== 'object') return false;

  const manifest = value as Partial<ThemeManifest>;
  const light = manifest.colors?.light;
  const dark = manifest.colors?.dark;
  const logos = manifest.brand?.logos;
  const links = manifest.links;
  const hasValidLogos =
    logos === undefined ||
    (isAssetLink(logos.full?.light) &&
      isAssetLink(logos.full.dark) &&
      isAssetLink(logos.compact?.light) &&
      isAssetLink(logos.compact.dark));

  return (
    manifest.schemaVersion === 1 &&
    isNonEmptyString(manifest.id) &&
    isNonEmptyString(manifest.brand?.name) &&
    hasValidLogos &&
    (manifest.brand.favicon === undefined ||
      isAssetLink(manifest.brand.favicon)) &&
    isAppLink(links?.documentation) &&
    isAppLink(links?.source) &&
    isAppLink(links?.support) &&
    isAppLink(links?.terms) &&
    isAppLink(links?.privacy) &&
    isThemeMode(manifest.defaultMode) &&
    typeof manifest.allowModeSelection === 'boolean' &&
    isThemeColor(light?.primary) &&
    isThemeColor(light.primaryHover) &&
    isThemeColor(light.primaryActive) &&
    isThemeColor(light.onPrimary) &&
    hasAccessibleInteractionContrast(light) &&
    isThemeColor(dark?.primary) &&
    isThemeColor(dark.primaryHover) &&
    isThemeColor(dark.primaryActive) &&
    isThemeColor(dark.onPrimary) &&
    hasAccessibleInteractionContrast(dark)
  );
};

export const getCachedThemeManifest = (): ThemeManifest | null => {
  const cachedTheme = safeStorageGet(THEME_CACHE_KEY);
  if (!cachedTheme) return null;

  try {
    const manifest: unknown = JSON.parse(cachedTheme);
    return isThemeManifest(manifest) ? manifest : null;
  } catch {
    return null;
  }
};

export const getBootstrapTheme = (): ThemeManifest =>
  getCachedThemeManifest() ?? DEFAULT_THEME;

const warnThemeFallback = (reason: string) => {
  if (process.env.NODE_ENV !== 'test') {
    console.warn(`[theme] ${reason}; using the default theme.`);
  }
};

export async function loadThemeManifest(
  timeoutMs = THEME_LOAD_TIMEOUT_MS,
): Promise<ThemeManifest> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(getThemeManifestUrl(), {
      cache: 'no-store',
      signal: controller.signal,
    });
    if (!response.ok) {
      warnThemeFallback(`manifest request failed with HTTP ${response.status}`);
      return DEFAULT_THEME;
    }

    const manifest: unknown = await response.json();
    if (!isThemeManifest(manifest)) {
      warnThemeFallback('manifest validation failed');
      return DEFAULT_THEME;
    }

    safeStorageSet(THEME_CACHE_KEY, JSON.stringify(manifest));
    return manifest;
  } catch (error) {
    warnThemeFallback(
      error instanceof DOMException && error.name === 'AbortError'
        ? `manifest request timed out after ${timeoutMs}ms`
        : 'manifest request failed',
    );
    return getCachedThemeManifest() ?? DEFAULT_THEME;
  } finally {
    window.clearTimeout(timeout);
  }
}
