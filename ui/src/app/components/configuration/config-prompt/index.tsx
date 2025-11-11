import { FC, useCallback } from 'react';
import { PromptRole } from '@/models/debug';
import AdvancedMessageInput from '@/app/components/configuration/config-prompt/advanced-prompt-input';
import {
  MAX_PROMPT_MESSAGE_LENGTH,
  SUPPORTED_PROMPT_VARIABLE_TYPE,
} from '@/configs';
import { IBlueBorderButton } from '@/app/components/Form/Button';
import { Plus } from 'lucide-react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { ScalableTextarea } from '@/app/components/Form/Textarea';
import { getNewVar, getVars } from '@/utils/var';
import { TypeOfVariable } from '@/app/components/configuration/config-prompt/type-of-variable';
export type IPromptProps = {
  existingPrompt: {
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  };
  instanceId?: string;
  onChange: (prompt: {
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }) => void;
};

export const ConfigPrompt: FC<IPromptProps> = ({
  existingPrompt,
  onChange,
  instanceId,
}) => {
  const handlePromptChange = useCallback(
    (newPrompt: typeof existingPrompt.prompt) => {
      onChange({
        ...existingPrompt,
        prompt: newPrompt,
      });
    },
    [onChange, existingPrompt],
  );

  const handleVariablesChange = useCallback(
    (newVariables: typeof existingPrompt.variables) => {
      onChange({
        ...existingPrompt,
        variables: newVariables,
      });
    },
    [onChange, existingPrompt],
  );

  const handleMessageTypeChange = useCallback(
    (index: number, role: PromptRole) => {
      handlePromptChange(
        existingPrompt.prompt.map((item, i) =>
          i === index ? { ...item, role } : item,
        ),
      );
    },
    [handlePromptChange, existingPrompt.prompt],
  );
  const handleValueChange = useCallback(
    (value: string, index: number) => {
      const updatedPrompt = existingPrompt.prompt.map((item, i) =>
        i === index ? { ...item, content: value } : item,
      );
      const allVars = updatedPrompt.flatMap(item => getVars(item.content));
      const uniqueVars = [...new Set(allVars)];

      const updatedVariables = uniqueVars.map(varName => {
        const existingVar = existingPrompt.variables.find(
          v => v.name === varName,
        );
        return existingVar || getNewVar(varName);
      });

      onChange({
        prompt: updatedPrompt,
        variables: updatedVariables,
      });
    },
    [existingPrompt, onChange],
  );
  const handleAddMessage = useCallback(() => {
    const lastMessageType =
      existingPrompt.prompt[existingPrompt.prompt.length - 1]?.role;
    const newRole =
      lastMessageType === PromptRole.user
        ? PromptRole.assistant
        : PromptRole.user;
    handlePromptChange([
      ...existingPrompt.prompt,
      { role: newRole, content: '' },
    ]);
  }, [handlePromptChange, existingPrompt.prompt]);

  const handlePromptDelete = useCallback(
    (index: number) => {
      handlePromptChange(existingPrompt.prompt.filter((_, i) => i !== index));
    },
    [handlePromptChange, existingPrompt.prompt],
  );

  const handleVariableChange = useCallback(
    (name: string, type: string, defaultValue: string) => {
      handleVariablesChange(
        existingPrompt.variables.map(v =>
          v.name === name ? { ...v, type, defaultvalue: defaultValue } : v,
        ),
      );
    },
    [handleVariablesChange, existingPrompt.variables],
  );

  return (
    <>
      <FieldSet>
        <FormLabel>Instruction</FormLabel>
        <div className="space-y-2">
          {existingPrompt.prompt.map((item, index) => (
            <AdvancedMessageInput
              key={`${item.role}-${index}`}
              isChatMode
              instanceId={`${instanceId}-${item.role}-${index}`}
              type={item.role as PromptRole}
              value={item.content}
              onTypeChange={type => handleMessageTypeChange(index, type)}
              canDelete={existingPrompt.prompt.length > 1}
              onDelete={() => handlePromptDelete(index)}
              onChange={value => handleValueChange(value, index)}
            />
          ))}
          {existingPrompt.prompt.length < MAX_PROMPT_MESSAGE_LENGTH && (
            <IBlueBorderButton
              onClick={handleAddMessage}
              className="w-full justify-between"
            >
              Add new message <Plus className="h-4 w-4 ml-1.5" />
            </IBlueBorderButton>
          )}
        </div>
      </FieldSet>

      {existingPrompt.variables.length > 0 && (
        <FieldSet>
          <FormLabel>Arguments ({existingPrompt.variables.length})</FormLabel>
          <div className="text-sm grid bg-white dark:bg-gray-950 w-full divide-y">
            {existingPrompt.variables.map((v, idx) => (
              <div key={idx} className="grid grid-cols-3 divide-x">
                <div className="flex col-span-1 items-center px-4">
                  {v.name}
                </div>
                <TypeOfVariable
                  allType={SUPPORTED_PROMPT_VARIABLE_TYPE()}
                  className="col-span-1 h-full border-0"
                  type={v.type}
                  onChange={t =>
                    handleVariableChange(v.name, t, v.defaultvalue)
                  }
                />
                <div className="col-span-1">
                  <ScalableTextarea
                    wrapperClassName="border-0 p-0 min-h-4"
                    placeholder={`Default value for '${v.name}'`}
                    value={v.defaultvalue}
                    row={1}
                    onChange={e =>
                      handleVariableChange(v.name, v.type, e.target.value)
                    }
                  />
                </div>
              </div>
            ))}
          </div>
        </FieldSet>
      )}
    </>
  );
};
