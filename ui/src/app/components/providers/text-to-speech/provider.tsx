import { Metadata } from '@rapidaai/react';
import { FC } from 'react';
import {
  ConfigureAzureTextToSpeech,
  GetAzureDefaultOptions,
  ValidateAzureOptions,
} from '@/app/components/providers/text-to-speech/azure';
import {
  ConfigureCartesiaTextToSpeech,
  GetCartesiaDefaultOptions,
  ValidateCartesiaOptions,
} from '@/app/components/providers/text-to-speech/cartesia';
import {
  ConfigureDeepgramTextToSpeech,
  GetDeepgramDefaultOptions,
} from '@/app/components/providers/text-to-speech/deepgram';
import { ValidateDeepgramOptions } from '@/app/components/providers/text-to-speech/deepgram/constant';
import {
  ConfigureElevanLabTextToSpeech,
  GetElevanLabDefaultOptions,
  ValidateElevanLabOptions,
} from '@/app/components/providers/text-to-speech/elevanlabs';
import {
  ConfigureGoogleTextToSpeech,
  GetGoogleDefaultOptions,
  ValidateGoogleOptions,
} from '@/app/components/providers/text-to-speech/google';
import {
  ConfigureOpenAITextToSpeech,
  GetOpenAIDefaultOptions,
  ValidateOpenAIOptions,
} from '@/app/components/providers/text-to-speech/openai';
import {
  ConfigurePlayhtTextToSpeech,
  GetPlayHTDefaultOptions,
  ValidatePlayHTOptions,
} from '@/app/components/providers/text-to-speech/playht';
import { ProviderConfig } from '@/app/components/providers';

/**
 *
 * @returns
 */
export const GetDefaultSpeakerConfig = (
  existing: Metadata[] = [],
): Metadata[] => {
  const defaultConfig = [
    {
      key: 'speaker.sentence.boundaries',
      value: [
        ' .',
        '!',
        '?',
        '|',
        ';',
        ':',
        '…',
        '。',
        '．',
        '।',
        '۔',
        '--',
      ].join('<|||>'),
    },
    {
      key: 'speaker.conjunction.boundaries',
      value: [
        'for',
        'and',
        'nor',
        'but',
        'or',
        'yet',
        'so',
        'after',
        'although',
        'as',
        'because',
        'before',
        'even',
        'if',
        'once',
        'since',
        'so that',
        'than',
        'that',
        'though',
        'unless',
        'until',
        'when',
        'whenever',
        'where',
        'wherever',
        'whereas',
        'whether',
        'while',
      ].join('<|||>'),
    },
    {
      key: 'speaker.conjunction.break',
      value: '240',
    },
    {
      key: 'speaker.pronunciation.dictionaries',
      value: [
        'currency',
        'date',
        'time',
        'numeral',
        'address',
        'url',
        'tech-abbreviation',
        'role-abbreviation',
        'general-abbreviation',
        'symbol',
      ].join('<|||>'),
    },
  ];

  const result = [...existing];

  defaultConfig.forEach(item => {
    if (!existing.some(e => e.getKey() === item.key)) {
      const metadata = new Metadata();
      metadata.setKey(item.key);
      metadata.setValue(item.value);
      result.push(metadata);
    }
  });

  return result;
};

export const GetDefaultTextToSpeechIfInvalid = (
  provider: string,
  parameters: Metadata[],
): Metadata[] => {
  switch (provider) {
    case 'google':
    case 'google-cloud':
      return GetGoogleDefaultOptions(parameters);
    case 'elevenlabs':
      return GetElevanLabDefaultOptions(parameters);
    case 'playht':
      return GetPlayHTDefaultOptions(parameters);
    case 'deepgram':
      return GetDeepgramDefaultOptions(parameters);
    case 'openai':
      return GetOpenAIDefaultOptions(parameters);
    case 'azure':
      return GetAzureDefaultOptions(parameters);
    case 'cartesia':
      return GetCartesiaDefaultOptions(parameters);
    default:
      return parameters;
  }
};

export const ValidateTextToSpeechIfInvalid = (
  provider: string,
  parameters: Metadata[],
): boolean => {
  switch (provider) {
    case 'google':
    case 'google-cloud':
      return ValidateGoogleOptions(parameters);
    case 'elevenlabs':
      return ValidateElevanLabOptions(parameters);
    case 'playht':
      return ValidatePlayHTOptions(parameters);
    case 'deepgram':
      return ValidateDeepgramOptions(parameters);
    case 'openai':
      return ValidateOpenAIOptions(parameters);
    case 'azure':
      return ValidateAzureOptions(parameters);
    case 'cartesia':
      return ValidateCartesiaOptions(parameters);
    default:
      return false;
  }
};

export const TextToSpeechConfigComponent: FC<{
  config: ProviderConfig;
  updateConfig: (config: Partial<ProviderConfig>) => void;
}> = ({ config, updateConfig }) => {
  switch (config?.provider) {
    case 'google':
    case 'google-cloud':
      return (
        <ConfigureGoogleTextToSpeech
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'elevenlabs':
      return (
        <ConfigureElevanLabTextToSpeech
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'playht':
      return (
        <ConfigurePlayhtTextToSpeech
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) => {
            updateConfig({ parameters: params });
          }}
        />
      );
    case 'deepgram':
      return (
        <ConfigureDeepgramTextToSpeech
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'openai':
      return (
        <ConfigureOpenAITextToSpeech
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'azure':
      return (
        <ConfigureAzureTextToSpeech
          parameters={config.parameters}
          onParameterChange={(params: Metadata[]) =>
            updateConfig({ parameters: params })
          }
        />
      );
    case 'cartesia':
      return (
        <ConfigureCartesiaTextToSpeech
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
