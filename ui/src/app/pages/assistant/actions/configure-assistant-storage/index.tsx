import React, { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { Add, ObjectStorage, Renew } from '@carbon/icons-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { SectionLoader } from '@/app/components/loader/section-loader';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { CreateAssistantStorage } from './create-assistant-storage';
import { UpdateAssistantStorage } from './update-assistant-storage';
import { useAssistantStoragePageStore } from '@/app/pages/assistant/actions/store/use-storage-page-store';
import { STORAGE_PROVIDER } from '@/providers';
import { IconOnlyButton, PrimaryButton } from '@/app/components/carbon/button';
import { RecordStatusIndicator } from '@/app/components/carbon/record-status-indicator';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  ComposedModal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  OverflowMenu,
  OverflowMenuItem,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Tag,
} from '@carbon/react';
import { AssistantConfiguration, Metadata } from '@rapidaai/react';
import { Pagination } from '@/app/components/carbon/pagination';
import {
  ScrollableTableSection,
  TableSection,
} from '@/app/components/sections/table-section';

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
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [pendingStorageAction, setPendingStorageAction] =
    useState<StorageAction | null>(null);

  useEffect(() => {
    showLoader('block');
    get();
  }, [assistantId, projectId, token, authId, action.page, action.pageSize]);

  const get = () => {
    action.getAssistantStorage(
      assistantId,
      projectId,
      token,
      authId,
      error => {
        toast.error(error);
        hideLoader();
      },
      () => {
        hideLoader();
      },
    );
  };

  const deleteStorage = (storageId: string) => {
    showLoader('block');
    action.deleteAssistantStorage(
      assistantId,
      storageId,
      projectId,
      token,
      authId,
      error => {
        toast.error(error);
        hideLoader();
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
    showLoader('block');
    action.updateAssistantStorageEnabled(
      assistantId,
      storage,
      enabled,
      projectId,
      token,
      authId,
      error => {
        toast.error(error);
        hideLoader();
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

  const filteredStorages = searchTerm.trim()
    ? action.storages.filter(row =>
        [
          providerNameByCode.get(row.getProvider()) || row.getProvider(),
          row.getConfigurationtype(),
          getStorageTarget(row),
          row.getEnabled() ? 'enabled' : 'disabled',
        ]
          .join(' ')
          .toLowerCase()
          .includes(searchTerm.trim().toLowerCase()),
      )
    : action.storages;

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

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col flex-1">
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
            kind={pendingStorageAction?.kind === 'enable' ? 'primary' : 'danger'}
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

      <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div>
          <Breadcrumb noTrailingSlash className="mb-2">
            <BreadcrumbItem
              href={`/deployment/assistant/${assistantId}/overview`}
            >
              Assistant
            </BreadcrumbItem>
          </Breadcrumb>
          <h1 className="text-2xl font-light tracking-tight">Storage</h1>
        </div>
      </div>

      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search storage..."
            onChange={(e: any) => setSearchTerm(e.target?.value || '')}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={get}
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            onClick={() => navigation.goToCreateAssistantStorage(assistantId)}
          >
            Create new storage
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>

      <TableSection>
        {action.storages.length > 0 && filteredStorages.length > 0 ? (
          <>
            <ScrollableTableSection>
              <Table className="min-w-max">
                <TableHead>
                  <TableRow>
                    <TableHeader>Provider</TableHeader>
                    <TableHeader>Type</TableHeader>
                    <TableHeader>Target</TableHeader>
                    <TableHeader>Status</TableHeader>
                    <TableHeader>Created</TableHeader>
                    <TableHeader>Actions</TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredStorages.map(row => {
                    const provider = row.getProvider();
                    const providerName =
                      providerNameByCode.get(provider) || provider;
                    return (
                      <TableRow key={row.getId()}>
                        <TableCell className="text-sm">
                          {providerName}
                        </TableCell>
                        <TableCell className="text-sm">
                          <Tag type="blue" size="sm">
                            {row.getConfigurationtype() || 'storage'}
                          </Tag>
                        </TableCell>
                        <TableCell className="text-sm">
                          {getStorageTarget(row)}
                        </TableCell>
                        <TableCell className="text-sm whitespace-nowrap">
                          <RecordStatusIndicator
                            state={row.getStatus()}
                            textSize={14}
                          />
                        </TableCell>
                        <TableCell className="text-[13px] whitespace-nowrap">
                          {row.getCreateddate()
                            ? toHumanReadableDateTime(row.getCreateddate()!)
                            : '-'}
                        </TableCell>
                        <TableCell
                          className="text-sm"
                          onClick={e => e.stopPropagation()}
                        >
                          <OverflowMenu
                            size="sm"
                            flipped
                            aria-label="Storage actions"
                          >
                            <OverflowMenuItem
                              itemText="Edit"
                              onClick={() =>
                                navigation.goToEditAssistantStorage(
                                  assistantId,
                                  row.getId(),
                                )
                              }
                            />
                            <OverflowMenuItem
                              itemText={row.getEnabled() ? 'Disable' : 'Enable'}
                              onClick={() =>
                                setPendingStorageAction({
                                  kind: row.getEnabled() ? 'disable' : 'enable',
                                  storage: row,
                                })
                              }
                            />
                            <OverflowMenuItem
                              itemText="Delete"
                              isDelete
                              onClick={() =>
                                setPendingStorageAction({
                                  kind: 'delete',
                                  storage: row,
                                })
                              }
                            />
                          </OverflowMenu>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </ScrollableTableSection>
            <Pagination
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
        ) : action.storages.length > 0 ? (
          <EmptyState
            className="w-full"
            icon={ObjectStorage}
            title="No storage found"
            subtitle="No storage configuration matched your search."
          />
        ) : (
          <EmptyState
            className="w-full"
            icon={ObjectStorage}
            title="No Storage"
            subtitle="There are no assistant storage configurations found."
          />
        )}
      </TableSection>
    </div>
  );
};
