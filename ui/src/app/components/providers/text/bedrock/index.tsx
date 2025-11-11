import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IButton } from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Popover } from '@/app/components/Popover';
import { AZURE_TEXT_MODEL } from '@/app/components/providers/text/azure/constants';
import { cn } from '@/styles/media';
import { Bolt, X } from 'lucide-react';
import { useState } from 'react';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { InputHelper } from '@/app/components/input-helper';
import { Textarea } from '@/app/components/Form/Textarea';

export const ConfigureAzureTextProviderModel: React.FC<{
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

  const [open, setOpen] = useState(true);
  return (
    <div className="flex-1 flex ">
      <Dropdown
        className="bg-light-background max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!"
        currentValue={AZURE_TEXT_MODEL.find(
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
        allValue={AZURE_TEXT_MODEL}
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
            <FormLabel>Frequency Penalty</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={-2}
                max={2}
                step={0.1}
                value={getParamValue('model.frequency_penalty')}
                onSlide={c => {
                  updateParameter('model.frequency_penalty', c);
                }}
              />
              <Input
                type="number"
                min={-2}
                max={2}
                step={0.1}
                value={getParamValue('model.frequency_penalty')}
                onChange={v => {
                  updateParameter('model.frequency_penalty', v.target.value);
                }}
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Number between -2.0 and 2.0. Penalizes new tokens based on their
              existing frequency.
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
                onSlide={c => {
                  updateParameter('model.temperature', c);
                }}
              />
              <Input
                type="number"
                min={0}
                max={2}
                step={0.1}
                value={getParamValue('model.temperature')}
                onChange={v => {
                  updateParameter('model.temperature', v.target.value);
                }}
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              What sampling temperature to use. Higher values make output more
              random.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Top P</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0}
                max={1}
                step={0.05}
                value={getParamValue('model.top_p')}
                onSlide={e => updateParameter('model.top_p', e.target.value)}
              />
              <Input
                type="number"
                min={0}
                max={1}
                step={0.05}
                value={getParamValue('model.top_p')}
                onChange={v => {
                  updateParameter('model.top_p', v.target.value);
                }}
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Alternative to sampling with temperature, for nucleus sampling.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Presence Penalty</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={-2}
                max={2}
                step={0.1}
                value={getParamValue('model.presence_penalty')}
                onSlide={c => {
                  updateParameter('model.presence_penalty', c);
                }}
              />
              <Input
                type="number"
                min={-2}
                max={2}
                step={0.1}
                value={getParamValue('model.presence_penalty')}
                onChange={v => {
                  updateParameter('model.presence_penalty', v.target.value);
                }}
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Number between -2.0 and 2.0. Penalizes new tokens based on whether
              they appear in the text so far.
            </InputHelper>
          </FieldSet>
          <FieldSet>
            <FormLabel>Max Completion Tokens</FormLabel>
            <Input
              type="number"
              value={getParamValue('model.max_completion_tokens')}
              onChange={e =>
                updateParameter('model.max_completion_tokens', e.target.value)
              }
            />
            <InputHelper className="text-xs">
              Upper bound for tokens generated in the completion.
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
              Specifies the format for model output. Enter as JSON.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Stop Sequences</FormLabel>
            <Input
              type="text"
              placeholder="Comma-separated sequences"
              value={
                getParamValue('model.stop') ? getParamValue('model.stop') : ''
              }
              onChange={e => updateParameter('model.stop', e.target.value)}
            />
            <InputHelper className="text-xs">
              Up to 4 sequences where the API will stop generating tokens.
            </InputHelper>
          </FieldSet>
        </Popover>
      </div>
    </div>
  );
};
