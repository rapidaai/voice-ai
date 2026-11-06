import { Helmet } from '@/app/components/app-shell/helmet';
import { useCredential } from '@/hooks/use-credential';
import { useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { Tab } from '@/app/components/ui/primitives';
import { Documents } from './documents';
import { cn } from '@/utils';
import { toHumanReadableRelativeTime } from '@/utils/date';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';
import { CreateTagDialog } from '@/app/components/dialogs/shared/create-tag-modal';
import { UpdateDescriptionDialog } from '@/app/components/dialogs/shared';
import { Tag } from '@rapidaai/react';
import { DocumentSegments } from '@/app/pages/knowledge-base/view/document-segments';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/layout/blocks/page-title-block';
import { CreateKnowledgeDocumentDialog } from '@/app/components/dialogs/knowledge';
import { Add } from '@carbon/icons-react';
import type { KnowledgeViewOutletContext } from './knowledge-view.layout';

/**
 *
 * @returns
 */
export function ViewKnowledgePage() {
  const [userId, token, projectId] = useCredential();
  const [createKnowledgeOpen, setCreateKnowledgeOpen] = useState(false);
  const { currentKnowledge, ...knowledgeActions } = useKnowledgePageStore();
  const { onReload } = useOutletContext<KnowledgeViewOutletContext>();

  return (
    <>
      <UpdateDescriptionDialog
        title="Update knowledge detail"
        name={currentKnowledge?.getName()}
        modalOpen={knowledgeActions.updateDescriptionVisible}
        setModalOpen={knowledgeActions.onHideUpdateDescription}
        description={currentKnowledge?.getDescription()}
        onUpdateDescription={(
          name: string,
          description: string,
          onError: (err: string) => void,
          onSuccess: () => void,
        ) => {
          let wId = currentKnowledge?.getId();
          if (!wId) {
            onError('Knowledge is undefined, please try again later.');
            return;
          }
          knowledgeActions.onUpdateKnowledgeDetail(
            wId,
            name,
            description,
            projectId,
            token,
            userId,
            onError,
            w => {
              onSuccess();
            },
          );
        }}
      />

      <CreateTagDialog
        title="Update knowledge tags"
        tags={currentKnowledge?.getKnowledgetag()?.getTagList()}
        modalOpen={knowledgeActions.editTagVisible}
        setModalOpen={knowledgeActions.onHideEditTagVisible}
        onCreateTag={(
          tags: string[],
          onError: (err: string) => void,
          onSuccess: (e: Tag) => void,
        ) => {
          let wId = currentKnowledge?.getId();
          if (!wId) {
            onError('Knowledge is undefined, please try again later.');
            return;
          }
          knowledgeActions.onEditKnowledgeTag(
            wId,
            tags,
            projectId,
            token,
            userId,
            onError,
            kn => {
              let tags = kn.getKnowledgetag();
              if (tags) onSuccess(tags);
            },
          );
        }}
      />
      <div className={cn('flex flex-col h-full relative overflow-auto')}>
        <Helmet title="Knowledge" />
        <PageHeaderBlock>
          <div className="flex items-center gap-3">
            <PageTitleBlock>{currentKnowledge?.getName()}</PageTitleBlock>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {currentKnowledge?.getCreateddate() &&
                toHumanReadableRelativeTime(
                  currentKnowledge?.getCreateddate()!,
                )}
            </div>
          </div>
          {/* Carbon UI Shell toolbar pattern */}
          <div className="flex items-stretch h-12 border-l border-gray-200 dark:border-gray-800">
            <button
              type="button"
              onClick={() => {
                if (currentKnowledge) setCreateKnowledgeOpen(true);
              }}
              className="flex items-center gap-2 px-4 text-sm text-white bg-primary hover:bg-primary/90 transition-colors whitespace-nowrap"
            >
              Add new document
              <Add className="w-4 h-4" />
            </button>
          </div>
        </PageHeaderBlock>

        {/*  */}
        <CreateKnowledgeDocumentDialog
          modalOpen={createKnowledgeOpen}
          setModalOpen={setCreateKnowledgeOpen}
          knowledgeId={currentKnowledge?.getId()!}
          onReload={onReload}
        />
        {currentKnowledge && (
          <Tab
            active="documents"
            strict={false}
            className={cn(
              'sticky top-0 z-1',
              'bg-white border-t border-b dark:bg-gray-900 dark:border-gray-800',
            )}
            tabs={[
              {
                label: 'documents',
                element: (
                  <Documents
                    currentKnowledge={currentKnowledge}
                    onAddKnowledgeDocument={() => {
                      setCreateKnowledgeOpen(true);
                    }}
                  />
                ),
              },
              {
                label: 'segments',
                element: (
                  <DocumentSegments
                    currentKnowledge={currentKnowledge}
                    onAddKnowledgeDocument={() => {
                      setCreateKnowledgeOpen(true);
                    }}
                  />
                ),
              },
            ]}
          />
        )}
      </div>
    </>
  );
}
