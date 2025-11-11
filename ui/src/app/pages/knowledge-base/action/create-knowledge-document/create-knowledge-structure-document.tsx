import { useParams } from 'react-router-dom';
import { useCallback, useEffect, useState } from 'react';
import { useCredential, useRapidaStore } from '@/hooks';
import { KnowledgeDocument } from '@rapidaai/react';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import { HoverButton, OutlineButton } from '@/app/components/Form/Button';
import { ManualFile } from '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { RapidaDocumentType } from '@/utils/rapida_document';
import { ArrowLeft, UploadIcon } from 'lucide-react';
import { Select } from '@/app/components/Form/Select';
import { Helmet } from '@/app/components/Helmet';
import { Label } from '@/app/components/Form/Label';

export function CreateKnowledgeStructureDocumentPage() {
  const { id } = useParams();
  const [knowledgeId, setKnowledgeId] = useState<string | null>(null);
  useEffect(() => {
    if (id) {
      setKnowledgeId(id);
    }
  }, [id]);

  //   return <UploadKnowledgeDocument knowledgeId={knowledgeId} />;
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
      <Helmet title="Create knowledge doucment" />
      <div className="my-6">
        <button className="flex items-center" onClick={goBack}>
          <ArrowLeft className="mr-2 w-5 h-5" />
          <span className="font-medium">Back to knowledge</span>
        </button>
      </div>

      <section className="border-gray-150 border bg-white dark:bg-gray-950 rounded-xl mb-8">
        <button
          className="group flex w-full items-center px-6 py-4 text-left text-base leading-tight"
          type="button"
        >
          <div className="items-center overflow-hidden md:flex">
            <div className="mr-3.5 flex items-center">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="mr-3 flex-none"
                width="1em"
                height="1em"
                fill="currentColor"
                viewBox="0 0 32 32"
              >
                <path d="M28.9244 11.6919H20.3081V3.0756H28.9244V11.6919ZM22.4622 9.5378H26.7703V5.22967H22.4622V9.5378Z" />
                <path d="M17.077 14.923V8.46076H8.46076V23.5392H23.5392V14.923H17.077ZM10.6148 10.6148H14.923V14.923H10.6148V10.6148ZM14.923 21.3852H10.6148V17.077H14.923V21.3852ZM21.3852 21.3852H17.077V17.077H21.3852V21.3852Z" />
                <path d="M26.7703 28.9244H5.22967C4.65855 28.9238 4.11098 28.6967 3.70714 28.2929C3.3033 27.889 3.07617 27.3414 3.0756 26.7703V5.22967C3.07617 4.65855 3.3033 4.11098 3.70714 3.70714C4.11098 3.3033 4.65855 3.07617 5.22967 3.0756H16V5.22967H5.22967V26.7703H26.7703V16H28.9244V26.7703C28.9238 27.3414 28.6967 27.889 28.2929 28.2929C27.889 28.6967 27.3414 28.9238 26.7703 28.9244Z" />
              </svg>
              <div className="flex-none font-semibold">Document Structure</div>
            </div>
          </div>
          <div className="mx-6 flex-1 md:border-t" />
        </button>
        <div className="">
          <div className="px-6 pb-6 pt-2">
            <div className="space-y-6">
              <label className="block ">
                <Label className="mb-3">Document Type</Label>
                <div className="-mt-1.5 mb-3 text-sm">
                  To ensure proper processing of your upload, please choose the
                  correct document type from the options below. Select the
                  category that best matches the content of the document you are
                  about to upload.
                </div>
                <Select
                  placeholder={'Select document type'}
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
                      name: ' Blog Article',
                      value: RapidaDocumentType.STRUCTURE_ARTICLE,
                    },
                  ]}
                  onChange={e =>
                    knowledgeDocumentAction.onChangeDocumentType(
                      e.target.value as RapidaDocumentType,
                    )
                  }
                ></Select>
              </label>
            </div>
          </div>
        </div>
      </section>

      {/* Deploy Model */}
      <section className="border-gray-150 border bg-white dark:bg-gray-950 rounded-xl mb-8">
        <button
          className="group flex w-full items-center px-6 py-4 text-left text-base leading-tight"
          type="button"
        >
          <div className="items-center overflow-hidden md:flex">
            <div className="mr-3.5 flex items-center">
              <UploadIcon width="1em" height="1em" className="mr-3 flex-none" />
              <div className="flex-none font-semibold">Documents</div>
            </div>
          </div>
          <div className="mx-6 flex-1 md:border-t" />
        </button>

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
            Upload your text files (e.g., .csv, .xls, .json). Maximum file size:
            10 MB per file.
          </div>
        </div>

        {/* Hardware Configuration */}
        <div className="p-6">
          <ManualFile
            accepts={{
              'application/vnd.ms-excel': ['.xls', '.xlsx'],
              'text/csv': ['.csv'],
              'application/json': ['.json'],
            }}
          />
        </div>
      </section>
      <section className="border-gray-150 border bg-white dark:bg-gray-950 rounded-xl mb-16">
        <div className="flex items-center justify-between px-6">
          <div>
            <ErrorMessage message={errorMessage} className="rounded-none!" />
          </div>
          <div className="flex space-x-3 py-6">
            <HoverButton
              onClick={() => goBack()}
              className="text-blue-600 hover:text-gray-600 dark:hover:text-gray-300"
            >
              Cancel
            </HoverButton>
            <OutlineButton
              isLoading={loading}
              type="button"
              onClick={onCreateKnowledgeDocument}
            >
              Upload new document
            </OutlineButton>
          </div>
        </div>
      </section>
    </div>
  );
}
