import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import {
  CARTESIA_LANGUAGE,
  CARTESIA_MODELS,
  CARTESIA_VOICE,
  EMOTION_LEVEL_COMBINATIONS,
  SPEED_OPTIONS,
} from '@/app/components/providers/text-to-speech/cartesia/constant';
export { GetCartesiaDefaultOptions, ValidateCartesiaOptions } from './constant';

const renderOption = c => (
  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
    {c.icon}
    <span className="truncate capitalize">{c.name}</span>
  </span>
);

export const ConfigureCartesiaTextToSpeech: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  //
  const getParamValue = (key: string) => {
    return parameters?.find(p => p.getKey() === key)?.getValue() ?? '';
  };

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
      label: 'Voice',
      key: 'speak.voice.id',
      options: CARTESIA_VOICE,
      findMatch: (val: string) => CARTESIA_VOICE.find(x => x.id === val),
      onChange: v => {
        updateParameter('speak.voice.id', v.id);
      },
    },
    {
      label: 'Language',
      key: 'speak.language',
      options: CARTESIA_LANGUAGE,
      findMatch: (val: string) => CARTESIA_LANGUAGE.find(x => x.code === val),
      onChange: v => {
        updateParameter('speak.language', v.code);
      },
    },
    {
      label: 'Models',
      key: 'speak.model',
      options: CARTESIA_MODELS,
      findMatch: (val: string) => CARTESIA_MODELS.find(x => x.id === val),
      onChange: v => {
        updateParameter('speak.model', v.id);
      },
    },

    {
      label: 'Speed (Experimental)',
      key: 'speak.voice.__experimental_controls.speed',
      options: SPEED_OPTIONS,
      findMatch: (val: string) => {
        return SPEED_OPTIONS.find(x => x.id === val) || '';
      },
      onChange: v => {
        updateParameter('speak.voice.__experimental_controls.speed', v.id);
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
            setValue={onChange || (() => {})}
            allValue={options}
            placeholder={`Select ${label.toLowerCase()}`}
            option={renderOption}
            label={renderOption}
          />
        </FieldSet>
      ))}
      <FieldSet className="relative col-span-2">
        <FormLabel>Emotion (Experimental)</FormLabel>
        <Dropdown
          multiple
          className="bg-light-background dark:bg-gray-950 max-w-6xl"
          currentValue={getParamValue(
            'speak.voice.__experimental_controls.emotion',
          ).split('<|||>')}
          setValue={v => {
            updateParameter(
              'speak.voice.__experimental_controls.emotion',
              v.join('<|||>'),
            );
          }}
          allValue={EMOTION_LEVEL_COMBINATIONS}
          placeholder="Select all that applies"
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
                {c.map(x => {
                  return (
                    <span key={x} className="truncate">
                      {x}
                    </span>
                  );
                })}
              </span>
            );
          }}
        />
      </FieldSet>
    </>
  );
};
