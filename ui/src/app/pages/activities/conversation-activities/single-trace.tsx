import React, { useState } from 'react';
import SourceIndicator from '@/app/components/indicators/source';
import { StatusIndicator } from '@/app/components/indicators/status';
import { toContentText } from '@/utils/rapida_content';
import { cn, toHumanReadableDateTime } from '@/styles/media';
import {
  getMetricValueOrDefault,
  getTimeTakenMetric,
  getTotalTokenMetric,
} from '@/utils/metadata';
import { AssistantConversationMessage } from '@rapidaai/react';
import { TraceDetail } from './trace-detail';
import { ChevronRight, ExternalLink, Telescope } from 'lucide-react';
import { TableRow } from '@/app/components/base/tables/table-row';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { CustomLink } from '@/app/components/custom-link';
import { useConversationLogPageStore } from '@/hooks/use-conversation-log-page-store';
import { formatNanoToReadableMilli } from '@/utils';

interface SingleTraceProps {
  row: AssistantConversationMessage;
  idx: number;
  onClick: () => void;
}

export const SingleTrace: React.FC<SingleTraceProps> = ({
  row,
  idx,
  onClick,
}) => {
  const conversationLogAction = useConversationLogPageStore();
  const [info, setInfo] = useState(false);
  return (
    <>
      <TableRow key={idx} data-id={row.getId()} onClick={onClick}>
        <TableCell className="py-0 px-0">
          <div className="flex space-x-2">
            <span
              onClick={event => {
                event.stopPropagation();
                setInfo(!info);
              }}
              className="h-6 w-6 flex items-center justify-center rounded-[2px] hover:bg-gray-300 dark:hover:bg-gray-800 cursor-pointer"
            >
              <ChevronRight
                strokeWidth={1.5}
                className={cn(
                  'w-5 h-5 transition-all duration-200',
                  info && 'rotate-90',
                )}
              />
            </span>
            <span className="h-6 w-6 flex items-center justify-center rounded-[2px] hover:bg-gray-300 dark:hover:bg-gray-800 cursor-pointer">
              <Telescope
                strokeWidth={1.5}
                className={cn('w-5 h-5 transition-all duration-200')}
              />
            </span>
          </div>
        </TableCell>
        {conversationLogAction.visibleColumn('id') && (
          <TableCell>{row.getMessageid().split('-')[0]}</TableCell>
        )}
        {conversationLogAction.visibleColumn('version') && (
          <TableCell>vrsn_{row.getAssistantprovidermodelid()}</TableCell>
        )}
        {conversationLogAction.visibleColumn('assistant_conversation_id') && (
          <TableCell>
            <CustomLink
              to={`/deployment/assistant/${row.getAssistantid()}/sessions/${row.getAssistantconversationid()}`}
              className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
            >
              <span>{row.getAssistantconversationid()}</span>
              <ExternalLink className="w-3 h-3" />
            </CustomLink>
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('assistant_id') && (
          <TableCell>
            <CustomLink
              to={`/deployment/assistant/${row.getAssistantid()}`}
              className="font-normal dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left flex items-center space-x-1"
            >
              <span>{row.getAssistantid()}</span>
              <ExternalLink className="w-3 h-3" />
            </CustomLink>
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('source') && (
          <TableCell>
            <SourceIndicator source={row.getSource()} />
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('request') && row.getRequest() && (
          <TableCell>
            <p className="line-clamp-2">
              {toContentText(row.getRequest()?.getContentsList())}
            </p>
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('response') &&
        row.getResponse() ? (
          <TableCell>
            <p className="line-clamp-2">
              {toContentText(row.getResponse()?.getContentsList())}
            </p>
          </TableCell>
        ) : (
          <TableCell>
            <p className="line-clamp-2 opacity-65">Not available</p>
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('created_date') && (
          <TableCell>
            {row.getCreateddate() &&
              toHumanReadableDateTime(row.getCreateddate()!)}
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('status') && (
          <TableCell>
            <StatusIndicator state={row.getStatus()} />
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('time_taken') && (
          <TableCell>
            {formatNanoToReadableMilli(
              getTimeTakenMetric(row.getMetricsList()),
            )}
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('total_token') && (
          <TableCell>{getTotalTokenMetric(row.getMetricsList())}</TableCell>
        )}
        {conversationLogAction.visibleColumn('user_feedback') && (
          <TableCell>
            {getMetricValueOrDefault(
              row.getMetricsList(),
              'custom.feedback',
              '__',
            )}
          </TableCell>
        )}
        {conversationLogAction.visibleColumn('user_text_feedback') && (
          <TableCell>
            {getMetricValueOrDefault(
              row.getMetricsList(),
              'custom.feedback_text',
              '--',
            )}
          </TableCell>
        )}
      </TableRow>

      <TableRow
        className={cn(
          'transition-all duration-200',
          info ? ' visible' : 'collapse pointer-events-none',
        )}
      >
        <TableCell
          className="px-0! py-0!"
          colSpan={
            conversationLogAction.columns.filter(x => x.visible).length + 1
          }
        >
          <TraceDetail currentAssistantMessage={row} />
        </TableCell>
      </TableRow>
    </>
  );
};
