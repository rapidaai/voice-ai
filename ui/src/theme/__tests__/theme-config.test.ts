import developmentConfig from '@/configs/config.development.json';
import { parseThemeManifest } from '@/theme/theme-config';

describe('theme config', () => {
  it('parses a complete version 2 theme without changing external CSS values', () => {
    const configuredTheme = {
      ...developmentConfig.theme,
      schemaVersion: 2,
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
          carbonTheme: 'g10',
          primary: 'color-mix(in srgb, var(--tenant-action), black 10%)',
          primaryHover: '#not-a-hex-value',
          primaryActive: 'oklch(60% 0.2 250)',
          onPrimary: 'var(--tenant-on-action)',
        },
        dark: {
          carbonTheme: 'g90',
          primary: 'oklch(70% 0.2 250)',
          primaryHover: 'client-dark-hover',
          primaryActive: 'rgb(1 2 3)',
          onPrimary: 'transparent',
        },
      },
    };

    expect(parseThemeManifest(configuredTheme)).toEqual({
      ok: true,
      theme: configuredTheme,
    });
  });

  it('retains version 1 fallbacks for partial and missing structures', () => {
    const result = parseThemeManifest({
      brand: {
        logos: { full: { light: 'client-logo' } },
      },
      colors: { light: { primary: 'client-primary' } },
    });

    expect(result).toEqual({
      ok: true,
      theme: {
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
            carbonTheme: 'white',
            primary: 'client-primary',
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
        },
      },
    });

    expect(parseThemeManifest(undefined)).toMatchObject({
      ok: true,
      theme: {
        schemaVersion: 1,
        id: 'default',
        brand: { name: 'Application' },
        defaultMode: 'system',
        allowModeSelection: true,
      },
    });
    expect(parseThemeManifest({})).toMatchObject({
      ok: true,
      theme: { schemaVersion: 1, id: 'default' },
    });
  });

  it.each([
    {
      name: 'unknown schema',
      theme: { ...developmentConfig.theme, schemaVersion: 3 },
      issue: { path: 'schemaVersion', code: 'unsupported' },
    },
    {
      name: 'missing version 2 role',
      theme: {
        ...developmentConfig.theme,
        schemaVersion: 2,
        colors: {
          ...developmentConfig.theme.colors,
          light: {
            carbonTheme: 'white',
            primary: '#0353e9',
            primaryHover: '#002d9c',
            onPrimary: '#ffffff',
          },
        },
      },
      issue: { path: 'colors.light.primaryActive', code: 'required' },
    },
    {
      name: 'empty version 2 role',
      theme: {
        ...developmentConfig.theme,
        schemaVersion: 2,
        colors: {
          ...developmentConfig.theme.colors,
          light: {
            ...developmentConfig.theme.colors.light,
            primaryActive: '   ',
          },
        },
      },
      issue: { path: 'colors.light.primaryActive', code: 'required' },
    },
    {
      name: 'invalid Carbon preset',
      theme: {
        ...developmentConfig.theme,
        schemaVersion: 2,
        colors: {
          ...developmentConfig.theme.colors,
          light: {
            ...developmentConfig.theme.colors.light,
            carbonTheme: 'g100',
          },
        },
      },
      issue: { path: 'colors.light.carbonTheme', code: 'unsupported' },
    },
    {
      name: 'internal Carbon reference',
      theme: {
        ...developmentConfig.theme,
        schemaVersion: 2,
        colors: {
          ...developmentConfig.theme.colors,
          dark: {
            ...developmentConfig.theme.colors.dark,
            carbonTheme: 'g100',
            primary: 'var(--cds-button-primary)',
          },
        },
      },
      issue: { path: 'colors.dark.primary', code: 'internal-reference' },
    },
    {
      name: 'internal brand reference',
      theme: {
        ...developmentConfig.theme,
        schemaVersion: 2,
        colors: {
          ...developmentConfig.theme.colors,
          dark: {
            ...developmentConfig.theme.colors.dark,
            carbonTheme: 'g100',
            onPrimary: 'var(--brand-on-primary)',
          },
        },
      },
      issue: { path: 'colors.dark.onPrimary', code: 'internal-reference' },
    },
  ])('rejects $name', ({ theme, issue }) => {
    const result = parseThemeManifest(theme);

    expect(result).toMatchObject({
      ok: false,
      issues: expect.arrayContaining([issue]),
    });
  });

  it.each(['constructor', 'toString'])(
    'rejects inherited Carbon preset name %s',
    carbonTheme => {
      const result = parseThemeManifest({
        ...developmentConfig.theme,
        schemaVersion: 2,
        colors: {
          ...developmentConfig.theme.colors,
          light: {
            ...developmentConfig.theme.colors.light,
            carbonTheme,
          },
        },
      });

      expect(result).toMatchObject({
        ok: false,
        issues: expect.arrayContaining([
          { path: 'colors.light.carbonTheme', code: 'unsupported' },
        ]),
      });
    },
  );
});
