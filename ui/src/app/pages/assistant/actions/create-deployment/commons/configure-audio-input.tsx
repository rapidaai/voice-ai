import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { SwitchWithLabel } from '@/app/components/Form/Switch';
import { InputGroup } from '@/app/components/input-group';
import { InputHelper } from '@/app/components/input-helper';
import { ProviderConfig } from '@/app/components/providers';
import { cn } from '@/styles/media';
import { useCallback } from 'react';
import { SpeechToTextProvider } from '@/app/components/providers/speech-to-text';
import { ConditionalInputGroup } from '@/app/components/conditional-input-group';

/**
 *
 * @param param0
 * @returns
 */

export const ConfigureAudioInputProvider: React.FC<{
  onChangeProvider: (i: string, v: string) => void;
  onChangeConfig: (config: ProviderConfig) => void;
  config: ProviderConfig | null;
  voiceInputEnable: boolean;
  onChangeVoiceInputEnable: (b: boolean) => void;
}> = ({
  onChangeProvider,
  onChangeConfig,
  config,
  voiceInputEnable,
  onChangeVoiceInputEnable,
}) => {
  //
  const updateConfig = (newConfig: Partial<ProviderConfig>) => {
    onChangeConfig({ ...config, ...newConfig } as ProviderConfig);
  };

  return (
    <ConditionalInputGroup
      title="Voice Input"
      enable={voiceInputEnable}
      onChangeEnable={onChangeVoiceInputEnable}
    >
      <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
        <SpeechToTextProvider
          onChangeProvider={onChangeProvider}
          onChangeConfig={onChangeConfig}
          config={config}
        />
      </div>
      {config?.provider && (
        <ConfigureMicrophoneExperience
          config={config}
          onChangeConfig={updateConfig}
        />
      )}
    </ConditionalInputGroup>
  );
};

const ConfigureMicrophoneExperience: React.FC<{
  onChangeConfig: (config: Partial<ProviderConfig>) => void;
  config: ProviderConfig;
}> = ({ onChangeConfig, config }) => {
  //
  const getParamValue = useCallback(
    (key: string, defaultValue: any) => {
      const param = config?.parameters?.find(p => p.getKey() === key);
      return param ? param.getValue() : defaultValue;
    },
    [config],
  );

  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...config?.parameters];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }

    onChangeConfig({ parameters: updatedParams });
  };

  const isEosEnabled =
    getParamValue('microphone.eos.enable', 'true') === 'true';
  const isDenoisingEnabled =
    getParamValue('microphone.denoising', 'true') === 'true';

  return (
    <InputGroup initiallyExpanded={false} title="Microphone Experience">
      <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
        <div className="space-y-6 w-full max-w-6xl">
          <div className="grid grid-cols-2 gap-8">
            <FieldSet className="col-span-1">
              <FormLabel className="normal-case">
                Rapida end of speech
              </FormLabel>
              <SwitchWithLabel
                className="bg-light-background"
                enable={isEosEnabled}
                setEnable={v => {
                  updateParameter(
                    'microphone.eos.enable',
                    v ? 'true' : 'false',
                  );
                }}
                label="Rapida to finalize the end of speech"
              />
              <InputHelper>
                Use NLP for sentence boundary, semantic end of speech with
                conversation context and segment silence timeout to trigger.
              </InputHelper>
            </FieldSet>
            <FieldSet className="col-span-1">
              <FormLabel>Segmentation Silence Timeout</FormLabel>
              <div className="flex space-x-2 justify-center items-center">
                <Slider
                  min={500}
                  max={4000}
                  step={100}
                  value={parseInt(
                    getParamValue('microphone.eos.timeout', '1500'),
                  )}
                  onSlide={v => {
                    updateParameter('microphone.eos.timeout', v.toString());
                  }}
                />
                <Input
                  min={500}
                  max={4000}
                  className="bg-light-background w-16"
                  value={getParamValue('microphone.eos.timeout', '1500')}
                  onChange={e =>
                    updateParameter('microphone.eos.timeout', e.target.value)
                  }
                />
              </div>
              <InputHelper>
                Duration of silence after which Rapida starts finalizing a
                message EOS: Based on silence and max time (1000-4000ms).
              </InputHelper>
            </FieldSet>
          </div>
          <FieldSet>
            <FormLabel className="normal-case">
              Background Denoising Enabled
            </FormLabel>
            <SwitchWithLabel
              className="bg-light-background"
              enable={isDenoisingEnabled}
              setEnable={v => {
                updateParameter('microphone.denoising', v ? 'true' : 'false');
              }}
              label="Filter background noise while the user is talking."
            />
          </FieldSet>
          <FieldSet className="w-1/2">
            <FormLabel>User Idle Silence Timeout</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={3000}
                max={10000}
                step={500}
                value={parseInt(
                  getParamValue('microphone.silence.timeout', '5000'),
                )}
                onSlide={v => {
                  updateParameter('microphone.silence.timeout', v.toString());
                }}
              />
              <Input
                className="bg-light-background w-16"
                value={getParamValue('microphone.silence.timeout', '5000')}
                onChange={e =>
                  updateParameter('microphone.silence.timeout', e.target.value)
                }
              />
            </div>
            <InputHelper>
              Duration of silence after which Rapida finalizes a phrase
              (3000-10000ms).
            </InputHelper>
          </FieldSet>
          <FieldSet className="relative col-span-1">
            <FormLabel>Idle Message</FormLabel>
            <Input
              className="bg-light-background"
              placeholder="Message that the assistant will speak when the user hasn't responded."
              value={getParamValue(
                'microphone.idle.message',
                'Are you still there?',
              )}
              onChange={e =>
                updateParameter('microphone.idle.message', e.target.value)
              }
            />
            <InputHelper>
              Message that the assistant will speak when the user hasn't
              responded.
            </InputHelper>
          </FieldSet>
        </div>
      </div>
    </InputGroup>
  );
};
