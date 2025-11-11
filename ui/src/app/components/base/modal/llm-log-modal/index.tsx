import React, { useState, useEffect } from 'react';
import { TableText } from '@/app/components/Table/Body';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';

import {
  AuditLog,
  GetAuditLogResponse,
  GetActivity,
  ConnectionConfig,
} from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { Metadata } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { Tab } from '@/app/components/Tab';
import { cn } from '@/styles/media';
import { DateTimeColumn } from '@/app/components/Table/DateColumn';
import { ChevronRight } from 'lucide-react';
import { StatusIndicator } from '@/app/components/indicators/status';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { HttpStatusSpanIndicator } from '@/app/components/indicators/http-status';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface LLMLogModalProps extends ModalProps {
  currentActivityId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function LLMLogDialog(props: LLMLogModalProps) {
  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  /**
   *
   */
  const [additionalData, setAdditionalData] = useState<Metadata[]>([]);
  /**
   *
   */
  const [activity, setActivity] = useState<AuditLog | null>(null);

  const getActivity = (currentProject: string, currentActivityId) => {
    return GetActivity(
      connectionConfig,
      currentProject,
      currentActivityId,
      afterActivities,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
    );
  };

  /**
   *
   */
  useEffect(() => {
    showLoader('overlay');
    getActivity(projectId, props.currentActivityId);
  }, [projectId, props.currentActivityId]);

  /**
   *
   * @param err
   * @param at
   */
  const afterActivities = (
    err: ServiceError | null,
    at: GetAuditLogResponse | null,
  ) => {
    hideLoader();
    if (at?.getSuccess()) {
      let data = at.getData();
      if (data) {
        setActivity(data);
        setAdditionalData(data?.getExternalauditmetadatasList());
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
                              {`${activity.getTimetaken() / 1000000}ms`}{' '}
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
                        <div className="space-y-1">
                          <div className="capitalize font-semibold">
                            Response body
                          </div>
                          <div className="">
                            {activity?.getResponsestatus() && (
                              <HttpStatusSpanIndicator
                                status={activity.getResponsestatus()}
                              />
                            )}
                          </div>
                        </div>

                        {/*  */}
                      </div>
                    )}
                  </section>
                  <div className="font-semibold text-lg border-b -mx-4 px-4 py-2">
                    Additional data
                  </div>
                  <section>
                    <div className="grid grid-cols-2 gap-4">
                      {additionalData.map((ad, idx) => {
                        return (
                          <div className="space-y-1" key={idx}>
                            <div className="capitalize font-semibold">
                              {ad.getKey().replaceAll('_', ' ')}{' '}
                            </div>
                            <div className="">
                              <TableText>{ad.getValue()} </TableText>
                            </div>
                          </div>
                        );
                      })}
                    </div>
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
            {
              label: 'Metrics',
              element: (
                <CodeHighlighting
                  lang="json"
                  lineNumbers={false}
                  foldGutter={false}
                  code={JSON.stringify(
                    activity?.getMetricsList().map(metric => metric.toObject()),
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
