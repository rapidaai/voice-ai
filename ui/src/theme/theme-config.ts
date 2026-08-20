import { ThemeManifest, ThemeMode } from './types';

export const THEME_STORAGE_KEY = 'ui-theme-mode';

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

export const assertThemeManifest = (
  value: unknown,
): asserts value is ThemeManifest => {
  if (!isThemeManifest(value)) {
    throw new Error(
      'CONFIG.theme is invalid. Update the selected UI config with a complete ThemeManifest.',
    );
  }
};
