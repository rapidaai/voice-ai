import { useState, useEffect, FC } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import { Endpoint, EndpointLog } from '@rapidaai/react';
import { SourceIndicator } from '@/app/components/domain/indicators/source';
import {
  formatNanoToReadableMilli,
  toDateString,
  toHumanReadableDateTime,
} from '@/utils/date';
import { getTimeTakenMetric, getTotalTokenMetric } from '@/utils/metadata';
import { EndpointTraceModal } from '@/app/components/dialogs/endpoint';
import { useEndpointLogPage } from '@/stores/endpoint/endpoint-log.store';
import { CarbonStatusIndicator } from '@/app/components/ui/feedback';
import { Pagination } from '@/app/components/ui/primitives';
import { IconOnlyButton } from '@/app/components/ui/primitives';
import { CopyButton } from '@/app/components/ui/primitives';
import { DateFilter } from '@/app/components/ui/composites';
import { EmptyState } from '@/app/components/ui/feedback';
import { Renew, View, Activity } from '@carbon/icons-react';
import { ScrollableTableSection } from '@/app/components/layout/sections/table-section';

import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Loading,
} from '@carbon/react';

export const EndpointTraces: FC<{ currentEndpoint: Endpoint }> = props => {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentTrace, setCurrentTrace] = useState<EndpointLog | null>(null);
  const [showTraceModal, setShowTraceModal] = useState(false);
  const [loadingTraceId, setLoadingTraceId] = useState<string | null>(null);

  const {
    getLogs,
    getLog,
    addCriterias,
    endpointLogs,
    onChangeLogs,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    setPageSize,
    visibleColumn,
  } = useEndpointLogPage();

  const onDateSelect = (to: Date, from: Date) => {
    addCriterias([
      { k: 'created_date', v: toDateString(to), logic: '<=' },
      { k: 'created_date', v: toDateString(from), logic: '>=' },
    ]);
  };

  useEffect(() => {
    onGetAllEndpointLogs();
  }, [
    projectId,
    page,
    pageSize,
    JSON.stringify(criteria),
    props.currentEndpoint.getId(),
  ]);

  const onGetAllEndpointLogs = () => {
    showLoader();
    getLogs(
      props.currentEndpoint.getId(),
      projectId,
      token,
      userId,
      err => {
        hideLoader();
        toast.error(err);
      },
      logs => {
        hideLoader();
        onChangeLogs(logs);
      },
    );
  };

  const onGetEndpointLog = (row: EndpointLog) => {
    const logId = String(row.getId());
    setLoadingTraceId(logId);
    getLog(
      props.currentEndpoint.getId(),
      logId,
      projectId,
      token,
      userId,
      err => {
        setLoadingTraceId(null);
        toast.error(err);
      },
      log => {
        setLoadingTraceId(null);
        setCurrentTrace(log);
        setShowTraceModal(true);
      },
    );
  };

  const visibleColumns = columns.filter(c => c.visible);

  return (
    <div className="flex flex-1 flex-col">
      <Helmet title="Endpoint Logs" />

      <EndpointTraceModal
        modalOpen={showTraceModal}
        setModalOpen={setShowTraceModal}
        currentTrace={currentTrace}
      />

      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search endpoint logs" />
          <DateFilter
            onApply={(from, to) => onDateSelect(to, from)}
            onReset={() => addCriterias([])}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={() => onGetAllEndpointLogs()}
          />
        </TableToolbarContent>
      </TableToolbar>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loading withOverlay={false} small />
        </div>
      ) : endpointLogs && endpointLogs.length > 0 ? (
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
              {endpointLogs.map((row, idx) => {
                return (
                  <TableRow key={idx}>
                    {visibleColumn('id') && (
                      <TableCell className="text-[13px]">
                        <span className="font-mono">{row.getId()}</span>
                      </TableCell>
                    )}
                    {visibleColumn('version') && (
                      <TableCell className="text-[13px]">
                        <div className="flex min-w-0 items-center gap-1">
                          <span className="font-mono">
                            {`vrsn_${row.getEndpointprovidermodelid()}`}
                          </span>
                          {row.getEndpointprovidermodelid() && (
                            <CopyButton className="h-6 w-6 shrink-0">
                              {`vrsn_${row.getEndpointprovidermodelid()}`}
                            </CopyButton>
                          )}
                        </div>
                      </TableCell>
                    )}
                    {visibleColumn('source') && (
                      <TableCell className="text-sm">
                        <SourceIndicator source={row.getSource()} />
                      </TableCell>
                    )}
                    {visibleColumn('status') && (
                      <TableCell className="text-sm">
                        <CarbonStatusIndicator state={row.getStatus()} />
                      </TableCell>
                    )}
                    {visibleColumn('action') && (
                      <TableCell>
                        <div className="flex items-center gap-0">
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={View}
                            iconDescription="View detail"
                            isLoading={loadingTraceId === String(row.getId())}
                            onClick={() => onGetEndpointLog(row)}
                          />
                        </div>
                      </TableCell>
                    )}
                    {visibleColumn('timetaken') && (
                      <TableCell className="font-mono text-[13px]">
                        {formatNanoToReadableMilli(row.getTimetaken())}
                      </TableCell>
                    )}
                    {visibleColumn('total_token') && (
                      <TableCell className="font-mono text-[13px]">
                        {getTotalTokenMetric(row.getMetricsList())}
                      </TableCell>
                    )}
                    {visibleColumn('time_taken') && (
                      <TableCell className="font-mono text-[13px]">
                        {formatNanoToReadableMilli(
                          getTimeTakenMetric(row.getMetricsList()),
                        )}
                      </TableCell>
                    )}
                    {visibleColumn('created_date') && (
                      <TableCell className="text-[13px] whitespace-nowrap">
                        {row.getCreateddate() &&
                          toHumanReadableDateTime(row.getCreateddate()!)}
                      </TableCell>
                    )}
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </ScrollableTableSection>
      ) : (
        <EmptyState
          icon={Activity}
          title="No endpoint logs found"
          subtitle="API requests made to this endpoint will appear here as logs with latency, token usage, and status details."
        />
      )}

      {endpointLogs && endpointLogs.length > 0 && (
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
  );
};
