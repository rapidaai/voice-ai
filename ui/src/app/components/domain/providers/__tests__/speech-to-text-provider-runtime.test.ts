import { Metadata, VaultCredential } from '@rapidaai/react';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '../speech-to-text/provider';
import { SPEECH_TO_TEXT_PROVIDER } from '@/providers';
import {
  loadProviderConfig,
  loadProviderData,
} from '@/providers/config-loader';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';

jest.mock(
  '@/app/components/domain/providers/provider-component-props',
  () => ({}),
);
jest.mock('@/app/components/domain/providers/config-renderer', () => ({
  ConfigRenderer: () => null,
}));

const createMetadata = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

const createCredential = (
  id: string,
  provider: string,
  apiCompatibility: string,
): VaultCredential => {
  const credential = new VaultCredential();
  credential.setId(id);
  credential.setProvider(provider);
  credential.setValue(Struct.fromJavaScript({ apiCompatibility }));
  return credential;
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

const getDeepgramModelLanguageData = (modelId: string): string[] => {
  const modelCatalog = loadProviderData(
    'deepgram',
    'speech-to-text-models.json',
  );
  const model = modelCatalog.find(item => item?.id === modelId);
  expect(model).toBeDefined();
  const languageParam = model?.config?.parameters?.find(
    (param: any) => param.key === 'listen.language',
  );
  expect(languageParam?.data).toBeDefined();
  return loadProviderData('deepgram', languageParam.data).map(
    item => item.code,
  );
};

const withValidCustomSttResponseRules = (source: Metadata[]): Metadata[] =>
  withMetadataValue(
    source,
    'listen.response_rules',
    JSON.stringify([
      {
        when: { frame: 'json', path: 'type', equals: 'final' },
        emit: { script: { $path: 'text' }, interim: false },
      },
    ]),
  );

describe('Speech-to-text provider runtime standard', () => {
  const configuredSttProviders = SPEECH_TO_TEXT_PROVIDER.filter(p =>
    Boolean(loadProviderConfig(p.code)?.stt),
  );
  const modelDrivenSttProviders = configuredSttProviders.filter(
    provider => provider.code !== 'custom-stt',
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

  it('allows custom-stt websocket credentials to keep binary audio request defaults', () => {
    const defaults = withValidCustomSttResponseRules(
      GetDefaultSpeechToTextIfInvalid('custom-stt', [
        createMetadata('rapida.credential_id', 'cred-custom-stt'),
      ]),
    );
    const err = ValidateSpeechToTextIfInvalid('custom-stt', defaults, [
      createCredential('cred-custom-stt', 'custom-stt', 'websocket_v1'),
    ]);

    expect(err).toBeUndefined();
  });

  it('rejects custom-stt http credentials when audio request rules use binary frames', () => {
    const defaults = withValidCustomSttResponseRules(
      GetDefaultSpeechToTextIfInvalid('custom-stt', [
        createMetadata('rapida.credential_id', 'cred-custom-stt'),
      ]),
    );
    const err = ValidateSpeechToTextIfInvalid('custom-stt', defaults, [
      createCredential('cred-custom-stt', 'custom-stt', 'http_v1'),
    ]);

    expect(err).toBe(
      'Custom STT HTTP v1 requires the first audio request rule to use send.frame "json".',
    );
  });

  it('uses selected credential compatibility instead of stale form metadata', () => {
    const defaults = withMetadataValue(
      withValidCustomSttResponseRules(
        GetDefaultSpeechToTextIfInvalid('custom-stt', [
          createMetadata('rapida.credential_id', 'cred-custom-stt'),
        ]),
      ),
      'rapida.credential_api_compatibility',
      'websocket_v1',
    );
    const err = ValidateSpeechToTextIfInvalid('custom-stt', defaults, [
      createCredential('cred-custom-stt', 'custom-stt', 'http_v1'),
    ]);

    expect(err).toBe(
      'Custom STT HTTP v1 requires the first audio request rule to use send.frame "json".',
    );
  });

  it('allows custom-stt http credentials when first audio request rule is json', () => {
    const defaults = withValidCustomSttResponseRules(
      GetDefaultSpeechToTextIfInvalid('custom-stt', [
        createMetadata('rapida.credential_id', 'cred-custom-stt'),
      ]),
    );
    const requestRules = JSON.stringify([
      {
        when: { packet: 'audio' },
        send: {
          frame: 'json',
          body: {
            audio: { $path: 'packet.audio.base64' },
            encoding: { $path: 'config.audio.encoding' },
            sample_rate: {
              $cast: 'number',
              value: { $path: 'config.audio.sample_rate' },
            },
          },
        },
      },
    ]);
    const err = ValidateSpeechToTextIfInvalid(
      'custom-stt',
      withMetadataValue(defaults, 'listen.request_rules', requestRules),
      [createCredential('cred-custom-stt', 'custom-stt', 'http_v1')],
    );

    expect(err).toBeUndefined();
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

  it('scopes deepgram language options by selected model', () => {
    const nova3GeneralLanguages =
      getDeepgramModelLanguageData('nova-3-general');
    expect(nova3GeneralLanguages).toEqual(
      expect.arrayContaining(['multi', 'as', 'mn', 'ps-AF', 'zh-Hant']),
    );

    const nova3MedicalLanguages =
      getDeepgramModelLanguageData('nova-3-medical');
    expect(nova3MedicalLanguages).toEqual(
      expect.arrayContaining(['multi', 'en-IE', 'en-IN', 'en-NZ']),
    );
    expect(nova3MedicalLanguages).not.toContain('as');
    expect(nova3MedicalLanguages).not.toContain('ja');

    const nova3PharmaLanguages =
      getDeepgramModelLanguageData('nova-3-pharma');
    expect(nova3PharmaLanguages).toEqual([
      'en',
      'en-US',
      'en-AU',
      'en-CA',
      'en-GB',
      'en-IE',
      'en-IN',
      'en-NZ',
    ]);

    const nova2PhonecallLanguages =
      getDeepgramModelLanguageData('nova-2-phonecall');
    expect(nova2PhonecallLanguages).toEqual(['en', 'en-US']);
  });

  it('rejects deepgram languages outside the selected model catalog', () => {
    const pharmaDefaults = GetDefaultSpeechToTextIfInvalid('deepgram', [
      createMetadata('rapida.credential_id', 'cred-deepgram'),
      createMetadata('listen.model', 'nova-3-pharma'),
    ]);

    expect(getMetadataValue(pharmaDefaults, 'listen.language')).toBe('en');

    const err = ValidateSpeechToTextIfInvalid(
      'deepgram',
      withMetadataValue(pharmaDefaults, 'listen.language', 'as'),
      ['cred-deepgram'],
    );

    expect(err).toBe(
      'Please provide a valid deepgram language for speech to text.',
    );
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

  it('microphone defaults apply non-empty overrides without replacing existing values', () => {
    const fullOverrideDefaults = GetDefaultMicrophoneConfig([], {
      'microphone.eos.provider': 'livekit_eos',
      'microphone.denoising.provider': 'none',
      'microphone.vad.provider': 'silero_vad',
      'microphone.barge_in_trigger': 'word',
      'microphone.vad.confidence': '0.8',
      'microphone.vad.start_secs': '0.4',
      'microphone.vad.stop_secs': '0.6',
      'microphone.vad.min_volume': '0.3',
      'microphone.eos.fallback_timeout': '700',
      'microphone.eos.threshold': '0.02',
      'microphone.eos.quick_timeout': '300',
      'microphone.eos.extended_timeout': '3500',
      'microphone.eos.model': 'en',
    });

    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.eos.provider'),
    ).toBe('livekit_eos');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.denoising.provider'),
    ).toBe('none');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.barge_in_trigger'),
    ).toBe('word');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.vad.confidence'),
    ).toBe('0.8');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.vad.start_secs'),
    ).toBe('0.4');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.vad.stop_secs'),
    ).toBe('0.6');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.vad.min_volume'),
    ).toBe('0.3');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.eos.fallback_timeout'),
    ).toBe('700');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.eos.threshold'),
    ).toBe('0.02');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.eos.quick_timeout'),
    ).toBe('300');
    expect(
      getMetadataValue(fullOverrideDefaults, 'microphone.eos.extended_timeout'),
    ).toBe('3500');
    expect(getMetadataValue(fullOverrideDefaults, 'microphone.eos.model')).toBe(
      'en',
    );

    const defaults = GetDefaultMicrophoneConfig(
      [
        createMetadata('microphone.vad.confidence', '0.4'),
        createMetadata('microphone.vad.barge_in_trigger', 'word'),
      ],
      {
        'microphone.vad.confidence': '0.9',
        'microphone.vad.start_secs': '0.5',
        'microphone.vad.stop_secs': '',
        'microphone.vad.barge_in_trigger': 'vad',
      },
    );

    expect(getMetadataValue(defaults, 'microphone.vad.confidence')).toBe('0.4');
    expect(getMetadataValue(defaults, 'microphone.vad.start_secs')).toBe('0.5');
    expect(getMetadataValue(defaults, 'microphone.vad.stop_secs')).toBe('');
    expect(getMetadataValue(defaults, 'microphone.barge_in_trigger')).toBe(
      'word',
    );
    expect(
      getMetadataValue(defaults, 'microphone.vad.barge_in_trigger'),
    ).toBeUndefined();
  });
});
