import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { AZURE_LANGUAGE } from '@/app/components/providers/speech-to-text/azure/constant';
export {
  GetAzureDefaultOptions,
  ValidateAzureOptions,
} from '@/app/components/providers/speech-to-text/azure/constant';

const renderOption = (c: { icon: React.ReactNode; name: string }) => (
  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
    {c.icon}
    <span className="truncate capitalize">{c.name}</span>
  </span>
);

export const ConfigureAzureSpeechToText: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
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

  const configItems = [
    {
      label: 'Language',
      key: 'listen.language',
      options: AZURE_LANGUAGE,
      findMatch: (val: string) => AZURE_LANGUAGE.find(x => x.code === val),
      onChange: (v: { code: string }) => {
        updateParameter('listen.language', v.code);
      },
    },
  ];

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
