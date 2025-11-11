import React, { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import toast from 'react-hot-toast/headless';
import SingleAssistant from './single-assistant';
import { useAssistantPageStore } from '@/hooks/use-assistant-page-store';
import { Assistant } from '@rapidaai/react';
import { Spinner } from '@/app/components/Loader/Spinner';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { HowAssistantWorksDialog } from '@/app/components/base/modal/how-it-works-modal/how-assistant-works';
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import { ChevronsLeftRightEllipsis, Code, Plus, RotateCw } from 'lucide-react';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { cn } from '@/styles/media';
import { Popover } from '@/app/components/Popover';

/**
 * Assistant page
 *
 * the list of workflow will be shown here as list
 * the list could contain the private workflow created by you and public workflow that you can discover
 *
 * @returns
 */
export function AssistantPage() {
  /**
   *
   */
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [userId, token, projectId] = useCredential();
  const assistantAction = useAssistantPageStore();
  const navigator = useNavigate();
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   *
   */
  useEffect(() => {
    if (searchParams) {
      const searchParamMap = Object.fromEntries(searchParams.entries());
      Object.entries(searchParamMap).forEach(([key, value]) =>
        assistantAction.addCriteria(key, value, '='),
      );
    }
  }, [searchParams]);

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: Assistant[]) => {
    hideLoader();
  }, []);
  /**
   * call the api
   */
  const getAssistants = useCallback((projectId, token, userId) => {
    showLoader();
    assistantAction.onGetAllAssistant(
      projectId,
      token,
      userId,
      onError,
      onSuccess,
    );
  }, []);

  useEffect(() => {
    getAssistants(projectId, token, userId);
  }, [
    projectId,
    assistantAction.page,
    assistantAction.pageSize,
    assistantAction.criteria,
  ]);

  //
  const [hiw, sethiw] = useState(false);
  const [createAssistantPopover, setCreateAssistantPopover] = useState(false);
  return (
    <div className="h-full flex flex-col overflow-auto flex-1">
      <Helmet title="Assistant" />
      <HowAssistantWorksDialog setModalOpen={sethiw} modalOpen={hiw} />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Assistants</PageTitleBlock>
          <div className="text-xs opacity-75">
            {assistantAction.pageSize}/{assistantAction.totalCount}
          </div>
        </div>
        <div className="flex dark:divide-gray-800 divide-x">
          <IButton
            onClick={() => {
              sethiw(!hiw);
            }}
          >
            How it works?
          </IButton>

          <div className="flex">
            <IBlueButton
              className={cn(
                'px-4',
                createAssistantPopover &&
                  'bg-light-background!  dark:bg-gray-950!',
              )}
              onClick={() => {
                setCreateAssistantPopover(true);
              }}
            >
              Add new assistant
              <Plus strokeWidth={1.5} className="ml-1.5 h-4 w-4" />
            </IBlueButton>
            <Popover
              align={'bottom-end'}
              className="bg-white w-64 mr-10"
              open={createAssistantPopover}
              setOpen={setCreateAssistantPopover}
            >
              <div className="space-y-0.5">
                <IButton
                  className="w-full justify-start"
                  onClick={() =>
                    navigate('/deployment/assistant/create-assistant')
                  }
                >
                  <Plus className="w-4 h-4 mr-2" />
                  <span>Create new Assistant</span>
                </IButton>
                <IButton
                  className="w-full justify-start"
                  onClick={() =>
                    navigate('/deployment/assistant/connect-websocket')
                  }
                >
                  <ChevronsLeftRightEllipsis className="w-4 h-4 mr-2" />
                  <span>Connect new Websocket</span>
                </IButton>

                <IButton
                  className="w-full justify-start"
                  onClick={() =>
                    navigate('/deployment/assistant/connect-agentkit')
                  }
                >
                  <Code className="w-4 h-4 mr-2" />
                  <span>Connect new AgentKit</span>
                </IButton>
              </div>
            </Popover>
          </div>
          {/* <IBlueButton
            onClick={() => {
              navigate('/deployment/assistant/create-assistant');
            }}
          >
            Add new assistant
            <Plus strokeWidth={1.5} className="ml-1.5 h-4 w-4" />
          </IBlueButton> */}
        </div>
      </PageHeaderBlock>
      <BluredWrapper className="sticky top-0 bg-white dark:bg-gray-900 z-11 p-0">
        <SearchIconInput className="bg-light-background" />
        <PaginationButtonBlock>
          <TablePagination
            currentPage={assistantAction.page}
            onChangeCurrentPage={assistantAction.setPage}
            totalItem={assistantAction.totalCount}
            pageSize={assistantAction.pageSize}
            onChangePageSize={assistantAction.setPageSize}
          />
          <IButton
            onClick={() => {
              getAssistants(projectId, token, userId);
            }}
          >
            <RotateCw strokeWidth={1.5} className="h-4 w-4" />
          </IButton>
        </PaginationButtonBlock>
      </BluredWrapper>
      {assistantAction.assistants && assistantAction.assistants.length > 0 ? (
        <section className="grid content-start grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 grow shrink-0 px-4 py-4">
          {assistantAction.assistants.map((ast, idx) => {
            return <SingleAssistant key={idx} assistant={ast} />;
          })}
        </section>
      ) : assistantAction.criteria.length > 0 ? (
        <div className="h-full flex justify-center items-center">
          <ActionableEmptyMessage
            title="No Assistant"
            subtitle=" There are no assistant matching with your criteria."
            action="Create new Assistant"
            onActionClick={() =>
              navigator('/deployment/assistant/create-assistant')
            }
          />
        </div>
      ) : !loading ? (
        <div className="h-full flex justify-center items-center">
          <ActionableEmptyMessage
            title="No Assistant"
            subtitle="There are no Assistants to display"
            action="Create new Assistant"
            onActionClick={() =>
              navigator('/deployment/assistant/create-assistant')
            }
          />
        </div>
      ) : (
        <div className="h-full flex justify-center items-center">
          <Spinner size="md" />
        </div>
      )}
    </div>
  );
}
