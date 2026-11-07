import { FC, useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { Edit, ObjectStorage, Power, TrashCan } from '@carbon/icons-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/ui/feedback';
import { CreateAssistantStorage } from './create-assistant-storage';
import { UpdateAssistantStorage } from './update-assistant-storage';
import { useAssistantStoragePageStore } from '@/stores/assistant/actions';
import { STORAGE_PROVIDER } from '@/providers';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
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
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
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

export function ConfigureAssistantStoragePage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantStorage assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantStoragePage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantStorage assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantStoragePage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantStorage assistantId={assistantId} />}</>
  );
}

const providerNameByCode = new Map(STORAGE_PROVIDER.map(p => [p.code, p.name]));

const getOptionValue = (options: Metadata[], key: string) =>
  options.find(option => option.getKey() === key)?.getValue() || '';

const getStorageTarget = (storage: AssistantConfiguration) => {
  const options = storage.getOptionsList();

  switch (storage.getProvider()) {
    case 'azure-cloud':
    case 'azure': {
      return getOptionValue(options, 'container') || '-';
    }
    case 'aws':
    case 'aws-cloud':
      return getOptionValue(options, 's3_bucket_name') || '-';
    default:
      return '-';
  }
};

const headers = [
  { key: 'provider', header: 'Provider' },
  { key: 'type', header: 'Type' },
  { key: 'target', header: 'Target' },
  { key: 'status', header: 'Status' },
  { key: 'created', header: 'Created' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

type StorageAction = {
  kind: 'enable' | 'disable' | 'delete';
  storage: AssistantConfiguration;
};

const ConfigureAssistantStorage: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const action = useAssistantStoragePageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const [isFetching, setIsFetching] = useState(true);
  const [pendingStorageAction, setPendingStorageAction] =
    useState<StorageAction | null>(null);

  const get = useCallback(() => {
    setIsFetching(true);
    action.getAssistantStorage(
      assistantId,
      projectId,
      token,
      authId,
      error => {
        toast.error(error);
        setIsFetching(false);
      },
      () => {
        setIsFetching(false);
      },
    );
  }, [assistantId, projectId, token, authId, action.getAssistantStorage]);

  useEffect(() => {
    get();
  }, [get, action.page, action.pageSize]);

  const deleteStorage = (storageId: string) => {
    setIsFetching(true);
    action.deleteAssistantStorage(
      assistantId,
      storageId,
      projectId,
      token,
      authId,
      error => {
        toast.error(error);
        setIsFetching(false);
      },
      () => {
        toast.success('Storage deleted successfully');
        setPendingStorageAction(null);
        get();
      },
    );
  };

  const updateStorageEnabled = (
    storage: AssistantConfiguration,
    enabled: boolean,
  ) => {
    setIsFetching(true);
    action.updateAssistantStorageEnabled(
      assistantId,
      storage,
      enabled,
      projectId,
      token,
      authId,
      error => {
        toast.error(error);
        setIsFetching(false);
      },
      () => {
        toast.success(
          `Storage ${enabled ? 'enabled' : 'disabled'} successfully`,
        );
        setPendingStorageAction(null);
        get();
      },
    );
  };

  const modalTitle =
    pendingStorageAction?.kind === 'enable'
      ? 'Enable storage?'
      : pendingStorageAction?.kind === 'disable'
        ? 'Disable storage?'
        : 'Delete storage?';
  const modalContent =
    pendingStorageAction?.kind === 'enable'
      ? 'Recordings will be pushed to this storage configuration.'
      : pendingStorageAction?.kind === 'disable'
        ? 'Recordings will stop using this storage configuration until it is enabled again.'
        : 'This storage configuration will be removed from the assistant.';
  const modalPrimaryLabel =
    pendingStorageAction?.kind === 'enable'
      ? 'Enable'
      : pendingStorageAction?.kind === 'disable'
        ? 'Disable'
        : 'Delete';

  const tableRows = action.storages.map(storage => {
    const provider = storage.getProvider();
    return {
      id: storage.getId(),
      provider: providerNameByCode.get(provider) || provider,
      type: storage.getConfigurationtype() || 'storage',
      target: getStorageTarget(storage),
      status: storage.getStatus(),
      created: storage.getCreateddate()?.toString() || '',
    };
  });

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col bg-surface">
      <ComposedModal
        open={Boolean(pendingStorageAction)}
        onClose={() => setPendingStorageAction(null)}
        size="sm"
        danger={pendingStorageAction?.kind !== 'enable'}
      >
        <ModalHeader title={modalTitle} />
        <ModalBody>
          <p>{modalContent}</p>
        </ModalBody>
        <ModalFooter danger={pendingStorageAction?.kind !== 'enable'}>
          <Button
            kind="secondary"
            size="md"
            onClick={() => setPendingStorageAction(null)}
          >
            Cancel
          </Button>
          <Button
            kind={
              pendingStorageAction?.kind === 'enable' ? 'primary' : 'danger'
            }
            size="md"
            onClick={() => {
              if (!pendingStorageAction) return;
              if (pendingStorageAction.kind === 'delete') {
                deleteStorage(pendingStorageAction.storage.getId());
                return;
              }
              updateStorageEnabled(
                pendingStorageAction.storage,
                pendingStorageAction.kind === 'enable',
              );
            }}
          >
            {modalPrimaryLabel}
          </Button>
        </ModalFooter>
      </ComposedModal>

      <AssistantSettingsShellHeader
        title="Storage"
        href={`/deployment/assistant/${assistantId}/configure-storage`}
        onReload={get}
        reloadLabel="Reload storage"
        primaryAction={{
          label: 'Create new storage',
          onClick: () => navigation.goToCreateAssistantStorage(assistantId),
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
      ) : action.storages.length > 0 ? (
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
              const selectedStorage = selectedRows[0]
                ? action.storages.find(
                    storage => storage.getId() === selectedRows[0].id,
                  )
                : undefined;

              return (
                <>
                  <TableToolbar {...getToolbarProps()}>
                    {selectedStorage && (
                      <TableBatchActions
                        {...batchActionProps}
                        shouldShowBatchActions
                      >
                        <TableBatchAction
                          renderIcon={Edit}
                          iconDescription="Edit"
                          onClick={() => {
                            navigation.goToEditAssistantStorage(
                              assistantId,
                              selectedStorage.getId(),
                            );
                            batchActionProps.onCancel();
                          }}
                        >
                          Edit
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={Power}
                          iconDescription={
                            selectedStorage.getEnabled() ? 'Disable' : 'Enable'
                          }
                          onClick={() => {
                            setPendingStorageAction({
                              kind: selectedStorage.getEnabled()
                                ? 'disable'
                                : 'enable',
                              storage: selectedStorage,
                            });
                            batchActionProps.onCancel();
                          }}
                        >
                          {selectedStorage.getEnabled() ? 'Disable' : 'Enable'}
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={TrashCan}
                          iconDescription="Delete"
                          onClick={() => {
                            setPendingStorageAction({
                              kind: 'delete',
                              storage: selectedStorage,
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
                        placeholder="Search storage..."
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
                          const storage = action.storages.find(
                            candidate => candidate.getId() === tableRow.id,
                          );
                          if (!storage) return null;

                          const provider = storage.getProvider();
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
                                  {storage.getConfigurationtype() || 'storage'}
                                </Tag>
                              </TableCell>
                              <TableCell className="text-sm">
                                {getStorageTarget(storage)}
                              </TableCell>
                              <TableCell className="text-sm whitespace-nowrap">
                                <RecordStatusIndicator
                                  state={storage.getStatus()}
                                  textSize={14}
                                />
                              </TableCell>
                              <TableCell className="text-[13px] whitespace-nowrap">
                                {storage.getCreateddate()
                                  ? toHumanReadableDateTime(
                                      storage.getCreateddate()!,
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
                    className="shrink-0"
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
            icon={ObjectStorage}
            title="No Storage"
            subtitle="There are no assistant storage configurations found."
          />
        </div>
      )}
    </div>
  );
};
