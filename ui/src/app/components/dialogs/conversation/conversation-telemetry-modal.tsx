import React, { useEffect, useState } from 'react';
import { ModalProps } from '@/app/components/ui/primitives';
import { useCurrentCredential } from '@/hooks/use-credential';
import { Modal, ModalHeader, ModalBody } from '@/app/components/ui/primitives';
import { Pagination } from '@/app/components/ui/primitives';
import { Tabs } from '@/app/components/ui/primitives';
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableExpandedRow,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Tag,
  DismissibleTag,
  Loading,
  CodeSnippet,
  Dropdown,
  MultiSelect,
} from '@carbon/react';
import { TableToolbarFilter } from '@/app/components/ui/composites/table-toolbar-filter';
import { ChevronRight } from '@carbon/icons-react';
import { TextInput } from '@/app/components/ui/primitives';
import {
  buildLatencySeries,
  buildTelemetryCriteriaInputs,
  EVENT_NAME_OPTIONS,
  formatDateTime,
  getTelemetryContextId,
  getTelemetryConversationId,
  getTelemetryRowData,
  getTelemetrySearchDocument,
  getTelemetryScopeAttributes,
  matchesTelemetryFilters,
  METRIC_SCOPE_OPTIONS,
  splitStructuredTelemetryCriteria,
} from './conversation-telemetry-utils';
import type {
  SelectOption,
  TelemetryRow,
} from './conversation-telemetry-utils';
import { LatencyStackChart } from './conversation-telemetry-latency-stack-chart';
import { listTelemetry } from '@/clients';

export {
  buildLatencySeries,
  buildTelemetryCriteriaInputs,
  matchesTelemetryFilters,
  splitStructuredTelemetryCriteria,
} from './conversation-telemetry-utils';

interface ConversationTelemetryDialogProps extends ModalProps {
  assistantId: string;
  criterias?: Array<{
    getKey: () => string;
    getValue: () => string;
  }>;
}

interface Chip {
  field: string;
  value: string | number;
  id: string;
}

const getTelemetryTime = (record: {
  getOccurredat?: () => { toDate?: () => Date } | undefined;
  getTime?: () => { toDate?: () => Date } | undefined;
}) =>
  record.getOccurredat?.()?.toDate?.() ??
  record.getTime?.()?.toDate?.() ??
  new Date(0);

export function ConversationTelemetryDialog(
  props: ConversationTelemetryDialogProps,
) {
  const { assistantId, criterias, modalOpen, setModalOpen } = props;
  const { token, authId, projectId } = useCurrentCredential();
  const [chips, setChips] = useState<Chip[]>([]);
  const [rows, setRows] = useState<TelemetryRow[]>([]);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [selectedTab, setSelectedTab] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalItem, setTotalItem] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [criteriaReady, setCriteriaReady] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [conversationIdInput, setConversationIdInput] = useState('');
  const [messageIdInput, setMessageIdInput] = useState('');
  const [eventNameInputs, setEventNameInputs] = useState<string[]>([]);
  const [eventDataTypeInput, setEventDataTypeInput] = useState('');
  const [metricScopeInput, setMetricScopeInput] = useState('');
  const [appliedConversationId, setAppliedConversationId] = useState('');
  const [appliedMessageId, setAppliedMessageId] = useState('');
  const [appliedEventNames, setAppliedEventNames] = useState<string[]>([]);
  const [appliedEventDataType, setAppliedEventDataType] = useState('');
  const [appliedMetricScope, setAppliedMetricScope] = useState('');
  const [structuredError, setStructuredError] = useState('');
  const activeTabKind: 'event' | 'metric' | 'latency' =
    selectedTab === 0 ? 'event' : selectedTab === 1 ? 'metric' : 'latency';
  const hasSearchQuery = searchText.trim() !== '';
  const hasLocalFilters =
    activeTabKind === 'event'
      ? appliedEventNames.length > 0 || appliedEventDataType !== ''
      : activeTabKind === 'metric'
        ? appliedMetricScope !== ''
        : false;
  const shouldFetchAllRows =
    activeTabKind === 'latency' || hasSearchQuery || hasLocalFilters;
  const requestPage = shouldFetchAllRows ? 1 : page;
  const requestPageSize = shouldFetchAllRows ? 100 : pageSize;

  useEffect(() => {
    if (!modalOpen) return;
    const normalized = splitStructuredTelemetryCriteria(
      (criterias || []).map(criteria => ({
        key: criteria.getKey(),
        value: criteria.getValue(),
      })),
    );
    const initialChips = normalized.remaining.map((criteria, index) => ({
      field: criteria.key,
      value: criteria.value,
      id: `${Date.now()}-${index}`,
    }));
    setRows([]);
    setExpandedRows(new Set());
    setTotalItem(0);
    setSelectedTab(0);
    setPage(1);
    setChips(initialChips);
    setSearchText('');
    setConversationIdInput(normalized.conversationId);
    setMessageIdInput(normalized.messageId);
    setEventNameInputs([]);
    setEventDataTypeInput('');
    setMetricScopeInput('');
    setAppliedConversationId(normalized.conversationId);
    setAppliedMessageId(normalized.messageId);
    setAppliedEventNames([]);
    setAppliedEventDataType('');
    setAppliedMetricScope('');
    setStructuredError('');
    setCriteriaReady(true);
  }, [criterias, modalOpen]);

  useEffect(() => {
    if (!modalOpen || !criteriaReady) return;
    let active = true;
    setIsLoading(true);
    setRows([]);
    setExpandedRows(new Set());

    const criteriaList = buildTelemetryCriteriaInputs(
      chips.map(chip => ({ key: chip.field, value: String(chip.value) })),
      appliedConversationId,
      appliedMessageId,
    ).map(c => ({ key: c.key, value: c.value, logic: 'match' }));

    const getCriteria = () => [
      { key: 'assistant_id', value: assistantId, logic: 'match' },
      ...criteriaList,
    ];

    const toTelemetryRows = (
      response: Awaited<ReturnType<typeof listTelemetry>>,
      pageOffset: number,
    ): TelemetryRow[] => {
      const merged: TelemetryRow[] = [];
      response.getDataList().forEach((record, index) => {
        const event = record.getEvent();
        const metric = record.getMetric();
        if (event) {
          merged.push({
            kind: 'event',
            ts: getTelemetryTime(event),
            key: `e-${pageOffset + index}`,
            record: event,
          });
        } else if (metric) {
          merged.push({
            kind: 'metric',
            ts: getTelemetryTime(metric),
            key: `m-${pageOffset + index}`,
            record: metric,
          });
        }
      });
      return merged;
    };

    const fetchTelemetry = async () => {
      try {
        const firstResponse = await listTelemetry({
          page: requestPage,
          pageSize: requestPageSize,
          criteria: getCriteria(),
          auth: { token, userId: authId, projectId },
        });
        if (!active) return;

        const total = firstResponse.getPaginated()?.getTotalitem() ?? 0;
        const mergedRows = toTelemetryRows(firstResponse, 0);

        if (shouldFetchAllRows && total > requestPageSize) {
          const totalPages = Math.ceil(total / requestPageSize);
          for (let nextPage = 2; nextPage <= totalPages; nextPage += 1) {
            const response = await listTelemetry({
              page: nextPage,
              pageSize: requestPageSize,
              criteria: getCriteria(),
              auth: { token, userId: authId, projectId },
            });
            if (!active) return;
            mergedRows.push(
              ...toTelemetryRows(response, (nextPage - 1) * requestPageSize),
            );
          }
        }

        mergedRows.sort((a, b) => a.ts.getTime() - b.ts.getTime());
        setRows(mergedRows);
        setTotalItem(total);
      } catch {
        if (!active) return;
        setRows([]);
        setTotalItem(0);
      } finally {
        if (!active) return;
        setIsLoading(false);
      }
    };

    fetchTelemetry();

    return () => {
      active = false;
    };
  }, [
    token,
    authId,
    projectId,
    assistantId,
    modalOpen,
    JSON.stringify(chips),
    appliedConversationId,
    appliedMessageId,
    requestPageSize,
    requestPage,
    criteriaReady,
    shouldFetchAllRows,
  ]);

  const toggleRow = (key: string) => {
    setExpandedRows(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const removeChip = (chipId: string) => {
    setChips(prev => prev.filter(c => c.id !== chipId));
    setPage(1);
  };

  const applyStructuredCriteria = (): boolean => {
    const nextConversationId = conversationIdInput.trim();
    const nextMessageId = messageIdInput.trim();
    if (nextConversationId && !/^\d+$/.test(nextConversationId)) {
      setStructuredError('Conversation ID must be numeric.');
      return false;
    }
    setStructuredError('');
    setAppliedConversationId(nextConversationId);
    setAppliedMessageId(nextMessageId);
    setPage(1);
    return true;
  };

  const resetStructuredCriteria = () => {
    setStructuredError('');
    setConversationIdInput('');
    setMessageIdInput('');
    setAppliedConversationId('');
    setAppliedMessageId('');
    setPage(1);
  };

  const applyEventFilters = (): boolean => {
    if (!applyStructuredCriteria()) {
      return false;
    }
    setAppliedEventNames(eventNameInputs);
    setAppliedEventDataType(eventDataTypeInput.trim());
    setPage(1);
    return true;
  };

  const resetEventFilters = () => {
    setEventNameInputs([]);
    setEventDataTypeInput('');
    setAppliedEventNames([]);
    setAppliedEventDataType('');
    resetStructuredCriteria();
  };

  const applyMetricFilters = (): boolean => {
    setAppliedMetricScope(metricScopeInput);
    setPage(1);
    return true;
  };

  const resetMetricFilters = () => {
    setMetricScopeInput('');
    setAppliedMetricScope('');
    setPage(1);
  };

  const getFilteredRows = (kind: 'event' | 'metric') =>
    rows.filter(row => {
      const { typeLabel, json } = getTelemetryRowData(row);
      if (row.kind !== kind) {
        return false;
      }
      return matchesTelemetryFilters(
        getTelemetrySearchDocument(row, typeLabel, json),
        {
          searchText,
          names: kind === 'event' ? appliedEventNames : [],
          messageOrContextId: kind === 'event' ? appliedMessageId : '',
          eventDataType: kind === 'event' ? appliedEventDataType : '',
          metricScope: kind === 'metric' ? appliedMetricScope : '',
        },
      );
    });

  const filteredRows =
    activeTabKind === 'latency' ? [] : getFilteredRows(activeTabKind);
  const latencySeries = buildLatencySeries(
    rows
      .filter(
        (row): row is Extract<TelemetryRow, { kind: 'metric' }> =>
          row.kind === 'metric',
      )
      .map(row => {
        const scopeAttributes = getTelemetryScopeAttributes(row.record);
        return {
          timestampMs: row.ts.getTime(),
          contextId: getTelemetryContextId(scopeAttributes),
          conversationId: getTelemetryConversationId(scopeAttributes),
          metrics: [
            { name: row.record.getName(), value: row.record.getValue() },
          ],
        };
      }),
  );

  useEffect(() => {
    if (!shouldFetchAllRows) return;
    const maxPage = Math.max(1, Math.ceil(filteredRows.length / pageSize));
    if (page > maxPage) {
      setPage(maxPage);
    }
  }, [filteredRows.length, page, pageSize, shouldFetchAllRows]);

  useEffect(() => {
    setExpandedRows(new Set());
    setPage(1);
  }, [selectedTab]);

  const totalItems = shouldFetchAllRows ? filteredRows.length : totalItem;
  const renderTelemetryTable = (kind: 'event' | 'metric') => {
    const isActiveTab = activeTabKind === kind;
    const isEventTab = kind === 'event';
    const tabTitle = isEventTab ? 'Events' : 'Metrics';
    const tabRows = isActiveTab ? filteredRows : getFilteredRows(kind);
    const tabVisibleRows =
      isActiveTab && shouldFetchAllRows
        ? tabRows.slice((page - 1) * pageSize, page * pageSize)
        : tabRows;
    const tabTotalItems = isActiveTab
      ? totalItems
      : shouldFetchAllRows
        ? tabRows.length
        : tabRows.length;
    const selectedEventNameItems = EVENT_NAME_OPTIONS.filter(item =>
      eventNameInputs.includes(item.id),
    );
    const selectedMetricScope = METRIC_SCOPE_OPTIONS.find(
      item => item.id === metricScopeInput,
    );
    const itemToLabel = (item: SelectOption | null) => item?.label || '';

    return (
      <>
        <TableToolbar>
          <TableToolbarContent>
            <TableToolbarSearch
              placeholder={`Search ${tabTitle.toLowerCase()} payload or text`}
              value={searchText}
              onChange={(_, value) => {
                setSearchText(value || '');
                setPage(1);
              }}
            />
            <TableToolbarFilter
              panelClassName="!w-[48rem] max-w-[calc(100vw-4rem)]"
              filters={[]}
              activeFilters={new Set()}
              onApplyFilter={() => {}}
              onResetFilter={() => {}}
              onApply={isEventTab ? applyEventFilters : applyMetricFilters}
              onReset={isEventTab ? resetEventFilters : resetMetricFilters}
              extraContent={
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  {isEventTab ? (
                    <>
                      <MultiSelect<SelectOption>
                        id="telemetry-filter-event-name"
                        titleText="Name"
                        label="Choose names"
                        items={EVENT_NAME_OPTIONS}
                        itemToString={itemToLabel}
                        selectedItems={selectedEventNameItems}
                        onChange={({ selectedItems }) =>
                          setEventNameInputs(
                            (selectedItems || []).map(item => item.id),
                          )
                        }
                      />
                      <TextInput
                        id="telemetry-filter-event-message-id"
                        labelText="MessageID / ContextID"
                        placeholder="MessageID or ContextID"
                        value={messageIdInput}
                        onChange={(
                          event: React.ChangeEvent<HTMLInputElement>,
                        ) => setMessageIdInput(event.target.value || '')}
                      />
                      <TextInput
                        id="telemetry-filter-event-data-type"
                        labelText="Type"
                        placeholder="Type"
                        value={eventDataTypeInput}
                        onChange={(
                          event: React.ChangeEvent<HTMLInputElement>,
                        ) => setEventDataTypeInput(event.target.value || '')}
                      />
                    </>
                  ) : (
                    <Dropdown<SelectOption>
                      id="telemetry-filter-metric-scope"
                      titleText="Scope"
                      label="Choose scope"
                      items={METRIC_SCOPE_OPTIONS}
                      itemToString={itemToLabel}
                      selectedItem={selectedMetricScope}
                      onChange={({ selectedItem }) =>
                        setMetricScopeInput(selectedItem?.id || '')
                      }
                    />
                  )}
                </div>
              }
            />
          </TableToolbarContent>
        </TableToolbar>

        {(chips.length > 0 ||
          (isEventTab
            ? appliedEventNames.length > 0 ||
              appliedMessageId !== '' ||
              appliedEventDataType !== ''
            : appliedMetricScope !== '') ||
          appliedConversationId !== '' ||
          appliedMessageId !== '') && (
          <div className="flex flex-wrap gap-1.5 border-b border-border-subtle px-4 py-2">
            {appliedConversationId !== '' && (
              <DismissibleTag
                type="teal"
                text={`assistantConversationId: ${appliedConversationId}`}
                onClose={() => {
                  setConversationIdInput('');
                  setAppliedConversationId('');
                  setPage(1);
                }}
              />
            )}
            {appliedMessageId !== '' && (
              <DismissibleTag
                type="teal"
                text={`messageId/contextId: ${appliedMessageId}`}
                onClose={() => {
                  setMessageIdInput('');
                  setAppliedMessageId('');
                  setPage(1);
                }}
              />
            )}
            {chips.map(chip => (
              <DismissibleTag
                key={chip.id}
                type="blue"
                text={`${chip.field}: ${chip.value}`}
                onClose={() => removeChip(chip.id)}
              />
            ))}
            {isEventTab &&
              appliedEventNames.map(appliedEventName => (
                <DismissibleTag
                  key={appliedEventName}
                  type="cyan"
                  text={`name: ${EVENT_NAME_OPTIONS.find(option => option.id === appliedEventName)?.label || appliedEventName}`}
                  onClose={() => {
                    setEventNameInputs(prev =>
                      prev.filter(value => value !== appliedEventName),
                    );
                    setAppliedEventNames(prev =>
                      prev.filter(value => value !== appliedEventName),
                    );
                    setPage(1);
                  }}
                />
              ))}
            {!isEventTab && appliedMetricScope !== '' && (
              <DismissibleTag
                type="cyan"
                text={`scope: ${appliedMetricScope}`}
                onClose={() => {
                  setMetricScopeInput('');
                  setAppliedMetricScope('');
                  setPage(1);
                }}
              />
            )}
            {isEventTab && appliedEventDataType !== '' && (
              <DismissibleTag
                type="cyan"
                text={`data.type: ${appliedEventDataType}`}
                onClose={() => {
                  setEventDataTypeInput('');
                  setAppliedEventDataType('');
                  setPage(1);
                }}
              />
            )}
          </div>
        )}
        {structuredError !== '' && (
          <div className="border-b border-border-subtle px-4 py-2 text-xs text-support-error">
            {structuredError}
          </div>
        )}

        <div className="flex-1 overflow-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loading withOverlay={false} small />
            </div>
          ) : tabVisibleRows.length === 0 ? (
            <div className="flex items-center justify-center py-16 text-sm text-muted">
              No {tabTitle.toLowerCase()} found
            </div>
          ) : (
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader className="!w-8" />
                  <TableHeader className="!w-[180px]">Time</TableHeader>
                  <TableHeader className="!w-[120px]">Type</TableHeader>
                  <TableHeader>Preview</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {tabVisibleRows.map(row => {
                  const { typeLabel, tagType, json } = getTelemetryRowData(row);
                  const isExpanded = expandedRows.has(row.key);
                  return (
                    <React.Fragment key={row.key}>
                      <TableRow
                        className="cursor-pointer hover:bg-layer-hover"
                        onClick={() => toggleRow(row.key)}
                      >
                        <TableCell className="!w-8 !px-2">
                          <ChevronRight
                            size={16}
                            className={`transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                          />
                        </TableCell>
                        <TableCell className="!font-mono !text-xs tabular-nums whitespace-nowrap">
                          {formatDateTime(row.ts)}
                        </TableCell>
                        <TableCell className="!w-[120px]">
                          <Tag size="sm" type={tagType}>
                            {typeLabel}
                          </Tag>
                        </TableCell>
                        <TableCell className="max-w-[300px] truncate !text-xs !text-muted">
                          {JSON.stringify(json)}
                        </TableCell>
                      </TableRow>
                      {isExpanded && (
                        <TableExpandedRow colSpan={5}>
                          <CodeSnippet
                            type="multi"
                            feedback="Copied!"
                            className="!max-w-full"
                          >
                            {JSON.stringify(json, null, 2)}
                          </CodeSnippet>
                        </TableExpandedRow>
                      )}
                    </React.Fragment>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </div>

        {isActiveTab && tabTotalItems > 0 && (
          <Pagination
            totalItems={tabTotalItems}
            page={page}
            pageSize={pageSize}
            pageSizes={[25, 50, 100]}
            onChange={({ page: p, pageSize: ps }) => {
              setPageSize(ps);
              setPage(p);
            }}
          />
        )}
      </>
    );
  };

  return (
    <Modal
      open={modalOpen}
      onClose={() => setModalOpen(false)}
      size="lg"
      preventCloseOnClickOutside
      containerClassName="!h-[90vh] !w-[90vw] !max-h-[90vh] !max-w-[90vw]"
    >
      <ModalHeader
        label="Observability"
        title="Telemetry Events"
        onClose={() => setModalOpen(false)}
      />
      <ModalBody className="!p-0 !overflow-hidden !flex !flex-col">
        <Tabs
          tabs={['Events', 'Metrics', 'Latency']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          fill
          aria-label="Telemetry tabs"
          panelClassName="!p-0"
        >
          <div className="flex flex-1 min-h-0 flex-col">
            {renderTelemetryTable('event')}
          </div>
          <div className="flex flex-1 min-h-0 flex-col">
            {renderTelemetryTable('metric')}
          </div>
          <div className="flex flex-1 min-h-0 flex-col">
            <LatencyStackChart
              isLoading={isLoading}
              latencySeries={latencySeries}
            />
          </div>
        </Tabs>
      </ModalBody>
    </Modal>
  );
}
