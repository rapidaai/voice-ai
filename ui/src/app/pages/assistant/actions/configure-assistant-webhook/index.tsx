import React, { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { SectionLoader } from '@/app/components/loader/section-loader';
import { CreateAssistantWebhook } from './create-assistant-webhook';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { UpdateAssistantWebhook } from '@/app/pages/assistant/actions/configure-assistant-webhook/update-assistant-webhook';
import { useAssistantWebhookPageStore } from '@/app/pages/assistant/actions/store/use-webhook-page-store';
import { IconOnlyButton, PrimaryButton } from '@/app/components/carbon/button';
import { RecordStatusIndicator } from '@/app/components/carbon/record-status-indicator';
import { UrlTableCell } from '@/app/components/carbon/url-table-cell';
import { Pagination } from '@/app/components/carbon/pagination';
import { Add, Renew, Webhook } from '@carbon/icons-react';
import { Tag } from '@carbon/react';
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
  Button,
  Breadcrumb,
  BreadcrumbItem,
  ComposedModal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  OverflowMenu,
  OverflowMenuItem,
} from '@carbon/react';
import {
  ScrollableTableSection,
  TableSection,
} from '@/app/components/sections/table-section';

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
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [pendingWebhookAction, setPendingWebhookAction] =
    useState<WebhookAction | null>(null);

  useEffect(() => {
    showLoader('block');
    get();
  }, [assistantId, projectId, token, authId, axtion.page, axtion.pageSize]);

  const get = () => {
    axtion.getAssistantWebhook(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        hideLoader();
      },
    );
  };

  const deleteAssistantWebhook = (assistantId: string, webhookId: string) => {
    showLoader('block');
    axtion.deleteAssistantWebhook(
      assistantId,
      webhookId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        toast.success('Webhook deleted successfully');
        setPendingWebhookAction(null);
        get();
      },
    );
  };

  const updateWebhookEnabled = (webhook: any, enabled: boolean) => {
    showLoader('block');
    axtion.updateAssistantWebhookEnabled(
      assistantId,
      webhook,
      enabled,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      () => {
        toast.success(`Webhook ${enabled ? 'enabled' : 'disabled'} successfully`);
        setPendingWebhookAction(null);
        get();
      },
    );
  };

  const filteredWebhooks = searchTerm.trim()
    ? axtion.webhooks.filter(row =>
        [
          getWebhookMethod(row),
          getWebhookUrl(row),
          getWebhookPriority(row),
          row.getEnabled() ? 'enabled' : 'disabled',
          row.getStatus(),
          ...getWebhookEvents(row),
        ]
          .join(' ')
          .toLowerCase()
          .includes(searchTerm.trim().toLowerCase()),
      )
    : axtion.webhooks;

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
            kind={pendingWebhookAction?.kind === 'enable' ? 'primary' : 'danger'}
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
      <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div>
          <Breadcrumb noTrailingSlash className="mb-2">
            <BreadcrumbItem
              href={`/deployment/assistant/${assistantId}/overview`}
            >
              Assistant
            </BreadcrumbItem>
          </Breadcrumb>
          <h1 className="text-2xl font-light tracking-tight">Webhooks</h1>
        </div>
      </div>
      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search webhooks..."
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
            onClick={() => navigation.goToCreateAssistantWebhook(assistantId)}
          >
            Create new webhook
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>
      <TableSection>
        {axtion.webhooks.length > 0 && filteredWebhooks.length > 0 ? (
          <>
            <ScrollableTableSection>
              <Table className="min-w-max">
                <TableHead>
                  <TableRow>
                    <TableHeader>Endpoint</TableHeader>
                    <TableHeader>Events</TableHeader>
                    <TableHeader>Retries</TableHeader>
                    <TableHeader>Timeout (s)</TableHeader>
                    <TableHeader>Priority</TableHeader>
                    <TableHeader>Status</TableHeader>
                    <TableHeader>Date</TableHeader>
                    <TableHeader>Actions</TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredWebhooks.map(row => {
                    return (
                      <TableRow key={row.getId()}>
                        <UrlTableCell
                          url={getWebhookUrl(row)}
                          prefix={
                            <span className="font-mono text-[13px] shrink-0">
                              {getWebhookMethod(row)}
                            </span>
                          }
                          maxWidthClassName="max-w-[560px]"
                        />
                        <TableCell className="text-sm">
                          <div className="flex flex-wrap gap-1">
                            {getWebhookEvents(row).map((event, index) => (
                              <Tag key={index} type="blue" size="sm">
                                {event}
                              </Tag>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell className="text-sm">
                          {getWebhookRetryCount(row)}
                        </TableCell>
                        <TableCell className="text-sm">
                          {getWebhookTimeoutSecond(row)}
                        </TableCell>
                        <TableCell className="text-sm">
                          {getWebhookPriority(row)}
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
                            : '—'}
                        </TableCell>
                        <TableCell
                          className="text-sm"
                          onClick={e => e.stopPropagation()}
                        >
                          <OverflowMenu
                            size="sm"
                            flipped
                            aria-label="Webhook actions"
                          >
                            <OverflowMenuItem
                              itemText="Edit"
                              onClick={() =>
                                navigation.goToEditAssistantWebhook(
                                  assistantId,
                                  row.getId(),
                                )
                              }
                            />
                            <OverflowMenuItem
                              itemText={row.getEnabled() ? 'Disable' : 'Enable'}
                              onClick={() =>
                                setPendingWebhookAction({
                                  kind: row.getEnabled() ? 'disable' : 'enable',
                                  webhook: row,
                                })
                              }
                            />
                            <OverflowMenuItem
                              itemText="Delete"
                              isDelete
                              onClick={() =>
                                setPendingWebhookAction({
                                  kind: 'delete',
                                  webhook: row,
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
        ) : axtion.webhooks.length > 0 ? (
          <EmptyState
            className="w-full"
            icon={Webhook}
            title="No webhooks found"
            subtitle="No webhook matched your search."
          />
        ) : (
          <EmptyState
            className="w-full"
            icon={Webhook}
            title="No Webhook"
            subtitle="There are no assistant webhook found."
          />
        )}
      </TableSection>
    </div>
  );
};
