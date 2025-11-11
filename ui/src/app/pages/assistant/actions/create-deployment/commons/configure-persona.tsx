import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { InputGroup } from '@/app/components/input-group';
import { InputHelper } from '@/app/components/input-helper';
import { cn } from '@/styles/media';
import { FC } from 'react';

/**
 * Persona configure
 * an interface provide the props
 */
export interface PersonaConfig {
  name?: string;
  role?: string;
  tone?: string;
  expertise?: string;
}

/**
 *
 * @param param0
 * @returns
 */
export const ConfigurePersona: FC<{
  personaConfig: PersonaConfig;
  onChangePersona: (p: PersonaConfig) => void;
}> = ({ personaConfig, onChangePersona }) => {
  /**
   *
   * @param k
   * @param v
   */
  const handleInputChange = (k: string, v) => {
    onChangePersona({ ...personaConfig, [k]: v });
  };

  /**
   *
   */
  return (
    <InputGroup title="Personality">
      <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
        <div className="grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2">
          <FieldSet>
            <FormLabel>Name</FormLabel>
            <Input
              maxLength={32}
              className="bg-light-background"
              placeholder="e.g. Alex"
              type="text"
              required
              value={personaConfig.name || ''}
              onChange={e => handleInputChange('name', e.target.value)}
            />
            <InputHelper>
              Choose a name that reflects the agent's purpose and creates a
              relatable personaConfig. Example: "Alex" for a general assistant,
              "Dr. Watson" for a medical advisor.
            </InputHelper>
          </FieldSet>
          <FieldSet>
            <FormLabel>Role</FormLabel>
            <Input
              className="bg-light-background"
              maxLength={32}
              required
              placeholder="e.g. Support Assistant"
              value={personaConfig.role || ''}
              onChange={e => handleInputChange('role', e.target.value)}
            />
            <InputHelper>
              Define the agent's specific function or job title. This helps set
              expectations about their expertise and responsibilities.
            </InputHelper>
          </FieldSet>
          <FieldSet>
            <FormLabel>Communication Tone</FormLabel>
            <Dropdown
              className="bg-light-background dark:bg-gray-950"
              placeholder="Select communication tone"
              currentValue={personaConfig.tone || ''}
              setValue={value => handleInputChange('tone', value)}
              option={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    <span className="truncate capitalize">{c}</span>
                  </span>
                );
              }}
              label={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    <span className="truncate capitalize">{c}</span>
                  </span>
                );
              }}
              allValue={[
                'Professional',
                'Friendly',
                'Casual',
                'Formal',
                'Enthusiastic',
                'Empathetic',
                'Informative',
                'Humorous',
                'Serious',
                'Neutral',
              ]}
            />
          </FieldSet>
          <FieldSet>
            <FormLabel>Primary expertise</FormLabel>

            <Dropdown
              className="bg-light-background dark:bg-gray-950"
              placeholder="Select expertise"
              currentValue={personaConfig.expertise || ''}
              setValue={value => handleInputChange('expertise', value)}
              option={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    <span className="truncate capitalize">{c}</span>
                  </span>
                );
              }}
              label={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    <span className="truncate capitalize">{c}</span>
                  </span>
                );
              }}
              allValue={[
                'general',
                'tech',
                'science',
                'math',
                'business',
                'finance',
                'health',
                'law',
                'education',
                'arts',
                'sports',
                'cooking',
                'travel',
                'history',
                'psychology',
              ]}
            />
          </FieldSet>
        </div>
      </div>
    </InputGroup>
  );
};
