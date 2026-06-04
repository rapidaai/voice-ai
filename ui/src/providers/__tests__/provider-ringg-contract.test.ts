// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

import { Metadata } from '@rapidaai/react';
import { loadProviderConfig } from '../config-loader';
import { getDefaultsFromConfig, validateFromConfig } from '../config-defaults';

function createMetadata(key: string, value: string): Metadata {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
}

function findMeta(source: Metadata[], key: string): string | undefined {
  return source.find(item => item.getKey() === key)?.getValue();
}

function getProvider(list: any[]) {
  return list.find(provider => provider.code === 'ringg');
}

describe('Ringg STT provider catalog', () => {
  it('exists in development and production with API key credential field', () => {
    const developmentProviders = require('../provider.development.json');
    const productionProviders = require('../provider.production.json');

    const developmentProvider = getProvider(developmentProviders);
    const productionProvider = getProvider(productionProviders);

    for (const provider of [developmentProvider, productionProvider]) {
      expect(provider).toBeDefined();
      expect(provider.featureList).toEqual(
        expect.arrayContaining(['stt', 'external']),
      );
      expect(provider.configurations.map((config: any) => config.name)).toEqual(
        expect.arrayContaining(['key']),
      );
      const apiKeyConfig = provider.configurations.find(
        (config: any) => config.name === 'key',
      );
      expect(apiKeyConfig?.label).toBe('API Key');
    }
  });
});

describe('Ringg STT config contract', () => {
  const config = loadProviderConfig('ringg')!;

  it('loads a language-only STT config with microphone prefix', () => {
    expect(config.stt).toBeDefined();
    expect(config.stt?.preservePrefix).toBe('microphone.');
    const keys = config.stt?.parameters.map(param => param.key) ?? [];
    expect(keys).toEqual(expect.arrayContaining(['listen.language']));
    expect(keys).not.toContain('listen.model');
    expect(config.stt?.parameters[0].type).toBe('input');
    expect(config.stt?.parameters[0].default).toBe('en');
  });

  it('applies defaults and preserves microphone metadata', () => {
    const defaults = getDefaultsFromConfig(
      config,
      'stt',
      [
        createMetadata('rapida.credential_id', 'cred-ringg-1'),
        createMetadata('microphone.volume', '0.8'),
      ],
      'ringg',
    );

    expect(findMeta(defaults, 'listen.language')).toBe('en');
    expect(findMeta(defaults, 'microphone.volume')).toBe('0.8');
    expect(findMeta(defaults, 'rapida.credential_id')).toBe('cred-ringg-1');
  });

  it('accepts valid credential input without language', () => {
    expect(
      validateFromConfig(config, 'stt', 'ringg', [
        createMetadata('rapida.credential_id', 'cred-ringg-1'),
      ]),
    ).toBeUndefined();
  });

  it('rejects missing credential', () => {
    expect(validateFromConfig(config, 'stt', 'ringg', [])).toBe(
      'Please provide a valid ringg credential.',
    );
  });
});
