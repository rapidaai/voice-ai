import { useCallback } from 'react';
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

interface DebuggerDeploymentSuccessDialogProps extends ModalProps {
  assistantId: string;
}

export function DebuggerDeploymentSuccessDialog({
  modalOpen,
  setModalOpen,
  assistantId,
}: DebuggerDeploymentSuccessDialogProps) {
  const closeDialog = useCallback(() => {
    setModalOpen(false);
  }, [setModalOpen]);

  return (
    <Modal open={modalOpen} onClose={closeDialog} size="sm">
      <ModalHeader title="Deployment completed" onClose={closeDialog} />
      <ModalBody>
        <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
          Your debugger is ready. Use the preview to test your assistant in a
          sandbox environment before deploying to other channels.
        </p>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={closeDialog}>
          Close
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          type="button"
          onClick={() =>
            window.open(
              `/deployment/assistant/${assistantId}/playground/chat`,
              '_blank',
            )
          }
          renderIcon={Launch}
        >
          Preview assistant
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
