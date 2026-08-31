import { useParams } from 'react-router-dom';
import { useCallback, useEffect, useState } from 'react';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks/use-rapida-store';
import { KnowledgeDocument } from '@rapidaai/react';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ButtonSet } from '@carbon/react';
import { ManualFile } from '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ErrorMessage } from '@/app/components/form/error-message';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';

export function CreateKnowledgeDocumentPage() {
  const { id } = useParams();
  const [knowledgeId, setKnowledgeId] = useState<string | null>(null);
  useEffect(() => {
    if (id) {
      setKnowledgeId(id);
    }
  }, [id]);

  const [errorMessage, setErrorMessage] = useState('');
  const { goToKnowledge, goBack } = useGlobalNavigation();
  const { clear } = useCreateKnowledgeDocumentPageStore();
  useEffect(() => {
    clear();
  }, [knowledgeId]);

  const [userId, token, projectId] = useCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const knowledgeDocumentAction = useCreateKnowledgeDocumentPageStore();

  const onSuccess = useCallback(
    (d: KnowledgeDocument[]) => {
      hideLoader();
      goToKnowledge(knowledgeId!);
    },
    [knowledgeId],
  );

  const onError = useCallback(
    (e: string) => {
      hideLoader();
      setErrorMessage(e);
    },
    [knowledgeId],
  );

  const onCreateKnowledgeDocument = () => {
    showLoader('overlay');
    knowledgeDocumentAction.onCreateKnowledgeDocument(
      knowledgeId!,
      projectId,
      token,
      userId,
      onSuccess,
      onError,
    );
  };

  if (!knowledgeId) return <div>Please check the url and try again.</div>;

  return (
    <div className="max-w-4xl mx-auto">
      <DocNoticeBlock docPath="/knowledge/overview">
        Upload your text files (e.g., .txt, .doc, .docx, .pdf). Maximum file
        size: 10 MB per file.
      </DocNoticeBlock>

      <div className="p-6">
        <ManualFile />
      </div>

      <div className="flex items-center justify-between px-6 pb-6">
        <ErrorMessage message={errorMessage} className="rounded-none!" />
        <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
          <SecondaryButton size="lg" onClick={() => goBack()}>
            Cancel
          </SecondaryButton>
          <PrimaryButton
            size="lg"
            isLoading={loading}
            onClick={onCreateKnowledgeDocument}
          >
            Upload new document
          </PrimaryButton>
        </ButtonSet>
      </div>
    </div>
  );
}
