import { FC, useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { ChartLine, Edit, Power, TrashCan } from '@carbon/icons-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/ui/feedback';
import { CreateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/create-assistant-analysis';
import { useAssistantAnalysisPageStore } from '@/stores/assistant/actions';
import { UpdateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/update-assistant-analysis';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import { Pagination } from '@/app/components/ui/primitives';
import {
  Button,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  ComposedModal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  DataTable,
  DataTableSkeleton,
  TableBatchAction,
  TableBatchActions,
  TableSelectRow,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
} from '@carbon/react';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';

const getAnalysisOptionMap = (row: any): Map<string, string> => {
  const map = new Map<string, string>();
  const options = row?.getOptionsList?.() || [];
  options.forEach((option: any) => {
    const key = option?.getKey?.();
    const value = option?.getValue?.();
    if (key && typeof value === 'string') {
      map.set(key, value);
    }
  });
  return map;
};

const getAnalysisEndpointId = (row: any): string =>
  getAnalysisOptionMap(row).get('endpoint_id') || '';

const getAnalysisEndpointVersion = (row: any): string =>
  getAnalysisOptionMap(row).get('endpoint_version') || 'latest';

const getAnalysisName = (row: any): string =>
  getAnalysisOptionMap(row).get('name') || '';

const getAnalysisPriority = (row: any): number => {
  const value = Number(getAnalysisOptionMap(row).get('execution_priority'));
  return Number.isFinite(value) ? value : 0;
};

const headers = [
  { key: 'name', header: 'Name' },
  { key: 'endpoint', header: 'Endpoint' },
  { key: 'version', header: 'Version' },
  { key: 'priority', header: 'Priority' },
  { key: 'status', header: 'Status' },
  { key: 'date', header: 'Date' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

type AnalysisAction = {
  kind: 'enable' | 'disable' | 'delete';
  analysis: any;
};

export function ConfigureAssistantAnalysisPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantAnalysis assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantAnalysisPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantAnalysis assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantAnalysisPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantAnalysis assistantId={assistantId} />}</>
  );
}

const ConfigureAssistantAnalysis: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const axtion = useAssistantAnalysisPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const [isFetching, setIsFetching] = useState(true);
  const [pendingAnalysisAction, setPendingAnalysisAction] =
    useState<AnalysisAction | null>(null);

  const get = useCallback(() => {
    setIsFetching(true);
    axtion.getAssistantAnalysis(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setIsFetching(false);
      },
      () => {
        setIsFetching(false);
      },
    );
  }, [assistantId, projectId, token, authId, axtion.getAssistantAnalysis]);

  useEffect(() => {
    get();
  }, [get, axtion.page, axtion.pageSize]);

  const deleteAssistantAnalysis = (assistantId: string, analysisId: string) => {
    setIsFetching(true);
    axtion.deleteAssistantAnalysis(
      assistantId,
      analysisId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setIsFetching(false);
      },
      () => {
        toast.success('Analysis deleted successfully');
        setPendingAnalysisAction(null);
        get();
      },
    );
  };

  const updateAnalysisEnabled = (analysis: any, enabled: boolean) => {
    setIsFetching(true);
    axtion.updateAssistantAnalysisEnabled(
      assistantId,
      analysis,
      enabled,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setIsFetching(false);
      },
      () => {
        toast.success(
          `Analysis ${enabled ? 'enabled' : 'disabled'} successfully`,
        );
        setPendingAnalysisAction(null);
        get();
      },
    );
  };

  const modalTitle =
    pendingAnalysisAction?.kind === 'enable'
      ? 'Enable analysis?'
      : pendingAnalysisAction?.kind === 'disable'
        ? 'Disable analysis?'
        : 'Delete analysis?';
  const modalContent =
    pendingAnalysisAction?.kind === 'enable'
      ? 'This analysis will run when assistant finalization triggers it.'
      : pendingAnalysisAction?.kind === 'disable'
        ? 'This analysis will stop running until it is enabled again.'
        : 'This analysis configuration will be removed from the assistant.';
  const modalPrimaryLabel =
    pendingAnalysisAction?.kind === 'enable'
      ? 'Enable'
      : pendingAnalysisAction?.kind === 'disable'
        ? 'Disable'
        : 'Delete';

  const tableRows = axtion.analysises.map(row => ({
    id: row.getId(),
    name: getAnalysisName(row),
    endpoint: getAnalysisEndpointId(row),
    version: getAnalysisEndpointVersion(row),
    priority: String(getAnalysisPriority(row)),
    status: row.getStatus(),
    date: row.getCreateddate()?.toString() || '',
  }));

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col bg-surface">
      <ComposedModal
        open={Boolean(pendingAnalysisAction)}
        onClose={() => setPendingAnalysisAction(null)}
        size="sm"
        danger={pendingAnalysisAction?.kind !== 'enable'}
      >
        <ModalHeader title={modalTitle} />
        <ModalBody>
          <p>{modalContent}</p>
        </ModalBody>
        <ModalFooter danger={pendingAnalysisAction?.kind !== 'enable'}>
          <Button
            kind="secondary"
            size="md"
            onClick={() => setPendingAnalysisAction(null)}
          >
            Cancel
          </Button>
          <Button
            kind={
              pendingAnalysisAction?.kind === 'enable' ? 'primary' : 'danger'
            }
            size="md"
            onClick={() => {
              if (!pendingAnalysisAction) return;
              if (pendingAnalysisAction.kind === 'delete') {
                deleteAssistantAnalysis(
                  assistantId,
                  pendingAnalysisAction.analysis.getId(),
                );
                return;
              }
              updateAnalysisEnabled(
                pendingAnalysisAction.analysis,
                pendingAnalysisAction.kind === 'enable',
              );
            }}
          >
            {modalPrimaryLabel}
          </Button>
        </ModalFooter>
      </ComposedModal>
      <AssistantSettingsShellHeader
        title="Analysis"
        href={`/deployment/assistant/${assistantId}/configure-analysis`}
        onReload={get}
        reloadLabel="Reload analysis"
        primaryAction={{
          label: 'Create analysis',
          onClick: () => navigation.goToCreateAssistantAnalysis(assistantId),
        }}
      />
      {isFetching ? (
        <TableSection className="min-h-0 overflow-hidden bg-surface">
          <DataTableSkeleton
            headers={skeletonHeaders}
            columnCount={skeletonHeaders.length}
            rowCount={5}
            showHeader={false}
            showToolbar
            size="lg"
            className="w-full"
          />
        </TableSection>
      ) : axtion.analysises.length > 0 ? (
        <TableSection className="min-h-0 bg-surface">
          <DataTable rows={tableRows} headers={headers} radio>
            {({
              rows,
              headers: tableHeaders,
              selectedRows,
              getBatchActionProps,
              getHeaderProps,
              getRowProps,
              getSelectionProps,
              getTableProps,
              getToolbarProps,
              onInputChange,
            }) => {
              const batchActionProps = getBatchActionProps();
              const selectedAnalysis = selectedRows[0]
                ? axtion.analysises.find(
                    row => row.getId() === selectedRows[0].id,
                  )
                : undefined;

              return (
                <>
                  <TableToolbar {...getToolbarProps()}>
                    {selectedAnalysis && (
                      <TableBatchActions
                        {...batchActionProps}
                        shouldShowBatchActions
                      >
                        <TableBatchAction
                          renderIcon={Edit}
                          iconDescription="Edit"
                          onClick={() => {
                            navigation.goToEditAssistantAnalysis(
                              assistantId,
                              selectedAnalysis.getId(),
                            );
                            batchActionProps.onCancel();
                          }}
                        >
                          Edit
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={Power}
                          iconDescription={
                            selectedAnalysis.getEnabled() ? 'Disable' : 'Enable'
                          }
                          onClick={() => {
                            setPendingAnalysisAction({
                              kind: selectedAnalysis.getEnabled()
                                ? 'disable'
                                : 'enable',
                              analysis: selectedAnalysis,
                            });
                            batchActionProps.onCancel();
                          }}
                        >
                          {selectedAnalysis.getEnabled() ? 'Disable' : 'Enable'}
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={TrashCan}
                          iconDescription="Delete"
                          onClick={() => {
                            setPendingAnalysisAction({
                              kind: 'delete',
                              analysis: selectedAnalysis,
                            });
                            batchActionProps.onCancel();
                          }}
                        >
                          Delete
                        </TableBatchAction>
                      </TableBatchActions>
                    )}
                    <TableToolbarContent>
                      <TableToolbarSearch
                        placeholder="Search analysis..."
                        onChange={onInputChange}
                      />
                    </TableToolbarContent>
                  </TableToolbar>
                  <div className="min-h-0 flex-1 overflow-auto">
                    <Table {...getTableProps()} className="min-w-max">
                      <TableHead>
                        <TableRow>
                          <TableHeader className="!w-12" />
                          {tableHeaders.map(header => (
                            <TableHeader
                              {...getHeaderProps({ header })}
                              key={header.key}
                            >
                              {header.header}
                            </TableHeader>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {rows.map(tableRow => {
                          const analysis = axtion.analysises.find(
                            row => row.getId() === tableRow.id,
                          );
                          if (!analysis) return null;

                          return (
                            <TableRow
                              {...getRowProps({ row: tableRow })}
                              key={tableRow.id}
                            >
                              <TableSelectRow
                                {...getSelectionProps({ row: tableRow })}
                              />
                              <TableCell className="text-sm">
                                {getAnalysisName(analysis)}
                              </TableCell>
                              <TableCell className="text-sm">
                                <span className="font-mono text-[13px]">
                                  {getAnalysisEndpointId(analysis) || '—'}
                                </span>
                              </TableCell>
                              <TableCell className="text-sm">
                                {getAnalysisEndpointVersion(analysis)}
                              </TableCell>
                              <TableCell className="text-sm">
                                {getAnalysisPriority(analysis)}
                              </TableCell>
                              <TableCell className="text-sm whitespace-nowrap">
                                <RecordStatusIndicator
                                  state={analysis.getStatus()}
                                  textSize={14}
                                />
                              </TableCell>
                              <TableCell className="text-[13px] whitespace-nowrap">
                                {analysis.getCreateddate()
                                  ? toHumanReadableDateTime(
                                      analysis.getCreateddate()!,
                                    )
                                  : '—'}
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </div>
                  <Pagination
                    sticky
                    totalItems={axtion.totalCount}
                    page={axtion.page}
                    pageSize={axtion.pageSize}
                    pageSizes={[10, 20, 50]}
                    onChange={({ page, pageSize }) => {
                      if (pageSize !== axtion.pageSize) {
                        axtion.setPageSize(pageSize);
                      } else {
                        axtion.setPage(page);
                      }
                    }}
                  />
                </>
              );
            }}
          </DataTable>
        </TableSection>
      ) : (
        <div className="flex flex-1 items-center justify-center">
          <EmptyState
            icon={ChartLine}
            title="No analysis"
            subtitle="Any analysis you add will be listed here."
          />
        </div>
      )}
    </div>
  );
};
