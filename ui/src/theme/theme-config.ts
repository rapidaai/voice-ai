import themeContract from './theme-contract.json';
import {
  DarkCarbonTheme,
  LightCarbonTheme,
  ThemeColorField,
  ThemeColors,
  ThemeConfigIssue,
  ThemeManifest,
  ThemeMode,
  ThemeParseResult,
} from './types';

export const THEME_STORAGE_KEY = themeContract.storageKey;

const COLOR_FIELDS = Object.keys(
  themeContract.colorFields,
) as ThemeColorField[];
const INTERNAL_PROPERTY_REFERENCE = /var\(\s*--(?:cds|brand)-/i;

const DEFAULT_COLORS: ThemeManifest['colors'] = {
  light: {
    carbonTheme: 'white',
    primary: '#525252',
    primaryHover: '#393939',
    primaryActive: '#262626',
    onPrimary: '#ffffff',
  },
  dark: {
    carbonTheme: 'g100',
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

const readLegacyColors = (
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

const readLegacyLogos = (
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

const parseVersion1 = (value: unknown): ThemeManifest => {
  const manifest = asRecord(value);
  const brand = asRecord(manifest.brand);
  const links = asRecord(manifest.links);
  const colors = asRecord(manifest.colors);
  const logos = readLegacyLogos(brand.logos);

  return {
    schemaVersion: 1,
    id: asString(manifest.id, 'default'),
    brand: {
      name: asString(brand.name, 'Application'),
      ...(asOptionalString(brand.environmentLabel)
        ? { environmentLabel: asOptionalString(brand.environmentLabel) }
        : {}),
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
      light: {
        carbonTheme: 'white',
        ...readLegacyColors(colors.light, DEFAULT_COLORS.light),
      },
      dark: {
        carbonTheme: 'g100',
        ...readLegacyColors(colors.dark, DEFAULT_COLORS.dark),
      },
    },
  };
};

const requireRecord = (
  value: unknown,
  path: string,
  issues: ThemeConfigIssue[],
): Record<string, unknown> => {
  if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }

  issues.push({
    path,
    code: value === undefined || value === null ? 'required' : 'invalid-type',
  });
  return {};
};

const requireString = (
  value: unknown,
  path: string,
  issues: ThemeConfigIssue[],
): string => {
  if (typeof value === 'string' && value.trim().length > 0) return value;

  issues.push({
    path,
    code:
      value === undefined ||
      value === null ||
      (typeof value === 'string' && value.trim().length === 0)
        ? 'required'
        : 'invalid-type',
  });
  return '';
};

const requireColor = (
  value: unknown,
  path: string,
  issues: ThemeConfigIssue[],
): string => {
  const color = requireString(value, path, issues);
  if (color && INTERNAL_PROPERTY_REFERENCE.test(color)) {
    issues.push({ path, code: 'internal-reference' });
  }
  return color;
};

const requireBoolean = (
  value: unknown,
  path: string,
  issues: ThemeConfigIssue[],
): boolean => {
  if (typeof value === 'boolean') return value;
  issues.push({
    path,
    code: value === undefined || value === null ? 'required' : 'invalid-type',
  });
  return false;
};

const requireMode = (
  value: unknown,
  path: string,
  issues: ThemeConfigIssue[],
): ThemeMode => {
  if (value === 'light' || value === 'dark' || value === 'system') return value;
  issues.push({
    path,
    code: value === undefined || value === null ? 'required' : 'unsupported',
  });
  return 'system';
};

const requireCarbonTheme = <T extends string>(
  value: unknown,
  allowed: Record<T, boolean>,
  path: string,
  issues: ThemeConfigIssue[],
): T => {
  if (typeof value === 'string' && Object.hasOwn(allowed, value)) {
    return value as T;
  }
  issues.push({
    path,
    code: value === undefined || value === null ? 'required' : 'unsupported',
  });
  return Object.keys(allowed)[0] as T;
};

const readVersion2Colors = <T extends LightCarbonTheme | DarkCarbonTheme>(
  value: unknown,
  path: 'colors.light' | 'colors.dark',
  allowedThemes: Record<T, boolean>,
  issues: ThemeConfigIssue[],
): { carbonTheme: T } & Record<ThemeColorField, string> => {
  const colors = requireRecord(value, path, issues);
  const result: Record<string, string> = {
    carbonTheme: requireCarbonTheme(
      colors.carbonTheme,
      allowedThemes,
      `${path}.carbonTheme`,
      issues,
    ),
  };

  for (const field of COLOR_FIELDS) {
    result[field] = requireColor(colors[field], `${path}.${field}`, issues);
  }

  return result as unknown as { carbonTheme: T } & Record<
    ThemeColorField,
    string
  >;
};

const readVersion2Logos = (
  value: unknown,
  issues: ThemeConfigIssue[],
): ThemeManifest['brand']['logos'] | undefined => {
  if (value === undefined) return undefined;

  const logos = requireRecord(value, 'brand.logos', issues);
  const full = requireRecord(logos.full, 'brand.logos.full', issues);
  const compact = requireRecord(logos.compact, 'brand.logos.compact', issues);

  return {
    full: {
      light: requireString(full.light, 'brand.logos.full.light', issues),
      dark: requireString(full.dark, 'brand.logos.full.dark', issues),
    },
    compact: {
      light: requireString(compact.light, 'brand.logos.compact.light', issues),
      dark: requireString(compact.dark, 'brand.logos.compact.dark', issues),
    },
  };
};

const parseVersion2 = (value: unknown): ThemeParseResult => {
  const issues: ThemeConfigIssue[] = [];
  const manifest = requireRecord(value, 'theme', issues);
  const brand = requireRecord(manifest.brand, 'brand', issues);
  const links = requireRecord(manifest.links, 'links', issues);
  const colors = requireRecord(manifest.colors, 'colors', issues);
  const logos = readVersion2Logos(brand.logos, issues);
  const environmentLabel =
    brand.environmentLabel === undefined
      ? undefined
      : requireString(brand.environmentLabel, 'brand.environmentLabel', issues);
  const favicon =
    brand.favicon === undefined
      ? undefined
      : requireString(brand.favicon, 'brand.favicon', issues);
  const light = readVersion2Colors(
    colors.light,
    'colors.light',
    themeContract.carbonThemes.light,
    issues,
  );
  const dark = readVersion2Colors(
    colors.dark,
    'colors.dark',
    themeContract.carbonThemes.dark,
    issues,
  );

  const theme: ThemeManifest = {
    schemaVersion: 2,
    id: requireString(manifest.id, 'id', issues),
    brand: {
      name: requireString(brand.name, 'brand.name', issues),
      ...(environmentLabel ? { environmentLabel } : {}),
      ...(logos ? { logos } : {}),
      ...(favicon ? { favicon } : {}),
    },
    links: {
      documentation: requireString(
        links.documentation,
        'links.documentation',
        issues,
      ),
      source: requireString(links.source, 'links.source', issues),
      support: requireString(links.support, 'links.support', issues),
      terms: requireString(links.terms, 'links.terms', issues),
      privacy: requireString(links.privacy, 'links.privacy', issues),
    },
    defaultMode: requireMode(manifest.defaultMode, 'defaultMode', issues),
    allowModeSelection: requireBoolean(
      manifest.allowModeSelection,
      'allowModeSelection',
      issues,
    ),
    colors: {
      light,
      dark,
    },
  };

  return issues.length > 0 ? { ok: false, issues } : { ok: true, theme };
};

export const parseThemeManifest = (value: unknown): ThemeParseResult => {
  const schemaVersion = asRecord(value).schemaVersion;

  if (schemaVersion === undefined || schemaVersion === 1) {
    return { ok: true, theme: parseVersion1(value) };
  }
  if (schemaVersion === themeContract.schemaVersion)
    return parseVersion2(value);

  return {
    ok: false,
    issues: [{ path: 'schemaVersion', code: 'unsupported' }],
  };
};
