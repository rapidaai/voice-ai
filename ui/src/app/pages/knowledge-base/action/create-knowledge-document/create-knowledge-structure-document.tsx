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
import { RapidaDocumentType } from '@/utils/rapida_document';
import { ArrowLeft, UploadIcon } from 'lucide-react';
import { Select } from '@/app/components/form/select';
import { Helmet } from '@/app/components/helmet';
import { FormLabel } from '@/app/components/form-label';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';
import { SectionDivider } from '@/app/components/blocks/section-divider';

export function CreateKnowledgeStructureDocumentPage() {
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
    if (
      knowledgeDocumentAction.documentType === RapidaDocumentType.UNSTRUCTURE
    ) {
      setErrorMessage('Please select document type of the file and try again.');
      return;
    }
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
      <Helmet title="Create knowledge document" />

      {/* Back link */}
      <div className="px-6 pt-6 pb-2">
        <button
          type="button"
          className="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400 hover:text-primary transition-colors"
          onClick={goBack}
        >
          <ArrowLeft className="w-4 h-4" strokeWidth={1.5} />
          Back to knowledge
        </button>
      </div>

      <div className="px-6 pb-8 flex flex-col gap-8">
        {/* Document type section */}
        <div className="flex flex-col gap-6">
          <SectionDivider label="Document Structure" />
          <div className="flex flex-col gap-2">
            <FormLabel className="text-xs tracking-wide uppercase">
              Document Type
            </FormLabel>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Select the category that best matches the content of the document
              you are about to upload.
            </p>
            <Select
              placeholder="Select document type"
              options={[
                {
                  name: 'Help / QnA',
                  value: RapidaDocumentType.STRUCTURE_QNA,
                },
                {
                  name: 'Product Catalog',
                  value: RapidaDocumentType.STRUCTURE_PRODUCT,
                },
                {
                  name: 'Blog Article',
                  value: RapidaDocumentType.STRUCTURE_ARTICLE,
                },
              ]}
              onChange={e =>
                knowledgeDocumentAction.onChangeDocumentType(
                  e.target.value as RapidaDocumentType,
                )
              }
            />
          </div>
        </div>

        {/* Upload section */}
        <div className="flex flex-col gap-6">
          <SectionDivider label="Documents" />
          <DocNoticeBlock docPath="/knowledge/overview">
            Upload your structured files (e.g., .csv, .xls, .json). Maximum file
            size: 10 MB per file.
          </DocNoticeBlock>
          <ManualFile
            accepts={{
              'application/vnd.ms-excel': ['.xls', '.xlsx'],
              'text/csv': ['.csv'],
              'application/json': ['.json'],
            }}
          />
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between">
          <ErrorMessage message={errorMessage} className="rounded-none!" />
          <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none ml-auto">
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
    </div>
  );
}
