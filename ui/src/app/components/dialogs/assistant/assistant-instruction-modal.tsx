import { useCallback, memo } from 'react';
import type { HTMLAttributes } from 'react';
import {
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  type ModalProps,
  PrimaryButton,
  SecondaryButton,
} from '@/app/components/ui/primitives';
import { Launch } from '@carbon/icons-react';
import { CodeHighlighting } from '@/app/components/ui/editor/code-highlighting';
import { DeploymentSectionHeader } from '@/app/components/dialogs/shared/deployment-modal-primitives';
import { useDocumentationUrl } from '@/theme/documentation-url';
import { CONFIG } from '@/configs';

interface AssistantInstructionDialogProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  assistantId: string;
}

function AssistantWebwidgetDeploymentDialogComponent({
  assistantId,
  modalOpen,
  setModalOpen,
}: AssistantInstructionDialogProps) {
  const documentationUrl = useDocumentationUrl();
  const closeDialog = useCallback(() => {
    setModalOpen(false);
  }, [setModalOpen]);

  return (
    <Modal open={modalOpen} onClose={closeDialog} size="md">
      <ModalHeader onClose={closeDialog} title="Deployment completed">
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
          Add the following snippets to your website to start receiving
          messages.
        </p>
      </ModalHeader>

      <ModalBody className="gap-0 !p-0">
        <DeploymentSectionHeader label="1. Add script to your HTML" />
        <div className="px-4 py-3">
          <CodeHighlighting
            className="min-h-[20px]"
            code={`<script src="${CONFIG.deployment.webWidgetScriptUrl}" defer></script>`}
          />
        </div>

        <DeploymentSectionHeader label="2. Initialize the assistant" />
        <div className="px-4 py-3">
          <CodeHighlighting
            className="min-h-[240px]"
            code={`<script>
window.chatbotConfig = {
  assistant_id: "${assistantId}",
  token: "{PROJECT_KEY}",
  user: {
    id: "{UNIQUE_IDENTIFIER}",
    name: "{NAME}",
  },
  layout: "docked-right",
  position: "bottom-right",
  showLauncher: true,
  name: "Assistant",
  theme: {
    mode: "light",
  },
};
</script>`}
          />
        </div>
      </ModalBody>

      <ModalFooter>
        <SecondaryButton size="lg" onClick={closeDialog}>
          Close
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          type="button"
          onClick={() => window.open(documentationUrl, '_blank')}
          renderIcon={Launch}
        >
          View Documentation
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}

export const AssistantWebwidgetDeploymentDialog = memo(
  AssistantWebwidgetDeploymentDialogComponent,
);
