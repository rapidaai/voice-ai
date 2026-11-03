import type { AssistantWebpluginDeployment } from '@rapidaai/react';
import { ButtonSet } from '@carbon/react';
import { Launch } from '@carbon/icons-react';
import type { ModalProps } from '@/app/components/ui/primitives';
import {
  PrimaryButton,
  SecondaryButton,
  Tabs,
} from '@/app/components/ui/primitives';
import { RightSideModal } from '@/app/components/dialogs/shared';
import { DeploymentAudioPanel } from '@/app/components/dialogs/shared/deployment-audio-panel';
import { DeploymentConfigurationPanel } from '@/app/components/dialogs/shared/deployment-configuration-panel';
import {
  DeploymentRow,
  DeploymentSectionHeader,
} from '@/app/components/dialogs/shared';
import { WidgetIntegration } from '@/app/components/domain/integration-document/widget-integration';
import { useDocumentationUrl } from '@/theme/documentation-url';
import { useState } from 'react';
import { createPortal } from 'react-dom';

interface AssistantWebWidgetlDeploymentDialogProps extends ModalProps {
  deployment: AssistantWebpluginDeployment;
}

export function AssistantWebWidgetlDeploymentDialog({
  deployment,
  modalOpen,
  setModalOpen,
}: AssistantWebWidgetlDeploymentDialogProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  const documentationUrl = useDocumentationUrl();
  const modalContent = (
    <RightSideModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      className="w-[580px] max-w-[calc(100vw-2rem)]"
      label="Web Widget Deployment"
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
          aria-label="Web widget deployment tabs"
          panelClassName="overflow-auto !p-0"
        >
          <WidgetIntegration
            assistantId={deployment.getAssistantid()}
            showDocumentationStep={false}
          />
          <DeploymentConfigurationPanel
            deployment={deployment}
            channelDetails={<WidgetConfiguration deployment={deployment} />}
          />
          <DeploymentAudioPanel
            inputAudio={deployment.getInputaudio()}
            outputAudio={deployment.getOutputaudio()}
          />
        </Tabs>
        <ButtonSet
          aria-label="Web widget deployment actions"
          className="!w-full shrink-0 [&>button]:!max-w-none [&>button]:!flex-1"
        >
          <SecondaryButton size="lg" onClick={() => setModalOpen(false)}>
            Close
          </SecondaryButton>
          <PrimaryButton
            size="lg"
            renderIcon={Launch}
            onClick={() => window.open(documentationUrl, '_blank')}
          >
            View documentation
          </PrimaryButton>
        </ButtonSet>
      </div>
    </RightSideModal>
  );

  if (typeof document === 'undefined') return modalContent;

  return createPortal(modalContent, document.body);
}

function WidgetConfiguration({
  deployment,
}: {
  deployment: AssistantWebpluginDeployment;
}) {
  const suggestions = deployment.getSuggestionList();

  return (
    <section>
      <DeploymentSectionHeader label="Widget settings" />
      <DeploymentRow label="Name">
        <span className="text-sm text-gray-900 dark:text-gray-100">
          {deployment.getName() || 'Not configured'}
        </span>
      </DeploymentRow>
      <DeploymentRow label="Suggested prompts" className="h-auto min-h-12 py-3">
        <span className="max-w-[320px] break-words text-right text-sm text-gray-900 dark:text-gray-100">
          {suggestions.length > 0 ? suggestions.join(', ') : 'Not configured'}
        </span>
      </DeploymentRow>
      <FeatureRow
        label="Help center"
        enabled={deployment.getHelpcenterenabled()}
      />
      <FeatureRow
        label="Product catalog"
        enabled={deployment.getProductcatalogenabled()}
      />
      <FeatureRow
        label="Article catalog"
        enabled={deployment.getArticlecatalogenabled()}
      />
      <FeatureRow
        label="File upload"
        enabled={deployment.getUploadfileenabled()}
      />
    </section>
  );
}

function FeatureRow({ label, enabled }: { label: string; enabled: boolean }) {
  return (
    <DeploymentRow label={label}>
      <span className="text-sm text-gray-900 dark:text-gray-100">
        {enabled ? 'Enabled' : 'Disabled'}
      </span>
    </DeploymentRow>
  );
}
