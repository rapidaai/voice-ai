import { Helmet } from '@/app/components/app-shell/helmet';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  Api,
  ApplicationWeb,
  ArrowUpRight,
  Debug,
  Phone,
  Deploy,
} from '@carbon/icons-react';
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Assistant } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import { toHumanReadableDateTime } from '@/utils/date';
import { AssistantPhoneCallDeploymentDialog } from '@/app/components/dialogs/assistant';
import { AssistantDebugDeploymentDialog } from '@/app/components/dialogs/assistant';
import { AssistantWebWidgetlDeploymentDialog } from '@/app/components/dialogs/assistant';
import { AssistantApiDeploymentDialog } from '@/app/components/dialogs/assistant';
import {
  AssistantDeploymentType,
  AssistantDeploymentVersionsModal,
} from '@/app/components/dialogs/assistant';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import { VersionIndicator } from '@/app/components/domain/indicators/version';
import { HelpToggletip } from '@/app/components/domain/providers/help-label';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';
import {
  Modal,
  ModalBody,
  ModalHeader,
  OverflowMenu,
  OverflowMenuItem,
  Text,
  typographyClassNames,
} from '@/app/components/ui/primitives';
import {
  ClickableTile,
  DataTable,
  Table,
  TableBody,
  TableCell,
  TableExpandedRow,
  TableExpandHeader,
  TableExpandRow,
  TableHead,
  TableHeader,
  TableRow,
} from '@carbon/react';
import {
  disableAssistantDeploymentByType,
  getAssistantById,
} from '@/clients/assistant.client';

type DeploymentType = 'debugger' | 'api' | 'web' | 'phone';
type SupportedDeployment =
  | NonNullable<ReturnType<Assistant['getDebuggerdeployment']>>
  | NonNullable<ReturnType<Assistant['getApideployment']>>
  | NonNullable<ReturnType<Assistant['getPhonedeployment']>>
  | NonNullable<ReturnType<Assistant['getWebplugindeployment']>>;
const NOT_AVAILABLE = 'Not available';
const deploymentTableHeaders = [
  { key: 'name', header: 'Deployment' },
  { key: 'version', header: 'Version' },
  { key: 'sttProvider', header: 'Speech to text' },
  { key: 'ttsProvider', header: 'Text to speech' },
  { key: 'status', header: 'Status' },
  { key: 'created', header: 'Created' },
  { key: 'actions', header: 'Actions' },
];

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
  const [isAddDeploymentOpen, setIsAddDeploymentOpen] = useState(false);
  const [isVersionsOpen, setIsVersionsOpen] = useState(false);
  const [versionType, setVersionType] =
    useState<AssistantDeploymentType | null>(null);

  const get = useCallback(
    (id: typeof assistantId) => {
      if (id) {
        showLoader('block');
        getAssistantById({
          assistantId: id,
          auth: { projectId, token, userId: authId },
        })
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
    [authId, hideLoader, projectId, showLoader, token],
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
    name: string;
    versionId: string | null;
    status: string;
    sttProvider: string;
    ttsProvider: string;
    created: string;
    assistantId: string;
    lastUpdated: string;
    maxSessionDuration: string;
    idleTimeout: string;
    greetingInterruptible: string;
    unclearInputTimeout: string;
    onEdit: () => void;
    onDetails: () => void;
    onPreview?: () => void;
    onDisable: () => void;
  }> = [];

  const disableDeployment = useCallback(
    async (type: DeploymentType) => {
      if (!assistantId) return;

      const labelByType = {
        api: 'API',
        debugger: 'Debugger',
        phone: 'Phone Call',
        web: 'Web Widget',
      } as const;

      try {
        const response = await disableAssistantDeploymentByType({
          assistantId,
          deploymentType: type,
          auth: { projectId, token, userId: authId },
        });
        if (response?.getSuccess()) {
          toast.success(`${labelByType[type]} deployment disabled.`);
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
      name: 'Debugger',
      versionId: deployment.getId() ? String(deployment.getId()) : null,
      status: getDeploymentStatus(deployment),
      sttProvider:
        deployment.getInputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      ttsProvider:
        deployment.getOutputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      created: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : NOT_AVAILABLE,
      ...getGeneralDeploymentDetails(deployment),
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
      name: 'SDK / API',
      versionId: deployment.getId() ? String(deployment.getId()) : null,
      status: getDeploymentStatus(deployment),
      sttProvider:
        deployment.getInputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      ttsProvider:
        deployment.getOutputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      created: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : NOT_AVAILABLE,
      ...getGeneralDeploymentDetails(deployment),
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
      name: 'Phone Call',
      versionId: deployment.getId() ? String(deployment.getId()) : null,
      status: getDeploymentStatus(deployment),
      sttProvider:
        deployment.getInputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      ttsProvider:
        deployment.getOutputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      created: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : NOT_AVAILABLE,
      ...getGeneralDeploymentDetails(deployment),
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
      name: 'Web Widget',
      versionId: deployment.getId() ? String(deployment.getId()) : null,
      status: getDeploymentStatus(deployment),
      sttProvider:
        deployment.getInputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      ttsProvider:
        deployment.getOutputaudio()?.getAudioprovider() || NOT_AVAILABLE,
      created: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : NOT_AVAILABLE,
      ...getGeneralDeploymentDetails(deployment),
      onEdit: () =>
        navi.goToEditWeb(assistantId!, String(deployment.getId() || 'latest')),
      onDetails: () => setIsWidgetExpanded(true),
      onDisable: () => void disableDeployment('web'),
    });
  }

  const deploymentRowsByType = new Map(rows.map(row => [row.type, row]));
  const deploymentTableRows = rows.map(row => ({
    id: row.type,
    name: row.name,
    version: row.versionId || NOT_AVAILABLE,
    sttProvider: getAudioProviderLabel(row.sttProvider),
    ttsProvider: getAudioProviderLabel(row.ttsProvider),
    status: row.status,
    created: row.created,
    actions: '',
  }));

  const openDeploymentSetup = (type: DeploymentType) => {
    setIsAddDeploymentOpen(false);
    if (type === 'debugger') navi.goToConfigureDebugger(assistantId!);
    if (type === 'web') navi.goToConfigureWeb(assistantId!);
    if (type === 'api') navi.goToConfigureApi(assistantId!);
    if (type === 'phone') navi.goToConfigureCall(assistantId!);
  };

  return (
    <div className="flex w-full flex-1 flex-col overflow-auto bg-surface">
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
      <AddDeploymentDialog
        open={isAddDeploymentOpen}
        onClose={() => setIsAddDeploymentOpen(false)}
        onSelect={openDeploymentSetup}
      />
      <Helmet title="Assistant deployment" />

      <AssistantSettingsShellHeader
        title="Deployments"
        href={`/deployment/assistant/${assistantId}/deployment/`}
        onReload={() => get(assistantId)}
        reloadLabel="Reload deployments"
        primaryAction={{
          label: 'Add deployment',
          onClick: () => setIsAddDeploymentOpen(true),
        }}
      />

      <div className="flex flex-1 flex-col bg-surface">
        <div className="flex w-full flex-1 flex-col">
          {hasAnyDeployment ? (
            <div
              aria-label="Deployment channels"
              className="no-scrollbar w-full overflow-x-auto"
            >
              <DataTable
                rows={deploymentTableRows}
                headers={deploymentTableHeaders}
              >
                {({
                  rows: tableRows,
                  headers,
                  getTableProps,
                  getHeaderProps,
                  getRowProps,
                  getExpandHeaderProps,
                  getExpandedRowProps,
                }) => (
                  <Table
                    {...getTableProps()}
                    size="lg"
                    className="min-w-[1240px]"
                  >
                    <TableHead>
                      <TableRow>
                        <TableExpandHeader
                          {...getExpandHeaderProps()}
                          id="deployment-expand"
                          aria-label="Expand all deployment details"
                        />
                        {headers.map(header => (
                          <TableHeader
                            {...getHeaderProps({ header })}
                            key={header.key}
                            className={getDeploymentHeaderClassName(header.key)}
                          >
                            {header.key === 'actions' ? (
                              <span className="flex w-full justify-end">
                                {header.header}
                              </span>
                            ) : (
                              header.header
                            )}
                          </TableHeader>
                        ))}
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {tableRows.map(tableRow => {
                        const row = deploymentRowsByType.get(
                          tableRow.id as DeploymentType,
                        );
                        if (!row) return null;

                        const expandedRowId = `${row.type}-deployment-details`;

                        return [
                          <TableExpandRow
                            {...getRowProps({ row: tableRow })}
                            key={tableRow.id}
                            aria-label={`${tableRow.isExpanded ? 'Collapse' : 'Expand'} ${row.name} deployment details`}
                            aria-controls={expandedRowId}
                            expandHeader="deployment-expand"
                          >
                            <TableCell>
                              <Text as="h3" variant="heading-compact-01">
                                {row.name}
                              </Text>
                            </TableCell>
                            <TableCell>
                              {row.versionId ? (
                                <VersionIndicator id={row.versionId} />
                              ) : (
                                <Text
                                  as="span"
                                  variant="body-compact-01"
                                  className="text-[var(--cds-text-secondary)]"
                                >
                                  {NOT_AVAILABLE}
                                </Text>
                              )}
                            </TableCell>
                            <TableCell>
                              <Text as="span" variant="body-compact-01">
                                {getAudioProviderLabel(row.sttProvider)}
                              </Text>
                            </TableCell>
                            <TableCell>
                              <Text as="span" variant="body-compact-01">
                                {getAudioProviderLabel(row.ttsProvider)}
                              </Text>
                            </TableCell>
                            <TableCell>
                              <RecordStatusIndicator state={row.status} />
                            </TableCell>
                            <TableCell>
                              <Text
                                as="span"
                                variant="body-compact-01"
                                className="whitespace-nowrap text-[var(--cds-text-secondary)]"
                              >
                                {row.created}
                              </Text>
                            </TableCell>
                            <TableCell className="w-20 min-w-20">
                              <div className="flex w-full items-center justify-end">
                                <OverflowMenu
                                  size="md"
                                  flipped
                                  iconDescription={`Actions for ${row.name}`}
                                >
                                  <OverflowMenuItem
                                    itemText="View deployment details"
                                    onClick={row.onDetails}
                                  />
                                  <OverflowMenuItem
                                    itemText="Edit deployment"
                                    onClick={row.onEdit}
                                  />
                                  {row.onPreview && (
                                    <OverflowMenuItem
                                      itemText="Preview deployment"
                                      onClick={row.onPreview}
                                    />
                                  )}
                                  <OverflowMenuItem
                                    itemText="View all versions"
                                    onClick={() => {
                                      setVersionType(row.type);
                                      setIsVersionsOpen(true);
                                    }}
                                  />
                                  <OverflowMenuItem
                                    itemText="Disable deployment"
                                    isDelete
                                    hasDivider
                                    onClick={row.onDisable}
                                  />
                                </OverflowMenu>
                              </div>
                            </TableCell>
                          </TableExpandRow>,
                          tableRow.isExpanded ? (
                            <TableExpandedRow
                              {...getExpandedRowProps({ row: tableRow })}
                              id={expandedRowId}
                              colSpan={8}
                              key={`${tableRow.id}-details`}
                            >
                              <div className="grid grid-cols-1 gap-8 px-6 py-6 md:grid-cols-2">
                                <section
                                  aria-labelledby={`${row.type}-session-title`}
                                >
                                  <Text
                                    as="h4"
                                    id={`${row.type}-session-title`}
                                    variant="heading-compact-01"
                                  >
                                    Session behavior
                                  </Text>
                                  <dl className="mt-4 grid grid-cols-1 gap-5 sm:grid-cols-2">
                                    <DeploymentDetail
                                      label="Maximum session duration"
                                      value={row.maxSessionDuration}
                                      helpText="The maximum time a conversation can remain active before it ends automatically."
                                    />
                                    <DeploymentDetail
                                      label="Idle timeout"
                                      value={row.idleTimeout}
                                      helpText="How long the assistant waits without user activity before applying its idle timeout behavior."
                                    />
                                    <DeploymentDetail
                                      label="Greeting interruptible"
                                      value={row.greetingInterruptible}
                                      helpText="Whether a user can speak while the assistant is playing its greeting."
                                    />
                                    <DeploymentDetail
                                      label="Unclear input timeout"
                                      value={row.unclearInputTimeout}
                                      helpText="How long the assistant waits for understandable input before using its unclear-input response."
                                    />
                                  </dl>
                                </section>
                                <section
                                  aria-labelledby={`${row.type}-general-title`}
                                >
                                  <Text
                                    as="h4"
                                    id={`${row.type}-general-title`}
                                    variant="heading-compact-01"
                                  >
                                    General information
                                  </Text>
                                  <dl className="mt-4 grid grid-cols-1 gap-5 sm:grid-cols-2 md:grid-cols-1">
                                    <DeploymentDetail
                                      label="Assistant ID"
                                      value={row.assistantId}
                                    />
                                    <DeploymentDetail
                                      label="Created"
                                      value={row.created}
                                    />
                                    <DeploymentDetail
                                      label="Last updated"
                                      value={row.lastUpdated}
                                    />
                                  </dl>
                                </section>
                              </div>
                            </TableExpandedRow>
                          ) : null,
                        ];
                      })}
                    </TableBody>
                  </Table>
                )}
              </DataTable>
            </div>
          ) : (
            <DeploymentEmptyState onSelect={openDeploymentSetup} />
          )}
        </div>
      </div>
    </div>
  );
};

const addDeploymentOptions: Array<{
  type: DeploymentType;
  title: string;
  description: string;
  icon: typeof Debug;
}> = [
  {
    type: 'web',
    title: 'Web Widget',
    description: 'Embed a voice experience on your website with a widget.',
    icon: ApplicationWeb,
  },
  {
    type: 'api',
    title: 'SDK / API',
    description:
      'Integrate the assistant into your application using an SDK or real-time API.',
    icon: Api,
  },
  {
    type: 'phone',
    title: 'Phone Call',
    description:
      'Connect the assistant to a phone number for live voice conversations.',
    icon: Phone,
  },
  {
    type: 'debugger',
    title: 'Debugger',
    description: 'Test voice conversations in your browser before deployment.',
    icon: Debug,
  },
];

function DeploymentEmptyState({
  onSelect,
}: {
  onSelect: (type: DeploymentType) => void;
}) {
  return (
    <section
      aria-labelledby="deployment-empty-state-title"
      className="flex flex-1 flex-col items-center justify-center px-4 py-12 text-center"
    >
      <Deploy size={32} className="text-[var(--cds-icon-secondary)]" />
      <Text
        as="h2"
        id="deployment-empty-state-title"
        variant="heading-04"
        className="mt-6 text-[var(--cds-text-primary)]"
      >
        Deployment
      </Text>
      <Text
        as="p"
        variant="body-02"
        className="mt-3 max-w-2xl text-[var(--cds-text-secondary)]"
      >
        Choose how you want to test or deliver your assistant.
      </Text>

      <div className="mt-8 grid w-full max-w-3xl grid-cols-1 gap-px border border-[var(--cds-border-subtle-01)] bg-[var(--cds-border-subtle-01)] text-left md:grid-cols-2">
        {addDeploymentOptions.map(option => {
          const Icon = option.icon;
          return (
            <ClickableTile
              key={option.type}
              onClick={() => onSelect(option.type)}
              className="group !relative !flex !min-h-40 !flex-col !p-6"
            >
              <ArrowUpRight
                aria-hidden
                size={20}
                className="absolute top-6 right-6 opacity-0 transition-opacity duration-150 group-hover:opacity-100 group-focus:opacity-100"
              />
              <Icon size={24} className="text-[var(--cds-icon-secondary)]" />
              <Text
                as="h3"
                variant="heading-03"
                className="mt-6 text-[var(--cds-text-primary)]"
              >
                {option.title}
              </Text>
              <Text
                as="p"
                variant="body-01"
                className="mt-2 text-[var(--cds-text-secondary)]"
              >
                {option.description}
              </Text>
            </ClickableTile>
          );
        })}
      </div>
    </section>
  );
}

function AddDeploymentDialog({
  open,
  onClose,
  onSelect,
}: {
  open: boolean;
  onClose: () => void;
  onSelect: (type: DeploymentType) => void;
}) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      size="lg"
      containerClassName="!max-w-[1120px]"
    >
      <ModalHeader
        label="Deployment"
        labelClassName={typographyClassNames['label-01']}
        title="Choose a deployment channel"
        titleClassName={typographyClassNames['heading-03']}
        onClose={onClose}
      />
      <ModalBody>
        <div className="grid grid-cols-1 gap-2 md:grid-cols-4">
          {addDeploymentOptions.map(option => {
            const Icon = option.icon;
            return (
              <ClickableTile
                key={option.type}
                onClick={() => onSelect(option.type)}
                className="group !flex !min-h-[220px] !flex-col !rounded-none !border !border-gray-200 !bg-gray-50 !p-0 transition-colors hover:!border-primary dark:!border-gray-700 dark:!bg-gray-800 dark:hover:!border-primary"
              >
                <div className="flex flex-1 flex-col p-5">
                  <div className="flex size-12 shrink-0 items-center justify-center bg-primary/10 text-primary dark:bg-primary/10">
                    <Icon size={24} />
                  </div>
                  <Text
                    as="h3"
                    variant="body-02"
                    className="mt-4 text-gray-900 dark:text-white"
                  >
                    {option.title}
                  </Text>
                  <Text
                    as="p"
                    variant="body-01"
                    className="mt-1 flex-1 text-gray-600 dark:text-gray-300"
                  >
                    {option.description}
                  </Text>
                </div>
              </ClickableTile>
            );
          })}
        </div>
      </ModalBody>
    </Modal>
  );
}

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

function getGeneralDeploymentDetails(deployment: SupportedDeployment) {
  return {
    assistantId: deployment.getAssistantid() || NOT_AVAILABLE,
    lastUpdated: deployment.getUpdateddate()
      ? toHumanReadableDateTime(deployment.getUpdateddate()!)
      : NOT_AVAILABLE,
    maxSessionDuration: deployment.getMaxsessionduration() || NOT_AVAILABLE,
    idleTimeout: deployment.getIdealtimeout() || NOT_AVAILABLE,
    greetingInterruptible: deployment.hasGreetinginterruptible()
      ? deployment.getGreetinginterruptible()
        ? 'Yes'
        : 'No'
      : NOT_AVAILABLE,
    unclearInputTimeout: deployment.hasUnclearinputtimeout()
      ? `${deployment.getUnclearinputtimeout()} seconds`
      : NOT_AVAILABLE,
  };
}

function getAudioProviderLabel(provider?: string) {
  const key = provider?.toLowerCase() || '';
  return key && key !== NOT_AVAILABLE.toLowerCase()
    ? providerLabels[key] || provider || 'Unknown'
    : NOT_AVAILABLE;
}

function getDeploymentHeaderClassName(key: string) {
  const classNames: Record<string, string> = {
    name: 'min-w-52',
    version: 'min-w-64',
    sttProvider: 'min-w-44',
    ttsProvider: 'min-w-44',
    status: 'w-32 min-w-32',
    created: 'min-w-52',
    actions: 'w-20 min-w-20',
  };
  return classNames[key];
}

function DeploymentDetail({
  label,
  value,
  helpText,
}: {
  label: string;
  value: string;
  helpText?: string;
}) {
  return (
    <div>
      <Text
        as="dt"
        variant="label-01"
        className="text-[var(--cds-text-secondary)]"
      >
        <span className="inline-flex items-center gap-1">
          {label}
          <HelpToggletip label={label} helpText={helpText} />
        </span>
      </Text>
      <Text
        as="dd"
        variant="body-01"
        className="mt-1 text-[var(--cds-text-primary)]"
      >
        {value || NOT_AVAILABLE}
      </Text>
    </div>
  );
}
