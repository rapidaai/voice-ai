import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { FormLabel } from '@/app/components/form-label';
import { CodeEditor } from '@/app/components/Form/editor/code-editor';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Textarea } from '@/app/components/Form/Textarea';
import { InputGroup } from '@/app/components/input-group';
import { cn } from '@/styles/media';
import { Metadata } from '@rapidaai/react';
import { ExternalLink, Info } from 'lucide-react';

export const ConfigureEndOfConversation: React.FC<{
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
}> = ({ inputClass, toolDefinition, onChangeToolDefinition }) => {
  return (
    <InputGroup title="Tool Definition">
      <YellowNoticeBlock className="flex items-center">
        <Info className="shrink-0 w-4 h-4" />
        <div className="ms-3 text-sm font-medium">
          Know more about <b>End of Conversation</b> that can be supported by
          rapida
        </div>
        <a
          target="_blank"
          href="https://doc.rapida.ai/assistants/tools/add-end-of-conversation-tool"
          className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
          rel="noreferrer"
        >
          Read documentation
          <ExternalLink className="shrink-0 w-4 h-4 ml-1.5" strokeWidth={1.5} />
        </a>
      </YellowNoticeBlock>
      <div className={cn('p-6 flex flex-col gap-8 max-w-6xl')}>
        <FieldSet className="relative w-full">
          <FormLabel>Name</FormLabel>
          <Input
            value={toolDefinition.name || 'end_conversation'}
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
              'min-h-40 max-h-dvh bg-light-background dark:bg-gray-950 ',
              inputClass,
            )}
          />
        </FieldSet>
      </div>
    </InputGroup>
  );
};
