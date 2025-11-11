import React, { useState, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';

import {
  ConnectionConfig,
  GetAssistantToolLog,
  GetAssistantToolLogRequest,
  AssistantToolLog,
} from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { Tab } from '@/app/components/Tab';
import { cn } from '@/styles/media';
import { ChevronRight } from 'lucide-react';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface ToolLogModalProps extends ModalProps {
  currentActivityId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function ToolLogDialog(props: ToolLogModalProps) {
  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [activity, setActivity] = useState<AssistantToolLog | null>(null);
  const getActivity = (currentProject: string, currentActivityId) => {
    const request = new GetAssistantToolLogRequest();
    request.setProjectid(currentProject);
    request.setId(currentActivityId);
    GetAssistantToolLog(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
    ).then(at => {
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
    });
  };

  /**
   *
   */
  useEffect(() => {
    showLoader('overlay');
    getActivity(projectId, props.currentActivityId);
  }, [projectId, props.currentActivityId]);

  return (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-2/3 xl:w-1/3 flex-1"
    >
      <div className="flex items-center p-4 border-b">
        <div className="font-medium text-lg">Log</div>
        <ChevronRight size={18} className="mx-2" />
        <div className="font-medium text-lg">Tool</div>
        <ChevronRight size={18} className="mx-2" />
        <div className="font-medium text-base">{props.currentActivityId}</div>
      </div>
      <div className="relative overflow-auto h-[calc(100vh-50px)]">
        <Tab
          active="Definition"
          className={cn(
            'text-sm',
            'bg-gray-50 border-b dark:bg-gray-900 dark:border-gray-800 sticky top-0 z-1',
          )}
          tabs={[
            {
              label: 'Definition',
              element: (
                <div className="flex-1 flex space-y-8">
                  {activity && (
                    <CodeHighlighting
                      code={JSON.stringify(
                        {
                          name: activity.getAssistanttool()?.getName(),
                          description: activity
                            .getAssistanttool()
                            ?.getDescription(),
                          parameters: activity
                            .getAssistanttool()
                            ?.getFields()
                            ?.toJavaScript(),
                        },
                        null,
                        2,
                      )}
                    />
                  )}
                </div>
              ),
            },
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
