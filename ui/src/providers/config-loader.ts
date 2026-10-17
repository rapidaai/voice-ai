import { getProviderAsset } from '@/providers/provider-assets';

export interface ParameterLinkedField {
  key: string;
  sourceField: string;
}

export interface ParameterShowWhen {
  key: string;
  pattern: string;
}

export interface ParameterChoice {
  label: string;
  value: string;
}

export type ProviderConfigCategory =
  | 'stt'
  | 'tts'
  | 'text'
  | 'telephony'
  | 'vad'
  | 'eos'
  | 'noise'
  | 'telemetry'
  | 'storage';

export interface ParameterConfig {
  key: string;
  label: string;
  type:
    | 'dropdown'
    | 'slider'
    | 'number'
    | 'input'
    | 'textarea'
    | 'select'
    | 'json'
    | 'key_value';
  required?: boolean;
  default?: string;
  errorMessage?: string;
  helpText?: string;
  helpTextDisplay?: 'toggletip';
  colSpan?: 1 | 2;
  advanced?: boolean;
  showWhen?: ParameterShowWhen;
  linkedField?: ParameterLinkedField;
  // dropdown
  data?: string;
  valueField?: string;
  searchable?: boolean;
  strict?: boolean;
  customValue?: boolean;
  // slider / number
  min?: number;
  max?: number;
  step?: number;
  // textarea / input
  placeholder?: string;
  rows?: number;
  // select
  choices?: ParameterChoice[];
  // custom editor rendering
  editor?: 'websocket_dsl_json';
  editorMode?: 'query_params' | 'request_rules' | 'response_rules';
}

export interface CategoryConfig {
  preservePrefix?: string;
  modelSelectionKey?: string;
  parameters: ParameterConfig[];
}

export interface ProviderConfig {
  stt?: CategoryConfig;
  tts?: CategoryConfig;
  text?: CategoryConfig;
  telephony?: CategoryConfig;
  vad?: CategoryConfig;
  eos?: CategoryConfig;
  noise?: CategoryConfig;
  telemetry?: CategoryConfig;
  storage?: CategoryConfig;
}

export interface MetadataLike {
  getKey(): string;
  getValue(): string;
}

export interface ModelConfigOverrides {
  parameters?: ParameterConfig[];
}

interface ProviderModelDataItem {
  id?: string;
  name?: string;
  config?: ModelConfigOverrides;
  [key: string]: any;
}

const configCache: Record<string, ProviderConfig | null> = {};
const dataCache: Record<string, any[]> = {};
const modelIndexCache: Record<
  string,
  {
    byToken: Record<string, ProviderModelDataItem>;
  }
> = {};
const categoryParameterCache: Record<string, ParameterConfig[]> = {};
const PROVIDER_PATH_ALIASES: Record<string, string> = {
  sarvamai: 'sarvam',
  'google-speech-service': 'google',
  'aws-cloud': 'aws',
  'azure-cloud': 'azure',
};
const MODEL_SELECTOR_CATEGORIES: ReadonlySet<ProviderConfigCategory> = new Set([
  'stt',
  'tts',
  'text',
]);
const TEXT_MODEL_DATA_CANDIDATES = ['text-models.json', 'models.json'] as const;
const TEXT_CUSTOM_MODEL_PROVIDERS = new Set([
  'azure-foundry',
  'vertexai',
  'custom-llm',
  'openrouter',
]);

function warnProviderLoadFailure(
  scope: 'config' | 'data',
  provider: string,
  filename?: string,
): void {
  if (process.env.NODE_ENV === 'test') return;
  const source = filename ? `${provider}/${filename}` : provider;
  console.warn(`[provider-config] failed to load ${scope}: ${source}`);
}

function resolveProviderPath(provider: string): string {
  return PROVIDER_PATH_ALIASES[provider] ?? provider;
}

function tryLoadProviderJson<T>(provider: string, filename: string): T | null {
  return getProviderAsset<T>(provider, filename);
}

function loadSplitCategoryConfig(
  provider: string,
  category: ProviderConfigCategory,
): CategoryConfig | null {
  const raw = tryLoadProviderJson<any>(provider, `${category}.json`);
  if (!raw) return null;

  if (Array.isArray(raw?.parameters)) {
    return raw as CategoryConfig;
  }

  const nested = raw?.[category];
  if (nested && Array.isArray(nested.parameters)) {
    return nested as CategoryConfig;
  }

  return null;
}

function synthesizeProviderConfig(provider: string): ProviderConfig | null {
  const resolvedProvider = resolveProviderPath(provider);

  for (const dataFile of TEXT_MODEL_DATA_CANDIDATES) {
    const probed = tryLoadProviderJson<any[]>(resolvedProvider, dataFile);
    if (!Array.isArray(probed) || probed.length === 0) continue;
    const catalog = probed;
    dataCache[`${resolvedProvider}/${dataFile}`] = catalog;

    const defaultModel = catalog.find(
      item => typeof item?.id === 'string' && item.id.trim().length > 0,
    );
    if (!defaultModel) continue;

    const isCustomProvider = TEXT_CUSTOM_MODEL_PROVIDERS.has(resolvedProvider);
    return {
      text: {
        parameters: [
          {
            key: 'model.id',
            label: 'Model',
            type: 'dropdown',
            required: true,
            default: String(defaultModel.id),
            data: dataFile,
            valueField: 'id',
            linkedField: {
              key: 'model.name',
              sourceField: 'name',
            },
            ...(isCustomProvider
              ? {
                  strict: false,
                  customValue: true,
                }
              : {}),
            errorMessage: 'Please check and select valid model from dropdown.',
          },
        ],
      },
    };
  }

  return null;
}

export function loadProviderConfig(provider: string): ProviderConfig | null {
  const resolvedProvider = resolveProviderPath(provider);
  if (resolvedProvider in configCache) {
    return configCache[resolvedProvider];
  }

  const mergedConfig: ProviderConfig = {};
  let loadedAny = false;

  const categories: ProviderConfigCategory[] = [
    'stt',
    'tts',
    'text',
    'telephony',
    'vad',
    'eos',
    'noise',
    'telemetry',
    'storage',
  ];

  for (const category of categories) {
    const splitCategory = loadSplitCategoryConfig(resolvedProvider, category);
    if (!splitCategory) continue;
    mergedConfig[category] = splitCategory;
    loadedAny = true;
  }

  if (!mergedConfig.text) {
    const synthesizedTextConfig = synthesizeProviderConfig(resolvedProvider);
    if (synthesizedTextConfig?.text) {
      mergedConfig.text = synthesizedTextConfig.text;
      loadedAny = true;
    }
  }

  if (loadedAny) {
    configCache[resolvedProvider] = mergedConfig;
    return mergedConfig;
  }

  warnProviderLoadFailure('config', resolvedProvider);
  configCache[resolvedProvider] = null;
  return null;
}

export function loadProviderData(provider: string, filename: string): any[] {
  const resolvedProvider = resolveProviderPath(provider);
  const cacheKey = `${resolvedProvider}/${filename}`;
  if (cacheKey in dataCache) {
    return dataCache[cacheKey];
  }
  const data = getProviderAsset<any[]>(resolvedProvider, filename);
  if (data) {
    dataCache[cacheKey] = data;
    return data;
  }
  warnProviderLoadFailure('data', resolvedProvider, filename);
  dataCache[cacheKey] = [];
  return [];
}

function getMetadataValue(metadata: MetadataLike[], key: string): string {
  return metadata.find(m => m.getKey() === key)?.getValue() ?? '';
}

function lastSegment(value: string): string {
  const parts = value.split('/');
  return parts[parts.length - 1];
}

function findModelInCatalog(
  provider: string,
  dataFile: string,
  catalog: ProviderModelDataItem[],
  valueField: string,
  nameField: string,
  token: string,
): ProviderModelDataItem | undefined {
  const target = token.trim();
  if (!target) return undefined;
  const cacheKey = `${provider}/${dataFile}/${valueField}/${nameField}`;
  let cached = modelIndexCache[cacheKey];

  if (!cached) {
    const byToken: Record<string, ProviderModelDataItem> = {};
    for (const item of catalog) {
      const value = String(item?.[valueField] ?? '').trim();
      const name = String(item?.[nameField] ?? '').trim();
      if (value) {
        byToken[value] = item;
        byToken[lastSegment(value)] = item;
      }
      if (name) {
        byToken[name] = item;
        byToken[lastSegment(name)] = item;
      }
    }
    cached = { byToken };
    modelIndexCache[cacheKey] = cached;
  }

  return cached.byToken[target];
}

export function isModelSelectorParameter(param: ParameterConfig): boolean {
  if (param.type !== 'dropdown' || !param.data) return false;
  return (
    param.key === 'model.id' ||
    param.key === 'model' ||
    param.key.endsWith('.model')
  );
}

function getModelSelectorParameter(
  config: CategoryConfig,
  category?: ProviderConfigCategory,
): ParameterConfig | null {
  if (config.modelSelectionKey) {
    return (
      config.parameters.find(p => p.key === config.modelSelectionKey) ?? null
    );
  }
  if (category && !MODEL_SELECTOR_CATEGORIES.has(category)) return null;
  return config.parameters.find(isModelSelectorParameter) ?? null;
}

function getSelectedModelConfig(
  category: ProviderConfigCategory,
  provider: string,
  categoryConfig: CategoryConfig,
  currentMetadata: MetadataLike[],
): { modelConfig: ModelConfigOverrides; cacheScope: string } | null {
  const modelParam = getModelSelectorParameter(categoryConfig, category);
  if (!modelParam?.data) return null;

  const valueField = modelParam.valueField || 'id';
  const nameField = modelParam.linkedField?.sourceField || 'name';
  const selectedValue =
    getMetadataValue(currentMetadata, modelParam.key) ||
    modelParam.default ||
    '';
  const selectedLinkedValue = modelParam.linkedField
    ? getMetadataValue(currentMetadata, modelParam.linkedField.key)
    : '';

  const catalog = loadProviderData(
    provider,
    modelParam.data,
  ) as ProviderModelDataItem[];
  if (!catalog || catalog.length === 0) return null;

  let selectedModel =
    findModelInCatalog(
      provider,
      modelParam.data,
      catalog,
      valueField,
      nameField,
      selectedValue,
    ) ||
    findModelInCatalog(
      provider,
      modelParam.data,
      catalog,
      valueField,
      nameField,
      selectedLinkedValue,
    );
  let cacheScope = selectedModel?.[valueField]
    ? String(selectedModel[valueField])
    : selectedValue || selectedLinkedValue;

  if (!selectedModel && modelParam.customValue && catalog.length > 0) {
    selectedModel =
      findModelInCatalog(
        provider,
        modelParam.data,
        catalog,
        valueField,
        nameField,
        modelParam.default || '',
      ) || catalog[0];
    cacheScope = selectedModel?.[valueField]
      ? `custom:${String(selectedModel[valueField])}`
      : 'custom:fallback';
  }

  if (!selectedModel) return null;

  const parameters = selectedModel.config?.parameters ?? [];
  if (!Array.isArray(parameters) || parameters.length === 0) {
    return null;
  }

  return {
    modelConfig: { parameters },
    cacheScope: `${category}:${cacheScope}`,
  };
}

function mergeModelOverrides(
  baseParameters: ParameterConfig[],
  modelSelector: ParameterConfig | null,
  overrides: ModelConfigOverrides,
): ParameterConfig[] {
  const selector = modelSelector ? [{ ...modelSelector }] : [];
  const modelParameters = overrides.parameters ?? [];
  if (!Array.isArray(modelParameters) || modelParameters.length === 0) {
    return baseParameters.map(param => ({ ...param }));
  }
  const dedupe = new Set(selector.map(param => param.key));
  const resolvedModelParameters = modelParameters
    .filter(param => !dedupe.has(param.key))
    .map(param => ({ ...param }));
  return [...selector, ...resolvedModelParameters];
}

export function resolveCategoryParameters(
  provider: string,
  category: ProviderConfigCategory,
  categoryConfig: CategoryConfig,
  currentMetadata: MetadataLike[] = [],
): ParameterConfig[] {
  const modelSelector = getModelSelectorParameter(categoryConfig, category);
  if (!modelSelector?.data) {
    return categoryConfig.parameters.map(param => ({ ...param }));
  }

  const selectedModelConfig = getSelectedModelConfig(
    category,
    provider,
    categoryConfig,
    currentMetadata,
  );
  if (!selectedModelConfig) {
    return categoryConfig.parameters.map(param => ({ ...param }));
  }

  const cacheKey = `${provider}:${selectedModelConfig.cacheScope}`;
  const cached = categoryParameterCache[cacheKey];
  if (cached) return cached.map(param => ({ ...param }));

  const merged = mergeModelOverrides(
    categoryConfig.parameters,
    modelSelector,
    selectedModelConfig.modelConfig,
  );
  categoryParameterCache[cacheKey] = merged;
  return merged.map(param => ({ ...param }));
}
