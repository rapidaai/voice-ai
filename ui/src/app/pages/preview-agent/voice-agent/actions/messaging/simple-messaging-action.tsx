import {
  Assistant,
  useConnectAgent,
  useInputModeToggleAgent,
  VoiceAgent,
} from '@rapidaai/react';
import { AudioLines, Send } from 'lucide-react';
import { FC, HTMLAttributes, useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { cn } from '@/utils';
import { AnimatePresence, motion } from 'framer-motion';
import { Spinner } from '@/app/components/loader/spinner';
import { ScalableTextarea } from '@/app/components/form/textarea';

interface SimpleMessagingAcitonProps extends HTMLAttributes<HTMLDivElement> {
  placeholder?: string;
  voiceAgent: VoiceAgent;
  assistant: Assistant | null;
}
export const SimpleMessagingAction: FC<SimpleMessagingAcitonProps> = ({
  className,
  voiceAgent,
  assistant,
  placeholder,
}) => {
  //   const ctx = useEnsureVoiceAgent();
  const { handleVoiceToggle } = useInputModeToggleAgent(voiceAgent);
  const { handleConnectAgent, handleDisconnectAgent, isConnected } =
    useConnectAgent(voiceAgent);
  const [isLoading, setIsLoading] = useState(false);
  useEffect(() => {
    if (!isConnected) {
      setIsLoading(false);
    }
  }, [isConnected]);

  const handleDisconnectClick = async () => {
    if (isConnected) {
      setIsLoading(true);
      await handleDisconnectAgent();
    } else {
      //
      await handleConnectAgent();
    }
  };

  const {
    register,
    handleSubmit,
    reset,
    formState: { isValid },
  } = useForm({
    mode: 'onChange',
  });

  const onSubmitForm = data => {
    voiceAgent?.onSendText(data.message);
    reset();
  };

  return (
    <div>
      <AnimatePresence>
        <motion.div
          className={cn(
            'flex justify-center items-center py-2',
            !isConnected && 'hidden',
          )}
        >
          <button
            onClick={async () => {
              handleDisconnectClick();
            }}
            disabled={isLoading}
            className={cn(
              'px-3 py-[4px] rounded-[2px] flex items-center space-x-1.5 bg-red-600 text-white border border-red-700/50',
            )}
          >
            {isLoading ? (
              <Spinner className="border-white!" />
            ) : (
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="currentColor"
                className="w-4 h-4"
                strokeWidth={1.5}
              >
                <path
                  fillRule="evenodd"
                  d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12Zm6-2.438c0-.724.588-1.312 1.313-1.312h4.874c.725 0 1.313.588 1.313 1.313v4.874c0 .725-.588 1.313-1.313 1.313H9.564a1.312 1.312 0 0 1-1.313-1.313V9.564Z"
                  clipRule="evenodd"
                />
              </svg>
            )}
            <span className="text-sm font-medium">End session</span>
          </button>
        </motion.div>
      </AnimatePresence>
      <form
        className={cn(
          'relative flex items-center gap-4 focus-within:border-primary  dark:border-gray-700 bg-light-background focus-within:bg-white',
        )}
        onSubmit={handleSubmit(onSubmitForm)}
      >
        <ScalableTextarea
          placeholder={placeholder}
          wrapperClassName="bg-light-background p-0"
          className="bg-light-background focus-within:bg-white"
          {...register('message', {
            required: 'Please write your message.',
          })}
          required
          onKeyDown={(e: React.KeyboardEvent<HTMLTextAreaElement>) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              handleSubmit(onSubmitForm)(e);
            }
          }}
        />

        <div className="absolute rounded-b-lg right-2 bottom-2 my-auto w-fit">
          {isValid ? (
            <button
              type="submit"
              className="inline-flex shrink-0 justify-center items-center h-8 w-8 text-white bg-primary hover:bg-primary focus:z-10 focus:outline-hidden focus:bg-primary"
            >
              <Send className="shrink-0 w-4 h-4" strokeWidth="1.5" />
            </button>
          ) : (
            <button
              onClick={async () => {
                await handleVoiceToggle();
                !isConnected && (await handleConnectAgent());
              }}
              className="voice-action relative flex h-8 px-3 items-center justify-center bg-primary text-white transition-colors focus-visible:outline-hidden focus-visible:outline-black disabled:text-gray-50 disabled:opacity-30 can-hover:hover:opacity-70 min-w-8 p-2"
            >
              <div className="flex items-center justify-center mr-1">
                <AudioLines className="shrink-0 w-4 h-4" strokeWidth="1.5" />
              </div>
              <span className="whitespace-nowrap pl-1 pr-1 text-[13px] font-medium">
                Voice
              </span>
            </button>
          )}
        </div>
      </form>
    </div>
  );
};
