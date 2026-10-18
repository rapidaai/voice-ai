const EXPECTED_COMPATIBILITY_VALUES = [
  'openai_chat_completions',
  'openai_responses',
  'anthropic_messages',
  'gemini_generate_content',
  'openai_compatible',
];

function getCustomLLMProviderConfig(list: any[]) {
  return list.find(provider => provider.code === 'custom-llm');
}

describe('Custom LLM provider config', () => {
  it('exposes expanded compatibility choices in development and production configs', () => {
    const developmentProviders = require('../provider.development.json');
    const productionProviders = require('../provider.production.json');

    const developmentCustomLLM =
      getCustomLLMProviderConfig(developmentProviders);
    const productionCustomLLM = getCustomLLMProviderConfig(productionProviders);

    const getCompatibilityValues = (provider: any) => {
      const apiCompatibilityConfig = provider.configurations.find(
        (config: any) => config.name === 'apiCompatibility',
      );
      return (apiCompatibilityConfig?.choices ?? []).map(
        (choice: any) => choice.value,
      );
    };

    expect(getCompatibilityValues(developmentCustomLLM)).toEqual(
      EXPECTED_COMPATIBILITY_VALUES,
    );
    expect(getCompatibilityValues(productionCustomLLM)).toEqual(
      EXPECTED_COMPATIBILITY_VALUES,
    );
  });

  it('keeps current credential key names for UI flow in both environments', () => {
    const developmentProviders = require('../provider.development.json');
    const productionProviders = require('../provider.production.json');

    const getConfigurationKeys = (provider: any) =>
      provider.configurations.map((config: any) => config.name);

    const developmentCustomLLM =
      getCustomLLMProviderConfig(developmentProviders);
    const productionCustomLLM = getCustomLLMProviderConfig(productionProviders);

    expect(getConfigurationKeys(developmentCustomLLM)).toEqual(
      expect.arrayContaining(['apiCompatibility', 'baseUrl', 'headers']),
    );
    expect(getConfigurationKeys(productionCustomLLM)).toEqual(
      expect.arrayContaining(['apiCompatibility', 'baseUrl', 'headers']),
    );
  });

  it('uses JSON model.parameters schema for custom-llm text models', () => {
    const textModels = require('../custom-llm/text-models.json');
    const parameters = textModels?.[0]?.config?.parameters ?? [];
    const modelParameters = parameters.find(
      (param: any) => param.key === 'model.parameters',
    );

    expect(modelParameters?.type).toBe('json');
    expect(modelParameters?.helpText).toContain('/v1/chat/completions');
    expect(modelParameters?.helpText).toContain('/v1/responses');
  });
});
