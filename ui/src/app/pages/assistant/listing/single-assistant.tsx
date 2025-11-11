import React, { FC } from 'react';
import { Assistant } from '@rapidaai/react';
import {
  cn,
  toHumanReadableDate,
  toHumanReadableDateFromDate,
  toHumanReadableRelativeDay,
} from '@/styles/media';
import TooltipPlus from '@/app/components/base/tooltip-plus';
import SourceIndicator from '@/app/components/indicators/source';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { AssistantConversation } from '@rapidaai/react';
import { IBlueBGButton } from '@/app/components/Form/Button';

const SingleAssistant: FC<{ assistant: Assistant }> = ({ assistant }) => {
  const gn = useGlobalNavigation();
  return (
    <div className="flex flex-col rounded-[2px] border-[0.1px] bg-white dark:bg-gray-950/20 shadow-sm transition-all hover:shadow-lg relative group">
      <div className="flex justify-between items-start px-4 pt-3 pb-0">
        <div className="flex items-center gap-2.5">
          <div className="flex flex-col space-y-1 w-full relative overflow-hidden">
            <div
              onClick={() => {
                gn.goToAssistant(assistant.getId());
              }}
              className={cn(
                'w-full max-w-full break-words',
                'text-base leading-tight capitalize hover:text-blue-600 hover:cursor-pointer',
              )}
            >
              {assistant.getName()}
            </div>
            <div className="text-sm flex items-centerdark:text-gray-500 space-x-2">
              <span>
                Sessions : {assistant.getAssistantconversationsList().length}
              </span>
              <div className="w-[1.7px] h-3.5 bg-gray-300 dark:bg-gray-600"></div>
              <span>
                Users :{' '}
                {
                  assistant
                    .getAssistantconversationsList()
                    .map(x => x.getIdentifier())
                    .filter(
                      (value, index, self) => self.indexOf(value) === index,
                    ).length
                }
              </span>
            </div>
          </div>
        </div>
        {/* Deployments */}
      </div>
      {/* CHART */}
      <div className="flex pt-2">
        <ConversationChart
          conversations={assistant.getAssistantconversationsList()}
        />
      </div>
      {/* FOOTER SECTION */}
      <div className="w-full flex justify-between items-end mt-0 px-4 py-3 rounded-b-xl">
        {/* Crash Free Sessions */}
        <div className="flex flex-col flex-1">
          <span className="text-sm font-mediumdark:text-gray-400">
            Deployments
          </span>
          <div className="flex flex-wrap gap-1 items-center w-full mt-2">
            {assistant.getApideployment() && (
              <TooltipPlus
                className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0"
                popupContent={
                  <div className="px-3 py-2 text-sm">
                    <span className="text-gray-600 dark:text-gray-500">
                      Api Deployment created on
                    </span>{' '}
                    {toHumanReadableRelativeDay(
                      assistant.getApideployment()?.getCreateddate()!,
                    )}
                  </div>
                }
              >
                <SourceIndicator source={'react-sdk'} withLabel={false} />
              </TooltipPlus>
            )}
            {assistant.getDebuggerdeployment() && (
              <TooltipPlus
                className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0"
                popupContent={
                  <div className="px-3 py-2 text-sm">
                    <span className="text-gray-600 dark:text-gray-500">
                      Debugger enabled on
                    </span>{' '}
                    {toHumanReadableRelativeDay(
                      assistant.getDebuggerdeployment()?.getCreateddate()!,
                    )}
                  </div>
                }
              >
                <SourceIndicator source={'debugger'} withLabel={false} />
              </TooltipPlus>
            )}
            {assistant.getWebplugindeployment() && (
              <TooltipPlus
                className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0"
                popupContent={
                  <div className="px-3 py-2 text-sm">
                    <span className="text-gray-600 dark:text-gray-500">
                      Web plugin Deployment created on
                    </span>{' '}
                    {toHumanReadableRelativeDay(
                      assistant.getWebplugindeployment()?.getCreateddate()!,
                    )}
                  </div>
                }
              >
                <SourceIndicator source={'web-plugin'} withLabel={false} />
              </TooltipPlus>
            )}
            {assistant.getPhonedeployment() && (
              <TooltipPlus
                className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0"
                popupContent={
                  <div className="px-3 py-2 text-sm">
                    <span className="text-gray-600 dark:text-gray-500">
                      Phone Deployment created on
                    </span>{' '}
                    {toHumanReadableRelativeDay(
                      assistant.getPhonedeployment()?.getCreateddate()!,
                    )}
                  </div>
                }
              >
                <SourceIndicator source={'twilio-call'} withLabel={false} />
              </TooltipPlus>
            )}
            {!assistant.getApideployment() &&
              !assistant.getDebuggerdeployment() &&
              !assistant.getWebplugindeployment() &&
              !assistant.getPhonedeployment() && (
                <div className="flex justify-between w-full items-center">
                  <span className="text-gray-600 dark:text-gray-500 text-sm">
                    No deployment
                  </span>
                  <IBlueBGButton
                    className="invisible group-hover:visible h-8 text-sm rounded-[2px]"
                    onClick={event => {
                      event.stopPropagation();
                      gn.goToManageAssistant(assistant.getId());
                    }}
                  >
                    Configure
                  </IBlueBGButton>
                </div>
              )}
          </div>
        </div>
        {/* Latest Deploys */}
      </div>
    </div>
  );
};

const ConversationChart: FC<{
  conversations: Array<AssistantConversation>;
}> = ({ conversations }) => {
  // Group conversations by date and calculate metrics
  const groupedData = conversations.reduce(
    (acc, conversation) => {
      const date = toHumanReadableDate(conversation.getCreateddate()!);
      if (!acc[date]) {
        acc[date] = { activeUsers: new Set(), totalSessions: 0 };
      }
      acc[date].activeUsers.add(conversation.getIdentifier());
      acc[date].totalSessions += 1;
      return acc;
    },
    {} as Record<string, { activeUsers: Set<string>; totalSessions: number }>,
  );

  // Create an array of the last 30 days
  const last30Days = Array.from({ length: 30 }, (_, i) => {
    const date = new Date();
    date.setDate(date.getDate() - i);
    return toHumanReadableDateFromDate(date);
  }).reverse();

  // Create initial data array with all dates
  const dataArray = last30Days.map(date => ({
    date,
    activeUsers: 0,
    totalSessions: 0,
  }));

  // Update dataArray with grouped data
  Object.entries(groupedData).forEach(([date, data]) => {
    const index = dataArray.findIndex(d => d.date === date);
    if (index !== -1) {
      dataArray[index].activeUsers = data.activeUsers.size;
      dataArray[index].totalSessions = data.totalSessions;
    }
  });

  const maxSessions = Math.max(...dataArray.map(d => d.totalSessions));
  const maxHeight = 50; // Maximum height of the bar in pixels

  return (
    <div className="relative w-full h-24 bg-blue-300/10 dark:bg-blue-600/10">
      <div className="absolute top-1/2 left-0 right-0 h-full flex flex-col justify-between z-0">
        <div className="w-full border-t border-dashed border-blue-300/10 opacity-50" />
      </div>
      {dataArray.length > 0 && (
        <div className="absolute z-10 left-0 right-0 top-0 bottom-0 flex items-end">
          {dataArray.map((data, i) => {
            const barHeight =
              (data.totalSessions / maxSessions) * maxHeight || 5;

            return (
              <div key={i} className="flex-1 px-px">
                <TooltipPlus
                  className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0 w-64"
                  popupContent={
                    <div className="divide-y text-sm dark:text-gray-400 text-gray-700">
                      <div className="px-3 py-3 space-y-1.5">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center space-x-1.5">
                            <div className="w-2 h-2 bg-gray-300 rounded-[2px]"></div>
                            <span className="font-medium">Active Users</span>
                          </div>
                          <span className="font-semibold">
                            {data.activeUsers}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <div className="flex items-center space-x-1.5">
                            <div className="w-2 h-2 bg-blue-400 rounded-[2px]"></div>
                            <span className="font-medium">Total Sessions</span>
                          </div>
                          <span className="font-semibold">
                            {data.totalSessions}
                          </span>
                        </div>
                      </div>
                      <div className="px-3 py-2">{data.date}</div>
                    </div>
                  }
                >
                  <div className="h-full grow flex items-end">
                    <div
                      className="bg-blue-600 w-full"
                      style={{
                        height: `${barHeight}px`,
                        opacity:
                          data.totalSessions === 0
                            ? 0.25
                            : 0.25 + (data.totalSessions / maxSessions) * 0.75,
                      }}
                    />
                  </div>
                </TooltipPlus>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
export default React.memo(SingleAssistant);
