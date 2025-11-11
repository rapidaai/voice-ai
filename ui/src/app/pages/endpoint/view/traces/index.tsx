import React, { useState, useEffect, FC } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { Datepicker } from '@/app/components/Datepicker';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { toDateString } from '@/utils';
import { Spinner } from '@/app/components/Loader/Spinner';
import { ScrollableResizableTable } from '@/app/components/data-table';
import { IButton } from '@/app/components/Form/Button';
import { RotateCw } from 'lucide-react';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { TableRow } from '@/app/components/base/tables/table-row';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { useEndpointLogPage } from '@/hooks/use-endpoint-log-page-store';
import { Endpoint, EndpointLog } from '@rapidaai/react';

import SourceIndicator from '@/app/components/indicators/source';
import { StatusIndicator } from '@/app/components/indicators/status';
import { cn, toHumanReadableDateTime } from '@/styles/media';
import { getTimeTakenMetric, getTotalTokenMetric } from '@/utils/metadata';
import { ChevronRight } from 'lucide-react';
import { SideTab } from '@/app/components/Tab';
import { EndpointMetrics } from '@/app/pages/endpoint/view/traces/endpoint-metrics';
import { EndpointMetadatas } from '@/app/pages/endpoint/view/traces/endpoint-metadatas';
import { EndpointArguments } from '@/app/pages/endpoint/view/traces/endpoint-arguments';
import { EndpointOptions } from '@/app/pages/endpoint/view/traces/endpoint-options';

/**
 * Listing all the audit log for the user organization and selected project
 * @returns
 */

export const EndpointTraces: FC<{ currentEndpoint: Endpoint }> = props => {
  /**
   * set loading context
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();

  const {
    getLogs,
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
    setColumns,
  } = useEndpointLogPage();

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
  return (
    <div className="flex flex-1 flex-col">
      <Helmet title="Endpoint Logs" />
      <BluredWrapper className="p-0">
        <div className="flex justify-center items-center">
          <SearchIconInput className="bg-light-background" />
          <Datepicker
            align="right"
            className="bg-light-background"
            onDateSelect={onDateSelect}
          />
        </div>

        <div className="flex flex-row divide-x dark:divide-gray-800">
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
              onGetAllEndpointLogs();
            }}
          >
            <RotateCw strokeWidth={1.5} className="h-4 w-4" />
          </IButton>
        </div>
      </BluredWrapper>

      {endpointLogs && endpointLogs.length > 0 ? (
        <ScrollableResizableTable
          isExpandable={true}
          isActionable={false}
          clms={columns.filter(x => {
            return x.visible;
          })}
        >
          {endpointLogs.map((at, idx) => {
            return <SingleTrace key={idx} row={at} idx={idx} />;
          })}
          {/* </TBody> */}
        </ScrollableResizableTable>
      ) : endpointLogs.length > 0 ? (
        <YellowNoticeBlock>
          <span className="font-semibold">No activity found</span>, There are no
          activities matching with your criteria..
        </YellowNoticeBlock>
      ) : !loading ? (
        <YellowNoticeBlock>
          <span className="font-semibold">No activity found</span>, There is no
          activities found for your account and project, Any activity made to
          webhooks will be listed here.
        </YellowNoticeBlock>
      ) : (
        <div className="h-full flex justify-center items-center grow">
          <Spinner size="md" />
        </div>
      )}
    </div>
  );
};

interface SingleTraceProps {
  row: EndpointLog;
  idx: number;
}

export const SingleTrace: React.FC<SingleTraceProps> = ({ row, idx }) => {
  const endpointAction = useEndpointLogPage();
  const [info, setInfo] = useState(false);
  return (
    <>
      <TableRow
        key={idx}
        data-id={row.getId()}
        onClick={event => {
          event.stopPropagation();
          setInfo(!info);
        }}
      >
        <TableCell className="py-0 px-0">
          <ChevronRight
            className={cn(
              'w-5 h-5 transition-all duration-200',
              info && 'rotate-90',
            )}
          />
        </TableCell>
        {endpointAction.visibleColumn('id') && (
          <TableCell>{row.getId()}</TableCell>
        )}
        {endpointAction.visibleColumn('version') && (
          <TableCell>vrsn_{row.getEndpointprovidermodelid()}</TableCell>
        )}

        {endpointAction.visibleColumn('source') && (
          <TableCell>
            <SourceIndicator source={row.getSource()} />
          </TableCell>
        )}

        {endpointAction.visibleColumn('status') && (
          <TableCell>
            <StatusIndicator state={row.getStatus()} />
          </TableCell>
        )}

        {endpointAction.visibleColumn('timetaken') && (
          <TableCell>{Number(row.getTimetaken()) / 1000000}ms</TableCell>
        )}

        {endpointAction.visibleColumn('total_token') && (
          <TableCell>{getTotalTokenMetric(row.getMetricsList())}</TableCell>
        )}

        {endpointAction.visibleColumn('time_taken') && (
          <TableCell>
            {getTimeTakenMetric(row.getMetricsList()) / 1000000}ms
          </TableCell>
        )}

        {endpointAction.visibleColumn('created_date') && (
          <TableCell>
            {row.getCreateddate() &&
              toHumanReadableDateTime(row.getCreateddate()!)}
          </TableCell>
        )}
      </TableRow>

      <TableRow
        className={cn(
          'transition-all duration-200',
          info ? ' visible' : 'collapse pointer-events-none',
        )}
      >
        <TableCell
          className="px-0! py-0!"
          colSpan={endpointAction.columns.filter(x => x.visible).length + 1}
        >
          <div className="flex p-3.5 dark:bg-gray-950 bg-gray-100">
            <div className="flex h-full w-full border dark:bg-gray-900 bg-white">
              <SideTab
                strict={false}
                active="metrics"
                className={cn('w-56')}
                tabs={[
                  {
                    label: 'metrics',
                    element: <EndpointMetrics metrics={row.getMetricsList()} />,
                  },
                  {
                    label: 'metadata',
                    element: (
                      <div className="gap-4">
                        <EndpointMetadatas metadata={row.getMetadataList()} />
                      </div>
                    ),
                  },
                  {
                    label: 'options',
                    element: (
                      <div className="gap-4">
                        <EndpointOptions options={row.getOptionsList()} />
                      </div>
                    ),
                  },
                  {
                    label: 'arguments',
                    element: (
                      <div className="gap-4">
                        <EndpointArguments args={row.getArgumentsList()} />
                      </div>
                    ),
                  },
                ]}
              />
            </div>
          </div>
        </TableCell>
      </TableRow>
    </>
  );
};
