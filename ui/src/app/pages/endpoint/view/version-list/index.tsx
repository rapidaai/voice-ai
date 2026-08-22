import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { useEndpointProviderModelPageStore } from '@/hooks';
import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import React, { useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import {
  toHumanReadableDateTime,
  toHumanReadableRelativeTime,
} from '@/utils/date';
import { TableSection } from '@/app/components/sections/table-section';
import { Pagination } from '@/app/components/carbon/pagination';
import IconIndicator from '@carbon/react/es/components/IconIndicator';
import { auditActorLabel, createdAuditActor } from '@/utils/audit-actor';
import {
  Button,
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
  RadioButton,
} from '@carbon/react';
import { Copy, Checkmark, Rocket, Renew } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/carbon/empty-state';

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
    <span className="inline-flex items-center gap-1 font-mono text-[13px] text-gray-600 dark:text-gray-400">
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
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(
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

  const filteredVersions = searchTerm
    ? versions.filter(
        epm =>
          `vrsn_${epm.getId()}`
            .toLowerCase()
            .includes(searchTerm.toLowerCase()) ||
          (epm.getDescription() || '')
            .toLowerCase()
            .includes(searchTerm.toLowerCase()),
      )
    : versions;

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

  return (
    <TableSection>
      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={!!selectedVersionId}
          totalSelected={selectedVersionId ? 1 : 0}
          onCancel={() => setSelectedVersionId(null)}
          totalCount={filteredVersions.length}
        >
          <TableBatchAction
            renderIcon={Rocket}
            onClick={() => {
              if (selectedVersionId) {
                deployRevision(selectedVersionId);
                setSelectedVersionId(null);
              }
            }}
          >
            Deploy version
          </TableBatchAction>
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search versions..."
            onChange={(e: any) => setSearchTerm(e.target?.value || '')}
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
      <Table>
        <TableHead>
          <TableRow>
            <TableHeader className="!w-12" />
            {headers.map(h => (
              <TableHeader key={h.key}>{h.header}</TableHeader>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {filteredVersions.map((epm, idx) => {
            const isDeployed =
              endpointProviderAction.currentEndpoint?.getEndpointprovidermodelid() ===
              epm.getId();

            return (
              <TableRow
                key={idx}
                isSelected={selectedVersionId === epm.getId()}
                onClick={() =>
                  !isDeployed &&
                  setSelectedVersionId(
                    selectedVersionId === epm.getId() ? null : epm.getId(),
                  )
                }
                className={!isDeployed ? 'cursor-pointer' : ''}
              >
                <TableCell
                  className="!w-12 !pr-0"
                  onClick={e => e.stopPropagation()}
                >
                  <RadioButton
                    id={`ep-version-select-${epm.getId()}`}
                    name="ep-version-select"
                    labelText=""
                    hideLabel
                    checked={selectedVersionId === epm.getId()}
                    onChange={() =>
                      setSelectedVersionId(
                        selectedVersionId === epm.getId() ? null : epm.getId(),
                      )
                    }
                    disabled={isDeployed}
                  />
                </TableCell>
                <TableCell className="text-sm">
                  {epm.getDescription() || 'Initial endpoint version'}
                </TableCell>
                <TableCell className="text-sm">
                  <VersionId id={epm.getId()} />
                </TableCell>
                <TableCell className="text-sm">
                  {isDeployed ? (
                    <IconIndicator kind="succeeded" label="In use" size={16} />
                  ) : (
                    <IconIndicator
                      kind="incomplete"
                      label="Available"
                      size={16}
                    />
                  )}
                </TableCell>
                <TableCell className="text-sm">
                  {auditActorLabel(createdAuditActor(epm))}
                </TableCell>
                <TableCell className="text-[13px] whitespace-nowrap">
                  {epm.getCreateddate()
                    ? toHumanReadableDateTime(epm.getCreateddate()!)
                    : '—'}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      <Pagination
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
    </TableSection>
  );
}
