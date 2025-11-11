import React, { useState, useEffect } from 'react';
import { TableText } from '@/app/components/Table/Body';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';
import {
  ConnectionConfig,
  GetKnowledgeLog,
  GetKnowledgeLogRequest,
  KnowledgeLog,
} from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { Tab } from '@/app/components/Tab';
import { cn } from '@/styles/media';
import { DateTimeColumn } from '@/app/components/Table/DateColumn';
import { ChevronRight } from 'lucide-react';
import { StatusIndicator } from '@/app/components/indicators/status';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface KnowledgeLogModalProps extends ModalProps {
  currentActivityId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function KnowledgeLogDialog(props: KnowledgeLogModalProps) {
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [activity, setActivity] = useState<KnowledgeLog | null>(null);

  const getActivity = (currentProject: string, currentActivityId) => {
    const request = new GetKnowledgeLogRequest();
    request.setId(currentActivityId);
    request.setProjectid(currentProject);
    return GetKnowledgeLog(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
    )
      .then(at => {
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
      })
      .catch(x => {
        hideLoader();
        toast.error('Unable to resolve the request, please try again later.');
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
        <div className="font-medium text-lg">LLM</div>
        <ChevronRight size={18} className="mx-2" />
        <div className="font-medium text-base">{props.currentActivityId}</div>
      </div>
      <div className="relative overflow-auto h-[calc(100vh-50px)]">
        <Tab
          active="Metadata"
          className={cn(
            'text-sm',
            'bg-gray-50 border-b dark:bg-gray-900 dark:border-gray-800 sticky top-0 z-1',
          )}
          tabs={[
            {
              label: 'Metadata',
              element: (
                <div className="flex-1 px-4 space-y-8">
                  <section>
                    {activity && (
                      <div className="grid grid-cols-2 gap-4 mt-4">
                        <div className="space-y-1">
                          <div className="capitalize font-semibold">Status</div>
                          <div className="">
                            <StatusIndicator state={activity.getStatus()} />
                          </div>
                        </div>
                        <div className="space-y-1">
                          <div className="capitalize font-semibold">
                            Time Taken
                          </div>
                          <div className="">
                            <TableText>
                              {`${Number(activity.getTimetaken()) / 1000000}ms`}{' '}
                            </TableText>
                          </div>
                        </div>
                        <div className="space-y-1">
                          <div className="capitalize font-semibold">
                            Request Created Time
                          </div>
                          <div className="">
                            <DateTimeColumn date={activity.getCreateddate()} />
                          </div>
                        </div>

                        {/*  */}
                      </div>
                    )}
                  </section>
                </div>
              ),
            },
            {
              label: 'Request',
              element: (
                <CodeHighlighting
                  lang="json"
                  lineNumbers={false}
                  foldGutter={false}
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
                  lang="json"
                  lineNumbers={false}
                  foldGutter={false}
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
