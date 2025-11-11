import { FC, HTMLAttributes } from 'react';
import { Channel, useInputModeToggleAgent, VoiceAgent } from '@rapidaai/react';
import { cn } from '@/styles/media';
import { AudioMessagingAction } from './audio-messsaging-action';
import { SimpleMessagingAction } from './simple-messaging-action';

/**
 *
 */
interface MessageActionProps extends HTMLAttributes<HTMLDivElement> {
  voiceAgent: VoiceAgent;
  suggestions?: string[];
  placeholder?: string;
}

/**
 *
 * @param param0
 * @returns
 */
export const MessagingAction: FC<MessageActionProps> = ({
  suggestions = [],
  className,
  ...attr
}) => {
  const { channel } = useInputModeToggleAgent(attr.voiceAgent);

  return (
    <div className={cn('', className)}>
      {channel === Channel.Audio ? (
        <AudioMessagingAction {...attr} />
      ) : (
        <SimpleMessagingAction {...attr} />
      )}
    </div>
  );
};
