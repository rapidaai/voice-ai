import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IButton } from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Popover } from '@/app/components/Popover';
import { ANTHROPIC_TEXT_MODEL } from '@/app/components/providers/text/anthropic/constants';
import { cn } from '@/styles/media';
import { Bolt, Settings, X } from 'lucide-react';
import { useState } from 'react';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { InputHelper } from '@/app/components/input-helper';
import { Select } from '@/app/components/Form/Select';
import { Textarea } from '@/app/components/Form/Textarea';

export const ConfigureAnthropicTextProviderModel: React.FC<{
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

  const updateMultipleParameters = (
    updates: { key: string; value: string }[],
  ) => {
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
  const [open, setOpen] = useState(false);

  const handleMetadataChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    updateParameter('model.metadata', newValue);
  };

  const handleThinkingChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    updateParameter('model.thinking', newValue);
  };
  return (
    <div className="flex-1 flex ">
      <Dropdown
        className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!"
        currentValue={ANTHROPIC_TEXT_MODEL.find(
          x =>
            x.id === getParamValue('model.id') &&
            getParamValue('model.name') === x.name,
        )}
        setValue={v => {
          updateMultipleParameters([
            { key: 'model.id', value: v.id },
            { key: 'model.name', value: v.name },
          ]);
        }}
        allValue={ANTHROPIC_TEXT_MODEL}
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
            <Bolt className={cn('w-4 h-4')} strokeWidth={1.5} />
          )}
        </IButton>
        <Popover
          align={'bottom-end'}
          open={open}
          setOpen={setOpen}
          className="z-50 min-w-fit p-4 grid grid-cols-3 gap-6"
        >
          {/* Core Parameters */}
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
              Max number of tokens to generate. Must be ≥ 1.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Temperature</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0}
                max={1}
                step={0.1}
                value={getParamValue('model.temperature')}
                onSlide={c => updateParameter('model.temperature', c)}
              />
              <Input
                type="number"
                min={0}
                max={1}
                step={0.1}
                value={getParamValue('model.temperature')}
                onChange={c =>
                  updateParameter('model.temperature', c.target.value)
                }
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Controls randomness. 0.0 = more deterministic, 1.0 = more
              creative.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Top K</FormLabel>
            <Input
              type="number"
              min={0}
              value={getParamValue('model.top_k')}
              onChange={e => updateParameter('model.top_k', e.target.value)}
            />
            <InputHelper className="text-xs">
              Samples from top-k tokens only. Reduces low-probability responses.
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
                onSlide={e => updateParameter('model.top_p', e)}
              />
              <Input
                type="number"
                min={0}
                max={1}
                step={0.05}
                value={getParamValue('model.top_p')}
                onChange={c => updateParameter('model.top_p', c.target.value)}
                className="w-16"
              />
            </div>
            <InputHelper className="text-xs">
              Nucleus sampling. Cuts off cumulative token probabilities above
              top_p.
            </InputHelper>
          </FieldSet>

          {/* Control & Stop Parameters */}
          <FieldSet>
            <FormLabel>Stop Sequences</FormLabel>
            <Input
              type="text"
              placeholder="Comma-separated sequences"
              value={getParamValue('model.stop_sequences') || ''}
              onChange={e =>
                updateParameter('model.stop_sequences', e.target.value)
              }
            />
            <InputHelper className="text-xs">
              Custom strings that stop generation when encountered.
            </InputHelper>
          </FieldSet>

          {/* Metadata & Configuration */}
          <FieldSet className="col-span-2">
            <FormLabel>Metadata</FormLabel>
            <Textarea
              placeholder="Enter as JSON"
              value={getParamValue('model.metadata') || ''}
              onChange={handleMetadataChange}
            />
            <InputHelper className="text-xs">
              Optional metadata about the request. Enter as JSON object.
            </InputHelper>
          </FieldSet>
          <FieldSet>
            <FormLabel>Container</FormLabel>
            <Input
              type="text"
              value={getParamValue('model.container') || ''}
              onChange={e => updateParameter('model.container', e.target.value)}
            />
            <InputHelper className="text-xs">
              Identifier to group requests (used for caching, reuse, etc.).
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Service Tier</FormLabel>
            <Select
              onChange={e =>
                updateParameter('model.service_tier', e.target.value)
              }
              placeholder="Service Tier"
              className="text-sm! h-9 pl-3"
              value={getParamValue('model.service_tier')}
              options={[
                { name: 'Auto', value: 'auto' },
                { name: 'Standard Only', value: 'standard_only' },
              ]}
            />
            <InputHelper className="text-xs">
              Controls if request uses priority capacity.
            </InputHelper>
          </FieldSet>

          {/* Extended Thinking */}

          <FieldSet className="col-span-2">
            <FormLabel>Thinking</FormLabel>
            <Textarea
              placeholder="Enter as JSON"
              value={getParamValue('model.thinking') || ''}
              onChange={handleThinkingChange}
            />
            <InputHelper className="text-xs">
              Enables Claude to show internal reasoning ("thinking").
            </InputHelper>
          </FieldSet>
        </Popover>
      </div>
    </div>
  );
};
