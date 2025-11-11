import { Helmet } from '@/app/components/Helmet';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { FC, HTMLAttributes, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Outlet, useParams } from 'react-router-dom';
import { cn, toHumanReadableRelativeTime } from '@/styles/media';
import {
  AssistantDefinition,
  ConnectionConfig,
  GetAssistant,
  GetAssistantRequest,
} from '@rapidaai/react';
import { useAssistantPageStore } from '@/hooks/use-assistant-page-store';
import { TabLink } from '@/app/components/tab-link';
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import {
  Bolt,
  ChevronsLeftRightEllipsis,
  Code,
  ExternalLink,
  GitPullRequestCreate,
  Plus,
} from 'lucide-react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { ErrorContainer } from '@/app/components/error-container';
import { connectionConfig } from '@/configs';
import { Popover } from '@/app/components/Popover';
/**
 *
 * @returns
 */
export const AssistantViewLayout: FC<HTMLAttributes<HTMLDivElement>> = () => {
  /**
   * authentication information
   */
  const [userId, token, projectId] = useCredential();

  /**
   * global loader
   */
  const { showLoader, hideLoader } = useRapidaStore();

  /**
   * zustand state for the page / this also contains of listing
   */
  const assistantAction = useAssistantPageStore();

  /**
   * get all the models when type change
   */
  const { assistantId } = useParams();

  //
  const [createVersionPopover, setCreateVersionPopover] = useState(false);

  /**
   * navigation
   */
  const {
    goToAssistantPreview,
    goToCreateAssistantVersion,
    goToCreateAssistantWebsocketVersion,
    goToCreateAssistantAgentKitVersion,
    goToAssistantListing,
    goToManageAssistant,
  } = useGlobalNavigation();

  /**
   *
   */

  const [unknownState, setUnknownState] = useState(false);
  /**
   *
   */
  useEffect(() => {
    assistantAction.clear();
    if (assistantId) {
      showLoader();
      const request = new GetAssistantRequest();
      const assistantDef = new AssistantDefinition();
      assistantDef.setAssistantid(assistantId);
      request.setAssistantdefinition(assistantDef);
      GetAssistant(
        connectionConfig,
        request,
        ConnectionConfig.WithDebugger({
          authorization: token,
          userId: userId,
          projectId: projectId,
        }),
      )
        .then(epmr => {
          hideLoader();
          if (epmr?.getSuccess()) {
            let assistant = epmr.getData();
            if (assistant) assistantAction.onChangeCurrentAssistant(assistant);
          } else {
            setUnknownState(true);
            const errorMessage =
              'Unable to get your assistant. please try again later.';
            const error = epmr?.getError();
            if (error) {
              toast.error(error.getHumanmessage());
              return;
            }
            toast.error(errorMessage);
            return;
          }
        })
        .catch(err => {
          hideLoader();
        });
    }
  }, [assistantId]);

  if (unknownState)
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={goToAssistantListing}
          code="403"
          actionLabel="Go to listing"
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );

  //
  return (
    <div className={cn('flex flex-col h-full flex-1 overflow-auto')}>
      <Helmet title="Hosted Assistant" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>
            Assistant<span className="px-1">/</span>
            {assistantAction.currentAssistant?.getName()}{' '}
          </PageTitleBlock>
          <div className="text-xs opacity-75">
            {assistantAction.currentAssistant
              ?.getAssistantprovidermodel()
              ?.getCreateddate() &&
              toHumanReadableRelativeTime(
                assistantAction.currentAssistant
                  ?.getAssistantprovidermodel()
                  ?.getCreateddate()!,
              )}
          </div>
        </div>
        {assistantAction.currentAssistant && (
          <div className="flex divide-x dark:divide-gray-800">
            <div className="flex border-l">
              <IBlueButton
                className={cn(
                  'px-4',
                  createVersionPopover &&
                    'bg-light-background!  dark:bg-gray-950!',
                )}
                onClick={() => {
                  setCreateVersionPopover(true);
                }}
              >
                Create New Version
                <GitPullRequestCreate className="w-4 h-4 ml-2" />
              </IBlueButton>
              <Popover
                align={'bottom-end'}
                className="w-auto"
                open={createVersionPopover}
                setOpen={setCreateVersionPopover}
              >
                <div className="space-y-0.5">
                  <IButton
                    className="w-full justify-start"
                    onClick={() => goToCreateAssistantVersion(assistantId!)}
                  >
                    <Plus className="w-4 h-4 mr-2" /> Create New version
                  </IButton>
                  <IButton
                    className="w-full justify-start"
                    onClick={() =>
                      goToCreateAssistantWebsocketVersion(assistantId!)
                    }
                  >
                    <ChevronsLeftRightEllipsis className="w-4 h-4 mr-2" />{' '}
                    Connect new Websocket
                  </IButton>

                  <IButton
                    className="w-full justify-start"
                    onClick={() =>
                      goToCreateAssistantAgentKitVersion(assistantId!)
                    }
                  >
                    <Code className="w-4 h-4 mr-2" /> Connect new AgentKit
                  </IButton>
                </div>
              </Popover>
            </div>

            <IButton onClick={() => goToManageAssistant(assistantId!)}>
              Configure assistant
              <Bolt className="w-4 h-4 ml-1.5" strokeWidth={1.5} />
            </IButton>

            <IButton onClick={() => goToAssistantPreview(assistantId!)}>
              Preview
              <ExternalLink className="w-4 h-4 ml-1.5" strokeWidth={1.5} />
            </IButton>
          </div>
        )}
      </PageHeaderBlock>
      <div
        className={cn(
          'sticky top-0 z-3',
          'bg-white border-t border-b dark:bg-gray-900 dark:border-gray-800',
        )}
      >
        <div className="flex items-center divide-x border-r w-fit">
          <TabLink to={`/deployment/assistant/${assistantId}/overview`}>
            Overview
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/sessions`}>
            Sessions
          </TabLink>
          <TabLink to={`/deployment/assistant/${assistantId}/version-history`}>
            Versions
          </TabLink>
        </div>
      </div>
      <Outlet />
    </div>
  );
};
