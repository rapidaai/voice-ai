import ConfigSelect from '@/app/components/configuration/config-var/config-select';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Textarea } from '@/app/components/Form/Textarea';
import { InputGroup } from '@/app/components/input-group';
import { cn } from '@/styles/media';
import { FC } from 'react';

export interface ExperienceConfig {
  greeting: string;
  suggestions: string[];
  messageOnError: string;
}

export const ConfigureExperience: FC<{
  experienceConfig: ExperienceConfig;
  setExperienceConfig: (config: ExperienceConfig) => void;
}> = ({ experienceConfig, setExperienceConfig }) => {
  const { greeting, suggestions, messageOnError } = experienceConfig;

  /**
   *
   * @param newGreeting
   */
  const onChangeGreeting = (newGreeting: string) => {
    setExperienceConfig({ ...experienceConfig, greeting: newGreeting });
  };

  const onChangeSuggestions = (newSuggestions: string[]) => {
    setExperienceConfig({ ...experienceConfig, suggestions: newSuggestions });
  };

  const onChangeMessageOnError = (newMessageOnError: string) => {
    setExperienceConfig({
      ...experienceConfig,
      messageOnError: newMessageOnError,
    });
  };

  return (
    <InputGroup title="General Experience">
      <div className={cn('p-6 pt-2 flex flex-col gap-8')}>
        <div className="flex flex-col space-y-6">
          <FieldSet className="block flex-1 md:col-span-2">
            <FormLabel>Greeting</FormLabel>
            <Textarea
              row={2}
              value={greeting}
              className="bg-light-background"
              onChange={e => onChangeGreeting(e.target.value)}
              placeholder={
                'Describe your agent so that users know how to use it. This will appear as a welcome message.'
              }
            />
          </FieldSet>
          <FieldSet className="block flex-1 md:col-span-2">
            <FormLabel>Quick start questions</FormLabel>
            <ConfigSelect
              options={suggestions}
              label="Add new questions"
              placeholder="Add frequently asked question."
              onChange={onChangeSuggestions}
            />
          </FieldSet>
          <FieldSet>
            <FormLabel>Error Message</FormLabel>
            <Input
              placeholder="Message that will be send to the user when error occured"
              value={messageOnError}
              className="bg-light-background"
              onChange={e => onChangeMessageOnError(e.target.value)}
            />
          </FieldSet>
        </div>
      </div>
    </InputGroup>
  );
};
