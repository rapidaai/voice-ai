import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Textarea } from '@/app/components/Form/Textarea';
import { InputGroup } from '@/app/components/input-group';
import { cn } from '@/styles/media';
import { FC } from 'react';

export interface ExperienceConfig {
  greeting: string;
  messageOnError: string;
}

export const ConfigureExperience: FC<{
  experienceConfig: ExperienceConfig;
  setExperienceConfig: (config: ExperienceConfig) => void;
}> = ({ experienceConfig, setExperienceConfig }) => {
  const { greeting, messageOnError } = experienceConfig;

  /**
   *
   * @param newGreeting
   */
  const onChangeGreeting = (newGreeting: string) => {
    setExperienceConfig({ ...experienceConfig, greeting: newGreeting });
  };

  const onChangeMessageOnError = (newMessageOnError: string) => {
    setExperienceConfig({
      ...experienceConfig,
      messageOnError: newMessageOnError,
    });
  };

  const onChangeMessageOnEnd = (newMessageOnEnd: string) => {
    setExperienceConfig({ ...experienceConfig });
  };

  return (
    <InputGroup title="General Experience">
      <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
        <div className="flex flex-col space-y-6 w-full">
          <FieldSet>
            <FormLabel>Greeting</FormLabel>
            <Textarea
              row={2}
              className="bg-light-background"
              value={greeting}
              onChange={e => onChangeGreeting(e.target.value)}
              placeholder={
                'Describe your agent so that users know how to use it. This will appear as a welcome message.'
              }
            />
          </FieldSet>
          <FieldSet>
            <FormLabel>Error Message</FormLabel>
            <Input
              className="bg-light-background"
              placeholder="Message that will be send to the user when error occured"
              value={messageOnError}
              onChange={e => onChangeMessageOnError(e.target.value)}
            />
          </FieldSet>
        </div>
      </div>
    </InputGroup>
  );
};
