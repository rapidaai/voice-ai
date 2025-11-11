import { IBlueButton, IButton } from '@/app/components/Form/Button';
import { useRapidaStore } from '@/hooks';
import { FC, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { SelectKnowledgeCard } from '@/app/components/base/cards/knowledge-card';
import { ExternalLink, Info, Plus } from 'lucide-react';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useAssistantKnowledgePageStore } from '@/app/pages/assistant/actions/store/use-knowledge-page-store';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { UpdateKnowledge } from '@/app/pages/assistant/actions/configure-assistant-knowledge/update-assistant-knowledge';
import { CreateKnowledge } from '@/app/pages/assistant/actions/configure-assistant-knowledge/create-assistant-knowledge';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';

export function ConfigureAssistantKnowledgePage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantKnowledge assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantKnowledgePage() {
  const { assistantId } = useParams();
  return <>{assistantId && <CreateKnowledge assistantId={assistantId} />}</>;
}

export function UpdateAssistantKnowledgePage() {
  const { assistantId } = useParams();
  return <>{assistantId && <UpdateKnowledge assistantId={assistantId} />}</>;
}

const ConfigureAssistantKnowledge: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const axtion = useAssistantKnowledgePageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();

  useEffect(() => {
    showLoader('block');
    get();
  }, []);

  const get = () => {
    axtion.getAssistantKnowledge(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        hideLoader();
      },
    );
  };

  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Are you sure?',
    content:
      'You want to delete? The knowledge will disconnected from assistant.',
  });

  const deleteAssistantKnowledge = (
    assistantId: string,
    assistantKnowledgeId: string,
  ) => {
    showLoader('block');
    axtion.deleteAssistantKnowledge(
      assistantId,
      assistantKnowledgeId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        toast.success('Assistant knowledge disconnected successfully');
        get();
      },
    );
  };

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="relative flex flex-col flex-1">
      <ConfirmDialogComponent />
      <PageHeaderBlock>
        <PageTitleBlock>Configure Knowledge</PageTitleBlock>
        <div className="flex divide-x">
          <TablePagination
            className="py-0"
            currentPage={axtion.page}
            onChangeCurrentPage={axtion.setPage}
            totalItem={axtion.totalCount}
            pageSize={axtion.pageSize}
            onChangePageSize={axtion.setPageSize}
            onChangeColumns={axtion.setColumns}
          />
          <IBlueButton
            onClick={() => {
              navigation.goToCreateAssistantKnowledge(assistantId);
            }}
          >
            Connect knowledge
            <Plus className="w-4 h-4 ml-1.5" />
          </IBlueButton>
          <IButton onClick={() => navigation.goToCreateKnowledge()}>
            Create new knowledge
            <ExternalLink className="w-4 h-4 ml-1.5" />
          </IButton>
        </div>
      </PageHeaderBlock>
      <div
        className="flex items-center p-2 px-4 text-blue-800 border-l-4 border-blue-300 bg-blue-50 dark:text-blue-400 dark:bg-gray-800 dark:border-blue-800"
        role="alert"
      >
        <Info className="shrink-0 w-4 h-4" />
        <div className="ms-3 text-sm font-medium">
          Provide the specific knowledge your assistant will use to deliver
          accurate and relevant answers.
        </div>
      </div>
      <div className="overflow-auto flex flex-col flex-1 pb-20">
        {axtion.knowledges.length > 0 ? (
          <div className="p-2 grid sm:grid-cols-2 lg:grid-cols-4 gap-3 w-full">
            {axtion.knowledges.map((itm, idx) => (
              <SelectKnowledgeCard
                className="col-span-1 bg-white max-w-none"
                knowledge={itm.getKnowledge()!}
                key={`knolwedge-card-${idx}`}
                knowledgeOptions={[
                  {
                    option: 'Update knowledge',
                    onActionClick: () => {
                      navigation.goToEditAssistantKnowledge(
                        assistantId,
                        itm.getId(),
                      );
                    },
                  },
                  {
                    option: (
                      <span className="text-rose-600">Delete knowledge</span>
                    ),
                    onActionClick: () => {
                      showDialog(() => {
                        deleteAssistantKnowledge(assistantId, itm.getId());
                      });
                    },
                  },
                ]}
              />
            ))}
          </div>
        ) : (
          <div className="my-auto mx-auto">
            <ActionableEmptyMessage
              title="No Context"
              subtitle="There are no Knowledge given added to the context"
              action="Connect knowlege"
              onActionClick={() => {
                navigation.goToCreateAssistantKnowledge(assistantId);
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
};
