import { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import {
  formatNanoToReadableMilli,
  toHumanReadableDateTime,
} from '@/utils/date';
import { HttpStatusSpanIndicator } from '@/app/components/domain/indicators/http-status';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import { useWebhookLogPage } from '@/stores/activity/webhook-log.store';
import { RequestLogDialog } from '@/app/components/dialogs/activity';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { retryWebhookActivityLog } from '@/clients';

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
import {
  IconOnlyButton,
  Pagination,
  Text,
} from '@/app/components/ui/primitives';
import { UrlTableCell } from '@/app/components/ui/table';
import { Renew, View, EventSchedule, Launch } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/ui/feedback';
import { RequestLogQuerySearch } from './request-query-search';

export function ListingPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentActivityId, setCurrentActivityId] = useState('');
  const [querySearchValue, setQuerySearchValue] = useState('');
  const [showLogModal, setShowLogModal] = useState(false);
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Retry request?',
    content:
      'This will re-run the selected HTTP request. Do you want to continue?',
  });

  const {
    getActivities,
    setCriterias,
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
  } = useWebhookLogPage();

  useEffect(() => {
    showLoader();
    onGetActivities();
  }, [projectId, page, pageSize, JSON.stringify(criteria)]);

  const onGetActivities = () => {
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

  const retryRequestLog = async (requestLogId: string) => {
    showLoader();

    try {
      const response = await retryWebhookActivityLog({
        projectId,
        requestLogId,
        auth: { projectId, token, userId },
      });

      if (response?.getSuccess()) {
        toast.success('Request retried successfully.');
        onGetActivities();
        return;
      }

      const message = response?.getError()?.getHumanmessage();
      toast.error(message || 'Unable to retry the request, please try again.');
    } catch {
      toast.error('Unable to retry the request, please try again.');
    } finally {
      hideLoader();
    }
  };

  const visibleColumns = columns.filter(c => c.visible);

  return (
    <>
      <ConfirmDialogComponent />
      {currentActivityId && (
        <RequestLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentRequestLogId={currentActivityId}
        />
      )}

      <div
        data-page-scroll
        className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
      >
        <Helmet title="Request Logs" />
        <PageHeader
          eyebrow={
            <Text variant="label-01" className="text-primary">
              Observability
            </Text>
          }
          title="Request Logs"
          description="Monitor HTTP request activity, responses, latency, retries, and failures across your workspace."
          contextualActions={
            <Text variant="label-01" className="text-muted tabular-nums">
              {webhookLogs.length} of {totalCount}
            </Text>
          }
        />

        <PageHeaderStickyToolbar>
          <TableToolbar>
            <TableToolbarContent>
              <RequestLogQuerySearch
                value={querySearchValue}
                onChange={setQuerySearchValue}
                onApply={setCriterias}
              />
              <IconOnlyButton
                kind="ghost"
                size="lg"
                renderIcon={Renew}
                iconDescription="Refresh"
                onClick={() => onGetActivities()}
              />
            </TableToolbarContent>
          </TableToolbar>
        </PageHeaderStickyToolbar>

        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loading withOverlay={false} small />
          </div>
        ) : webhookLogs.length > 0 ? (
          <div className="shrink-0 overflow-x-auto">
            <Table className="min-w-max">
              <TableHead>
                <TableRow>
                  {visibleColumns.map(col => (
                    <TableHeader key={col.key}>{col.name}</TableHeader>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {webhookLogs.map((at, idx) => (
                  <TableRow key={idx}>
                    {visibleColumn('sourcerefid') && (
                      <TableCell className="text-[13px]">
                        <span className="font-mono">{at.getSourcerefid()}</span>
                      </TableCell>
                    )}
                    {visibleColumn('sessionid') && (
                      <TableCell className="text-sm">
                        <Link
                          href={`/deployment/assistant/${at.getAssistantid()}/sessions/${at.getAssistantconversationid()}`}
                          className="!text-sm !inline-flex !items-center !gap-1"
                        >
                          <span>{at.getAssistantconversationid()}</span>
                          <Launch size={12} />
                        </Link>
                      </TableCell>
                    )}
                    {visibleColumn('event') && (
                      <TableCell className="text-sm">
                        <Tag size="sm" type="blue">
                          {at.getSourceevent()}
                        </Tag>
                      </TableCell>
                    )}
                    {visibleColumn('endpoint') && (
                      <UrlTableCell
                        url={at.getHttpurl()}
                        prefix={
                          at.getHttpmethod() ? (
                            <span className="font-mono text-[13px] shrink-0">
                              {at.getHttpmethod()}:
                            </span>
                          ) : null
                        }
                        maxWidthClassName="max-w-[560px]"
                      />
                    )}
                    {visibleColumn('action') && (
                      <TableCell className="text-sm">
                        <IconOnlyButton
                          kind="ghost"
                          size="md"
                          renderIcon={Renew}
                          iconDescription="Retry request"
                          onClick={() =>
                            showDialog(() => retryRequestLog(at.getId()))
                          }
                        />
                        <IconOnlyButton
                          kind="ghost"
                          size="md"
                          renderIcon={View}
                          iconDescription="View detail"
                          onClick={() => {
                            setCurrentActivityId(at.getId());
                            setShowLogModal(true);
                          }}
                        />
                      </TableCell>
                    )}
                    {visibleColumn('responsestatus') && (
                      <TableCell className="text-sm">
                        <HttpStatusSpanIndicator
                          status={Number(at.getResponsestatus())}
                        />
                      </TableCell>
                    )}
                    {visibleColumn('timetaken') && (
                      <TableCell className="font-mono text-[13px]">
                        {formatNanoToReadableMilli(at.getTimetaken())}
                      </TableCell>
                    )}
                    {visibleColumn('retrycount') && (
                      <TableCell className="text-sm">
                        {at.getRetrycount()}
                      </TableCell>
                    )}
                    {visibleColumn('created_date') && (
                      <TableCell className="text-[13px] whitespace-nowrap">
                        {at.getCreateddate() &&
                          toHumanReadableDateTime(at.getCreateddate()!)}
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={EventSchedule}
            title="No request logs found"
            subtitle="HTTP request logs will appear here once requests are triggered by assistant workflows."
          />
        )}

        {webhookLogs.length > 0 && (
          <Pagination
            sticky
            totalItems={totalCount}
            page={page}
            pageSize={pageSize}
            pageSizes={[10, 20, 25, 50, 100]}
            onChange={({ page: p, pageSize: ps }) => {
              if (ps !== pageSize) setPageSize(ps);
              else setPage(p);
            }}
          />
        )}
      </div>
    </>
  );
}
