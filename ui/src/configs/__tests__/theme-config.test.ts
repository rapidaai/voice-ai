import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import developmentConfig from '@/configs/config.development.json';
import productionConfig from '@/configs/config.production.json';
import { getConfig } from '@/configs';
import { isThemeManifest } from '@/theme/theme-config';

const dockerConfigs = [
  'config.community.json',
  'config.enterprise.json',
  'config.local.json',
  'config.local-knowledge.json',
];

describe('UI theme configuration', () => {
  it('selects the environment config with a complete theme', () => {
    expect(getConfig('development')).toEqual(developmentConfig);
    expect(getConfig('test')).toEqual(developmentConfig);
    expect(getConfig('production')).toEqual(productionConfig);
    expect(isThemeManifest(developmentConfig.theme)).toBe(true);
    expect(isThemeManifest(productionConfig.theme)).toBe(true);
  });

  it.each(dockerConfigs)('%s contains a deployable theme', configFile => {
    const config = JSON.parse(
      readFileSync(resolve(process.cwd(), '../docker/ui', configFile), 'utf8'),
    );

    expect(isThemeManifest(config.theme)).toBe(true);
  });
});
