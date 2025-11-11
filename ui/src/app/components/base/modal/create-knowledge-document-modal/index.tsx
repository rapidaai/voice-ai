import React, { FC, useCallback, useEffect, useState } from 'react';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ManualFile } from '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file';
import { KnowledgeDocument } from '@rapidaai/react';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import { useCredential, useRapidaStore } from '@/hooks';

interface CreateKnowledgeDocumentDialogProps extends ModalProps {
  knowledgeId: string;
  onReload: () => void;
}

export const CreateKnowledgeDocumentDialog: FC<
  CreateKnowledgeDocumentDialogProps
> = props => {
  const [errorMessage, setErrorMessage] = useState('');
  const { clear } = useCreateKnowledgeDocumentPageStore();
  useEffect(() => {
    clear();
  }, [props.knowledgeId]);
  /**
   * all the credentials you will need do things
   */
  const [userId, token, projectId] = useCredential();

  /**
   * show and hide loaders
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   *
   */

  const knowledgeDocumentAction = useCreateKnowledgeDocumentPageStore();

  /**
   *
   */
  const onSuccess = useCallback(
    (d: KnowledgeDocument[]) => {
      hideLoader();
      props.setModalOpen(false);
      props.onReload();
    },
    [props.knowledgeId],
  );

  /**
   *
   */
  const onError = useCallback(
    (e: string) => {
      hideLoader();
      setErrorMessage(e);
    },
    [props.knowledgeId],
  );

  /**
   *
   */
  const onCreateKnowledgeDocument = () => {
    showLoader('overlay');
    knowledgeDocumentAction.onCreateKnowledgeDocument(
      props.knowledgeId!,
      projectId,
      token,
      userId,
      onSuccess,
      onError,
    );
  };
  return (
    <GenericModal
      className="flex"
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
    >
      <ModalFitHeightBlock className="w-[1000px]">
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
          title={'Configure Assistant Tool'}
        >
          <ModalTitleBlock>Configure Assistant Tool</ModalTitleBlock>
        </ModalHeader>
        <ModalBody className="overflow-auto max-h-[80dvh]">
          <ManualFile />
        </ModalBody>
        <ModalFooter errorMessage={errorMessage}>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => {
              props.setModalOpen(false);
            }}
          >
            Cancel
          </ICancelButton>
          <IBlueBGArrowButton
            className="px-4 rounded-[2px]"
            type="button"
            isLoading={loading}
            onClick={onCreateKnowledgeDocument}
          >
            Create Document
          </IBlueBGArrowButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};
