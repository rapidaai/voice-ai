import { ICancelButton, IRedBGButton } from '@/app/components/Form/Button';
import { InfoIcon } from '@/app/components/Icon/Info';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import type { FC } from 'react';
import React from 'react';

export type ConfirmDialogProps = {
  showing: boolean;
  type: 'info' | 'warning';
  title: string;
  content: string;
  confirmText?: string;
  onConfirm: () => void;
  cancelText?: string;
  onCancel: () => void;
  onClose: () => void;
};

const ConfirmDialog: FC<ConfirmDialogProps> = ({
  showing,
  type,
  title,
  content,
  confirmText,
  cancelText,
  onClose,
  onConfirm,
  onCancel,
}) => {
  return (
    <GenericModal modalOpen={showing} setModalOpen={onClose}>
      <div className="flex flex-col dark:bg-gray-900 bg-white">
        <div className="w-[200px] min-w-max rounded-2xl relative py-8 px-4 flex flex-col">
          {type === 'info' && <InfoIcon className="w-10 h-10 text-blue-500" />}
          {type === 'warning' && (
            <InfoIcon className="w-10 h-10 text-red-500" />
          )}
          <div className="text-lg font-medium mt-2">{title}</div>
          <div className="text-base leading-normal">{content}</div>
        </div>
        <ModalFooter>
          <ICancelButton className="px-4 rounded-[2px]" onClick={onCancel}>
            {cancelText}
          </ICancelButton>
          <IRedBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={onConfirm}
          >
            {confirmText}
          </IRedBGButton>
        </ModalFooter>
      </div>
    </GenericModal>
  );
};
export default React.memo(ConfirmDialog);
