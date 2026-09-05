import { Tooltip } from '@/app/components/base/tooltip';
import { BrandedLogo } from '@/app/components/brand/branded-logo';
import { TextImage } from '@/app/components/text-image';
import { toHumanReadableRelativeTimeFromDate } from '@/utils/date';
import {
  Feedback,
  Message,
  MessageRole,
  MessageStatus,
  VoiceAgent,
  useMessageFeedback,
  useAgentMessages,
} from '@rapidaai/react';
import { FC, useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useCurrentCredential } from '@/hooks/use-credential';
import { MessageFeedbackDialog } from '@/app/components/base/modal/message-feedback-modal';
import MarkdownPreview from '@uiw/react-markdown-preview';
import { cn } from '@/utils';
import { useTheme } from '@/theme/theme-provider';

export const ConversationMessages: FC<{ vag: VoiceAgent }> = ({ vag }) => {
  const messages = useAgentMessages(vag);
  const [searchParams] = useSearchParams();
  const { user } = useCurrentCredential();
  const { theme } = useTheme();
  const queryName = searchParams.get('name');
  const ctrRef = useRef<HTMLDivElement>(null);
  /**
   *
   * @param ref
   */
  const scrollTo = ref => {
    setTimeout(
      () =>
        ref.current?.scrollIntoView({ inline: 'center', behavior: 'smooth' }),
      777,
    );
  };

  //   on change of message to scroll
  useEffect(() => {
    scrollTo(ctrRef);
  }, [JSON.stringify(messages)]);

  const groupedMessages = messages.messages.reduce<
    { role: MessageRole; items: typeof messages.messages }[]
  >((acc, msg) => {
    const last = acc[acc.length - 1];

    if (last && last.role === msg.role) {
      last.items.push(msg);
    } else {
      acc.push({ role: msg.role, items: [msg] });
    }

    return acc;
  }, []);

  return (
    <div className="parent group [&_.feedback-btn]:hidden [&_.message-cntnt:last-of-type_.feedback-btn]:flex font-mono!">
      {groupedMessages.map((group, groupIdx) => {
        const isUser = group.role === MessageRole.User;
        const isLastGroup = groupIdx === groupedMessages.length - 1;

        return (
          <div
            key={groupIdx}
            className="message-cntnt p-2.5 pb-4 group relative grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 hover:bg-gray-100 rounded-xl dark:hover:bg-gray-950"
          >
            <div className="row-span-2 min-w-(--space-9)">
              {isUser ? (
                <TextImage name={user?.name || queryName || 'user'} size={10} />
              ) : (
                <div className="bg-primary text-on-primary w-10 h-10 flex items-center justify-center rounded-xs">
                  <BrandedLogo
                    variant="compact"
                    className="h-full w-full p-1.5 object-contain text-on-primary"
                    textClassName="px-1 text-center text-[10px] leading-tight"
                  />
                </div>
              )}
            </div>

            {/* HEADER */}
            <div className="flex items-baseline gap-2">
              <div className="text-md font-semibold">
                {isUser ? user?.name || queryName || 'User' : theme.brand.name}
              </div>
              <span className="text-sm opacity-70">
                {toHumanReadableRelativeTimeFromDate(group.items[0].time)}
              </span>
            </div>

            {/* MESSAGES */}
            <div className="text-md space-y-2">
              {group.items.map((msg, idx) =>
                msg.messages.map((x, midx) => (
                  <div key={`${idx}-${midx}`} data-key={msg.id}>
                    <MarkdownPreview
                      source={x}
                      className="prose! dark:prose-invert! prose-sm! font-mono! max-w-6xl!"
                      style={{
                        background: 'transparent',
                        fontSize: '14px',
                      }}
                    />
                  </div>
                )),
              )}
              {!isUser && (
                <MessageAction message={group.items.at(-1)!} vag={vag} />
              )}
            </div>

            {isLastGroup && <div ref={ctrRef} />}
          </div>
        );
      })}
    </div>
  );
};

export const MessageAction: FC<{ vag: VoiceAgent; message: Message }> = ({
  message,
  vag,
}) => {
  const { handleHelpfulnessFeedback, handleMessageFeedback } =
    useMessageFeedback(vag);
  const [showFeedbackModal, setShowFeedbackModal] = useState(false);
  const onSubmitFeedback = (feedbackText: string) => {
    handleMessageFeedback(
      message.id,
      'feedback_text',
      'Feedback text given after marking message not helpful',
      feedbackText,
    );
    handleHelpfulnessFeedback(message.id, Feedback.NotHelpful);
  };

  return (
    <>
      <div
        role="group"
        className={cn(
          message.status !== MessageStatus.Complete && 'hidden!',
          'feedback-btn flex justify-start items-center mt-4',
        )}
      >
        <Tooltip content={'Copy'}>
          <button
            type="button"
            onClick={() => {
              navigator.clipboard.writeText(message.messages.join('\n'));
            }}
            className={cn(
              'text-gray-500 hover:text-gray-700 cursor-pointer group flex items-center justify-center flex-none text-sm p-1 rounded-[2px] hover:bg-gray-600/10',
            )}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth="1.5"
              stroke="currentColor"
              className="w-5 h-5"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 0 0-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 0 1-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H9.75"
              />
            </svg>
          </button>
        </Tooltip>
        <Tooltip content={'Helpful'}>
          <button
            type="button"
            onClick={() => {
              handleHelpfulnessFeedback(message.id, Feedback.Helpful);
            }}
            className={cn(
              message.feedback === Feedback.Helpful &&
                'bg-blue-600/10! text-blue-600!',
              'text-gray-500 hover:text-blue-600 cursor-pointer group flex items-center justify-center flex-none text-sm p-1 rounded-[2px] hover:bg-blue-600/10',
            )}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.2"
              viewBox="0 0 24 24"
              className="w-6 h-6"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M18 12.25h-1.25m1.25 0-.734 5.277a2 2 0 0 1-2.185 1.714l-6.933-.712a1 1 0 0 1-.898-.994V9.75c2.024 0 2.455-2.515 2.521-4.151.024-.6.672-1.026 1.208-.758a2.298 2.298 0 0 1 1.271 2.056V9.75H18a1.25 1.25 0 1 1 0 2.5Z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M4.75 9.75a1 1 0 0 1 1-1h.5a1 1 0 0 1 1 1v8.5a1 1 0 0 1-1 1h-.5a1 1 0 0 1-1-1v-8.5Z"
              />
            </svg>
          </button>
        </Tooltip>

        <Tooltip content={'Not helpful'}>
          <button
            type="button"
            onClick={() => {
              setShowFeedbackModal(true);
            }}
            className={cn(
              message.feedback === Feedback.NotHelpful &&
                'bg-red-600/10! text-red-600!',
              'text-gray-500 hover:text-red-600 cursor-pointer group flex items-center justify-center flex-none text-sm p-1 rounded-[2px] hover:bg-red-600/10',
            )}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.2"
              aria-hidden="true"
              viewBox="0 0 24 24"
              className="w-6 h-6"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M18 11.758h-1.25m1.25 0-.734-5.277a2 2 0 0 0-2.185-1.714l-6.933.711a1 1 0 0 0-.898.995v7.785c2.024 0 2.455 2.515 2.521 4.15.024.6.672 1.027 1.208.758a2.298 2.298 0 0 0 1.271-2.055v-2.853H18a1.25 1.25 0 1 0 0-2.5Z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M4.75 5.75a1 1 0 0 1 1-1h.5a1 1 0 0 1 1 1v8.5a1 1 0 0 1-1 1h-.5a1 1 0 0 1-1-1v-8.5Z"
              />
            </svg>
          </button>
        </Tooltip>
      </div>

      <MessageFeedbackDialog
        modalOpen={showFeedbackModal}
        setModalOpen={setShowFeedbackModal}
        onSubmitFeedback={onSubmitFeedback}
      />
    </>
  );
};
