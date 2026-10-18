import { Add, Information, TrashCan } from '@carbon/icons-react';
import {
  Button,
  Select,
  SelectItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  Tooltip,
} from '@carbon/react';
import { tableFormContainerClassName } from '@/app/components/ui/table';

export interface ConditionOption {
  label: string;
  value: string;
}

export interface SourceOption {
  label: string;
  value: string;
}

export interface KeyOption {
  label: string;
  value: string;
}

export interface SourceCondition {
  key: string;
  condition: string;
  value: string;
}

interface SourceConditionRuleProps<TCondition extends SourceCondition> {
  conditions: ReadonlyArray<TCondition>;
  onChangeConditions: (next: TCondition[]) => void;
  conditionOptions: ReadonlyArray<ConditionOption>;
  sourceOptions: ReadonlyArray<SourceOption>;
  keyOptions?: ReadonlyArray<KeyOption>;
  valueOptionsByKey?: Record<string, ReadonlyArray<SourceOption>>;
  keyTooltipText?: string;
}

export function SourceConditionRule<TCondition extends SourceCondition>({
  conditions,
  onChangeConditions,
  conditionOptions,
  sourceOptions,
  keyOptions = [{ label: 'Source', value: 'source' }],
  valueOptionsByKey,
  keyTooltipText = "The variable to evaluate for this condition. 'source' refers to the channel the call is coming from.",
}: SourceConditionRuleProps<TCondition>) {
  const defaultKey = keyOptions[0]?.value || 'source';
  const getValueOptions = (key: string) =>
    valueOptionsByKey?.[key] || sourceOptions;
  const createDefaultRow = (key: string): TCondition =>
    ({
      key,
      condition: conditionOptions[0]?.value || '=',
      value: getValueOptions(key)[0]?.value || '',
    }) as TCondition;
  const rows =
    conditions.length > 0 ? [...conditions] : [createDefaultRow(defaultKey)];
  const updateRow = (index: number, next: TCondition) => {
    const nextRows = [...rows];
    nextRows[index] = next;
    onChangeConditions(nextRows);
  };
  const addRow = () => {
    onChangeConditions([...rows, createDefaultRow(defaultKey)]);
  };
  const removeRow = (index: number) => {
    onChangeConditions(rows.filter((_, i) => i !== index));
  };

  return (
    <>
      <div className="mb-2 text-xs text-gray-500 flex items-center gap-1">
        <span>Rule</span>
        <Tooltip
          align="right"
          label="This rule is tested before the tool is added to the LLM tool list."
        >
          <Information size={14} />
        </Tooltip>
      </div>
      <TableContainer className={tableFormContainerClassName}>
        <Table size="sm" useZebraStyles={false}>
          <TableHead>
            <TableRow>
              <TableHeader>
                <span className="inline-flex items-center gap-1">
                  Key
                  <Tooltip align="right" label={keyTooltipText}>
                    <Information size={11} />
                  </Tooltip>
                </span>
              </TableHeader>
              <TableHeader>
                <span className="inline-flex items-center gap-1">
                  Condition
                  <Tooltip
                    align="right"
                    label="The condition to evaluate for this variable."
                  >
                    <Information size={11} />
                  </Tooltip>
                </span>
              </TableHeader>
              <TableHeader>
                <span className="inline-flex items-center gap-1">
                  Value
                  <Tooltip
                    align="right"
                    label="The value to compare against the variable."
                  >
                    <Information size={11} />
                  </Tooltip>
                </span>
              </TableHeader>
              <TableHeader className="!w-10 !text-center">Actions</TableHeader>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row, index) => (
              <TableRow key={`${row.key}-${index}`}>
                <TableCell className="!p-0">
                  <Select
                    id={
                      index === 0
                        ? 'tool-condition-key'
                        : `tool-condition-key-${index}`
                    }
                    labelText="Key"
                    hideLabel
                    value={row.key}
                    onChange={e => {
                      const nextKey = e.target.value;
                      const nextValueOptions = getValueOptions(nextKey);
                      updateRow(index, {
                        ...row,
                        key: nextKey,
                        value: nextValueOptions[0]?.value || '',
                      } as TCondition);
                    }}
                    size="md"
                    className="w-full"
                  >
                    {keyOptions.map(option => (
                      <SelectItem
                        key={option.value}
                        value={option.value}
                        text={option.label}
                      />
                    ))}
                  </Select>
                </TableCell>
                <TableCell className="!p-0">
                  <Select
                    id={
                      index === 0
                        ? 'tool-condition-op'
                        : `tool-condition-op-${index}`
                    }
                    labelText="Condition"
                    hideLabel
                    value={row.condition}
                    onChange={e =>
                      updateRow(index, {
                        ...row,
                        condition: e.target.value,
                      } as TCondition)
                    }
                    size="md"
                    className="w-full"
                  >
                    {conditionOptions.map(option => (
                      <SelectItem
                        key={option.value}
                        value={option.value}
                        text={option.label}
                      />
                    ))}
                  </Select>
                </TableCell>
                <TableCell className="!p-0">
                  <Select
                    id={
                      index === 0
                        ? 'tool-condition-source-value'
                        : `tool-condition-source-value-${index}`
                    }
                    labelText="Value"
                    hideLabel
                    value={row.value}
                    onChange={e =>
                      updateRow(index, {
                        ...row,
                        value: e.target.value,
                      } as TCondition)
                    }
                    size="md"
                    className="w-full"
                  >
                    {getValueOptions(row.key).map(option => (
                      <SelectItem
                        key={option.value}
                        value={option.value}
                        text={option.label}
                      />
                    ))}
                  </Select>
                </TableCell>
                <TableCell className="!w-10 !p-0 !text-center">
                  <Button
                    hasIconOnly
                    renderIcon={TrashCan}
                    iconDescription="Remove"
                    kind="ghost"
                    size="sm"
                    className="!text-[var(--cds-support-error)]"
                    disabled={rows.length <= 1}
                    onClick={() => removeRow(index)}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <div className="pt-4">
        <Button
          kind="tertiary"
          size="md"
          renderIcon={Add}
          onClick={addRow}
          className="!w-full !max-w-none"
        >
          Add rule
        </Button>
      </div>
    </>
  );
}
