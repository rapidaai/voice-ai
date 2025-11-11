import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { FC, HTMLAttributes, memo } from 'react';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { CheckCircle, ExternalLink } from 'lucide-react';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ICancelButton, ILinkButton } from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface AssistantInstructionDialogProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  assistantId: string;
}

export const AssistantWebwidgetDeploymentDialog: FC<AssistantInstructionDialogProps> =
  memo(({ assistantId, ...mldAttr }) => {
    return (
      <GenericModal {...mldAttr}>
        <ModalFitHeightBlock className="w-[1000px]">
          <ModalHeader
            onClose={() => {
              mldAttr.setModalOpen(false);
            }}
          >
            <div className="flex items-center gap-3 mb-2">
              <div className="w-8 h-8 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center">
                <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400" />
              </div>
              <div className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                Deployment Completed Successfully!
              </div>
            </div>
            <div className="text-gray-600 dark:text-gray-400">
              Your AI assistant has been deployed to web widget. Follow the
              integration steps below to start receiving messages.
            </div>
          </ModalHeader>
          <ModalBody>
            <FieldSet>
              <div className="text-muted-foreground">
                Add the Rapida.js script to your HTML
              </div>
              <CodeHighlighting
                className="h-[20px]"
                lang="html"
                code='<script src="https://cdn-01.rapida.ai/public/scripts/app.min.js" defer></script>'
              ></CodeHighlighting>
            </FieldSet>
            <FieldSet>
              <div className="text-muted-foreground">
                Add the chatbot configuration script
              </div>
              <CodeHighlighting
                lang="html"
                className="h-[320px]"
                code={`<script>
window.chatbotConfig = {
  theme: {
    color: "black"
  },
  assistant_id: ${assistantId},
  token: "{RAPIDA_PROJECT_KEY}",
  user: {
    id: "{UNIQUE_IDENITFIER}",
    name: "{NAME}"
  }
};
</script>`}
              ></CodeHighlighting>
            </FieldSet>
          </ModalBody>
          <ModalFooter>
            <ICancelButton
              className="px-4 rounded-[2px]"
              onClick={() => mldAttr.setModalOpen(true)}
            >
              Close
            </ICancelButton>
            <ILinkButton
              href="https://doc.rapida.ai"
              className="px-4 rounded-[2px]"
              target="_blank"
            >
              View Documentation
              <ExternalLink className="w-4 h-4 ml-2" strokeWidth={1.5} />
            </ILinkButton>
          </ModalFooter>
        </ModalFitHeightBlock>
      </GenericModal>
    );
  });
