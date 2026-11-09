import { useState } from 'react';
import {
  DangerButton,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  SecondaryButton,
  TextInput,
} from '@/app/components/ui/primitives';

type ConfirmDeleteDialogProps = {
  showing: boolean;
  title: string;
  content: string;
  confirmText?: string;
  objectName: string;
  onConfirm: () => void;
  cancelText?: string;
  onCancel: () => void;
  onClose: () => void;
};

export function ConfirmDeleteDialog({
  showing,
  title,
  content,
  confirmText = 'Delete',
  cancelText = 'Cancel',
  objectName,
  onClose,
  onConfirm,
  onCancel,
}: ConfirmDeleteDialogProps) {
  const [inputName, setInputName] = useState('');

  const closeDialog = () => {
    setInputName('');
    onClose();
  };

  const cancelDialog = () => {
    setInputName('');
    onCancel();
  };

  const handleConfirm = () => {
    if (inputName === objectName) {
      onConfirm();
      setInputName('');
    }
  };

  return (
    <Modal
      aria-label={title}
      danger
      open={showing}
      onClose={closeDialog}
      size="sm"
    >
      <ModalHeader label="Confirm action" title={title} onClose={closeDialog} />
      <ModalBody hasForm>
        <p className="mb-4 text-sm text-foreground">{content}</p>
        <TextInput
          id="confirm-delete-input"
          labelText={`Type "${objectName}" to confirm`}
          value={inputName}
          onChange={e => setInputName(e.target.value)}
          placeholder={objectName}
          autoComplete="off"
        />
      </ModalBody>
      <ModalFooter danger>
        <SecondaryButton size="lg" onClick={cancelDialog}>
          {cancelText}
        </SecondaryButton>
        <DangerButton
          size="lg"
          onClick={handleConfirm}
          disabled={inputName !== objectName}
        >
          {confirmText}
        </DangerButton>
      </ModalFooter>
    </Modal>
  );
}
