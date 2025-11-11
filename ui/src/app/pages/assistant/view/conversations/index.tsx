import { useEffect, useState } from 'react';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { Assistant } from '@rapidaai/react';
import { useCredential, useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { toDate, toHumanReadableDateTime } from '@/styles/media';
import { useAssistantConversationListPageStore } from '@/hooks/use-assistant-conversation-list-page-store';
import { AssistantConversation } from '@rapidaai/react';
import { StatusIndicator } from '@/app/components/indicators/status';
import SourceIndicator from '@/app/components/indicators/source';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { useBoolean } from 'ahooks';
import { getStatusMetric } from '@/utils/metadata';
import { TableSection } from '@/app/components/sections/table-section';
import { TableRow } from '@/app/components/base/tables/table-row';
import { AssistantConversationFilterDialog } from '@/app/components/base/modal/assistant-conversation-filter-modal';
import { IButton } from '@/app/components/Form/Button';
import TooltipPlus from '@/app/components/base/tooltip-plus';
import { Download, ExternalLink, ListFilterPlus, RotateCw } from 'lucide-react';
import { Spinner } from '@/app/components/Loader/Spinner';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { CustomLink } from '@/app/components/custom-link';
import { getMetricValue } from '@/utils/metadata';
import { formatNanoToReadableMinute } from '@/utils';
import { ConversationDirectionIndicator } from '@/app/components/indicators/conversation-direction';

interface ConversationProps {
  currentAssistant: Assistant;
}

export function Conversations({ currentAssistant }: ConversationProps) {
  const [userId, token, projectId] = useCredential();
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const rapidaContext = useRapidaStore();
  const navigation = useGlobalNavigation();
  const [isFilterOpen, { setTrue: setFilterOpen, setFalse: setFilterClose }] =
    useBoolean(false);
  const assistantConversationListAction =
    useAssistantConversationListPageStore();

  const [filters, setFilters] = useState<{
    search?: string;
    dateFrom?: string;
    dateTo?: string;
    source?: string;
    id?: string;
    status?: string;
  }>({});

  const applyFilter = (newFilter: {
    search?: string;
    dateFrom?: string;
    dateTo?: string;
    source?: string;
    id?: string;
    status?: string;
  }) => {
    setFilters(newFilter);
    const criterias: { k: string; v: string; logic: string }[] = [];
    if (newFilter.dateFrom) {
      criterias.push({
        k: 'assistant_conversations.created_date',
        v: newFilter.dateFrom,
        logic: '>=',
      });
    }

    if (newFilter.dateTo) {
      criterias.push({
        k: 'assistant_conversations.created_date',
        v: newFilter.dateTo,
        logic: '<=',
      });
    }

    if (newFilter.source) {
      criterias.push({
        k: 'assistant_conversations.source',
        v: newFilter.source,
        logic: '=',
      });
    }

    if (newFilter.id) {
      criterias.push({
        k: 'assistant_conversations.id',
        v: newFilter.id,
        logic: '=',
      });
    }

    if (newFilter.status) {
      criterias.push({
        k: 'assistant_conversations.status',
        v: newFilter.status,
        logic: '=',
      });
    }
    assistantConversationListAction.setCriterias(criterias);
  };

  useEffect(() => {
    assistantConversationListAction.clear();
  }, []);

  const get = () => {
    rapidaContext.showLoader();
    assistantConversationListAction.getAssistantConversations(
      currentAssistant.getId(),
      projectId,
      token,
      userId,
      (err: string) => {
        rapidaContext.hideLoader();
        toast.error(err);
      },
      (data: AssistantConversation[]) => {
        rapidaContext.hideLoader();
      },
    );
  };
  useEffect(() => {
    get();
  }, [
    currentAssistant.getId(),
    projectId,
    assistantConversationListAction.page,
    assistantConversationListAction.pageSize,
    assistantConversationListAction.criteria,
  ]);

  const onToggleSelect = (id: string) => {
    setSelectedIds(prevSelectedIds => {
      if (prevSelectedIds.includes(id)) {
        // Remove from selected if already in the array
        return prevSelectedIds.filter(selectedId => selectedId !== id);
      } else {
        // Add to selected if not in the array
        return [...prevSelectedIds, id];
      }
    });
  };

  const csvEscape = (str: string): string => {
    return `"${str.replace(/"/g, '""')}"`;
  };

  const [downloading, setDownloading] = useState(false);

  const onDownloadAllConversation = () => {
    setDownloading(true);
    const csvContent = [
      // Header row using column names
      assistantConversationListAction.columns
        .filter(column => column.visible)
        .map(column => column.name)
        .join(','),
      // Data rows
      ...assistantConversationListAction.assistantConversations.map(
        (row: AssistantConversation) =>
          assistantConversationListAction.columns
            .filter(column => column.visible)
            .map(column => {
              switch (column.key) {
                case 'id':
                  return row.getId();
                case 'assistant_id':
                  return row.getAssistantid();
                case 'assistant_provider_model_id':
                  return `vrsn_${row.getAssistantprovidermodelid()}`;
                case 'identifier':
                  return csvEscape(row.getIdentifier());
                case 'source':
                  return row.getSource();
                case 'status':
                  return getStatusMetric(row.getMetricsList());
                case 'created_date':
                  return row.getCreateddate()
                    ? toDate(row.getCreateddate()!)
                    : '';
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
    link.setAttribute('download', currentAssistant.getId() + '-sessions.csv');
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    setDownloading(false);
  };

  if (rapidaContext.loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }
  return (
    <div className="h-full flex flex-col flex-1">
      <AssistantConversationFilterDialog
        modalOpen={isFilterOpen}
        setModalOpen={setFilterClose}
        filters={filters}
        onFiltersChange={applyFilter}
      />

      <BluredWrapper className="border-none p-0">
        <SearchIconInput className="bg-light-background" />
        <div className="divide-x dark:divide-gray-800 flex">
          <TablePagination
            columns={assistantConversationListAction.columns}
            currentPage={assistantConversationListAction.page}
            onChangeCurrentPage={assistantConversationListAction.setPage}
            totalItem={assistantConversationListAction.totalCount}
            pageSize={assistantConversationListAction.pageSize}
            onChangePageSize={assistantConversationListAction.setPageSize}
            onChangeColumns={assistantConversationListAction.setColumns}
          />

          <IButton
            type="button"
            onClick={() => {
              setFilterOpen();
            }}
          >
            <TooltipPlus
              className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0"
              popupContent={
                <div className="px-3 py-2 text-sm text-gray-600 dark:text-gray-500">
                  Filter
                </div>
              }
            >
              <ListFilterPlus className="w-4 h-4" strokeWidth={1.5} />
            </TooltipPlus>
          </IButton>
          <IButton
            type="button"
            disabled={downloading}
            onClick={() => {
              onDownloadAllConversation();
            }}
          >
            <TooltipPlus
              className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0"
              popupContent={
                <div className="px-3 py-2 text-sm text-gray-600 dark:text-gray-500">
                  Export as report
                </div>
              }
            >
              {downloading ? (
                <Spinner size="sm"></Spinner>
              ) : (
                <Download className="w-4 h-4" strokeWidth={1.5} />
              )}
            </TooltipPlus>
          </IButton>
          <IButton
            onClick={() => {
              get();
            }}
          >
            <RotateCw strokeWidth={1.5} className="h-4 w-4" />
          </IButton>
        </div>
      </BluredWrapper>
      <TableSection>
        {assistantConversationListAction.assistantConversations.length > 0 ? (
          <ScrollableResizableTable
            clms={assistantConversationListAction.columns.filter(
              x => x.visible,
            )}
          >
            {assistantConversationListAction.assistantConversations.map(
              (row, idx) => (
                <TableRow
                  key={idx}
                  className="cursor-pointer"
                  data-id={row.getId()}
                  onClick={event => {
                    event.stopPropagation();
                    navigation.goToAssistantSession(
                      row.getAssistantid(),
                      row.getId(),
                    );
                    // assistantConversationListAction.showDialog(row);
                  }}
                >
                  <TableCell>
                    {selectedIds.indexOf(row.getId()) > 0 && (
                      <div className="absolute top-0 bottom-0 left-0 bg-blue-500 w-[2px]"></div>
                    )}
                    <div className="w-8 h-8 flex justify-center items-center">
                      <input
                        type="checkbox"
                        name={`checkbox-${row.getId()}--name`}
                        id={`checkbox-${row.getId()}--name`}
                        checked={selectedIds.includes(row.getId())}
                        onClick={event => {
                          event.stopPropagation(); // Prevent <tr> onClick from firing
                        }}
                        onChange={() => {
                          onToggleSelect(row.getId());
                        }}
                      />
                    </div>
                  </TableCell>
                  {assistantConversationListAction.visibleColumn('id') && (
                    <TableCell>
                      <CustomLink
                        to={`/deployment/assistant/${row.getAssistantid()}/sessions/${row.getId()}`}
                        className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
                      >
                        <span>{row.getId()}</span>
                        <ExternalLink className="w-3 h-3" />
                      </CustomLink>
                    </TableCell>
                  )}
                  {assistantConversationListAction.visibleColumn(
                    'assistant_id',
                  ) && <TableCell>{row.getAssistantid()}</TableCell>}

                  {assistantConversationListAction.visibleColumn(
                    'assistant_provider_model_id',
                  ) && (
                    <TableCell>
                      vrsn_{row.getAssistantprovidermodelid()}
                    </TableCell>
                  )}

                  {assistantConversationListAction.visibleColumn(
                    'direction',
                  ) && (
                    <TableCell className="truncate max-w-20">
                      <ConversationDirectionIndicator
                        direction={row.getDirection() || 'inbound'}
                        source={row.getSource()}
                      />
                    </TableCell>
                  )}
                  {assistantConversationListAction.visibleColumn(
                    'identifier',
                  ) && (
                    <TableCell className="truncate max-w-20">
                      {row.getIdentifier()}
                    </TableCell>
                  )}
                  {assistantConversationListAction.visibleColumn('source') && (
                    <TableCell>
                      <SourceIndicator source={row.getSource()} />
                    </TableCell>
                  )}

                  {assistantConversationListAction.visibleColumn(
                    'duration',
                  ) && (
                    <TableCell>
                      {formatNanoToReadableMinute(
                        getMetricValue(row.getMetricsList(), 'TIME_TAKEN'),
                      )}{' '}
                    </TableCell>
                  )}
                  {assistantConversationListAction.visibleColumn('status') && (
                    <TableCell>
                      <StatusIndicator
                        state={getStatusMetric(row.getMetricsList())}
                      />
                    </TableCell>
                  )}

                  {assistantConversationListAction.visibleColumn(
                    'created_date',
                  ) && (
                    <TableCell>
                      {row.getCreateddate() &&
                        toHumanReadableDateTime(row.getCreateddate()!)}
                    </TableCell>
                  )}
                </TableRow>
              ),
            )}
          </ScrollableResizableTable>
        ) : (
          <YellowNoticeBlock>
            <span className="font-semibold">
              No conversations found for this assistant.
            </span>{' '}
            Any conversations made with the assistant will be listed here.
          </YellowNoticeBlock>
        )}
      </TableSection>
    </div>
  );
}
