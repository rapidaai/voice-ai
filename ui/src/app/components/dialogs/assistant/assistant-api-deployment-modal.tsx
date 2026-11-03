import type { AssistantApiDeployment } from '@rapidaai/react';
import type { ModalProps } from '@/app/components/ui/primitives';
import { RightSideModal } from '@/app/components/dialogs/shared';
import { DeploymentAudioPanel } from '@/app/components/dialogs/shared/deployment-audio-panel';
import { DeploymentConfigurationPanel } from '@/app/components/dialogs/shared/deployment-configuration-panel';
import { ApiDeploymentSetup } from '@/app/components/domain/integration-document/assistant-deployment-setup';
import { Tabs } from '@/app/components/ui/primitives';
import { useState } from 'react';
import { createPortal } from 'react-dom';

interface AssistantApiDeploymentDialogProps extends ModalProps {
  deployment: AssistantApiDeployment;
}

export function AssistantApiDeploymentDialog({
  deployment,
  modalOpen,
  setModalOpen,
}: AssistantApiDeploymentDialogProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  const modalContent = (
    <RightSideModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      className="w-[580px] max-w-[calc(100vw-2rem)]"
      label="SDK / API Deployment"
      title={deployment.getId()}
    >
      <div className="relative flex min-h-0 flex-1 flex-col">
        <Tabs
          tabs={['Setup', 'Configuration', 'Audio']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          fullWidth
          fill
          aria-label="SDK / API deployment tabs"
          panelClassName="overflow-auto !p-0"
        >
          <ApiDeploymentSetup />
          <DeploymentConfigurationPanel deployment={deployment} />
          <DeploymentAudioPanel
            inputAudio={deployment.getInputaudio()}
            outputAudio={deployment.getOutputaudio()}
          />
        </Tabs>
      </div>
    </RightSideModal>
  );

  if (typeof document === 'undefined') return modalContent;

  return createPortal(modalContent, document.body);
}
