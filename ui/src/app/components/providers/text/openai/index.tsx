import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IButton } from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Popover } from '@/app/components/Popover';
import { OPENAI_TEXT_MODEL } from '@/app/components/providers/text/openai/constants';
import { cn } from '@/styles/media';
import { Bolt, Settings, X } from 'lucide-react';
import { useState } from 'react';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { InputHelper } from '@/app/components/input-helper';
import { InputCheckbox } from '@/app/components/Form/Checkbox';
import { Select } from '@/app/components/Form/Select';
import { Textarea } from '@/app/components/Form/Textarea';

export const ConfigureOpenaiTextProviderModel: React.FC<{
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

  const handleMetadataChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    updateParameter('model.metadata', newValue);
  };

  const handleResponseFormat = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    updateParameter('model.response_format', newValue);
  };

  const [open, setOpen] = useState(false);
  return (
    <div className="flex-1 flex ">
      <Dropdown
        className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!"
        currentValue={OPENAI_TEXT_MODEL.find(
          x => x.id === getParamValue('model.id'),
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
        allValue={OPENAI_TEXT_MODEL}
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
          className="z-50 min-w-fit p-4 grid grid-cols-3 gap-6"
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
                onChange={c => {
                  updateParameter('model.frequency_penalty', c.target.value);
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
            <FormLabel>Tool Choices</FormLabel>
            <Select
              onChange={e =>
                updateParameter('model.tool_choice', e.target.value)
              }
              placeholder="Tool Choices"
              className="text-sm! h-9 pl-3"
              value={getParamValue('model.tool_choice')}
              options={[
                { name: 'None', value: 'none' },
                { name: 'Auto', value: 'auto' },
                { name: 'Required', value: 'required' },
              ]}
            />
            <InputHelper className="text-xs">
              How the model should select which tool (or tools) to use when
              generating a response.
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
                onChange={c => {
                  updateParameter('model.temperature', c.target.value);
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
                onChange={c => {
                  updateParameter('model.top_p', c.target.value);
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
                onChange={c => {
                  updateParameter('model.presence_penalty', c.target.value);
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
            <FormLabel>Metadata</FormLabel>
            <Textarea
              placeholder="Enter as JSON"
              value={getParamValue('model.metadata') || '{}'}
              onChange={handleMetadataChange}
            />
            <InputHelper className="text-xs">
              Optional metadata about the request. Enter as JSON object.
            </InputHelper>
          </FieldSet>

          {getParamValue('model.name').startsWith('o') && (
            <FieldSet>
              <FormLabel>Reasoning Effort</FormLabel>
              <Select
                onChange={e => {
                  updateParameter('model.reasoning_effort', e.target.value);
                }}
                className="text-sm! h-9 pl-3"
                value={getParamValue('model.reasoning_effort')}
                placeholder="Select reasoning"
                options={[
                  { name: 'Low', value: 'low' },
                  { name: 'Medium', value: 'medium' },
                  { name: 'High', value: 'high' },
                ]}
              />
              <InputHelper className="text-xs">
                Constrains effort on reasoning for reasoning models.
              </InputHelper>
            </FieldSet>
          )}

          <FieldSet className="col-span-2">
            <FormLabel>Response Format</FormLabel>
            <Textarea
              placeholder="Enter as JSON"
              value={getParamValue('model.response_format') || '{}'}
              onChange={handleResponseFormat}
            />
            <InputHelper className="text-xs">
              Specifies the format for model output. Enter as JSON.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Seed</FormLabel>
            <Input
              type="number"
              value={getParamValue('model.seed')}
              placeholder="Seed"
              onChange={e => updateParameter('model.seed', e.target.value)}
            />
            <InputHelper className="text-xs">
              Seed for deterministic sampling.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Service Tier</FormLabel>

            <Select
              onChange={e => {
                updateParameter('model.service_tier', e.target.value);
              }}
              placeholder="Service Tier"
              className="text-sm! h-9 pl-3"
              value={getParamValue('model.service_tier')}
              options={[
                { name: 'Auto', value: 'auto' },
                { name: 'Default', value: 'default' },
                { name: 'Flex', value: 'flex' },
                { name: 'Priority', value: 'priority' },
              ]}
            />

            <InputHelper className="text-xs">
              Specifies the processing type for the request.
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

          <FieldSet>
            <FormLabel>Top Log Probabilities</FormLabel>
            <Slider
              min={0}
              max={20}
              value={getParamValue('model.top_logprobs')}
              onSlide={e =>
                updateParameter('model.top_logprobs', e.target.value)
              }
            />
            <InputHelper className="text-xs">
              Number of most likely tokens to return at each position (0-20).
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Log Probabilities</FormLabel>
            <InputCheckbox
              checked={getParamValue('model.frequency_penalty') === 'true'}
              onChange={e =>
                updateParameter('model.frequency_penalty', e ? 'true' : 'false')
              }
            />
            <InputHelper className="text-xs">
              Whether to return log probabilities of the output tokens.
            </InputHelper>
          </FieldSet>
        </Popover>
      </div>
    </div>
  );
};
