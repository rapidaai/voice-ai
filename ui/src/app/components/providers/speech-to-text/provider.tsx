import { Metadata } from '@rapidaai/react';
import { ProviderConfig } from '@/app/components/providers';
import {
  ConfigureAssemblyAISpeechToText,
  GetAssemblyAIDefaultOptions,
  ValidateAssemblyAIOptions,
} from '@/app/components/providers/speech-to-text/assemblyai';
import {
  ConfigureAzureSpeechToText,
  GetAzureDefaultOptions,
  ValidateAzureOptions,
} from '@/app/components/providers/speech-to-text/azure';
import {
  ConfigureCartesiaSpeechToText,
  GetCartesiaDefaultOptions,
  ValidateCartesiaOptions,
} from '@/app/components/providers/speech-to-text/cartesia';
import {
  ConfigureDeepgramSpeechToText,
  GetDeepgramDefaultOptions,
  ValidateDeepgramOptions,
} from '@/app/components/providers/speech-to-text/deepgram';
import {
  ConfigureGoogleSpeechToText,
  GetGoogleDefaultOptions,
  ValidateGoogleOptions,
} from '@/app/components/providers/speech-to-text/google';
import {
  ConfigureOpenAISpeechToText,
  GetOpenAIDefaultOptions,
  ValidateOpenAIOptions,
} from '@/app/components/providers/speech-to-text/openai';
import { FC } from 'react';

export const GetDefaultSpeechToTextIfInvalid = (
  provider: string,
  parameters: Metadata[],
) => {
  switch (provider) {
    case 'google':
    case 'google-cloud':
      return GetGoogleDefaultOptions(parameters);
    case 'deepgram':
      return GetDeepgramDefaultOptions(parameters);
    case 'openai':
      return GetOpenAIDefaultOptions(parameters);
    case 'azure':
    case 'azure-cloud':
      return GetAzureDefaultOptions(parameters);
    case 'assemblyai':
      return GetAssemblyAIDefaultOptions(parameters);
    case 'cartesia':
      return GetCartesiaDefaultOptions(parameters);
    default:
      return parameters;
  }
};

export const ValidateSpeechToTextIfInvalid = (
  provider: string,
  parameters: Metadata[],
): boolean => {
  switch (provider) {
    case 'google-cloud':
    case 'google':
      return ValidateGoogleOptions(parameters);
    case 'deepgram':
      return ValidateDeepgramOptions(parameters);
    case 'openai':
      return ValidateOpenAIOptions(parameters);
    case 'azure':
    case 'azure-cloud':
      return ValidateAzureOptions(parameters);
    case 'assemblyai':
      return ValidateAssemblyAIOptions(parameters);
    case 'cartesia':
      return ValidateCartesiaOptions(parameters);
    default:
      return false;
  }
};

/**
 *
 * @returns
 */
export const GetDefaultMicrophoneConfig = (
  existing: Metadata[] = [],
): Metadata[] => {
  const defaultConfig = [
    { key: 'microphone.eos.enable', value: 'true' },
    { key: 'microphone.eos.timeout', value: '1500' },
    { key: 'microphone.denoising', value: 'true' },
    { key: 'microphone.silence.timeout', value: '5000' },
    { key: 'microphone.idle.message', value: 'Are you still there?' },
  ];

  const existingKeys = new Set(existing.map(m => m.getKey()));

  const newConfigs = defaultConfig
    .filter(({ key }) => !existingKeys.has(key))
    .map(({ key, value }) => {
      const metadata = new Metadata();
      metadata.setKey(key);
      metadata.setValue(value);
      return metadata;
    });

  return [...existing, ...newConfigs];
};

export const SpeechToTextConfigComponent: FC<{
  config: ProviderConfig;
  updateConfig: (config: Partial<ProviderConfig>) => void;
}> = ({ config, updateConfig }) => {
  switch (config?.provider) {
    case 'google':
    case 'google-cloud':
      return (
        <ConfigureGoogleSpeechToText
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );

    case 'deepgram':
      return (
        <ConfigureDeepgramSpeechToText
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'openai':
      return (
        <ConfigureOpenAISpeechToText
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'azure':
    case 'azure-cloud':
      return (
        <ConfigureAzureSpeechToText
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'assemblyai':
      return (
        <ConfigureAssemblyAISpeechToText
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'cartesia':
      return (
        <ConfigureCartesiaSpeechToText
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) => {
            updateConfig({ parameters: params });
          }}
        />
      );
    default:
      return null;
  }
};
