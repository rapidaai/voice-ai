import { useCallback, useState } from 'react';
import { PromptRole } from '@/models/prompt';
import AdvancedMessageInput from '@/app/components/domain/configuration/config-prompt/advanced-prompt-input';
import {
  MAX_PROMPT_MESSAGE_LENGTH,
  SUPPORTED_PROMPT_VARIABLE_TYPE,
} from '@/configs';
import { TertiaryButton } from '@/app/components/ui/primitives';
import { FormLabel } from '@/app/components/ui/primitives';
import { FieldSet } from '@/app/components/ui/primitives';
import { getNewVar, getVars } from '@/utils/var';
import { TypeOfVariable } from '@/app/components/domain/configuration/config-prompt/type-of-variable';
import { InputHelper } from '@/app/components/ui/primitives';
import { Input } from '@/app/components/ui/primitives';
import {
  RAPIDA_RESERVED_RUNTIME_VARIABLE_KEYS,
  RAPIDA_RESERVED_RUNTIME_VARIABLES,
} from '@/utils/prompt-reserved-variables';
import {
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import {
  Add,
  Checkmark,
  ChevronDown,
  Close,
  Edit,
  Information,
} from '@carbon/icons-react';
import { tableFormContainerClassName } from '@/app/components/ui/table';

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

type PromptVariable = IPromptProps['existingPrompt']['variables'][number];

export function ConfigPrompt({
  existingPrompt,
  onChange,
  instanceId,
  showRuntimeReplacementHint = false,
  hideArgumentRuntimeHint = false,
  enableReservedVariableSuggestions = false,
}: IPromptProps) {
  const [showReservedVariables, setShowReservedVariables] = useState(false);
  const [editingVariableName, setEditingVariableName] = useState<string | null>(
    null,
  );
  const [draftVariable, setDraftVariable] = useState<PromptVariable | null>(
    null,
  );

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

  const editArgument = useCallback((variable: PromptVariable) => {
    setEditingVariableName(variable.name);
    setDraftVariable(variable);
  }, []);

  const cancelArgumentEdit = useCallback(() => {
    setEditingVariableName(null);
    setDraftVariable(null);
  }, []);

  const handleDraftVariableChange = useCallback(
    (type: string, defaultValue: string) => {
      setDraftVariable(current =>
        current ? { ...current, type, defaultvalue: defaultValue } : current,
      );
    },
    [],
  );

  const saveArgument = useCallback(() => {
    if (!draftVariable) return;

    handleVariablesChange(
      existingPrompt.variables.map(variable =>
        variable.name === draftVariable.name ? draftVariable : variable,
      ),
    );
    setEditingVariableName(null);
    setDraftVariable(null);
  }, [draftVariable, existingPrompt.variables, handleVariablesChange]);

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
            <Button
              type="button"
              kind="ghost"
              size="md"
              className="!h-auto !min-h-0 !w-full !max-w-none !justify-between !px-4 !py-4 !text-left"
              aria-expanded={showReservedVariables}
              onClick={() => setShowReservedVariables(v => !v)}
            >
              <span className="flex w-full min-w-0 flex-col items-stretch gap-1 text-left">
                <span className="flex w-full min-w-0 items-center justify-between gap-2">
                  <span className="text-[11px] font-semibold tracking-[0.08em] uppercase text-[var(--cds-text-secondary)]">
                    Reserved Variables
                  </span>
                  <ChevronDown
                    className={`h-4 w-4 shrink-0 text-gray-500 transition-transform ${showReservedVariables ? 'rotate-180' : ''}`}
                  />
                </span>
                <InputHelper>
                  These variables are preserved and replaced at runtime.
                </InputHelper>
              </span>
            </Button>

            {showReservedVariables && (
              <div className="mt-2 border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950">
                <TableContainer>
                  <Table size="sm" useZebraStyles={false}>
                    <TableHead>
                      <TableRow>
                        <TableHeader>Variable</TableHeader>
                        <TableHeader>Runtime value</TableHeader>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {RAPIDA_RESERVED_RUNTIME_VARIABLES.map(item => (
                        <TableRow key={item.variable}>
                          <TableCell>
                            <code className="text-xs text-gray-700 dark:text-gray-200">
                              {item.variable}
                            </code>
                          </TableCell>
                          <TableCell>{item.runtimeValue}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
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
              renderIcon={Add}
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
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <div className="flex items-center gap-1">
                <FormLabel>Arguments</FormLabel>
                {showRuntimeReplacementHint && !hideArgumentRuntimeHint && (
                  <Toggletip align="right">
                    <ToggletipButton label="Show argument information">
                      <Information size={14} />
                    </ToggletipButton>
                    <ToggletipContent>
                      Add only your template-specific variables here. Reserved
                      variables are preserved and replaced at runtime.
                    </ToggletipContent>
                  </Toggletip>
                )}
                <span className="text-xs tabular-nums text-gray-400 dark:text-gray-600">
                  {existingPrompt.variables.length}
                </span>
              </div>
            </div>
          </div>
          <TableContainer className={tableFormContainerClassName}>
            <Table size="sm" useZebraStyles={false}>
              <TableHead>
                <TableRow>
                  <TableHeader>Variable</TableHeader>
                  <TableHeader>Type</TableHeader>
                  <TableHeader>Default value</TableHeader>
                  <TableHeader className="w-20 text-center">Action</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {existingPrompt.variables.map(v => {
                  const isReservedVariable =
                    showRuntimeReplacementHint &&
                    isRapidaReservedRuntimeVariable(v.name);

                  const isEditing = editingVariableName === v.name;
                  const variable =
                    isEditing && draftVariable ? draftVariable : v;

                  return (
                    <TableRow key={v.name}>
                      <TableCell className={isEditing ? '!p-0' : undefined}>
                        {isEditing ? (
                          <Input
                            id={`argument-name-${v.name}`}
                            aria-label={`Variable ${v.name}`}
                            labelText={`Variable ${v.name}`}
                            value={variable.name}
                            disabled
                            className="!m-0 w-full"
                          />
                        ) : (
                          <span className="flex min-w-0 items-center gap-2">
                            <code className="truncate font-mono text-xs font-semibold text-gray-800 dark:text-gray-200">
                              {v.name}
                            </code>
                            {isReservedVariable && (
                              <span className="shrink-0 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-blue-700 dark:text-blue-300 border border-blue-300/70 dark:border-blue-700/70">
                                Reserved
                              </span>
                            )}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className={isEditing ? '!p-0' : 'capitalize'}>
                        {isEditing ? (
                          <TypeOfVariable
                            id={`argument-type-${v.name}`}
                            aria-label={`Type for ${v.name}`}
                            labelText={`Type for ${v.name}`}
                            allType={SUPPORTED_PROMPT_VARIABLE_TYPE()}
                            className="!m-0 w-full [&_.cds--select-input]:capitalize"
                            type={variable.type}
                            onChange={type =>
                              handleDraftVariableChange(
                                type,
                                variable.defaultvalue,
                              )
                            }
                          />
                        ) : (
                          v.type
                        )}
                      </TableCell>
                      <TableCell className={isEditing ? '!p-0' : undefined}>
                        {isEditing ? (
                          <Input
                            id={`argument-default-${v.name}`}
                            aria-label={`Default value for ${v.name}`}
                            labelText={`Default value for ${v.name}`}
                            placeholder="Optional default value"
                            value={variable.defaultvalue}
                            className="!m-0 w-full"
                            onChange={event =>
                              handleDraftVariableChange(
                                variable.type,
                                event.target.value,
                              )
                            }
                          />
                        ) : (
                          <span className="text-gray-600 dark:text-gray-400">
                            {v.defaultvalue || 'No default value'}
                          </span>
                        )}
                      </TableCell>
                      <TableCell
                        className={isEditing ? '!p-0' : '!text-center'}
                      >
                        {isEditing ? (
                          <span className="flex h-full items-stretch justify-center">
                            <Button
                              type="button"
                              kind="ghost"
                              size="sm"
                              hasIconOnly
                              renderIcon={Checkmark}
                              iconDescription={`Save ${v.name}`}
                              tooltipPosition="left"
                              onClick={saveArgument}
                            />
                            <Button
                              type="button"
                              kind="ghost"
                              size="sm"
                              hasIconOnly
                              renderIcon={Close}
                              iconDescription={`Cancel ${v.name}`}
                              tooltipPosition="left"
                              onClick={cancelArgumentEdit}
                            />
                          </span>
                        ) : (
                          <Button
                            type="button"
                            kind="ghost"
                            size="sm"
                            hasIconOnly
                            renderIcon={Edit}
                            iconDescription={`Edit ${v.name}`}
                            tooltipPosition="left"
                            onClick={() => editArgument(v)}
                          />
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
                {existingPrompt.variables.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4}>
                      <span className="text-sm text-gray-600 dark:text-gray-400">
                        No template-specific variables yet. Add placeholders
                        like <code>{'{{customer_name}}'}</code> in instruction
                        messages to populate this list.
                      </span>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </FieldSet>
      )}
    </>
  );
}
