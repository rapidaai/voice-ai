import { Assistant } from '@rapidaai/react';
import { AssistantConversationMessage } from '@rapidaai/react';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { Analytics } from '@/app/pages/assistant/view/overview/analytics';
import { useCredential, useRapidaStore } from '@/hooks';
import { useAssistantTracePageStore } from '@/hooks/use-assistant-trace-page-store';
import { toDateString } from '@/utils';
import { FC, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { ExternalLink, Info } from 'lucide-react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';

/**
 *
 * @param props
 * @returns
 */
export const Overview: FC<{ currentAssistant: Assistant }> = (props: {
  currentAssistant: Assistant;
}) => {
  const [userId, token, projectId] = useCredential();
  const rapidaContext = useRapidaStore();
  const navigation = useGlobalNavigation();
  const assistantTraceAction = useAssistantTracePageStore();

  const getDateRangeCriteria = (range: string) => {
    const now = new Date();
    let startDate: Date;

    switch (range) {
      case 'last_24_hours':
        startDate = new Date(now.setDate(now.getDate() - 1));
        break;
      case 'last_3_days':
        startDate = new Date(now.setDate(now.getDate() - 3));
        break;
      case 'last_7_days':
        startDate = new Date(now.setDate(now.getDate() - 7));
        break;
      case 'last_30_days':
      default:
        startDate = new Date(now.setDate(now.getDate() - 30));
        break;
    }

    return {
      k: 'assistant_conversation_messages.created_date',
      v: toDateString(startDate),
      logic: '>=',
    };
  };

  useEffect(() => {
    assistantTraceAction.clear();
    assistantTraceAction.addCriterias([getDateRangeCriteria('last_30_days')]);
  }, []);

  useEffect(() => {
    rapidaContext.showLoader();
    assistantTraceAction.setPageSize(0);
    assistantTraceAction.setFields(['metadata', 'metric']);
    assistantTraceAction.getAssistantMessages(
      props.currentAssistant.getId(),
      projectId,
      token,
      userId,
      (err: string) => {
        rapidaContext.hideLoader();
        toast.error(err);
      },
      (data: AssistantConversationMessage[]) => {
        rapidaContext.hideLoader();
      },
    );
  }, [
    props.currentAssistant.getId(),
    projectId,
    JSON.stringify(assistantTraceAction.criteria),
    token,
    userId,
  ]);

  if (rapidaContext.loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="flex flex-col flex-1 grow">
      {!props.currentAssistant.getApideployment() &&
        !props.currentAssistant.getDebuggerdeployment() &&
        !props.currentAssistant.getWebplugindeployment() &&
        !props.currentAssistant.getPhonedeployment() && (
          <YellowNoticeBlock className="flex items-center">
            <Info className="shrink-0 w-4 h-4" strokeWidth={1.5} />
            <div className="ms-3 text-sm font-medium">
              <strong className="font-semibold">
                Your assistant is ready, but not live yet,
              </strong>{' '}
              It looks like your assistant isn’t deployed to any channel.
            </div>
            <button
              type="button"
              onClick={() => {
                navigation.goToDeploymentAssistant(
                  props.currentAssistant.getId(),
                );
              }}
              className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
            >
              Enable deployment
              <ExternalLink
                className="shrink-0 w-4 h-4 ml-1.5"
                strokeWidth={1.5}
              />
            </button>
          </YellowNoticeBlock>
        )}
      <Analytics data={assistantTraceAction.assistantMessages} />
    </div>
  );
};
