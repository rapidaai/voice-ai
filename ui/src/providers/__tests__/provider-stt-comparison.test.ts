/**
 * Per-provider comparison tests for STT.
 *
 * Each test verifies that config-driven defaults + validation produce
 * the same structural output (keys, default values, valid/invalid decisions)
 * as the original hand-written constant.ts functions.
 */
import { Metadata } from '@rapidaai/react';
import { loadProviderConfig, resolveCategoryParameters } from '../config-loader';
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

function getKeys(arr: Metadata[]): string[] {
  return arr.map(m => m.getKey()).sort();
}

// Valid credential for all tests
const cred = () => createMetadata('rapida.credential_id', 'test-cred');

describe('Groq STT — config vs original', () => {
  const config = loadProviderConfig('groq')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'groq');
    expect(findMeta(result, 'listen.model')).toBe('whisper-large-v3-turbo');
    expect(findMeta(result, 'listen.language')).toBe('en');
    // credential not set from empty input
    expect(getKeys(result)).toEqual(
      expect.arrayContaining(['listen.model', 'listen.language']),
    );
  });

  it('preserves existing valid values', () => {
    const existing = [
      cred(),
      createMetadata('listen.model', 'whisper-large-v3-turbo'),
      createMetadata('listen.language', 'en'),
    ];
    const result = getDefaultsFromConfig(config, 'stt', existing, 'groq');
    expect(findMeta(result, 'listen.model')).toBe('whisper-large-v3-turbo');
    expect(findMeta(result, 'listen.language')).toBe('en');
    expect(findMeta(result, 'rapida.credential_id')).toBe('test-cred');
  });

  it('preserves microphone.* params', () => {
    const existing = [createMetadata('microphone.volume', '0.8')];
    const result = getDefaultsFromConfig(config, 'stt', existing, 'groq');
    expect(findMeta(result, 'microphone.volume')).toBe('0.8');
  });

  it('validates: missing credential returns error', () => {
    const result = validateFromConfig(config, 'stt', 'groq', []);
    expect(result).toBeDefined();
    expect(result).toContain('credential');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'whisper-large-v3-turbo'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'groq', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'invalid-model'),
      createMetadata('listen.language', 'en'),
    ];
    const result = validateFromConfig(config, 'stt', 'groq', opts);
    expect(result).toBe('Please provide a valid groq model for speech to text.');
  });

  it('validates: invalid language returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'whisper-large-v3-turbo'),
      createMetadata('listen.language', 'invalid-lang'),
    ];
    const result = validateFromConfig(config, 'stt', 'groq', opts);
    expect(result).toBe('Please provide a valid groq language for speech to text.');
  });
});

describe('Deepgram STT — config vs original', () => {
  const config = loadProviderConfig('deepgram')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'deepgram');
    expect(findMeta(result, 'listen.model')).toBe('nova-3');
    expect(findMeta(result, 'listen.smart_format')).toBe('true');
    expect(findMeta(result, 'listen.filler_words')).toBe('true');
    expect(findMeta(result, 'listen.endpointing')).toBe('5');
    expect(findMeta(result, 'listen.punctuate')).toBe('true');
    expect(findMeta(result, 'listen.diarize')).toBe('false');
    expect(findMeta(result, 'listen.language')).toBe('multi');
    expect(findMeta(result, 'listen.threshold')).toBe('0.5');
    expect(findMeta(result, 'listen.keywords')).toBe('');
  });

  it('includes all expected keys', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'deepgram');
    const keys = getKeys(result);
    expect(keys).toEqual(
      expect.arrayContaining([
        'listen.model',
        'listen.smart_format',
        'listen.filler_words',
        'listen.endpointing',
        'listen.punctuate',
        'listen.diarize',
        'listen.language',
        'listen.threshold',
        'listen.keywords',
      ]),
    );
  });

  it('renders endpointing as a bounded number input', () => {
    const params = resolveCategoryParameters(
      'deepgram',
      'stt',
      config.stt!,
      [createMetadata('listen.model', 'nova-3')],
    );
    const endpointing = params.find(p => p.key === 'listen.endpointing');

    expect(endpointing).toMatchObject({
      label: 'Endpointing (ms)',
      type: 'number',
      min: 5,
      max: 500,
      step: 1,
    });
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'nova-3'),
      createMetadata('listen.language', 'multi'),
    ];
    expect(validateFromConfig(config, 'stt', 'deepgram', opts)).toBeUndefined();
  });

  it('validates: missing threshold is OK (not required)', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'nova-3'),
      createMetadata('listen.language', 'multi'),
      // no threshold, no keywords
    ];
    expect(validateFromConfig(config, 'stt', 'deepgram', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'bad-model'),
      createMetadata('listen.language', 'multi'),
    ];
    expect(validateFromConfig(config, 'stt', 'deepgram', opts)).toBe(
      'Please provide a valid deepgram model for speech to text.',
    );
  });
});

describe('Speechmatics STT — config vs original', () => {
  const config = loadProviderConfig('speechmatics')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'speechmatics');
    expect(findMeta(result, 'listen.model')).toBe('speechmatics-universal');
    expect(findMeta(result, 'listen.language')).toBe('en');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'speechmatics-universal'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'speechmatics', opts)).toBeUndefined();
  });

  it('validates: invalid language returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'speechmatics-universal'),
      createMetadata('listen.language', 'xx-invalid'),
    ];
    expect(validateFromConfig(config, 'stt', 'speechmatics', opts)).toBe(
      'Please provide a valid speechmatics language for speech to text.',
    );
  });
});

describe('Nvidia STT — config vs original', () => {
  const config = loadProviderConfig('nvidia')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'nvidia');
    expect(findMeta(result, 'listen.model')).toBe('nvidia-parakeet');
    expect(findMeta(result, 'listen.language')).toBe('en-US');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'nvidia-parakeet'),
      createMetadata('listen.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'stt', 'nvidia', opts)).toBeUndefined();
  });
});

describe('AssemblyAI STT — config vs original', () => {
  const config = loadProviderConfig('assemblyai')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'assemblyai');
    expect(findMeta(result, 'listen.model')).toBe('slam-1');
    expect(findMeta(result, 'listen.language')).toBe('en');
    expect(findMeta(result, 'listen.threshold')).toBe('0.5');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'slam-1'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'assemblyai', opts)).toBeUndefined();
  });

  it('validates: threshold is not required', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'slam-1'),
      createMetadata('listen.language', 'en'),
      // no threshold
    ];
    expect(validateFromConfig(config, 'stt', 'assemblyai', opts)).toBeUndefined();
  });
});

describe('AWS STT — config vs original', () => {
  const config = loadProviderConfig('aws')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'aws');
    expect(findMeta(result, 'listen.model')).toBe('aws-transcribe');
    expect(findMeta(result, 'listen.language')).toBe('en-US');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'aws-transcribe'),
      createMetadata('listen.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'stt', 'aws', opts)).toBeUndefined();
  });

  it('validates: invalid language returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'aws-transcribe'),
      createMetadata('listen.language', 'invalid'),
    ];
    expect(validateFromConfig(config, 'stt', 'aws', opts)).toBe(
      'Please provide a valid aws language for speech to text.',
    );
  });
});

describe('Azure Speech Service STT — config vs original', () => {
  const config = loadProviderConfig('azure-speech-service')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'azure-speech-service');
    expect(findMeta(result, 'listen.model')).toBe('azure-speech');
    expect(findMeta(result, 'listen.language')).toBe('en-US');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'azure-speech'),
      createMetadata('listen.language', 'en-US'),
    ];
    expect(validateFromConfig(config, 'stt', 'azure-speech-service', opts)).toBeUndefined();
  });
});

describe('Google Speech Service STT — config vs original', () => {
  const config = loadProviderConfig('google-speech-service')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'google-speech-service');
    expect(findMeta(result, 'listen.threshold')).toBe('0.5');
    expect(findMeta(result, 'listen.region')).toBe('global');
    expect(findMeta(result, 'listen.model')).toBe('latest_long');
    expect(findMeta(result, 'listen.language')).toBe('en-US');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'latest_long'),
      createMetadata('listen.region', 'global'),
      createMetadata('listen.language', 'en-SG'),
    ];
    expect(validateFromConfig(config, 'stt', 'google-speech-service', opts)).toBeUndefined();
  });
});

describe('Sarvam STT — config vs original', () => {
  const config = loadProviderConfig('sarvam')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'sarvam');
    expect(findMeta(result, 'listen.model')).toBe('saarika:v2.5');
    expect(findMeta(result, 'listen.language')).toBe('en-IN');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'saarika:v2.5'),
      createMetadata('listen.language', 'en-IN'),
    ];
    expect(validateFromConfig(config, 'stt', 'sarvam', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'bad-model'),
      createMetadata('listen.language', 'en-IN'),
    ];
    expect(validateFromConfig(config, 'stt', 'sarvam', opts)).toBe(
      'Please provide valid sarvam model for speech to text.',
    );
  });
});

describe('Sarvamai STT alias — config resolution', () => {
  const config = loadProviderConfig('sarvamai')!;

  it('loads aliased config and produces defaults', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'sarvamai');
    expect(findMeta(result, 'listen.model')).toBe('saarika:v2.5');
    expect(findMeta(result, 'listen.language')).toBe('en-IN');
  });

  it('validates using aliased data files', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'saarika:v2.5'),
      createMetadata('listen.language', 'en-IN'),
    ];
    expect(validateFromConfig(config, 'stt', 'sarvamai', opts)).toBeUndefined();
  });
});

describe('Cartesia STT — config vs original', () => {
  const config = loadProviderConfig('cartesia')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'cartesia');
    expect(findMeta(result, 'listen.model')).toBe('ink-whisper');
    expect(findMeta(result, 'listen.language')).toBe('en');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'ink-whisper'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'cartesia', opts)).toBeUndefined();
  });
});

describe('Smallest STT — config resolution', () => {
  const config = loadProviderConfig('smallest')!;

  it('produces the expected default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'smallest');
    expect(findMeta(result, 'listen.model')).toBe('pulse');
    expect(findMeta(result, 'listen.language')).toBe('en');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'pulse'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'smallest', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'bad-model'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'smallest', opts)).toBe(
      'Please provide a valid Smallest AI model for speech to text.',
    );
  });
});

describe('OpenAI STT — config vs original', () => {
  const config = loadProviderConfig('openai')!;

  it('produces the same default keys and values', () => {
    const result = getDefaultsFromConfig(config, 'stt', [], 'openai');
    expect(findMeta(result, 'listen.model')).toBe('gpt4o-transcribe');
    expect(findMeta(result, 'listen.language')).toBe('en');
  });

  it('validates: valid options returns undefined', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'gpt4o-transcribe'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'openai', opts)).toBeUndefined();
  });

  it('validates: invalid model returns error', () => {
    const opts = [
      cred(),
      createMetadata('listen.model', 'nonexistent'),
      createMetadata('listen.language', 'en'),
    ];
    expect(validateFromConfig(config, 'stt', 'openai', opts)).toBe(
      'Please provide a valid openai model for speech to text.',
    );
  });
});
