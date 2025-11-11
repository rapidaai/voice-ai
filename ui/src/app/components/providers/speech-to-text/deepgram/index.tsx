import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { Textarea } from '@/app/components/Form/Textarea';
import { InputHelper } from '@/app/components/input-helper';
import {
  DEEPGRAM_LANGUAGES,
  DEEPGRAM_MODELS,
} from '@/app/components/providers/speech-to-text/deepgram/constant';
export {
  GetDeepgramDefaultOptions,
  ValidateDeepgramOptions,
} from '@/app/components/providers/speech-to-text/deepgram/constant';

const renderOption = (c: { icon: React.ReactNode; name: string }) => (
  <span className="inline-flex items-center gap-2 sm:gap-2.5 text-sm font-medium">
    {c.icon}
    <span className="truncate capitalize">{c.name}</span>
  </span>
);

export const ConfigureDeepgramSpeechToText: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  const updateParameter = (key: string, value: string) => {
    const updatedParams = parameters ? parameters.map(p => p.clone()) : [];
    const existingParamIndex = updatedParams.findIndex(p => p.getKey() === key);

    if (existingParamIndex !== -1) {
      updatedParams[existingParamIndex].setValue(value);
    } else {
      const newParam = new Metadata();
      newParam.setKey(key);
      newParam.setValue(value);
      updatedParams.push(newParam);
    }

    onParameterChange(updatedParams);
  };

  const configItems = [
    {
      label: 'Model',
      key: 'listen.model',
      options: DEEPGRAM_MODELS,
      findMatch: (val: string) => DEEPGRAM_MODELS.find(x => x.id === val),
      onChange: (v: { id: string }) => {
        updateParameter('listen.model', v.id);
      },
    },
    {
      label: 'Language',
      key: 'listen.language',
      options: DEEPGRAM_LANGUAGES,
      findMatch: (val: string) => DEEPGRAM_LANGUAGES.find(x => x.code === val),
      onChange: v => {
        updateParameter('listen.language', v.code);
      },
    },
  ];

  return (
    <>
      {configItems.map(({ label, key, options, findMatch, onChange }) => (
        <FieldSet className="col-span-1 h-fit" key={key}>
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
      <FieldSet className="col-span-1">
        <FormLabel>Transcript Confidence Threshold</FormLabel>
        <div className="flex space-x-2 justify-center items-center">
          <Slider
            min={0.1}
            max={0.9}
            step={0.1}
            value={parseFloat(getParamValue('listen.threshold')) || 0.1}
            onSlide={c => {
              updateParameter('listen.threshold', c.toString());
            }}
          />
          <Input
            type="number"
            min={0.1}
            max={0.9}
            step={0.1}
            value={getParamValue('listen.threshold')}
            onChange={v => {
              updateParameter('listen.threshold', v.target.value);
            }}
            className="bg-light-background w-16"
          />
        </div>
        <InputHelper>
          Transcripts with a confidence score below this threshold will be
          filtered out.
        </InputHelper>
      </FieldSet>
      <FieldSet className="col-span-2">
        <FormLabel>Keyword / Keyterms</FormLabel>
        <Textarea
          required={false}
          value={getParamValue('listen.keywords')}
          onChange={v => {
            updateParameter('listen.keywords', v.target.value);
          }}
          rows={2}
          className="bg-light-background"
          placeholder="Enter keywords or key terms separated by space"
        />
        <InputHelper>
          Enter keywords separated by spaces. These will be used as key terms
          for transcription.
        </InputHelper>
      </FieldSet>
    </>
  );
};
