/**
 * Per-provider comparison tests for TTS.
 *
 * Each test verifies that config-driven defaults + validation produce
 * the same structural output (keys, default values, valid/invalid decisions)
 * as the original hand-written constant.ts functions.
 */
import { Metadata } from '@rapidaai/react';
import {
  loadProviderConfig,
  loadProviderData,
  resolveCategoryParameters,
} from '../config-loader';
import { getDefaultsFromConfig, validateFromConfig } from '../config-defaults';

function createMetadata(key: string, value: string): Metadata {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
}

function findMeta(arr: Metadata[], key: string): string | undefined {
  return arr.find(m => m.getKey() === key)?.getValue();
}

const cred = () => createMetadata('rapida.credential_id', 'test-cred');

describe('Groq TTS — config vs original', () => {
  const config = loadProviderConfig('groq')!;

  it('config has tts section', () => {
    expect(config.tts).toBeDefined();
  });

  it('produces defaults with no pre-set values (no defaults in original)', () => {
    const result = getDefaultsFromConfig(config, 'tts', [], 'groq');
    // Original: no defaults for model or voice
    // Config-driven: no defaults set in tts.json either
    // Only keys present should be from the parameters (with undefined values if no defaults)
    const keys = result.map(m => m.getKey());
    expect(keys).not.toContain('speak.language'); // groq TTS has no language param
  });

  it('preserves speaker.* params', () => {
    const existing = [createMetadata('speaker.rate', '1.2')];
    const result = getDefaultsFromConfig(config, 'tts', existing, 'groq');
    expect(findMeta(result, 'speaker.rate')).toBe('1.2');
  });

  it('validates: missing credential returns error', () => {
    const result = validateFromConfig(config, 'tts', 'groq', []);
    expect(result).toBeDefined();
    expect(result).toContain('credential');
  });

  it('validates: voice.id accepts any value (strict:false)', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'playai-tts'),
      createMetadata('speak.voice.id', 'any-custom-voice-id'),
    ];
    expect(validateFromConfig(config, 'tts', 'groq', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'nonexistent-model'),
      createMetadata('speak.voice.id', 'some-voice'),
    ];
    expect(validateFromConfig(config, 'tts', 'groq', opts)).toBe(
      'Please select valid model for text to speech.',
    );
  });

  it('validates: empty voice.id returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'playai-tts'),
      // no voice.id
    ];
    expect(validateFromConfig(config, 'tts', 'groq', opts)).toBe(
      'Please select a valid voice ID for text to speech.',
    );
  });
});

describe('Speechmatics TTS — config vs original', () => {
  const config = loadProviderConfig('speechmatics')!;

  it('config has tts section', () => {
    expect(config.tts).toBeDefined();
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('speak.voice.id', 'some-voice'),
      createMetadata('speak.language', 'en'),
    ];
    expect(
      validateFromConfig(config, 'tts', 'speechmatics', opts),
    ).toBeUndefined();
  });

  it('validates: voice.id accepts any value (strict:false)', () => {
    const opts = [
      cred(),
      createMetadata('speak.voice.id', 'custom-voice-xyz'),
      createMetadata('speak.language', 'en'),
    ];
    expect(
      validateFromConfig(config, 'tts', 'speechmatics', opts),
    ).toBeUndefined();
  });

  it('validates: invalid language returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.voice.id', 'some-voice'),
      createMetadata('speak.language', 'invalid-lang'),
    ];
    expect(validateFromConfig(config, 'tts', 'speechmatics', opts)).toBe(
      'Please select valid language for text to speech.',
    );
  });
});

describe('Nvidia TTS — config vs original', () => {
  const config = loadProviderConfig('nvidia')!;

  it('config has tts section', () => {
    expect(config.tts).toBeDefined();
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('speak.voice.id', 'some-voice'),
      createMetadata('speak.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'tts', 'nvidia', opts)).toBeUndefined();
  });

  it('validates: voice.id accepts any value (strict:false)', () => {
    const opts = [
      cred(),
      createMetadata('speak.voice.id', 'custom-voice'),
      createMetadata('speak.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'tts', 'nvidia', opts)).toBeUndefined();
  });
});

describe('AWS TTS — config vs original', () => {
  const config = loadProviderConfig('aws')!;

  it('config has tts section', () => {
    expect(config.tts).toBeDefined();
  });

  it('produces defaults with expected keys', () => {
    const result = getDefaultsFromConfig(config, 'tts', [], 'aws');
    const keys = result.map(m => m.getKey()).sort();
    // AWS TTS has no defaults in legacy path either
    expect(keys).toEqual([]);
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'standard'),
      createMetadata('speak.voice.id', 'some-voice'),
      createMetadata('speak.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'tts', 'aws', opts)).toBeUndefined();
  });

  it('validates: voice.id accepts any value (strict:false)', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'standard'),
      createMetadata('speak.voice.id', 'custom-voice'),
      createMetadata('speak.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'tts', 'aws', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'nonexistent'),
      createMetadata('speak.voice.id', 'some-voice'),
      createMetadata('speak.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'tts', 'aws', opts)).toBe(
      'Please select valid model for text to speech.',
    );
  });

  it('validates: invalid language returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'standard'),
      createMetadata('speak.voice.id', 'some-voice'),
      createMetadata('speak.language', 'invalid'),
    ];
    expect(validateFromConfig(config, 'tts', 'aws', opts)).toBe(
      'Please select valid language for text to speech.',
    );
  });
});

describe('OpenAI TTS — config vs original', () => {
  const config = loadProviderConfig('openai')!;

  it('config has tts section', () => {
    expect(config.tts).toBeDefined();
  });

  it('produces the same default values as original', () => {
    const result = getDefaultsFromConfig(config, 'tts', [], 'openai');
    expect(findMeta(result, 'speak.model')).toBe('gpt-4o-mini-tts');
    expect(findMeta(result, 'speak.voice.id')).toBe('alloy');
    expect(findMeta(result, 'speak.language')).toBe('en');
  });

  it('preserves existing valid values', () => {
    const existing = [
      cred(),
      createMetadata('speak.model', 'tts-1'),
      createMetadata('speak.voice.id', 'echo'),
      createMetadata('speak.language', 'fr'),
    ];
    const result = getDefaultsFromConfig(config, 'tts', existing, 'openai');
    expect(findMeta(result, 'speak.model')).toBe('tts-1');
    expect(findMeta(result, 'speak.voice.id')).toBe('echo');
    expect(findMeta(result, 'speak.language')).toBe('fr');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'gpt-4o-mini-tts'),
      createMetadata('speak.voice.id', 'alloy'),
      createMetadata('speak.language', 'en'),
    ];
    expect(validateFromConfig(config, 'tts', 'openai', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'nonexistent'),
      createMetadata('speak.voice.id', 'alloy'),
      createMetadata('speak.language', 'en'),
    ];
    expect(validateFromConfig(config, 'tts', 'openai', opts)).toBe(
      'Please select valid model for text to speech.',
    );
  });

  it('validates: invalid voice returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'tts-1'),
      createMetadata('speak.voice.id', 'nonexistent-voice'),
      createMetadata('speak.language', 'en'),
    ];
    expect(validateFromConfig(config, 'tts', 'openai', opts)).toBe(
      'Please select valid voice for text to speech.',
    );
  });

  it('validates: invalid language returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'tts-1'),
      createMetadata('speak.voice.id', 'alloy'),
      createMetadata('speak.language', 'invalid'),
    ];
    expect(validateFromConfig(config, 'tts', 'openai', opts)).toBe(
      'Please select valid language option for text to speech.',
    );
  });

  it('preserves speaker.* params', () => {
    const existing = [createMetadata('speaker.speed', '1.5')];
    const result = getDefaultsFromConfig(config, 'tts', existing, 'openai');
    expect(findMeta(result, 'speaker.speed')).toBe('1.5');
  });
});

describe('Smallest TTS — config resolution', () => {
  const config = loadProviderConfig('smallest')!;

  it('config has tts section', () => {
    expect(config.tts).toBeDefined();
  });

  it('produces the expected default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'tts', [], 'smallest');
    expect(findMeta(result, 'speak.model')).toBe('lightning_v3.1');
    expect(findMeta(result, 'speak.voice.id')).toBe('magnus');
    expect(findMeta(result, 'speak.language')).toBe('en');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'lightning_v3.1'),
      createMetadata('speak.voice.id', 'magnus'),
      createMetadata('speak.language', 'en'),
    ];
    expect(validateFromConfig(config, 'tts', 'smallest', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'nonexistent'),
      createMetadata('speak.voice.id', 'magnus'),
      createMetadata('speak.language', 'en'),
    ];
    expect(validateFromConfig(config, 'tts', 'smallest', opts)).toBe(
      'Please select a valid model for text to speech.',
    );
  });

  it('validates: voice.id accepts any value (strict:false, customValue:true)', () => {
    const opts = [
      cred(),
      createMetadata('speak.model', 'lightning_v3.1'),
      createMetadata('speak.voice.id', 'voice_custom-clone-id'),
      createMetadata('speak.language', 'en'),
    ];
    expect(validateFromConfig(config, 'tts', 'smallest', opts)).toBeUndefined();
  });

  it('scopes the default voice to the selected model', () => {
    const withStandard = getDefaultsFromConfig(
      config,
      'tts',
      [createMetadata('speak.model', 'lightning_v3.1')],
      'smallest',
    );
    expect(findMeta(withStandard, 'speak.voice.id')).toBe('magnus');

    const withPro = getDefaultsFromConfig(
      config,
      'tts',
      [createMetadata('speak.model', 'lightning_v3.1_pro')],
      'smallest',
    );
    expect(findMeta(withPro, 'speak.voice.id')).toBe('meher');
  });

  it('resolves a distinct, non-overlapping voice catalog per model', () => {
    const standardParams = resolveCategoryParameters(
      'smallest',
      'tts',
      config.tts!,
      [createMetadata('speak.model', 'lightning_v3.1')],
    );
    const proParams = resolveCategoryParameters(
      'smallest',
      'tts',
      config.tts!,
      [createMetadata('speak.model', 'lightning_v3.1_pro')],
    );

    const standardVoiceParam = standardParams.find(
      p => p.key === 'speak.voice.id',
    );
    const proVoiceParam = proParams.find(p => p.key === 'speak.voice.id');
    expect(standardVoiceParam?.data).toBe(
      'text-to-speech-voices-lightning-v3-1.json',
    );
    expect(proVoiceParam?.data).toBe(
      'text-to-speech-voices-lightning-v3-1-pro.json',
    );

    const standardVoices = loadProviderData(
      'smallest',
      standardVoiceParam!.data!,
    ) as Array<{ voice_id: string }>;
    const proVoices = loadProviderData(
      'smallest',
      proVoiceParam!.data!,
    ) as Array<{ voice_id: string }>;
    const standardIds = new Set(standardVoices.map(v => v.voice_id));
    const proIds = new Set(proVoices.map(v => v.voice_id));

    // Guard against a missing/invalid catalog silently resolving to `[]`,
    // which would make every assertion below vacuously true.
    expect(standardVoices.length).toBeGreaterThan(0);
    expect(proVoices.length).toBeGreaterThan(0);

    // No voice should be pickable under both models — that's exactly the
    // pairing the live API rejects (see smallest_integration_test.go's
    // TestSmallestTTSVoiceModelMismatch).
    expect(standardIds.has('meher')).toBe(false);
    expect(proIds.has('magnus')).toBe(false);
    expect([...standardIds].some(id => proIds.has(id))).toBe(false);
  });
});

describe('Cartesia TTS — config resolution', () => {
  const config = loadProviderConfig('cartesia')!;

  const resolveModelLanguageCodes = (modelId: string): string[] => {
    const params = resolveCategoryParameters('cartesia', 'tts', config.tts!, [
      createMetadata('speak.model', modelId),
    ]);
    const languageParam = params.find(param => param.key === 'speak.language');
    expect(languageParam?.data).toBeDefined();

    return loadProviderData('cartesia', languageParam!.data!).map(
      language => language.code,
    );
  };

  it('defaults to Sonic 3.6 with shared voices and model-specific language options', () => {
    const result = getDefaultsFromConfig(config, 'tts', [], 'cartesia');

    expect(findMeta(result, 'speak.model')).toBe('sonic-3.6');
    expect(findMeta(result, 'speak.language')).toBe('en');

    const params = resolveCategoryParameters('cartesia', 'tts', config.tts!, [
      createMetadata('speak.model', 'sonic-3.6'),
    ]);
    const voiceParam = params.find(param => param.key === 'speak.voice.id');
    const languageParam = params.find(param => param.key === 'speak.language');

    expect(voiceParam?.data).toBe('voices.json');
    expect(languageParam?.data).toBe('languages-sonic-3-6.json');
  });

  it('scopes Cartesia TTS languages to the selected model', () => {
    const sonic36 = resolveModelLanguageCodes('sonic-3.6');
    expect(sonic36).toEqual(expect.arrayContaining(['or', 'ur']));

    const sonic35 = resolveModelLanguageCodes('sonic-3.5-2026-05-04');
    expect(sonic35).toEqual(expect.arrayContaining(['pa', 'mr']));
    expect(sonic35).not.toContain('or');
    expect(sonic35).not.toContain('ur');

    const sonic2 = resolveModelLanguageCodes('sonic-2');
    expect(sonic2).toEqual(['en', 'fr', 'de', 'es', 'pt', 'zh', 'ja', 'ko']);
  });

  it('validates Cartesia TTS languages against the selected model catalog', () => {
    expect(
      validateFromConfig(config, 'tts', 'cartesia', [
        cred(),
        createMetadata('speak.model', 'sonic-3.6'),
        createMetadata('speak.voice.id', 'custom-voice'),
        createMetadata('speak.language', 'ur'),
      ]),
    ).toBeUndefined();

    expect(
      validateFromConfig(config, 'tts', 'cartesia', [
        cred(),
        createMetadata('speak.model', 'sonic-2'),
        createMetadata('speak.voice.id', 'custom-voice'),
        createMetadata('speak.language', 'ur'),
      ]),
    ).toBe('Please select a valid language for text to speech.');
  });

  it('keeps only existing old aliases and labels sunsetted Cartesia TTS models', () => {
    const models = loadProviderData('cartesia', 'text-to-speech-models.json');
    const modelIds = models.map(model => model.id);

    expect(modelIds).toEqual([
      'sonic-3.6',
      'sonic-3.6-2026-08-27',
      'sonic-preview',
      'sonic-3.5-2026-05-04',
      'sonic-3',
      'sonic-2',
      'sonic-turbo',
      'sonic',
    ]);
    expect(models.find(model => model.id === 'sonic-2')?.status).toBe(
      'Sunsetted',
    );
    expect(models.find(model => model.id === 'sonic-turbo')?.status).toBe(
      'Sunsetted',
    );
    expect(models.find(model => model.id === 'sonic')?.status).toBe(
      'Sunsetted',
    );
  });
});
