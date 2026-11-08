import {
  Assistant,
  useConnectAgent,
  useInputModeToggleAgent,
  VoiceAgent,
} from '@rapidaai/react';
import { Send, VoiceMode, StopFilledAlt } from '@carbon/icons-react';
import { FC, HTMLAttributes } from 'react';
import { useForm } from 'react-hook-form';
import { ScalableTextarea } from '@/app/components/ui/primitives';
import { PrimaryButton } from '@/app/components/ui/primitives';

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
  const { handleVoiceToggle } = useInputModeToggleAgent(voiceAgent);
  const {
    handleConnectAgent,
    handleDisconnectAgent,
    isConnected,
    isConnecting,
  } = useConnectAgent(voiceAgent);

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
    <div className="bg-surface ">
      <form className="flex flex-col" onSubmit={handleSubmit(onSubmitForm)}>
        {/* Textarea — grows with content, no overlap with buttons */}
        <ScalableTextarea
          placeholder={placeholder}
          wrapperClassName="bg-surface border-t border-l border-r border-border-subtle p-2"
          className="bg-transparent"
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

        {/* Action row — always below textarea, right-aligned */}
        <div className="flex items-center justify-end px-3 pb-3 pt-1 border-t border-border-subtle ">
          {isValid ? (
            <PrimaryButton size="md" type="submit" renderIcon={Send}>
              Send
            </PrimaryButton>
          ) : (
            <PrimaryButton
              size="md"
              isLoading={isConnecting}
              renderIcon={VoiceMode}
              onClick={async () => {
                await handleVoiceToggle();
                !isConnected && (await handleConnectAgent());
              }}
            >
              {isConnecting ? 'Connecting...' : 'Voice'}
            </PrimaryButton>
          )}
          {(isConnected || isConnecting) && (
            <button
              type="button"
              disabled={!isConnected && !isConnecting}
              onClick={async () => {
                await handleDisconnectAgent();
              }}
              className="group h-10 px-3 flex items-center justify-center bg-support-error hover:bg-support-error disabled:opacity-50 text-inverse transition-all duration-200 cursor-pointer"
            >
              <StopFilledAlt size={16} className="shrink-0" />
              <span className="max-w-0 group-hover:max-w-15 overflow-hidden transition-all duration-200 whitespace-nowrap group-hover:ml-2 text-sm font-medium">
                Stop
              </span>
            </button>
          )}
        </div>
      </form>
    </div>
  );
};
