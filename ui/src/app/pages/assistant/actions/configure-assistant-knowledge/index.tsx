import { GhostButton, PrimaryButton } from '@/app/components/ui/primitives';
import { useRapidaStore } from '@/stores/app';
import { FC, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { EmptyState } from '@/app/components/ui/feedback';
import { SelectKnowledgeCard } from '@/app/components/domain/cards/knowledge-card';
import { Add, Information, Launch } from '@carbon/icons-react';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/layout/blocks/page-title-block';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useAssistantKnowledgePageStore } from '@/stores/assistant/actions';
import { SectionLoader } from '@/app/components/ui/feedback';
import { TablePagination } from '@/app/components/ui/table';
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
        <div className="flex items-stretch h-12 border-l border-border-subtle ">
          <TablePagination
            className="py-0 border-r border-border-subtle "
            currentPage={axtion.page}
            onChangeCurrentPage={axtion.setPage}
            totalItem={axtion.totalCount}
            pageSize={axtion.pageSize}
            onChangePageSize={axtion.setPageSize}
            onChangeColumns={axtion.setColumns}
          />
          <GhostButton
            size="md"
            type="button"
            onClick={() => navigation.goToCreateKnowledge()}
            className="h-full whitespace-nowrap border-r border-border-subtle"
            renderIcon={Launch}
          >
            Create new knowledge
          </GhostButton>
          <PrimaryButton
            size="md"
            type="button"
            onClick={() => navigation.goToCreateAssistantKnowledge(assistantId)}
            className="h-full whitespace-nowrap"
            renderIcon={Add}
          >
            Connect knowledge
          </PrimaryButton>
        </div>
      </PageHeaderBlock>
      <div
        className="flex items-center p-2 px-4 text-support-info border-l-4 border-support-info bg-support-info/10 "
        role="alert"
      >
        <Information className="shrink-0 w-4 h-4" />
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
                className="col-span-1 bg-surface max-w-none"
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
                      <span className="text-support-error">
                        Delete knowledge
                      </span>
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
            <EmptyState
              title="No Context"
              subtitle="There are no Knowledge given added to the context"
              action="Connect knowlege"
              onAction={() => {
                navigation.goToCreateAssistantKnowledge(assistantId);
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
};
