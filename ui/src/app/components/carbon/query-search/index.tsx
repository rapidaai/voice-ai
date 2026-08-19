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

export type QuerySearchOption = {
  id: string;
  text: string;
};

export type QuerySearchField = {
  category?: string;
  formatValue?: (value: string) => string;
  items?: QuerySearchOption[];
  logicLabel?: string;
  logicOptions?: QuerySearchLogicOption[];
  queryKey: string;
  text: string;
  type: 'date' | 'multi-select' | 'number' | 'string';
};

export type QuerySearchLogicOption = {
  label: string;
  logic: string;
};

export type QuerySearchDateTimeMode = 'local-to-utc' | 'raw';

export type QuerySearchTab = {
  id: string;
  text: string;
};

export type QuerySearchLabels = {
  allTab: string;
  includeTime: string;
  nextMonth: string;
  previousMonth: string;
  save: string;
  time: string;
};

export type QuerySearchProps = {
  className?: string;
  dateTimeMode?: QuerySearchDateTimeMode;
  fields: QuerySearchField[];
  labels?: Partial<QuerySearchLabels>;
  maxOptions?: number;
  onApply: (value: string) => void;
  onChange: (value: string) => void;
  placeholder?: string;
  preserveDateOnly?: boolean;
  tabs?: QuerySearchTab[];
  timeOptions?: string[];
  value: string;
};

type QueryTokenPart = {
  end: number;
  start: number;
  text: string;
};

type QueryFilterChip = {
  key: string;
  label: string;
  logic: string;
  raw: string;
  value: string;
};

export type QuerySearchFilter = QueryFilterChip;

const padDatePart = (value: number): string => String(value).padStart(2, '0');

const hasExplicitTimeZone = (value: string): boolean =>
  /(?:z|[+-]\d{2}:?\d{2})$/i.test(value.trim());

const getLocalDateParts = (
  date: Date,
): { dateValue: string; hasTime: boolean; timeValue: string } => ({
  dateValue: [
    date.getFullYear(),
    padDatePart(date.getMonth() + 1),
    padDatePart(date.getDate()),
  ].join('-'),
  hasTime: true,
  timeValue: `${padDatePart(date.getHours())}:${padDatePart(
    date.getMinutes(),
  )}`,
});

const getDateInputParts = (
  value: string,
  dateTimeMode: QuerySearchDateTimeMode,
): { dateValue: string; hasTime: boolean; timeValue: string } => {
  const emptyParts = { dateValue: '', hasTime: false, timeValue: '00:00' };
  if (!value) return emptyParts;

  if (dateTimeMode === 'local-to-utc' && hasExplicitTimeZone(value)) {
    const date = new Date(value);
    if (Number.isFinite(date.getTime())) return getLocalDateParts(date);
  }

  const localDateTime = value.match(
    /^(\d{4}-\d{2}-\d{2})(?:[T\s](\d{2}:\d{2})(?::\d{2}(?:\.\d{1,3})?)?)?/,
  );
  if (localDateTime) {
    const hasTime = /^[0-9]{4}-[0-9]{2}-[0-9]{2}[T\s]/.test(value);
    return {
      dateValue: localDateTime[1],
      hasTime,
      timeValue: hasTime ? localDateTime[2] : emptyParts.timeValue,
    };
  }

  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return emptyParts;

  return getLocalDateParts(date);
};

const formatDateTimeValue = (
  value: string,
  dateTimeMode: QuerySearchDateTimeMode,
): string => {
  const dateParts = getDateInputParts(value, dateTimeMode);
  if (!dateParts.dateValue) return value;
  return dateParts.hasTime
    ? `${dateParts.dateValue} ${dateParts.timeValue}`
    : dateParts.dateValue;
};

const formatDateValue = (date: Date): string =>
  [
    date.getFullYear(),
    padDatePart(date.getMonth() + 1),
    padDatePart(date.getDate()),
  ].join('-');

const formatDateFilterValue = (
  dateValue: string,
  timeValue: string,
  includeTime: boolean,
  dateTimeMode: QuerySearchDateTimeMode,
  preserveDateOnly: boolean,
): string => {
  if (!includeTime && preserveDateOnly) return dateValue;

  if (dateTimeMode === 'raw') {
    return includeTime ? `${dateValue}T${timeValue || '00:00'}` : dateValue;
  }

  const [year, month, day] = dateValue.split('-').map(Number);
  if (!year || !month || !day) return dateValue;

  const [hour = 0, minute = 0] = includeTime
    ? (timeValue || '00:00').split(':').map(Number)
    : [];
  return new Date(year, month - 1, day, hour, minute, 0, 0).toISOString();
};

const MONTH_NAMES = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
];

const getDateFromValue = (value: string): Date | null => {
  if (!value) return null;
  const [year, month, day] = value.split('-').map(Number);
  if (!year || !month || !day) return null;
  return new Date(year, month - 1, day);
};

const isSameDateValue = (date: Date, dateValue: string): boolean =>
  formatDateValue(date) === dateValue;

const getCalendarOffset = (month: Date): number =>
  new Date(month.getFullYear(), month.getMonth(), 1).getDay();

const getDaysInMonth = (month: Date): number =>
  new Date(month.getFullYear(), month.getMonth() + 1, 0).getDate();

const TIME_OPTIONS = Array.from({ length: 96 }, (_, index) => {
  const minutes = index * 15;
  return `${padDatePart(Math.floor(minutes / 60))}:${padDatePart(
    minutes % 60,
  )}`;
});

const DEFAULT_QUERY_SEARCH_LABELS: QuerySearchLabels = {
  allTab: 'All',
  includeTime: 'Include time',
  nextMonth: 'Next month',
  previousMonth: 'Previous month',
  save: 'Save',
  time: 'Time',
};

const QUERY_TOKEN_PATTERN = /"([^"\\]|\\.)*"|'([^'\\]|\\.)*'|\S+/g;

const splitQueryParts = (query: string): QueryTokenPart[] =>
  Array.from(query.matchAll(QUERY_TOKEN_PATTERN)).map(match => ({
    end: (match.index || 0) + match[0].length,
    start: match.index || 0,
    text: match[0],
  }));

const getCurrentTokenRange = (value: string) => {
  const queryParts = splitQueryParts(value);
  const lastPart = queryParts[queryParts.length - 1];
  if (!lastPart || /\s$/.test(value)) {
    return {
      end: value.length,
      start: value.length,
      text: '',
    };
  }
  return lastPart;
};

const quoteFilterValue = (value: string): string =>
  /\s/.test(value) ? `"${value.replace(/"/g, '\\"')}"` : value;

const splitMultiSelectValue = (value: string): string[] =>
  Array.from(
    new Set(
      value
        .split(',')
        .map(item => item.trim())
        .filter(Boolean),
    ),
  );

const joinMultiSelectValue = (values: string[]): string =>
  Array.from(new Set(values.map(value => value.trim()).filter(Boolean))).join(
    ',',
  );

const getOptionText = (field: QuerySearchField, value: string): string =>
  field.items?.find(option => option.id === value)?.text || value;

const formatMultiSelectDisplayValue = (
  field: QuerySearchField,
  value: string,
): string =>
  splitMultiSelectValue(value)
    .map(selectedValue => getOptionText(field, selectedValue))
    .join(' or ');

const formatFilterDisplayValue = (
  field: QuerySearchField,
  value: string,
  dateTimeMode: QuerySearchDateTimeMode,
): string => {
  if (field.type === 'date') return formatDateTimeValue(value, dateTimeMode);
  if (field.type === 'multi-select') {
    return formatMultiSelectDisplayValue(field, value);
  }
  return field.formatValue?.(value) || value;
};

const unquoteFilterValue = (value: string): string => {
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1);
  }
  return value;
};

const splitFilterToken = (
  token: string,
): { key: string; logic?: string; value: string } | null => {
  const separatorIndex = token.indexOf(':');
  if (separatorIndex <= 0) return null;
  const keyToken = token.slice(0, separatorIndex);
  const logicIndex = keyToken.indexOf('~');

  return {
    key: logicIndex > 0 ? keyToken.slice(0, logicIndex) : keyToken,
    logic: logicIndex > 0 ? keyToken.slice(logicIndex + 1) : undefined,
    value: token.slice(separatorIndex + 1),
  };
};

const joinQueryParts = (chips: QueryFilterChip[], draft: string): string =>
  [...chips.map(chip => chip.raw), draft.trim()].filter(Boolean).join(' ');

const replaceCurrentToken = (value: string, nextToken: string): string => {
  const token = getCurrentTokenRange(value);
  const before = value.slice(0, token.start).trimEnd();
  const after = value.slice(token.end).trimStart();
  return [before, nextToken, after].filter(Boolean).join(' ');
};

const completeCurrentToken = (value: string, nextToken: string): string =>
  replaceCurrentToken(value, nextToken).trim();

const getFieldByKey = (
  fields: QuerySearchField[],
  queryKey: string,
): QuerySearchField | undefined =>
  fields.find(field => field.queryKey === queryKey);

const getSelectedLogic = (
  field: QuerySearchField,
  logic?: string,
): QuerySearchLogicOption =>
  field.logicOptions?.find(option => option.logic === logic) || {
    label: field.logicLabel || 'is',
    logic: field.logicOptions?.[0]?.logic || '=',
  };

const getFilterRawKey = (field: QuerySearchField, logic: string): string => {
  const defaultLogic = getSelectedLogic(field).logic;
  return logic === defaultLogic ? field.queryKey : `${field.queryKey}~${logic}`;
};

const matchesOptionSearch = (option: QuerySearchOption, search: string) =>
  option.id.toLowerCase().includes(search) ||
  option.text.toLowerCase().includes(search);

const parseQueryFilterChip = (
  fields: QuerySearchField[],
  token: string,
): QueryFilterChip | null => {
  const filterToken = splitFilterToken(token);
  if (!filterToken) return null;

  const { key, value: rawValue } = filterToken;
  const field = getFieldByKey(fields, key);
  if (!field) return null;
  const logic = getSelectedLogic(field, filterToken.logic).logic;

  return {
    key,
    label: field.text,
    logic,
    raw: token,
    value: unquoteFilterValue(rawValue),
  };
};

export const parseQuerySearchFilters = (
  fields: QuerySearchField[],
  query: string,
): QuerySearchFilter[] =>
  splitQueryParts(query)
    .map(part => parseQueryFilterChip(fields, part.text))
    .filter((filter): filter is QuerySearchFilter => Boolean(filter));

const createFilterChip = (
  field: QuerySearchField,
  logic: string,
  value: string,
): QueryFilterChip => ({
  key: field.queryKey,
  label: field.text,
  logic,
  raw: `${getFilterRawKey(field, logic)}:${quoteFilterValue(value)}`,
  value,
});

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
        className="whitespace-nowrap bg-transparent text-[var(--cds-link-primary)] outline-none hover:underline"
        onClick={() => setIsOpen(open => !open)}
      >
        {selectedField.text}
      </button>
      {isOpen && (
        <div
          className="absolute left-0 top-[calc(100%+0.25rem)] z-50 max-h-[320px] w-[min(320px,calc(100vw-2rem))] overflow-auto border border-gray-200 bg-white py-2 shadow-xl dark:border-gray-800 dark:bg-gray-950"
          onMouseDown={event => event.preventDefault()}
        >
          {fields.map(field => (
            <button
              key={field.queryKey}
              type="button"
              className={[
                'grid w-full grid-cols-[minmax(0,1fr)_auto] gap-3 px-4 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-900',
                field.queryKey === selectedField.queryKey
                  ? 'text-[var(--cds-link-primary)]'
                  : 'text-[var(--cds-text-primary)]',
              ].join(' ')}
              onClick={() => {
                onSelect(field);
                setIsOpen(false);
              }}
            >
              <span className="truncate font-mono">{field.text}</span>
              <span className="text-gray-500">{field.logicLabel || 'is'}</span>
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
        className="whitespace-nowrap bg-transparent text-gray-500 outline-none hover:text-[var(--cds-text-primary)]"
        onClick={() => setIsOpen(open => !open)}
      >
        {selectedLogic?.label || options[0].label}
      </button>
      {isOpen && options.length > 1 && (
        <div
          className="absolute left-0 top-[calc(100%+0.25rem)] z-50 min-w-32 overflow-hidden border border-gray-200 bg-white shadow-xl dark:border-gray-800 dark:bg-gray-950"
          onMouseDown={event => event.preventDefault()}
        >
          {options.map(option => (
            <button
              key={option.logic}
              type="button"
              className={[
                'block w-full whitespace-nowrap px-3 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-900',
                option.logic === selectedLogic?.logic
                  ? 'text-[var(--cds-link-primary)]'
                  : 'text-[var(--cds-text-primary)]',
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
        className="bg-transparent text-[var(--cds-link-primary)] outline-none placeholder:text-gray-400"
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
          className="absolute left-0 top-[calc(100%+0.25rem)] z-40 w-[min(360px,calc(100vw-2rem))] overflow-hidden border border-gray-200 bg-white shadow-xl dark:border-gray-800 dark:bg-gray-950"
          onMouseDown={event => event.preventDefault()}
        >
          <div className="max-h-[320px] overflow-auto py-2">
            {valueOptions.map(option => (
              <button
                key={option.id}
                type="button"
                className="grid w-full grid-cols-[minmax(0,1fr)_auto] gap-3 px-4 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-900"
                onClick={() => onSelectValue(field, option)}
              >
                <span className="truncate font-mono">{option.text}</span>
                <span className="text-[var(--cds-link-primary)]">
                  {field.type}
                </span>
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
  const normalizedSearch = searchText.trim().toLowerCase();
  const options = (field.items || []).filter(option => {
    if (!normalizedSearch) return true;
    return matchesOptionSearch(option, normalizedSearch);
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
        className="bg-transparent text-[var(--cds-link-primary)] outline-none placeholder:text-[var(--cds-link-primary)]"
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
          className="absolute left-0 top-[calc(100%+0.25rem)] z-40 w-[min(360px,calc(100vw-2rem))] overflow-hidden border border-gray-200 bg-white shadow-xl dark:border-gray-800 dark:bg-gray-950"
          onMouseDown={event => event.preventDefault()}
        >
          <div className="max-h-[320px] overflow-auto py-2">
            {options.length > 0 ? (
              options.map(option => (
                <div
                  key={option.id}
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-900"
                >
                  <span className="truncate font-mono text-sm text-[var(--cds-text-primary)]">
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
              <p className="px-4 py-3 text-sm text-gray-500">No values found</p>
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
        className="bg-transparent text-[var(--cds-link-primary)] outline-none placeholder:text-gray-400"
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
          className="query-search-date-calendar absolute left-0 top-[calc(100%+0.25rem)] z-40 w-[min(300px,calc(100vw-2rem))] overflow-hidden border border-gray-200 bg-white shadow-xl dark:border-gray-800 dark:bg-gray-950"
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
                className="flex h-6 w-6 items-center justify-center text-[var(--cds-icon-primary)] hover:bg-gray-100 dark:hover:bg-gray-900"
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
              <div className="flex items-center gap-4 text-sm text-[var(--cds-text-primary)]">
                <span>{MONTH_NAMES[calendarMonth.getMonth()]}</span>
                <span>{calendarMonth.getFullYear()}</span>
              </div>
              <button
                type="button"
                aria-label={labels.nextMonth}
                className="flex h-6 w-6 items-center justify-center text-[var(--cds-icon-primary)] hover:bg-gray-100 dark:hover:bg-gray-900"
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
                      'mx-auto flex h-7 w-7 items-center justify-center text-xs hover:bg-gray-100 dark:hover:bg-gray-900',
                      isSelected
                        ? 'bg-[var(--cds-layer-selected)] outline outline-2 -outline-offset-2 outline-[var(--cds-border-interactive)] text-[var(--cds-link-primary)]'
                        : 'text-[var(--cds-text-primary)]',
                    ].join(' ')}
                    onClick={() => setDraftDateValue(dateValue)}
                  >
                    {index + 1}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="query-search-date-time border-t border-gray-200 px-3 py-2 dark:border-gray-800">
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
    <span className="relative inline-flex w-fit shrink-0 items-center gap-2 border border-gray-200 bg-gray-50 px-2 py-1 font-mono text-sm dark:border-gray-800 dark:bg-gray-900">
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
          className="min-w-4 whitespace-nowrap bg-transparent text-left text-[var(--cds-link-primary)] outline-none hover:underline"
          onClick={onEditValue}
        >
          {displayValue}
        </button>
      )}
      <button
        type="button"
        aria-label={`Remove ${field.text} filter`}
        className="ml-1 flex h-4 w-4 cursor-pointer items-center justify-center text-gray-500 hover:text-gray-900 dark:hover:text-gray-100"
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
    if (!selectedField) return;
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
    if (!selectedField) return;
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
    if (!selectedField) return;
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
      className="grid w-full grid-cols-[minmax(0,1fr)_auto] gap-3 px-4 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-900"
      onClick={() => selectField(field)}
    >
      <span className="truncate font-mono">{field.text}</span>
      <span className="text-[var(--cds-link-primary)]">
        {field.logicLabel || field.type}
      </span>
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
          'flex h-12 min-w-0 items-center gap-2 border-0 bg-transparent px-3 text-sm text-[var(--cds-text-primary)] transition-colors hover:bg-[var(--cds-field)]',
          isOpen
            ? 'bg-[var(--cds-field)] outline outline-2 -outline-offset-2 outline-[var(--cds-focus)]'
            : '',
        ].join(' ')}
      >
        <Search className="h-4 w-4 shrink-0 text-gray-500" />
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
            className="min-w-[180px] flex-1 bg-transparent font-mono text-sm outline-none placeholder:font-sans placeholder:text-gray-500"
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
          className="absolute left-0 top-[calc(100%+0.125rem)] z-40 w-[min(760px,calc(100vw-2rem))] overflow-hidden border border-gray-200 bg-white shadow-xl dark:border-gray-800 dark:bg-gray-950"
          onMouseDown={event => event.preventDefault()}
        >
          {!isValueMode && (
            <div className="flex border-b border-gray-200 px-2 pt-2 dark:border-gray-800">
              {resolvedTabs.map(tab => (
                <button
                  key={tab.id}
                  type="button"
                  className={[
                    'px-3 py-2 text-sm',
                    activeTab === tab.id
                      ? 'border-b-2 border-[var(--cds-border-interactive)] text-[var(--cds-text-primary)]'
                      : 'text-gray-500 hover:text-gray-900 dark:hover:text-gray-200',
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
