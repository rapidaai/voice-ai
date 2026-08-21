import developmentConfig from '@/configs/config.development.json';
import { normalizeThemeManifest } from '@/theme/theme-config';

describe('theme config', () => {
  it('preserves arbitrary configured color and URL-like strings', () => {
    const configuredTheme = {
      ...developmentConfig.theme,
      brand: {
        ...developmentConfig.theme.brand,
        favicon: 'custom-asset://client/favicon',
      },
      links: {
        documentation: 'custom-docs://client/home',
        source: 'source repository',
        support: 'contact the client team',
        terms: 'client terms',
        privacy: 'client privacy',
      },
      colors: {
        light: {
          primary: 'brand(primary)',
          primaryHover: 'color-mix(in srgb, brand(primary), black 10%)',
          primaryActive: '#not-a-hex-value',
          onPrimary: 'var(--client-on-primary)',
        },
        dark: {
          primary: 'oklch(70% 0.2 250)',
          primaryHover: 'client-dark-hover',
          primaryActive: 'rgb(1 2 3)',
          onPrimary: 'transparent',
        },
      },
    };

    expect(normalizeThemeManifest(configuredTheme)).toMatchObject({
      brand: { favicon: 'custom-asset://client/favicon' },
      links: configuredTheme.links,
      colors: configuredTheme.colors,
    });
  });

  it('normalizes partial and missing structures with safe defaults', () => {
    expect(
      normalizeThemeManifest({
        brand: {
          logos: { full: { light: 'client-logo' } },
        },
        colors: { light: { primary: 'client-primary' } },
      }),
    ).toEqual({
      schemaVersion: 1,
      id: 'default',
      brand: {
        name: 'Application',
        logos: {
          full: { light: 'client-logo', dark: 'client-logo' },
          compact: { light: 'client-logo', dark: 'client-logo' },
        },
      },
      links: {
        documentation: '#',
        source: '#',
        support: '#',
        terms: '#',
        privacy: '#',
      },
      defaultMode: 'system',
      allowModeSelection: true,
      colors: {
        light: {
          primary: 'client-primary',
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
      },
    });

    expect(normalizeThemeManifest(undefined)).toMatchObject({
      id: 'default',
      brand: { name: 'Application' },
      defaultMode: 'system',
      allowModeSelection: true,
    });
  });
});
