import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/form/fieldset';
import { Input } from '@/app/components/form/input';
import { Slider } from '@/app/components/form/slider';
import { InputHelper } from '@/app/components/input-helper';
import { GOOGLE_SPEECH_TO_TEXT_MODEL } from '@/providers/index';

export {
  GetGoogleDefaultOptions,
  ValidateGoogleOptions,
} from '@/app/components/providers/speech-to-text/google/constant';

export const ConfigureGoogleSpeechToText: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  //
  const updateParameter = (updates: Array<{ key: string; value: string }>) => {
    const updatedParams = [...(parameters || [])];

    updates.forEach(({ key, value }) => {
      const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
      const newParam = new Metadata();
      newParam.setKey(key);
      newParam.setValue(value);
      if (existingIndex >= 0) {
        updatedParams[existingIndex] = newParam;
      } else {
        updatedParams.push(newParam);
      }
    });

    onParameterChange(updatedParams);
  };
  const getAvailableModels = () => {
    const region = getParamValue('listen.region');
    return GOOGLE_SPEECH_TO_TEXT_MODEL()[region]?.model || [];
  };

  const getAvailableLanguages = () => {
    const region = getParamValue('listen.region');
    const model = getParamValue('listen.model');
    const langs =
      GOOGLE_SPEECH_TO_TEXT_MODEL()[region]?.model.find(m => m.id === model)
        ?.language_codes || [];
    return langs;
  };

  return (
    <>
      {/* Region Dropdown */}
      <FieldSet>
        <FormLabel>Region</FormLabel>
        <Dropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={getParamValue('listen.region')}
          setValue={(value: string) => {
            updateParameter([
              { key: 'listen.model', value: '' },
              { key: 'listen.language', value: '' },
              { key: 'listen.region', value: value },
            ]);
          }}
          allValue={Object.keys(GOOGLE_SPEECH_TO_TEXT_MODEL())}
          placeholder="Select region"
          option={region => (
            <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
              {region}
            </span>
          )}
          label={region => (
            <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
              {region}
            </span>
          )}
        />
      </FieldSet>

      {/* Model Dropdown */}
      {getAvailableModels().length > 0 && (
        <FieldSet>
          <FormLabel>Model</FormLabel>
          <Dropdown
            className="bg-light-background max-w-full dark:bg-gray-950"
            currentValue={getAvailableModels().find(
              x => x.id === getParamValue('listen.model'),
            )}
            setValue={(model: { id: string; name: string }) => {
              updateParameter([
                { key: 'listen.model', value: model.id },
                { key: 'listen.language', value: '' },
              ]);
            }}
            allValue={getAvailableModels()}
            placeholder="Select model"
            option={model => (
              <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                {model.name}
              </span>
            )}
            label={model => (
              <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                {model.id}
              </span>
            )}
          />
        </FieldSet>
      )}

      {/* Language Dropdown */}
      {getAvailableLanguages().length > 0 && (
        <FieldSet>
          <FormLabel>Language</FormLabel>
          <Dropdown
            multiple={true}
            className="bg-light-background max-w-full dark:bg-gray-950"
            currentValue={
              getParamValue('listen.language')?.split('<|||>') || []
            }
            setValue={v => {
              updateParameter([
                { key: 'listen.language', value: v.at(v.length - 1) },
              ]);
            }}
            allValue={getAvailableLanguages().map(lang => lang.code)}
            placeholder="Select language"
            option={lang => (
              <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                {lang}
              </span>
            )}
            label={lang => (
              <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                {lang.map(x => {
                  return (
                    <span key={x} className="truncate">
                      {x}
                    </span>
                  );
                })}
              </span>
            )}
          />
        </FieldSet>
      )}

      {/* Transcript Confidence */}
      <FieldSet>
        <FormLabel>Transcript Confidence Threshold</FormLabel>
        <div className="flex space-x-2 justify-center items-center">
          <Slider
            min={0.1}
            max={0.9}
            step={0.1}
            value={parseFloat(getParamValue('listen.threshold')) || 0.1}
            onSlide={value => {
              updateParameter([
                { key: 'listen.threshold', value: value.toString() },
              ]);
            }}
          />
          <Input
            type="number"
            min={0.1}
            max={0.9}
            step={0.1}
            value={getParamValue('listen.threshold')}
            onChange={event => {
              updateParameter([
                { key: 'listen.threshold', value: event.target.value },
              ]);
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
