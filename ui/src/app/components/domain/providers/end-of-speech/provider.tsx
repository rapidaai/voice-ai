import { loadProviderConfig } from '@/providers/config-loader';
import { getDefaultsFromConfig } from '@/providers/config-defaults';
import { Metadata } from '@rapidaai/react';
import { ConfigRenderer } from '@/app/components/domain/providers/config-renderer';
import type { ProviderComponentProps } from '@/app/components/domain/providers/provider-component-props';

export const EOS_MODEL_PATH_KEYS = new Set<string>([
  'microphone.eos.livekit.model_path',
  'microphone.eos.livekit.tokenizer_path',
  'microphone.eos.pipecat.model_path',
]);

const upsertScopedProvider = (
  parameters: Metadata[],
  scopePrefix: string,
  key: string,
  value: string,
): Metadata[] => {
  const nonScoped = parameters.filter(p => !p.getKey().startsWith(scopePrefix));
  const scoped = parameters.filter(
    p => p.getKey().startsWith(scopePrefix) && p.getKey() !== key,
  );

  const providerMetadata = new Metadata();
  providerMetadata.setKey(key);
  providerMetadata.setValue(value);

  return [...nonScoped, providerMetadata, ...scoped];
};

export const GetDefaultEOSConfig = (
  provider: string,
  current: Metadata[],
): Metadata[] => {
  const config = loadProviderConfig(provider);
  if (!config?.eos) return current;

  const defaults = getDefaultsFromConfig(config, 'eos', current, provider, {
    includeCredential: false,
    replacePrefix: 'microphone.eos.',
  });
  const backendModelPaths = current.filter(parameter =>
    EOS_MODEL_PATH_KEYS.has(parameter.getKey()),
  );

  return upsertScopedProvider(
    [...defaults, ...backendModelPaths],
    'microphone.eos.',
    'microphone.eos.provider',
    provider,
  );
};

export function EndOfSpeechConfigComponent({
  provider,
  parameters,
  onChangeParameter,
}: ProviderComponentProps) {
  const config = loadProviderConfig(provider);
  if (!config?.eos) return null;

  return (
    <ConfigRenderer
      provider={provider}
      category="eos"
      config={config.eos}
      parameters={parameters}
      onParameterChange={onChangeParameter}
    />
  );
}
