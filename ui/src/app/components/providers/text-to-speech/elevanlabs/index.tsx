import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import {
  ELEVANLABS_VOICE,
  ELEVANLABS_MODELS,
  ELEVANLABS_LANGUAGES,
} from '@/app/components/providers/text-to-speech/elevanlabs/constant';
import { useState } from 'react';
export {
  GetElevanLabDefaultOptions,
  ValidateElevanLabOptions,
} from './constant';

const renderOption = (c: { name: string }) => {
  return (
    <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
      <span className="truncate capitalize">{c.name}</span>
    </span>
  );
};

export const ConfigureElevanLabTextToSpeech: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const [filteredVoices, setFilteredVoices] = useState(ELEVANLABS_VOICE);

  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  //
  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...(parameters || [])];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    onParameterChange(updatedParams);
  };

  return (
    <>
      <FieldSet className="col-span-1" key="speak.voice.id">
        <FormLabel>Voice</FormLabel>
        <Dropdown
          searchable
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={filteredVoices.find(
            x => x.voice_id === getParamValue('speak.voice.id'),
          )}
          setValue={(v: { voice_id: string }) =>
            updateParameter('speak.voice.id', v.voice_id)
          }
          allValue={filteredVoices} // Updated to use `filteredVoices` state
          onSearching={t => {
            if (t?.target.value) {
              setFilteredVoices(
                ELEVANLABS_VOICE.filter(voice =>
                  voice.name.toLowerCase().includes(t.target.value),
                ),
              );
              return;
            }
            setFilteredVoices(ELEVANLABS_VOICE);
          }}
          placeholder="Select voice"
          option={renderOption}
          label={renderOption}
        />
      </FieldSet>

      <FieldSet className="col-span-1" key="speak.model">
        <FormLabel>Models</FormLabel>
        <Dropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={ELEVANLABS_MODELS.find(
            x => x.model_id === getParamValue('speak.model'),
          )}
          setValue={(v: { model_id: string }) =>
            updateParameter('speak.model', v.model_id)
          }
          allValue={ELEVANLABS_MODELS}
          placeholder="Select model"
          option={renderOption}
          label={renderOption}
        />
      </FieldSet>

      <FieldSet className="col-span-1" key="speak.language">
        <FormLabel>Language</FormLabel>
        <Dropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={ELEVANLABS_LANGUAGES.find(
            x => x.language_id === getParamValue('speak.language'),
          )}
          setValue={(v: { language_id: string }) =>
            updateParameter('speak.language', v.language_id)
          }
          allValue={ELEVANLABS_LANGUAGES}
          placeholder="Select language"
          option={renderOption}
          label={renderOption}
        />
      </FieldSet>
    </>
  );
};
