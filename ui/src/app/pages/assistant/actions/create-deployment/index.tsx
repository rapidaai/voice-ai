import { Helmet } from '@/app/components/helmet';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { IconOnlyButton } from '@/app/components/carbon/button';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  Renew,
  View,
  Edit,
  Launch,
  Copy,
  Checkmark,
  Microphone,
  VolumeUp,
  Deploy,
} from '@carbon/icons-react';
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { useParams } from 'react-router-dom';
import {
  Assistant,
  AssistantDefinition,
  ConnectionConfig,
  DisableAssistantApiDeployment,
  DisableAssistantDebuggerDeployment,
  DisableAssistantPhoneDeployment,
  DisableAssistantWebpluginDeployment,
  GetAssistant,
  GetAssistantDeploymentRequest,
  GetAssistantRequest,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { connectionConfig } from '@/configs';
import { useRapidaStore } from '@/hooks';
import { toHumanReadableDateTime } from '@/utils/date';
import { AssistantPhoneCallDeploymentDialog } from '@/app/components/base/modal/assistant-phone-call-deployment-modal';
import { AssistantDebugDeploymentDialog } from '@/app/components/base/modal/assistant-debug-deployment-modal';
import { AssistantWebWidgetlDeploymentDialog } from '@/app/components/base/modal/assistant-web-widget-deployment-modal';
import { AssistantApiDeploymentDialog } from '@/app/components/base/modal/assistant-api-deployment-modal';
import {
  AssistantDeploymentType,
  AssistantDeploymentVersionsModal,
} from '@/app/components/base/modal/assistant-deployment-versions-modal';
import SourceIndicator from '@/app/components/indicators/source';
import { RecordStatusIndicator } from '@/app/components/carbon/record-status-indicator';
import {
  OverflowMenu,
  OverflowMenuItem,
} from '@/app/components/carbon/overflow-menu';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  MenuButton,
  MenuItem,
  MenuItemDivider,
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
  Tag,
} from '@carbon/react';

type DeploymentType = 'debugger' | 'api' | 'web' | 'phone';

export const ConfigureAssistantDeploymentPage = () => {
  const { assistantId } = useParams();
  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const navi = useGlobalNavigation();
  const { token, authId, projectId } = useCurrentCredential();
  const { showLoader, hideLoader } = useRapidaStore();

  const [isExpanded, setIsExpanded] = useState(false);
  const [isApiExpanded, setIsApiExpanded] = useState(false);
  const [isPhoneExpanded, setIsPhoneExpanded] = useState(false);
  const [isWidgetExpanded, setIsWidgetExpanded] = useState(false);
  const [isVersionsOpen, setIsVersionsOpen] = useState(false);
  const [versionType, setVersionType] =
    useState<AssistantDeploymentType | null>(null);
  const [openActionMenuType, setOpenActionMenuType] =
    useState<DeploymentType | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [copiedVersion, setCopiedVersion] = useState<string | null>(null);
  const [selectedDeploymentType, setSelectedDeploymentType] =
    useState<DeploymentType | null>(null);

  const get = useCallback(
    (id: typeof assistantId) => {
      if (id) {
        showLoader('block');
        const request = new GetAssistantRequest();
        const assistantDef = new AssistantDefinition();
        assistantDef.setAssistantid(id);
        request.setAssistantdefinition(assistantDef);
        GetAssistant(
          connectionConfig,
          request,
          ConnectionConfig.WithDebugger({
            authorization: token,
            userId: authId,
            projectId: projectId,
          }),
        )
          .then(epmr => {
            hideLoader();
            if (epmr?.getSuccess()) {
              const a = epmr.getData();
              if (a) setAssistant(a);
            } else {
              const error = epmr?.getError();
              toast.error(
                error?.getHumanmessage() ??
                  'Unable to get your assistant. please try again later.',
              );
            }
          })
          .catch(() => hideLoader());
      }
    },
    [token, authId, projectId],
  );

  useEffect(() => {
    get(assistantId);
  }, [assistantId]);

  const deploymentCount = [
    assistant?.getApideployment(),
    assistant?.getWebplugindeployment(),
    assistant?.getDebuggerdeployment(),
    assistant?.getPhonedeployment(),
  ].filter(Boolean).length;

  const hasAnyDeployment = deploymentCount > 0;

  const rows: Array<{
    type: DeploymentType;
    source: string;
    name: string;
    version: string;
    status: string;
    sttProvider: string;
    ttsProvider: string;
    updated: string;
    onEdit: () => void;
    onDetails: () => void;
    onPreview?: () => void;
    onDisable: () => void;
  }> = [];

  const disableDeployment = useCallback(
    async (type: DeploymentType) => {
      if (!assistantId) return;
      const request = new GetAssistantDeploymentRequest();
      request.setAssistantid(assistantId);

      const auth = ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId,
      });

      const disableByType = {
        api: DisableAssistantApiDeployment,
        debugger: DisableAssistantDebuggerDeployment,
        phone: DisableAssistantPhoneDeployment,
        web: DisableAssistantWebpluginDeployment,
      } as const;

      const labelByType = {
        api: 'API',
        debugger: 'Debugger',
        phone: 'Phone Call',
        web: 'Web Widget',
      } as const;

      try {
        const response = await disableByType[type](
          connectionConfig,
          request,
          auth,
        );
        if (response?.getSuccess()) {
          toast.success(`${labelByType[type]} deployment disabled.`);
          setSelectedDeploymentType(null);
          get(assistantId);
          return;
        }
        toast.error(
          response?.getError?.()?.getHumanmessage?.() ||
            `Unable to disable ${labelByType[type]} deployment.`,
        );
      } catch {
        toast.error(`Unable to disable ${labelByType[type]} deployment.`);
      }
    },
    [assistantId, token, authId, projectId, get],
  );

  if (assistant?.hasDebuggerdeployment()) {
    const deployment = assistant.getDebuggerdeployment()!;
    rows.push({
      type: 'debugger',
      source: 'debugger',
      name: 'Debugger',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      status: getDeploymentStatus(deployment),
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () =>
        navi.goToEditDebugger(
          assistantId!,
          String(deployment.getId() || 'latest'),
        ),
      onDetails: () => setIsExpanded(true),
      onPreview: () => navi.goToAssistantPreview(assistantId!),
      onDisable: () => void disableDeployment('debugger'),
    });
  }

  if (assistant?.hasApideployment()) {
    const deployment = assistant.getApideployment()!;
    rows.push({
      type: 'api',
      source: 'sdk',
      name: 'SDK / API',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      status: getDeploymentStatus(deployment),
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () =>
        navi.goToEditApi(assistantId!, String(deployment.getId() || 'latest')),
      onDetails: () => setIsApiExpanded(true),
      onDisable: () => void disableDeployment('api'),
    });
  }

  if (assistant?.hasPhonedeployment()) {
    const deployment = assistant.getPhonedeployment()!;
    rows.push({
      type: 'phone',
      source: 'phone-call',
      name: 'Phone Call',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      status: getDeploymentStatus(deployment),
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () =>
        navi.goToEditCall(assistantId!, String(deployment.getId() || 'latest')),
      onDetails: () => setIsPhoneExpanded(true),
      onPreview: () => navi.goToAssistantPreviewCall(assistantId!),
      onDisable: () => void disableDeployment('phone'),
    });
  }

  if (assistant?.hasWebplugindeployment()) {
    const deployment = assistant.getWebplugindeployment()!;
    rows.push({
      type: 'web',
      source: 'web-plugin',
      name: 'Web Widget',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      status: getDeploymentStatus(deployment),
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () =>
        navi.goToEditWeb(assistantId!, String(deployment.getId() || 'latest')),
      onDetails: () => setIsWidgetExpanded(true),
      onDisable: () => void disableDeployment('web'),
    });
  }

  const copyVersion = useCallback((version: string) => {
    navigator.clipboard.writeText(version);
    setCopiedVersion(version);
    setTimeout(() => setCopiedVersion(null), 2000);
  }, []);

  const addDeploymentMenu = useMemo(
    () => (
      <MenuButton
        className="deployment-add-menu"
        label="Add deployment"
        size="md"
        kind="primary"
        menuBorder
        menuBackgroundToken="background"
      >
        <MenuItem
          label="Debugger"
          onClick={() => navi.goToConfigureDebugger(assistantId!)}
        />
        <MenuItemDivider />
        <MenuItem
          label="Web Widget"
          onClick={() => navi.goToConfigureWeb(assistantId!)}
        />
        <MenuItemDivider />
        <MenuItem
          label="SDK / API"
          onClick={() => navi.goToConfigureApi(assistantId!)}
        />
        <MenuItemDivider />
        <MenuItem
          label="Phone Call"
          onClick={() => navi.goToConfigureCall(assistantId!)}
        />
      </MenuButton>
    ),
    [assistantId, navi],
  );

  const normalizedSearch = searchTerm.trim().toLowerCase();
  const filteredRows = normalizedSearch
    ? rows.filter(row =>
        [
          row.name,
          row.version,
          row.status,
          row.sttProvider,
          row.ttsProvider,
          row.updated,
        ]
          .join(' ')
          .toLowerCase()
          .includes(normalizedSearch),
      )
    : rows;

  const selectedDeployment = filteredRows.find(
    row => row.type === selectedDeploymentType,
  );

  return (
    <div className="flex flex-col w-full flex-1 overflow-auto">
      {/* Modals */}
      {assistant?.getPhonedeployment() && (
        <AssistantPhoneCallDeploymentDialog
          modalOpen={isPhoneExpanded}
          setModalOpen={setIsPhoneExpanded}
          deployment={assistant.getPhonedeployment()!}
        />
      )}
      {assistant?.getDebuggerdeployment() && (
        <AssistantDebugDeploymentDialog
          modalOpen={isExpanded}
          setModalOpen={setIsExpanded}
          deployment={assistant.getDebuggerdeployment()!}
        />
      )}
      {assistant?.getWebplugindeployment() && (
        <AssistantWebWidgetlDeploymentDialog
          modalOpen={isWidgetExpanded}
          setModalOpen={setIsWidgetExpanded}
          deployment={assistant.getWebplugindeployment()!}
        />
      )}
      {assistant?.getApideployment() && (
        <AssistantApiDeploymentDialog
          modalOpen={isApiExpanded}
          setModalOpen={setIsApiExpanded}
          deployment={assistant.getApideployment()!}
        />
      )}
      {assistantId && (
        <AssistantDeploymentVersionsModal
          modalOpen={isVersionsOpen}
          setModalOpen={setIsVersionsOpen}
          assistantId={assistantId}
          deploymentType={versionType}
          authId={authId}
          token={token}
          projectId={projectId}
        />
      )}
      <Helmet title="Assistant deployment" />

      {/* Page header */}
      <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div>
          <Breadcrumb noTrailingSlash className="mb-2">
            <BreadcrumbItem
              href={`/deployment/assistant/${assistantId}/overview`}
            >
              Assistant
            </BreadcrumbItem>
          </Breadcrumb>
          <h1 className="text-2xl font-light tracking-tight">Deployments</h1>
        </div>
      </div>

      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={!!selectedDeployment}
          totalSelected={selectedDeployment ? 1 : 0}
          totalCount={filteredRows.length}
          onCancel={() => setSelectedDeploymentType(null)}
          className="[&_[class*=divider]]:hidden [&_.cds--btn]:transition-colors [&_.cds--btn:hover]:!bg-primary [&_.cds--btn:hover]:!text-white"
        >
          {selectedDeployment && (
            <>
              <TableBatchAction
                renderIcon={Edit}
                kind="ghost"
                onClick={() => {
                  selectedDeployment.onEdit();
                  setSelectedDeploymentType(null);
                }}
              >
                Edit deployment
              </TableBatchAction>
            </>
          )}
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search deployments"
            onChange={(e: any) => setSearchTerm(e.target?.value || '')}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={() => get(assistantId)}
          />
          {addDeploymentMenu}
        </TableToolbarContent>
      </TableToolbar>

      {hasAnyDeployment ? (
        <>
          {filteredRows.length > 0 ? (
            <div className="overflow-auto flex-1">
              <Table>
                <TableHead>
                  <TableRow>
                    <TableHeader className="!w-12" />
                    <TableHeader>Channel</TableHeader>
                    <TableHeader>Version</TableHeader>
                    <TableHeader>Status</TableHeader>
                    <TableHeader>STT Provider</TableHeader>
                    <TableHeader>TTS Provider</TableHeader>
                    <TableHeader>Date</TableHeader>
                    <TableHeader>Actions</TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredRows.map(row => (
                    <TableRow
                      key={row.type}
                      isSelected={selectedDeploymentType === row.type}
                      onClick={() =>
                        setSelectedDeploymentType(
                          selectedDeploymentType === row.type ? null : row.type,
                        )
                      }
                      className="cursor-pointer"
                    >
                      <TableCell
                        className="!w-12 !pr-0"
                        onClick={e => e.stopPropagation()}
                      >
                        <RadioButton
                          id={`deployment-select-${row.type}`}
                          name="deployment-select"
                          labelText=""
                          hideLabel
                          checked={selectedDeploymentType === row.type}
                          onChange={() =>
                            setSelectedDeploymentType(
                              selectedDeploymentType === row.type
                                ? null
                                : row.type,
                            )
                          }
                        />
                      </TableCell>
                      <TableCell className="text-sm">
                        <SourceIndicator source={row.source} />
                      </TableCell>
                      <TableCell className="font-mono text-[13px]">
                        {row.version !== '—' ? (
                          <span className="inline-flex items-center gap-1">
                            {row.version}
                            <Button
                              hasIconOnly
                              renderIcon={
                                copiedVersion === row.version ? Checkmark : Copy
                              }
                              iconDescription="Copy version id"
                              kind="ghost"
                              size="sm"
                              onClick={() => copyVersion(row.version)}
                              className="!min-h-0 !p-1"
                            />
                          </span>
                        ) : (
                          row.version
                        )}
                      </TableCell>
                      <TableCell className="text-sm">
                        <RecordStatusIndicator state={row.status} />
                      </TableCell>
                      <TableCell className="text-sm">
                        <AudioProviderTag
                          provider={row.sttProvider}
                          icon={<Microphone size={14} />}
                        />
                      </TableCell>
                      <TableCell className="text-sm">
                        <AudioProviderTag
                          provider={row.ttsProvider}
                          icon={<VolumeUp size={14} />}
                        />
                      </TableCell>
                      <TableCell className="text-[13px] whitespace-nowrap">
                        {row.updated}
                      </TableCell>
                      <TableCell
                        className="text-sm"
                        onClick={e => e.stopPropagation()}
                      >
                        <div className="flex items-center gap-0">
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={View}
                            iconDescription="View details"
                            onClick={row.onDetails}
                          />
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={Edit}
                            iconDescription="Edit deployment"
                            onClick={row.onEdit}
                          />
                          {row.onPreview && (
                            <IconOnlyButton
                              kind="ghost"
                              size="md"
                              renderIcon={Launch}
                              iconDescription="Preview deployment"
                              onClick={row.onPreview}
                            />
                          )}
                          <OverflowMenu
                            size="md"
                            flipped
                            iconDescription="Deployment actions"
                            open={openActionMenuType === row.type}
                            onOpen={() => setOpenActionMenuType(row.type)}
                            onClose={() => setOpenActionMenuType(null)}
                          >
                            <OverflowMenuItem
                              itemText="View all versions"
                              onClick={() => {
                                setOpenActionMenuType(null);
                                setVersionType(row.type);
                                setIsVersionsOpen(true);
                              }}
                            />
                            <OverflowMenuItem
                              itemText="Disable deployment"
                              isDelete
                              hasDivider
                              onClick={() => {
                                setOpenActionMenuType(null);
                                row.onDisable();
                              }}
                            />
                          </OverflowMenu>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <EmptyState
              icon={Deploy}
              title="No deployments found"
              subtitle="No deployment matched your search."
            />
          )}
        </>
      ) : (
        <EmptyState
          icon={Deploy}
          title="No deployments found"
          subtitle="Any assistant deployments you configure will be listed here."
        />
      )}
    </div>
  );
};

const providerLabels: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  google: 'Google',
  gemini: 'Gemini',
  azure: 'Azure',
  'azure-openai': 'Azure OpenAI',
  groq: 'Groq',
  mistral: 'Mistral',
  cohere: 'Cohere',
  deepseek: 'DeepSeek',
  deepgram: 'Deepgram',
  cartesia: 'Cartesia',
  elevenlabs: 'ElevenLabs',
  'custom-tts': 'Custom TTS',
  'custom-stt': 'Custom STT',
};

function getDeploymentStatus(
  deployment: { getStatus?: () => string } | null | undefined,
) {
  if (!deployment || typeof deployment.getStatus !== 'function') {
    return 'unknown';
  }
  return deployment.getStatus() || 'unknown';
}

function AudioProviderTag({
  provider,
  icon,
}: {
  provider?: string;
  icon: ReactNode;
}) {
  const key = provider?.toLowerCase() || '';
  const label =
    key && key !== '—'
      ? providerLabels[key] || provider || 'Unknown'
      : 'Unavailable';

  return (
    <Tag size="sm" type="cool-gray">
      <span className="flex items-center gap-1.5 leading-none">
        {icon}
        {label}
      </span>
    </Tag>
  );
}
