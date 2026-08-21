import developmentConfig from '@/configs/config.development.json';
import {
  assertThemeManifest,
  getContrastRatio,
  getThemeManifestValidationErrors,
  isThemeManifest,
  ThemeManifestValidationError,
} from '@/theme/theme-config';
import { ThemeManifest } from '@/theme/types';

const theme = developmentConfig.theme as unknown as ThemeManifest;

describe('theme config', () => {
  it('accepts the configured enterprise theme and accessible interactions', () => {
    expect(isThemeManifest(theme)).toBe(true);

    for (const colors of Object.values(theme.colors)) {
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

  it('reports the exact field when a theme color is malformed', () => {
    const invalidTheme = {
      ...theme,
      colors: {
        ...theme.colors,
        dark: { ...theme.colors.dark, primaryHover: '#7ccdf7l' },
      },
    };

    expect(getThemeManifestValidationErrors(invalidTheme)).toContain(
      'CONFIG.theme.colors.dark.primaryHover must be a 6-digit hexadecimal color such as #0f62fe.',
    );
    expect(() => assertThemeManifest(invalidTheme)).toThrow(
      ThemeManifestValidationError,
    );
    expect(() => assertThemeManifest(invalidTheme)).toThrow(
      'CONFIG.theme.colors.dark.primaryHover must be a 6-digit hexadecimal color',
    );
  });

  it('reports the exact interaction field when WCAG contrast is insufficient', () => {
    const inaccessibleTheme = {
      ...theme,
      colors: {
        ...theme.colors,
        light: { ...theme.colors.light, primary: '#3186df' },
      },
    };

    expect(getThemeManifestValidationErrors(inaccessibleTheme)).toContain(
      'CONFIG.theme.colors.light.primary must have at least 4.5:1 contrast with the corresponding onPrimary color.',
    );
    expect(() => assertThemeManifest(inaccessibleTheme)).toThrow(
      'CONFIG.theme.colors.light.primary must have at least 4.5:1 contrast',
    );
  });

  it('rejects unsafe or incomplete configured themes', () => {
    expect(
      isThemeManifest({
        ...theme,
        brand: { ...theme.brand, favicon: '//attacker.example/favicon.ico' },
      }),
    ).toBe(false);

    expect(
      isThemeManifest({
        ...theme,
        brand: {
          ...theme.brand,
          logos: { full: theme.brand.logos?.full },
        },
      }),
    ).toBe(false);
  });
});
