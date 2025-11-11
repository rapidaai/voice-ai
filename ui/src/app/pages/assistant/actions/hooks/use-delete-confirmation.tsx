// hooks/useConfirmDialog.tsx

import { useState, useCallback } from 'react';
import { ConfirmDeleteDialog } from '@/app/components/base/modal/confirm-delete';

interface UseConfirmDialogProps {
  onConfirm: () => void;
  name: string;
}

export const useDeleteConfirmDialog = ({
  onConfirm,
  name,
}: UseConfirmDialogProps) => {
  //
  const [isShow, setIsShow] = useState(false);
  const showDialog = useCallback(() => {
    setIsShow(true);
  }, []);

  const hideDialog = useCallback(() => {
    setIsShow(false);
  }, []);

  const handleConfirm = useCallback(() => {
    onConfirm();
    hideDialog();
  }, [onConfirm, hideDialog]);

  const ConfirmDeleteDialogComponent = () => (
    <ConfirmDeleteDialog
      showing={isShow}
      title="Confirm Deletion"
      content="Are you sure you want to delete this item? This action cannot be undone."
      objectName={name}
      onConfirm={handleConfirm}
      onCancel={() => {
        setIsShow(false);
      }}
      onClose={() => {
        setIsShow(false);
      }}
    />
  );

  return {
    isShow,
    showDialog,
    hideDialog,
    ConfirmDeleteDialogComponent,
  };
};
