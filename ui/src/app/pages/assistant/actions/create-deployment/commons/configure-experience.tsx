import { FC, ReactNode } from 'react';
import { TextArea, Stack } from '@/app/components/carbon/form';
import {
  Button,
  ComboBox,
  FormLabel,
  Slider,
  Toggletip,
  ToggletipActions,
  ToggletipButton,
  ToggletipContent,
  Toggle,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';

export interface ExperienceConfig {
  greeting?: string;
  greetingInterruptible?: boolean;
  messageOnError?: string;
  unclearInputTimeout?: string;
  unclearInputMessage?: string;
  idealTimeout?: string;
  idealMessage?: string;
  maxCallDuration?: string;
  idleTimeoutBackoffTimes?: string;
  suggestions?: string[];
}

const EXPERIENCE_DOCS_URL =
  'https://doc.rapida.ai/assistants/configuration/experience';

const ERROR_MESSAGE_OPTIONS = [
  'Sorry, something went wrong. Please try again.',
  'I’m having trouble completing that right now.',
  'Sorry, I couldn’t process that. Could you try again?',
  'Something went wrong on my side. Please repeat that.',
  'I’m sorry, I ran into an issue. Let’s try again.',
];

export const DEFAULT_UNCLEAR_INPUT_TIMEOUT = '0.5';
export const DEFAULT_UNCLEAR_INPUT_MESSAGE =
  "I didn't catch that. Could you repeat that?";

const UNCLEAR_INPUT_MESSAGE_OPTIONS = [
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  'Sorry, I missed that. Could you say it again?',
  'I could not hear that clearly. Please repeat it.',
  'Could you repeat that once more?',
  'I did not get that clearly. Can you try again?',
];

const IDLE_MESSAGE_OPTIONS = [
  'Are you there?',
  'Are you still with me?',
  'Take your time. I’m here when you’re ready.',
  'I didn’t hear anything. Would you like to continue?',
  'Let me know if you need anything else.',
];

export const ConfigureExperience: FC<{
  experienceConfig: ExperienceConfig;
  setExperienceConfig: (config: ExperienceConfig) => void;
}> = ({ experienceConfig, setExperienceConfig }) => {
  const update = (field: keyof ExperienceConfig, value: string) =>
    setExperienceConfig({ ...experienceConfig, [field]: value });
  const updateBoolean = (field: keyof ExperienceConfig, value: boolean) =>
    setExperienceConfig({ ...experienceConfig, [field]: value });

  const labelWithToggletip = (
    label: string,
    content: ReactNode,
    labelId?: string,
  ) => (
    <div className="form-wrapper">
      <FormLabel id={labelId}>{label}</FormLabel>
      <Toggletip align="bottom-left">
        <ToggletipButton
          label={`${label} information`}
          title={`${label} information`}
        >
          <Information size={14} />
        </ToggletipButton>
        <ToggletipContent>
          <p>{content}</p>
          <ToggletipActions>
            <Button
              kind="primary"
              size="sm"
              href={EXPERIENCE_DOCS_URL}
              target="_blank"
              rel="noreferrer"
            >
              Read more
            </Button>
          </ToggletipActions>
        </ToggletipContent>
      </Toggletip>
    </div>
  );

  const getSelectedMessage = (value: string | undefined) => value || null;

  const handleMessageSelection = (
    field: keyof ExperienceConfig,
    selectedItem?: string | null,
    inputValue?: string | null,
  ) => {
    update(field, selectedItem ?? inputValue ?? '');
  };

  return (
    <div className="max-w-4xl p-6">
      <Stack gap={6}>
        <Stack gap={3}>
          {labelWithToggletip(
            'Greeting',
            <>
              Opening message sent when a session starts. Supports{' '}
              {'{{variable}}'} syntax for dynamic content.
            </>,
          )}
          <TextArea
            id="experience-greeting"
            labelText="Greeting"
            hideLabel
            rows={3}
            value={experienceConfig.greeting || ''}
            onChange={e => update('greeting', e.target.value)}
            placeholder="Write a custom greeting message. You can use {{variable}} to include dynamic content."
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Allow users to interrupt greeting',
            'When enabled, user speech can interrupt the opening greeting. When disabled, input is ignored until the greeting finishes.',
            'experience-greeting-interruptible-label',
          )}
          <Toggle
            id="experience-greeting-interruptible"
            aria-labelledby="experience-greeting-interruptible-label"
            hideLabel
            toggled={experienceConfig.greetingInterruptible ?? true}
            onToggle={checked =>
              updateBoolean('greetingInterruptible', checked)
            }
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Error Message',
            'Fallback message sent when the assistant encounters an error during the session.',
          )}
          <ComboBox
            id="experience-error-message"
            aria-label="Error Message"
            items={ERROR_MESSAGE_OPTIONS}
            itemToString={item => item || ''}
            selectedItem={getSelectedMessage(experienceConfig.messageOnError)}
            allowCustomValue
            placeholder="Message sent to the user when an error occurs"
            onChange={({ selectedItem, inputValue }) =>
              handleMessageSelection('messageOnError', selectedItem, inputValue)
            }
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Unclear Speech Wait (Seconds)',
            'Seconds to wait after an interrupted or unclear user turn before asking the user to repeat.',
          )}
          <Slider
            id="experience-unclear-input-timeout"
            labelText="Unclear Speech Wait (Seconds)"
            hideLabel
            min={0.5}
            max={5}
            step={0.1}
            value={parseFloat(
              experienceConfig.unclearInputTimeout ||
                DEFAULT_UNCLEAR_INPUT_TIMEOUT,
            )}
            onChange={({ value }: { value: number }) =>
              update('unclearInputTimeout', value.toString())
            }
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Unclear Speech Message',
            'Message sent when the user interrupted but did not continue, or speech was not recognized confidently.',
          )}
          <ComboBox
            id="experience-unclear-input-message"
            aria-label="Unclear Speech Message"
            items={UNCLEAR_INPUT_MESSAGE_OPTIONS}
            itemToString={item => item || ''}
            selectedItem={getSelectedMessage(
              experienceConfig.unclearInputMessage ||
                DEFAULT_UNCLEAR_INPUT_MESSAGE,
            )}
            allowCustomValue
            placeholder="Message spoken when the assistant could not understand the user"
            onChange={({ selectedItem, inputValue }) =>
              handleMessageSelection(
                'unclearInputMessage',
                selectedItem,
                inputValue,
              )
            }
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Idle Silence Timeout (Seconds)',
            'Seconds of user silence before the assistant sends the idle message.',
          )}
          <Slider
            id="experience-idle-timeout"
            labelText="Idle Silence Timeout (Seconds)"
            hideLabel
            min={5}
            max={120}
            step={1}
            value={parseInt(experienceConfig.idealTimeout || '30')}
            onChange={({ value }: { value: number }) =>
              update('idealTimeout', value.toString())
            }
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Idle Timeout Backoff (Times)',
            'Number of idle prompts before the session stops waiting for a response.',
          )}
          <Slider
            id="experience-backoff"
            labelText="Idle Timeout Backoff (Times)"
            hideLabel
            min={0}
            max={5}
            step={1}
            value={parseInt(experienceConfig.idleTimeoutBackoffTimes || '2')}
            onChange={({ value }: { value: number }) =>
              update('idleTimeoutBackoffTimes', value.toString())
            }
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Idle Message',
            'Message sent when the user has not responded before the idle silence timeout.',
          )}
          <ComboBox
            id="experience-idle-message"
            aria-label="Idle Message"
            items={IDLE_MESSAGE_OPTIONS}
            itemToString={item => item || ''}
            selectedItem={getSelectedMessage(experienceConfig.idealMessage)}
            allowCustomValue
            placeholder="Message spoken when the user hasn't responded"
            onChange={({ selectedItem, inputValue }) =>
              handleMessageSelection('idealMessage', selectedItem, inputValue)
            }
          />
        </Stack>

        <Stack gap={3}>
          {labelWithToggletip(
            'Maximum Session Duration (Seconds)',
            'Maximum session length in seconds before the assistant ends the session.',
          )}
          <Slider
            id="experience-max-duration"
            labelText="Maximum Session Duration (Seconds)"
            hideLabel
            min={180}
            max={600}
            step={1}
            value={parseInt(experienceConfig.maxCallDuration || '300')}
            onChange={({ value }: { value: number }) =>
              update('maxCallDuration', value.toString())
            }
          />
        </Stack>
      </Stack>
    </div>
  );
};
