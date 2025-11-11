import { useCredential } from '@/hooks';
import {
  AssistantConversation,
  AssistantConversationMessage,
} from '@rapidaai/react';
import { toContentText } from '@/utils/rapida_content';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { FC, useCallback, useContext, useEffect, useRef } from 'react';
import { AssistantChatContext } from '@/hooks/use-assistant-chat';
import { useBoolean } from 'ahooks';
import { SectionLoader } from '@/app/components/Loader/section-loader';
import { ArrowDownToLine, RotateCw } from 'lucide-react';
import { IButton } from '@/app/components/Form/Button';

import MarkdownPreview from '@uiw/react-markdown-preview';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { AudioPlayer } from '@/app/components/audio-player';
import { getStatusMetric } from '@/utils/metadata';
import { StatusIndicator } from '@/app/components/indicators/status';

export const ConversationMessages: FC<{
  conversation: AssistantConversation;
  assistantId: string;
  conversationId: string;
}> = ({ conversation, conversationId, assistantId }) => {
  //
  const [userId, token, projectId] = useCredential();
  const [loading, { setTrue: showLoader, setFalse: hideLoader }] =
    useBoolean(false);

  //
  const {
    conversations,
    onGetConversationMessages,
    onChangeConversationMessages,
  } = useContext(AssistantChatContext);

  //
  const ctrRef = useRef<HTMLDivElement>(null);
  //
  const get = () => {
    showLoader();
    onGetConversationMessages(
      assistantId,
      conversationId,
      projectId,
      token,
      userId,
      err => {
        hideLoader();
      },
      callbackOnGetConversationMessages,
    );
  };

  useEffect(() => {
    get();
  }, [assistantId, conversationId]);

  const callbackOnGetConversationMessages = useCallback(
    (msgs: Array<AssistantConversationMessage>) => {
      onChangeConversationMessages(msgs);
      scrollTo(ctrRef);
      hideLoader();
    },
    [],
  );

  const scrollTo = ref => {
    setTimeout(
      () =>
        ref.current?.scrollIntoView({ inline: 'center', behavior: 'smooth' }),
      777,
    );
  };

  if (loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }
  function csvEscape(str: string): string {
    return `"${str.replace(/"/g, '""')}"`;
  }
  const downloadAllMessages = () => {
    const csvContent = [
      'role,message',
      ...conversations.flatMap((row: AssistantConversationMessage) => [
        `user,${csvEscape(toContentText(row.getRequest()?.getContentsList()))}`,
        `system,${csvEscape(toContentText(row.getResponse()?.getContentsList()))}`,
      ]),
    ].join('\n');
    const url = URL.createObjectURL(
      new Blob([csvContent], { type: 'text/csv;charset=utf-8;' }),
    );
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', conversationId + '-message.csv');
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex-1 flex flex-col h-full relative">
      <PageHeaderBlock className="border-b sticky top-0 z-[2]">
        <PageTitleBlock>All messages</PageTitleBlock>
        <PaginationButtonBlock>
          <div className="border-l flex items-center justify-center px-4">
            <StatusIndicator
              state={getStatusMetric(conversation.getMetricsList())}
            />
          </div>
          <IButton
            onClick={() => {
              get();
            }}
          >
            <RotateCw strokeWidth={1.5} className="h-4 w-4" />
          </IButton>
          <IButton
            onClick={() => {
              downloadAllMessages();
            }}
          >
            <ArrowDownToLine className="h-4 w-4 mr-1" /> <span>Text</span>
          </IButton>
        </PaginationButtonBlock>
      </PageHeaderBlock>

      {conversation.getRecordingsList().map((x, idx) => {
        return <AudioPlayer key={idx} src={x.getRecordingurl()} />;
      })}
      {conversations.length === 0 && (
        <div className="my-auto mx-auto">
          <ActionableEmptyMessage
            title="No messages yet"
            subtitle="There are no message yet for the conversation"
          />
        </div>
      )}
      {conversations.map((x, idx) => {
        return (
          <div
            className="flex flex-col w-full  bg-white dark:bg-gray-900"
            key={idx}
          >
            {x.getRequest() && (
              <div className="flex items-center space-x-4 border-b-[0.5px] px-6 py-4 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-900 ">
                <div className="h-9 w-9 rounded-[2px] flex-shrink-0 bg-zinc-200/80 dark:bg-zinc-800/80 border-[0.5px] flex items-center justify-center dark:border-gray-700">
                  <span className="font-bold text-sm opacity-80">U</span>
                </div>
                <div className="text-md [&_:is([data-link],a:link,a:visited,a:hover,a:active)]:text-primary [&_:is([data-link],a:link,a:visited,a:hover,a:active):hover]:underline [&_:is(code,div[data-lang])]:font-mono [&_:is(code,div[data-lang])]:bg-overlay [&_:is(code,div[data-lang])]:rounded-[2px] [&_is:(code)]:p-0.5 [&_div[data-lang]]:p-2 [&_div[data-lang]]:overflow-auto [&_:is(p,ul,ol,dl,table,blockquote,div[data-lang],h4,h5,h6,hr):not(:first-child)]:mt-2 [&_:is(p,ul,ol,dl,table,blockquote,div[data-lang],h3,h4,h5,h6,hr):not(:last-child)]:mb-2 [&_:is(ul,ol)]:pl-5 [&_ul]:list-disc [&_ol]:list-decimal [&_ol>li>ol]:list-[lower-alpha] [&_ol>li>ol>li>ol]:list-[lower-roman] [&_ol>li>ol>li>ol>li>ol]:list-[list-decimal] [&_[data-user]]:text-primary [&_:is(strong,h1,h2,h3,h4,h5,h6)]:font-semibold [&_:is(h1)]:text-2xl [&_:is(h2)]:text-lg [&_:is(h3)]:text-md [&_h1:not(:first-child)]:mt-8 [&_h1:not(:last-child)]:mb-6 [&_h2:not(:first-child)]:mt-6 [&_h2:not(:last-child)]:mb-4 [&_h3:not(:first-child)]:mt-4 whitespace-pre-wrap break-words">
                  {toContentText(x.getRequest()?.getContentsList())}
                </div>
              </div>
            )}
            {x.getResponse() && x.getStatus() !== 'FAILED' && (
              <div className="flex items-center space-x-4 border-b-[0.5px] px-6 py-4 dark:border-gray-800 relative overflow-hidden hover:bg-gray-50 dark:hover:bg-gray-900">
                <RapidaIcon className="h-8 w-8 text-blue-600 shrink-0" />
                <div className="flex-1 min-w-0">
                  <MarkdownPreview
                    source={toContentText(x.getResponse()?.getContentsList())}
                    className="!text-gray-700 dark:!text-gray-400 prose prose-base break-words !max-w-none prose-img:rounded-xl prose-headings:underline prose-a:text-blue-600 prose-strong:font-bold prose-headings:font-bold dark:prose-strong:text-white dark:prose-headings:text-white"
                    style={{ background: 'transparent' }}
                  />
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
