import { FC, useState } from 'react';
import { TextInput, TextArea, Stack } from '@/app/components/carbon/form';
import { Slider } from '@carbon/react';
import { ChevronDown } from '@carbon/icons-react';
import { cn } from '@/utils';
import {
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';

export const ConfigureExperienceModalForm: FC<{
  experienceConfig: ExperienceConfig;
  setExperienceConfig: (config: ExperienceConfig) => void;
}> = ({ experienceConfig, setExperienceConfig }) => {
  const [showAdvanced, setShowAdvanced] = useState(false);

  const update = (field: keyof ExperienceConfig, value: string) =>
    setExperienceConfig({ ...experienceConfig, [field]: value });

  return (
    <Stack gap={6}>
        <TextArea
          id="experience-greeting"
          labelText="Greeting"
          rows={3}
          value={experienceConfig.greeting || ''}
          onChange={e => update('greeting', e.target.value)}
          placeholder="Write a custom greeting message. You can use {{variable}} to include dynamic content."
        />

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
          <div className="pt-6 border-t border-gray-200 dark:border-gray-800">
            <Stack gap={6}>
              <TextInput
                id="experience-error-message"
                labelText="Error Message"
                placeholder="Message sent to the user when an error occurs"
                value={experienceConfig.messageOnError || ''}
                onChange={e => update('messageOnError', e.target.value)}
              />

              <Slider
                id="experience-unclear-input-timeout"
                labelText="Unclear Speech Wait (Seconds)"
                min={2}
                max={10}
                step={0.1}
                value={parseFloat(
                  experienceConfig.unclearInputTimeout ||
                    DEFAULT_UNCLEAR_INPUT_TIMEOUT,
                )}
                onChange={({ value }: { value: number }) =>
                  update('unclearInputTimeout', value.toString())
                }
                helperText="How long to wait after interrupted or unclear speech before asking the user to repeat."
              />

              <TextInput
                id="experience-unclear-input-message"
                labelText="Unclear Speech Message"
                placeholder="Message spoken when the assistant could not understand the user"
                value={
                  experienceConfig.unclearInputMessage ||
                  DEFAULT_UNCLEAR_INPUT_MESSAGE
                }
                onChange={e => update('unclearInputMessage', e.target.value)}
              />

              <Slider
                id="experience-idle-timeout"
                labelText="Idle Silence Timeout (Seconds)"
                min={5}
                max={120}
                step={1}
                value={parseInt(experienceConfig.idealTimeout || '30')}
                onChange={({ value }: { value: number }) =>
                  update('idealTimeout', value.toString())
                }
                helperText="Duration of silence after which Rapida will prompt the user (5-120 s)."
              />

              <Slider
                id="experience-backoff"
                labelText="Idle Timeout Backoff (Times)"
                min={0}
                max={5}
                step={1}
                value={parseInt(experienceConfig.idleTimeoutBackoffTimes || '2')}
                onChange={({ value }: { value: number }) =>
                  update('idleTimeoutBackoffTimes', value.toString())
                }
                helperText="How many times the idle timeout multiplies before ending the session."
              />

              <TextInput
                id="experience-idle-message"
                labelText="Idle Message"
                placeholder="Message spoken when the user hasn't responded"
                value={experienceConfig.idealMessage || ''}
                onChange={e => update('idealMessage', e.target.value)}
              />

              <Slider
                id="experience-max-duration"
                labelText="Maximum Session Duration (Seconds)"
                min={180}
                max={600}
                step={1}
                value={parseInt(experienceConfig.maxCallDuration || '300')}
                onChange={({ value }: { value: number }) =>
                  update('maxCallDuration', value.toString())
                }
                helperText="Maximum session length before the call is automatically ended (180–600 s)."
              />
            </Stack>
          </div>
        )}
    </Stack>
  );
};
