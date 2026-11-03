import type {
  AssistantDebuggerDeployment,
  DeploymentAudioProvider,
} from '@rapidaai/react';
import {
  Accordion,
  AccordionItem,
  ButtonSet,
  StructuredListBody,
  StructuredListCell,
  StructuredListRow,
  StructuredListWrapper,
} from '@carbon/react';
import {
  Chat,
  Edit,
  Launch,
  VoiceActivate,
  VolumeUp,
  type CarbonIconType,
} from '@carbon/icons-react';
import type { ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { RightSideModal } from '@/app/components/dialogs/shared';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import {
  PrimaryButton,
  SecondaryButton,
  type ModalProps,
} from '@/app/components/ui/primitives';
import {
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_IDLE_PROMPT_COUNT,
  DEFAULT_MAX_SESSION_DURATION,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';

interface AssistantDebugDeploymentDialogProps extends ModalProps {
  deployment: AssistantDebuggerDeployment;
}

interface DebuggerConfigurationOption {
  id: string;
  key: string;
  label: string;
  value: ReactNode;
}

export function AssistantDebugDeploymentDialog({
  deployment,
  modalOpen,
  setModalOpen,
}: AssistantDebugDeploymentDialogProps) {
  const navigation = useGlobalNavigation();
  const assistantId = deployment.getAssistantid();
  const deploymentId = deployment.getId();
  const inputAudio = deployment.getInputaudio();
  const outputAudio = deployment.getOutputaudio();
  const savedMaxSessionDuration = deployment.getMaxsessionduration();
  const maxSessionDuration =
    !savedMaxSessionDuration || savedMaxSessionDuration === '0'
      ? DEFAULT_MAX_SESSION_DURATION
      : savedMaxSessionDuration;
  const idleTimeout = getEffectiveIdleTimeout(deployment.getIdealtimeout());
  const idlePromptCount =
    deployment.getIdealtimeoutbackoff() || DEFAULT_IDLE_PROMPT_COUNT;
  const greetingInterruptible = deployment.hasGreetinginterruptible()
    ? deployment.getGreetinginterruptible()
    : true;
  const unclearInputTimeout = deployment.hasUnclearinputtimeout?.()
    ? deployment.getUnclearinputtimeout()
    : Number(DEFAULT_UNCLEAR_INPUT_TIMEOUT);

  const editConfiguration = () => {
    setModalOpen(false);
    navigation.goToEditDebugger(assistantId, deploymentId);
  };

  const openDebugger = () => {
    setModalOpen(false);
    navigation.goToAssistantPreview(assistantId);
  };

  const generalConfiguration: DebuggerConfigurationOption[] = [
    {
      id: 'status',
      key: 'status',
      label: 'Status',
      value: (
        <span className="flex justify-end">
          <RecordStatusIndicator state={deployment.getStatus()} />
        </span>
      ),
    },
    {
      id: 'greeting',
      key: 'greeting',
      label: 'Greeting',
      value: deployment.getGreeting() || '—',
    },
    {
      id: 'error-response',
      key: 'mistake',
      label: 'Error response',
      value: deployment.getMistake() || '—',
    },
    {
      id: 'maximum-session',
      key: 'max_session_duration',
      label: 'Maximum session',
      value: formatSeconds(maxSessionDuration),
    },
    {
      id: 'idle-timeout',
      key: 'idle_timeout',
      label: 'Idle timeout',
      value: formatIdleTimeout(idleTimeout, idlePromptCount),
    },
    {
      id: 'greeting-playback',
      key: 'greeting_interruptible',
      label: 'Greeting playback',
      value: greetingInterruptible ? 'Interruptible' : 'Not interruptible',
    },
    {
      id: 'unclear-input',
      key: 'unclear_input_timeout',
      label: 'Unclear input',
      value: formatSeconds(unclearInputTimeout),
    },
  ];

  const modalContent = (
    <RightSideModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      className="w-[480px] max-w-[calc(100vw-2rem)]"
      label={deploymentId}
      resizable={false}
      title="Debugger"
    >
      <div className="flex min-h-0 flex-1 flex-col">
        <div className="min-h-0 flex-1 overflow-auto bg-[var(--cds-layer-01)]">
          <Accordion
            align="start"
            aria-label="Debugger configuration"
            className="debugger-configuration-accordion"
            size="md"
          >
            <DebuggerConfigurationSection
              icon={Chat}
              label="General experience"
              open
              options={generalConfiguration}
              summary={`${formatCompactSeconds(maxSessionDuration)} · ${formatCompactSeconds(
                idleTimeout,
              )} idle`}
            />
            <DebuggerConfigurationSection
              icon={VoiceActivate}
              label="Voice input"
              options={getAudioConfiguration(inputAudio, 'listen.')}
              summary={getAudioSummary(inputAudio, 'listen.')}
            />
            <DebuggerConfigurationSection
              icon={VolumeUp}
              label="Voice output"
              options={getAudioConfiguration(outputAudio, 'speak.')}
              summary={getAudioSummary(outputAudio, 'speak.')}
            />
          </Accordion>
        </div>

        <ButtonSet className="!w-full shrink-0 [&>button]:!max-w-none [&>button]:!flex-1">
          <SecondaryButton
            size="lg"
            renderIcon={Edit}
            onClick={editConfiguration}
          >
            Edit configuration
          </SecondaryButton>
          <PrimaryButton size="lg" renderIcon={Launch} onClick={openDebugger}>
            Open debugger
          </PrimaryButton>
        </ButtonSet>
      </div>
    </RightSideModal>
  );

  if (typeof document === 'undefined') return modalContent;

  return createPortal(modalContent, document.body);
}

function DebuggerConfigurationSection({
  icon: Icon,
  label,
  open = false,
  options,
  summary,
}: {
  icon: CarbonIconType;
  label: string;
  open?: boolean;
  options: DebuggerConfigurationOption[];
  summary: string;
}) {
  return (
    <AccordionItem
      aria-label={`${label}, ${summary}`}
      open={open}
      title={
        <span className="flex min-w-0 items-center gap-2">
          <Icon
            size={16}
            className="shrink-0 text-[var(--cds-icon-secondary)]"
          />
          <span className="min-w-0 flex-1 truncate text-sm font-medium text-[var(--cds-text-primary)]">
            {label}
          </span>
          <span className="max-w-36 shrink truncate text-xs text-[var(--cds-text-secondary)]">
            {summary}
          </span>
        </span>
      }
    >
      <StructuredListWrapper
        aria-label={`${label} configuration`}
        className="!mb-0 !w-full"
        isCondensed
        selection={false}
      >
        <StructuredListBody>
          {options.map(option => (
            <StructuredListRow key={option.id}>
              <StructuredListCell className="!w-[48%] !py-2 !pl-4 !pr-2 align-top">
                <span className="block text-sm text-[var(--cds-text-primary)]">
                  {option.label}
                </span>
                <span className="block break-all font-mono text-[0.6875rem] leading-4 text-[var(--cds-text-helper)]">
                  {option.key}
                </span>
              </StructuredListCell>
              <StructuredListCell className="min-w-0 !py-2 !pl-2 !pr-4 align-top">
                <div className="min-w-0 break-words pt-1.5 text-right text-sm text-[var(--cds-text-primary)]">
                  {option.value}
                </div>
              </StructuredListCell>
            </StructuredListRow>
          ))}
        </StructuredListBody>
      </StructuredListWrapper>
    </AccordionItem>
  );
}

function getAudioConfiguration(
  audio: DeploymentAudioProvider | undefined,
  optionPrefix: string,
): DebuggerConfigurationOption[] {
  if (!audio) {
    return [{ id: 'status', key: 'status', label: 'Status', value: 'Off' }];
  }

  return [
    {
      id: 'provider',
      key: 'provider',
      label: 'Provider',
      value: formatProviderName(audio.getAudioprovider()),
    },
    ...audio
      .getAudiooptionsList()
      .filter(option => option.getKey().startsWith(optionPrefix))
      .map(option => ({
        id: option.getKey(),
        key: option.getKey(),
        label: formatAudioOptionLabel(option.getKey()),
        value: option.getValue() || '—',
      })),
  ];
}

function getAudioSummary(
  audio: DeploymentAudioProvider | undefined,
  optionPrefix: string,
) {
  if (!audio) return 'Off';

  const firstOption = audio
    .getAudiooptionsList()
    .find(option => option.getKey().startsWith(optionPrefix))
    ?.getValue();
  return [formatProviderName(audio.getAudioprovider()), firstOption]
    .filter(Boolean)
    .join(' · ');
}

function formatSeconds(value: string | number) {
  const seconds = Number(value);
  return Number.isFinite(seconds) && seconds > 0 ? `${seconds} seconds` : 'Off';
}

function getEffectiveIdleTimeout(value: string) {
  return !value || value === '0' ? DEFAULT_IDEAL_TIMEOUT : value;
}

function formatCompactSeconds(value: string | number) {
  const seconds = Number(value);
  return Number.isFinite(seconds) && seconds > 0 ? `${seconds} sec` : 'Off';
}

function formatIdleTimeout(timeout: string, reminderCount: string) {
  const seconds = Number(timeout);
  if (!Number.isFinite(seconds) || seconds <= 0) return 'Off';

  const reminders = Number(reminderCount);
  if (reminders === 0) return `${seconds} seconds · Unlimited`;

  const reminderLabel = reminders === 1 ? 'reminder' : 'reminders';
  return `${seconds} seconds · ${reminders} ${reminderLabel}`;
}

function formatAudioOptionLabel(key: string) {
  if (key.includes('voice')) return 'Voice';
  if (key.endsWith('model')) return 'Model';

  const name = key.split('.').at(-1)?.replaceAll('_', ' ') || key;
  return `${name.charAt(0).toUpperCase()}${name.slice(1)}`;
}

function formatProviderName(provider: string) {
  if (!provider) return '—';
  return `${provider.charAt(0).toUpperCase()}${provider.slice(1)}`;
}
