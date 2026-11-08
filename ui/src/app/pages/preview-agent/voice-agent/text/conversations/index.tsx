import { BrandedLogo } from '@/app/components/layout/brand/branded-logo';
import { IconOnlyButton, UserAvatar } from '@/app/components/ui/primitives';
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
import { MessageFeedbackDialog } from '@/app/components/dialogs/shared/message-feedback-modal';
import MarkdownPreview from '@uiw/react-markdown-preview';
import { cn } from '@/utils';
import { useTheme } from '@/theme/theme-provider';
import { Copy, ThumbsDown, ThumbsUp } from '@carbon/icons-react';

export const ConversationMessages: FC<{ vag: VoiceAgent }> = ({ vag }) => {
  const messages = useAgentMessages(vag);
  const [searchParams] = useSearchParams();
  const { user } = useCurrentCredential();
  const { theme } = useTheme();
  const queryName = searchParams.get('name');
  const userName = (user?.name || queryName || 'user').trim() || 'user';
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
            className="message-cntnt p-2.5 pb-4 group relative grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 hover:bg-layer rounded-xl "
          >
            <div className="row-span-2 min-w-(--space-9)">
              {isUser ? (
                <UserAvatar
                  aria-label={userName}
                  name={userName}
                  role="img"
                  size="lg"
                />
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
                      className="prose! prose-sm! max-w-6xl! bg-transparent! font-mono! prose-headings:text-foreground prose-p:text-foreground prose-li:text-foreground prose-strong:text-foreground prose-code:text-foreground prose-a:text-link"
                      style={{
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
        <IconOnlyButton
          kind="ghost"
          size="sm"
          renderIcon={Copy}
          iconDescription="Copy message"
          onClick={() => navigator.clipboard.writeText(message.messages.join('\n'))}
        />
        <IconOnlyButton
          kind="ghost"
          size="sm"
          renderIcon={ThumbsUp}
          iconDescription="Helpful"
          aria-pressed={message.feedback === Feedback.Helpful}
          className={cn(
            message.feedback === Feedback.Helpful && 'text-support-info!',
          )}
          onClick={() =>
            handleHelpfulnessFeedback(message.id, Feedback.Helpful)
          }
        />
        <IconOnlyButton
          kind="ghost"
          size="sm"
          renderIcon={ThumbsDown}
          iconDescription="Not helpful"
          aria-pressed={message.feedback === Feedback.NotHelpful}
          className={cn(
            message.feedback === Feedback.NotHelpful && 'text-support-error!',
          )}
          onClick={() => setShowFeedbackModal(true)}
        />
      </div>

      <MessageFeedbackDialog
        modalOpen={showFeedbackModal}
        setModalOpen={setShowFeedbackModal}
        onSubmitFeedback={onSubmitFeedback}
      />
    </>
  );
};
