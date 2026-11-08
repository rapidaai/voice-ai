import type { ReactNode } from 'react';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import {
  DEFAULT_IDEAL_TIMEOUT,
  DEFAULT_IDLE_PROMPT_COUNT,
  DEFAULT_MAX_SESSION_DURATION,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
} from '@/app/pages/assistant/actions/create-deployment/commons/configure-experience';
import {
  DeploymentRow,
  DeploymentSectionHeader,
} from './deployment-modal-primitives';

export interface DeploymentConfiguration {
  getAssistantid(): string;
  getStatus(): string;
  getMaxsessionduration(): string;
  getIdealtimeout(): string;
  getIdealtimeoutmessage(): string;
  getIdealtimeoutbackoff(): string;
  hasGreetinginterruptible?(): boolean;
  getGreetinginterruptible(): boolean;
  hasUnclearinputtimeout?(): boolean;
  getUnclearinputtimeout(): number;
  getUnclearinputmessage(): string;
  getGreeting(): string;
  getMistake(): string;
}

interface DeploymentConfigurationPanelProps {
  deployment: DeploymentConfiguration;
  channelDetails?: ReactNode;
}

export function DeploymentConfigurationPanel({
  deployment,
  channelDetails,
}: DeploymentConfigurationPanelProps) {
  const status = deployment.getStatus();
  const savedMaximumDuration = deployment.getMaxsessionduration();
  const maximumDuration =
    !savedMaximumDuration || savedMaximumDuration === '0'
      ? DEFAULT_MAX_SESSION_DURATION
      : savedMaximumDuration;
  const savedIdleTimeout = deployment.getIdealtimeout();
  const idleTimeout =
    !savedIdleTimeout || savedIdleTimeout === '0'
      ? DEFAULT_IDEAL_TIMEOUT
      : savedIdleTimeout;
  const idleReminderCount =
    deployment.getIdealtimeoutbackoff() || DEFAULT_IDLE_PROMPT_COUNT;
  const greetingInterruptible = deployment.hasGreetinginterruptible?.()
    ? deployment.getGreetinginterruptible()
    : true;
  const unclearInputTimeout = deployment.hasUnclearinputtimeout?.()
    ? deployment.getUnclearinputtimeout()
    : Number(DEFAULT_UNCLEAR_INPUT_TIMEOUT);

  return (
    <div className="w-full divide-y divide-border-subtle ">
      <section>
        <DeploymentRow label="Status">
          <RecordStatusIndicator state={status || 'UNKNOWN'} />
        </DeploymentRow>
        <DeploymentRow label="Assistant ID">
          <ConfigurationValue value={deployment.getAssistantid()} mono />
        </DeploymentRow>
      </section>

      {channelDetails}

      <section>
        <DeploymentSectionHeader label="Session settings" />
        <DeploymentRow label="Maximum duration">
          <ConfigurationValue value={withSeconds(maximumDuration)} />
        </DeploymentRow>
        <DeploymentRow label="Idle timeout">
          <ConfigurationValue value={withSeconds(idleTimeout)} />
        </DeploymentRow>
        <DeploymentRow label="Idle reminders">
          <ConfigurationValue
            value={idleReminderCount === '0' ? 'Unlimited' : idleReminderCount}
          />
        </DeploymentRow>
        <DeploymentRow label="Greeting">
          <ConfigurationValue
            value={
              greetingInterruptible ? 'Interruptible' : 'Not interruptible'
            }
          />
        </DeploymentRow>
        <DeploymentRow label="Unclear input timeout">
          <ConfigurationValue
            value={withSeconds(unclearInputTimeout.toString(), 'Disabled')}
          />
        </DeploymentRow>
      </section>

      <section>
        <DeploymentSectionHeader label="Conversation messages" />
        <MessageRow label="Greeting" value={deployment.getGreeting()} />
        <MessageRow label="Error response" value={deployment.getMistake()} />
        <MessageRow
          label="Idle reminder"
          value={deployment.getIdealtimeoutmessage()}
        />
        <MessageRow
          label="Unclear input"
          value={deployment.getUnclearinputmessage()}
        />
      </section>
    </div>
  );
}

function MessageRow({ label, value }: { label: string; value: string }) {
  return (
    <DeploymentRow label={label} className="h-auto min-h-12 py-3">
      <ConfigurationValue value={value} />
    </DeploymentRow>
  );
}

function ConfigurationValue({
  value,
  mono = false,
}: {
  value?: string;
  mono?: boolean;
}) {
  return (
    <span
      className={`max-w-[320px] break-words text-right text-sm text-foreground ${
        mono ? 'font-mono' : ''
      }`}
    >
      {value || 'Not configured'}
    </span>
  );
}

function withSeconds(value: string, emptyValue = 'Not configured') {
  return value && value !== '0' ? `${value} seconds` : emptyValue;
}
