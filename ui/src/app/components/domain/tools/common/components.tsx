import { useState, useCallback, useEffect } from 'react';
import { CodeEditor } from '@/app/components/ui/editor/code-editor';
import { DocNoticeBlock } from '@/app/components/layout/container/message/notice-block/doc-notice-block';
import { Add, TrashCan, ArrowRight, Information } from '@carbon/icons-react';
import { TertiaryButton } from '@/app/components/ui/primitives';
import { Stack, TextInput, TextArea } from '@/app/components/ui/primitives';
import { Select, SelectItem, Button, Tooltip } from '@carbon/react';
import {
  ToolDefinition,
  KeyValueParameter,
  PARAMETER_TYPE_OPTIONS,
  ASSISTANT_KEY_OPTIONS,
  CONVERSATION_KEY_OPTIONS,
  TOOL_KEY_OPTIONS,
  CLIENT_KEY_OPTIONS,
} from './types';
import { parseJsonParameters, stringifyParameters } from './hooks';
import {
  TableForm,
  TableFormCell,
  TableFormHeaderCell,
  TableFormRow,
} from '@/app/components/ui/table';

// ============================================================================
// Documentation Notice Block
// ============================================================================

interface DocumentationNoticeProps {
  title?: string;
  documentationPath: string;
}

export function DocumentationNotice({
  title = 'Know more about supported knowledge tool definitions',
  documentationPath,
}: DocumentationNoticeProps) {
  return <DocNoticeBlock docPath={documentationPath}>{title}</DocNoticeBlock>;
}

// ============================================================================
// Tool Definition Form
// ============================================================================

interface ToolDefinitionFormProps {
  toolDefinition: ToolDefinition;
  onChangeToolDefinition: (value: ToolDefinition) => void;
  inputClass?: string;
  documentationPath?: string;
  documentationTitle?: string;
}

export function ToolDefinitionForm({
  toolDefinition,
  onChangeToolDefinition,
  inputClass,
  documentationPath = '/assistants/overview',
  documentationTitle,
}: ToolDefinitionFormProps) {
  const llmTooltip =
    'This value is sent to the LLM as part of the tool definition.';

  return (
    <div>
      <DocumentationNotice
        title={documentationTitle}
        documentationPath={documentationPath}
      />
      <div className="px-6 pb-6 mt-4 max-w-6xl">
        <Stack gap={6}>
          <TextInput
            id="tool-def-name"
            labelText={
              <span className="inline-flex items-center gap-1">
                Name
                <Tooltip align="right" label={llmTooltip}>
                  <Information size={14} />
                </Tooltip>
              </span>
            }
            value={toolDefinition.name}
            onChange={e =>
              onChangeToolDefinition({
                ...toolDefinition,
                name: e.target.value,
              })
            }
            placeholder="Enter tool name"
          />
          <TextArea
            id="tool-def-description"
            labelText={
              <span className="inline-flex items-center gap-1">
                Description
                <Tooltip align="right" label={llmTooltip}>
                  <Information size={14} />
                </Tooltip>
              </span>
            }
            value={toolDefinition.description}
            onChange={e =>
              onChangeToolDefinition({
                ...toolDefinition,
                description: e.target.value,
              })
            }
            placeholder="A tool description or definition of when this tool will get triggered."
            rows={2}
          />
          <CodeEditor
            labelText={
              <span className="inline-flex items-center gap-1">
                Parameters
                <Tooltip align="right" label={llmTooltip}>
                  <Information size={14} />
                </Tooltip>
              </span>
            }
            placeholder="Provide tool parameters as JSON that will be passed to LLM"
            value={toolDefinition.parameters}
            onChange={value =>
              onChangeToolDefinition({ ...toolDefinition, parameters: value })
            }
          />
        </Stack>
      </div>
    </div>
  );
}

// ============================================================================
// Type Key Selector
// ============================================================================

type MappingOption = { name: string; value: string };

const DEFAULT_KEY_OPTIONS_BY_TYPE: Record<string, MappingOption[]> = {
  assistant: [...ASSISTANT_KEY_OPTIONS],
  conversation: [...CONVERSATION_KEY_OPTIONS],
  tool: [...TOOL_KEY_OPTIONS],
  client: [...CLIENT_KEY_OPTIONS],
};

interface TypeKeySelectorProps {
  id?: string;
  type: string;
  value: string;
  onChange: (newValue: string) => void;
  keyOptionsByType?: Record<string, MappingOption[]>;
  includeEmptyKeyOption?: boolean;
  inputClass?: string;
}

export function TypeKeySelector({
  id,
  type,
  value,
  onChange,
  keyOptionsByType = DEFAULT_KEY_OPTIONS_BY_TYPE,
  includeEmptyKeyOption = false,
  inputClass,
}: TypeKeySelectorProps) {
  const options = keyOptionsByType[type] ?? null;

  if (options) {
    return (
      <Select
        id={id || `type-key-${type}`}
        labelText=""
        hideLabel
        value={value}
        onChange={e => onChange(e.target.value)}
        size="md"
        className={inputClass}
      >
        {includeEmptyKeyOption && <SelectItem value="" text="Select key" />}
        {options.map(opt => (
          <SelectItem key={opt.value} value={opt.value} text={opt.name} />
        ))}
      </Select>
    );
  }

  return (
    <TextInput
      id={id || 'type-key-custom'}
      labelText=""
      hideLabel
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder="Key"
      size="md"
      className={inputClass}
    />
  );
}

// ============================================================================
// Assistant Mapping Table
// ============================================================================

export interface AssistantMappingItem<TType extends string = string> {
  type: TType;
  key: string;
  value: string;
}

interface AssistantMappingTableProps<
  TType extends string = string,
  TItem extends AssistantMappingItem<TType> = AssistantMappingItem<TType>,
> {
  parameters: TItem[];
  onChange: (params: TItem[]) => void;
  typeOptions?: Array<{ name: string; value: string }>;
  defaultNewType?: TType;
  getDefaultParameterKey?: (type: TType) => string;
  resetValueOnTypeChange?: boolean;
  includeEmptyKeyOption?: boolean;
  keyOptionsByType?: Record<string, MappingOption[]>;
  createNewParameter?: () => TItem;
  title?: string;
  addButtonLabel?: string;
  valuePlaceholder?: string;
  isRemoveDestructive?: boolean;
  inputClass?: string;
}

export const AssistantMappingTable = <
  TType extends string = string,
  TItem extends AssistantMappingItem<TType> = AssistantMappingItem<TType>,
>({
  parameters,
  onChange,
  typeOptions = [...PARAMETER_TYPE_OPTIONS],
  defaultNewType = 'assistant' as TType,
  getDefaultParameterKey,
  resetValueOnTypeChange = false,
  includeEmptyKeyOption = false,
  keyOptionsByType = DEFAULT_KEY_OPTIONS_BY_TYPE,
  createNewParameter,
  title = 'Mapping',
  addButtonLabel = 'Add parameter',
  valuePlaceholder = 'Value',
  isRemoveDestructive = false,
}: AssistantMappingTableProps<TType, TItem>) => {
  const getDefaultKey = useCallback(
    (type: TType) => {
      if (getDefaultParameterKey) return getDefaultParameterKey(type);
      const options = keyOptionsByType[type];
      return options && options.length > 0 ? options[0].value : '';
    },
    [getDefaultParameterKey, keyOptionsByType],
  );

  const handleTypeChange = useCallback(
    (index: number, newType: TType) => {
      const next = [...parameters];
      const nextValue = resetValueOnTypeChange ? '' : next[index].value;
      next[index] = {
        ...next[index],
        type: newType,
        key: getDefaultKey(newType),
        value: nextValue,
      } as TItem;
      onChange(next);
    },
    [parameters, onChange, getDefaultKey, resetValueOnTypeChange],
  );

  const handleKeyChange = useCallback(
    (index: number, newKey: string) => {
      const next = [...parameters];
      next[index] = { ...next[index], key: newKey };
      onChange(next);
    },
    [parameters, onChange],
  );

  const handleValueChange = useCallback(
    (index: number, newValue: string) => {
      const next = [...parameters];
      next[index] = { ...next[index], value: newValue };
      onChange(next);
    },
    [parameters, onChange],
  );

  const handleRemove = useCallback(
    (index: number) => {
      onChange(parameters.filter((_, i) => i !== index));
    },
    [parameters, onChange],
  );

  const handleAdd = useCallback(() => {
    if (createNewParameter) {
      onChange([...parameters, createNewParameter()]);
      return;
    }
    onChange([
      ...parameters,
      {
        type: defaultNewType,
        key: getDefaultKey(defaultNewType),
        value: '',
      } as TItem,
    ]);
  }, [parameters, onChange, createNewParameter, defaultNewType, getDefaultKey]);

  return (
    <div>
      <p className="text-xs font-medium mb-2">
        {title} ({parameters.length})
      </p>
      <TableForm>
        <thead>
          <tr className="bg-[var(--cds-layer-accent-01)]">
            <TableFormHeaderCell className="w-1/4">Type</TableFormHeaderCell>
            <TableFormHeaderCell className="w-1/4">Key</TableFormHeaderCell>
            <TableFormHeaderCell className="w-8" />
            <TableFormHeaderCell className="w-1/4">Value</TableFormHeaderCell>
            <TableFormHeaderCell className="w-10" divider={false} />
          </tr>
        </thead>
        <tbody>
          {parameters.map(({ type, key, value: val }, index) => {
            return (
              <TableFormRow key={index}>
                <TableFormCell>
                  <Select
                    id={`param-type-${index}`}
                    labelText=""
                    hideLabel
                    value={type}
                    className="w-full"
                    onChange={e =>
                      handleTypeChange(index, e.target.value as TType)
                    }
                    size="md"
                  >
                    {typeOptions.map(opt => (
                      <SelectItem
                        key={opt.value}
                        value={opt.value}
                        text={opt.name}
                      />
                    ))}
                  </Select>
                </TableFormCell>
                <TableFormCell>
                  <TypeKeySelector
                    id={`param-key-${index}`}
                    type={type}
                    value={key}
                    onChange={newKey => handleKeyChange(index, newKey)}
                    keyOptionsByType={keyOptionsByType}
                    includeEmptyKeyOption={includeEmptyKeyOption}
                    inputClass="w-full"
                  />
                </TableFormCell>
                <TableFormCell className="w-8 text-center text-muted">
                  <ArrowRight size={16} className="mx-auto" />
                </TableFormCell>
                <TableFormCell>
                  <TextInput
                    id={`param-val-${index}`}
                    labelText=""
                    hideLabel
                    value={val}
                    onChange={e => handleValueChange(index, e.target.value)}
                    placeholder={valuePlaceholder}
                    size="md"
                  />
                </TableFormCell>
                <TableFormCell className="w-10 text-center" divider={false}>
                  <Button
                    hasIconOnly
                    renderIcon={TrashCan}
                    iconDescription="Remove"
                    kind="ghost"
                    size="sm"
                    className={
                      isRemoveDestructive
                        ? '!text-[var(--cds-support-error)]'
                        : undefined
                    }
                    onClick={() => handleRemove(index)}
                  />
                </TableFormCell>
              </TableFormRow>
            );
          })}
        </tbody>
      </TableForm>
      <div className="pt-2">
        <TertiaryButton
          size="md"
          renderIcon={Add}
          onClick={handleAdd}
          className="!w-full !max-w-none"
        >
          {addButtonLabel}
        </TertiaryButton>
      </div>
    </div>
  );
};

// ============================================================================
// Parameter Editor
// ============================================================================

interface ParameterEditorProps {
  value: string;
  onChange: (value: string) => void;
  typeOptions?: Array<{ name: string; value: string }>;
  defaultNewType?: string;
  getDefaultParameterKey?: (type: string) => string;
  resetValueOnTypeChange?: boolean;
  includeEmptyKeyOption?: boolean;
  keyOptionsByType?: Record<string, MappingOption[]>;
  title?: string;
  addButtonLabel?: string;
  valuePlaceholder?: string;
  isRemoveDestructive?: boolean;
  inputClass?: string;
}

export function ParameterEditor({
  value,
  onChange,
  typeOptions = [...PARAMETER_TYPE_OPTIONS],
  defaultNewType = 'assistant',
  getDefaultParameterKey,
  resetValueOnTypeChange = false,
  includeEmptyKeyOption = false,
  keyOptionsByType = DEFAULT_KEY_OPTIONS_BY_TYPE,
  title = 'Mapping',
  addButtonLabel = 'Add parameter',
  valuePlaceholder = 'Value',
  isRemoveDestructive = false,
}: ParameterEditorProps) {
  const [params, setParams] = useState<AssistantMappingItem[]>(() =>
    parseJsonParameters(value).map(({ key, value: parameterValue }) => {
      const [type, parameterKey = ''] = key.split('.');
      return { type, key: parameterKey, value: parameterValue };
    }),
  );

  useEffect(() => {
    setParams(
      parseJsonParameters(value).map(({ key, value: parameterValue }) => {
        const [type, parameterKey = ''] = key.split('.');
        return { type, key: parameterKey, value: parameterValue };
      }),
    );
  }, [value]);

  const handleChange = useCallback(
    (next: AssistantMappingItem[]) => {
      setParams(next);
      const serialized: KeyValueParameter[] = next.map(item => ({
        key: `${item.type}.${item.key}`,
        value: item.value,
      }));
      onChange(stringifyParameters(serialized));
    },
    [onChange],
  );

  return (
    <AssistantMappingTable
      parameters={params}
      onChange={handleChange}
      typeOptions={typeOptions}
      defaultNewType={defaultNewType}
      getDefaultParameterKey={getDefaultParameterKey}
      resetValueOnTypeChange={resetValueOnTypeChange}
      includeEmptyKeyOption={includeEmptyKeyOption}
      keyOptionsByType={keyOptionsByType}
      title={title}
      addButtonLabel={addButtonLabel}
      valuePlaceholder={valuePlaceholder}
      isRemoveDestructive={isRemoveDestructive}
    />
  );
}
