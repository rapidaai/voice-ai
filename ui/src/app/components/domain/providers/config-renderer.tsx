import type { FocusEvent, ReactNode } from 'react';
import { useMemo, useRef, useState } from 'react';
import { Metadata } from '@rapidaai/react';
import { SettingsAdjust, Add, TrashCan } from '@carbon/icons-react';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import { cn } from '@/utils';
import { TextInput, TextArea } from '@/app/components/ui/primitives';
import { TertiaryButton } from '@/app/components/ui/primitives';
import {
  Select as CarbonSelect,
  SelectItem,
  Slider,
  Button,
  Dropdown as CarbonDropdown,
  ComboBox,
  ComposedModal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@carbon/react';
import {
  CategoryConfig,
  ParameterConfig,
  ProviderConfig,
  isModelSelectorParameter,
  loadProviderData,
  resolveCategoryParameters,
} from '@/providers/config-loader';
import { getDefaultsFromConfig } from '@/providers/config-defaults';
import { JsonEditor } from '@/app/components/ui/editor/json-editor';
import { WebsocketDslEditor } from '@/app/components/domain/providers/websocket-dsl-editor';
import { HelpToggletip } from '@/app/components/domain/providers/help-label';
import { FormLabel } from '@/app/components/ui/primitives';
import {
  TableForm,
  TableFormCell,
  TableFormHeaderCell,
  TableFormRow,
} from '@/app/components/ui/table';

type ProviderDataItem = Record<string, unknown> & {
  name?: string;
};

const getProviderDataItemName = (item: ProviderDataItem | null): string =>
  item?.name ?? '';

const getProviderDataItemValue = (
  item: ProviderDataItem,
  field: string,
): string => {
  const value = item[field];
  return value === undefined || value === null ? '' : String(value);
};

interface FieldLabelRowProps {
  param: ParameterConfig;
  htmlFor?: string;
  className?: string;
}

function FieldLabelRow({ param, htmlFor, className }: FieldLabelRowProps) {
  return (
    <span className={cn('inline-flex items-center gap-1', className)}>
      <FormLabel htmlFor={htmlFor}>{param.label}</FormLabel>
      <HelpToggletip label={param.label} helpText={param.helpText} />
    </span>
  );
}

interface ConfigRendererProps {
  provider: string;
  category:
    | 'stt'
    | 'tts'
    | 'text'
    | 'telephony'
    | 'vad'
    | 'eos'
    | 'noise'
    | 'telemetry'
    | 'storage';
  config: CategoryConfig;
  parameters: Metadata[] | null;
  onParameterChange: (parameters: Metadata[]) => void;
}

export function ConfigRenderer({
  provider,
  category,
  config,
  parameters,
  onParameterChange,
}: ConfigRendererProps) {
  const effectiveParameters = useMemo(
    () =>
      resolveCategoryParameters(provider, category, config, parameters ?? []),
    [provider, category, config, parameters],
  );

  const getParamValue = (key: string) =>
    (parameters ?? []).find(p => p.getKey() === key)?.getValue() ?? '';

  const isModelSelector = (param: ParameterConfig): boolean =>
    isModelSelectorParameter(param) &&
    (category === 'stt' || category === 'tts' || category === 'text');

  const applyUpdates = (
    updates: { key: string; value: string }[],
    sourceParam?: ParameterConfig,
  ) => {
    const updatedParams = [...(parameters ?? [])];
    const currentModelValue =
      sourceParam && isModelSelector(sourceParam)
        ? getParamValue(sourceParam.key)
        : '';

    for (const { key, value } of updates) {
      const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
      const newParam = new Metadata();
      newParam.setKey(key);
      newParam.setValue(value);
      if (existingIndex >= 0) {
        updatedParams[existingIndex] = newParam;
      } else {
        updatedParams.push(newParam);
      }
    }

    if (!sourceParam || !isModelSelector(sourceParam)) {
      onParameterChange(updatedParams);
      return;
    }
    const nextModelValue = updates.find(
      update => update.key === sourceParam.key,
    )?.value;
    if (nextModelValue === undefined || nextModelValue === currentModelValue) {
      onParameterChange(updatedParams);
      return;
    }
    if (sourceParam.customValue && nextModelValue !== '') {
      const data = sourceParam.data
        ? loadProviderData(provider, sourceParam.data)
        : [];
      const valueField = sourceParam.valueField || 'id';
      const hasCatalogMatch = data.some((item: ProviderDataItem) => {
        const catalogValue = item[valueField];
        if (
          catalogValue !== undefined &&
          String(catalogValue) === nextModelValue
        ) {
          return true;
        }
        return (
          typeof item.name === 'string' &&
          item.name.toLowerCase() === nextModelValue.toLowerCase()
        );
      });
      if (!hasCatalogMatch) {
        onParameterChange(updatedParams);
        return;
      }
    }

    const includeCredential = updatedParams.some(
      p => p.getKey() === 'rapida.credential_id',
    );
    const wrappedConfig = { [category]: config } as ProviderConfig;
    const hydrated = getDefaultsFromConfig(
      wrappedConfig,
      category,
      updatedParams,
      provider,
      { includeCredential },
    );
    onParameterChange(hydrated);
  };

  const updateParameter = (
    key: string,
    value: string,
    sourceParam?: ParameterConfig,
  ) => {
    applyUpdates([{ key, value }], sourceParam);
  };

  const updateMultipleParameters = (
    updates: { key: string; value: string }[],
    sourceParam?: ParameterConfig,
  ) => {
    applyUpdates(updates, sourceParam);
  };

  const isVisible = (param: ParameterConfig): boolean => {
    if (!param.showWhen) return true;
    const refValue = getParamValue(param.showWhen.key);
    return new RegExp(param.showWhen.pattern).test(refValue);
  };

  const visibleParameters = effectiveParameters.filter(isVisible);
  const regularParams = visibleParameters.filter(p => !p.advanced);
  const advancedParams = visibleParameters.filter(p => p.advanced);
  const hasAdvanced = advancedParams.length > 0;
  const textMainParam =
    category === 'text' ? regularParams.find(isModelSelectorParameter) : null;
  const shouldUseTextLayout =
    category === 'text' &&
    textMainParam &&
    (hasAdvanced || regularParams.length === 1);

  const renderField = (param: ParameterConfig) => {
    if (!isVisible(param)) return null;

    const isSingleAdvancedTextField =
      category === 'text' && param.advanced && advancedParams.length === 1;
    const colSpanClass = cn(
      'min-w-0',
      isSingleAdvancedTextField
        ? 'col-span-full'
        : param.colSpan === 2
          ? 'col-span-2'
          : 'col-span-1',
    );

    switch (param.type) {
      case 'dropdown':
        return (
          <DropdownField
            key={param.key}
            param={param}
            provider={provider}
            value={getParamValue(param.key)}
            onChange={(value, selectedItem) => {
              if (param.linkedField && selectedItem) {
                updateMultipleParameters(
                  [
                    { key: param.key, value },
                    {
                      key: param.linkedField.key,
                      value: getProviderDataItemValue(
                        selectedItem,
                        param.linkedField.sourceField,
                      ),
                    },
                  ],
                  param,
                );
              } else {
                updateParameter(param.key, value, param);
              }
            }}
            colSpanClass={colSpanClass}
          />
        );

      case 'slider':
        const sliderRawValue = getParamValue(param.key);
        const sliderSourceValue =
          sliderRawValue !== ''
            ? sliderRawValue
            : param.default !== undefined
              ? String(param.default)
              : '';
        const sliderParsedValue = Number.parseFloat(sliderSourceValue);
        const sliderValue = Number.isNaN(sliderParsedValue)
          ? param.min ?? 0
          : sliderParsedValue;
        return (
          <div className={cn(colSpanClass)} key={param.key}>
            <FieldLabelRow param={param} htmlFor={`slider-${param.key}`} />
            <Slider
              id={`slider-${param.key}`}
              labelText=""
              min={param.min ?? 0}
              max={param.max ?? 1}
              step={param.step ?? 0.1}
              value={sliderValue}
              onChange={({ value: v }: { value: number }) =>
                updateParameter(param.key, v.toString())
              }
            />
          </div>
        );

      case 'number':
        const numberInputId = `num-${param.key}`;
        return (
          <div className={cn(colSpanClass)} key={param.key}>
            <FieldLabelRow param={param} htmlFor={numberInputId} />
            <TextInput
              id={numberInputId}
              labelText=""
              type="number"
              min={param.min}
              max={param.max}
              step={param.step}
              value={getParamValue(param.key)}
              placeholder={param.placeholder}
              onChange={e => updateParameter(param.key, e.target.value)}
            />
          </div>
        );

      case 'input':
        const inputId = `input-${param.key}`;
        return (
          <div className={cn(colSpanClass)} key={param.key}>
            <FieldLabelRow param={param} htmlFor={inputId} />
            <TextInput
              id={inputId}
              labelText=""
              value={getParamValue(param.key)}
              placeholder={param.placeholder}
              onChange={e => updateParameter(param.key, e.target.value)}
            />
          </div>
        );

      case 'textarea':
        const textareaId = `textarea-${param.key}`;
        return (
          <div className={cn(colSpanClass)} key={param.key}>
            <FieldLabelRow param={param} htmlFor={textareaId} />
            <TextArea
              id={textareaId}
              labelText=""
              required={param.required !== false}
              value={getParamValue(param.key)}
              rows={param.rows ?? 2}
              placeholder={param.placeholder}
              onChange={e => updateParameter(param.key, e.target.value)}
            />
          </div>
        );

      case 'select':
        const selectId = `select-${param.key}`;
        return (
          <div className={cn(colSpanClass)} key={param.key}>
            <FieldLabelRow param={param} htmlFor={selectId} />
            <CarbonSelect
              id={selectId}
              labelText=""
              value={getParamValue(param.key)}
              onChange={e => updateParameter(param.key, e.target.value)}
            >
              <SelectItem
                value=""
                text={`Select ${param.label.toLowerCase()}`}
              />
              {(param.choices ?? []).map(c => (
                <SelectItem key={c.value} value={c.value} text={c.label} />
              ))}
            </CarbonSelect>
          </div>
        );

      case 'json':
        const jsonId = `json-${param.key}`;
        return (
          <div className={cn(colSpanClass)} key={param.key}>
            <FieldLabelRow param={param} htmlFor={jsonId} />
            <div className="mt-1 w-full min-w-0">
              {param.editor === 'websocket_dsl_json' ? (
                <WebsocketDslEditor
                  provider={
                    provider === 'custom-stt' ? 'custom-stt' : 'custom-tts'
                  }
                  mode={param.editorMode ?? 'query_params'}
                  value={getParamValue(param.key)}
                  placeholder={param.placeholder ?? 'Enter as JSON'}
                  onChange={value => updateParameter(param.key, value)}
                  height="160px"
                  className="w-full"
                />
              ) : (
                <JsonEditor
                  value={getParamValue(param.key)}
                  placeholder={param.placeholder ?? 'Enter as JSON'}
                  onChange={value => updateParameter(param.key, value)}
                  height="160px"
                  className="w-full"
                />
              )}
            </div>
          </div>
        );

      case 'key_value':
        return (
          <KeyValueField
            key={param.key}
            param={param}
            value={getParamValue(param.key)}
            onChange={value => updateParameter(param.key, value)}
            colSpanClass={colSpanClass}
          />
        );

      default:
        return null;
    }
  };

  if (shouldUseTextLayout) {
    return (
      <TextCategoryLayout
        mainParam={textMainParam}
        provider={provider}
        advancedParams={advancedParams}
        getParamValue={getParamValue}
        updateMultipleParameters={updateMultipleParameters}
        updateParameter={updateParameter}
        renderField={renderField}
        parameters={parameters}
        onParameterChange={onParameterChange}
      />
    );
  }

  return <>{effectiveParameters.map(renderField)}</>;
}

interface TextCategoryLayoutProps {
  mainParam?: ParameterConfig;
  provider: string;
  advancedParams: ParameterConfig[];
  getParamValue: (key: string) => string;
  updateMultipleParameters: (
    updates: { key: string; value: string }[],
    sourceParam?: ParameterConfig,
  ) => void;
  updateParameter: (
    key: string,
    value: string,
    sourceParam?: ParameterConfig,
  ) => void;
  renderField: (param: ParameterConfig) => ReactNode;
  parameters: Metadata[] | null;
  onParameterChange: (parameters: Metadata[]) => void;
}

function TextCategoryLayout({
  mainParam,
  provider,
  advancedParams,
  getParamValue,
  updateMultipleParameters,
  updateParameter,
  renderField,
  parameters,
  onParameterChange,
}: TextCategoryLayoutProps) {
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const snapshotRef = useRef<Metadata[] | null>(null);

  const handleOpen = () => {
    snapshotRef.current = parameters ? [...parameters] : null;
    setAdvancedOpen(true);
  };

  const handleClose = () => {
    if (snapshotRef.current) {
      onParameterChange(snapshotRef.current);
    }
    snapshotRef.current = null;
    setAdvancedOpen(false);
  };

  const handleComplete = () => {
    snapshotRef.current = null;
    setAdvancedOpen(false);
  };

  return (
    <div className="flex-1 flex items-stretch min-w-0">
      <div className="flex-1 min-w-0">
        {mainParam?.type === 'dropdown' &&
          renderTextMainDropdown(
            mainParam,
            provider,
            getParamValue,
            updateMultipleParameters,
            updateParameter,
          )}
      </div>
      {advancedParams.length > 0 && (
        <div className="shrink-0 flex border-l border-gray-200 dark:border-gray-700">
          <Button
            hasIconOnly
            renderIcon={SettingsAdjust}
            iconDescription="Advanced settings"
            kind="ghost"
            size="md"
            className="!h-10 !min-h-10 !w-10 !p-0 !bg-[var(--cds-field)]"
            onClick={handleOpen}
          />
          <ComposedModal
            open={advancedOpen}
            onClose={handleClose}
            preventCloseOnClickOutside
            size="lg"
          >
            <ModalHeader title="Advanced Settings" />
            <ModalBody>
              <div className="grid grid-cols-3 gap-4">
                {advancedParams.map(renderField)}
              </div>
            </ModalBody>
            <ModalFooter>
              <SecondaryButton size="md" onClick={handleClose}>
                Close
              </SecondaryButton>
              <PrimaryButton size="md" onClick={handleComplete}>
                Complete
              </PrimaryButton>
            </ModalFooter>
          </ComposedModal>
        </div>
      )}
    </div>
  );
}

interface DropdownFieldProps {
  param: ParameterConfig;
  provider: string;
  value: string;
  onChange: (value: string, selectedItem?: ProviderDataItem) => void;
  colSpanClass: string;
}

function DropdownField({
  param,
  provider,
  value,
  onChange,
  colSpanClass,
}: DropdownFieldProps) {
  const data = param.data
    ? (loadProviderData(provider, param.data) as ProviderDataItem[])
    : [];
  const valueField = param.valueField || 'id';
  const selectedItem =
    data.find(item => getProviderDataItemValue(item, valueField) === value) ||
    (param.customValue && value ? { [valueField]: value, name: value } : null);
  const commitCustomValue = (rawInput: string) => {
    const inputValue = rawInput.trim();
    if (!param.customValue || !inputValue) return;
    const hasMatch = data.some(
      item =>
        getProviderDataItemName(item).toLowerCase() ===
          inputValue.toLowerCase() ||
        getProviderDataItemValue(item, valueField) === inputValue,
    );
    if (!hasMatch) {
      onChange(inputValue);
    }
  };

  if (param.customValue || param.searchable) {
    return (
      <div className={cn(colSpanClass)} key={param.key}>
        <FieldLabelRow param={param} />
        <ComboBox
          id={`combo-${param.key}`}
          aria-label={param.label}
          items={data}
          selectedItem={selectedItem}
          itemToString={getProviderDataItemName}
          placeholder={`Select ${param.label.toLowerCase()}`}
          onChange={({
            selectedItem: item,
            inputValue,
          }: {
            selectedItem: ProviderDataItem | null | undefined;
            inputValue?: string | null;
          }) => {
            if (item) {
              const selectedValue =
                item[valueField] ?? item.name ?? inputValue ?? '';
              if (selectedValue) {
                onChange(String(selectedValue), item);
              }
              return;
            }
            commitCustomValue(inputValue || '');
          }}
          onBlur={(e: FocusEvent<HTMLInputElement>) =>
            commitCustomValue(e.target.value)
          }
          allowCustomValue={param.customValue}
        />
      </div>
    );
  }

  return (
    <div className={cn(colSpanClass)} key={param.key}>
      <FieldLabelRow param={param} />
      <CarbonDropdown
        id={`dropdown-${param.key}`}
        titleText={param.label}
        hideLabel
        label={`Select ${param.label.toLowerCase()}`}
        items={data}
        selectedItem={selectedItem}
        itemToString={getProviderDataItemName}
        onChange={({
          selectedItem: item,
        }: {
          selectedItem?: ProviderDataItem | null;
        }) => {
          if (item) {
            onChange(getProviderDataItemValue(item, valueField), item);
          }
        }}
      />
    </div>
  );
}

interface KeyValueFieldProps {
  param: ParameterConfig;
  value: string;
  onChange: (value: string) => void;
  colSpanClass: string;
}

function KeyValueField({
  param,
  value,
  onChange,
  colSpanClass,
}: KeyValueFieldProps) {
  const parseEntries = (raw: string): { key: string; value: string }[] => {
    if (!raw) return [];
    try {
      const obj = JSON.parse(raw);
      return Object.entries(obj).map(([key, value]) => ({
        key,
        value: String(value),
      }));
    } catch {
      return [];
    }
  };

  const serialize = (entries: { key: string; value: string }[]): string => {
    const obj: Record<string, string> = {};
    for (const e of entries) {
      if (e.key) obj[e.key] = e.value;
    }
    return Object.keys(obj).length > 0 ? JSON.stringify(obj) : '';
  };

  const [entries, setEntries] = useState<{ key: string; value: string }[]>(() =>
    parseEntries(value),
  );

  const syncEntries = (next: { key: string; value: string }[]) => {
    setEntries(next);
    onChange(serialize(next));
  };

  const updateEntry = (index: number, field: 'key' | 'value', val: string) => {
    const next = [...entries];
    next[index] = { ...next[index], [field]: val };
    syncEntries(next);
  };

  const removeEntry = (index: number) => {
    syncEntries(entries.filter((_, i) => i !== index));
  };

  const addEntry = () => {
    setEntries(prev => [...prev, { key: '', value: '' }]);
  };

  return (
    <div className={cn(colSpanClass, 'flex flex-col gap-4')} key={param.key}>
      <div className="inline-flex items-center gap-1">
        <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400">
          {param.label} ({entries.length})
        </p>
        <HelpToggletip label={param.label} helpText={param.helpText} />
      </div>
      <TableForm>
        <thead>
          <tr className="bg-[var(--cds-layer-accent-01)]">
            <TableFormHeaderCell className="w-1/2">Key</TableFormHeaderCell>
            <TableFormHeaderCell className="w-1/2">Value</TableFormHeaderCell>
            <TableFormHeaderCell className="w-10" divider={false} />
          </tr>
        </thead>
        <tbody>
          {entries.map((entry, index) => (
            <TableFormRow key={index}>
              <TableFormCell>
                <TextInput
                  id={`kv-key-${param.key}-${index}`}
                  labelText=""
                  hideLabel
                  value={entry.key}
                  onChange={e => updateEntry(index, 'key', e.target.value)}
                  placeholder="Key"
                  size="md"
                />
              </TableFormCell>
              <TableFormCell>
                <TextInput
                  id={`kv-val-${param.key}-${index}`}
                  labelText=""
                  hideLabel
                  value={entry.value}
                  onChange={e => updateEntry(index, 'value', e.target.value)}
                  placeholder="Value"
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
                  className="!text-[var(--cds-support-error)]"
                  onClick={() => removeEntry(index)}
                />
              </TableFormCell>
            </TableFormRow>
          ))}
        </tbody>
      </TableForm>
      <TertiaryButton
        size="md"
        renderIcon={Add}
        onClick={addEntry}
        className="!w-full !max-w-none"
      >
        Add {param.label.toLowerCase()}
      </TertiaryButton>
    </div>
  );
}

function renderTextMainDropdown(
  param: ParameterConfig,
  provider: string,
  getParamValue: (key: string) => string,
  updateMultipleParameters: (
    updates: { key: string; value: string }[],
    sourceParam?: ParameterConfig,
  ) => void,
  updateParameter: (
    key: string,
    value: string,
    sourceParam?: ParameterConfig,
  ) => void,
) {
  const data = param.data
    ? (loadProviderData(provider, param.data) as ProviderDataItem[])
    : [];
  const valueField = param.valueField || 'id';
  const currentValue = getParamValue(param.key);
  const selectedItem =
    data.find(
      item => getProviderDataItemValue(item, valueField) === currentValue,
    ) ||
    (param.customValue && currentValue
      ? { [valueField]: currentValue, name: currentValue }
      : null);
  const commitCustom = (rawInput: string) => {
    const vl = rawInput.trim();
    if (!vl) return;
    const hasMatch = data.some(
      item =>
        getProviderDataItemName(item).toLowerCase() === vl.toLowerCase() ||
        getProviderDataItemValue(item, valueField) === vl,
    );
    if (!hasMatch) {
      handleCustom(vl);
    }
  };

  const handleSelect = (item: ProviderDataItem | null) => {
    if (!item) return;
    const selectedValue = item[valueField] ?? item.name;
    if (!selectedValue) return;
    if (param.linkedField) {
      updateMultipleParameters(
        [
          { key: param.key, value: String(selectedValue) },
          {
            key: param.linkedField.key,
            value:
              getProviderDataItemValue(item, param.linkedField.sourceField) ||
              String(selectedValue),
          },
        ],
        param,
      );
    } else {
      updateParameter(
        param.key,
        getProviderDataItemValue(item, valueField),
        param,
      );
    }
  };

  const handleCustom = (vl: string) => {
    if (param.linkedField) {
      updateMultipleParameters(
        [
          { key: param.key, value: vl },
          { key: param.linkedField.key, value: vl },
        ],
        param,
      );
    } else {
      updateParameter(param.key, vl, param);
    }
  };

  if (param.customValue) {
    return (
      <ComboBox
        id={`text-main-combo-${param.key}`}
        aria-label={param.label}
        items={data}
        size="md"
        selectedItem={selectedItem}
        itemToString={getProviderDataItemName}
        placeholder="Select model"
        onChange={({
          selectedItem: item,
          inputValue,
        }: {
          selectedItem: ProviderDataItem | null | undefined;
          inputValue?: string | null;
        }) => {
          if (item) handleSelect(item);
          else commitCustom(inputValue || '');
        }}
        onBlur={(e: FocusEvent<HTMLInputElement>) =>
          commitCustom(e.target.value)
        }
        allowCustomValue
      />
    );
  }

  return (
    <CarbonDropdown
      id={`text-main-dropdown-${param.key}`}
      aria-label={param.label}
      titleText=""
      label="Select model"
      size="md"
      items={data}
      selectedItem={selectedItem}
      itemToString={getProviderDataItemName}
      onChange={({
        selectedItem: item,
      }: {
        selectedItem?: ProviderDataItem | null;
      }) => {
        if (item) handleSelect(item);
      }}
    />
  );
}
