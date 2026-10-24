import {
  loadProviderConfig,
  loadProviderData,
  resolveCategoryParameters,
} from '../config-loader';
import { Metadata } from '@rapidaai/react';

// Clear require cache between tests to reset the module-level caches
beforeEach(() => {
  jest.resetModules();
});

describe('loadProviderConfig', () => {
  it('loads split category config and returns parsed config', () => {
    const config = loadProviderConfig('groq');
    expect(config).not.toBeNull();
    expect(config?.stt).toBeDefined();
    expect(config?.stt?.parameters).toBeInstanceOf(Array);
    expect(config?.stt?.parameters.length).toBeGreaterThan(0);
  });

  it('returns null for provider without split category config', () => {
    const config = loadProviderConfig('nonexistent-provider-xyz');
    expect(config).toBeNull();
  });

  it('synthesizes text config from model catalog when text.json is absent', () => {
    const config = loadProviderConfig('anthropic');
    expect(config).not.toBeNull();
    expect(config?.text).toBeDefined();
    expect(config?.text?.parameters).toBeInstanceOf(Array);
    expect(config?.text?.parameters[0]?.key).toBe('model.id');
    expect(config?.text?.parameters[0]?.data).toBe('text-models.json');
  });

  it('returns config with tts section for providers that have it', () => {
    const config = loadProviderConfig('groq');
    expect(config?.tts).toBeDefined();
    expect(config?.tts?.parameters).toBeInstanceOf(Array);
  });

  it('returns correct parameter structure', () => {
    const config = loadProviderConfig('groq');
    const sttParams = config?.stt?.parameters;
    expect(sttParams).toBeDefined();

    const modelParam = sttParams?.find(p => p.key === 'listen.model');
    expect(modelParam).toBeDefined();
    expect(modelParam?.label).toBe('Model');
    expect(modelParam?.type).toBe('dropdown');
    expect(modelParam?.required).toBe(true);
    expect(modelParam?.data).toBe('speech-to-text-models.json');
    expect(modelParam?.valueField).toBe('id');
  });

  it('returns preservePrefix for stt and tts', () => {
    const config = loadProviderConfig('groq');
    expect(config?.stt?.preservePrefix).toBe('microphone.');
    expect(config?.tts?.preservePrefix).toBe('speaker.');
  });

  it('supports provider code aliases when loading config', () => {
    const config = loadProviderConfig('sarvamai');
    expect(config).not.toBeNull();
    expect(config?.stt).toBeDefined();
  });

  it('supports google-speech-service alias when loading config', () => {
    const config = loadProviderConfig('google-speech-service');
    expect(config).not.toBeNull();
    expect(config?.stt).toBeDefined();
  });
});

describe('loadProviderData', () => {
  it('loads data from a valid JSON file', () => {
    const data = loadProviderData('groq', 'speech-to-text-models.json');
    expect(data).toBeInstanceOf(Array);
    expect(data.length).toBeGreaterThan(0);
  });

  it('returns empty array for missing data file', () => {
    const data = loadProviderData('groq', 'nonexistent-file.json');
    expect(data).toEqual([]);
  });

  it('returns empty array for missing provider', () => {
    const data = loadProviderData('nonexistent-provider-xyz', 'models.json');
    expect(data).toEqual([]);
  });

  it('loads voice data with correct fields', () => {
    const data = loadProviderData('groq', 'voices.json');
    expect(data).toBeInstanceOf(Array);
    if (data.length > 0) {
      expect(data[0]).toHaveProperty('voice_id');
      expect(data[0]).toHaveProperty('name');
    }
  });

  it('supports provider code aliases when loading data', () => {
    const data = loadProviderData('sarvamai', 'speech-to-text-models.json');
    expect(data).toBeInstanceOf(Array);
    expect(data.length).toBeGreaterThan(0);
    expect(data[0]).toHaveProperty('model_id');
  });

  it('supports google-speech-service alias when loading data', () => {
    const data = loadProviderData(
      'google-speech-service',
      'speech-to-text-language.json',
    );
    expect(data).toBeInstanceOf(Array);
    expect(data.length).toBeGreaterThan(0);
    expect(data[0]).toHaveProperty('code');
  });

  it('loads the current Anthropic Claude model catalog', () => {
    const data = loadProviderData('anthropic', 'text-models.json');
    expect(data.slice(0, 4).map(model => model.id)).toEqual([
      'anthropic/claude-fable-5-1',
      'anthropic/claude-opus-5-5',
      'anthropic/claude-sonnet-5-5',
      'anthropic/claude-haiku-5-5',
    ]);
    expect(data.slice(0, 4).map(model => model.name)).toEqual([
      'claude-fable-5-1',
      'claude-opus-5-5',
      'claude-sonnet-5-5',
      'claude-haiku-5-5',
    ]);
  });
});

describe('resolveCategoryParameters', () => {
  const createMetadata = (key: string, value: string): Metadata => {
    const m = new Metadata();
    m.setKey(key);
    m.setValue(value);
    return m;
  };

  it('includes model selector + model-defined parameters', () => {
    const config = loadProviderConfig('openai');
    expect(config?.text).toBeDefined();

    const resolved = resolveCategoryParameters(
      'openai',
      'text',
      config!.text!,
      [
        createMetadata('model.id', 'openai/gpt-4o'),
        createMetadata('model.name', 'gpt-4o'),
      ],
    );

    expect(resolved.find(p => p.key === 'model.id')).toBeDefined();
    const temperature = resolved.find(p => p.key === 'model.temperature');
    expect(temperature).toBeDefined();
    expect(temperature?.type).toBe('slider');
  });

  it('applies per-model defaults from model config parameters', () => {
    const config = loadProviderConfig('openai');
    expect(config?.text).toBeDefined();

    const resolvedDefault = resolveCategoryParameters(
      'openai',
      'text',
      config!.text!,
      [
        createMetadata('model.id', 'openai/gpt-4o'),
        createMetadata('model.name', 'gpt-4o'),
      ],
    );

    const resolvedMini = resolveCategoryParameters(
      'openai',
      'text',
      config!.text!,
      [
        createMetadata('model.id', 'openai/gpt-4o-mini'),
        createMetadata('model.name', 'gpt-4o-mini'),
      ],
    );

    const defaultTemperature = resolvedDefault.find(
      p => p.key === 'model.temperature',
    );
    const miniTemperature = resolvedMini.find(
      p => p.key === 'model.temperature',
    );

    expect(defaultTemperature?.default).toBe('0.7');
    expect(miniTemperature?.default).toBe('0.3');
  });

  it('keeps model-specific params for custom-model providers via fallback schema', () => {
    const config = loadProviderConfig('azure-foundry');
    expect(config?.text).toBeDefined();

    const resolved = resolveCategoryParameters(
      'azure-foundry',
      'text',
      config!.text!,
      [
        createMetadata('model.id', 'my-custom-deployment'),
        createMetadata('model.name', 'my-custom-deployment'),
      ],
    );

    expect(resolved.find(p => p.key === 'model.id')).toBeDefined();
    expect(resolved.find(p => p.key === 'model.temperature')).toBeDefined();
  });

  it('enables free-form model selection for openrouter', () => {
    const config = loadProviderConfig('openrouter');
    expect(config?.text).toBeDefined();

    const modelSelector = config?.text?.parameters.find(
      p => p.key === 'model.id',
    );
    expect(modelSelector).toBeDefined();
    expect(modelSelector?.customValue).toBe(true);
    expect(modelSelector?.strict).toBe(false);
    expect(modelSelector?.data).toBe('text-models.json');
  });

  it('uses reasoning-style params for gpt-5.5 without sampling controls', () => {
    const config = loadProviderConfig('openai');
    expect(config?.text).toBeDefined();

    const resolved = resolveCategoryParameters(
      'openai',
      'text',
      config!.text!,
      [
        createMetadata('model.id', 'openai/gpt-5.5'),
        createMetadata('model.name', 'gpt-5.5'),
      ],
    );

    expect(resolved.find(p => p.key === 'model.id')).toBeDefined();
    expect(
      resolved.find(p => p.key === 'model.max_completion_tokens'),
    ).toBeDefined();
    expect(
      resolved.find(p => p.key === 'model.reasoning_effort'),
    ).toBeDefined();
    expect(resolved.find(p => p.key === 'model.temperature')).toBeUndefined();
    expect(resolved.find(p => p.key === 'model.top_p')).toBeUndefined();
    expect(
      resolved.find(p => p.key === 'model.frequency_penalty'),
    ).toBeUndefined();
    expect(
      resolved.find(p => p.key === 'model.presence_penalty'),
    ).toBeUndefined();
    expect(resolved.find(p => p.key === 'model.seed')).toBeUndefined();
  });

  it('applies model-specific reasoning_effort choices for gpt-5 variants', () => {
    const config = loadProviderConfig('openai');
    expect(config?.text).toBeDefined();

    const resolveChoices = (model: string) =>
      resolveCategoryParameters('openai', 'text', config!.text!, [
        createMetadata('model.id', `openai/${model}`),
        createMetadata('model.name', model),
      ]).find(p => p.key === 'model.reasoning_effort')?.choices;

    const gpt55Choices = resolveChoices('gpt-5.5')?.map(c => c.value);
    const gpt52Choices = resolveChoices('gpt-5.2')?.map(c => c.value);
    const gpt51Choices = resolveChoices('gpt-5.1')?.map(c => c.value);
    const gpt54MiniChoices = resolveChoices('gpt-5.4-mini')?.map(c => c.value);
    const gpt54NanoChoices = resolveChoices('gpt-5.4-nano')?.map(c => c.value);

    expect(gpt55Choices).toEqual([
      'none',
      'low',
      'medium',
      'high',
      'xhigh',
      'xhigh',
    ]);
    expect(gpt52Choices).toEqual([
      'none',
      'low',
      'medium',
      'high',
      'xhigh',
      'xhigh',
    ]);
    expect(gpt51Choices).toEqual(['none', 'low', 'medium', 'high', 'xhigh']);
    expect(gpt54MiniChoices).toEqual([
      'none',
      'low',
      'medium',
      'high',
      'xhigh',
    ]);
    expect(gpt54NanoChoices).toEqual([
      'none',
      'low',
      'medium',
      'high',
      'xhigh',
    ]);
  });

  it('uses the configured reasoning effort help text for gpt-5 family', () => {
    const config = loadProviderConfig('openai');
    expect(config?.text).toBeDefined();

    const resolved = resolveCategoryParameters(
      'openai',
      'text',
      config!.text!,
      [
        createMetadata('model.id', 'openai/gpt-5.5'),
        createMetadata('model.name', 'gpt-5.5'),
      ],
    );

    const reasoningParam = resolved.find(
      p => p.key === 'model.reasoning_effort',
    );
    expect(reasoningParam).toBeDefined();
    expect(reasoningParam?.helpText).toBeUndefined();
  });
});
