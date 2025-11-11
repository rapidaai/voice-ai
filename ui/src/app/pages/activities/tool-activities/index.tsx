import React, { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { Datepicker } from '@/app/components/Datepicker';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { Metadata } from '@rapidaai/react';
import { CustomLink } from '@/app/components/custom-link';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { DateTimeColumn } from '@/app/components/Table/DateColumn';
import { formatNanoToReadableMilli, toDateString } from '@/utils';
import { getMetadataValue } from '@/utils/metadata';
import { Spinner } from '@/app/components/Loader/Spinner';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { IButton } from '@/app/components/Form/Button';
import { ExternalLink, RotateCw } from 'lucide-react';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { TableRow } from '@/app/components/base/tables/table-row';
import { StatusIndicator } from '@/app/components/indicators/status';
import { LLMLogDialog } from '@/app/components/base/modal/llm-log-modal';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { useToolActivityLogPage } from '@/hooks/use-tool-activity-log-page-store';
import { ToolLogDialog } from '@/app/components/base/modal/tool-log-modal';
import { ToolActionPill } from '@/app/components/Pill/tool-action-pill';

/**
 * Listing all the audit log for the user organization and selected project
 * @returns
 */

export function ListingPage() {
  /**
   * set loading context
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();

  /**
   * Current activity Id
   */
  const [currentActivityId, setCurrentActivityId] = useState('');

  /**
   *  open modal
   */
  const [showLogModal, setShowLogModal] = useState(false);

  const {
    getActivities,
    addCriterias,
    activities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
    setColumns,
  } = useToolActivityLogPage();

  const onDateSelect = (to: Date, from: Date) => {
    addCriterias([
      { k: 'created_date', v: toDateString(to), logic: '<=' },
      { k: 'created_date', v: toDateString(from), logic: '>=' },
    ]);
  };

  /**
   *
   */
  useEffect(() => {
    showLoader();
    onGetAcitvities();
  }, [projectId, page, pageSize, JSON.stringify(criteria)]);

  const onGetAcitvities = () => {
    getActivities(
      projectId,
      token,
      userId,
      err => {
        hideLoader();
        toast.error(err);
      },
      logs => {
        hideLoader();
      },
    );
  };
  return (
    <>
      {currentActivityId && (
        <ToolLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentActivityId={currentActivityId}
        />
      )}

      <Helmet title="Tool Logs" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Tool Logs</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${activities.length}/${totalCount}`}
          </div>
        </div>
      </PageHeaderBlock>

      <BluredWrapper className="p-0">
        <div className="flex justify-center items-center">
          <SearchIconInput className="bg-light-background" />
          <Datepicker
            align="right"
            className="bg-light-background"
            onDateSelect={onDateSelect}
          />
        </div>
        <PaginationButtonBlock>
          <TablePagination
            columns={columns}
            currentPage={page}
            onChangeCurrentPage={setPage}
            totalItem={totalCount}
            pageSize={pageSize}
            onChangePageSize={setPageSize}
            onChangeColumns={setColumns}
          />
          <IButton
            onClick={() => {
              onGetAcitvities();
            }}
          >
            <RotateCw strokeWidth={1.5} className="h-4 w-4" />
          </IButton>
        </PaginationButtonBlock>
      </BluredWrapper>

      {activities && activities.length > 0 ? (
        <ScrollableResizableTable
          isActionable={false}
          clms={columns.filter(x => {
            return x.visible;
          })}
        >
          {activities.map((at, idx) => {
            return (
              <TableRow
                key={idx}
                data-id={at.getId()}
                onClick={event => {
                  event.stopPropagation();
                  setCurrentActivityId(at.getId());
                  setShowLogModal(true);
                }}
              >
                {visibleColumn('assistant_id') && (
                  <TableCell>
                    <CustomLink
                      to={`/deployment/assistant/${at.getAssistantid()}`}
                      className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
                    >
                      <span>{at.getAssistantid()}</span>
                      <ExternalLink className="w-3 h-3" />
                    </CustomLink>
                  </TableCell>
                )}
                {visibleColumn('assistant_conversation_id') && (
                  <TableCell>
                    <CustomLink
                      to={`/deployment/assistant/${at.getAssistantid()}/sessions/${at.getAssistantconversationid()}`}
                      className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
                    >
                      <span>{at.getAssistantconversationid()}</span>
                      <ExternalLink className="w-3 h-3" />
                    </CustomLink>
                  </TableCell>
                )}
                {visibleColumn('assistant_tool_name') && (
                  <TableCell>{at.getAssistanttoolname()}</TableCell>
                )}

                {visibleColumn('execution_method') && (
                  <TableCell>
                    <ToolActionPill code={at.getExecutionmethod()} />
                  </TableCell>
                )}

                {visibleColumn('status') && (
                  <TableCell>
                    <StatusIndicator state={at.getStatus()} />
                  </TableCell>
                )}
                {visibleColumn('time_taken') && (
                  <TableCell>
                    {formatNanoToReadableMilli(at.getTimetaken())}
                  </TableCell>
                )}
                {visibleColumn('created_date') && (
                  <TableCell>
                    <DateTimeColumn date={at.getCreateddate()} />
                  </TableCell>
                )}
              </TableRow>
            );
          })}
          {/* </TBody> */}
        </ScrollableResizableTable>
      ) : activities.length > 0 ? (
        <YellowNoticeBlock>
          <span className="font-semibold">No activities found</span>, There are
          no activities matching with your criteria..
        </YellowNoticeBlock>
      ) : !loading ? (
        <YellowNoticeBlock>
          <span className="font-semibold">No activities found</span>, There is
          no activities found for your account and project, Any activity made to
          any of the llms will be listed here.
        </YellowNoticeBlock>
      ) : (
        <div className="h-full flex justify-center items-center grow">
          <Spinner size="md" />
        </div>
      )}
    </>
  );
}

function ActivitySource(props: { metadatas: Metadata[] }) {
  const [acivitySource, setActivitySource] = useState('');
  const [activityLink, setActivityLink] = useState('');
  const [external, setExternal] = useState(true);
  useEffect(() => {
    const endpoint = getMetadataValue(props.metadatas, 'endpoint_id');
    if (endpoint) {
      setActivitySource(endpoint);
      setActivityLink(`/deployment/endpoint/${endpoint}`);
      setExternal(false);
      return;
    }

    const assistant = getMetadataValue(props.metadatas, 'assistant_id');
    if (assistant) {
      setActivitySource(assistant);
      setActivityLink(`/deployment/assistant/${assistant}`);
      setExternal(false);
      return;
    }

    const knowledge = getMetadataValue(props.metadatas, 'knowledge_id');
    if (knowledge) {
      setActivitySource(knowledge);
      setActivityLink(`/knowledge/${knowledge}`);
      setExternal(false);
      return;
    }

    const source = getMetadataValue(props.metadatas, 'source');
    if (source) {
      setActivitySource(source);
      setActivityLink('');
      setExternal(false);
      return;
    }
  }, [props.metadatas]);
  return (
    <CustomLink
      to={activityLink}
      className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
    >
      <span>{acivitySource}</span>
      <ExternalLink className="w-3 h-3" />
    </CustomLink>
  );
}
