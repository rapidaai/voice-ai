import React, { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { Datepicker } from '@/app/components/Datepicker';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { CustomLink } from '@/app/components/custom-link';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { DateTimeColumn } from '@/app/components/Table/DateColumn';
import { formatNanoToReadableMilli, toDateString } from '@/utils';
import { Spinner } from '@/app/components/Loader/Spinner';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { IButton } from '@/app/components/Form/Button';
import { ExternalLink, RotateCw } from 'lucide-react';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { TableRow } from '@/app/components/base/tables/table-row';
import { HttpStatusSpanIndicator } from '@/app/components/indicators/http-status';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { useWebhookLogPage } from '@/hooks/use-webhook-log-page-store';
import { WebhookLogDialog } from '@/app/components/base/modal/webhook-log-modal';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';

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
    webhookLogs,
    onChangeActivities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
    setColumns,
  } = useWebhookLogPage();

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
        onChangeActivities(logs);
      },
    );
  };
  return (
    <>
      {currentActivityId && (
        <WebhookLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentWebhookId={currentActivityId}
        />
      )}

      <Helmet title="Webhook Logs" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Webhook Logs</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${webhookLogs.length}/${totalCount}`}
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

      {webhookLogs && webhookLogs.length > 0 ? (
        <ScrollableResizableTable
          isActionable={false}
          clms={columns.filter(x => {
            return x.visible;
          })}
        >
          {webhookLogs.map((at, idx) => {
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
                {visibleColumn('webhookid') && (
                  <TableCell>
                    <CustomLink
                      to={`/deployment/assistant/${at.getAssistantid()}/manage/configure-webhook`}
                      className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
                    >
                      <span>{at.getWebhookid()}</span>
                      <ExternalLink className="w-3 h-3" />
                    </CustomLink>
                  </TableCell>
                )}

                {visibleColumn('sessionid') && (
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
                {visibleColumn('event') && (
                  <TableCell>
                    <span className="px-2 py-1 text-sm font-mono bg-blue-600/10 text-blue-600">
                      {at.getEvent()}
                    </span>
                  </TableCell>
                )}
                {visibleColumn('created_date') && (
                  <TableCell>
                    {at.getHttpmethod()}:{at.getHttpurl()}
                  </TableCell>
                )}

                {visibleColumn('responsestatus') && (
                  <TableCell>
                    <HttpStatusSpanIndicator
                      status={Number(at.getResponsestatus())}
                    />
                  </TableCell>
                )}
                {visibleColumn('timetaken') && (
                  <TableCell>
                    {formatNanoToReadableMilli(at.getTimetaken())}
                  </TableCell>
                )}

                {visibleColumn('retrycount') && (
                  <TableCell>{at.getRetrycount()}</TableCell>
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
      ) : webhookLogs.length > 0 ? (
        <YellowNoticeBlock>
          <span className="font-semibold">No webhook log found</span>, There are
          no activities matching with your criteria..
        </YellowNoticeBlock>
      ) : !loading ? (
        <YellowNoticeBlock>
          <span className="font-semibold">No webhook log found</span>, There is
          no activities found for your account and project, Any activity made to
          webhooks will be listed here.
        </YellowNoticeBlock>
      ) : (
        <div className="h-full flex justify-center items-center grow">
          <Spinner size="md" />
        </div>
      )}
    </>
  );
}
