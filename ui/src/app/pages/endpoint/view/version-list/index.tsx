import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { useEndpointProviderModelPageStore } from '@/stores/endpoint';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { toHumanReadableDateTime } from '@/utils/date';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { ConfirmDialog } from '@/app/components/dialogs/shared';
import { Pagination } from '@/app/components/ui/primitives';
import IconIndicator from '@carbon/react/es/components/IconIndicator';
import { auditActorLabel, createdAuditActor } from '@/utils/audit-actor';
import {
  Button,
  DataTable,
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
import { Copy, Checkmark, Rocket, Renew } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/ui/feedback';

const headers = [
  { key: 'description', header: 'Description' },
  { key: 'version', header: 'Version' },
  { key: 'status', header: 'Status' },
  { key: 'createdBy', header: 'Created By' },
  { key: 'createdDate', header: 'Date' },
];

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

export function Version(props: {
  currentEndpoint: Endpoint;
  onReload: () => void;
}) {
  const { authId, token, projectId } = useCurrentCredential();
  const rapidaContext = useRapidaStore();
  const endpointProviderAction = useEndpointProviderModelPageStore();
  const [pendingDeploymentId, setPendingDeploymentId] = useState<string | null>(
    null,
  );

  const fetchVersions = () => {
    rapidaContext.showLoader();
    endpointProviderAction.onChangeCurrentEndpoint(props.currentEndpoint);
    endpointProviderAction.getEndpointProviderModels(
      props.currentEndpoint.getId(),
      projectId,
      token,
      authId,
      (err: string) => {
        rapidaContext.hideLoader();
        toast.error(err);
      },
      (_data: EndpointProviderModel[]) => {
        rapidaContext.hideLoader();
      },
    );
  };

  useEffect(() => {
    fetchVersions();
  }, [
    props.currentEndpoint,
    projectId,
    endpointProviderAction.page,
    endpointProviderAction.pageSize,
    endpointProviderAction.criteria,
  ]);

  const deployRevision = (endpointProviderModelId: string) => {
    rapidaContext.showLoader('overlay');
    endpointProviderAction.onReleaseVersion(
      endpointProviderModelId,
      projectId,
      token,
      authId,
      error => {
        rapidaContext.hideLoader();
        toast.error(error);
      },
      e => {
        toast.success(
          'New version of endpoint has been deployed successfully.',
        );
        endpointProviderAction.onChangeCurrentEndpoint(e);
        props.onReload();
        rapidaContext.hideLoader();
      },
    );
  };

  const versions = endpointProviderAction.endpointProviderModels;

  if (versions.length === 0 && !rapidaContext.loading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <EmptyState
          title="No versions found"
          subtitle="Create a new version of this endpoint to get started."
        />
      </div>
    );
  }

  const tableRows = versions.map(version => {
    const isDeployed =
      endpointProviderAction.currentEndpoint?.getEndpointprovidermodelid() ===
      version.getId();

    return {
      id: version.getId(),
      description: version.getDescription() || 'Initial endpoint version',
      version: `vrsn_${version.getId()}`,
      status: isDeployed ? 'In use' : 'Available',
      createdBy: auditActorLabel(createdAuditActor(version)),
      createdDate: version.getCreateddate()
        ? toHumanReadableDateTime(version.getCreateddate()!)
        : '—',
      disabled: isDeployed,
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
            ? versions.find(version => version.getId() === selectedRows[0].id)
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
                      iconDescription="Deploy Version"
                      onClick={() =>
                        setPendingDeploymentId(selectedVersion.getId())
                      }
                    >
                      Deploy Version
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
                    onClick={fetchVersions}
                    tooltipPosition="bottom"
                  />
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
                      const version = versions.find(
                        candidate => candidate.getId() === tableRow.id,
                      );
                      if (!version) return null;

                      const isDeployed = Boolean(tableRow.disabled);

                      return (
                        <TableRow
                          {...getRowProps({ row: tableRow })}
                          key={tableRow.id}
                        >
                          <TableSelectRow
                            {...getSelectionProps({ row: tableRow })}
                          />
                          <TableCell className="text-sm">
                            {version.getDescription() ||
                              'Initial endpoint version'}
                          </TableCell>
                          <TableCell className="text-sm">
                            <VersionId id={version.getId()} />
                          </TableCell>
                          <TableCell className="text-sm">
                            {isDeployed ? (
                              <IconIndicator
                                kind="succeeded"
                                label="In use"
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
                            {auditActorLabel(createdAuditActor(version))}
                          </TableCell>
                          <TableCell className="text-[13px] whitespace-nowrap">
                            {version.getCreateddate()
                              ? toHumanReadableDateTime(
                                  version.getCreateddate()!,
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
                className="shrink-0"
                totalItems={endpointProviderAction.totalCount}
                page={endpointProviderAction.page}
                pageSize={endpointProviderAction.pageSize}
                pageSizes={[10, 20, 50]}
                onChange={({ page, pageSize }) => {
                  endpointProviderAction.setPage(page);
                  if (pageSize !== endpointProviderAction.pageSize)
                    endpointProviderAction.setPageSize(pageSize);
                }}
              />

              <ConfirmDialog
                showing={pendingDeploymentId !== null}
                type="info"
                title="Deploy version?"
                content={
                  pendingDeploymentId
                    ? `Deploying vrsn_${pendingDeploymentId} will replace the version currently in use.`
                    : ''
                }
                confirmText="Deploy"
                cancelText="Cancel"
                onConfirm={() => {
                  if (!pendingDeploymentId) return;
                  const versionId = pendingDeploymentId;
                  setPendingDeploymentId(null);
                  batchActionProps.onCancel();
                  deployRevision(versionId);
                }}
                onCancel={() => setPendingDeploymentId(null)}
                onClose={() => setPendingDeploymentId(null)}
              />
            </>
          );
        }}
      </DataTable>
    </TableSection>
  );
}
