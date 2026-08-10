import { Metadata, VaultCredential } from '@rapidaai/react';
import { ProviderComponentProps } from '@/app/components/providers';
import { GetDefaultEOSConfig } from '@/app/components/providers/end-of-speech/provider';
import { loadProviderConfig } from '@/providers/config-loader';
import {
  getDefaultsFromConfig,
  validateFromConfig,
} from '@/providers/config-defaults';
import {
  CUSTOM_STT_REQUEST_RULES_KEY,
  parseCustomSttRequestRules,
} from '@/providers/custom-stt/contract';
import { ConfigRenderer } from '@/app/components/providers/config-renderer';
import { FC } from 'react';
import {
  LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY,
  MICROPHONE_BARGE_IN_TRIGGER_KEY,
} from '@/app/components/providers/microphone/barge-in-trigger-control';

type ProviderCredentialRef = string | VaultCredential;

const CUSTOM_STT_PROVIDER = 'custom-stt';
const CUSTOM_STT_HTTP_V1 = 'http_v1';

const getProviderCredentialId = (credential: ProviderCredentialRef): string =>
  typeof credential === 'string' ? credential : credential.getId();

const getMetadataValue = (
  parameters: Metadata[],
  key: string,
): string | undefined =>
  parameters.find(param => param.getKey() === key)?.getValue();

const getCustomSttCredentialCompatibility = (
  credential?: VaultCredential | null,
): string | undefined => {
  const fields = credential?.getValue?.()?.getFieldsMap();
  const compatibility =
    fields?.get('apiCompatibility')?.getStringValue()?.trim() ||
    fields?.get('api_compatibility')?.getStringValue()?.trim();

  return compatibility || undefined;
};

const findSelectedProviderCredential = (
  credentialId: string | undefined,
  providerCredentials?: ProviderCredentialRef[],
): VaultCredential | undefined => {
  if (!credentialId || !providerCredentials) return undefined;

  return providerCredentials.find(
    (credential): credential is VaultCredential =>
      typeof credential !== 'string' && credential.getId() === credentialId,
  );
};

const getSelectedCustomSttCompatibility = (
  parameters: Metadata[],
  providerCredentials?: ProviderCredentialRef[],
): string | undefined => {
  const credentialId = getMetadataValue(parameters, 'rapida.credential_id');
  return getCustomSttCredentialCompatibility(
    findSelectedProviderCredential(credentialId, providerCredentials),
  );
};

const validateCustomSttHttpRequestRules = (
  parameters: Metadata[],
): string | undefined => {
  const requestRules = getMetadataValue(
    parameters,
    CUSTOM_STT_REQUEST_RULES_KEY,
  );
  if (!requestRules) {
    return 'Please provide valid custom STT request rules.';
  }

  try {
    const firstAudioRule = parseCustomSttRequestRules(requestRules).find(
      rule => rule.when?.packet === 'audio',
    );
    if (!firstAudioRule) {
      return 'Custom STT request rules must contain at least one rule with when.packet "audio".';
    }
    if (firstAudioRule.send?.frame !== 'json') {
      return 'Custom STT HTTP v1 requires the first audio request rule to use send.frame "json".';
    }
  } catch (error) {
    return error instanceof Error
      ? error.message
      : 'Please provide valid custom STT request rules.';
  }

  return undefined;
};

export const GetDefaultSpeechToTextIfInvalid = (
  provider: string,
  parameters: Metadata[],
) => {
  const config = loadProviderConfig(provider);
  if (!config?.stt) return parameters;
  return getDefaultsFromConfig(config, 'stt', parameters, provider);
};

export const ValidateSpeechToTextIfInvalid = (
  provider: string,
  parameters: Metadata[],
  providerCredentials?: ProviderCredentialRef[],
): string | undefined => {
  const config = loadProviderConfig(provider);
  if (!config?.stt) return undefined;
  const validationError = validateFromConfig(
    config,
    'stt',
    provider,
    parameters,
  );
  if (validationError) return validationError;

  const credentialID = parameters
    .find(opt => opt.getKey() === 'rapida.credential_id')
    ?.getValue();
  if (providerCredentials) {
    if (!credentialID) {
      return `Please provide a valid ${provider} credential.`;
    }
    if (
      !providerCredentials
        .map(credential => getProviderCredentialId(credential))
        .includes(credentialID)
    ) {
      return `Please select a valid ${provider} credential.`;
    }
  }

  const compatibility = getSelectedCustomSttCompatibility(
    parameters,
    providerCredentials,
  );
  if (
    provider === CUSTOM_STT_PROVIDER &&
    compatibility === CUSTOM_STT_HTTP_V1
  ) {
    const httpValidationError = validateCustomSttHttpRequestRules(parameters);
    if (httpValidationError) return httpValidationError;
  }

  return undefined;
};

/**
 *
 * @returns
 */
export const GetDefaultMicrophoneConfig = (
  existing: Metadata[] = [],
  defaults?: {
    'microphone.eos.fallback_timeout'?: string;
    'microphone.eos.threshold'?: string;
    'microphone.eos.quick_timeout'?: string;
    'microphone.eos.extended_timeout'?: string;
    'microphone.eos.model'?: string;
    'microphone.eos.provider'?: string;
    'microphone.denoising.provider'?: string;
    'microphone.barge_in_trigger'?: string;
    'microphone.vad.provider'?: string;
    'microphone.vad.barge_in_trigger'?: string;
    'microphone.vad.threshold'?: string;
  },
): Metadata[] => {
  const upsertMetadata = (
    parameters: Metadata[],
    key: string,
    value: string,
  ): Metadata[] => {
    const metadata = new Metadata();
    metadata.setKey(key);
    metadata.setValue(value);

    const index = parameters.findIndex(param => param.getKey() === key);
    if (index === -1) return [...parameters, metadata];

    const updated = [...parameters];
    updated[index] = metadata;
    return updated;
  };

  const defaultConfig = [
    {
      key: 'microphone.eos.provider',
      value: defaults?.['microphone.eos.provider'] ?? 'pipecat_smart_turn_eos',
    },
    {
      key: 'microphone.denoising.provider',
      value: defaults?.['microphone.denoising.provider'] ?? 'rn_noise',
    },
    {
      key: 'microphone.vad.provider',
      value: defaults?.['microphone.vad.provider'] ?? 'silero_vad',
    },
    {
      key: MICROPHONE_BARGE_IN_TRIGGER_KEY,
      value:
        defaults?.[MICROPHONE_BARGE_IN_TRIGGER_KEY] ??
        defaults?.[LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY] ??
        'vad',
    },
    {
      key: 'microphone.vad.threshold',
      value: defaults?.['microphone.vad.threshold'] ?? '0.5',
    },
  ];

  const legacyBargeInValue = existing.find(
    m => m.getKey() === LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY,
  )?.getValue();
  const existingWithoutLegacyBargeIn = existing.filter(
    m => m.getKey() !== LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY,
  );
  const existingKeys = new Set(
    existingWithoutLegacyBargeIn.map(m => m.getKey()),
  );

  const newConfigs = defaultConfig
    .filter(({ key }) => !existingKeys.has(key))
    .map(({ key, value }) => {
      const metadata = new Metadata();
      metadata.setKey(key);
      metadata.setValue(
        key === MICROPHONE_BARGE_IN_TRIGGER_KEY && legacyBargeInValue
          ? legacyBargeInValue
          : value,
      );
      return metadata;
    });

  const eosProvider =
    defaults?.['microphone.eos.provider'] ??
    existing.find(m => m.getKey() === 'microphone.eos.provider')?.getValue() ??
    'pipecat_smart_turn_eos';

  let hydrated = GetDefaultEOSConfig(eosProvider, [
    ...existingWithoutLegacyBargeIn,
    ...newConfigs,
  ]);

  for (const [key, value] of Object.entries(defaults ?? {})) {
    if (key === LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY) continue;
    if (!value || existingKeys.has(key)) continue;
    hydrated = upsertMetadata(hydrated, key, value);
  }

  return hydrated;
};

export const SpeechToTextConfigComponent: FC<ProviderComponentProps> = ({
  provider,
  parameters,
  onChangeParameter,
}) => {
  const config = loadProviderConfig(provider);
  if (!config?.stt) return null;
  return (
    <ConfigRenderer
      provider={provider}
      category="stt"
      config={config.stt}
      parameters={parameters}
      onParameterChange={onChangeParameter}
    />
  );
};
