import { Metadata } from '@rapidaai/react';
import {
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '../provider';
import { SPEECH_TO_TEXT_PROVIDER } from '@/providers';
import {
  loadProviderConfig,
  loadProviderData,
} from '@/providers/config-loader';

jest.mock('@/app/components/providers', () => ({}));
jest.mock('@/app/components/providers/config-renderer', () => ({
  ConfigRenderer: () => null,
}));

const createMetadata = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

const cloneMetadata = (source: Metadata[]): Metadata[] =>
  source.map(m => createMetadata(m.getKey(), m.getValue()));

const normalizeMetadata = (source: Metadata[]): string[] =>
  source
    .map(m => `${m.getKey()}=${m.getValue()}`)
    .sort((a, b) => a.localeCompare(b));

const withCredential = (source: Metadata[]): Metadata[] => {
  const cloned = cloneMetadata(source);
  const credential = cloned.find(m => m.getKey() === 'rapida.credential_id');
  if (credential) {
    credential.setValue('test-credential');
    return cloned;
  }
  cloned.push(createMetadata('rapida.credential_id', 'test-credential'));
  return cloned;
};

const withMetadataValue = (
  source: Metadata[],
  key: string,
  value: string,
): Metadata[] => {
  const cloned = cloneMetadata(source);
  const item = cloned.find(m => m.getKey() === key);
  if (item) {
    item.setValue(value);
    return cloned;
  }
  cloned.push(createMetadata(key, value));
  return cloned;
};

const getMetadataValue = (
  source: Metadata[],
  key: string,
): string | undefined => source.find(m => m.getKey() === key)?.getValue();

describe('Speech-to-text provider runtime standard', () => {
  const configuredSttProviders = SPEECH_TO_TEXT_PROVIDER.filter(p =>
    Boolean(loadProviderConfig(p.code)?.stt),
  );
  const modelDrivenSttProviders = configuredSttProviders.filter(
    provider => provider.code !== 'custom-stt' && provider.code !== 'ringg',
  );

  it('all active speech-to-text providers are config-driven', () => {
    expect(configuredSttProviders.length).toBeGreaterThan(0);
    for (const provider of configuredSttProviders) {
      expect(loadProviderConfig(provider.code)?.stt).toBeDefined();
    }
  });

  it.each(modelDrivenSttProviders.map(p => p.code))(
    '%s stt config is model-driven with speech-to-text-models catalog',
    provider => {
      const sttConfig = loadProviderConfig(provider)?.stt;
      expect(sttConfig).toBeDefined();
      expect(sttConfig?.parameters.length).toBe(1);
      expect(sttConfig?.parameters[0].key).toBe('listen.model');
      expect(sttConfig?.parameters[0].type).toBe('dropdown');
      expect(sttConfig?.parameters[0].data).toBe('speech-to-text-models.json');
    },
  );

  it.each(modelDrivenSttProviders.map(p => p.code))(
    '%s model catalog carries per-model stt parameter config',
    provider => {
      const sttConfig = loadProviderConfig(provider)?.stt;
      const dataFile = sttConfig?.parameters[0]?.data;
      expect(dataFile).toBeDefined();

      const modelCatalog = loadProviderData(provider, dataFile!);
      expect(modelCatalog.length).toBeGreaterThan(0);
      for (const model of modelCatalog) {
        expect(Array.isArray(model?.config?.parameters)).toBe(true);
        expect(model.config.parameters.length).toBeGreaterThan(0);
      }
    },
  );

  it.each(modelDrivenSttProviders.map(p => p.code))(
    '%s defaults + validation are stable with model-level parameters',
    provider => {
      const seed = [
        createMetadata('custom.key', 'custom'),
        createMetadata('rapida.credential_id', 'seed-cred'),
      ];
      const defaults = GetDefaultSpeechToTextIfInvalid(
        provider,
        cloneMetadata(seed),
      );

      expect(defaults.some(m => m.getKey() === 'listen.model')).toBe(true);
      expect(defaults.some(m => m.getKey() === 'rapida.credential_id')).toBe(
        true,
      );

      const validated = ValidateSpeechToTextIfInvalid(
        provider,
        withCredential(defaults),
        ['test-credential'],
      );
      expect(validated).toBeUndefined();
    },
  );

  it('custom-stt uses websocket contract fields instead of model catalog wiring', () => {
    const sttConfig = loadProviderConfig('custom-stt')?.stt;
    expect(sttConfig).toBeDefined();
    const keys = sttConfig?.parameters.map(param => param.key) ?? [];

    expect(keys).toEqual(
      expect.arrayContaining([
        'listen.audio.encoding',
        'listen.audio.sample_rate',
        'listen.query_params',
        'listen.request_rules',
        'listen.response_rules',
      ]),
    );
    expect(sttConfig?.parameters[0].data).toBeUndefined();
  });

  it('rejects stale credential ids that do not belong to selected provider', () => {
    const defaults = GetDefaultSpeechToTextIfInvalid('deepgram', [
      createMetadata('rapida.credential_id', 'cred-from-other-provider'),
    ]);
    const err = ValidateSpeechToTextIfInvalid('deepgram', defaults, [
      'cred-deepgram-1',
      'cred-deepgram-2',
    ]);

    expect(err).toBe('Please select a valid deepgram credential.');
  });

  it('validates model ids against stt model catalog', () => {
    const defaults = GetDefaultSpeechToTextIfInvalid('deepgram', [
      createMetadata('rapida.credential_id', 'cred-deepgram'),
    ]);
    const invalidModel = withMetadataValue(
      defaults,
      'listen.model',
      'invalid-model-id',
    );
    const err = ValidateSpeechToTextIfInvalid('deepgram', invalidModel, [
      'cred-deepgram',
    ]);

    expect(err).toBe(
      'Please provide a valid deepgram model for speech to text.',
    );
  });

  it('supports deepgram model switch with parameter changes', () => {
    const defaults = GetDefaultSpeechToTextIfInvalid('deepgram', [
      createMetadata('rapida.credential_id', 'cred-deepgram'),
    ]);
    const defaultModel = getMetadataValue(defaults, 'listen.model');

    const modelCatalog = loadProviderData(
      'deepgram',
      'speech-to-text-models.json',
    );
    const alternateModel = modelCatalog.find(
      m => m?.id && m.id !== defaultModel,
    )?.id as string | undefined;
    expect(alternateModel).toBeDefined();

    const updated = GetDefaultSpeechToTextIfInvalid(
      'deepgram',
      withMetadataValue(
        withMetadataValue(defaults, 'listen.model', alternateModel!),
        'listen.threshold',
        '0.8',
      ),
    );

    expect(getMetadataValue(updated, 'listen.model')).toBe(alternateModel);
    expect(getMetadataValue(updated, 'listen.threshold')).toBe('0.8');

    const err = ValidateSpeechToTextIfInvalid('deepgram', updated, [
      'cred-deepgram',
    ]);
    expect(err).toBeUndefined();
  });

  it('unknown provider remains no-op when no config exists', () => {
    const seed = [createMetadata('custom.key', 'custom')];
    expect(
      normalizeMetadata(
        GetDefaultSpeechToTextIfInvalid(
          'unknown-provider',
          cloneMetadata(seed),
        ),
      ),
    ).toEqual(normalizeMetadata(seed));
    expect(
      ValidateSpeechToTextIfInvalid('unknown-provider', []),
    ).toBeUndefined();
  });
});

describe('Ringg STT — config vs original', () => {
  const config = loadProviderConfig('ringg')!;

  it('loads a language-only STT config with microphone prefix', () => {
    const sttConfig = config.stt;
    expect(sttConfig).toBeDefined();
    expect(sttConfig?.preservePrefix).toBe('microphone.');
    const keys = sttConfig?.parameters.map(param => param.key) ?? [];
    expect(keys).toEqual(expect.arrayContaining(['listen.language']));
    expect(keys).not.toContain('listen.model');
    expect(sttConfig?.parameters[0].type).toBe('input');
    expect(sttConfig?.parameters[0].default).toBe('en');
  });

  it('preserves microphone metadata and validates credentials', () => {
    const seed = [
      createMetadata('rapida.credential_id', 'seed-cred'),
      createMetadata('microphone.volume', '0.8'),
    ];

    const defaults = GetDefaultSpeechToTextIfInvalid('ringg', cloneMetadata(seed));
    expect(getMetadataValue(defaults, 'listen.language')).toBe('en');
    expect(getMetadataValue(defaults, 'microphone.volume')).toBe('0.8');
    expect(getMetadataValue(defaults, 'rapida.credential_id')).toBe('seed-cred');

    expect(
      ValidateSpeechToTextIfInvalid('ringg', defaults, ['seed-cred']),
    ).toBeUndefined();

    const missingCred = ValidateSpeechToTextIfInvalid('ringg', [
      createMetadata('listen.language', 'en'),
    ]);
    expect(missingCred).toBe('Please provide a valid ringg credential.');
  });

  it('keeps an existing language override', () => {
    const defaults = GetDefaultSpeechToTextIfInvalid('ringg', [
      createMetadata('rapida.credential_id', 'seed-cred'),
      createMetadata('listen.language', 'hi'),
    ]);
    expect(getMetadataValue(defaults, 'listen.language')).toBe('hi');
  });
});
