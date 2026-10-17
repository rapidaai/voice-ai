import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import developmentConfig from '@/configs/config.development.json';
import productionConfig from '@/configs/config.production.json';
import { getConfig } from '@/configs';
import { normalizeThemeManifest } from '@/theme/theme-config';

const dockerConfigs = [
  'config.community.json',
  'config.enterprise.json',
  'config.local.json',
  'config.local-knowledge.json',
];

describe('UI theme configuration', () => {
  it('selects and normalizes each environment theme', () => {
    expect(getConfig('development')).toEqual(developmentConfig);
    expect(getConfig('test')).toEqual(developmentConfig);
    expect(getConfig('production')).toEqual(productionConfig);
    expect(normalizeThemeManifest(developmentConfig.theme)).toEqual(
      developmentConfig.theme,
    );
    expect(normalizeThemeManifest(productionConfig.theme)).toEqual(
      productionConfig.theme,
    );
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

  it.each(dockerConfigs)('%s contains a normalizable theme', configFile => {
    const config = JSON.parse(
      readFileSync(resolve(process.cwd(), '../docker/ui', configFile), 'utf8'),
    );

    expect(normalizeThemeManifest(config.theme)).toEqual(config.theme);
  });
});
