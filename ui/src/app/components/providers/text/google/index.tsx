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
import { GOOGLE_TEXT_MODEL } from '@/app/components/providers/text/google/constants';
import { Textarea } from '@/app/components/Form/Textarea';

export const ConfigureGoogleTextProviderModel: React.FC<{
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

  const handleResponseSchema = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    updateParameter('model.response_format', newValue);
  };
  return (
    <div className="flex-1 flex ">
      <Dropdown
        className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!"
        currentValue={GOOGLE_TEXT_MODEL.find(
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
        allValue={GOOGLE_TEXT_MODEL}
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
            <FormLabel>Temperature</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={0}
                max={2}
                step={0.1}
                value={getParamValue('model.temperature')}
                onSlide={(c: number) => {
                  updateParameter('model.temperature', c.toString());
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
                onSlide={(c: number) =>
                  updateParameter('model.top_p', c.toString())
                }
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
            <FormLabel>Top K</FormLabel>
            <Input
              type="number"
              placeholder="Top K"
              value={getParamValue('model.top_k')}
              onChange={e => updateParameter('model.top_k', e.target.value)}
            />
            <InputHelper className="text-xs">
              The number of highest probability vocabulary tokens to keep for
              top-k-filtering.
            </InputHelper>
          </FieldSet>

          <FieldSet>
            <FormLabel>Candidate Count</FormLabel>
            <Input
              type="number"
              placeholder="Number of responses"
              value={getParamValue('model.candidate_count')}
              onChange={e =>
                updateParameter('model.candidate_count', e.target.value)
              }
            />
            <InputHelper className="text-xs">
              Number of candidate responses to generate.
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
            <FormLabel>Max Output Tokens</FormLabel>
            <Input
              type="number"
              placeholder="Max Output Tokens"
              value={getParamValue('model.max_output_tokens')}
              onChange={e =>
                updateParameter('model.max_output_tokens', e.target.value)
              }
            />
            <InputHelper className="text-xs">
              Upper bound for tokens generated in the completion.
            </InputHelper>
          </FieldSet>

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
              Sequences where the API will stop generating tokens.
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
                onSlide={(c: number) => {
                  updateParameter('model.presence_penalty', c.toString());
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
            <FormLabel>Frequency Penalty</FormLabel>
            <div className="flex space-x-2 justify-center items-center">
              <Slider
                min={-2}
                max={2}
                step={0.1}
                value={getParamValue('model.frequency_penalty')}
                onSlide={(c: number) => {
                  updateParameter('model.frequency_penalty', c.toString());
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
          <FieldSet className="col-span-2">
            <FormLabel>Response schema</FormLabel>
            <Textarea
              placeholder="Enter as JSON"
              value={getParamValue('model.response_format') || ''}
              onChange={handleResponseSchema}
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
