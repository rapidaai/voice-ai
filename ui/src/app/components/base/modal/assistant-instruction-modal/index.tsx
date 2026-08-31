import { ModalProps } from '@/app/components/base/modal';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import { FC, HTMLAttributes, memo } from 'react';
import { ExternalLink } from 'lucide-react';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { CodeHighlighting } from '@/app/components/code-highlighting';
import { DeploymentSectionHeader } from '@/app/components/base/modal/deployment-modal-primitives';
import { useDocumentationUrl } from '@/theme/documentation-url';

interface AssistantInstructionDialogProps
  extends ModalProps, HTMLAttributes<HTMLDivElement> {
  assistantId: string;
}

export const AssistantWebwidgetDeploymentDialog: FC<AssistantInstructionDialogProps> =
  memo(({ assistantId, modalOpen, setModalOpen }) => {
    const documentationUrl = useDocumentationUrl();

    return (
      <Modal open={modalOpen} onClose={() => setModalOpen(false)} size="md">
        <ModalHeader
          onClose={() => setModalOpen(false)}
          title="Deployment completed"
        >
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
              code='<script src="https://cdn-01.rapida.ai/public/scripts/app.min.js" defer></script>'
            />
          </div>

          <DeploymentSectionHeader label="2. Initialize the assistant" />
          <div className="px-4 py-3">
            <CodeHighlighting
              className="min-h-[240px]"
              code={`<script>
window.chatbotConfig = {
  assistant_id: "${assistantId}",
  token: "{RAPIDA_PROJECT_KEY}",
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
          <SecondaryButton size="lg" onClick={() => setModalOpen(false)}>
            Close
          </SecondaryButton>
          <PrimaryButton
            size="lg"
            type="button"
            onClick={() => window.open(documentationUrl, '_blank')}
          >
            <span>View Documentation</span>
            <ExternalLink className="w-4 h-4 ml-1" strokeWidth={1.5} />
          </PrimaryButton>
        </ModalFooter>
      </Modal>
    );
  });
