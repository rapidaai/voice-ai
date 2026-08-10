import ConfigSelect from '@/app/components/configuration/config-var/config-select';
import { TextInput, TextArea, Stack } from '@/app/components/carbon/form';
import {
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
  ExperienceConfig,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { Slider } from '@carbon/react';
import { ChevronDown } from '@carbon/icons-react';
import { cn } from '@/utils';
import { FC, useState } from 'react';

export interface WebWidgetExperienceConfig extends ExperienceConfig {
  suggestions: string[];
}

export const ConfigureExperience: FC<{
  experienceConfig: WebWidgetExperienceConfig;
  setExperienceConfig: (config: WebWidgetExperienceConfig) => void;
}> = ({ experienceConfig, setExperienceConfig }) => {
  const [showAdvanced, setShowAdvanced] = useState(false);

  const update = (field: keyof WebWidgetExperienceConfig, value: any) =>
    setExperienceConfig({ ...experienceConfig, [field]: value });

  return (
    <div className="max-w-4xl px-6">
      <Stack gap={6}>
        <TextArea
          id="widget-greeting"
          labelText="Greeting"
          rows={3}
          value={experienceConfig.greeting || ''}
          onChange={e => update('greeting', e.target.value)}
          placeholder="Describe your agent so users know how to use it. This will appear as a welcome message."
        />

        <div>
          <p className="text-xs font-medium mb-2">Quick-start questions</p>
          <ConfigSelect
            options={experienceConfig.suggestions || []}
            label="Add question"
            placeholder="Enter a common user question."
            helperText="Appears as starter prompts in the web widget. Drag to reorder."
            onChange={suggestions => update('suggestions', suggestions)}
          />
        </div>

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
                id="widget-error-message"
                labelText="Error Message"
                placeholder="Message sent to the user when an error occurs"
                value={experienceConfig.messageOnError || ''}
                onChange={e => update('messageOnError', e.target.value)}
              />

              <Slider
                id="widget-unclear-input-timeout"
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
              />

              <TextInput
                id="widget-unclear-input-message"
                labelText="Unclear Speech Message"
                placeholder="Message spoken when the assistant could not understand the user"
                value={
                  experienceConfig.unclearInputMessage ||
                  DEFAULT_UNCLEAR_INPUT_MESSAGE
                }
                onChange={e => update('unclearInputMessage', e.target.value)}
              />

              <Slider
                id="widget-idle-timeout"
                labelText="Idle Silence Timeout (Seconds)"
                min={5}
                max={120}
                step={1}
                value={parseInt(
                  experienceConfig.idealTimeout || DEFAULT_IDEAL_TIMEOUT,
                )}
                onChange={({ value }: { value: number }) =>
                  update('idealTimeout', value.toString())
                }
              />

              <Slider
                id="widget-backoff"
                labelText="Idle Timeout Backoff (Times)"
                min={0}
                max={5}
                step={1}
                value={parseInt(
                  experienceConfig.idleTimeoutBackoffTimes || '2',
                )}
                onChange={({ value }: { value: number }) =>
                  update('idleTimeoutBackoffTimes', value.toString())
                }
              />

              <TextInput
                id="widget-idle-message"
                labelText="Idle Message"
                placeholder="Message spoken when the user hasn't responded"
                value={experienceConfig.idealMessage || ''}
                onChange={e => update('idealMessage', e.target.value)}
              />

              <Slider
                id="widget-max-duration"
                labelText="Maximum Session Duration (Seconds)"
                min={180}
                max={600}
                step={1}
                value={parseInt(experienceConfig.maxCallDuration || '300')}
                onChange={({ value }: { value: number }) =>
                  update('maxCallDuration', value.toString())
                }
              />
            </Stack>
          </div>
        )}
      </Stack>
    </div>
  );
};
