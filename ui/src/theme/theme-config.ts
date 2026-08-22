import { ThemeColors, ThemeManifest, ThemeMode } from './types';

export const THEME_STORAGE_KEY = 'ui-theme-mode';

const DEFAULT_COLORS: ThemeManifest['colors'] = {
  light: {
    primary: '#525252',
    primaryHover: '#393939',
    primaryActive: '#262626',
    onPrimary: '#ffffff',
  },
  dark: {
    primary: '#c6c6c6',
    primaryHover: '#e0e0e0',
    primaryActive: '#f4f4f4',
    onPrimary: '#161616',
  },
};

const DEFAULT_LINKS: ThemeManifest['links'] = {
  documentation: '#',
  source: '#',
  support: '#',
  terms: '#',
  privacy: '#',
};

const asRecord = (value: unknown): Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};

const asString = (value: unknown, fallback: string): string =>
  typeof value === 'string' && value.trim().length > 0 ? value : fallback;

const asOptionalString = (value: unknown): string | undefined =>
  typeof value === 'string' && value.trim().length > 0 ? value : undefined;

const asThemeMode = (value: unknown): ThemeMode =>
  value === 'light' || value === 'dark' || value === 'system'
    ? value
    : 'system';

const normalizeColors = (
  value: unknown,
  defaults: ThemeColors,
): ThemeColors => {
  const colors = asRecord(value);

  return {
    primary: asString(colors.primary, defaults.primary),
    primaryHover: asString(colors.primaryHover, defaults.primaryHover),
    primaryActive: asString(colors.primaryActive, defaults.primaryActive),
    onPrimary: asString(colors.onPrimary, defaults.onPrimary),
  };
};

const normalizeLogos = (
  value: unknown,
): ThemeManifest['brand']['logos'] | undefined => {
  const logos = asRecord(value);
  const full = asRecord(logos.full);
  const compact = asRecord(logos.compact);
  const configured = [
    asOptionalString(full.light),
    asOptionalString(full.dark),
    asOptionalString(compact.light),
    asOptionalString(compact.dark),
  ];
  const fallback = configured.find(Boolean);

  if (!fallback) return undefined;

  return {
    full: {
      light: configured[0] ?? fallback,
      dark: configured[1] ?? fallback,
    },
    compact: {
      light: configured[2] ?? fallback,
      dark: configured[3] ?? fallback,
    },
  };
};

export const normalizeThemeManifest = (value: unknown): ThemeManifest => {
  const manifest = asRecord(value);
  const brand = asRecord(manifest.brand);
  const links = asRecord(manifest.links);
  const colors = asRecord(manifest.colors);
  const logos = normalizeLogos(brand.logos);

  return {
    schemaVersion: 1,
    id: asString(manifest.id, 'default'),
    brand: {
      name: asString(brand.name, 'Application'),
      ...(logos ? { logos } : {}),
      ...(asOptionalString(brand.favicon)
        ? { favicon: asOptionalString(brand.favicon) }
        : {}),
    },
    links: {
      documentation: asString(links.documentation, DEFAULT_LINKS.documentation),
      source: asString(links.source, DEFAULT_LINKS.source),
      support: asString(links.support, DEFAULT_LINKS.support),
      terms: asString(links.terms, DEFAULT_LINKS.terms),
      privacy: asString(links.privacy, DEFAULT_LINKS.privacy),
    },
    defaultMode: asThemeMode(manifest.defaultMode),
    allowModeSelection:
      typeof manifest.allowModeSelection === 'boolean'
        ? manifest.allowModeSelection
        : true,
    colors: {
      light: normalizeColors(colors.light, DEFAULT_COLORS.light),
      dark: normalizeColors(colors.dark, DEFAULT_COLORS.dark),
    },
  };
};
