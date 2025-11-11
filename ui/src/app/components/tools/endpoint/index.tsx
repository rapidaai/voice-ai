import { Metadata, Endpoint } from '@rapidaai/react';
import { EndpointDropdown } from '@/app/components/Dropdown/endpoint-dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IBlueBorderButton, ICancelButton } from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Select } from '@/app/components/Form/Select';
import { cn } from '@/styles/media';
import { ArrowRight, ExternalLink, Info, Plus, Trash2 } from 'lucide-react';
import { FC, useState } from 'react';
import { InputGroup } from '@/app/components/input-group';
import { CodeEditor } from '@/app/components/Form/editor/code-editor';
import { Textarea } from '@/app/components/Form/Textarea';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';

export const ConfigureEndpoint: React.FC<{
  toolDefinition: {
    name: string;
    description: string;
    parameters: string;
  };
  onChangeToolDefinition: (vl: {
    name: string;
    description: string;
    parameters: string;
  }) => void;
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
  inputClass?: string;
}> = ({
  toolDefinition,
  onChangeToolDefinition,
  onParameterChange,
  parameters,
  inputClass,
}) => {
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

  return (
    <>
      <InputGroup title="Action Definition">
        <div className={cn('p-6 pt-2 flex flex-col gap-8 max-w-6xl')}>
          <EndpointDropdown
            className={cn('bg-light-background', inputClass)}
            currentEndpoint={getParamValue('tool.endpoint_id')}
            onChangeEndpoint={(e: Endpoint) => {
              if (e) updateParameter('tool.endpoint_id', e.getId());
            }}
          />
          <EndpointArgument
            endpointParameters={getParamValue('tool.parameters')}
            setEndpointParameters={e => updateParameter('tool.parameters', e)}
          />
        </div>
      </InputGroup>
      <InputGroup title="Tool Definition">
        <YellowNoticeBlock className="flex items-center">
          <Info className="shrink-0 w-4 h-4" />
          <div className="ms-3 text-sm font-medium">
            Know more about knowledge tool definiation that can be supported by
            rapida
          </div>
          <a
            target="_blank"
            href="https://doc.rapida.ai/assistants/overview"
            className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
            rel="noreferrer"
          >
            Read documentation
            <ExternalLink
              className="shrink-0 w-4 h-4 ml-1.5"
              strokeWidth={1.5}
            />
          </a>
        </YellowNoticeBlock>
        <div className={cn('p-6 flex flex-col gap-8 max-w-6xl')}>
          <FieldSet className="relative w-full">
            <FormLabel>Name</FormLabel>
            <Input
              value={toolDefinition.name}
              onChange={e =>
                onChangeToolDefinition({
                  ...toolDefinition,
                  name: e.target.value,
                })
              }
              placeholder="Enter tool name"
              className={cn('bg-light-background', inputClass)}
            />
          </FieldSet>
          <FieldSet className="relative w-full">
            <FormLabel>Description</FormLabel>
            <Textarea
              value={toolDefinition.description}
              onChange={e =>
                onChangeToolDefinition({
                  ...toolDefinition,
                  description: e.target.value,
                })
              }
              className={cn('bg-light-background', inputClass)}
              placeholder="A tool description or definition of when this tool will get triggered."
              rows={2}
            />
          </FieldSet>

          <FieldSet className="relative w-full">
            <FormLabel>Parameters</FormLabel>
            <CodeEditor
              placeholder="Provide a tool parameters that will be passed to llm"
              value={toolDefinition.parameters}
              onChange={value => {
                onChangeToolDefinition({
                  ...toolDefinition,
                  parameters: value,
                });
              }}
              className={cn(
                'min-h-40 max-h-dvh bg-light-background dark:bg-gray-950',
                inputClass,
              )}
            />
          </FieldSet>
        </div>
      </InputGroup>
    </>
  );
};

const EndpointArgument: FC<{
  inputClass?: string;
  endpointParameters: string;
  setEndpointParameters: (s: string) => void;
}> = ({ endpointParameters, setEndpointParameters, inputClass }) => {
  const [requestParameters, setRequestParameters] = useState<
    Array<{ key: string; value: string }>
  >(() => {
    try {
      return Object.entries(JSON.parse(endpointParameters)).map(
        ([key, value]) => ({ key, value: value as string }),
      );
    } catch {
      return [];
    }
  });

  const updateParameters = (
    newParams: Array<{ key: string; value: string }>,
  ) => {
    setRequestParameters(newParams);
    setEndpointParameters(
      JSON.stringify(
        Object.fromEntries(newParams.map(({ key, value }) => [key, value])),
      ),
    );
  };

  return (
    <FieldSet>
      <FormLabel>Parameters ({requestParameters.length})</FormLabel>
      <div className="text-sm grid w-full">
        {requestParameters.map(({ key, value }, index) => {
          const [type, paramKey] = key.split('.');
          return (
            <div
              key={index}
              className="grid grid-cols-2 border-b border-gray-400 dark:border-gray-600"
            >
              <div className="flex col-span-1 items-center">
                <Select
                  value={type}
                  onChange={e => {
                    const newParams = [...requestParameters];
                    newParams[index] = {
                      key: `${e.target.value}.`,
                      value: '', // Reset value when type changes
                    };
                    updateParameters(newParams);
                  }}
                  className={cn('bg-light-background border-none', inputClass)}
                  options={[
                    { name: 'Tool', value: 'tool' },
                    { name: 'Assistant', value: 'assistant' },
                    { name: 'Conversation', value: 'conversation' },
                    { name: 'Argument', value: 'argument' },
                    { name: 'Metadata', value: 'metadata' },
                    { name: 'Option', value: 'option' },
                  ]}
                />
                <TypeKeySelector
                  type={
                    type as
                      | 'tool'
                      | 'assistant'
                      | 'conversation'
                      | 'argument'
                      | 'metadata'
                      | 'option'
                  }
                  inputClass={inputClass}
                  value={paramKey}
                  onChange={newKey => {
                    const newParams = [...requestParameters];
                    newParams[index] = { key: `${type}.${newKey}`, value };
                    updateParameters(newParams);
                  }}
                />
                <div className="bg-light-background dark:bg-gray-950 h-full flex items-center justify-center">
                  <ArrowRight strokeWidth={1.5} className="text-blue-600" />
                </div>
              </div>
              <div className="col-span-1 flex">
                <Input
                  value={value}
                  onChange={e => {
                    const newParams = [...requestParameters];
                    newParams[index] = { key, value: e.target.value };
                    updateParameters(newParams);
                  }}
                  placeholder="Value"
                  className="bg-light-background w-full border-none"
                />
                <ICancelButton
                  className="border-none outline-hidden bg-light-background"
                  onClick={() => {
                    const newParams = requestParameters.filter(
                      (_, i) => i !== index,
                    );
                    updateParameters(newParams);
                  }}
                  type="button"
                >
                  <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                </ICancelButton>
              </div>
            </div>
          );
        })}
      </div>
      <IBlueBorderButton
        onClick={() => {
          const newParams = [
            ...requestParameters,
            { key: 'assistant.', value: '' },
          ];
          updateParameters(newParams);
        }}
        className="justify-between space-x-8"
      >
        <span>Add parameters</span> <Plus className="h-4 w-4 ml-1.5" />
      </IBlueBorderButton>
    </FieldSet>
  );
};

const TypeKeySelector: FC<{
  inputClass?: string;
  type:
    | 'assistant'
    | 'conversation'
    | 'argument'
    | 'metadata'
    | 'option'
    | 'tool';
  value: string;
  onChange: (newValue: string) => void;
}> = ({ type, value, onChange, inputClass }) => {
  switch (type) {
    case 'assistant':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className={cn('bg-light-background border-none', inputClass)}
          options={[
            { name: 'Name', value: 'name' },
            { name: 'Prompt', value: 'prompt' },
          ]}
        />
      );
    case 'conversation':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className={cn('bg-light-background border-none', inputClass)}
          options={[{ name: 'Messages', value: 'messages' }]}
        />
      );
    case 'tool':
      return (
        <Select
          value={value}
          onChange={e => onChange(e.target.value)}
          className={cn('bg-light-background border-none', inputClass)}
          options={[
            { name: 'Argument', value: 'argument' },
            { name: 'Name', value: 'name' },
          ]}
        />
      );
    default:
      return (
        <Input
          value={value}
          onChange={e => onChange(e.target.value)}
          placeholder="Key"
          className="bg-light-background w-full border-none"
        />
      );
  }
};
