import { SideTab } from '@/app/components/Tab';
import { cn } from '@/styles/media';
import { toContentText } from '@/utils/rapida_content';
import { FC } from 'react';
import { MessageMetadatas } from './message-metadatas';
import { MessageMetrics } from './message-metrics';
import { AssistantConversationMessage } from '@rapidaai/react';
import { MarkdownViewer } from '@/app/components/MarkdownViewer';

/**
 *
 * @param param0
 * @returns
 */
export const TraceDetail: FC<{
  currentAssistantMessage: AssistantConversationMessage;
}> = ({ currentAssistantMessage }) => {
  return (
    <div className="flex p-3.5 dark:bg-gray-950 bg-gray-100">
      <div className="flex h-full w-full border dark:bg-gray-900 bg-white">
        <SideTab
          strict={false}
          active="Request"
          className={cn('w-56')}
          tabs={[
            {
              label: 'Request',
              element: (
                <div className="flex-1 p-4 min-w-full">
                  {currentAssistantMessage.getRequest() ? (
                    <div className="border rounded-[2px]">
                      <MarkdownViewer
                        text={toContentText(
                          currentAssistantMessage
                            .getRequest()
                            ?.getContentsList(),
                        )}
                      />
                    </div>
                  ) : (
                    <div className="opacity-60 w-full p-4">
                      Request will be available here after the completion of
                      execution
                    </div>
                  )}
                </div>
              ),
            },
            {
              label: 'Response',
              element: (
                <div className="flex-1 p-4 min-w-full">
                  {currentAssistantMessage.getResponse() ? (
                    <div className="border rounded-[2px]">
                      <MarkdownViewer
                        text={toContentText(
                          currentAssistantMessage
                            .getResponse()
                            ?.getContentsList(),
                        )}
                      />
                    </div>
                  ) : (
                    <div className="opacity-60 w-full p-4">
                      Response will be available here after the completion of
                      execution
                    </div>
                  )}
                </div>
              ),
            },
            {
              label: 'metrics',
              element: (
                <MessageMetrics
                  metrics={currentAssistantMessage.getMetricsList()}
                />
              ),
            },
            {
              label: 'metadata',
              element: (
                <div className="gap-4">
                  <MessageMetadatas
                    metadata={currentAssistantMessage.getMetadataList()}
                  />
                </div>
              ),
            },
          ]}
        />
      </div>
    </div>
  );
};
