import { FC, useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { useCurrentCredential } from '@/hooks/use-credential';
import { CreateAssistantWebhook } from './create-assistant-webhook';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/ui/feedback';
import { UpdateAssistantWebhook } from '@/app/pages/assistant/actions/configure-assistant-webhook/update-assistant-webhook';
import { useAssistantWebhookPageStore } from '@/stores/assistant/actions';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import { UrlTableCell } from '@/app/components/ui/table';
import { Pagination } from '@/app/components/ui/primitives';
import { Edit, Power, TrashCan, Webhook } from '@carbon/icons-react';
import {
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
  Button,
  ComposedModal,
  ModalBody,
  ModalFooter,
  ModalHeader,
} from '@carbon/react';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';
import { TableSection } from '@/app/components/layout/sections/table-section';

const getWebhookOptionMap = (row: any): Map<string, string> => {
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

const getWebhookMethod = (row: any): string =>
  getWebhookOptionMap(row).get('http_method') || '';

const getWebhookUrl = (row: any): string =>
  getWebhookOptionMap(row).get('http_url') || '';

const getWebhookEvents = (row: any): string[] =>
  parseStringList(getWebhookOptionMap(row).get('assistant_events'));

const parseStringList = (raw?: string): string[] => {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      return parsed.filter((item): item is string => typeof item === 'string');
    }
  } catch {}
  return [];
};

const getWebhookRetryCount = (row: any): number => {
  const value = Number(getWebhookOptionMap(row).get('max_retry_count'));
  return Number.isFinite(value) ? value : 0;
};

const getWebhookTimeoutSecond = (row: any): number => {
  const value = Number(getWebhookOptionMap(row).get('timeout_seconds'));
  return Number.isFinite(value) ? value : 0;
};

const getWebhookPriority = (row: any): number => {
  const value = Number(getWebhookOptionMap(row).get('execution_priority'));
  return Number.isFinite(value) ? value : 0;
};

const headers = [
  { key: 'endpoint', header: 'Endpoint' },
  { key: 'events', header: 'Events' },
  { key: 'retries', header: 'Retries' },
  { key: 'timeout', header: 'Timeout (s)' },
  { key: 'priority', header: 'Priority' },
  { key: 'status', header: 'Status' },
  { key: 'date', header: 'Date' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

type WebhookAction = {
  kind: 'enable' | 'disable' | 'delete';
  webhook: any;
};

export function ConfigureAssistantWebhookPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantWebhook assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantWebhookPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantWebhook assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantWebhookPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantWebhook assistantId={assistantId} />}</>
  );
}

const ConfigureAssistantWebhook: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const axtion = useAssistantWebhookPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const [isFetching, setIsFetching] = useState(true);
  const [pendingWebhookAction, setPendingWebhookAction] =
    useState<WebhookAction | null>(null);

  const get = useCallback(() => {
    setIsFetching(true);
    axtion.getAssistantWebhook(
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
  }, [assistantId, projectId, token, authId, axtion.getAssistantWebhook]);

  useEffect(() => {
    get();
  }, [get, axtion.page, axtion.pageSize]);

  const deleteAssistantWebhook = (assistantId: string, webhookId: string) => {
    setIsFetching(true);
    axtion.deleteAssistantWebhook(
      assistantId,
      webhookId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setIsFetching(false);
      },
      () => {
        toast.success('Webhook deleted successfully');
        setPendingWebhookAction(null);
        get();
      },
    );
  };

  const updateWebhookEnabled = (webhook: any, enabled: boolean) => {
    setIsFetching(true);
    axtion.updateAssistantWebhookEnabled(
      assistantId,
      webhook,
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
          `Webhook ${enabled ? 'enabled' : 'disabled'} successfully`,
        );
        setPendingWebhookAction(null);
        get();
      },
    );
  };

  const modalTitle =
    pendingWebhookAction?.kind === 'enable'
      ? 'Enable webhook?'
      : pendingWebhookAction?.kind === 'disable'
        ? 'Disable webhook?'
        : 'Delete webhook?';
  const modalContent =
    pendingWebhookAction?.kind === 'enable'
      ? 'This webhook will start receiving matching assistant events.'
      : pendingWebhookAction?.kind === 'disable'
        ? 'This webhook will stop receiving assistant events until it is enabled again.'
        : 'This webhook configuration will be removed from the assistant.';
  const modalPrimaryLabel =
    pendingWebhookAction?.kind === 'enable'
      ? 'Enable'
      : pendingWebhookAction?.kind === 'disable'
        ? 'Disable'
        : 'Delete';

  const tableRows = axtion.webhooks.map(webhook => ({
    id: webhook.getId(),
    endpoint: `${getWebhookMethod(webhook)} ${getWebhookUrl(webhook)}`,
    events: getWebhookEvents(webhook).join(', '),
    retries: String(getWebhookRetryCount(webhook)),
    timeout: String(getWebhookTimeoutSecond(webhook)),
    priority: String(getWebhookPriority(webhook)),
    status: webhook.getStatus(),
    date: webhook.getCreateddate()?.toString() || '',
  }));

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col bg-surface">
      <ComposedModal
        open={Boolean(pendingWebhookAction)}
        onClose={() => setPendingWebhookAction(null)}
        size="sm"
        danger={pendingWebhookAction?.kind !== 'enable'}
      >
        <ModalHeader title={modalTitle} />
        <ModalBody>
          <p>{modalContent}</p>
        </ModalBody>
        <ModalFooter danger={pendingWebhookAction?.kind !== 'enable'}>
          <Button
            kind="secondary"
            size="md"
            onClick={() => setPendingWebhookAction(null)}
          >
            Cancel
          </Button>
          <Button
            kind={
              pendingWebhookAction?.kind === 'enable' ? 'primary' : 'danger'
            }
            size="md"
            onClick={() => {
              if (!pendingWebhookAction) return;
              if (pendingWebhookAction.kind === 'delete') {
                deleteAssistantWebhook(
                  assistantId,
                  pendingWebhookAction.webhook.getId(),
                );
                return;
              }
              updateWebhookEnabled(
                pendingWebhookAction.webhook,
                pendingWebhookAction.kind === 'enable',
              );
            }}
          >
            {modalPrimaryLabel}
          </Button>
        </ModalFooter>
      </ComposedModal>
      <AssistantSettingsShellHeader
        title="Webhooks"
        href={`/deployment/assistant/${assistantId}/configure-webhook`}
        onReload={get}
        reloadLabel="Reload webhooks"
        primaryAction={{
          label: 'Create new webhook',
          onClick: () => navigation.goToCreateAssistantWebhook(assistantId),
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
      ) : axtion.webhooks.length > 0 ? (
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
              const selectedWebhook = selectedRows[0]
                ? axtion.webhooks.find(
                    webhook => webhook.getId() === selectedRows[0].id,
                  )
                : undefined;

              return (
                <>
                  <TableToolbar {...getToolbarProps()}>
                    {selectedWebhook && (
                      <TableBatchActions
                        {...batchActionProps}
                        shouldShowBatchActions
                      >
                        <TableBatchAction
                          renderIcon={Edit}
                          iconDescription="Edit"
                          onClick={() => {
                            navigation.goToEditAssistantWebhook(
                              assistantId,
                              selectedWebhook.getId(),
                            );
                            batchActionProps.onCancel();
                          }}
                        >
                          Edit
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={Power}
                          iconDescription={
                            selectedWebhook.getEnabled() ? 'Disable' : 'Enable'
                          }
                          onClick={() => {
                            setPendingWebhookAction({
                              kind: selectedWebhook.getEnabled()
                                ? 'disable'
                                : 'enable',
                              webhook: selectedWebhook,
                            });
                            batchActionProps.onCancel();
                          }}
                        >
                          {selectedWebhook.getEnabled() ? 'Disable' : 'Enable'}
                        </TableBatchAction>
                        <TableBatchAction
                          renderIcon={TrashCan}
                          iconDescription="Delete"
                          onClick={() => {
                            setPendingWebhookAction({
                              kind: 'delete',
                              webhook: selectedWebhook,
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
                        placeholder="Search webhooks..."
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
                          const webhook = axtion.webhooks.find(
                            candidate => candidate.getId() === tableRow.id,
                          );
                          if (!webhook) return null;

                          return (
                            <TableRow
                              {...getRowProps({ row: tableRow })}
                              key={tableRow.id}
                            >
                              <TableSelectRow
                                {...getSelectionProps({ row: tableRow })}
                              />
                              <UrlTableCell
                                url={getWebhookUrl(webhook)}
                                prefix={
                                  <span className="font-mono text-[13px] shrink-0">
                                    {getWebhookMethod(webhook)}
                                  </span>
                                }
                                maxWidthClassName="max-w-[560px]"
                              />
                              <TableCell className="text-sm">
                                <div className="flex flex-wrap gap-1">
                                  {getWebhookEvents(webhook).map(
                                    (event, index) => (
                                      <Tag key={index} type="blue" size="sm">
                                        {event}
                                      </Tag>
                                    ),
                                  )}
                                </div>
                              </TableCell>
                              <TableCell className="text-sm">
                                {getWebhookRetryCount(webhook)}
                              </TableCell>
                              <TableCell className="text-sm">
                                {getWebhookTimeoutSecond(webhook)}
                              </TableCell>
                              <TableCell className="text-sm">
                                {getWebhookPriority(webhook)}
                              </TableCell>
                              <TableCell className="text-sm whitespace-nowrap">
                                <RecordStatusIndicator
                                  state={webhook.getStatus()}
                                  textSize={14}
                                />
                              </TableCell>
                              <TableCell className="text-[13px] whitespace-nowrap">
                                {webhook.getCreateddate()
                                  ? toHumanReadableDateTime(
                                      webhook.getCreateddate()!,
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
                    onChange={({ page: newPage, pageSize: newSize }) => {
                      if (newSize !== axtion.pageSize) {
                        axtion.setPageSize(newSize);
                        return;
                      }
                      axtion.setPage(newPage);
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
            icon={Webhook}
            title="No Webhook"
            subtitle="There are no assistant webhook found."
          />
        </div>
      )}
    </div>
  );
};
