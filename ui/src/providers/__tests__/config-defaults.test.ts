import { Metadata } from '@rapidaai/react';
import { ProviderConfig, loadProviderConfig } from '../config-loader';
import { getDefaultsFromConfig, validateFromConfig } from '../config-defaults';

// Helper to create a Metadata with key+value
function createMetadata(key: string, value: string): Metadata {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
}

// Helper to find a metadata by key
function findMeta(arr: Metadata[], key: string): Metadata | undefined {
  return arr.find(m => m.getKey() === key);
}

// Minimal STT config for testing
const minimalSttConfig: ProviderConfig = {
  stt: {
    preservePrefix: 'microphone.',
    parameters: [
      {
        key: 'listen.model',
        label: 'Model',
        type: 'dropdown',
        required: true,
        default: 'whisper-large-v3-turbo',
        data: 'speech-to-text-models.json',
        valueField: 'id',
        errorMessage: 'Please provide a valid groq model for speech to text.',
      },
      {
        key: 'listen.language',
        label: 'Language',
        type: 'dropdown',
        required: true,
        default: 'en',
        data: 'speech-to-text-languages.json',
        valueField: 'code',
        errorMessage:
          'Please provide a valid groq language for speech to text.',
      },
    ],
  },
};

// TTS config with strict:false for voice
const ttsConfig: ProviderConfig = {
  tts: {
    preservePrefix: 'speaker.',
    parameters: [
      {
        key: 'speak.model',
        label: 'Model',
        type: 'dropdown',
        required: true,
        data: 'text-to-speech-models.json',
        valueField: 'model_id',
        errorMessage: 'Please select valid model for text to speech.',
      },
      {
        key: 'speak.voice.id',
        label: 'Voice',
        type: 'dropdown',
        required: true,
        data: 'voices.json',
        valueField: 'voice_id',
        searchable: true,
        strict: false,
        errorMessage:
          'Please select a valid voice ID for text to speech.',
      },
    ],
  },
};

// Config with slider parameter
const sliderConfig: ProviderConfig = {
  stt: {
    preservePrefix: 'microphone.',
    parameters: [
      {
        key: 'listen.threshold',
        label: 'Transcript Confidence Threshold',
        type: 'slider',
        default: '0.5',
        min: 0.1,
        max: 0.9,
        step: 0.1,
        required: false,
      },
    ],
  },
};

// Config with linkedField
const linkedFieldConfig: ProviderConfig = {
  text: {
    parameters: [
      {
        key: 'model.id',
        label: 'Model',
        type: 'dropdown',
        required: true,
        data: 'text-models.json',
        valueField: 'id',
        linkedField: { key: 'model.name', sourceField: 'name' },
      },
    ],
  },
};

// Config with select parameter
const selectConfig: ProviderConfig = {
  text: {
    parameters: [
      {
        key: 'model.reasoning_effort',
        label: 'Reasoning Effort',
        type: 'select',
        required: false,
        choices: [
          { label: 'Low', value: 'low' },
          { label: 'Medium', value: 'medium' },
          { label: 'High', value: 'high' },
        ],
      },
    ],
  },
};

// Config with json parameter
const jsonConfig: ProviderConfig = {
  text: {
    parameters: [
      {
        key: 'model.response_format',
        label: 'Response Format',
        type: 'json',
        required: false,
      },
    ],
  },
};

const scopedVadConfig: ProviderConfig = {
  vad: {
    parameters: [
      {
        key: 'microphone.vad.confidence',
        label: 'VAD Confidence',
        type: 'slider',
        default: '0.6',
      },
    ],
  },
};

describe('getDefaultsFromConfig', () => {
  it('generates correct Metadata[] from config with defaults', () => {
    const result = getDefaultsFromConfig(
      minimalSttConfig,
      'stt',
      [],
      'groq',
    );

    expect(result.length).toBeGreaterThanOrEqual(2);
    const model = findMeta(result, 'listen.model');
    expect(model).toBeDefined();
    expect(model?.getValue()).toBe('whisper-large-v3-turbo');

    const language = findMeta(result, 'listen.language');
    expect(language).toBeDefined();
    expect(language?.getValue()).toBe('en');
  });

  it('always includes rapida.credential_id', () => {
    const cred = createMetadata('rapida.credential_id', 'test-cred-123');
    const result = getDefaultsFromConfig(
      minimalSttConfig,
      'stt',
      [cred],
      'groq',
    );

    const credResult = findMeta(result, 'rapida.credential_id');
    expect(credResult).toBeDefined();
    expect(credResult?.getValue()).toBe('test-cred-123');
  });

  it('preserves existing values when they pass validation', () => {
    // Create existing metadata with valid values from groq data files
    const existingModel = createMetadata(
      'listen.model',
      'whisper-large-v3-turbo',
    );
    const existingLang = createMetadata('listen.language', 'en');

    const result = getDefaultsFromConfig(
      minimalSttConfig,
      'stt',
      [existingModel, existingLang],
      'groq',
    );

    expect(findMeta(result, 'listen.model')?.getValue()).toBe(
      'whisper-large-v3-turbo',
    );
    expect(findMeta(result, 'listen.language')?.getValue()).toBe('en');
  });

  it('falls back to default when existing value fails validation', () => {
    const invalidModel = createMetadata('listen.model', 'invalid-model-xyz');

    const result = getDefaultsFromConfig(
      minimalSttConfig,
      'stt',
      [invalidModel],
      'groq',
    );

    // Should fall back to the default since 'invalid-model-xyz' is not in the data file
    const model = findMeta(result, 'listen.model');
    expect(model).toBeDefined();
    expect(model?.getValue()).toBe('whisper-large-v3-turbo');
  });

  it('preserves microphone.* params for STT (preservePrefix)', () => {
    const micParam = createMetadata('microphone.volume', '0.8');

    const result = getDefaultsFromConfig(
      minimalSttConfig,
      'stt',
      [micParam],
      'groq',
    );

    const preserved = findMeta(result, 'microphone.volume');
    expect(preserved).toBeDefined();
    expect(preserved?.getValue()).toBe('0.8');
  });

  it('preserves speaker.* params for TTS (preservePrefix)', () => {
    const speakerParam = createMetadata('speaker.rate', '1.0');

    const result = getDefaultsFromConfig(
      ttsConfig,
      'tts',
      [speakerParam],
      'groq',
    );

    const preserved = findMeta(result, 'speaker.rate');
    expect(preserved).toBeDefined();
    expect(preserved?.getValue()).toBe('1.0');
  });

  it('returns currentMetadata unchanged when category not in config', () => {
    const meta = [createMetadata('some.key', 'value')];
    const result = getDefaultsFromConfig(
      minimalSttConfig,
      'tts', // STT config has no TTS
      meta,
      'groq',
    );

    expect(result).toBe(meta);
  });

  it('handles parameters with no default (optional fields)', () => {
    const result = getDefaultsFromConfig(
      sliderConfig,
      'stt',
      [],
      'deepgram',
    );

    const threshold = findMeta(result, 'listen.threshold');
    expect(threshold).toBeDefined();
    expect(threshold?.getValue()).toBe('0.5');
  });

  it('does not validate against data for strict:false params', () => {
    // voice.id with strict:false should accept any value
    const customVoice = createMetadata('speak.voice.id', 'my-custom-voice');

    const result = getDefaultsFromConfig(
      ttsConfig,
      'tts',
      [customVoice],
      'groq',
    );

    const voice = findMeta(result, 'speak.voice.id');
    expect(voice).toBeDefined();
    expect(voice?.getValue()).toBe('my-custom-voice');
  });

  it('applies model-level defaults from selected model config', () => {
    const openaiConfig = loadProviderConfig('openai') as ProviderConfig;
    const result = getDefaultsFromConfig(
      openaiConfig,
      'text',
      [
        createMetadata('model.id', 'openai/gpt-4o-mini'),
        createMetadata('model.name', 'gpt-4o-mini'),
      ],
      'openai',
      { includeCredential: false },
    );

    expect(findMeta(result, 'model.temperature')?.getValue()).toBe('0.3');
    expect(findMeta(result, 'model.max_completion_tokens')?.getValue()).toBe('1048');
  });

  it('preserves valid existing values and patches invalid values against model constraints', () => {
    const openaiConfig = loadProviderConfig('openai') as ProviderConfig;
    const withValidExisting = getDefaultsFromConfig(
      openaiConfig,
      'text',
      [
        createMetadata('model.id', 'openai/gpt-4o-mini'),
        createMetadata('model.name', 'gpt-4o-mini'),
        createMetadata('model.temperature', '0.8'),
      ],
      'openai',
      { includeCredential: false },
    );

    expect(findMeta(withValidExisting, 'model.temperature')?.getValue()).toBe('0.8');

    const withInvalidExisting = getDefaultsFromConfig(
      openaiConfig,
      'text',
      [
        createMetadata('model.id', 'openai/gpt-4o-mini'),
        createMetadata('model.name', 'gpt-4o-mini'),
        createMetadata('model.temperature', '2'),
      ],
      'openai',
      { includeCredential: false },
    );

    expect(findMeta(withInvalidExisting, 'model.temperature')?.getValue()).toBe('0.3');
  });

  it('replaces scoped prefix params while keeping non-scoped metadata', () => {
    const existing = [
      createMetadata('rapida.credential_id', 'cred-1'),
      createMetadata('listen.model', 'nova-3'),
      createMetadata('microphone.vad.provider', 'ten_vad'),
      createMetadata('microphone.vad.confidence', '0.8'),
      createMetadata('microphone.vad.stop_secs', '0.12'),
    ];

    const result = getDefaultsFromConfig(
      scopedVadConfig,
      'vad',
      existing,
      'silero_vad',
      {
        includeCredential: false,
        replacePrefix: 'microphone.vad.',
      },
    );

    expect(findMeta(result, 'listen.model')?.getValue()).toBe('nova-3');
    expect(findMeta(result, 'rapida.credential_id')?.getValue()).toBe('cred-1');
    expect(findMeta(result, 'microphone.vad.confidence')?.getValue()).toBe('0.8');
    expect(findMeta(result, 'microphone.vad.stop_secs')).toBeUndefined();
  });
});

describe('validateFromConfig', () => {
  it('returns error when rapida.credential_id is missing', () => {
    const result = validateFromConfig(
      minimalSttConfig,
      'stt',
      'groq',
      [],
    );
    expect(result).toBe('Please provide a valid groq credential.');
  });

  it('returns error when rapida.credential_id is empty', () => {
    const cred = createMetadata('rapida.credential_id', '');
    const result = validateFromConfig(
      minimalSttConfig,
      'stt',
      'groq',
      [cred],
    );
    expect(result).toBe('Please provide a valid groq credential.');
  });

  it('returns error when required field is empty', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const result = validateFromConfig(
      minimalSttConfig,
      'stt',
      'groq',
      [cred],
    );
    expect(result).toBe(
      'Please provide a valid groq model for speech to text.',
    );
  });

  it('returns error when required dropdown value not in data file', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const model = createMetadata('listen.model', 'nonexistent-model');
    const lang = createMetadata('listen.language', 'en');

    const result = validateFromConfig(
      minimalSttConfig,
      'stt',
      'groq',
      [cred, model, lang],
    );
    expect(result).toBe(
      'Please provide a valid groq model for speech to text.',
    );
  });

  it('returns undefined (no error) when all required fields valid', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const model = createMetadata('listen.model', 'whisper-large-v3-turbo');
    const lang = createMetadata('listen.language', 'en');

    const result = validateFromConfig(
      minimalSttConfig,
      'stt',
      'groq',
      [cred, model, lang],
    );
    expect(result).toBeUndefined();
  });

  it('uses custom errorMessage from config when provided', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const result = validateFromConfig(
      minimalSttConfig,
      'stt',
      'groq',
      [cred],
    );
    // The first required param without value should return its custom errorMessage
    expect(result).toBe(
      'Please provide a valid groq model for speech to text.',
    );
  });

  it('skips validation for optional (required: false) empty fields', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    // sliderConfig has required:false on threshold
    const result = validateFromConfig(
      sliderConfig,
      'stt',
      'deepgram',
      [cred],
    );
    expect(result).toBeUndefined();
  });

  it('validates number ranges (slider min/max)', () => {
    const config: ProviderConfig = {
      stt: {
        parameters: [
          {
            key: 'listen.threshold',
            label: 'Threshold',
            type: 'slider',
            required: true,
            min: 0.1,
            max: 0.9,
            errorMessage: 'Threshold must be between 0.1 and 0.9.',
          },
        ],
      },
    };

    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const tooLow = createMetadata('listen.threshold', '0.05');
    expect(
      validateFromConfig(config, 'stt', 'test', [cred, tooLow]),
    ).toBe('Threshold must be between 0.1 and 0.9.');

    const tooHigh = createMetadata('listen.threshold', '1.5');
    expect(
      validateFromConfig(config, 'stt', 'test', [cred, tooHigh]),
    ).toBe('Threshold must be between 0.1 and 0.9.');

    const valid = createMetadata('listen.threshold', '0.5');
    expect(
      validateFromConfig(config, 'stt', 'test', [cred, valid]),
    ).toBeUndefined();
  });

  it('validates JSON fields are parseable', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const validJson = createMetadata(
      'model.response_format',
      '{"type":"json"}',
    );
    expect(
      validateFromConfig(jsonConfig, 'text', 'openai', [cred, validJson]),
    ).toBeUndefined();

    const invalidJson = createMetadata(
      'model.response_format',
      '{invalid json',
    );
    expect(
      validateFromConfig(jsonConfig, 'text', 'openai', [cred, invalidJson]),
    ).toBeDefined();
  });

  it('validates custom-llm model.parameters as JSON object', () => {
    const customLLMConfig: ProviderConfig = {
      text: {
        parameters: [
          {
            key: 'model.parameters',
            label: 'Model Parameters',
            type: 'json',
            required: false,
          },
        ],
      },
    };

    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const validObject = createMetadata(
      'model.parameters',
      '{"temperature":0.7}',
    );
    expect(
      validateFromConfig(customLLMConfig, 'text', 'custom-llm', [
        cred,
        validObject,
      ]),
    ).toBeUndefined();

    const invalidArray = createMetadata('model.parameters', '[1,2,3]');
    expect(
      validateFromConfig(customLLMConfig, 'text', 'custom-llm', [
        cred,
        invalidArray,
      ]),
    ).toBeDefined();
  });

  it('validates select choices', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const validChoice = createMetadata('model.reasoning_effort', 'medium');
    expect(
      validateFromConfig(selectConfig, 'text', 'openai', [
        cred,
        validChoice,
      ]),
    ).toBeUndefined();

    const invalidChoice = createMetadata(
      'model.reasoning_effort',
      'ultra',
    );
    expect(
      validateFromConfig(selectConfig, 'text', 'openai', [
        cred,
        invalidChoice,
      ]),
    ).toBeDefined();
  });

  it('skips dropdown data validation for strict:false', () => {
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const model = createMetadata('speak.model', 'playai-tts');
    const voice = createMetadata('speak.voice.id', 'any-custom-voice');

    const result = validateFromConfig(
      ttsConfig,
      'tts',
      'groq',
      [cred, model, voice],
    );
    expect(result).toBeUndefined();
  });

  it('validates using model-level parameters loaded from selected model', () => {
    const openaiConfig = loadProviderConfig('openai') as ProviderConfig;
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const model = createMetadata('model.id', 'openai/gpt-4o');
    const modelName = createMetadata('model.name', 'gpt-4o');
    const frequencyPenalty = createMetadata('model.frequency_penalty', '0');
    const topP = createMetadata('model.top_p', '1');
    const presencePenalty = createMetadata('model.presence_penalty', '0');
    const maxTokens = createMetadata('model.max_completion_tokens', '2048');
    const temperature = createMetadata('model.temperature', '2');

    const result = validateFromConfig(
      openaiConfig,
      'text',
      'openai',
      [
        cred,
        model,
        modelName,
        frequencyPenalty,
        topP,
        presencePenalty,
        maxTokens,
        temperature,
      ],
    );

    expect(result).toBe(
      'Please check and provide a correct value for temperature any decimal value between 0 to 1',
    );
  });

  it('returns undefined when category not in config', () => {
    const result = validateFromConfig(
      minimalSttConfig,
      'tts',
      'groq',
      [],
    );
    expect(result).toBeUndefined();
  });

  it('validates NaN as invalid for number type', () => {
    const config: ProviderConfig = {
      stt: {
        parameters: [
          {
            key: 'listen.threshold',
            label: 'Threshold',
            type: 'number',
            required: true,
            errorMessage: 'Invalid threshold.',
          },
        ],
      },
    };
    const cred = createMetadata('rapida.credential_id', 'valid-cred');
    const nan = createMetadata('listen.threshold', 'not-a-number');
    expect(
      validateFromConfig(config, 'stt', 'test', [cred, nan]),
    ).toBe('Invalid threshold.');
  });
});
