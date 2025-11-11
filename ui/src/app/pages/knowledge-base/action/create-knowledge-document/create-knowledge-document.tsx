import { useParams } from 'react-router-dom';
import { useCallback, useEffect, useState } from 'react';
import { useCredential, useRapidaStore } from '@/hooks';
import { KnowledgeDocument } from '@rapidaai/react';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { ManualFile } from '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ErrorMessage } from '@/app/components/Form/error-message';

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
      goToKnowledge(knowledgeId!);
    },
    [knowledgeId],
  );

  /**
   *
   */
  const onError = useCallback(
    (e: string) => {
      hideLoader();
      setErrorMessage(e);
    },
    [knowledgeId],
  );

  /**
   *
   */
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
    <div className="max-w-4xl mx-auto p-6">
      {/* Deploy Model */}

      {/* Info Banner */}
      <div
        className="flex items-center p-2 px-4 text-blue-800 border-l-4 border-blue-300 bg-blue-50 dark:text-blue-400 dark:bg-gray-800 dark:border-blue-800 w-full"
        role="alert"
      >
        <svg
          className="shrink-0 w-4 h-4"
          aria-hidden="true"
          xmlns="http://www.w3.org/2000/svg"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path d="M10 .5a9.5 9.5 0 1 0 9.5 9.5A9.51 9.51 0 0 0 10 .5ZM9.5 4a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3ZM12 15H8a1 1 0 0 1 0-2h1v-3H8a1 1 0 0 1 0-2h2a1 1 0 0 1 1 1v4h1a1 1 0 0 1 0 2Z" />
        </svg>
        <div className="ms-3 text-sm font-medium">
          Upload your text files (e.g., .txt, .doc, .docx, .pdf). Maximum file
          size: 10 MB per file.
        </div>
      </div>

      {/* Hardware Configuration */}
      <div className="p-6">
        <ManualFile />
      </div>
      <div className="flex items-center justify-between px-6">
        <div>
          <ErrorMessage message={errorMessage} className="rounded-none!" />
        </div>
        <div className="flex space-x-3 py-6">
          <ICancelButton
            onClick={() => goBack()}
            className="px-4 rounded-[2px]"
          >
            Cancel
          </ICancelButton>
          <IBlueBGArrowButton
            isLoading={loading}
            className="px-4 rounded-[2px]"
            type="button"
            onClick={onCreateKnowledgeDocument}
          >
            Upload new document
          </IBlueBGArrowButton>
        </div>
      </div>
    </div>
  );
}
