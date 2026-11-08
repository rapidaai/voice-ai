import { FC, useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { Activity, Edit, Power, TrashCan } from '@carbon/icons-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/ui/feedback';
import { CreateAssistantTelemetry } from './create-assistant-telemetry';
import { UpdateAssistantTelemetry } from './update-assistant-telemetry';
import { useAssistantTelemetryPageStore } from '@/stores/assistant/actions';
import { TELEMETRY_PROVIDER } from '@/providers';
import { CarbonShapeIndicator } from '@/app/components/ui/feedback';
import {
  Button,
  ComposedModal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  DataTable,
  DataTableSkeleton,
  Table,
  TableBatchAction,
  TableBatchActions,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableSelectRow,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Tag,
} from '@carbon/react';
import { AssistantConfiguration, Metadata } from '@rapidaai/react';
import { Pagination } from '@/app/components/ui/primitives';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';

export function ConfigureAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantTelemetry assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantTelemetry assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantTelemetry assistantId={assistantId} />}</>
  );
}

const providerNameByCode = new Map(
  TELEMETRY_PROVIDER.map(p => [p.code, p.name]),
);

const getOptionValue = (options: Metadata[], key: string) =>
  options.find(option => option.getKey() === key)?.getValue() || '';

const getTelemetryTarget = (telemetry: AssistantConfiguration) => {
  return getOptionValue(telemetry.getOptionsList(), 'endpoint') || '-';
};

const headers = [
  { key: 'provider', header: 'Provider' },
  { key: 'type', header: 'Type' },
  { key: 'target', header: 'Target' },
  { key: 'status', header: 'Status' },
  { key: 'created', header: 'Created' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

type TelemetryAction = {
  kind: 'enable' | 'disable' | 'delete';
  telemetry: AssistantConfiguration;
};

const ConfigureAssistantTelemetry: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const action = useAssistantTelemetryPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const [isFetching, setIsFetching] = useState(true);
  const [pendingTelemetryAction, setPendingTelemetryAction] =
    useState<TelemetryAction | null>(null);

  const get = useCallback(() => {
    setIsFetching(true);
    action.getAssistantTelemetry(
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
  }, [assistantId, projectId, token, authId, action.getAssistantTelemetry]);

  useEffect(() => {
    get();
  }, [get, action.page, action.pageSize]);

  const deleteTelemetry = (telemetryId: string) => {
    setIsFetching(true);
    action.deleteAssistantTelemetry(
      assistantId,
      telemetryId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setIsFetching(false);
      },
      () => {
        toast.success('Telemetry provider deleted successfully');
        setPendingTelemetryAction(null);
        get();
      },
    );
  };

  const updateTelemetryEnabled = (
    telemetry: AssistantConfiguration,
    enabled: boolean,
  ) => {
    setIsFetching(true);
    action.updateAssistantTelemetryEnabled(
      assistantId,
      telemetry,
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
          `Telemetry ${enabled ? 'enabled' : 'disabled'} successfully`,
        );
        setPendingTelemetryAction(null);
        get();
      },
    );
  };

  const modalTitle =
    pendingTelemetryAction?.kind === 'enable'
      ? 'Enable telemetry?'
      : pendingTelemetryAction?.kind === 'disable'
        ? 'Disable telemetry?'
        : 'Delete telemetry?';
  const modalContent =
    pendingTelemetryAction?.kind === 'enable'
      ? 'Telemetry will start being pushed to this provider.'
      : pendingTelemetryAction?.kind === 'disable'
        ? 'Telemetry will stop being pushed to this provider until it is enabled again.'
        : 'This telemetry provider will be removed from the assistant.';
  const modalPrimaryLabel =
    pendingTelemetryAction?.kind === 'enable'
      ? 'Enable'
      : pendingTelemetryAction?.kind === 'disable'
        ? 'Disable'
        : 'Delete';

  const tableRows = action.telemetries.map(telemetry => {
    const provider = telemetry.getProvider();
    return {
      id: telemetry.getId(),
      provider: providerNameByCode.get(provider) || provider,
      type: telemetry.getConfigurationtype() || 'telemetry',
      target: getTelemetryTarget(telemetry),
      status: telemetry.getEnabled() ? 'Enabled' : 'Disabled',
      created: telemetry.getCreateddate()?.toString() || '',
    };
  });

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col bg-surface">
      <ComposedModal
        open={Boolean(pendingTelemetryAction)}
        onClose={() => setPendingTelemetryAction(null)}
        size="sm"
        danger={pendingTelemetryAction?.kind !== 'enable'}
      >
        <ModalHeader title={modalTitle} />
        <ModalBody>
          <p>{modalContent}</p>
        </ModalBody>
        <ModalFooter danger={pendingTelemetryAction?.kind !== 'enable'}>
          <Button
            kind="secondary"
            size="md"
            onClick={() => setPendingTelemetryAction(null)}
          >
            Cancel
          </Button>
          <Button
            kind={
              pendingTelemetryAction?.kind === 'enable' ? 'primary' : 'danger'
            }
            size="md"
            onClick={() => {
              if (!pendingTelemetryAction) return;
              if (pendingTelemetryAction.kind === 'delete') {
                deleteTelemetry(pendingTelemetryAction.telemetry.getId());
                return;
              }
              updateTelemetryEnabled(
                pendingTelemetryAction.telemetry,
                pendingTelemetryAction.kind === 'enable',
              );
            }}
          >
            {modalPrimaryLabel}
          </Button>
        </ModalFooter>
      </ComposedModal>

      <AssistantSettingsShellHeader
        title="Telemetry"
        href={`/deployment/assistant/${assistantId}/configure-telemetry`}
        onReload={get}
        reloadLabel="Reload telemetry"
        primaryAction={{
          label: 'Add telemetry',
          onClick: () => navigation.goToCreateAssistantTelemetry(assistantId),
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
      ) : action.telemetries.length > 0 ? (
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
              const selectedTelemetry = selectedRows[0]
                ? action.telemetries.find(
                    telemetry => telemetry.getId() === selectedRows[0].id,
                  )
                : undefined;

              return (
                <>
                  <TableToolbar {...getToolbarProps()}>
                    {selectedTelemetry && (
                      <TableBatchActions
                        {...batchActionProps}
                        shouldShowBatchActions
                      >
                        <TableBatchAction
                          renderIcon={Edit}
                          iconDescription="Edit"
                          onClick={() => {
                            navigation.goToEditAssistantTelemetry(
                              assistantId,
                              selectedTelemetry.getId(),
                            );
                            batchActionProps.onCancel();
                          }}
                        >
                          Edit
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={Power}
                          iconDescription={
                            selectedTelemetry.getEnabled()
                              ? 'Disable'
                              : 'Enable'
                          }
                          onClick={() => {
                            setPendingTelemetryAction({
                              kind: selectedTelemetry.getEnabled()
                                ? 'disable'
                                : 'enable',
                              telemetry: selectedTelemetry,
                            });
                            batchActionProps.onCancel();
                          }}
                        >
                          {selectedTelemetry.getEnabled()
                            ? 'Disable'
                            : 'Enable'}
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={TrashCan}
                          iconDescription="Delete"
                          onClick={() => {
                            setPendingTelemetryAction({
                              kind: 'delete',
                              telemetry: selectedTelemetry,
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
                        placeholder="Search telemetry..."
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
                          const telemetry = action.telemetries.find(
                            candidate => candidate.getId() === tableRow.id,
                          );
                          if (!telemetry) return null;

                          const provider = telemetry.getProvider();
                          const providerName =
                            providerNameByCode.get(provider) || provider;

                          return (
                            <TableRow
                              {...getRowProps({ row: tableRow })}
                              key={tableRow.id}
                            >
                              <TableSelectRow
                                {...getSelectionProps({ row: tableRow })}
                              />
                              <TableCell className="text-sm">
                                {providerName}
                              </TableCell>
                              <TableCell className="text-sm">
                                <Tag type="blue" size="sm">
                                  {telemetry.getConfigurationtype() ||
                                    'telemetry'}
                                </Tag>
                              </TableCell>
                              <TableCell className="text-sm">
                                {getTelemetryTarget(telemetry)}
                              </TableCell>
                              <TableCell className="text-sm whitespace-nowrap">
                                <CarbonShapeIndicator
                                  kind={
                                    telemetry.getEnabled() ? 'stable' : 'draft'
                                  }
                                  label={
                                    telemetry.getEnabled()
                                      ? 'Enabled'
                                      : 'Disabled'
                                  }
                                  textSize={14}
                                />
                              </TableCell>
                              <TableCell className="text-[13px] whitespace-nowrap">
                                {telemetry.getCreateddate()
                                  ? toHumanReadableDateTime(
                                      telemetry.getCreateddate()!,
                                    )
                                  : '-'}
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </div>
                  <Pagination
                    sticky
                    totalItems={action.totalCount}
                    page={action.page}
                    pageSize={action.pageSize}
                    pageSizes={[10, 20, 50]}
                    onChange={({ page: newPage, pageSize: newSize }) => {
                      if (newSize !== action.pageSize) {
                        action.setPageSize(newSize);
                        return;
                      }
                      action.setPage(newPage);
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
            icon={Activity}
            title="No telemetry providers"
            subtitle="Any telemetry providers you add will be listed here."
          />
        </div>
      )}
    </div>
  );
};
