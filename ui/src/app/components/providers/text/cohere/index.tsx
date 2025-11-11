import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IButton } from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Popover } from '@/app/components/Popover';
import { cn } from '@/styles/media';
import { Bolt, X } from 'lucide-react';
import { useState } from 'react';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { InputHelper } from '@/app/components/input-helper';
import { Textarea } from '@/app/components/Form/Textarea';
import { COHERE_TEXT_MODEL } from '@/app/components/providers/text/cohere/constants';
import { Select } from '@/app/components/Form/Select';

export const ConfigureCohereTextProviderModel: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

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

  const [open, setOpen] = useState(false);
  return (
    <div className="flex-1 flex ">
      <Dropdown
        className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!"
        currentValue={COHERE_TEXT_MODEL.find(
          x =>
            x.id === getParamValue('model.id') &&
            x.name === getParamValue('model.name'),
        )}
        setValue={v => {
          const updatedParams = [...(parameters || [])];
          const newIdParam = new Metadata();
          const newNameParam = new Metadata();

          newIdParam.setKey('model.id');
          newIdParam.setValue(v.id);
          newNameParam.setKey('model.name');
          newNameParam.setValue(v.name);

          // Remove existing parameters if they exist
          const filteredParams = updatedParams.filter(
            p => p.getKey() !== 'model.id' && p.getKey() !== 'model.name',
          );
          filteredParams.push(newIdParam, newNameParam);
          onParameterChange(filteredParams);
        }}
        allValue={COHERE_TEXT_MODEL}
        placeholder="Select voice ouput provider"
        option={c => {
          return (
            <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
              <span className="truncate capitalize">{c.name}</span>
            </span>
          );
        }}
        label={c => {
          return (
            <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
              <span className="truncate capitalize">{c.name}</span>
            </span>
          );
        }}
      />
      <div>
        <IButton
          className="bg-white dark:bg-gray-950"
          onClick={() => {
            setOpen(!open);
          }}
        >
          {open ? (
            <X className={cn('w-4 h-4')} strokeWidth="1.5" />
          ) : (
            <Bolt className={cn('w-4 h-4')} strokeWidth="1.5" />
          )}
        </IButton>
        <Popover
          align={'bottom-end'}
          open={open}
          setOpen={setOpen}
          className="z-50 min-w-fit p-6 grid grid-cols-3 gap-6"
        >
          <FieldSet>
            <FormLabel>Max Tokens</FormLabel>
            <Input
              type="number"
              min={1}
              value={getParamValue('model.max_tokens')}
              onChange={e =>
                updateParameter('model.max_tokens', e.target.value)
              }
            />
            <InputHelper className="text-xs">
              Sets the maximum number of tokens to generate. A low value may
              truncate the output.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Temperature</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0}
                max={2}
                step={0.1}
                value={getParamValue('model.temperature')}
                onSlide={c => updateParameter('model.temperature', c)}
              />
              <Input
                type="number"
                min={0}
                max={2}
                step={0.1}
                value={getParamValue('model.temperature')}
                onChange={v =>
                  updateParameter('model.temperature', v.target.value)
                }
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Controls randomness in generation. Range: 0.0+ (default: 0.3).
              Higher values = more diverse outputs.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>P (Top-p)</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0.01}
                max={0.99}
                step={0.01}
                value={getParamValue('model.p')}
                onSlide={c => updateParameter('model.p', c)}
              />
              <Input
                type="number"
                min={0.01}
                max={0.99}
                step={0.01}
                value={getParamValue('model.p')}
                onChange={v => updateParameter('model.p', v.target.value)}
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Top-p (nucleus) sampling. Range: 0.01 to 0.99 (default: 0.75).
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>K (Top-K)</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0}
                max={500}
                step={1}
                value={getParamValue('model.k')}
                onSlide={c => updateParameter('model.k', c)}
              />
              <Input
                type="number"
                min={0}
                max={500}
                step={1}
                value={getParamValue('model.k')}
                onChange={v => updateParameter('model.k', v.target.value)}
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Top-K sampling. Range: 0 to 500 (default: 0, disabled).
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Frequency Penalty</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0}
                max={1}
                step={0.1}
                value={getParamValue('model.frequency_penalty')}
                onSlide={c => updateParameter('model.frequency_penalty', c)}
              />
              <Input
                type="number"
                min={0}
                max={1}
                step={0.1}
                value={getParamValue('model.frequency_penalty')}
                onChange={v =>
                  updateParameter('model.frequency_penalty', v.target.value)
                }
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Reduces repetition by penalizing tokens based on how often they've
              appeared. Range: 0.0 to 1.0 (default: 0.0).
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Presence Penalty</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0}
                max={1}
                step={0.1}
                value={getParamValue('model.presence_penalty')}
                onSlide={c => updateParameter('model.presence_penalty', c)}
              />
              <Input
                type="number"
                min={0}
                max={1}
                step={0.1}
                value={getParamValue('model.presence_penalty')}
                onChange={v =>
                  updateParameter('model.presence_penalty', v.target.value)
                }
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Discourages reusing any previously seen tokens, regardless of
              frequency. Range: 0.0 to 1.0 (default: 0.0).
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Stop Sequences</FormLabel>
            <Input
              type="text"
              placeholder="Comma-separated sequences"
              value={getParamValue('model.stop_sequences')}
              onChange={e =>
                updateParameter('model.stop_sequences', e.target.value)
              }
            />
            <InputHelper className="text-xs">
              Up to 5 sequences where the API will stop generating tokens.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Safety Mode</FormLabel>
            <Select
              value={getParamValue('model.safety_mode')}
              onChange={e =>
                updateParameter('model.safety_mode', e.target.value)
              }
              className="bg-light-background"
              options={[
                { name: 'CONTEXTUAL', value: 'CONTEXTUAL' },
                { name: 'STRICT', value: 'STRICT' },
                { name: 'OFF', value: 'OFF' },
              ]}
            ></Select>
            <InputHelper className="text-xs">
              Controls built-in safety filtering. Only available on newer
              models.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Seed</FormLabel>
            <Input
              type="number"
              value={getParamValue('model.seed')}
              onChange={e => updateParameter('model.seed', e.target.value)}
            />
            <InputHelper className="text-xs">
              Ensures repeatable generations. Same seed + same parameters =
              similar output. Not fully guaranteed.
            </InputHelper>
          </FieldSet>

          <FieldSet className="col-span-2">
            <FormLabel>Response Format</FormLabel>
            <Textarea
              placeholder="Enter as JSON"
              value={JSON.stringify(getParamValue('model.response_format'))}
              onChange={e =>
                updateParameter(
                  'model.response_format',
                  JSON.parse(e.target.value),
                )
              }
            />
            <InputHelper className="text-xs">
              Specifies the format for model output.
            </InputHelper>
          </FieldSet>
        </Popover>
      </div>
    </div>
  );
};
