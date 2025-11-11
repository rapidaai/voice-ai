import { Metadata } from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { Tooltip } from '@/app/components/Tooltip';
import { cn } from '@/styles/media';
import { ExternalLink, Info, InfoIcon } from 'lucide-react';
import { CodeEditor } from '@/app/components/Form/editor/code-editor';
import { Textarea } from '@/app/components/Form/Textarea';
import { InputGroup } from '@/app/components/input-group';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';

export const ConfigurePutOnHold: React.FC<{
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
          <div className="grid grid-cols-2 w-full gap-4">
            <FieldSet className="flex justify-between">
              <FormLabel htmlFor="top_k">
                Max hold time second
                <Tooltip icon={<InfoIcon className="w-4 h-4 ml-1" />}>
                  <p className={cn('font-normal text-sm p-1 w-64')}>
                    Maximum hold duration before auto-resume or callback.
                  </p>
                </Tooltip>
              </FormLabel>
              <div className="flex justify-between items-center space-x-2">
                <Slider
                  min={3}
                  max={10}
                  step={1}
                  value={getParamValue('tool.max_hold_time')}
                  onSlide={(c: number) => {
                    updateParameter('tool.max_hold_time', c.toString());
                  }}
                />
                <Input
                  id="max_hold_time"
                  className={cn(
                    'py-0 px-1 tabular-nums border w-10 h-6 text-xs',
                    inputClass,
                  )}
                  min={0}
                  max={10}
                  type="number"
                  value={Number(getParamValue('tool.max_hold_time'))}
                  onChange={c => {
                    updateParameter('tool.max_hold_time', c.target.value);
                  }}
                />
              </div>
            </FieldSet>
          </div>
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
