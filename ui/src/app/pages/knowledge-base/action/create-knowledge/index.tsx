import { FieldSet } from '@/app/components/form/fieldset';
import { Helmet } from '@/app/components/helmet';
import { Input } from '@/app/components/form/input';
import { TagInput } from '@/app/components/form/tag-input';
import { useCreateKnowledgePageStore } from '@/hooks/use-create-knowledge-page-store';
import { useCallback, useEffect, useState } from 'react';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks/use-rapida-store';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ButtonSet } from '@carbon/react';
import { KnowledgeDocument } from '@rapidaai/react';
import { TabForm } from '@/app/components/form/tab-form';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import { Knowledge } from '@rapidaai/react';
import { KnowledgeTags } from '@/app/components/form/tag-input/knowledge-tags';
import toast from 'react-hot-toast/headless';
import { create_knowledge_success_message } from '@/utils/messages';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ManualFile } from '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file';
import { FormLabel } from '@/app/components/form-label/index';
import ConfirmDialog from '@/app/components/base/modal/confirm-ui';
import { useNavigate } from 'react-router-dom';
import { Textarea } from '@/app/components/form/textarea';
import { EmbeddingProvider } from '@/app/components/providers/embedding';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';
import { InputHelper } from '@/app/components/input-helper';
import { SectionDivider } from '@/app/components/blocks/section-divider';

export function CreateKnowledgePage() {
  const [activeTab, setActiveTab] = useState('configure-embedding');
  const [errorMessage, setErrorMessage] = useState('');
  const { goToKnowledge } = useGlobalNavigation();
  const {
    name,
    clear,
    onChangeName,
    description,
    onChangeDescription,
    tags,
    onAddTag,
    onRemoveTag,
    onCreateKnowledge,
    provider,
    onChangeProvider,
    providerParamters,
    onChangeProviderParameter,
  } = useCreateKnowledgePageStore();

  const knowledgeDocumentAction = useCreateKnowledgeDocumentPageStore();
  const [userId, token, projectId] = useCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [knowledge, setKnowledge] = useState<Knowledge | null>(null);

  useEffect(() => {
    clear();
  }, []);

  const onsuccessofcreateknowledge = (kn: Knowledge) => {
    toast.success(create_knowledge_success_message(kn?.getName()));
    setKnowledge(kn);
    hideLoader();
    setActiveTab('add-documents');
  };

  const onerror = useCallback((err: string) => {
    hideLoader();
    setErrorMessage(err);
  }, []);

  const onValidateEmbedding = () => {
    setErrorMessage('');
    setActiveTab('define-knowledge');
  };

  const createKnowledge = () => {
    if (name.trim() === '') {
      setErrorMessage('Please provide a name for your knowledge base.');
      return;
    }
    setErrorMessage('');
    showLoader();
    onCreateKnowledge(
      projectId,
      token,
      userId,
      onsuccessofcreateknowledge,
      onerror,
    );
  };

  const onsuccessknowledgedocument = useCallback(
    (d: KnowledgeDocument[]) => {
      hideLoader();
      if (knowledge?.getId()) goToKnowledge(knowledge?.getId());
    },
    [JSON.stringify(knowledge)],
  );

  const createKnowledgeDocument = () => {
    if (knowledge) {
      showLoader();
      knowledgeDocumentAction.onCreateKnowledgeDocument(
        knowledge.getId(),
        projectId,
        token,
        userId,
        onsuccessknowledgedocument,
        onerror,
      );
    }
  };

  const navigator = useNavigate();
  const [isShow, setIsShow] = useState(false);

  return (
    <>
      <Helmet title="Create a knowledge" />

      <ConfirmDialog
        showing={isShow}
        type="warning"
        title="Are you sure?"
        content="You want to cancel creating the knowledge? Any unsaved changes will be lost."
        confirmText="Confirm"
        cancelText="Cancel"
        onConfirm={() => navigator(-1)}
        onCancel={() => setIsShow(false)}
        onClose={() => setIsShow(false)}
      />

      <TabForm
        activeTab={activeTab}
        formHeading="Complete all steps to create a knowledge base."
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            name: 'Configure Embedding',
            description:
              'Select the embedding provider and model that will be used to vectorise your documents.',
            code: 'configure-embedding',
            body: (
              <>
                <DocNoticeBlock docPath="/knowledge/overview">
                  A knowledge base is a curated collection of documents grouped
                  by domain or topic. The embedding provider controls how your
                  documents are vectorised for semantic retrieval.
                </DocNoticeBlock>
                <div className="px-8 pt-6 pb-8 max-w-4xl flex flex-col gap-8">
                  <div className="flex flex-col gap-6">
                    <SectionDivider label="Embedding Configuration" />
                    <EmbeddingProvider
                      onChangeParameter={onChangeProviderParameter}
                      onChangeProvider={onChangeProvider}
                      parameters={providerParamters}
                      provider={provider}
                    />
                  </div>
                </div>
              </>
            ),
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => setIsShow(true)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton size="lg" onClick={onValidateEmbedding}>
                  Define knowledge profile
                </PrimaryButton>
              </ButtonSet>,
            ],
          },
          {
            name: 'Define Knowledge Profile',
            description:
              'Give your knowledge base a name, description, and labels to make it easy to find and manage.',
            code: 'define-knowledge',
            body: (
              <div className="px-8 pt-8 pb-8 max-w-2xl flex flex-col gap-10">
                {/* Identity section */}
                <div className="flex flex-col gap-6">
                  <SectionDivider label="Identity" />

                  <FieldSet>
                    <div className="flex items-baseline justify-between">
                      <FormLabel
                        htmlFor="knowledge_name"
                        className="text-xs tracking-wide uppercase"
                      >
                        Knowledge name{' '}
                        <span className="text-red-500 ml-0.5 normal-case">
                          *
                        </span>
                      </FormLabel>
                      <span className="text-xs text-gray-500 dark:text-gray-400 tabular-nums">
                        {name.length}/100
                      </span>
                    </div>
                    <Input
                      name="knowledge_name"
                      maxLength={100}
                      onChange={e => onChangeName(e.target.value)}
                      value={name}
                      placeholder="e.g. product-documentation"
                    />
                    <InputHelper>
                      A unique name for this knowledge base. Use lowercase
                      letters, numbers, and hyphens.
                    </InputHelper>
                  </FieldSet>

                  <FieldSet>
                    <FormLabel
                      htmlFor="description"
                      className="text-xs tracking-wide uppercase"
                    >
                      Description
                    </FormLabel>
                    <Textarea
                      row={4}
                      name="description"
                      value={description}
                      placeholder="What does this knowledge base contain? What topics does it cover?"
                      onChange={t => onChangeDescription(t.target.value)}
                    />
                    <InputHelper>
                      A clear description helps your team understand the purpose
                      and content of this knowledge base.
                    </InputHelper>
                  </FieldSet>
                </div>

                {/* Labels section */}
                <div className="flex flex-col gap-6">
                  <div className="flex items-center gap-3">
                    <span className="text-[10px] font-semibold tracking-[0.12em] uppercase text-gray-500 dark:text-gray-400 whitespace-nowrap">
                      Labels
                    </span>
                    <div className="flex-1 h-px bg-gray-100 dark:bg-gray-800" />
                    {tags.length > 0 && (
                      <span className="text-xs tabular-nums bg-primary/10 text-primary px-2 py-0.5 rounded-full font-medium">
                        {tags.length}
                      </span>
                    )}
                  </div>
                  <TagInput
                    tags={tags}
                    addTag={onAddTag}
                    allTags={KnowledgeTags}
                    removeTag={onRemoveTag}
                  />
                  <InputHelper>
                    Tags help you organise and filter knowledge bases across
                    your workspace.
                  </InputHelper>
                </div>
              </div>
            ),
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton size="lg" onClick={() => setIsShow(true)}>
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  onClick={createKnowledge}
                  isLoading={loading}
                >
                  Create knowledge
                </PrimaryButton>
              </ButtonSet>,
            ],
          },
          {
            name: 'Add Documents',
            description:
              'Upload documents to populate your knowledge base. You can skip this and add documents later.',
            code: 'add-documents',
            body: <ManualFile />,
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => {
                    if (knowledge?.getId()) goToKnowledge(knowledge?.getId());
                  }}
                >
                  Skip
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  onClick={createKnowledgeDocument}
                  isLoading={loading}
                >
                  Upload documents
                </PrimaryButton>
              </ButtonSet>,
            ],
          },
        ]}
      />
    </>
  );
}
