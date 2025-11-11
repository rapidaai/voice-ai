import { IBlueButton, IButton } from '@/app/components/Form/Button';
import { useRapidaStore } from '@/hooks';
import { FC, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { SelectToolCard } from '@/app/components/base/cards/tool-card';
import { ExternalLink, Info, Plus, RotateCw } from 'lucide-react';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { CreateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/create-assistant-tool';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { useAssistantToolPageStore } from '@/app/pages/assistant/actions/store/use-tool-page-store';
import { useCurrentCredential } from '@/hooks/use-credential';
import { UpdateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/update-assistant-tool';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';

export function ConfigureAssistantToolPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <ConfigureAssistantTool assistantId={assistantId} />}</>
  );
}

export function CreateAssistantToolPage() {
  const { assistantId } = useParams();
  return <>{assistantId && <CreateTool assistantId={assistantId} />}</>;
}

export function UpdateAssistantToolPage() {
  const { assistantId } = useParams();
  return <>{assistantId && <UpdateTool assistantId={assistantId} />}</>;
}

const ConfigureAssistantTool: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  let navigator = useGlobalNavigation();
  const navigation = useGlobalNavigation();
  const axtion = useAssistantToolPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();

  useEffect(() => {
    showLoader('block');
    get();
  }, []);

  const get = () => {
    axtion.getAssistantTool(
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
    content: 'You want to delete? The tool will removed from assistant.',
  });

  const deleteAssistantTool = (
    assistantId: string,
    assistantToolId: string,
  ) => {
    showLoader('block');
    axtion.deleteAssistantTool(
      assistantId,
      assistantToolId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        hideLoader();
      },
      v => {
        toast.success('Assistant tool deleted successfully');
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
        <PageTitleBlock>Configure Tools and MCPs</PageTitleBlock>
        <div className="flex divide-x border-l">
          <IBlueButton
            onClick={() => {
              navigator.goToCreateAssistantTool(assistantId);
            }}
          >
            Add another tool
            <Plus className="w-4 h-4 ml-1.5" />
          </IBlueButton>
          <IButton type="button" onClick={() => get()}>
            <RotateCw className="w-4 h-4" strokeWidth={1.5} />
          </IButton>
        </div>
      </PageHeaderBlock>
      <YellowNoticeBlock className="flex items-center">
        <Info className="shrink-0 w-4 h-4" />
        <div className="ms-3 text-sm font-medium">
          Rapida Assistant enables you to deploy intelligent conversational
          agents across multiple channels.
        </div>
        <a
          target="_blank"
          href="https://doc.rapida.ai/assistants/overview"
          className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
          rel="noreferrer"
        >
          Read documentation
          <ExternalLink className="shrink-0 w-4 h-4 ml-1.5" strokeWidth={1.5} />
        </a>
      </YellowNoticeBlock>
      <div className="overflow-auto flex flex-col flex-1 pb-20">
        {axtion.tools.length > 0 ? (
          <div className="p-2 grid sm:grid-cols-2 lg:grid-cols-4 gap-3 w-full">
            {axtion.tools.map((itm, idx) => (
              <SelectToolCard
                className="col-span-1 bg-white h-full"
                tool={itm}
                key={`tool-card-${idx}`}
                options={[
                  {
                    option: 'Edit tool',
                    onActionClick: () => {
                      navigation.goToEditAssistantTool(
                        assistantId,
                        itm.getId(),
                      );
                    },
                  },
                  {
                    option: <span className="text-rose-600">Delete Tool</span>,
                    onActionClick: () => {
                      showDialog(() => {
                        deleteAssistantTool(assistantId, itm.getId());
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
              title="No Tools"
              subtitle="There are no tools given added to the assistant"
              action="Add Tools"
              onActionClick={() => {
                navigation.goToCreateAssistantTool(assistantId);
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
};
