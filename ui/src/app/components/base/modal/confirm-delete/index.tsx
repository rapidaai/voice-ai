import { ICancelButton, IRedBGButton } from '@/app/components/Form/Button';
import { Input } from '@/app/components/Form/Input';
import { InfoIcon } from '@/app/components/Icon/Info';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import type { FC } from 'react';
import React, { useState } from 'react';

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

export const ConfirmDeleteDialog: FC<ConfirmDeleteDialogProps> = ({
  showing,
  title,
  content,
  confirmText = 'Delete',
  cancelText = 'Cancel',
  objectName,
  onClose,
  onConfirm,
  onCancel,
}) => {
  const [inputName, setInputName] = useState('');

  const handleConfirm = () => {
    if (inputName === objectName) {
      onConfirm();
    }
  };

  return (
    <GenericModal modalOpen={showing} setModalOpen={onClose}>
      <ModalFitHeightBlock className="w-[300px] min-w-max ">
        <div className="rounded-2xl relative py-8 px-4 flex flex-col">
          <InfoIcon className="w-10 h-10 text-red-500" />
          <div className="text-lg font-medium mt-2">{title}</div>
          <div className="text-base leading-normal mb-4">{content}</div>
          <Input
            type="text"
            value={inputName}
            onChange={e => setInputName(e.target.value)}
            placeholder={`Type "${objectName}" to confirm`}
          />
        </div>
        <ModalFooter>
          <ICancelButton className="px-4 rounded-[2px]" onClick={onCancel}>
            {cancelText}
          </ICancelButton>
          <IRedBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={handleConfirm}
            disabled={inputName !== objectName}
          >
            {confirmText}
          </IRedBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};
