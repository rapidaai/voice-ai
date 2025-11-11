import { FC, useEffect, useState } from 'react';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { useCredential, useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { AssistantConversationMessage, Criteria } from '@rapidaai/react';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { SingleTrace } from './single-trace';
import { IButton } from '@/app/components/Form/Button';
import { Download, ListFilterPlus, RotateCw } from 'lucide-react';
import TooltipPlus from '@/app/components/base/tooltip-plus';
import { AssistantTraceFilterDialog } from '@/app/components/base/modal/assistant-trace-filter-modal';
import { useBoolean } from 'ahooks';
import { toContentText } from '@/utils/rapida_content';
import { toDate } from '@/styles/media';
import { getTimeTakenMetric } from '@/utils/metadata';
import { Spinner } from '@/app/components/Loader/Spinner';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { useConversationLogPageStore } from '@/hooks/use-conversation-log-page-store';
import { Helmet } from '@/app/components/Helmet';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { ConversationTelemetryDialog } from '@/app/components/base/modal/conversation-telemetry-modal';

export const ListingPage: FC<{}> = ({}) => {
  const [userId, token, projectId] = useCredential();
  const rapidaContext = useRapidaStore();
  const [downloading, setDownloading] = useState(false);
  const conversationLogAction = useConversationLogPageStore();
  const [isFilterOpen, { setTrue: setFilterOpen, setFalse: setFilterClose }] =
    useBoolean(false);

  const [criterias, setCriterias] = useState<Criteria[]>([]);
  const [isTelemetryDialogOpen, setTelemetryDialogOpen] = useState(false);

  const handleTraceClick = (trace: AssistantConversationMessage) => {
    const ctr = new Criteria();
    ctr.setKey('assistantId');
    ctr.setLogic('match');
    ctr.setValue(trace.getAssistantid());

    const ctr2 = new Criteria();
    ctr2.setKey('assistantConversationId');
    ctr2.setLogic('match');
    ctr2.setValue(trace.getAssistantconversationid());

    const ctr3 = new Criteria();
    ctr3.setKey('attributes.messageId');
    ctr3.setLogic('match');
    ctr3.setValue(trace.getMessageid());
    setCriterias([ctr, ctr2, ctr3]);
    setTelemetryDialogOpen(true);
  };

  const [filters, setFilters] = useState<{
    search?: string;
    dateFrom?: string;
    dateTo?: string;
    source?: string;
    sessionId?: string;
    id?: string;
    status?: string;
  }>({});

  const applyFilter = (newFilter: {
    search?: string;
    dateFrom?: string;
    dateTo?: string;
    source?: string;
    sessionId?: string;
    id?: string;
    status?: string;
  }) => {
    setFilters(newFilter);
    const criterias: { k: string; v: string; logic: string }[] = [];
    if (newFilter.dateFrom) {
      criterias.push({
        k: 'assistant_conversation_messages.created_date',
        v: newFilter.dateFrom,
        logic: '>=',
      });
    }

    if (newFilter.dateTo) {
      criterias.push({
        k: 'assistant_conversation_messages.created_date',
        v: newFilter.dateTo,
        logic: '<=',
      });
    }

    if (newFilter.source) {
      criterias.push({
        k: 'assistant_conversation_messages.source',
        v: newFilter.source,
        logic: '=',
      });
    }

    if (newFilter.sessionId) {
      criterias.push({
        k: 'assistant_conversation_messages.assistant_conversation_id',
        v: newFilter.sessionId,
        logic: '=',
      });
    }
    if (newFilter.id) {
      criterias.push({
        k: 'assistant_conversation_messages.id',
        v: newFilter.id,
        logic: '=',
      });
    }

    if (newFilter.status) {
      criterias.push({
        k: 'assistant_conversation_messages.status',
        v: newFilter.status,
        logic: '=',
      });
    }
    conversationLogAction.setCriterias(criterias);
  };

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

  if (rapidaContext.loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  const csvEscape = (str: string): string => {
    return `"${str.replace(/"/g, '""')}"`;
  };
  const onDownloadAllTraces = () => {
    setDownloading(true);
    const csvContent = [
      // Header row using column names
      conversationLogAction.columns
        .filter(column => column.visible)
        .map(column => column.name)
        .join(','),
      // Data rows
      ...conversationLogAction.assistantMessages.map(
        (row: AssistantConversationMessage) =>
          conversationLogAction.columns
            .filter(column => column.visible)
            .map(column => {
              switch (column.key) {
                case 'id':
                  return row.getId();
                case 'session_id':
                  return row.getAssistantconversationid();
                case 'assistant_id':
                  return row.getAssistantid();
                case 'source':
                  return row.getSource();
                case 'request':
                  return csvEscape(
                    toContentText(row.getRequest()?.getContentsList()),
                  );
                case 'response':
                  return csvEscape(
                    toContentText(row.getResponse()?.getContentsList()),
                  );
                case 'created_date':
                  return row.getCreateddate()
                    ? toDate(row.getCreateddate()!)
                    : '';
                case 'status':
                  return row.getStatus();
                case 'time_taken':
                  return `${getTimeTakenMetric(row.getMetricsList()) / 1000000}ms`;
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

  return (
    <>
      {isTelemetryDialogOpen && (
        <ConversationTelemetryDialog
          modalOpen={isTelemetryDialogOpen}
          setModalOpen={setTelemetryDialogOpen}
          criterias={criterias}
        />
      )}
      <Helmet title="Conversation Logs" />
      <AssistantTraceFilterDialog
        modalOpen={isFilterOpen}
        setModalOpen={setFilterClose}
        filters={filters}
        onFiltersChange={applyFilter}
      />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Conversation Logs</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${conversationLogAction.assistantMessages.length}/${conversationLogAction.totalCount}`}
          </div>
        </div>
      </PageHeaderBlock>
      <BluredWrapper className="border-t p-0">
        <div className="flex">
          <SearchIconInput
            className="bg-light-background"
            value={filters.search}
            onChange={value => {
              const newValue = value.target.value;
              const newFilters = { ...filters };
              const filterRegex = /(id|session):(\S+)/g;
              let match;
              let hasMatch = false;
              newFilters.id = '';
              newFilters.sessionId = '';

              if (newValue === '') {
                // Reset all filters when input is cleared
                setFilters({ search: '', id: '', sessionId: '' });
                applyFilter({ search: '', id: '', sessionId: '' });
                return;
              }

              while ((match = filterRegex.exec(newValue)) !== null) {
                const [, filterType, filterValue] = match;
                hasMatch = true;
                switch (filterType) {
                  case 'id':
                    newFilters.id = filterValue;
                    break;
                  case 'session':
                    newFilters.sessionId = filterValue;
                    break;
                }
              }
              newFilters.search = newValue;
              setFilters(newFilters);
              if (hasMatch) {
                applyFilter(newFilters);
              }
            }}
            placeholder="Search by id:trace-id, session:session-id"
          />
        </div>
        <PaginationButtonBlock>
          <TablePagination
            columns={conversationLogAction.columns}
            currentPage={conversationLogAction.page}
            onChangeCurrentPage={conversationLogAction.setPage}
            totalItem={conversationLogAction.totalCount}
            pageSize={conversationLogAction.pageSize}
            onChangePageSize={conversationLogAction.setPageSize}
            onChangeColumns={conversationLogAction.setColumns}
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
            onClick={() => {
              onDownloadAllTraces();
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
        </PaginationButtonBlock>
      </BluredWrapper>
      {conversationLogAction.assistantMessages.length > 0 ? (
        <ScrollableResizableTable
          isExpandable={true}
          isActionable={false}
          clms={conversationLogAction.columns.filter(x => x.visible)}
        >
          {conversationLogAction.assistantMessages.map((row, idx) => (
            <SingleTrace
              key={idx}
              row={row}
              idx={idx}
              onClick={() => handleTraceClick(row)}
            />
          ))}
        </ScrollableResizableTable>
      ) : (
        <YellowNoticeBlock>
          <span className="font-semibold">No activities found</span>, Any
          activities performed by the assistant will be listed here.
        </YellowNoticeBlock>
      )}
    </>
  );
};
