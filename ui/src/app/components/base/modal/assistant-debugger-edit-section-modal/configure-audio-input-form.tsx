import { useCallback, useState } from 'react';
import { Stack } from '@/app/components/carbon/form';
import { SpeechToTextProvider } from '@/app/components/providers/speech-to-text';
import { NoiseCancellationProvider } from '@/app/components/providers/noise-removal';
import { GetDefaultNoiseCancellationConfig } from '@/app/components/providers/noise-removal/provider';
import { EndOfSpeechProvider } from '@/app/components/providers/end-of-speech';
import { Metadata } from '@rapidaai/react';
import {
  GetDefaultMicrophoneConfig,
  GetDefaultSpeechToTextIfInvalid,
} from '@/app/components/providers/speech-to-text/provider';
import { GetDefaultEOSConfig } from '@/app/components/providers/end-of-speech/provider';
import { GetDefaultVADConfig } from '@/app/components/providers/vad/provider';
import { VADProvider } from '@/app/components/providers/vad';
import { ChevronDown } from '@carbon/icons-react';
import { cn } from '@/utils';
import {
  BargeInTriggerControl,
  MICROPHONE_BARGE_IN_TRIGGER_KEY,
} from '@/app/components/providers/microphone/barge-in-trigger-control';

interface ConfigureAudioInputModalFormProps {
  audioInputConfig: { provider: string; parameters: Metadata[] };
  setAudioInputConfig: (config: {
    provider: string;
    parameters: Metadata[];
  }) => void;
}

export const ConfigureAudioInputModalForm: React.FC<
  ConfigureAudioInputModalFormProps
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
    <Stack gap={6}>
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
            className="flex items-center gap-1.5 text-xs font-medium text-gray-500 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
          >
            <ChevronDown
              size={16}
              className={cn(
                'transition-transform duration-200',
                showAdvanced && 'rotate-180',
              )}
            />
            {showAdvanced ? 'Hide' : 'Show'} advanced settings
          </button>

          {showAdvanced && (
            <div className="space-y-6 pt-6 border-t border-gray-200 dark:border-gray-800">
              <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                Barge-in Control
              </p>
              <BargeInTriggerControl
                parameters={audioInputConfig.parameters}
                onChangeParameter={onChangeAudioInputParameter}
              />
              <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                Voice Activity Detection
              </p>
              <VADProvider
                provider={getParamValue('microphone.vad.provider', 'silero_vad')}
                onChangeProvider={v =>
                  onChangeAudioInputParameter(
                    GetDefaultVADConfig(v, audioInputConfig.parameters),
                  )
                }
                parameters={audioInputConfig.parameters}
                onChangeParameter={onChangeAudioInputParameter}
              />
              <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                Background Noise
              </p>
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
              <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
                End of Speech
              </p>
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
                        p => !p.getKey().startsWith('microphone.eos.'),
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
    </Stack>
  );
};
