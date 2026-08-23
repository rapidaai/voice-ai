import { useAssistantProviderPageStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Assistant, GetAllAssistantProviderResponse } from '@rapidaai/react';
import { SectionLoader } from '@/app/components/loader/section-loader';
import { TableSection } from '@/app/components/sections/table-section';
import { Pagination } from '@/app/components/carbon/pagination';
import { toHumanReadableDateTime } from '@/utils/date';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { auditActorLabel, createdAuditActor } from '@/utils/audit-actor';
import {
  Tag,
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

export function Version(props: VersionProps) {
  const [userId, token, projectId] = useCredential();
  const navigation = useGlobalNavigation();
  const assistantProviderAction = useAssistantProviderPageStore();
  const [isFetching, setIsFetching] = useState(true);
  const [deployingProviderId, setDeployingProviderId] = useState<string | null>(
    null,
  );
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(
    null,
  );

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
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
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

  const filteredRows = searchTerm
    ? allRows.filter(
        ({ data }) =>
          `vrsn_${data.id}`.toLowerCase().includes(searchTerm.toLowerCase()) ||
          data.type.toLowerCase().includes(searchTerm.toLowerCase()) ||
          data.description.toLowerCase().includes(searchTerm.toLowerCase()),
      )
    : allRows;

  return (
    <TableSection>
      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={!!selectedVersionId}
          totalSelected={selectedVersionId ? 1 : 0}
          onCancel={() => setSelectedVersionId(null)}
          totalCount={filteredRows.length}
          className="[&_[class*=divider]]:hidden [&_.cds--btn]:transition-colors [&_.cds--btn:hover]:!bg-primary [&_.cds--btn:hover]:!text-white"
        >
          <TableBatchAction
            renderIcon={Rocket}
            onClick={() => {
              const row = filteredRows.find(
                r => r.data.id === selectedVersionId,
              );
              if (row) {
                deployRevision(row.data.deployType, row.data.id);
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
            onClick={refresh}
            tooltipPosition="bottom"
          />
          <Button renderIcon={Add} onClick={goToCreateVersion}>
            Add new version
          </Button>
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
          {filteredRows.map(({ data }, idx) => {
            const isCurrent =
              assistantProviderAction.assistant?.getAssistantproviderid() ===
              data.id;
            const isDeploying = deployingProviderId === data.id;

            return (
              <TableRow
                key={idx}
                isSelected={selectedVersionId === data.id}
                onClick={() =>
                  !isCurrent &&
                  setSelectedVersionId(
                    selectedVersionId === data.id ? null : data.id,
                  )
                }
                className={!isCurrent ? 'cursor-pointer' : ''}
              >
                <TableCell
                  className="!w-12 !pr-0"
                  onClick={e => e.stopPropagation()}
                >
                  <RadioButton
                    id={`version-select-${data.id}`}
                    name="version-select"
                    labelText=""
                    hideLabel
                    checked={selectedVersionId === data.id}
                    onChange={() =>
                      setSelectedVersionId(
                        selectedVersionId === data.id ? null : data.id,
                      )
                    }
                    disabled={isCurrent}
                  />
                </TableCell>
                <TableCell className="text-sm">
                  <VersionId id={data.id} />
                </TableCell>
                <TableCell className="text-sm">
                  <Tag type={data.typeColor} size="sm">
                    {data.type}
                  </Tag>
                </TableCell>
                <TableCell className="text-sm">{data.description}</TableCell>
                <TableCell className="text-sm">
                  {isCurrent ? (
                    <IconIndicator kind="succeeded" label="In use" size={16} />
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
                <TableCell className="text-sm">{data.createdBy}</TableCell>
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
      <Pagination
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
    </TableSection>
  );
}
