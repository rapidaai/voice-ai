import type { DeploymentAudioProvider } from '@rapidaai/react';
import { ProviderPill } from '@/app/components/domain/pills/provider-model-pill';
import { CopyButton } from '@/app/components/ui/primitives';
import {
  DeploymentRow,
  DeploymentSectionHeader,
} from './deployment-modal-primitives';

interface DeploymentAudioPanelProps {
  inputAudio?: DeploymentAudioProvider;
  outputAudio?: DeploymentAudioProvider;
}

export function DeploymentAudioPanel({
  inputAudio,
  outputAudio,
}: DeploymentAudioPanelProps) {
  return (
    <section className="w-full">
      <DeploymentSectionHeader label="Voice configuration" />
      <AudioProviderRows
        deployment={inputAudio}
        direction="Input"
        optionPrefix="listen."
      />
      <AudioProviderRows
        deployment={outputAudio}
        direction="Output"
        optionPrefix="speak."
      />
    </section>
  );
}

function AudioProviderRows({
  deployment,
  direction,
  optionPrefix,
}: {
  deployment?: DeploymentAudioProvider;
  direction: 'Input' | 'Output';
  optionPrefix: string;
}) {
  const options = (deployment?.getAudiooptionsList() ?? []).filter(
    option => option.getValue() && option.getKey().startsWith(optionPrefix),
  );

  if (!deployment) {
    return (
      <DeploymentRow label={`${direction} provider`}>
        <span className="text-sm text-muted ">Disabled</span>
      </DeploymentRow>
    );
  }

  return (
    <>
      <DeploymentRow label={`${direction} provider`}>
        <ProviderPill provider={deployment.getAudioprovider()} />
      </DeploymentRow>
      {options.length > 0 ? (
        options.map((option, index) => (
          <DeploymentRow
            key={option.getKey() || `${optionPrefix}-${index}`}
            label={`${direction} ${formatOptionLabel(
              option.getKey(),
              optionPrefix,
            )}`}
          >
            <span className="max-w-[200px] truncate text-right font-mono text-sm text-foreground ">
              {option.getValue()}
            </span>
            <CopyButton className="h-6 w-6 shrink-0">
              {option.getValue()}
            </CopyButton>
          </DeploymentRow>
        ))
      ) : (
        <DeploymentRow label={`${direction} options`}>
          <span className="text-sm text-muted ">Not configured</span>
        </DeploymentRow>
      )}
    </>
  );
}

function formatOptionLabel(key: string, prefix: string) {
  return key.slice(prefix.length).replace(/[._-]+/g, ' ');
}
