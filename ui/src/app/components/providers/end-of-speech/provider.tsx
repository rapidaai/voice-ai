import { ProviderComponentProps } from '@/app/components/providers';
import { loadProviderConfig } from '@/providers/config-loader';
import { getDefaultsFromConfig } from '@/providers/config-defaults';
import { Metadata } from '@rapidaai/react';
import { FC } from 'react';
import { ConfigRenderer } from '@/app/components/providers/config-renderer';

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
  const parameters = [...current];
  const retainedAliases: Metadata[] = [];
  const aliases: [string, string[]][] =
    provider === 'livekit_eos'
      ? [
          ['quick_timeout', ['fallback_timeout', 'timeout']],
          ['extended_timeout', ['silence_timeout']],
        ]
      : provider === 'pipecat_smart_turn_eos'
        ? [
            ['fallback_timeout', ['timeout']],
            ['extended_timeout', ['silence_timeout']],
          ]
        : [];

  for (const [option, legacyOptions] of aliases) {
    const key = `microphone.eos.${option}`;
    const index = parameters.findIndex(parameter => parameter.getKey() === key);
    for (const candidateOption of [option, ...legacyOptions]) {
      const candidate = current.find(
        parameter => parameter.getKey() === `microphone.eos.${candidateOption}`,
      );
      const value = String(candidate?.getValue() ?? '');
      if (value !== value.trim()) continue;
      const isDecimal =
        /^[+-]?(?:[0-9](?:_?[0-9])*(?:\.(?:[0-9](?:_?[0-9])*)?)?|\.[0-9](?:_?[0-9])*)(?:[eE][+-]?[0-9](?:_?[0-9])*)?$/.test(
          value,
        ) && Number.isFinite(Number(value.replaceAll('_', '')));
      const isSpecial = /^(?:nan|[+-]?inf(?:inity)?)$/i.test(value);
      const isHexFloat =
        /^[+-]?0x(?:_?[0-9a-f](?:_?[0-9a-f])*(?:\.(?:[0-9a-f](?:_?[0-9a-f])*)?)?|\.[0-9a-f](?:_?[0-9a-f])*)p[+-]?[0-9](?:_?[0-9])*$/i.test(
          value,
        );
      if (!isDecimal && !isSpecial && !isHexFloat) continue;

      if (candidateOption !== option) {
        const metadata = new Metadata();
        metadata.setKey(key);
        metadata.setValue(value);
        if (index >= 0) {
          parameters[index] = metadata;
        } else {
          parameters.push(metadata);
        }
      }
      if (isHexFloat) {
        // Go owns hex-float range checks, so retain its fallback chain.
        retainedAliases.push(
          ...current.filter(parameter =>
            legacyOptions.some(
              legacyOption =>
                parameter.getKey() === `microphone.eos.${legacyOption}`,
            ),
          ),
        );
      }
      break;
    }
  }

  const defaults = getDefaultsFromConfig(config, 'eos', parameters, provider, {
    includeCredential: false,
    replacePrefix: 'microphone.eos.',
  });
  return upsertScopedProvider(
    [...defaults, ...retainedAliases],
    'microphone.eos.',
    'microphone.eos.provider',
    provider,
  );
};

export const EndOfSpeechConfigComponent: FC<ProviderComponentProps> = ({
  provider,
  parameters,
  onChangeParameter,
}) => {
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
};
