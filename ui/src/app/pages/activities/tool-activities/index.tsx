import { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import {
  formatNanoToReadableMilli,
  toHumanReadableDateTime,
} from '@/utils/date';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import { useToolActivityLogPage } from '@/stores/activity/tool-activity-log.store';
import { ToolLogDialog } from '@/app/components/dialogs/activity';
import { CarbonStatusIndicator } from '@/app/components/ui/feedback';
import {
  IconOnlyButton,
  Pagination,
  Text,
} from '@/app/components/ui/primitives';

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
  Link,
} from '@carbon/react';
import { Renew, View, Launch, ToolKit } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/ui/feedback';
import { ToolLogQuerySearch } from './tool-query-search';

export function ListingPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentActivityId, setCurrentActivityId] = useState('');
  const [querySearchValue, setQuerySearchValue] = useState('');
  const [showLogModal, setShowLogModal] = useState(false);

  const {
    getActivities,
    setCriterias,
    activities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
  } = useToolActivityLogPage();

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
      _logs => {
        hideLoader();
      },
    );
  };

  const visibleColumns = columns.filter(c => c.visible);

  return (
    <>
      {currentActivityId && (
        <ToolLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentActivityId={currentActivityId}
        />
      )}

      <div
        data-page-scroll
        className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
      >
        <Helmet title="Tool Logs" />
        <PageHeader
          eyebrow={
            <Text variant="label-01" className="text-primary">
              Observability
            </Text>
          }
          title="Tool Logs"
          description="Monitor tool executions, inputs, outputs, latency, and failures across assistant sessions."
          contextualActions={
            <Text variant="label-01" className="text-muted tabular-nums">
              {activities.length} of {totalCount}
            </Text>
          }
        />

        <PageHeaderStickyToolbar>
          <TableToolbar>
            <TableToolbarContent>
              <ToolLogQuerySearch
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
        ) : activities.length > 0 ? (
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
                {activities.map((at, idx) => (
                  <TableRow key={idx}>
                    {visibleColumn('assistant_id') && (
                      <TableCell className="text-sm">
                        <Link
                          href={`/deployment/assistant/${at.getAssistantid()}`}
                          className="!text-sm !inline-flex !items-center !gap-1"
                        >
                          <span>{at.getAssistantid()}</span>
                          <Launch size={12} />
                        </Link>
                      </TableCell>
                    )}
                    {visibleColumn('assistant_conversation_id') && (
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
                    {visibleColumn('assistant_tool_name') && (
                      <TableCell className="text-sm">
                        {at.getAssistanttoolname()}
                      </TableCell>
                    )}
                    {visibleColumn('tool_call_id') && (
                      <TableCell className="text-[13px]">
                        <span className="font-mono">{at.getToolcallid()}</span>
                      </TableCell>
                    )}
                    {visibleColumn('action') && (
                      <TableCell className="text-sm">
                        <div className="flex items-center gap-0">
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
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={Launch}
                            iconDescription="View conversation"
                            onClick={() => {
                              window.location.href = `/deployment/assistant/${at.getAssistantid()}/sessions/${at.getAssistantconversationid()}`;
                            }}
                          />
                        </div>
                      </TableCell>
                    )}
                    {visibleColumn('status') && (
                      <TableCell className="text-sm">
                        <CarbonStatusIndicator state={at.getStatus()} />
                      </TableCell>
                    )}
                    {visibleColumn('time_taken') && (
                      <TableCell className="font-mono text-[13px]">
                        {formatNanoToReadableMilli(at.getTimetaken())}
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
            icon={ToolKit}
            title="No tool activity logs found"
            subtitle="Tool calls made by your assistants during conversations will appear here once tools are configured and invoked."
          />
        )}

        {activities.length > 0 && (
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
