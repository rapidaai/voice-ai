import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import {
  OPENAI_LANGUAGES,
  OPENAI_MODELS,
} from '@/app/components/providers/speech-to-text/openai/constant';

export {
  GetOpenAIDefaultOptions,
  ValidateOpenAIOptions,
} from '@/app/components/providers/speech-to-text/openai/constant';

const renderOption = (c: { icon: React.ReactNode; name: string }) => (
  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
    {c.icon}
    <span className="truncate capitalize">{c.name}</span>
  </span>
);

export const ConfigureOpenAISpeechToText: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
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

  const configItems = [
    {
      label: 'language',
      key: 'listen.language',
      options: OPENAI_LANGUAGES,
      findMatch: (val: string) => OPENAI_LANGUAGES.find(x => x.code === val),
      onChange: (v: { id: string }) => {
        updateParameter('listen.language', v.id);
      },
    },
    {
      label: 'Model',
      key: 'listen.model',
      options: OPENAI_MODELS,
      findMatch: (val: string) => OPENAI_MODELS.find(x => x.id === val),
      onChange: (v: { id: string }) => {
        updateParameter('listen.model', v.id);
      },
    },
  ];

  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  return (
    <>
      {configItems.map(({ label, key, options, findMatch, onChange }) => (
        <FieldSet className="col-span-1" key={key}>
          <FormLabel>{label}</FormLabel>
          <Dropdown
            className="bg-light-background max-w-full dark:bg-gray-950"
            currentValue={findMatch(getParamValue(key))}
            setValue={onChange}
            allValue={options}
            placeholder={`Select ${label.toLowerCase()}`}
            option={renderOption}
            label={renderOption}
          />
        </FieldSet>
      ))}
    </>
  );
};
