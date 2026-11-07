import { useAssistantProviderPageStore } from '@/stores/assistant';
import { useCredential } from '@/hooks/use-credential';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Assistant, GetAllAssistantProviderResponse } from '@rapidaai/react';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { ConfirmDialog } from '@/app/components/dialogs/shared';
import { Pagination } from '@/app/components/ui/primitives';
import { toHumanReadableDateTime } from '@/utils/date';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { auditActorLabel, createdAuditActor } from '@/utils/audit-actor';
import {
  Tag,
  Button,
  DataTable,
  DataTableSkeleton,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  TableBatchActions,
  TableBatchAction,
  TableSelectRow,
} from '@carbon/react';
import { Add, Copy, Checkmark, Rocket, Renew } from '@carbon/icons-react';
import IconIndicator from '@carbon/react/es/components/IconIndicator';

interface VersionProps {
  assistant: Assistant;
  onReload: () => void;
}

const headers = [
  { key: 'version', header: 'Version' },
  { key: 'type', header: 'Type' },
  { key: 'description', header: 'Description' },
  { key: 'status', header: 'Status' },
  { key: 'createdBy', header: 'Created By' },
  { key: 'createdDate', header: 'Date' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

function VersionId({ id }: { id: string }) {
  const [copied, setCopied] = useState(false);
  const version = `vrsn_${id}`;
  const copy = () => {
    navigator.clipboard.writeText(version);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <span className="inline-flex items-center gap-1 font-mono text-[13px] text-muted ">
      {version}
      <Button
        hasIconOnly
        renderIcon={copied ? Checkmark : Copy}
        iconDescription="Copy"
        kind="ghost"
        size="sm"
        onClick={copy}
        className="!min-h-0 !p-1"
      />
    </span>
  );
}

export function Version(props: VersionProps) {
  const [userId, token, projectId] = useCredential();
  const navigation = useGlobalNavigation();
  const assistantProviderAction = useAssistantProviderPageStore();
  const [isFetching, setIsFetching] = useState(true);
  const [deployingProviderId, setDeployingProviderId] = useState<string | null>(
    null,
  );
  const [pendingDeployment, setPendingDeployment] = useState<{
    assistantProvider: string;
    assistantProviderId: string;
  } | null>(null);

  useEffect(() => {
    setIsFetching(true);
    assistantProviderAction.onChangeAssistant(props.assistant);
    assistantProviderAction.getAssistantProviders(
      props.assistant.getId(),
      projectId,
      token,
      userId,
      (err: string) => {
        setIsFetching(false);
        toast.error(err);
      },
      data => {
        setIsFetching(false);
      },
    );
  }, [
    props.assistant.getId(),
    projectId,
    assistantProviderAction.page,
    assistantProviderAction.pageSize,
    assistantProviderAction.criteria,
  ]);

  const deployRevision = (
    assistantProvider: string,
    assistantProviderId: string,
  ) => {
    setDeployingProviderId(assistantProviderId);
    assistantProviderAction.onReleaseVersion(
      assistantProvider,
      assistantProviderId,
      projectId,
      token,
      userId,
      error => {
        setDeployingProviderId(null);
        toast.error(error);
      },
      e => {
        toast.success(
          'New version of assistant has been deployed successfully.',
        );
        assistantProviderAction.onChangeAssistant(e);
        props.onReload();
        setDeployingProviderId(null);
      },
    );
  };

  if (isFetching) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center overflow-hidden bg-surface">
        <DataTableSkeleton
          headers={skeletonHeaders}
          columnCount={skeletonHeaders.length}
          rowCount={5}
          showHeader={false}
          showToolbar
          size="lg"
          className="w-full"
        />
      </div>
    );
  }

  const getProviderData = (apm: any) => {
    const caseType = apm.getAssistantproviderCase();
    const Cases =
      GetAllAssistantProviderResponse.AssistantProvider.AssistantproviderCase;

    switch (caseType) {
      case Cases.ASSISTANTPROVIDERMODEL: {
        const m = apm.getAssistantprovidermodel();
        return {
          id: m?.getId()!,
          type: 'LLM',
          typeColor: 'blue' as const,
          description: m?.getDescription() || 'Initial assistant version',
          createdBy: auditActorLabel(createdAuditActor(m)),
          createdDate: m?.getCreateddate(),
          deployType: 'MODEL',
        };
      }
      case Cases.ASSISTANTPROVIDERAGENTKIT: {
        const a = apm.getAssistantprovideragentkit();
        return {
          id: a?.getId()!,
          type: 'AgentKit',
          typeColor: 'purple' as const,
          description: a?.getDescription() || 'Initial assistant version',
          createdBy: auditActorLabel(createdAuditActor(a)),
          createdDate: a?.getCreateddate(),
          deployType: 'AGENTKIT',
        };
      }
      case Cases.ASSISTANTPROVIDERWEBSOCKET: {
        const w = apm.getAssistantproviderwebsocket();
        return {
          id: w?.getId()!,
          type: 'WebSocket',
          typeColor: 'teal' as const,
          description: w?.getDescription() || 'Initial assistant version',
          createdBy: auditActorLabel(createdAuditActor(w)),
          createdDate: w?.getCreateddate(),
          deployType: 'WEBSOCKET',
        };
      }
      case Cases.ASSISTANTPROVIDERAGENTFLOW: {
        const a = apm.getAssistantprovideragentflow();
        return {
          id: a?.getId()!,
          type: 'Agentflow',
          typeColor: 'green' as const,
          description: a?.getDescription() || 'Initial assistant version',
          createdBy: auditActorLabel(createdAuditActor(a)),
          createdDate: a?.getCreateddate(),
          deployType: 'AGENTFLOW',
        };
      }
      default:
        return null;
    }
  };

  const refresh = () => {
    setIsFetching(true);
    assistantProviderAction.getAssistantProviders(
      props.assistant.getId(),
      projectId,
      token,
      userId,
      (err: string) => {
        setIsFetching(false);
        toast.error(err);
      },
      () => {
        setIsFetching(false);
      },
    );
  };

  const allRows = assistantProviderAction.assistantProviders
    .map(apm => ({ apm, data: getProviderData(apm) }))
    .filter(({ data }) => data !== null) as {
    apm: any;
    data: NonNullable<ReturnType<typeof getProviderData>>;
  }[];

  const goToCreateVersion = () => {
    const activeProviderId =
      assistantProviderAction.assistant?.getAssistantproviderid() ??
      props.assistant.getAssistantproviderid();
    const activeVersion = allRows.find(row => row.data.id === activeProviderId);

    if (
      activeVersion?.data.deployType === 'AGENTKIT' ||
      props.assistant.hasAssistantprovideragentkit()
    ) {
      navigation.goToCreateAssistantAgentKitVersion(props.assistant.getId());
      return;
    }

    if (
      activeVersion?.data.deployType === 'AGENTFLOW' ||
      props.assistant.hasAssistantprovideragentflow()
    ) {
      navigation.goToCreateAssistantAgentflowVersion(props.assistant.getId());
      return;
    }

    navigation.goToCreateAssistantVersion(props.assistant.getId());
  };

  const tableRows = allRows.map(({ data }) => {
    const isCurrent =
      assistantProviderAction.assistant?.getAssistantproviderid() === data.id;

    return {
      id: data.id,
      version: `vrsn_${data.id}`,
      type: data.type,
      description: data.description,
      status: isCurrent ? 'In use' : 'Available',
      createdBy: data.createdBy,
      createdDate: data.createdDate
        ? toHumanReadableDateTime(data.createdDate)
        : '—',
      disabled: isCurrent,
    };
  });

  return (
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
          const selectedVersion = selectedRows[0]
            ? allRows.find(row => row.data.id === selectedRows[0].id)
            : undefined;

          return (
            <>
              <TableToolbar {...getToolbarProps()}>
                {selectedVersion && (
                  <TableBatchActions
                    {...batchActionProps}
                    shouldShowBatchActions
                  >
                    <TableBatchAction
                      renderIcon={Rocket}
                      iconDescription="Deploy version"
                      onClick={() => {
                        setPendingDeployment({
                          assistantProvider: selectedVersion.data.deployType,
                          assistantProviderId: selectedVersion.data.id,
                        });
                      }}
                    >
                      Deploy version
                    </TableBatchAction>
                  </TableBatchActions>
                )}
                <TableToolbarContent>
                  <TableToolbarSearch
                    placeholder="Search versions..."
                    onChange={onInputChange}
                  />
                  <Button
                    hasIconOnly
                    renderIcon={Renew}
                    iconDescription="Refresh"
                    kind="ghost"
                    onClick={refresh}
                    tooltipPosition="bottom"
                  />
                  <Button renderIcon={Add} onClick={goToCreateVersion}>
                    Add new version
                  </Button>
                </TableToolbarContent>
              </TableToolbar>

              <div className="min-h-0 flex-1 overflow-auto">
                <Table {...getTableProps()}>
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
                      const versionRow = allRows.find(
                        row => row.data.id === tableRow.id,
                      );
                      if (!versionRow) return null;

                      const { data } = versionRow;
                      const isCurrent = Boolean(tableRow.disabled);
                      const isDeploying = deployingProviderId === data.id;

                      return (
                        <TableRow
                          {...getRowProps({ row: tableRow })}
                          key={tableRow.id}
                        >
                          <TableSelectRow
                            {...getSelectionProps({ row: tableRow })}
                          />
                          <TableCell className="text-sm">
                            <VersionId id={data.id} />
                          </TableCell>
                          <TableCell className="text-sm">
                            <Tag type={data.typeColor} size="sm">
                              {data.type}
                            </Tag>
                          </TableCell>
                          <TableCell className="text-sm">
                            {data.description}
                          </TableCell>
                          <TableCell className="text-sm">
                            {isCurrent ? (
                              <IconIndicator
                                kind="succeeded"
                                label="In use"
                                size={16}
                              />
                            ) : isDeploying ? (
                              <IconIndicator
                                kind="in-progress"
                                label="Deploying"
                                size={16}
                              />
                            ) : (
                              <IconIndicator
                                kind="incomplete"
                                label="Available"
                                size={16}
                              />
                            )}
                          </TableCell>
                          <TableCell className="text-sm">
                            {data.createdBy}
                          </TableCell>
                          <TableCell className="text-[13px] whitespace-nowrap">
                            {data.createdDate
                              ? toHumanReadableDateTime(data.createdDate)
                              : '—'}
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>

              <Pagination
                className="shrink-0"
                totalItems={assistantProviderAction.totalCount}
                page={assistantProviderAction.page}
                pageSize={assistantProviderAction.pageSize}
                pageSizes={[10, 20, 50]}
                onChange={({ page, pageSize }) => {
                  assistantProviderAction.setPage(page);
                  if (pageSize !== assistantProviderAction.pageSize)
                    assistantProviderAction.setPageSize(pageSize);
                }}
              />

              <ConfirmDialog
                showing={pendingDeployment !== null}
                type="info"
                title="Deploy version?"
                content={
                  pendingDeployment
                    ? `Deploying vrsn_${pendingDeployment.assistantProviderId} will replace the version currently in use.`
                    : ''
                }
                confirmText="Deploy"
                cancelText="Cancel"
                onConfirm={() => {
                  if (!pendingDeployment) return;
                  const { assistantProvider, assistantProviderId } =
                    pendingDeployment;
                  setPendingDeployment(null);
                  batchActionProps.onCancel();
                  deployRevision(assistantProvider, assistantProviderId);
                }}
                onCancel={() => setPendingDeployment(null)}
                onClose={() => setPendingDeployment(null)}
              />
            </>
          );
        }}
      </DataTable>
    </TableSection>
  );
}
