import { FC, useCallback, useState } from 'react';
import { PromptRole } from '@/models/prompt';
import AdvancedMessageInput from '@/app/components/configuration/config-prompt/advanced-prompt-input';
import {
  MAX_PROMPT_MESSAGE_LENGTH,
  SUPPORTED_PROMPT_VARIABLE_TYPE,
} from '@/configs';
import { TertiaryButton } from '@/app/components/carbon/button';
import { ChevronDown, Plus } from 'lucide-react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/form/fieldset';
import { ScalableTextarea } from '@/app/components/form/textarea';
import { getNewVar, getVars } from '@/utils/var';
import { TypeOfVariable } from '@/app/components/configuration/config-prompt/type-of-variable';
import { InputHelper } from '@/app/components/input-helper';
import {
  RAPIDA_RESERVED_RUNTIME_VARIABLE_KEYS,
  RAPIDA_RESERVED_RUNTIME_VARIABLES,
} from '@/utils/prompt-reserved-variables';
import { Toggletip, ToggletipButton, ToggletipContent } from '@carbon/react';
import { Information } from '@carbon/icons-react';

const isRapidaReservedRuntimeVariable = (variableName: string): boolean =>
  RAPIDA_RESERVED_RUNTIME_VARIABLE_KEYS.has(variableName) ||
  variableName.startsWith('args.');
export type IPromptProps = {
  existingPrompt: {
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  };
  instanceId?: string;
  showRuntimeReplacementHint?: boolean;
  hideArgumentRuntimeHint?: boolean;
  enableReservedVariableSuggestions?: boolean;
  onChange: (prompt: {
    prompt: { role: string; content: string }[];
    variables: { name: string; type: string; defaultvalue: string }[];
  }) => void;
};

export const ConfigPrompt: FC<IPromptProps> = ({
  existingPrompt,
  onChange,
  instanceId,
  showRuntimeReplacementHint = false,
  hideArgumentRuntimeHint = false,
  enableReservedVariableSuggestions = false,
}) => {
  const [showReservedVariables, setShowReservedVariables] = useState(false);

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
        <div className="flex items-center gap-1">
          <FormLabel>Instruction</FormLabel>
          <Toggletip align="right">
            <ToggletipButton label="Show information">
              <Information size={14} />
            </ToggletipButton>
            <ToggletipContent>
              Define the messages and variables that guide the agent response.
            </ToggletipContent>
          </Toggletip>
        </div>
        {showRuntimeReplacementHint && (
          <div className="border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900">
            <button
              type="button"
              className="w-full text-left px-4 py-4"
              aria-expanded={showReservedVariables}
              onClick={() => setShowReservedVariables(v => !v)}
            >
              <div className="w-full flex items-center justify-between gap-2 text-left">
                <span className="text-[11px] font-semibold tracking-[0.08em] uppercase text-gray-500 dark:text-gray-400">
                  Reserved Variables
                </span>
                <ChevronDown
                  className={`h-4 w-4 text-gray-500 transition-transform ${showReservedVariables ? 'rotate-180' : ''}`}
                  strokeWidth={1.6}
                />
              </div>
              <InputHelper className="mt-1">
                These variables are preserved and replaced at runtime.
              </InputHelper>
            </button>

            {showReservedVariables && (
              <div className="mt-2 border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950">
                <div className="grid grid-cols-2 divide-x divide-gray-200 dark:divide-gray-800 bg-gray-100 dark:bg-gray-900">
                  <div className="px-3 py-2 text-[11px] font-semibold tracking-[0.08em] uppercase text-gray-500">
                    Variable
                  </div>
                  <div className="px-3 py-2 text-[11px] font-semibold tracking-[0.08em] uppercase text-gray-500">
                    Runtime value
                  </div>
                </div>
                <div className="divide-y divide-gray-200 dark:divide-gray-800">
                  {RAPIDA_RESERVED_RUNTIME_VARIABLES.map(item => (
                    <div
                      key={item.variable}
                      className="grid grid-cols-2 divide-x divide-gray-200 dark:divide-gray-800"
                    >
                      <div className="px-3 py-2">
                        <code className="text-xs text-gray-700 dark:text-gray-200">
                          {item.variable}
                        </code>
                      </div>
                      <div className="px-3 py-2 text-xs text-gray-600 dark:text-gray-300">
                        {item.runtimeValue}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
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
              enableReservedVariableSuggestions={
                enableReservedVariableSuggestions
              }
            />
          ))}
          {existingPrompt.prompt.length < MAX_PROMPT_MESSAGE_LENGTH && (
            <TertiaryButton
              size="md"
              renderIcon={Plus}
              onClick={handleAddMessage}
              className="!w-full !max-w-none !justify-between !text-left"
            >
              Add new message
            </TertiaryButton>
          )}
        </div>
      </FieldSet>

      {(showRuntimeReplacementHint || existingPrompt.variables.length > 0) && (
        <FieldSet>
          <div className="flex items-center gap-2">
            <FormLabel>Arguments</FormLabel>
            <span className="text-xs tabular-nums text-gray-400 dark:text-gray-600">
              {existingPrompt.variables.length}
            </span>
          </div>
          {showRuntimeReplacementHint && !hideArgumentRuntimeHint && (
            <InputHelper className="mb-2">
              Add only your template-specific variables here. Reserved variables
              are preserved and replaced at runtime.
            </InputHelper>
          )}
          <div className="text-sm grid bg-light-background dark:bg-gray-950 w-full border border-gray-300 dark:border-gray-700 divide-y divide-gray-300 dark:divide-gray-700">
            {/* Carbon table header row */}
            <div className="grid grid-cols-3 divide-x divide-gray-300 dark:divide-gray-700 bg-gray-50 dark:bg-gray-900">
              <div className="px-4 py-2 text-xs font-semibold tracking-[0.08em] uppercase text-gray-500 dark:text-gray-500">
                Variable
              </div>
              <div className="px-4 py-2 text-xs font-semibold tracking-[0.08em] uppercase text-gray-500 dark:text-gray-500">
                Type
              </div>
              <div className="px-4 py-2 text-xs font-semibold tracking-[0.08em] uppercase text-gray-500 dark:text-gray-500">
                Default value
              </div>
            </div>
            {existingPrompt.variables.map((v, idx) => (
              <div
                key={idx}
                className="grid grid-cols-3 divide-x divide-gray-300 dark:divide-gray-700"
              >
                <div className="flex col-span-1 items-center gap-2 px-4 py-2.5 text-sm text-gray-800 dark:text-gray-200 font-medium">
                  {v.name}
                  {showRuntimeReplacementHint &&
                    isRapidaReservedRuntimeVariable(v.name) && (
                      <span className="px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-blue-700 dark:text-blue-300 border border-blue-300/70 dark:border-blue-700/70">
                        Reserved
                      </span>
                    )}
                </div>
                <TypeOfVariable
                  allType={SUPPORTED_PROMPT_VARIABLE_TYPE()}
                  className="col-span-1 h-full border-0"
                  type={v.type}
                  onChange={t =>
                    handleVariableChange(v.name, t, v.defaultvalue)
                  }
                />
                <div className="col-span-1 h-full">
                  <ScalableTextarea
                    wrapperClassName="border-0 bg-transparent h-full"
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
            {existingPrompt.variables.length === 0 && (
              <div className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400">
                No template-specific variables yet. Add placeholders like{' '}
                <code>{'{{customer_name}}'}</code> in instruction messages to
                populate this list.
              </div>
            )}
          </div>
        </FieldSet>
      )}
    </>
  );
};
