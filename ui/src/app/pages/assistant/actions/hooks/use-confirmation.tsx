import { useState, useCallback } from 'react';
import ConfirmDialog from '@/app/components/base/modal/confirm-ui';

interface UseConfirmDialogProps {
  title?: string;
  content?: string;
}

export const useConfirmDialog = ({
  title = 'Are you sure?',
  content = 'You want to cancel? Any unsaved changes will be lost.',
}: UseConfirmDialogProps) => {
  const [isShow, setIsShow] = useState(false);
  const [currentOnConfirm, setCurrentOnConfirm] = useState<() => void>(
    () => {},
  );

  const showDialog = useCallback((onConfirm: () => void) => {
    setCurrentOnConfirm(() => onConfirm);
    setIsShow(true);
  }, []);

  const hideDialog = useCallback(() => {
    setIsShow(false);
  }, []);

  const handleConfirm = useCallback(() => {
    currentOnConfirm();
    hideDialog();
  }, [currentOnConfirm, hideDialog]);

  const ConfirmDialogComponent = () => (
    <ConfirmDialog
      showing={isShow}
      type="warning"
      title={title}
      content={content}
      confirmText={'Confirm'}
      cancelText="Cancel"
      onConfirm={handleConfirm}
      onCancel={hideDialog}
      onClose={hideDialog}
    />
  );

  return {
    isShow,
    showDialog,
    hideDialog,
    ConfirmDialogComponent,
  };
};
