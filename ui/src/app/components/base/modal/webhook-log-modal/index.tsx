import React, { useState, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ConnectionConfig, ServiceError } from '@rapidaai/react';
import { Tab } from '@/app/components/Tab';
import { cn } from '@/styles/media';
import { ChevronRight } from 'lucide-react';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { GetWebhookLog } from '@rapidaai/react';
import {
  AssistantWebhookLog,
  GetAssistantWebhookLogResponse,
} from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface WebhookLogModalProps extends ModalProps {
  currentWebhookId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function WebhookLogDialog(props: WebhookLogModalProps) {
  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [activity, setActivity] = useState<AssistantWebhookLog | null>(null);

  const getActivity = (currentProject: string, currentActivityId) => {
    return GetWebhookLog(
      connectionConfig,
      currentProject,
      currentActivityId,
      afterActivities,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: userId,
        projectId: projectId,
      }),
    );
  };

  /**
   *
   */
  useEffect(() => {
    showLoader('overlay');
    getActivity(projectId, props.currentWebhookId);
  }, [projectId, props.currentWebhookId]);

  /**
   *
   * @param err
   * @param at
   */
  const afterActivities = (
    err: ServiceError | null,
    at: GetAssistantWebhookLogResponse | null,
  ) => {
    hideLoader();
    if (at?.getSuccess()) {
      let data = at.getData();
      if (data) {
        setActivity(data);
      }
    } else {
      let err = at?.getError();
      if (err) toast.error(err?.getHumanmessage());
      toast.error('Unable to resolve the request, please try again later.');
    }
  };

  return (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-2/3 xl:w-1/3 flex-1"
    >
      <div className="flex items-center p-4 border-b">
        <div className="font-medium text-lg">Log</div>
        <ChevronRight size={18} className="mx-2" />
        <div className="font-medium text-lg">Webhook</div>
        <ChevronRight size={18} className="mx-2" />
        <div className="font-medium text-base">{props.currentWebhookId}</div>
      </div>
      <div className="relative overflow-auto h-[calc(100vh-50px)]">
        <Tab
          active="Request"
          className={cn(
            'text-sm',
            'bg-gray-50 border-b dark:bg-gray-900 dark:border-gray-800 sticky top-0 z-1',
          )}
          tabs={[
            {
              label: 'Request',
              element: (
                <CodeHighlighting
                  code={JSON.stringify(
                    activity?.getRequest()?.toJavaScript(),
                    null,
                    2,
                  )}
                />
              ),
            },
            {
              label: 'Response',
              element: (
                <CodeHighlighting
                  code={JSON.stringify(
                    activity?.getResponse()?.toJavaScript(),
                    null,
                    2,
                  )}
                />
              ),
            },
          ]}
        />
      </div>
    </RightSideModal>
  );
}
