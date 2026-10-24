import { useCallback, useState } from 'react';
import { SpeechToTextProvider } from '@/app/components/domain/providers/speech-to-text';
import { NoiseCancellationProvider } from '@/app/components/domain/providers/noise-removal';
import { GetDefaultNoiseCancellationConfig } from '@/app/components/domain/providers/noise-removal/provider';
import { EndOfSpeechProvider } from '@/app/components/domain/providers/end-of-speech';
import { Metadata } from '@rapidaai/react';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
} from '@/app/components/domain/providers/speech-to-text/provider';
import {
  EOS_MODEL_PATH_KEYS,
  GetDefaultEOSConfig,
} from '@/app/components/domain/providers/end-of-speech/provider';
import { GetDefaultVADConfig } from '@/app/components/domain/providers/vad/provider';
import { VADProvider } from '@/app/components/domain/providers/vad';
import { ChevronDown } from '@carbon/icons-react';
import { cn } from '@/utils';
import {
  BargeInTriggerControl,
  MICROPHONE_BARGE_IN_TRIGGER_KEY,
} from '@/app/components/domain/providers/microphone/barge-in-trigger-control';
import { Text } from '@/app/components/ui/primitives/text';

interface ConfigureAudioInputProviderProps {
  audioInputConfig: { provider: string; parameters: Metadata[] };
  setAudioInputConfig: (config: {
    provider: string;
    parameters: Metadata[];
  }) => void;
}

export const ConfigureAudioInputProvider: React.FC<
  ConfigureAudioInputProviderProps
> = ({ audioInputConfig, setAudioInputConfig }) => {
  const [showAdvanced, setShowAdvanced] = useState(false);

  const keepAdvancedMicrophoneParams = (parameters: Metadata[]) =>
    parameters.filter(p => {
      const key = p.getKey();
      return (
        key.startsWith('microphone.eos.') ||
        key === MICROPHONE_BARGE_IN_TRIGGER_KEY ||
        key.startsWith('microphone.vad.') ||
        key.startsWith('microphone.denoising.')
      );
    });

  const onChangeAudioInputProvider = (providerName: string) => {
    const microphoneScopedParameters = keepAdvancedMicrophoneParams(
      audioInputConfig.parameters,
    );
    setAudioInputConfig({
      provider: providerName,
      parameters: GetDefaultSpeechToTextIfInvalid(
        providerName,
        GetDefaultMicrophoneConfig(microphoneScopedParameters),
      ),
    });
  };

  const onChangeAudioInputParameter = (parameters: Metadata[]) => {
    if (audioInputConfig)
      setAudioInputConfig({ ...audioInputConfig, parameters });
  };

  const getParamValue = useCallback(
    (key: string, defaultValue: any) => {
      const param = audioInputConfig.parameters?.find(p => p.getKey() === key);
      return param ? param.getValue() : defaultValue;
    },
    [audioInputConfig.parameters],
  );

  return (
    <div className="flex flex-col gap-6 max-w-4xl p-6">
      <SpeechToTextProvider
        onChangeProvider={onChangeAudioInputProvider}
        onChangeParameter={onChangeAudioInputParameter}
        provider={audioInputConfig.provider}
        parameters={audioInputConfig.parameters}
      />
      {audioInputConfig.provider && (
        <>
          <button
            type="button"
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="flex items-center gap-1.5 text-[var(--cds-text-secondary)] transition-colors hover:text-[var(--cds-text-primary)]"
          >
            <ChevronDown
              size={16}
              className={cn(
                'transition-transform duration-200',
                showAdvanced && 'rotate-180',
              )}
            />
            <Text as="span" variant="body-compact-01">
              {showAdvanced ? 'Hide' : 'Show'} advanced settings
            </Text>
          </button>

          {showAdvanced && (
            <div className="flex flex-col gap-6 pt-6 border-t border-gray-200 dark:border-gray-800">
              <Text
                as="p"
                variant="label-01"
                className="uppercase text-[var(--cds-text-secondary)]"
              >
                Barge-in Control
              </Text>
              <BargeInTriggerControl
                parameters={audioInputConfig.parameters}
                onChangeParameter={onChangeAudioInputParameter}
              />
              <Text
                as="p"
                variant="label-01"
                className="uppercase text-[var(--cds-text-secondary)]"
              >
                Voice Activity Detection
              </Text>
              <VADProvider
                provider={getParamValue(
                  'microphone.vad.provider',
                  'silero_vad',
                )}
                onChangeProvider={v =>
                  onChangeAudioInputParameter(
                    GetDefaultVADConfig(v, audioInputConfig.parameters),
                  )
                }
                parameters={audioInputConfig.parameters}
                onChangeParameter={onChangeAudioInputParameter}
              />
              <Text
                as="p"
                variant="label-01"
                className="uppercase text-[var(--cds-text-secondary)]"
              >
                Background Noise
              </Text>
              <NoiseCancellationProvider
                noiseCancellationProvider={getParamValue(
                  'microphone.denoising.provider',
                  'rn_noise',
                )}
                parameters={audioInputConfig.parameters}
                onChangeParameter={onChangeAudioInputParameter}
                onChangeNoiseCancellationProvider={v =>
                  onChangeAudioInputParameter(
                    GetDefaultNoiseCancellationConfig(
                      v,
                      audioInputConfig.parameters,
                    ),
                  )
                }
              />
              <Text
                as="p"
                variant="label-01"
                className="uppercase text-[var(--cds-text-secondary)]"
              >
                End of Speech
              </Text>
              <EndOfSpeechProvider
                provider={getParamValue(
                  'microphone.eos.provider',
                  'pipecat_smart_turn_eos',
                )}
                onChangeProvider={provider =>
                  onChangeAudioInputParameter(
                    GetDefaultEOSConfig(
                      provider,
                      audioInputConfig.parameters.filter(
                        p =>
                          !p.getKey().startsWith('microphone.eos.') ||
                          EOS_MODEL_PATH_KEYS.has(p.getKey()),
                      ),
                    ),
                  )
                }
                parameters={audioInputConfig.parameters}
                onChangeParameter={onChangeAudioInputParameter}
              />
            </div>
          )}
        </>
      )}
    </div>
  );
};
