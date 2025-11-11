import { Helmet } from '@/app/components/Helmet';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { Tab } from '@/app/components/Tab';
import { Documents } from './documents';
import { ConnectionConfig, GetKnowledgeBase } from '@rapidaai/react';
import { GetKnowledgeResponse } from '@rapidaai/react';
import { cn, toHumanReadableRelativeTime } from '@/styles/media';
import { useKnowledgePageStore } from '@/hooks/use-knowledge-page-store';
import { CreateTagDialog } from '@/app/components/base/modal/create-tag-modal';
import { UpdateDescriptionDialog } from '@/app/components/base/modal/update-description-modal';
import { Tag } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { DocumentSegments } from '@/app/pages/knowledge-base/view/document-segments';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { connectionConfig } from '@/configs';
import { CreateKnowledgeDocumentDialog } from '@/app/components/base/modal/create-knowledge-document-modal';
import { IBlueButton } from '@/app/components/Form/Button';
import { PlusIcon } from '@/app/components/Icon/plus';

/**
 *
 * @returns
 */
export function ViewKnowledgePage() {
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [createKnowledgeOpen, setCreateKnowledgeOpen] = useState(false);
  const { onChangeCurrentKnowledge, currentKnowledge, ...knowledgeActions } =
    useKnowledgePageStore();

  /**
   * get all the models when type change
   */

  const { id } = useParams();
  const getKnowledge = useCallback(
    id => {
      if (id) {
        showLoader('overlay');
        GetKnowledgeBase(
          connectionConfig,
          id,
          afterGetKnowledge,
          ConnectionConfig.WithDebugger({
            authorization: token,
            userId: userId,
            projectId: projectId,
          }),
        );
      }
    },
    [id],
  );
  //
  useEffect(() => {
    getKnowledge(id);
  }, [id]);

  const afterGetKnowledge = useCallback(
    (err: ServiceError | null, uvcr: GetKnowledgeResponse | null) => {
      hideLoader();
      if (uvcr?.getSuccess()) {
        const kb = uvcr.getData();
        if (kb) onChangeCurrentKnowledge(kb);
      } else {
        let errorMessage = uvcr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
        }
        toast.error(
          'Unable to get your knowledgebase, please try again later.',
        );
      }
    },
    [],
  );
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
            <div>Knowledge / {currentKnowledge?.getName()} </div>
            <div className="text-xs opacity-75">
              {currentKnowledge?.getCreateddate() &&
                toHumanReadableRelativeTime(
                  currentKnowledge?.getCreateddate()!,
                )}
            </div>
          </div>
          <div className="flex divide-x">
            <IBlueButton
              className="px-4"
              onClick={() => {
                if (currentKnowledge) setCreateKnowledgeOpen(true);
              }}
            >
              Add new document
              <PlusIcon className="w-4 h-4 ml-2" />
            </IBlueButton>
          </div>
        </PageHeaderBlock>

        {/*  */}
        <CreateKnowledgeDocumentDialog
          modalOpen={createKnowledgeOpen}
          setModalOpen={setCreateKnowledgeOpen}
          knowledgeId={currentKnowledge?.getId()!}
          onReload={() => {
            getKnowledge(id);
          }}
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
                element: <Documents currentKnowledge={currentKnowledge} />,
              },
              {
                label: 'segments',
                element: (
                  <DocumentSegments currentKnowledge={currentKnowledge} />
                ),
              },
            ]}
          />
        )}
      </div>
    </>
  );
}
