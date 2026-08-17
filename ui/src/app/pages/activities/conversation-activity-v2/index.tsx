import { useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Button,
  CodeSnippet,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  Tag,
  Loading,
} from '@carbon/react';
import {
  Area,
  AreaChart,
  ResponsiveContainer,
  Tooltip as RechartsTooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { Activity, Close, Renew, WarningAlt } from '@carbon/icons-react';
import {
  ConnectionConfig,
  Criteria,
  GetAllTelemetry,
  GetAllTelemetryRequest,
  Ordering,
  Paginate,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/helmet';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { Pagination } from '@/app/components/carbon/pagination';
import { ScrollableTableSection } from '@/app/components/sections/table-section';
import { CopyButton } from '@/app/components/carbon/button/copy-button';
import { connectionConfig } from '@/configs';
import { useCurrentCredential } from '@/hooks/use-credential';
import {
  EventInspectorContent,
  EventMainTableSummary,
} from './event-renderers';
import { LogInspectorContent, LogMainTableSummary } from './log-renderers';
import { TraceQuerySearch } from './components/trace-query-search';
import {
  ALL_EVENT_OPTION,
  FilterOption,
  KIND_OPTIONS,
  LEVEL_OPTIONS,
  METRIC_NAME_OPTIONS,
  ROLE_OPTIONS,
  SCOPE_OPTIONS,
  getEventOptionsForComponent,
} from './constants';
import type { TimelineDocument } from './types';
import {
  formatDateTime,
  formatDurationMs,
  formatTime,
  createTraceFilter,
  dedupeTraceFilters,
  matchesTraceFilters,
  matchesTimelineSearch,
  parseTraceFilterQuery,
  telemetryRecordToTimelineDocument,
  getTraceFilterValues,
} from './utils';
import type { TraceFilterToken } from './utils';

type MetricValue = {
  description?: string;
  name?: string;
  value?: number | string;
};

type MetricChartPoint = {
  label: string;
  timestamp: number;
  value: number;
};

type TraceFilterState = {
  assistantIdInput: string;
  conversationIdInput: string;
  dateRange: [Date, Date] | null;
  messageIdInput: string;
  metricNameInput: string;
  searchText: string;
  selectedComponents: string[];
  selectedEvent: FilterOption;
  selectedKind: FilterOption;
  selectedLevel: FilterOption;
  selectedRole: FilterOption;
  selectedScope: FilterOption;
  traceIdInput: string;
};

const DEFAULT_TRACE_FILTERS: TraceFilterState = {
  assistantIdInput: '',
  conversationIdInput: '',
  dateRange: null,
  messageIdInput: '',
  metricNameInput: METRIC_NAME_OPTIONS[0].id,
  searchText: '',
  selectedComponents: [],
  selectedEvent: ALL_EVENT_OPTION,
  selectedKind: KIND_OPTIONS[0],
  selectedLevel: LEVEL_OPTIONS[0],
  selectedRole: ROLE_OPTIONS[0],
  selectedScope: SCOPE_OPTIONS[0],
  traceIdInput: '',
};

const getQueryValue = (
  searchParams: URLSearchParams,
  keys: string[],
): string => {
  for (const key of keys) {
    const value = searchParams.get(key)?.trim();
    if (value) return value;
  }
  return '';
};

const getTraceFiltersFromSearchParams = (
  searchParams: URLSearchParams,
): TraceFilterState => {
  const queryText = getQueryValue(searchParams, ['query']);

  return {
    ...DEFAULT_TRACE_FILTERS,
    searchText: queryText,
  };
};

const TRACE_LOAD_ERROR_MESSAGE = 'Unable to load trace records.';

const getTelemetryErrorMessage = (error: unknown): string => {
  if (error instanceof Error && error.message) return error.message;
  if (typeof error === 'string' && error.trim()) return error;
  return TRACE_LOAD_ERROR_MESSAGE;
};

const compactFilters = (
  filters: Array<TraceFilterToken | null>,
): TraceFilterToken[] =>
  filters.filter((filter): filter is TraceFilterToken => Boolean(filter));

const createRequestCriteria = (
  key: string,
  value: string | number | undefined,
  logic = '=',
): Criteria | null => {
  const normalizedValue = String(value ?? '').trim();
  if (!normalizedValue) return null;

  const criteria = new Criteria();
  criteria.setKey(key);
  criteria.setValue(normalizedValue);
  criteria.setLogic(logic);
  return criteria;
};

const getScopedInspectorCriteria = (
  document: TimelineDocument,
): Array<Criteria | null> => {
  const scope = String(document.scope || '').toLowerCase();

  if (scope === 'message') {
    return [
      createRequestCriteria('scope', document.scope),
      createRequestCriteria(
        'assistantConversationId',
        document.assistantConversationId,
      ) || createRequestCriteria('contextId', document.contextId),
    ];
  }

  if (scope === 'conversation') {
    return [
      createRequestCriteria('scope', document.scope),
      createRequestCriteria('assistantId', document.assistantId) ||
        createRequestCriteria(
          'assistantConversationId',
          document.assistantConversationId,
        ) ||
        createRequestCriteria('contextId', document.contextId),
    ];
  }

  if (scope === 'assistant') {
    return [
      createRequestCriteria('scope', document.scope),
      createRequestCriteria('assistantId', document.assistantId) ||
        createRequestCriteria('contextId', document.contextId),
    ];
  }

  return [
    createRequestCriteria(
      'assistantConversationId',
      document.assistantConversationId,
    ) ||
      createRequestCriteria('assistantId', document.assistantId) ||
      createRequestCriteria('contextId', document.contextId),
  ];
};

const getInspectorCriteria = (document: TimelineDocument): Criteria[] => {
  const criteria: Array<Criteria | null> = [];

  if (document.kind === 'metric') {
    criteria.push(
      createRequestCriteria('kind', 'metric'),
      createRequestCriteria('name', document.name),
      ...getScopedInspectorCriteria(document),
    );
  } else if (document.kind === 'event') {
    criteria.push(
      createRequestCriteria('kind', 'event'),
      createRequestCriteria('event', document.name),
      ...getScopedInspectorCriteria(document),
    );
  } else {
    criteria.push(
      createRequestCriteria(
        'assistantConversationId',
        document.assistantConversationId,
      ),
    );
  }

  if (document.kind === 'log' && !document.assistantConversationId) {
    criteria.push(createRequestCriteria('contextId', document.contextId));
  }

  return criteria.filter((item): item is Criteria => Boolean(item));
};

const getFacetTraceFilters = (filters: TraceFilterState): TraceFilterToken[] =>
  compactFilters([
    createTraceFilter('trace', filters.traceIdInput, 'facet'),
    filters.selectedKind.id !== KIND_OPTIONS[0].id
      ? createTraceFilter('kind', filters.selectedKind.id, 'facet')
      : null,
    filters.selectedKind.id === 'log' &&
    filters.selectedLevel.id !== LEVEL_OPTIONS[0].id
      ? createTraceFilter('level', filters.selectedLevel.id, 'facet')
      : null,
    filters.selectedKind.id === 'event' &&
    filters.selectedEvent.id !== ALL_EVENT_OPTION.id
      ? createTraceFilter('event', filters.selectedEvent.id, 'facet')
      : null,
    filters.selectedKind.id === 'event' && filters.selectedComponents.length > 0
      ? createTraceFilter('component', filters.selectedComponents[0], 'facet')
      : null,
    filters.selectedKind.id === 'metric' &&
    filters.metricNameInput &&
    filters.metricNameInput !== METRIC_NAME_OPTIONS[0].id
      ? createTraceFilter('metric', filters.metricNameInput, 'facet')
      : null,
    filters.selectedScope.id !== SCOPE_OPTIONS[0].id
      ? createTraceFilter('scope', filters.selectedScope.id, 'facet')
      : null,
    createTraceFilter('assistant', filters.assistantIdInput, 'facet'),
    createTraceFilter('conversation', filters.conversationIdInput, 'facet'),
    createTraceFilter('message', filters.messageIdInput, 'facet'),
    filters.selectedRole.id !== ROLE_OPTIONS[0].id
      ? createTraceFilter('role', filters.selectedRole.id, 'facet')
      : null,
  ]);

const getRequestTraceFilterSets = (
  filters: TraceFilterToken[],
): TraceFilterToken[][] =>
  filters.reduce<TraceFilterToken[][]>(
    (sets, filter) => {
      const filterValues = getTraceFilterValues(filter);
      if (filterValues.length <= 1) {
        return sets.map(set => [...set, filter]);
      }

      return sets.flatMap(set =>
        filterValues.map(filterValue => [
          ...set,
          { ...filter, value: filterValue },
        ]),
      );
    },
    [[]],
  );

const getRequestCriteriaSets = ({
  dateRange,
  freeText,
  filters,
}: {
  dateRange: TraceFilterState['dateRange'];
  filters: TraceFilterToken[];
  freeText: string;
}): Criteria[][] =>
  getRequestTraceFilterSets(filters).map(filterSet => {
    const criteria: Array<Criteria | null> = [
      createRequestCriteria('search', freeText, 'match'),
      ...filterSet.map(filter =>
        createRequestCriteria(filter.criteriaKey, filter.value, filter.logic),
      ),
    ];

    if (dateRange) {
      const endDate = new Date(dateRange[1]);
      endDate.setHours(23, 59, 59, 999);
      criteria.push(
        createRequestCriteria('timestamp', dateRange[0].toISOString(), '>='),
        createRequestCriteria('timestamp', endDate.toISOString(), '<='),
      );
    }

    return criteria.filter((item): item is Criteria => Boolean(item));
  });

const mergeTimelineDocuments = (documents: TimelineDocument[]) =>
  Array.from(
    documents
      .reduce((documentsById, document) => {
        documentsById.set(document.id, document);
        return documentsById;
      }, new Map<string, TimelineDocument>())
      .values(),
  ).sort(
    (left, right) =>
      new Date(right.occurredAt).getTime() -
      new Date(left.occurredAt).getTime(),
  );

const getMetricValues = (document: TimelineDocument): MetricValue[] =>
  (
    (document.data?.metrics as MetricValue[] | undefined)?.filter(Boolean) || [
      {
        description: document.data?.description as string | undefined,
        name: document.name,
      },
    ]
  ).filter(metric => metric.name || metric.value || metric.description);

const getMetricSummary = (document: TimelineDocument) =>
  getMetricValues(document)[0];

const getMetricNumericValue = (document: TimelineDocument): number | null => {
  const metricValue = getMetricSummary(document)?.value;
  const numericValue = Number(metricValue);
  return Number.isFinite(numericValue) ? numericValue : null;
};

const getMetricChartData = (records: TimelineDocument[]): MetricChartPoint[] =>
  records
    .map(record => {
      const value = getMetricNumericValue(record);
      const timestamp = new Date(record.occurredAt).getTime();

      if (value === null || !Number.isFinite(timestamp)) return null;

      return {
        label: formatTime(record.occurredAt),
        timestamp,
        value,
      };
    })
    .filter((point): point is MetricChartPoint => Boolean(point))
    .sort((a, b) => a.timestamp - b.timestamp);

const getRelatedRecords = (
  document: TimelineDocument,
  records: TimelineDocument[],
) => {
  const relatedRecords = records.filter(
    record => record.kind === document.kind,
  );
  return relatedRecords.length > 0 ? relatedRecords : [document];
};

const getLeftPanelTitle = (document: TimelineDocument) => {
  if (document.kind === 'log') return 'Logs';
  if (document.kind === 'metric') return 'Metrics';
  return 'Events';
};

const getRecordCountLabel = (document: TimelineDocument, count: number) => {
  if (document.kind === 'log') {
    return `${count} ${count === 1 ? 'log' : 'logs'}`;
  }
  if (document.kind === 'metric') {
    return `${count} ${count === 1 ? 'metric' : 'metrics'}`;
  }
  return `${count} ${count === 1 ? 'event' : 'events'}`;
};

const getTimeWindowLabel = (records: TimelineDocument[]): string => {
  const times = records
    .map(record => new Date(record.occurredAt).getTime())
    .filter(Number.isFinite)
    .sort((a, b) => a - b);

  if (times.length === 0) return 'No time window';

  const start = new Date(times[0]).toISOString();
  const end = new Date(times[times.length - 1]).toISOString();

  if (start === end) return formatTime(start);
  return `${formatTime(start)} - ${formatTime(end)}`;
};

const getEventInspectorScopeLabel = (document: TimelineDocument): string => {
  const scope = String(document.scope || '').toLowerCase();

  if (scope === 'message') return 'Conversation scope';
  if (scope === 'conversation') return 'Assistant scope';
  if (scope) return `${scope[0].toUpperCase()}${scope.slice(1)} scope`;
  return 'Unknown scope';
};

const LogRecordList = ({
  records,
  selectedDocumentId,
  onSelectDocument,
}: {
  records: TimelineDocument[];
  selectedDocumentId?: string;
  onSelectDocument: (document: TimelineDocument) => void;
}) => (
  <div className="min-h-0 flex-1 overflow-auto">
    {records.map(record => (
      <button
        key={record.id}
        type="button"
        className={[
          'w-full border-t border-gray-100 bg-white px-4 py-3 text-left first:border-t-0 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:hover:bg-gray-900',
          record.id === selectedDocumentId
            ? 'outline outline-2 -outline-offset-2 outline-[var(--cds-border-interactive)]'
            : '',
        ].join(' ')}
        onClick={() => onSelectDocument(record)}
      >
        <LogInspectorContent
          document={record}
          occurredAtLabel={formatTime(record.occurredAt)}
        />
      </button>
    ))}
  </div>
);

const EventRecordList = ({
  records,
  selectedDocumentId,
  onSelectDocument,
}: {
  records: TimelineDocument[];
  selectedDocumentId?: string;
  onSelectDocument: (document: TimelineDocument) => void;
}) => (
  <div className="min-h-0 flex-1 overflow-auto">
    {records.map(record => (
      <button
        key={record.id}
        type="button"
        className={[
          'w-full border-t border-gray-100 bg-white px-4 py-3 text-left first:border-t-0 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:hover:bg-gray-900',
          record.id === selectedDocumentId
            ? 'outline outline-2 -outline-offset-2 outline-[var(--cds-border-interactive)]'
            : '',
        ].join(' ')}
        onClick={() => onSelectDocument(record)}
      >
        <EventInspectorContent
          document={record}
          occurredAtLabel={formatTime(record.occurredAt)}
        />
      </button>
    ))}
  </div>
);

const MetricTrendChart = ({ records }: { records: TimelineDocument[] }) => {
  const chartData = getMetricChartData(records);

  if (chartData.length === 0) {
    return (
      <div className="flex h-full min-h-[180px] items-center justify-center px-4 text-sm text-gray-500 dark:text-gray-400">
        No numeric metric values recorded for this selection.
      </div>
    );
  }

  return (
    <div className="h-full min-h-[180px] py-3 [&_.recharts-surface:focus]:outline-none [&_.recharts-surface]:outline-none [&_.recharts-wrapper:focus]:outline-none [&_.recharts-wrapper]:outline-none">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={chartData}
          margin={{ top: 4, right: 8, left: 0, bottom: 0 }}
        >
          <defs>
            <linearGradient
              id="metricTrendGradient"
              x1="0"
              y1="0"
              x2="0"
              y2="1"
            >
              <stop
                offset="0%"
                stopColor="var(--cds-interactive, #0f62fe)"
                stopOpacity={0.28}
              />
              <stop
                offset="100%"
                stopColor="var(--cds-interactive, #0f62fe)"
                stopOpacity={0.02}
              />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="label"
            tick={{ fontSize: 11, fill: '#9ca3af' }}
            tickLine={false}
            axisLine={false}
            interval="preserveStartEnd"
            minTickGap={24}
          />
          <YAxis
            tick={{ fontSize: 11, fill: '#9ca3af' }}
            tickLine={false}
            axisLine={false}
            width={44}
          />
          <RechartsTooltip
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null;
              return (
                <div className="min-w-[140px] border border-gray-200 bg-white px-3 py-2 text-sm shadow-lg dark:border-gray-800 dark:bg-gray-900">
                  <p className="mb-1.5 text-xs text-gray-400">
                    {payload[0]?.payload?.label}
                  </p>
                  <div className="flex items-center gap-2">
                    <div
                      className="h-2 w-2"
                      style={{
                        backgroundColor: 'var(--cds-interactive, #0f62fe)',
                      }}
                    />
                    <span className="text-xs uppercase text-gray-600 dark:text-gray-300">
                      Value
                    </span>
                    <span className="ml-auto font-semibold tabular-nums">
                      {payload[0]?.value}
                    </span>
                  </div>
                </div>
              );
            }}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke="var(--cds-interactive, #0f62fe)"
            strokeWidth={1.5}
            fill="url(#metricTrendGradient)"
            dot={false}
            activeDot={{ r: 3 }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
};

const InspectorPrimaryPanel = ({
  document,
  records,
  selectedDocumentId,
  onSelectDocument,
  isLoading,
}: {
  document: TimelineDocument;
  isLoading?: boolean;
  records: TimelineDocument[];
  selectedDocumentId?: string;
  onSelectDocument: (document: TimelineDocument) => void;
}) => {
  const relatedRecords = getRelatedRecords(document, records);

  return (
    <div className="flex min-w-0 min-h-0 flex-col border-r border-gray-200 dark:border-gray-800">
      <div className="flex items-center justify-between border-b border-gray-200 bg-gray-50 px-4 py-2 dark:border-gray-800 dark:bg-gray-900">
        <div className="min-w-0">
          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
            {getLeftPanelTitle(document)}
          </p>
          {document.kind === 'event' && !isLoading && (
            <p className="mt-1 truncate font-mono text-xs text-gray-500">
              {getEventInspectorScopeLabel(document)} ·{' '}
              {getTimeWindowLabel(relatedRecords)}
            </p>
          )}
        </div>
        <Tag type="cool-gray">
          {isLoading
            ? 'Loading...'
            : getRecordCountLabel(document, relatedRecords.length)}
        </Tag>
      </div>
      {isLoading ? (
        <div className="flex min-h-0 flex-1 items-center justify-center">
          <Loading withOverlay={false} small />
        </div>
      ) : document.kind === 'log' ? (
        <LogRecordList
          records={relatedRecords}
          selectedDocumentId={selectedDocumentId}
          onSelectDocument={onSelectDocument}
        />
      ) : document.kind === 'metric' ? (
        <MetricTrendChart records={relatedRecords} />
      ) : (
        <EventRecordList
          records={relatedRecords}
          selectedDocumentId={selectedDocumentId}
          onSelectDocument={onSelectDocument}
        />
      )}
    </div>
  );
};

const TelemetryStreamTable = ({
  selectedDocumentId,
  records,
  onSelectRecord,
}: {
  selectedDocumentId: string | undefined;
  records: TimelineDocument[];
  onSelectRecord: (document: TimelineDocument) => void;
}) => (
  <ScrollableTableSection>
    <Table className="min-w-[1040px]">
      <TableHead>
        <TableRow>
          <TableHeader>ID</TableHeader>
          <TableHeader>traceID</TableHeader>
          <TableHeader>Kind</TableHeader>
          <TableHeader>Scope</TableHeader>
          <TableHeader>Summary</TableHeader>
          <TableHeader>Occurred at</TableHeader>
        </TableRow>
      </TableHead>
      <TableBody>
        {records.map(document => (
          <TableRow
            key={document.id}
            className={[
              'cursor-pointer',
              document.id === selectedDocumentId
                ? 'outline outline-2 -outline-offset-2 outline-[var(--cds-border-interactive)]'
                : '',
            ].join(' ')}
            tabIndex={0}
            onClick={() => onSelectRecord(document)}
            onKeyDown={event => {
              if (event.key === 'Enter') onSelectRecord(document);
            }}
          >
            <TableCell className="max-w-[220px] truncate font-mono text-sm text-blue-600">
              {document.id}
            </TableCell>
            <TableCell className="max-w-[260px]">
              <div className="flex min-w-0 items-center gap-1">
                <span className="truncate font-mono text-[13px]">
                  {document.traceId || '-'}
                </span>
                {document.traceId && (
                  <span
                    className="shrink-0"
                    onClick={event => event.stopPropagation()}
                  >
                    <CopyButton className="h-6 w-6">
                      {document.traceId}
                    </CopyButton>
                  </span>
                )}
              </div>
            </TableCell>
            <TableCell>
              <Tag type="cool-gray">{document.kind}</Tag>
            </TableCell>
            <TableCell>
              <Tag type="blue">{document.scope}</Tag>
            </TableCell>
            <TableCell>
              <div className="max-w-[520px]">
                {document.kind === 'metric' ? (
                  <p className="truncate font-mono text-[13px]">
                    [{document.name}] {getMetricSummary(document)?.value}
                    <span className="text-gray-500">
                      {' '}
                      {getMetricSummary(document)?.description}
                    </span>
                  </p>
                ) : (
                  <>
                    {document.kind === 'log' ? (
                      <LogMainTableSummary document={document} />
                    ) : (
                      <EventMainTableSummary document={document} />
                    )}
                  </>
                )}
              </div>
            </TableCell>
            <TableCell className="text-[13px]! whitespace-nowrap">
              {formatDateTime(document.occurredAt)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  </ScrollableTableSection>
);

const TraceInspectorPanel = ({
  document,
  isLoading,
  records,
  recordCount,
  selectedContextId,
  selectedDocumentId,
  onSelectDocument,
  onClose,
}: {
  document: TimelineDocument | null;
  isLoading?: boolean;
  records: TimelineDocument[];
  recordCount: number;
  selectedContextId: string;
  selectedDocumentId?: string;
  onSelectDocument: (document: TimelineDocument) => void;
  onClose: () => void;
}) => {
  if (!document) return null;

  return (
    <aside className="flex max-h-[48vh] min-h-[320px] shrink-0 border-t border-gray-200 bg-white shadow-xl dark:border-gray-800 dark:bg-gray-950">
      <section className="relative flex min-h-0 flex-1 flex-col">
        <Button
          hasIconOnly
          kind="ghost"
          size="lg"
          renderIcon={Close}
          iconDescription="Close"
          tooltipPosition="left"
          className="!absolute !right-0 !top-0 !z-10 !h-12 !min-h-12 !w-12 !min-w-12 !p-0 border-l border-gray-200 dark:border-gray-800"
          onClick={onClose}
        />
        <div className="border-b border-gray-200 px-4 py-3 pr-14 dark:border-gray-800">
          <div className="min-w-0">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <Tag type={document.outcome === 'failure' ? 'red' : 'green'}>
                {document.outcome || 'unknown'}
              </Tag>
              <Tag type="cool-gray">{document.kind}</Tag>
              <Tag type="blue">{document.scope}</Tag>
            </div>
            <h2 className="truncate text-base font-medium text-gray-900 dark:text-gray-100">
              {document.title || document.name}
            </h2>
            <p className="mt-1 font-mono text-xs text-gray-500">
              contextId:{selectedContextId || '-'} · {recordCount} records
            </p>
          </div>
        </div>
        <div className="grid min-h-0 flex-1 overflow-hidden md:grid-cols-[minmax(0,1.3fr)_minmax(420px,0.7fr)]">
          <InspectorPrimaryPanel
            document={document}
            isLoading={isLoading}
            records={records}
            selectedDocumentId={selectedDocumentId}
            onSelectDocument={onSelectDocument}
          />
          <div className="min-h-0 overflow-auto p-4">
            <div className="mb-4 grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-xs uppercase text-gray-500">Occurred</p>
                <p className="font-mono text-xs">
                  {formatTime(document.occurredAt)}
                </p>
              </div>
              <div>
                <p className="text-xs uppercase text-gray-500">Duration</p>
                <p className="font-mono text-xs">
                  {formatDurationMs(document.durationMs)}
                </p>
              </div>
              <div>
                <p className="text-xs uppercase text-gray-500">Category</p>
                <p className="font-mono text-xs">{document.category || '-'}</p>
              </div>
              <div>
                <p className="text-xs uppercase text-gray-500">Role</p>
                <p className="font-mono text-xs">
                  {document.messageRole || '-'}
                </p>
              </div>
              <div className="col-span-2">
                <p className="text-xs uppercase text-gray-500">Record ID</p>
                <p className="break-all font-mono text-xs">{document.id}</p>
              </div>
              <div className="col-span-2">
                <p className="text-xs uppercase text-gray-500">traceID</p>
                <div className="flex min-w-0 items-center gap-1">
                  <p className="break-all font-mono text-xs">
                    {document.traceId || '-'}
                  </p>
                  {document.traceId && (
                    <CopyButton className="h-6 w-6 shrink-0">
                      {document.traceId}
                    </CopyButton>
                  )}
                </div>
              </div>
              <div>
                <p className="text-xs uppercase text-gray-500">Context</p>
                <p className="break-all font-mono text-xs">
                  {document.contextId || '-'}
                </p>
              </div>
              <div>
                <p className="text-xs uppercase text-gray-500">Message</p>
                <p className="break-all font-mono text-xs">
                  {document.messageId || '-'}
                </p>
              </div>
            </div>
            <div className="space-y-4">
              <div className="min-w-0">
                <p className="mb-2 text-xs font-medium uppercase text-gray-500">
                  Attributes
                </p>
                <CodeSnippet type="multi" feedback="Copied">
                  {JSON.stringify(document.attributes || {}, null, 2)}
                </CodeSnippet>
              </div>
              <div className="min-w-0">
                <p className="mb-2 text-xs font-medium uppercase text-gray-500">
                  Data
                </p>
                <CodeSnippet type="multi" feedback="Copied">
                  {JSON.stringify(document.data || {}, null, 2)}
                </CodeSnippet>
              </div>
            </div>
          </div>
        </div>
      </section>
    </aside>
  );
};

export const ListingPage = () => {
  const { token, authId, projectId } = useCurrentCredential();
  const [searchParams] = useSearchParams();
  const searchParamsKey = searchParams.toString();
  const queryFilters = useMemo(
    () => getTraceFiltersFromSearchParams(new URLSearchParams(searchParamsKey)),
    [searchParamsKey],
  );
  const lastSearchParamsKey = useRef(searchParamsKey);
  const [searchText, setSearchText] = useState(queryFilters.searchText);
  const [selectedKind, setSelectedKind] = useState(queryFilters.selectedKind);
  const [selectedLevel, setSelectedLevel] = useState(
    queryFilters.selectedLevel,
  );
  const [selectedScope, setSelectedScope] = useState(
    queryFilters.selectedScope,
  );
  const [selectedRole, setSelectedRole] = useState(queryFilters.selectedRole);
  const [selectedComponents, setSelectedComponents] = useState<string[]>([]);
  const [selectedEvent, setSelectedEvent] = useState(
    queryFilters.selectedEvent,
  );
  const [metricNameInput, setMetricNameInput] = useState(
    queryFilters.metricNameInput,
  );
  const [assistantIdInput, setAssistantIdInput] = useState(
    queryFilters.assistantIdInput,
  );
  const [conversationIdInput, setConversationIdInput] = useState(
    queryFilters.conversationIdInput,
  );
  const [messageIdInput, setMessageIdInput] = useState(
    queryFilters.messageIdInput,
  );
  const [traceIdInput, setTraceIdInput] = useState(queryFilters.traceIdInput);
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(
    queryFilters.dateRange,
  );
  const [appliedFilters, setAppliedFilters] =
    useState<TraceFilterState>(queryFilters);
  const [documents, setDocuments] = useState<TimelineDocument[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalItem, setTotalItem] = useState(0);
  const [refreshKey, setRefreshKey] = useState(0);
  const [selectedContextId, setSelectedContextId] = useState('');
  const [selectedDocument, setSelectedDocument] =
    useState<TimelineDocument | null>(null);
  const [inspectorDocuments, setInspectorDocuments] = useState<
    TimelineDocument[]
  >([]);
  const [isInspectorLoading, setIsInspectorLoading] = useState(false);

  const selectedComponentId = selectedComponents[0] || 'all';

  const eventFilterOptions = useMemo(
    () => getEventOptionsForComponent(selectedComponentId),
    [selectedComponentId],
  );

  const currentFilters = useMemo<TraceFilterState>(
    () => ({
      assistantIdInput,
      conversationIdInput,
      dateRange,
      messageIdInput,
      metricNameInput,
      searchText,
      selectedComponents,
      selectedEvent,
      selectedKind,
      selectedLevel,
      selectedRole,
      selectedScope,
      traceIdInput,
    }),
    [
      assistantIdInput,
      conversationIdInput,
      dateRange,
      messageIdInput,
      metricNameInput,
      searchText,
      selectedComponents,
      selectedEvent,
      selectedKind,
      selectedLevel,
      selectedRole,
      selectedScope,
      traceIdInput,
    ],
  );

  const appliedQuery = useMemo(
    () => parseTraceFilterQuery(appliedFilters.searchText),
    [appliedFilters.searchText],
  );

  const appliedTraceFilters = useMemo(
    () =>
      dedupeTraceFilters([
        ...appliedQuery.filters,
        ...getFacetTraceFilters(appliedFilters),
      ]),
    [appliedFilters, appliedQuery.filters],
  );

  const setFilterState = (nextFilters: TraceFilterState) => {
    setSearchText(nextFilters.searchText);
    setSelectedKind(nextFilters.selectedKind);
    setSelectedLevel(nextFilters.selectedLevel);
    setSelectedScope(nextFilters.selectedScope);
    setSelectedRole(nextFilters.selectedRole);
    setSelectedComponents(nextFilters.selectedComponents);
    setSelectedEvent(nextFilters.selectedEvent);
    setMetricNameInput(nextFilters.metricNameInput);
    setAssistantIdInput(nextFilters.assistantIdInput);
    setConversationIdInput(nextFilters.conversationIdInput);
    setMessageIdInput(nextFilters.messageIdInput);
    setTraceIdInput(nextFilters.traceIdInput);
    setDateRange(nextFilters.dateRange);
  };

  const applyFilterState = (nextFilters: TraceFilterState) => {
    setFilterState(nextFilters);
    setAppliedFilters(nextFilters);
    setPage(1);
    setRefreshKey(key => key + 1);
  };

  const applyQuerySearch = (nextSearchText: string) => {
    applyFilterState({ ...currentFilters, searchText: nextSearchText });
  };

  useEffect(() => {
    if (lastSearchParamsKey.current === searchParamsKey) return;
    lastSearchParamsKey.current = searchParamsKey;

    setSearchText(queryFilters.searchText);
    setSelectedKind(queryFilters.selectedKind);
    setSelectedLevel(queryFilters.selectedLevel);
    setSelectedScope(queryFilters.selectedScope);
    setSelectedRole(queryFilters.selectedRole);
    setSelectedComponents(queryFilters.selectedComponents);
    setSelectedEvent(queryFilters.selectedEvent);
    setMetricNameInput(queryFilters.metricNameInput);
    setAssistantIdInput(queryFilters.assistantIdInput);
    setConversationIdInput(queryFilters.conversationIdInput);
    setMessageIdInput(queryFilters.messageIdInput);
    setTraceIdInput(queryFilters.traceIdInput);
    setDateRange(queryFilters.dateRange);
    setAppliedFilters(queryFilters);
    setPage(1);
    setRefreshKey(key => key + 1);
  }, [queryFilters, searchParamsKey]);

  const requestCriteriaSets = useMemo(
    () =>
      getRequestCriteriaSets({
        dateRange: appliedFilters.dateRange,
        filters: appliedTraceFilters,
        freeText: appliedQuery.freeText,
      }),
    [appliedFilters.dateRange, appliedQuery.freeText, appliedTraceFilters],
  );

  useEffect(() => {
    if (selectedKind.id !== 'log') setSelectedLevel(LEVEL_OPTIONS[0]);
    if (selectedKind.id !== 'event') {
      setSelectedComponents([]);
      setSelectedEvent(ALL_EVENT_OPTION);
    }
    if (selectedKind.id !== 'metric') {
      setMetricNameInput(METRIC_NAME_OPTIONS[0].id);
    }
  }, [selectedKind]);

  useEffect(() => {
    let active = true;

    const fetchTelemetry = async () => {
      setIsLoading(true);
      const shouldMergeRequests = requestCriteriaSets.length > 1;
      const requestPage = shouldMergeRequests ? 1 : page;
      const requestPageSize = shouldMergeRequests ? page * pageSize : pageSize;

      const createTelemetryRequest = (criteria: Criteria[]) => {
        const request = new GetAllTelemetryRequest();
        const paginate = new Paginate();
        paginate.setPage(requestPage);
        paginate.setPagesize(requestPageSize);
        request.setPaginate(paginate);
        request.setCriteriasList(criteria);

        const order = new Ordering();
        order.setColumn('occurredAt');
        order.setOrder('desc');
        request.setOrder(order);
        return request;
      };

      try {
        const responses = await Promise.all(
          requestCriteriaSets.map(criteria =>
            GetAllTelemetry(
              connectionConfig,
              createTelemetryRequest(criteria),
              ConnectionConfig.WithDebugger({
                authorization: token,
                userId: authId,
                projectId,
              }),
            ),
          ),
        );
        if (!active) return;

        const failedResponse = responses.find(
          response => !response.getSuccess(),
        );
        if (failedResponse) {
          const message =
            failedResponse.getError()?.getHumanmessage() ||
            TRACE_LOAD_ERROR_MESSAGE;
          toast.error(message);
          return;
        }

        const mergedDocuments = mergeTimelineDocuments(
          responses.flatMap(
            response =>
              response
                .getDataList()
                .map(telemetryRecordToTimelineDocument)
                .filter(Boolean) as TimelineDocument[],
          ),
        );
        const nextDocuments = shouldMergeRequests
          ? mergedDocuments.slice((page - 1) * pageSize, page * pageSize)
          : mergedDocuments;

        setDocuments(nextDocuments);
        setTotalItem(
          shouldMergeRequests
            ? responses.reduce(
                (total, response) =>
                  total +
                  (response.getPaginated()?.getTotalitem() ||
                    response.getDataList().length),
                0,
              )
            : responses[0]?.getPaginated()?.getTotalitem() ||
                nextDocuments.length,
        );
      } catch (error) {
        if (!active) return;
        const message = getTelemetryErrorMessage(error);
        toast.error(message);
      } finally {
        if (active) setIsLoading(false);
      }
    };

    fetchTelemetry();

    return () => {
      active = false;
    };
  }, [
    authId,
    page,
    pageSize,
    projectId,
    refreshKey,
    requestCriteriaSets,
    token,
  ]);

  const filteredDocuments = useMemo(
    () =>
      documents.filter(document => {
        const occurredMs = new Date(document.occurredAt).getTime();
        const matchesDate =
          !appliedFilters.dateRange ||
          (occurredMs >= appliedFilters.dateRange[0].getTime() &&
            occurredMs <=
              appliedFilters.dateRange[1].getTime() + 24 * 60 * 60 * 1000);

        return (
          matchesTimelineSearch(document, appliedQuery.freeText) &&
          matchesDate &&
          matchesTraceFilters(document, appliedTraceFilters)
        );
      }),
    [
      appliedFilters.dateRange,
      appliedQuery.freeText,
      appliedTraceFilters,
      documents,
    ],
  );

  const selectedTimelineDocuments = useMemo(
    () =>
      filteredDocuments.filter(
        document => document.contextId === selectedContextId,
      ),
    [filteredDocuments, selectedContextId],
  );

  const visibleInspectorDocuments =
    inspectorDocuments.length > 0
      ? inspectorDocuments
      : selectedTimelineDocuments;

  useEffect(() => {
    const currentContextExists = filteredDocuments.some(
      document => document.contextId === selectedContextId,
    );

    if (currentContextExists) return;

    const nextDocument = filteredDocuments[0] || null;
    setSelectedContextId(nextDocument?.contextId || '');
    setSelectedDocument(null);
  }, [filteredDocuments, selectedContextId]);

  useEffect(() => {
    if (
      selectedEvent.id !== ALL_EVENT_OPTION.id &&
      !eventFilterOptions.some(option => option.id === selectedEvent.id)
    ) {
      setSelectedEvent(ALL_EVENT_OPTION);
    }
  }, [eventFilterOptions, selectedEvent]);

  useEffect(() => {
    if (!selectedDocument) {
      setInspectorDocuments([]);
      setIsInspectorLoading(false);
      return;
    }

    let active = true;

    const fetchInspectorDocuments = async () => {
      setInspectorDocuments([]);
      setIsInspectorLoading(true);

      const request = new GetAllTelemetryRequest();
      const paginate = new Paginate();
      paginate.setPage(1);
      paginate.setPagesize(500);
      request.setPaginate(paginate);
      request.setCriteriasList(getInspectorCriteria(selectedDocument));

      const order = new Ordering();
      order.setColumn('occurredAt');
      order.setOrder('asc');
      request.setOrder(order);

      try {
        const response = await GetAllTelemetry(
          connectionConfig,
          request,
          ConnectionConfig.WithDebugger({
            authorization: token,
            userId: authId,
            projectId,
          }),
        );
        if (!active) return;

        if (!response.getSuccess()) {
          const message =
            response.getError()?.getHumanmessage() || TRACE_LOAD_ERROR_MESSAGE;
          toast.error(message);
          return;
        }

        const nextDocuments = response
          .getDataList()
          .map(telemetryRecordToTimelineDocument)
          .filter(Boolean) as TimelineDocument[];

        setInspectorDocuments(nextDocuments);
      } catch (error) {
        if (!active) return;
        toast.error(getTelemetryErrorMessage(error));
      } finally {
        if (active) setIsInspectorLoading(false);
      }
    };

    fetchInspectorDocuments();

    return () => {
      active = false;
    };
  }, [authId, projectId, selectedDocument, token]);

  const selectRecord = (document: TimelineDocument) => {
    setInspectorDocuments([]);
    setIsInspectorLoading(true);
    setSelectedContextId(document.contextId);
    setSelectedDocument(document);
  };

  return (
    <div className="flex h-full overflow-hidden">
      <Helmet title="Trace" />

      <div className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
        <TableToolbar>
          <TableToolbarContent>
            <TraceQuerySearch
              value={searchText}
              onChange={setSearchText}
              onApply={applyQuerySearch}
            />
            <Button
              hasIconOnly
              kind="ghost"
              renderIcon={Renew}
              iconDescription="Reload"
              tooltipPosition="bottom"
              onClick={() => setRefreshKey(key => key + 1)}
            />
          </TableToolbarContent>
        </TableToolbar>

        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex min-h-0 min-w-0 flex-1">
            {isLoading ? (
              <div className="flex flex-1 items-center justify-center">
                <Loading withOverlay={false} small />
              </div>
            ) : filteredDocuments.length === 0 ? (
              <EmptyState
                icon={appliedFilters.searchText ? WarningAlt : Activity}
                title="No traces found"
                subtitle="Adjust search, scope, component, event, or date filters."
              />
            ) : (
              <div className="flex min-w-0 flex-1 flex-col">
                <TelemetryStreamTable
                  selectedDocumentId={selectedDocument?.id}
                  records={filteredDocuments}
                  onSelectRecord={selectRecord}
                />

                <Pagination
                  className="shrink-0 border-t border-gray-200 dark:border-gray-800"
                  totalItems={totalItem}
                  page={page}
                  pageSize={pageSize}
                  pageSizes={[25, 50, 100]}
                  onChange={({ page: nextPage, pageSize: nextPageSize }) => {
                    if (nextPageSize !== pageSize) {
                      setPageSize(nextPageSize);
                      setPage(1);
                      return;
                    }
                    setPage(nextPage);
                  }}
                />
              </div>
            )}
          </div>
          <TraceInspectorPanel
            document={selectedDocument}
            isLoading={isInspectorLoading}
            records={visibleInspectorDocuments}
            recordCount={visibleInspectorDocuments.length}
            selectedContextId={selectedContextId}
            selectedDocumentId={selectedDocument?.id}
            onSelectDocument={setSelectedDocument}
            onClose={() => setSelectedDocument(null)}
          />
        </div>
      </div>
    </div>
  );
};
