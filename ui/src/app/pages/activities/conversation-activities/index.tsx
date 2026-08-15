import { FC, useEffect, useState } from 'react';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks/use-rapida-store';
import toast from 'react-hot-toast/headless';
import { AssistantConversationMessage } from '@rapidaai/react';
import {
  formatNanoToReadableMilli,
  toDate,
  toHumanReadableDateTime,
} from '@/utils/date';
import {
  getMetricValueOrDefault,
  getMetadataValueOrDefault,
  getTimeTakenMetric,
  getTotalTokenMetric,
} from '@/utils/metadata';
import { useConversationLogPageStore } from '@/hooks/use-conversation-log-page-store';
import { Helmet } from '@/app/components/helmet';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { CONFIG } from '@/configs';
import { CarbonStatusIndicator } from '@/app/components/carbon/status-indicator';
import SourceIndicator from '@/app/components/indicators/source';
import { ConversationLogDialog } from '@/app/components/base/modal/conversation-log-modal';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';

import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  Loading,
  Tag,
  Link,
} from '@carbon/react';
import { Pagination } from '@/app/components/carbon/pagination';
import { IconOnlyButton } from '@/app/components/carbon/button';
import {
  Download,
  Renew,
  View,
  Launch,
  DataCheck,
  Bot,
  User as UserIcon,
  Chat,
} from '@carbon/icons-react';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { ScrollableTableSection } from '@/app/components/sections/table-section';
import { ConversationLogQuerySearch } from './conversation-query-search';

export const ListingPage: FC<{}> = () => {
  const [userId, token, projectId] = useCredential();
  const rapidaContext = useRapidaStore();
  const [downloading, setDownloading] = useState(false);
  const conversationLogAction = useConversationLogPageStore();
  const navigation = useGlobalNavigation();

  const [currentActivity, setCurrentActivity] =
    useState<AssistantConversationMessage | null>(null);
  const [showLogModal, setShowLogModal] = useState(false);

  const [querySearchValue, setQuerySearchValue] = useState('');

  useEffect(() => {
    conversationLogAction.clear();
  }, []);

  const get = () => {
    rapidaContext.showLoader();
    conversationLogAction.getMessages(
      projectId,
      token,
      userId,
      (err: string) => {
        rapidaContext.hideLoader();
        toast.error(err);
      },
      (data: AssistantConversationMessage[]) => {
        rapidaContext.hideLoader();
      },
    );
  };

  useEffect(() => {
    get();
  }, [
    projectId,
    conversationLogAction.page,
    conversationLogAction.pageSize,
    JSON.stringify(conversationLogAction.criteria),
  ]);

  const csvEscape = (str: string): string => {
    return `"${str.replace(/"/g, '""')}"`;
  };

  const formatMetricMilliseconds = (value: string): string => {
    const numericValue = Number(value);
    if (!Number.isFinite(numericValue)) return value;
    return `${Number.isInteger(numericValue) ? numericValue : numericValue.toFixed(2)} ms`;
  };

  const getMetricDisplayValue = (
    row: AssistantConversationMessage,
    metricName: string,
    formatter: (value: string) => string = value => value,
  ): string => {
    const value = getMetricValueOrDefault(row.getMetricsList(), metricName, '');
    return value ? formatter(value) : '--';
  };

  const getLanguageDisplayValue = (
    row: AssistantConversationMessage,
  ): string => {
    const language = getMetadataValueOrDefault(
      row.getMetadataList(),
      'language',
      '',
    );
    return language || '--';
  };

  const onDownloadAllTraces = () => {
    setDownloading(true);
    const csvContent = [
      conversationLogAction.columns
        .filter(column => column.visible)
        .map(column => column.name)
        .join(','),
      ...conversationLogAction.assistantMessages.map(
        (row: AssistantConversationMessage) =>
          conversationLogAction.columns
            .filter(column => column.visible)
            .map(column => {
              switch (column.key) {
                case 'id':
                  return row.getId();
                case 'session_id':
                case 'assistant_conversation_id':
                  return row.getAssistantconversationid();
                case 'assistant_id':
                  return row.getAssistantid();
                case 'source':
                  return row.getSource();
                case 'role':
                  return csvEscape(row.getRole());
                case 'message':
                  return csvEscape(row.getBody());
                case 'created_date':
                  return row.getCreateddate()
                    ? toDate(row.getCreateddate()!)
                    : '';
                case 'status':
                  return row.getStatus();
                case 'stt.latency_ms':
                  return getMetricDisplayValue(
                    row,
                    'stt.latency_ms',
                    formatMetricMilliseconds,
                  );
                case 'llm_latency_ms':
                  return getMetricDisplayValue(
                    row,
                    'llm_latency_ms',
                    formatMetricMilliseconds,
                  );
                case 'agent.ttft_ms':
                  return getMetricDisplayValue(
                    row,
                    'agent.ttft_ms',
                    formatMetricMilliseconds,
                  );
                case 'tts_latency_ms':
                  return getMetricDisplayValue(
                    row,
                    'tts_latency_ms',
                    formatMetricMilliseconds,
                  );
                case 'eos_latency_ms':
                  return getMetricDisplayValue(
                    row,
                    'eos_latency_ms',
                    formatMetricMilliseconds,
                  );
                case 'time_taken':
                  return `${getTimeTakenMetric(row.getMetricsList()) / 1000000}ms`;
                case 'total_token':
                  return getTotalTokenMetric(row.getMetricsList());
                case 'language':
                  return getLanguageDisplayValue(row);
                default:
                  return '';
              }
            })
            .join(','),
      ),
    ].join('\n');
    const url = URL.createObjectURL(
      new Blob([csvContent], { type: 'text/csv;charset=utf-8;' }),
    );

    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', projectId + '-trace-messages.csv');
    document.body.appendChild(link);
    setDownloading(false);

    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  const visibleColumns = conversationLogAction.columns.filter(c => c.visible);

  return (
    <>
      {currentActivity && (
        <ConversationLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentAssistantMessage={currentActivity}
        />
      )}
      <div className="h-full flex flex-col overflow-hidden">
        <Helmet title="Conversation Logs" />
        <PageHeaderBlock>
          <PageTitleWithCount
            count={conversationLogAction.assistantMessages.length}
            total={conversationLogAction.totalCount}
          >
            Conversation Logs
          </PageTitleWithCount>
        </PageHeaderBlock>

        {/* ── Carbon Toolbar ── */}
        <TableToolbar>
          <TableToolbarContent>
            <ConversationLogQuerySearch
              value={querySearchValue}
              onChange={setQuerySearchValue}
              onApply={conversationLogAction.setCriterias}
            />
            <IconOnlyButton
              kind="ghost"
              size="lg"
              renderIcon={Download}
              iconDescription="Export as CSV"
              isLoading={downloading}
              onClick={() => onDownloadAllTraces()}
            />
            <IconOnlyButton
              kind="ghost"
              size="lg"
              renderIcon={Renew}
              iconDescription="Refresh"
              onClick={() => get()}
            />
          </TableToolbarContent>
        </TableToolbar>

        {/* ── Table ── */}
        {rapidaContext.loading ? (
          <div className="flex items-center justify-center py-16">
            <Loading withOverlay={false} small />
          </div>
        ) : conversationLogAction.assistantMessages.length > 0 ? (
          <ScrollableTableSection>
            <Table className="min-w-max">
              <TableHead>
                <TableRow>
                  {visibleColumns.map(col => (
                    <TableHeader key={col.key}>{col.name}</TableHeader>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {conversationLogAction.assistantMessages.map((row, idx) => (
                  <TableRow key={idx}>
                    {conversationLogAction.visibleColumn('id') && (
                      <TableCell className="font-mono text-[13px]">
                        {row.getMessageid().split('-').pop()}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('version') && (
                      <TableCell className="text-sm">
                        vrsn_{row.getAssistantprovidermodelid()}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn(
                      'assistant_conversation_id',
                    ) && (
                      <TableCell className="text-sm">
                        <Link
                          href={`/deployment/assistant/${row.getAssistantid()}/sessions/${row.getAssistantconversationid()}`}
                          className="text-sm inline-flex! items-center gap-1"
                        >
                          <span>{row.getAssistantconversationid()}</span>
                          <Launch size={12} />
                        </Link>
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('assistant_id') && (
                      <TableCell className="text-sm">
                        <Link
                          href={`/deployment/assistant/${row.getAssistantid()}`}
                          className="text-sm inline-flex! items-center gap-1"
                        >
                          <span>{row.getAssistantid()}</span>
                          <Launch size={12} />
                        </Link>
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('source') && (
                      <TableCell className="text-sm">
                        <SourceIndicator source={row.getSource()} />
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('role') && (
                      <TableCell className="text-sm">
                        {row.getRole() ? (
                          <Tag
                            size="md"
                            type={
                              row.getRole().toLowerCase() === 'assistant'
                                ? 'blue'
                                : 'cool-gray'
                            }
                          >
                            <span className="flex items-center gap-1.5 leading-none">
                              {row.getRole().toLowerCase() === 'assistant' ? (
                                <Bot size={16} />
                              ) : (
                                <UserIcon size={16} />
                              )}
                              {row.getRole().toLowerCase() === 'assistant'
                                ? 'Assistant'
                                : 'User'}
                            </span>
                          </Tag>
                        ) : (
                          <span className="text-gray-400 text-sm">N/A</span>
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('message') && (
                      <TableCell className="max-w-75 text-sm">
                        {row.getBody() ? (
                          <p className="line-clamp-2 text-sm">
                            {row.getBody()}
                          </p>
                        ) : (
                          <span className="text-gray-400 text-sm">N/A</span>
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('created_date') && (
                      <TableCell className="text-[13px] whitespace-nowrap">
                        {row.getCreateddate() &&
                          toHumanReadableDateTime(row.getCreateddate()!)}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('action') && (
                      <TableCell className="text-sm">
                        <div className="flex items-center gap-0">
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={View}
                            iconDescription="View detail"
                            onClick={() => {
                              setCurrentActivity(row);
                              setShowLogModal(true);
                            }}
                          />
                          {CONFIG.workspace.features?.telemetry !== false && (
                            <IconOnlyButton
                              kind="ghost"
                              size="md"
                              renderIcon={DataCheck}
                              iconDescription="View telemetry"
                              onClick={() =>
                                navigation.goToMessageTelemetry(
                                  row.getMessageid(),
                                )
                              }
                            />
                          )}
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={Launch}
                            iconDescription="View conversation"
                            onClick={() => {
                              navigation.goToAssistantSession(
                                row.getAssistantid(),
                                row.getAssistantconversationid(),
                              );
                            }}
                          />
                        </div>
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('status') && (
                      <TableCell className="text-sm">
                        <CarbonStatusIndicator
                          state={
                            row.getRole()?.toLowerCase() === 'assistant'
                              ? getMetricValueOrDefault(
                                  row.getMetricsList(),
                                  'assistant_turn',
                                  row.getStatus(),
                                )
                              : row.getRole()?.toLowerCase() === 'user'
                                ? getMetricValueOrDefault(
                                    row.getMetricsList(),
                                    'user_turn',
                                    row.getStatus(),
                                  )
                                : row.getStatus()
                          }
                        />
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('time_taken') && (
                      <TableCell className="font-mono text-[13px]">
                        {formatNanoToReadableMilli(
                          getTimeTakenMetric(row.getMetricsList()),
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('stt.latency_ms') && (
                      <TableCell className="font-mono text-[13px]">
                        {getMetricDisplayValue(
                          row,
                          'stt.latency_ms',
                          formatMetricMilliseconds,
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('llm_latency_ms') && (
                      <TableCell className="font-mono text-[13px]">
                        {getMetricDisplayValue(
                          row,
                          'llm_latency_ms',
                          formatMetricMilliseconds,
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('agent.ttft_ms') && (
                      <TableCell className="font-mono text-[13px]">
                        {getMetricDisplayValue(
                          row,
                          'agent.ttft_ms',
                          formatMetricMilliseconds,
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('tts_latency_ms') && (
                      <TableCell className="font-mono text-[13px]">
                        {getMetricDisplayValue(
                          row,
                          'tts_latency_ms',
                          formatMetricMilliseconds,
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('eos_latency_ms') && (
                      <TableCell className="font-mono text-[13px]">
                        {getMetricDisplayValue(
                          row,
                          'eos_latency_ms',
                          formatMetricMilliseconds,
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('total_token') && (
                      <TableCell className="text-sm tabular-nums">
                        {getTotalTokenMetric(row.getMetricsList())}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('language') && (
                      <TableCell className="text-sm">
                        {getLanguageDisplayValue(row)}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn('user_feedback') && (
                      <TableCell className="text-sm">
                        {getMetricValueOrDefault(
                          row.getMetricsList(),
                          'custom.feedback',
                          '__',
                        )}
                      </TableCell>
                    )}
                    {conversationLogAction.visibleColumn(
                      'user_text_feedback',
                    ) && (
                      <TableCell className="text-sm">
                        {getMetricValueOrDefault(
                          row.getMetricsList(),
                          'custom.feedback_text',
                          '--',
                        )}
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollableTableSection>
        ) : (
          <EmptyState
            icon={Chat}
            title="No conversation logs found"
            subtitle="Messages exchanged between users and your assistants will appear here as conversations take place."
          />
        )}

        {/* ── Pagination ── */}
        {conversationLogAction.assistantMessages.length > 0 && (
          <Pagination
            totalItems={conversationLogAction.totalCount}
            page={conversationLogAction.page}
            pageSize={conversationLogAction.pageSize}
            pageSizes={[10, 20, 25, 50, 100]}
            onChange={({ page: p, pageSize: ps }) => {
              if (ps !== conversationLogAction.pageSize) {
                conversationLogAction.setPageSize(ps);
              } else {
                conversationLogAction.setPage(p);
              }
            }}
          />
        )}
      </div>
    </>
  );
};
