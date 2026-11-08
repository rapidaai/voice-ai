import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import developmentConfig from '@/configs/config.development.json';
import productionConfig from '@/configs/config.production.json';
import { getConfig, withConfigDefaults } from '@/configs';
import { parseThemeManifest } from '@/theme/theme-config';

const dockerConfigs = [
  'config.community.json',
  'config.enterprise.json',
  'config.local.json',
  'config.local-knowledge.json',
];

const readDockerConfig = (configFile: string) =>
  JSON.parse(
    readFileSync(resolve(process.cwd(), '../docker/ui', configFile), 'utf8'),
  );

const luminance = (hex: string) => {
  const channels = hex
    .slice(1)
    .match(/.{2}/g)!
    .map(channel => parseInt(channel, 16) / 255)
    .map(channel =>
      channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4,
    );
  return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
};

const contrast = (left: string, right: string) => {
  const values = [luminance(left), luminance(right)].sort((a, b) => b - a);
  return (values[0] + 0.05) / (values[1] + 0.05);
};

describe('UI theme configuration', () => {
  it('selects and parses each environment theme', () => {
    expect(getConfig('development')).toEqual(developmentConfig);
    expect(getConfig('test')).toEqual(developmentConfig);
    expect(getConfig('production')).toEqual(productionConfig);
    expect(getConfig('development').deployment.webWidgetScriptUrl).toContain(
      '/public/scripts/app.min.js',
    );
    expect(getConfig('production').deployment.webWidgetScriptUrl).toContain(
      '/public/scripts/app.min.js',
    );
    expect(parseThemeManifest(developmentConfig.theme)).toMatchObject({
      ok: true,
    });
    expect(parseThemeManifest(productionConfig.theme)).toMatchObject({
      ok: true,
    });
  });

  it('uses the app mode when NODE_ENV stays production', () => {
    const originalEnv = process.env;
    try {
      process.env = {
        ...originalEnv,
        MODE: 'development',
        NODE_ENV: 'production',
      };
      delete process.env.VITE_MODE;

      expect(getConfig()).toEqual(developmentConfig);
    } finally {
      process.env = originalEnv;
    }
  });

  it('fills a domain-based web widget script URL for legacy configs', () => {
    const config = withConfigDefaults({
      ...developmentConfig,
      deployment: undefined,
      workspace: {
        ...developmentConfig.workspace,
        domain: 'client.example',
      },
    });

    expect(config.deployment.webWidgetScriptUrl).toBe(
      'https://cdn.client.example/public/scripts/app.min.js',
    );
  });

  it('uses the configured web origin for local legacy configs', () => {
    const config = withConfigDefaults({
      ...developmentConfig,
      deployment: undefined,
      connection: {
        ...developmentConfig.connection,
        web: 'http://localhost:8080/',
      },
      workspace: {
        ...developmentConfig.workspace,
        domain: 'localhost',
      },
    });

    expect(config.deployment.webWidgetScriptUrl).toBe(
      'http://localhost:8080/public/scripts/app.min.js',
    );
  });

  it.each(dockerConfigs)('%s contains a valid theme', configFile => {
    const config = readDockerConfig(configFile);

    expect(parseThemeManifest(config.theme)).toMatchObject({ ok: true });
  });

  it.each(dockerConfigs)('%s contains deployment config', configFile => {
    const config = readDockerConfig(configFile);

    expect(config.deployment.webWidgetScriptUrl).toContain(
      '/public/scripts/app.min.js',
    );
  });

  it.each([
    ['development', developmentConfig],
    ['production', productionConfig],
    ...dockerConfigs.map(configFile => [
      configFile,
      readDockerConfig(configFile),
    ]),
  ])('%s primary pairs meet WCAG AA contrast', (_name, config) => {
    for (const colors of Object.values(config.theme.colors) as Array<{
      primary: string;
      primaryHover: string;
      primaryActive: string;
      onPrimary: string;
    }>) {
      expect(contrast(colors.onPrimary, colors.primary)).toBeGreaterThanOrEqual(
        4.5,
      );
      expect(
        contrast(colors.onPrimary, colors.primaryHover),
      ).toBeGreaterThanOrEqual(4.5);
      expect(
        contrast(colors.onPrimary, colors.primaryActive),
      ).toBeGreaterThanOrEqual(4.5);
    }
  });
});
