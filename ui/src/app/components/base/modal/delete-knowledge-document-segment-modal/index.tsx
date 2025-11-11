import { DeleteKnowledgeDocumentSegment } from '@rapidaai/react';
import { BaseResponse } from '@rapidaai/react';
import { KnowledgeDocumentSegment } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { GenericModal } from '@/app/components/base/modal';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { FormLabel } from '@/app/components/form-label';
import { Button, HoverButton } from '@/app/components/Form/Button';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Textarea } from '@/app/components/Form/Textarea';
import { useCurrentCredential } from '@/hooks/use-credential';
import { FC, useState } from 'react';
import { connectionConfig } from '@/configs';

export const DeleteKnowledgeDocumentSegmentDialog: FC<{
  segment: KnowledgeDocumentSegment;
  onClose: () => void;
  onDelete: () => void;
}> = ({ segment, onClose, onDelete }) => {
  const { authId, token, projectId } = useCurrentCredential();
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleDelete = () => {
    if (!reason.trim()) {
      setError('Please provide a reason for deletion.');
      return;
    }
    setError(null);
    DeleteKnowledgeDocumentSegment(
      connectionConfig,
      segment.getDocumentId(),
      segment.getIndex().toString(),
      reason.trim(),
      (err: ServiceError | null, response: BaseResponse | null) => {
        if (err) {
          console.error('Error deleting segment:', err);
          setError('Failed to delete the segment. Please try again.');
        } else {
          console.log('Segment deleted successfully:', response);
          onDelete();
          onClose();
        }
      },
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': authId,
      },
    );
  };

  return (
    <GenericModal modalOpen={true} setModalOpen={onClose}>
      <ModalHeader onClose={onClose}>
        <ModalTitleBlock>
          Are you sure you want to delete this document segment?
        </ModalTitleBlock>
      </ModalHeader>
      <ModalBody>
        <FieldSet>
          <FormLabel htmlFor="delete-reason">Reason</FormLabel>
          <Textarea
            name="delete-reason"
            value={reason}
            onChange={e => setReason(e.target.value)}
            placeholder="Please provide a reason for deleting this segment"
            rows={4}
          />
        </FieldSet>
        <ErrorMessage message={error || ''} />
      </ModalBody>
      <ModalFooter>
        <HoverButton type="button" onClick={onClose}>
          Cancel
        </HoverButton>
        <Button type="button" onClick={handleDelete}>
          Delete Segment
        </Button>
      </ModalFooter>
    </GenericModal>
  );
};
