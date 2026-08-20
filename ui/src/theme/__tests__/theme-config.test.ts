import developmentConfig from '@/configs/config.development.json';
import {
  assertThemeManifest,
  getContrastRatio,
  isThemeManifest,
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

  it('rejects unsafe, incomplete, or inaccessible configured themes', () => {
    expect(() =>
      assertThemeManifest({
        ...theme,
        brand: { ...theme.brand, favicon: '//attacker.example/favicon.ico' },
        colors: {
          ...theme.colors,
          dark: { ...theme.colors.dark, onPrimary: '#ffffff' },
        },
      }),
    ).toThrow('CONFIG.theme is invalid');

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
