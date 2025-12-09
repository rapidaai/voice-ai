import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IButton } from '@/app/components/form/button';
import { FieldSet } from '@/app/components/form/fieldset';
import { Popover } from '@/app/components/popover';
import { cn } from '@/utils';
import { Bolt, X } from 'lucide-react';
import { useCallback, useState } from 'react';
import { Input } from '@/app/components/form/input';
import { Slider } from '@/app/components/form/slider';
import { InputHelper } from '@/app/components/input-helper';
import { Textarea } from '@/app/components/form/textarea';
import { Select } from '@/app/components/form/select';
import { AZURE_TEXT_MODEL } from '@/providers';
export {
  GetAzureTextProviderDefaultOptions,
  ValidateAzureTextProviderDefaultOptions,
} from './constants';
export const ConfigureAzureTextProviderModel: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const getParamValue = useCallback(
    (key: string) => {
      return parameters?.find(p => p.getKey() === key)?.getValue() ?? '';
    },
    [parameters],
  );

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
  const handleMetadataChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    updateParameter('model.metadata', newValue);
  };

  const handleResponseFormat = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    updateParameter('model.response_format', newValue);
  };

  return (
    <div className="flex-1 flex items-center divide-x">
      <Dropdown
        className="max-w-full  focus-within:border-none! focus-within:outline-hidden! border-none!"
        currentValue={AZURE_TEXT_MODEL().find(
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
        allValue={AZURE_TEXT_MODEL()}
        placeholder="Select model"
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
                onSlide={(e: number) =>
                  updateParameter('model.top_p', e.toString())
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
              value={getParamValue('model.response_format') || '{}'}
              onChange={handleResponseFormat}
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

          <FieldSet>
            <FormLabel>User</FormLabel>
            <Input
              type="text"
              placeholder="user"
              value={
                getParamValue('model.user') ? getParamValue('model.user') : ''
              }
              onChange={e => updateParameter('model.user', e.target.value)}
            />
            <InputHelper className="text-xs">
              A stable identifier for your end-users.
            </InputHelper>
          </FieldSet>
        </Popover>
      </div>
    </div>
  );
};
