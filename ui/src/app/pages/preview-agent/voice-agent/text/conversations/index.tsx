import { Tooltip } from '@/app/components/base/tooltip';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { TextImage } from '@/app/components/Image/TextImage';
import { cn, toHumanReadableRelativeTimeFromDate } from '@/styles/media';
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

import { motion } from 'framer-motion';
import { useSearchParams } from 'react-router-dom';
import { useCurrentCredential } from '@/hooks/use-credential';
import { MessageFeedbackDialog } from '@/app/components/base/modal/message-feedback-modal';
import MarkdownPreview from '@uiw/react-markdown-preview';

export const ConversationMessages: FC<{ vag: VoiceAgent }> = ({ vag }) => {
  const messages = useAgentMessages(vag);
  const [searchParams] = useSearchParams();
  const { user } = useCurrentCredential();
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

  return (
    <div className="parent group [&_.feedback-btn]:hidden [&_.message-cntnt:last-of-type_.feedback-btn]:flex">
      {messages.messages.map((msg, idx) => {
        return msg.role === MessageRole.User ? (
          <div
            key={`user-${idx}`}
            className="message-cntnt p-2.5 pb-4 group relative grid grid-cols-[auto_minmax(0,1fr)] auto-rows-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 hover:bg-gray-100 rounded-xl dark:hover:bg-gray-950"
          >
            <div className="row-span-2 min-w-(--space-9)">
              <TextImage name={user?.name || queryName || 'user'} size={10} />
            </div>
            <div className="flex items-baseline gap-2">
              <div className="text-md font-semibold capitalize">
                {user?.name || queryName || 'user'}
              </div>
              <div className="text-sm text-muted">
                <div className="flex flex-row gap-2 items-center justify-center text-palette-gray-800">
                  <span className="opacity-70">
                    {toHumanReadableRelativeTimeFromDate(msg.time)}
                  </span>
                </div>
              </div>
            </div>
            <div className="text-md">
              {msg.messages.map((x, idx) => {
                return (
                  <motion.div
                    variants={{
                      hidden: {
                        opacity: 0,
                        y: 20,
                      },
                      visible: {
                        opacity: 1,
                        y: 0,
                        transition: {
                          duration: 0.1,
                        },
                      },
                    }}
                    key={idx}
                    className={cn(
                      'w-fit text-md',
                      '[&_:is([data-link],a:link,a:visited,a:hover,a:active)]:text-primary',
                      '[&_:is([data-link],a:link,a:visited,a:hover,a:active):hover]:underline',
                      '[&_:is(code,div[data-lang])]:font-mono',
                      '[&_:is(code,div[data-lang])]:bg-overlay',
                      '[&_:is(code,div[data-lang])]:rounded-[2px]',
                      '[&_:is(code)]:p-0.5',
                      '[&_div[data-lang]]:p-2',
                      '[&_div[data-lang]]:overflow-auto',
                      '[&_:is(p,ul,ol,dl,table,blockquote,div[data-lang],h4,h5,h6,hr):not(:first-child)]:mt-2',
                      '[&_:is(p,ul,ol,dl,table,blockquote,div[data-lang],h3,h4,h5,h6,hr):not(:last-child)]:mb-2',
                      '[&_:is(ul,ol)]:pl-5',
                      '[&_ul]:list-disc',
                      '[&_ol]:list-decimal',
                      '[&_ol>li>ol]:list-[lower-alpha]',
                      '[&_ol>li>ol>li>ol]:list-[lower-roman]',
                      '[&_ol>li>ol>li>ol>li>ol]:list-[list-decimal]',
                      '[&_:is(strong,h1,h2,h3,h4,h5,h6)]:font-semibold',
                      '[&_:is(h1)]:text-2xl',
                      '[&_:is(h2)]:text-lg',
                      '[&_:is(li)]:py-2',
                      '[&_:is(h3)]:text-md',
                      '[&_h1:not(:first-child)]:mt-8',
                      '[&_h1:not(:last-child)]:mb-6',
                      '[&_h2:not(:first-child)]:mt-6',
                      '[&_h2:not(:last-child)]:mb-4',
                      '[&_h3:not(:first-child)]:mt-4',
                      '[&_li::marker]:inline-block',
                      '[&_li::marker]:align-top',
                      'break-words',
                      'leading-7',
                    )}
                  >
                    {x}
                    {msg.messages.length - 1 === idx &&
                      msg.status === MessageStatus.Pending && (
                        <motion.span
                          transition={{
                            staggerChildren: 0.25,
                          }}
                          initial="initial"
                          animate="animate"
                          className="pl-2 relative"
                        >
                          <motion.span
                            variants={{
                              initial: {
                                scaleY: 0.2,
                                opacity: 0.2,
                              },
                              animate: {
                                scaleY: 1,
                                opacity: 1,
                                transition: {
                                  repeat: Infinity,
                                  repeatType: 'mirror',
                                  duration: 0.5,
                                  ease: 'circIn',
                                },
                              },
                            }}
                            className="absolute bottom-0 h-5 w-2 bg-primary inline-block"
                          ></motion.span>
                        </motion.span>
                      )}
                  </motion.div>
                );
              })}
            </div>
            {messages.messages.length - 1 === idx && <div ref={ctrRef} />}
          </div>
        ) : (
          <div
            key={`assistant-${idx}`}
            className="message-cntnt p-2.5 pb-4 group relative grid grid-cols-[auto_minmax(0,1fr)] auto-rows-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 hover:bg-gray-100 rounded-xl dark:hover:bg-gray-950"
          >
            <div className="row-span-2 min-w-(--space-9)">
              <div className="bg-blue-600 w-10 h-10 flex items-center justify-center rounded-[2px]">
                <RapidaIcon className="text-white h-full w-full p-1.5" />
              </div>
            </div>
            <div className="flex items-baseline gap-2">
              <div className="text-md font-semibold">Rapida</div>
              <div className="text-sm text-muted">
                <div className="flex flex-row gap-2 items-center justify-center text-palette-gray-800">
                  <span className="opacity-70">
                    {toHumanReadableRelativeTimeFromDate(msg.time)}
                  </span>
                </div>
              </div>
            </div>
            <div className="text-md">
              {msg.messages.map((x, idx) => {
                return (
                  <motion.div
                    variants={{
                      hidden: { opacity: 0 },
                      visible: {
                        opacity: 1,
                        transition: {
                          staggerChildren: 0.05,
                          delayChildren: 0.3,
                        },
                      },
                    }}
                    initial="hidden"
                    animate="visible"
                    key={idx}
                    className="flex-1 min-w-0"
                  >
                    <MarkdownPreview
                      source={x}
                      className="text-gray-700! dark:text-gray-400! prose prose-base break-words max-w-none! prose-img:rounded-xl prose-headings:underline prose-a:text-blue-600 prose-strong:font-bold prose-headings:font-bold dark:prose-strong:text-white dark:prose-headings:text-white"
                      style={{ background: 'transparent' }}
                    />

                    {msg.messages.length - 1 === idx &&
                      msg.status === MessageStatus.Pending && (
                        <motion.span
                          transition={{
                            staggerChildren: 0.25,
                          }}
                          initial="initial"
                          animate="animate"
                          className="pl-2 relative"
                        >
                          <motion.span
                            variants={{
                              initial: {
                                scaleY: 0.2,
                                opacity: 0.2,
                              },
                              animate: {
                                scaleY: 1,
                                opacity: 1,
                                transition: {
                                  repeat: Infinity,
                                  repeatType: 'mirror',
                                  duration: 0.5,
                                  ease: 'circIn',
                                },
                              },
                            }}
                            className="absolute bottom-0 h-5 w-2 bg-primary inline-block"
                          ></motion.span>
                        </motion.span>
                      )}
                  </motion.div>
                );
              })}
              <MessageAction message={msg} vag={vag} />
            </div>
            {messages.messages.length - 1 === idx && <div ref={ctrRef} />}
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
