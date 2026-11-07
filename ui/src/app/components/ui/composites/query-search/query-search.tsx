import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEventHandler,
  type RefObject,
} from 'react';
import { ChevronLeft, ChevronRight, Close, Search } from '@carbon/icons-react';
import { Button, Checkbox, FormLabel, Select, SelectItem } from '@carbon/react';
import {
  DEFAULT_QUERY_SEARCH_LABELS,
  MONTH_NAMES,
  TIME_OPTIONS,
  completeCurrentToken,
  createFilterChip,
  formatDateFilterValue,
  formatDateTimeValue,
  formatDateValue,
  formatFilterDisplayValue,
  formatMultiSelectDisplayValue,
  getCalendarOffset,
  getDateFromValue,
  getDateInputParts,
  getDaysInMonth,
  getFieldByKey,
  getCurrentTokenRange,
  getFilterRawKey,
  getSelectedLogic,
  isSameDateValue,
  joinMultiSelectValue,
  joinQueryParts,
  matchesOptionSearch,
  parseQueryFilterChip,
  parseQuerySearchFilters,
  quoteFilterValue,
  replaceCurrentToken,
  splitFilterToken,
  splitMultiSelectValue,
  splitQueryParts,
  unquoteFilterValue,
  type QueryFilterChip,
  type QuerySearchDateTimeMode,
  type QuerySearchField,
  type QuerySearchFilter,
  type QuerySearchLabels,
  type QuerySearchLogicOption,
  type QuerySearchOption,
  type QuerySearchProps,
  type QuerySearchTab,
} from './query-search-model';

export { parseQuerySearchFilters };
export type {
  QuerySearchDateTimeMode,
  QuerySearchField,
  QuerySearchFilter,
  QuerySearchLabels,
  QuerySearchLogicOption,
  QuerySearchOption,
  QuerySearchProps,
  QuerySearchTab,
} from './query-search-model';

type KeyPickerProps = {
  fields: QuerySearchField[];
  onSelect: (field: QuerySearchField) => void;
  selectedField: QuerySearchField;
};

const KeyPicker = ({ fields, onSelect, selectedField }: KeyPickerProps) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <span className="relative inline-flex">
      <button
        type="button"
        className="whitespace-nowrap bg-transparent text-link outline-none hover:underline"
        onClick={() => setIsOpen(open => !open)}
      >
        {selectedField.text}
      </button>
      {isOpen && (
        <div
          className="absolute left-0 top-[calc(100%+0.25rem)] z-50 max-h-[320px] w-[min(320px,calc(100vw-2rem))] overflow-auto border border-border-subtle bg-surface py-2 shadow-xl "
          onMouseDown={event => event.preventDefault()}
        >
          {fields.map(field => (
            <button
              key={field.queryKey}
              type="button"
              className={[
                'grid w-full grid-cols-[minmax(0,1fr)_auto] gap-3 px-4 py-2 text-left text-sm hover:bg-layer ',
                field.queryKey === selectedField.queryKey
                  ? 'text-link'
                  : 'text-foreground',
              ].join(' ')}
              onClick={() => {
                onSelect(field);
                setIsOpen(false);
              }}
            >
              <span className="truncate font-mono">{field.text}</span>
              <span className="text-muted">{field.logicLabel || 'is'}</span>
            </button>
          ))}
        </div>
      )}
    </span>
  );
};

type LogicPickerProps = {
  field: QuerySearchField;
  onSelect: (logic: string) => void;
  selectedLogic?: QuerySearchLogicOption;
};

const LogicPicker = ({ field, onSelect, selectedLogic }: LogicPickerProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const options = field.logicOptions || [
    {
      label: field.logicLabel || 'is',
      logic: '=',
    },
  ];

  return (
    <span className="relative inline-flex">
      <button
        type="button"
        className="whitespace-nowrap bg-transparent text-muted outline-none hover:text-foreground"
        onClick={() => setIsOpen(open => !open)}
      >
        {selectedLogic?.label || options[0].label}
      </button>
      {isOpen && options.length > 1 && (
        <div
          className="absolute left-0 top-[calc(100%+0.25rem)] z-50 min-w-32 overflow-hidden border border-border-subtle bg-surface shadow-xl "
          onMouseDown={event => event.preventDefault()}
        >
          {options.map(option => (
            <button
              key={option.logic}
              type="button"
              className={[
                'block w-full whitespace-nowrap px-3 py-2 text-left text-sm hover:bg-layer ',
                option.logic === selectedLogic?.logic
                  ? 'text-link'
                  : 'text-foreground',
              ].join(' ')}
              onClick={() => {
                onSelect(option.logic);
                setIsOpen(false);
              }}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </span>
  );
};

type ValueEditorProps = {
  dateTimeMode: QuerySearchDateTimeMode;
  field: QuerySearchField;
  inputRef: RefObject<HTMLInputElement>;
  isOpen: boolean;
  labels: QuerySearchLabels;
  onApplyValue: (nextValue: string) => void;
  onBlur: () => void;
  onChangeValue: (nextValue: string) => void;
  onCommitValue: (nextValue: string) => void;
  onFocus: () => void;
  onKeyDown: KeyboardEventHandler<HTMLInputElement>;
  timeOptions: string[];
  value: string;
  valueOptions: QuerySearchOption[];
  onSelectValue: (field: QuerySearchField, option: QuerySearchOption) => void;
  preserveDateOnly: boolean;
};

const getTextInputWidth = (value: string, minWidth = 5, maxWidth = 48) =>
  `${Math.min(Math.max(value.length || minWidth, minWidth), maxWidth)}ch`;

const TextFilterValueEditor = ({
  dateTimeMode: _dateTimeMode,
  field,
  inputRef,
  isOpen,
  labels: _labels,
  onApplyValue: _onApplyValue,
  onBlur,
  onCommitValue: _onCommitValue,
  onChangeValue,
  onFocus,
  onKeyDown,
  timeOptions: _timeOptions,
  onSelectValue,
  value,
  valueOptions,
}: ValueEditorProps) => {
  const hasValueOptions = valueOptions.length > 0;

  return (
    <>
      <input
        ref={inputRef}
        className="bg-transparent text-link outline-none placeholder:text-muted"
        type="text"
        style={{ width: getTextInputWidth(value) }}
        value={value}
        onBlur={onBlur}
        onChange={event => onChangeValue(event.target.value)}
        onFocus={onFocus}
        onKeyDown={onKeyDown}
      />
      {isOpen && hasValueOptions && (
        <div
          className="absolute left-0 top-[calc(100%+0.25rem)] z-40 w-[min(360px,calc(100vw-2rem))] overflow-hidden border border-border-subtle bg-surface shadow-xl "
          onMouseDown={event => event.preventDefault()}
        >
          <div className="max-h-[320px] overflow-auto py-2">
            {valueOptions.map(option => (
              <button
                key={option.id}
                type="button"
                className="grid w-full grid-cols-[minmax(0,1fr)_auto] gap-3 px-4 py-2 text-left text-sm hover:bg-layer "
                onClick={() => onSelectValue(field, option)}
              >
                <span className="truncate font-mono">{option.text}</span>
                <span className="text-link">{field.type}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </>
  );
};

const MultiSelectFilterValueEditor = ({
  dateTimeMode: _dateTimeMode,
  field,
  inputRef,
  isOpen,
  labels: _labels,
  onApplyValue,
  onBlur: _onBlur,
  onChangeValue: _onChangeValue,
  onCommitValue: _onCommitValue,
  onFocus,
  onKeyDown,
  timeOptions: _timeOptions,
  onSelectValue: _onSelectValue,
  value,
  valueOptions: _valueOptions,
}: ValueEditorProps) => {
  const [searchText, setSearchText] = useState('');
  const selectedValues = splitMultiSelectValue(value);
  const selectedValueSet = new Set(selectedValues);
  const selectedText = formatMultiSelectDisplayValue(field, value);
  const searchTerm = searchText.trim().toLowerCase();
  const options = (field.items || []).filter(option => {
    if (!searchTerm) return true;
    return matchesOptionSearch(option, searchTerm);
  });

  useEffect(() => {
    if (!isOpen) setSearchText('');
  }, [isOpen]);

  const toggleValue = (option: QuerySearchOption, checked: boolean) => {
    const nextValues = checked
      ? [...selectedValues, option.id]
      : selectedValues.filter(selectedValue => selectedValue !== option.id);
    onApplyValue(joinMultiSelectValue(nextValues));
  };

  return (
    <>
      <input
        ref={inputRef}
        className="bg-transparent text-link outline-none placeholder:text-link"
        type="text"
        style={{
          width: getTextInputWidth(searchText || selectedText, 14, 48),
        }}
        value={searchText}
        placeholder={selectedText}
        onChange={event => setSearchText(event.target.value)}
        onFocus={onFocus}
        onKeyDown={onKeyDown}
      />
      {isOpen && (
        <div
          className="absolute left-0 top-[calc(100%+0.25rem)] z-40 w-[min(360px,calc(100vw-2rem))] overflow-hidden border border-border-subtle bg-surface shadow-xl "
          onMouseDown={event => event.preventDefault()}
        >
          <div className="max-h-[320px] overflow-auto py-2">
            {options.length > 0 ? (
              options.map(option => (
                <div
                  key={option.id}
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-4 py-2 hover:bg-layer "
                >
                  <span className="truncate font-mono text-sm text-foreground">
                    {option.text}
                  </span>
                  <Checkbox
                    id={`query-search-${field.queryKey}-${option.id}`}
                    labelText={option.text}
                    hideLabel
                    checked={selectedValueSet.has(option.id)}
                    onChange={(_, data) => toggleValue(option, data.checked)}
                  />
                </div>
              ))
            ) : (
              <p className="px-4 py-3 text-sm text-muted">No values found</p>
            )}
          </div>
        </div>
      )}
    </>
  );
};

const DateFilterValueEditor = ({
  dateTimeMode,
  field,
  inputRef,
  isOpen,
  labels,
  onBlur,
  onCommitValue,
  onChangeValue,
  onFocus,
  onKeyDown,
  preserveDateOnly,
  timeOptions,
  value,
}: ValueEditorProps) => {
  const dateInputParts = getDateInputParts(value, dateTimeMode);
  const displayValue = formatDateTimeValue(value, dateTimeMode);
  const selectedDate = getDateFromValue(dateInputParts.dateValue);
  const [draftDateValue, setDraftDateValue] = useState(
    dateInputParts.dateValue,
  );
  const [draftTimeValue, setDraftTimeValue] = useState(
    dateInputParts.timeValue,
  );
  const [includeTime, setIncludeTime] = useState(dateInputParts.hasTime);
  const [calendarMonth, setCalendarMonth] = useState(
    selectedDate || new Date(),
  );
  const daysInMonth = getDaysInMonth(calendarMonth);
  const calendarOffset = getCalendarOffset(calendarMonth);

  useEffect(() => {
    const nextSelectedDate = getDateFromValue(dateInputParts.dateValue);
    setDraftDateValue(dateInputParts.dateValue);
    setDraftTimeValue(dateInputParts.timeValue);
    setIncludeTime(dateInputParts.hasTime);
    setCalendarMonth(nextSelectedDate || new Date());
  }, [
    dateInputParts.dateValue,
    dateInputParts.hasTime,
    dateInputParts.timeValue,
    isOpen,
  ]);

  return (
    <>
      <input
        ref={inputRef}
        className="bg-transparent text-link outline-none placeholder:text-muted"
        type="text"
        style={{ width: getTextInputWidth(displayValue, 10, 28) }}
        value={displayValue}
        onBlur={onBlur}
        onChange={event => onChangeValue(event.target.value)}
        onFocus={onFocus}
        onKeyDown={onKeyDown}
      />
      {isOpen && (
        <div
          className="query-search-date-calendar absolute left-0 top-[calc(100%+0.25rem)] z-40 w-[min(300px,calc(100vw-2rem))] overflow-hidden border border-border-subtle bg-surface shadow-xl "
          onMouseDown={event => {
            const target = event.target as HTMLElement;
            if (target.closest('.query-search-date-time')) return;
            event.preventDefault();
          }}
        >
          <div className="px-3 py-2">
            <div className="mb-2 flex items-center justify-between">
              <button
                type="button"
                aria-label={labels.previousMonth}
                className="flex h-6 w-6 items-center justify-center text-icon-primary hover:bg-layer "
                onClick={() =>
                  setCalendarMonth(
                    new Date(
                      calendarMonth.getFullYear(),
                      calendarMonth.getMonth() - 1,
                      1,
                    ),
                  )
                }
              >
                <ChevronLeft className="h-3.5 w-3.5" />
              </button>
              <div className="flex items-center gap-4 text-sm text-foreground">
                <span>{MONTH_NAMES[calendarMonth.getMonth()]}</span>
                <span>{calendarMonth.getFullYear()}</span>
              </div>
              <button
                type="button"
                aria-label={labels.nextMonth}
                className="flex h-6 w-6 items-center justify-center text-icon-primary hover:bg-layer "
                onClick={() =>
                  setCalendarMonth(
                    new Date(
                      calendarMonth.getFullYear(),
                      calendarMonth.getMonth() + 1,
                      1,
                    ),
                  )
                }
              >
                <ChevronRight className="h-3.5 w-3.5" />
              </button>
            </div>
            <div className="grid grid-cols-7 gap-y-1">
              {Array.from({ length: calendarOffset }).map((_, index) => (
                <span key={`empty-${index}`} className="h-7" />
              ))}
              {Array.from({ length: daysInMonth }).map((_, index) => {
                const date = new Date(
                  calendarMonth.getFullYear(),
                  calendarMonth.getMonth(),
                  index + 1,
                );
                const dateValue = formatDateValue(date);
                const isSelected = isSameDateValue(date, draftDateValue);

                return (
                  <button
                    key={dateValue}
                    type="button"
                    className={[
                      'mx-auto flex h-7 w-7 items-center justify-center text-xs hover:bg-layer ',
                      isSelected
                        ? 'bg-layer-selected outline outline-2 -outline-offset-2 outline-primary text-link'
                        : 'text-foreground',
                    ].join(' ')}
                    onClick={() => setDraftDateValue(dateValue)}
                  >
                    {index + 1}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="query-search-date-time border-t border-border-subtle px-3 py-2 ">
            <div className="query-search-date-form-row">
              <FormLabel
                id={`query-search-include-time-label-${field.queryKey}`}
                className="query-search-date-form-label"
              >
                {labels.includeTime}
              </FormLabel>
              <Checkbox
                id={`query-search-include-time-${field.queryKey}`}
                className="query-search-include-time"
                labelText={labels.includeTime}
                hideLabel
                checked={includeTime}
                onChange={(_, data) => {
                  setIncludeTime(data.checked);
                }}
              />
            </div>
            <div className="query-search-date-form-row mt-2">
              <FormLabel
                id={`query-search-time-label-${field.queryKey}`}
                className="query-search-date-form-label"
              >
                {labels.time}
              </FormLabel>
              <Select
                id={`query-search-time-${field.queryKey}`}
                labelText={labels.time}
                hideLabel
                size="sm"
                className="query-search-time-select"
                disabled={!includeTime || !draftDateValue}
                value={draftTimeValue}
                onChange={event => setDraftTimeValue(event.target.value)}
              >
                {timeOptions.map(timeOption => (
                  <SelectItem
                    key={timeOption}
                    value={timeOption}
                    text={timeOption}
                  />
                ))}
              </Select>
            </div>
            <div className="mt-3 flex justify-end">
              <Button
                size="sm"
                disabled={!draftDateValue}
                onClick={() =>
                  onCommitValue(
                    formatDateFilterValue(
                      draftDateValue,
                      draftTimeValue,
                      includeTime,
                      dateTimeMode,
                      preserveDateOnly,
                    ),
                  )
                }
              >
                {labels.save}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};

const FILTER_VALUE_EDITORS = {
  date: DateFilterValueEditor,
  'multi-select': MultiSelectFilterValueEditor,
  number: TextFilterValueEditor,
  string: TextFilterValueEditor,
};

type FilterPillProps = {
  dateTimeMode: QuerySearchDateTimeMode;
  fields: QuerySearchField[];
  field: QuerySearchField;
  inputRef: RefObject<HTMLInputElement>;
  isEditingValue: boolean;
  isOpen: boolean;
  labels: QuerySearchLabels;
  logic: QuerySearchLogicOption;
  onApplyValue: (nextValue: string) => void;
  onBlur: () => void;
  onChangeValue: (nextValue: string) => void;
  onCommitValue: (nextValue: string) => void;
  onEditValue: () => void;
  onKeyDown: KeyboardEventHandler<HTMLInputElement>;
  onRemove: () => void;
  onSelectField: (field: QuerySearchField) => void;
  onSelectLogic: (logic: string) => void;
  onSelectValue: (field: QuerySearchField, option: QuerySearchOption) => void;
  preserveDateOnly: boolean;
  timeOptions: string[];
  value: string;
  valueOptions: QuerySearchOption[];
};

const FilterPill = ({
  dateTimeMode,
  fields,
  field,
  inputRef,
  isEditingValue,
  isOpen,
  labels,
  logic,
  onApplyValue,
  onBlur,
  onChangeValue,
  onCommitValue,
  onEditValue,
  onKeyDown,
  onRemove,
  onSelectField,
  onSelectLogic,
  onSelectValue,
  preserveDateOnly,
  timeOptions,
  value,
  valueOptions,
}: FilterPillProps) => {
  const ValueEditor = FILTER_VALUE_EDITORS[field.type];
  const displayValue = formatFilterDisplayValue(field, value, dateTimeMode);

  return (
    <span className="relative inline-flex w-fit shrink-0 items-center gap-2 border border-border-subtle bg-layer px-2 py-1 font-mono text-sm ">
      <KeyPicker
        fields={fields}
        selectedField={field}
        onSelect={onSelectField}
      />
      <LogicPicker
        field={field}
        selectedLogic={logic}
        onSelect={onSelectLogic}
      />
      {isEditingValue ? (
        <ValueEditor
          dateTimeMode={dateTimeMode}
          field={field}
          inputRef={inputRef}
          isOpen={isOpen}
          labels={labels}
          value={value}
          valueOptions={valueOptions}
          onApplyValue={onApplyValue}
          onBlur={onBlur}
          onCommitValue={onCommitValue}
          onChangeValue={onChangeValue}
          onFocus={onEditValue}
          onKeyDown={onKeyDown}
          preserveDateOnly={preserveDateOnly}
          timeOptions={timeOptions}
          onSelectValue={onSelectValue}
        />
      ) : (
        <button
          type="button"
          className="min-w-4 whitespace-nowrap bg-transparent text-left text-link outline-none hover:underline"
          onClick={onEditValue}
        >
          {displayValue}
        </button>
      )}
      <button
        type="button"
        aria-label={`Remove ${field.text} filter`}
        className="ml-1 flex h-4 w-4 cursor-pointer items-center justify-center text-muted hover:text-foreground "
        onClick={onRemove}
      >
        <Close className="h-3.5 w-3.5" />
      </button>
    </span>
  );
};

export const QuerySearch = ({
  className,
  dateTimeMode = 'raw',
  fields,
  labels,
  maxOptions = 10,
  onApply,
  onChange,
  placeholder = 'Search or filter',
  preserveDateOnly = false,
  tabs,
  timeOptions = TIME_OPTIONS,
  value,
}: QuerySearchProps) => {
  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const resolvedLabels = {
    ...DEFAULT_QUERY_SEARCH_LABELS,
    ...labels,
  };
  const resolvedTabs = tabs?.length
    ? tabs
    : [{ id: 'all', text: resolvedLabels.allTab }];
  const [activeTab, setActiveTab] = useState(resolvedTabs[0]?.id || 'all');
  const [editingChipIndex, setEditingChipIndex] = useState<number | null>(null);
  const [isEditingDraft, setIsEditingDraft] = useState(false);
  const [isOpen, setIsOpen] = useState(false);
  const valueEndsWithSpace = /\s$/.test(value);
  const queryParts = splitQueryParts(value);
  const displayParts = queryParts.reduce(
    (next, part, index) => {
      const chip = parseQueryFilterChip(fields, part.text);
      const isCurrentPart = index === queryParts.length - 1;
      if (chip && (!isCurrentPart || valueEndsWithSpace || !isEditingDraft)) {
        next.chips.push(chip);
        return next;
      }
      next.draftParts.push(part.text);
      return next;
    },
    { chips: [] as QueryFilterChip[], draftParts: [] as string[] },
  );
  const chipTokens = displayParts.chips;
  const draftValue = displayParts.draftParts.join(' ');
  const hasSearchValue = value.trim() !== '';
  const currentToken = getCurrentTokenRange(draftValue);
  const currentFilterToken = splitFilterToken(currentToken.text);
  const currentValue = currentFilterToken?.value || '';
  const selectedField = currentFilterToken
    ? getFieldByKey(fields, currentFilterToken.key)
    : undefined;
  const isValueMode = Boolean(selectedField);
  const currentDisplayValue = unquoteFilterValue(currentValue);
  const selectedLogic = selectedField
    ? getSelectedLogic(selectedField, currentFilterToken?.logic)
    : undefined;
  const allTabId = resolvedTabs[0]?.id || 'all';

  const fieldOptions = useMemo(
    () =>
      fields
        .filter(field => {
          const matchesTab =
            activeTab === allTabId || field.category === activeTab;
          const matchesSearch =
            !currentToken.text ||
            field.text
              .toLowerCase()
              .includes(currentToken.text.toLowerCase()) ||
            field.queryKey
              .toLowerCase()
              .includes(currentToken.text.toLowerCase());
          return matchesTab && matchesSearch;
        })
        .slice(0, maxOptions),
    [activeTab, allTabId, currentToken.text, fields, maxOptions],
  );

  const valueOptions = useMemo(() => {
    if (!selectedField?.items) return [];
    const search = currentValue.toLowerCase();
    return selectedField.items
      .filter(option => matchesOptionSearch(option, search))
      .slice(0, maxOptions);
  }, [currentValue, maxOptions, selectedField]);

  const getValueOptions = (field: QuerySearchField, fieldValue: string) => {
    if (!field.items) return [];
    const search = fieldValue.toLowerCase();
    return field.items
      .filter(option => matchesOptionSearch(option, search))
      .slice(0, maxOptions);
  };

  const applyNextValue = (nextValue: string) => {
    setIsEditingDraft(false);
    onChange(nextValue);
    onApply(nextValue);
  };

  const setDraftValue = (nextDraft: string) => {
    setIsEditingDraft(true);
    onChange(joinQueryParts(chipTokens, nextDraft));
  };

  const closeWhenFocusLeaves = () => {
    window.setTimeout(() => {
      if (rootRef.current?.contains(document.activeElement)) return;
      setEditingChipIndex(null);
      setIsEditingDraft(false);
      setIsOpen(false);
    }, 150);
  };

  useEffect(() => {
    if (!isOpen) return;

    const closeWhenClickLeaves = (event: MouseEvent) => {
      if (rootRef.current?.contains(event.target as Node)) return;
      setEditingChipIndex(null);
      setIsEditingDraft(false);
      setIsOpen(false);
    };

    document.addEventListener('mousedown', closeWhenClickLeaves);
    return () => {
      document.removeEventListener('mousedown', closeWhenClickLeaves);
    };
  }, [isOpen]);

  const setChipTokens = (nextChips: QueryFilterChip[], shouldApply = false) => {
    const nextValue = joinQueryParts(nextChips, draftValue);
    if (shouldApply) {
      applyNextValue(nextValue);
      return;
    }
    setIsEditingDraft(false);
    onChange(nextValue);
  };

  const updateChip = (
    index: number,
    nextChip: QueryFilterChip,
    shouldApply = false,
  ) => {
    const nextChips = chipTokens.map((chip, chipIndex) =>
      chipIndex === index ? nextChip : chip,
    );
    setChipTokens(nextChips, shouldApply);
  };

  const editChipValue = (index: number) => {
    setEditingChipIndex(index);
    setIsOpen(true);
    window.requestAnimationFrame(() => inputRef.current?.focus());
  };

  const updateChipField = (index: number, field: QuerySearchField) => {
    const chip = chipTokens[index];
    const logic = getSelectedLogic(field, chip.logic);
    updateChip(index, createFilterChip(field, logic.logic, chip.value), true);
  };

  const updateChipLogic = (index: number, logic: string) => {
    const chip = chipTokens[index];
    const field = getFieldByKey(fields, chip.key);
    if (!field) return;
    updateChip(index, createFilterChip(field, logic, chip.value), true);
  };

  const updateChipValue = (
    index: number,
    nextValue: string,
    shouldApply = false,
  ) => {
    const chip = chipTokens[index];
    const field = getFieldByKey(fields, chip.key);
    if (!field) return;
    updateChip(
      index,
      createFilterChip(field, chip.logic, nextValue),
      shouldApply,
    );
    setIsOpen(true);
  };

  const selectChipValue = (
    index: number,
    field: QuerySearchField,
    option: QuerySearchOption,
  ) => {
    const chip = chipTokens[index];
    updateChip(index, createFilterChip(field, chip.logic, option.id), true);
    setEditingChipIndex(null);
    setIsOpen(false);
  };

  const updateCurrentFilterValue = (nextValue: string) => {
    if (!selectedField || !selectedLogic) return;
    setDraftValue(
      replaceCurrentToken(
        draftValue,
        `${getFilterRawKey(
          selectedField,
          selectedLogic.logic,
        )}:${quoteFilterValue(nextValue)}`,
      ),
    );
    setIsOpen(true);
  };

  const commitCurrentFilterValue = (nextValue: string) => {
    if (!selectedField || !selectedLogic) return;
    const nextDraft = completeCurrentToken(
      draftValue,
      `${getFilterRawKey(
        selectedField,
        selectedLogic.logic,
      )}:${quoteFilterValue(nextValue)}`,
    );
    applyNextValue(`${joinQueryParts(chipTokens, nextDraft)} `);
    setIsOpen(false);
  };

  const applyCurrentFilterValue = (nextValue: string) => {
    if (!selectedField || !selectedLogic) return;
    const nextDraft = completeCurrentToken(
      draftValue,
      `${getFilterRawKey(
        selectedField,
        selectedLogic.logic,
      )}:${quoteFilterValue(nextValue)}`,
    );
    applyNextValue(`${joinQueryParts(chipTokens, nextDraft)} `);
    setEditingChipIndex(chipTokens.length);
    setIsOpen(true);
  };

  const selectField = (field: QuerySearchField) => {
    const nextDraft = replaceCurrentToken(draftValue, `${field.queryKey}:`);
    setDraftValue(nextDraft);
    setIsOpen(true);
    window.requestAnimationFrame(() => inputRef.current?.focus());
  };

  const selectDraftField = (field: QuerySearchField) => {
    const logic = getSelectedLogic(field);
    const nextDraft = replaceCurrentToken(
      draftValue,
      `${getFilterRawKey(field, logic.logic)}:${quoteFilterValue(
        currentDisplayValue,
      )}`,
    );
    setDraftValue(nextDraft);
    setIsOpen(true);
    window.requestAnimationFrame(() => inputRef.current?.focus());
  };

  const selectLogic = (field: QuerySearchField, logic: string) => {
    const nextDraft = replaceCurrentToken(
      draftValue,
      `${getFilterRawKey(field, logic)}:${quoteFilterValue(
        currentDisplayValue,
      )}`,
    );
    setDraftValue(nextDraft);
    setIsOpen(true);
    window.requestAnimationFrame(() => inputRef.current?.focus());
  };

  const selectValue = (field: QuerySearchField, option: QuerySearchOption) => {
    const logic = selectedLogic?.logic || getSelectedLogic(field).logic;
    const nextDraft = completeCurrentToken(
      draftValue,
      `${getFilterRawKey(field, logic)}:${quoteFilterValue(option.id)}`,
    );
    applyNextValue(`${joinQueryParts(chipTokens, nextDraft)} `);
    setIsOpen(false);
  };

  const removeChip = (index: number) => {
    const nextChips = chipTokens.filter((_, chipIndex) => chipIndex !== index);
    setEditingChipIndex(null);
    applyNextValue(joinQueryParts(nextChips, draftValue));
  };

  const clearDraftFilter = () => {
    setDraftValue(replaceCurrentToken(draftValue, ''));
    setIsOpen(false);
    window.requestAnimationFrame(() => inputRef.current?.focus());
  };

  const fieldOptionRows = fieldOptions.map(field => (
    <button
      key={field.queryKey}
      type="button"
      className="grid w-full grid-cols-[minmax(0,1fr)_auto] gap-3 px-4 py-2 text-left text-sm hover:bg-layer "
      onClick={() => selectField(field)}
    >
      <span className="truncate font-mono">{field.text}</span>
      <span className="text-link">{field.logicLabel || field.type}</span>
    </button>
  ));

  const handleValueKeyDown: KeyboardEventHandler<HTMLInputElement> = event => {
    if (event.key === 'Backspace' && !currentDisplayValue) {
      clearDraftFilter();
    }
    if (event.key === 'Enter') {
      const nextDraftChip = parseQueryFilterChip(fields, draftValue.trim());
      const nextValue = nextDraftChip
        ? `${joinQueryParts([...chipTokens, nextDraftChip], '')} `
        : joinQueryParts(chipTokens, draftValue);
      applyNextValue(nextValue);
      setIsOpen(false);
    }
    if (event.key === 'Escape') setIsOpen(false);
  };

  const getChipValueKeyDown =
    (index: number): KeyboardEventHandler<HTMLInputElement> =>
    event => {
      if (event.key === 'Backspace' && !chipTokens[index]?.value) {
        removeChip(index);
      }
      if (event.key === 'Enter') {
        const chip = chipTokens[index];
        if (chip) updateChipValue(index, chip.value, true);
        setEditingChipIndex(null);
        setIsOpen(false);
      }
      if (event.key === 'Escape') {
        setEditingChipIndex(null);
        setIsOpen(false);
      }
    };

  return (
    <div
      ref={rootRef}
      className={['relative min-w-0 flex-1', className]
        .filter(Boolean)
        .join(' ')}
    >
      <div
        className={[
          'flex h-12 min-w-0 items-center gap-2 border-0 bg-transparent px-3 text-sm text-foreground transition-colors hover:bg-field',
          isOpen
            ? 'bg-field outline outline-2 -outline-offset-2 outline-primary'
            : '',
        ].join(' ')}
      >
        <Search className="h-4 w-4 shrink-0 text-muted" />
        {chipTokens.map((chip, index) => {
          const chipField = getFieldByKey(fields, chip.key);
          if (!chipField) return null;
          const chipLogic = getSelectedLogic(chipField, chip.logic);

          return (
            <FilterPill
              key={`chip-${index}`}
              dateTimeMode={dateTimeMode}
              fields={fields}
              field={chipField}
              inputRef={inputRef}
              isEditingValue={editingChipIndex === index}
              isOpen={isOpen && editingChipIndex === index}
              labels={resolvedLabels}
              logic={chipLogic}
              value={chip.value}
              valueOptions={getValueOptions(chipField, chip.value)}
              onApplyValue={nextValue =>
                updateChipValue(index, nextValue, true)
              }
              onBlur={closeWhenFocusLeaves}
              onChangeValue={nextValue => updateChipValue(index, nextValue)}
              onCommitValue={nextValue => {
                updateChipValue(index, nextValue, true);
                setEditingChipIndex(null);
                setIsOpen(false);
              }}
              onEditValue={() => editChipValue(index)}
              onKeyDown={getChipValueKeyDown(index)}
              onRemove={() => removeChip(index)}
              onSelectField={field => updateChipField(index, field)}
              onSelectLogic={queryKey => updateChipLogic(index, queryKey)}
              onSelectValue={(field, option) =>
                selectChipValue(index, field, option)
              }
              preserveDateOnly={preserveDateOnly}
              timeOptions={timeOptions}
            />
          );
        })}
        {isValueMode && selectedField && selectedLogic ? (
          <FilterPill
            dateTimeMode={dateTimeMode}
            fields={fields}
            field={selectedField}
            inputRef={inputRef}
            isEditingValue
            isOpen={isOpen && editingChipIndex === null}
            labels={resolvedLabels}
            logic={selectedLogic}
            value={currentDisplayValue}
            valueOptions={valueOptions}
            onApplyValue={applyCurrentFilterValue}
            onBlur={closeWhenFocusLeaves}
            onChangeValue={updateCurrentFilterValue}
            onCommitValue={commitCurrentFilterValue}
            onEditValue={() => {
              setEditingChipIndex(null);
              setIsOpen(true);
            }}
            onKeyDown={handleValueKeyDown}
            onRemove={clearDraftFilter}
            onSelectField={selectDraftField}
            onSelectLogic={queryKey => selectLogic(selectedField, queryKey)}
            onSelectValue={selectValue}
            preserveDateOnly={preserveDateOnly}
            timeOptions={timeOptions}
          />
        ) : (
          <input
            ref={inputRef}
            className="min-w-[180px] flex-1 bg-transparent font-mono text-sm outline-none placeholder:font-sans placeholder:text-muted"
            placeholder={hasSearchValue ? '' : placeholder}
            value={draftValue}
            onBlur={closeWhenFocusLeaves}
            onChange={event => {
              const nextDraft = event.target.value;
              const nextDraftChip = parseQueryFilterChip(
                fields,
                nextDraft.trim(),
              );
              if (nextDraftChip && /\s$/.test(nextDraft)) {
                onChange(
                  `${joinQueryParts([...chipTokens, nextDraftChip], '')} `,
                );
              } else {
                setDraftValue(nextDraft);
              }
              setIsOpen(true);
            }}
            onFocus={() => setIsOpen(true)}
            onKeyDown={event => {
              if (event.key === 'Enter') {
                const nextDraftChip = parseQueryFilterChip(
                  fields,
                  draftValue.trim(),
                );
                const nextValue = nextDraftChip
                  ? `${joinQueryParts([...chipTokens, nextDraftChip], '')} `
                  : joinQueryParts(chipTokens, draftValue);
                applyNextValue(nextValue);
                setIsOpen(false);
              }
              if (event.key === 'Escape') setIsOpen(false);
            }}
          />
        )}
      </div>

      {isOpen && !isValueMode && editingChipIndex === null && (
        <div
          className="absolute left-0 top-[calc(100%+0.125rem)] z-40 w-[min(760px,calc(100vw-2rem))] overflow-hidden border border-border-subtle bg-surface shadow-xl "
          onMouseDown={event => event.preventDefault()}
        >
          {!isValueMode && (
            <div className="flex border-b border-border-subtle px-2 pt-2 ">
              {resolvedTabs.map(tab => (
                <button
                  key={tab.id}
                  type="button"
                  className={[
                    'px-3 py-2 text-sm',
                    activeTab === tab.id
                      ? 'border-b-2 border-primary text-foreground'
                      : 'text-muted hover:text-foreground ',
                  ].join(' ')}
                  onClick={() => setActiveTab(tab.id)}
                >
                  {tab.text}
                </button>
              ))}
            </div>
          )}

          <div className="max-h-[320px] overflow-auto py-2">
            {fieldOptionRows}
          </div>
        </div>
      )}
    </div>
  );
};
