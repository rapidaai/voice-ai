import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { InputHelper } from '@/app/components/input-helper';
import {
  SARVAM_ENCODINGS,
  SARVAM_LANGUAGE,
  SARVAM_MODELS,
  SARVAM_SAMPLE_RATES,
} from '@/app/components/providers/speech-to-text/sarvam/constant';

export {
  GetSarvamDefaultOptions,
  ValidateGoogleOptions,
} from '@/app/components/providers/speech-to-text/sarvam/constant';

const renderOption = (c: { icon: React.ReactNode; name: string }) => (
  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
    {c.icon}
    <span className="truncate capitalize">{c.name}</span>
  </span>
);

export const ConfigureSarvamSpeechToText: React.FC<{
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
      options: SARVAM_LANGUAGE,
      findMatch: (val: string) => SARVAM_LANGUAGE.find(x => x.code === val),
      onChange: (v: { code: string }) => {
        updateParameter('listen.language', v.code);
      },
    },
    {
      label: 'Model',
      key: 'listen.model',
      options: SARVAM_MODELS,
      findMatch: (val: string) => SARVAM_MODELS.find(x => x.id === val),
      onChange: (v: { id: string }) => {
        updateParameter('listen.model', v.id);
      },
    },
    {
      label: 'Encoding',
      key: 'listen.output_format.encoding',
      options: SARVAM_ENCODINGS,
      findMatch: (val: string) => SARVAM_ENCODINGS.find(x => x.value === val),
      onChange: v => {
        updateParameter('listen.output_format.encoding', v.value);
      },
    },
    {
      label: 'Sample Rate',
      key: 'listen.output_format.sample_rate',
      options: SARVAM_SAMPLE_RATES,
      findMatch: (val: string) =>
        SARVAM_SAMPLE_RATES.find(x => x.value === val),
      onChange: v => {
        updateParameter('listen.output_format.sample_rate', v.value);
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
    </>
  );
};
