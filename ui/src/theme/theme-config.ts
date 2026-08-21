import { ThemeManifest, ThemeMode } from './types';

export const THEME_STORAGE_KEY = 'ui-theme-mode';

const isThemeMode = (value: unknown): value is ThemeMode =>
  value === 'light' || value === 'dark' || value === 'system';

const isNonEmptyString = (value: unknown): value is string =>
  typeof value === 'string' && value.trim().length > 0;

const HEX_COLOR_PATTERN = /^#[0-9a-f]{6}$/i;

const isThemeColor = (value: unknown): value is string =>
  isNonEmptyString(value) && HEX_COLOR_PATTERN.test(value);

const APP_LINK_RULE =
  'must be a root-relative path or an absolute HTTPS or mailto URL.';
const ASSET_LINK_RULE =
  'must be a root-relative path or an absolute HTTPS URL.';
const COLOR_RULE = 'must be a 6-digit hexadecimal color such as #0f62fe.';
const CONTRAST_RULE =
  'must have at least 4.5:1 contrast with the corresponding onPrimary color.';

export class ThemeManifestValidationError extends Error {
  constructor(public readonly diagnostics: readonly string[]) {
    super(`CONFIG.theme is invalid. ${diagnostics.join(' ')}`);
    this.name = 'ThemeManifestValidationError';
  }
}

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

const isAppLink = (value: unknown): value is string => {
  if (!isNonEmptyString(value)) return false;
  if (value.startsWith('/') && !value.startsWith('//')) return true;

  try {
    return ['https:', 'mailto:'].includes(new URL(value).protocol);
  } catch {
    return false;
  }
};

const isAssetLink = (value: unknown): value is string => {
  if (!isNonEmptyString(value)) return false;
  if (value.startsWith('/') && !value.startsWith('//')) return true;

  try {
    return new URL(value).protocol === 'https:';
  } catch {
    return false;
  }
};

const validateRequiredString = (
  diagnostics: string[],
  field: string,
  value: unknown,
) => {
  if (!isNonEmptyString(value)) {
    diagnostics.push(`${field} must be a non-empty string.`);
  }
};

const validateLink = (
  diagnostics: string[],
  field: string,
  value: unknown,
  validator: (candidate: unknown) => candidate is string,
  rule: string,
) => {
  if (!validator(value)) {
    diagnostics.push(`${field} ${rule}`);
  }
};

const validateColors = (
  diagnostics: string[],
  mode: 'light' | 'dark',
  value: unknown,
) => {
  const colors =
    value && typeof value === 'object'
      ? (value as Partial<ThemeManifest['colors']['light']>)
      : undefined;
  const fields = [
    'primary',
    'primaryHover',
    'primaryActive',
    'onPrimary',
  ] as const;

  fields.forEach(field => {
    if (!isThemeColor(colors?.[field])) {
      diagnostics.push(`CONFIG.theme.colors.${mode}.${field} ${COLOR_RULE}`);
    }
  });

  if (
    !colors ||
    !isThemeColor(colors.primary) ||
    !isThemeColor(colors.primaryHover) ||
    !isThemeColor(colors.primaryActive) ||
    !isThemeColor(colors.onPrimary)
  ) {
    return;
  }

  (['primary', 'primaryHover', 'primaryActive'] as const).forEach(field => {
    if (getContrastRatio(colors[field], colors.onPrimary) < 4.5) {
      diagnostics.push(`CONFIG.theme.colors.${mode}.${field} ${CONTRAST_RULE}`);
    }
  });
};

export const getThemeManifestValidationErrors = (value: unknown): string[] => {
  if (!value || typeof value !== 'object') {
    return ['CONFIG.theme must be an object.'];
  }

  const diagnostics: string[] = [];
  const manifest = value as Partial<ThemeManifest>;
  const brand =
    manifest.brand && typeof manifest.brand === 'object'
      ? manifest.brand
      : undefined;
  const links =
    manifest.links && typeof manifest.links === 'object'
      ? manifest.links
      : undefined;
  const colors =
    manifest.colors && typeof manifest.colors === 'object'
      ? manifest.colors
      : undefined;

  if (manifest.schemaVersion !== 1) {
    diagnostics.push('CONFIG.theme.schemaVersion must be 1.');
  }
  validateRequiredString(diagnostics, 'CONFIG.theme.id', manifest.id);
  validateRequiredString(diagnostics, 'CONFIG.theme.brand.name', brand?.name);

  const logosValue = brand?.logos;
  if (logosValue !== undefined) {
    const logos =
      logosValue && typeof logosValue === 'object' ? logosValue : undefined;
    if (!logos) {
      diagnostics.push('CONFIG.theme.brand.logos must be an object.');
    } else {
      const full =
        logos.full && typeof logos.full === 'object' ? logos.full : undefined;
      const compact =
        logos.compact && typeof logos.compact === 'object'
          ? logos.compact
          : undefined;

      validateLink(
        diagnostics,
        'CONFIG.theme.brand.logos.full.light',
        full?.light,
        isAssetLink,
        ASSET_LINK_RULE,
      );
      validateLink(
        diagnostics,
        'CONFIG.theme.brand.logos.full.dark',
        full?.dark,
        isAssetLink,
        ASSET_LINK_RULE,
      );
      validateLink(
        diagnostics,
        'CONFIG.theme.brand.logos.compact.light',
        compact?.light,
        isAssetLink,
        ASSET_LINK_RULE,
      );
      validateLink(
        diagnostics,
        'CONFIG.theme.brand.logos.compact.dark',
        compact?.dark,
        isAssetLink,
        ASSET_LINK_RULE,
      );
    }
  }

  if (brand?.favicon !== undefined) {
    validateLink(
      diagnostics,
      'CONFIG.theme.brand.favicon',
      brand.favicon,
      isAssetLink,
      ASSET_LINK_RULE,
    );
  }

  (['documentation', 'source', 'support', 'terms', 'privacy'] as const).forEach(
    field => {
      validateLink(
        diagnostics,
        `CONFIG.theme.links.${field}`,
        links?.[field],
        isAppLink,
        APP_LINK_RULE,
      );
    },
  );

  if (!isThemeMode(manifest.defaultMode)) {
    diagnostics.push(
      'CONFIG.theme.defaultMode must be light, dark, or system.',
    );
  }
  if (typeof manifest.allowModeSelection !== 'boolean') {
    diagnostics.push('CONFIG.theme.allowModeSelection must be a boolean.');
  }

  validateColors(diagnostics, 'light', colors?.light);
  validateColors(diagnostics, 'dark', colors?.dark);

  return diagnostics;
};

export const isThemeManifest = (value: unknown): value is ThemeManifest => {
  return getThemeManifestValidationErrors(value).length === 0;
};

export const assertThemeManifest = (
  value: unknown,
): asserts value is ThemeManifest => {
  const diagnostics = getThemeManifestValidationErrors(value);
  if (diagnostics.length > 0) {
    throw new ThemeManifestValidationError(diagnostics);
  }
};
