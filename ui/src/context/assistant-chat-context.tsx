import {
  AssistantChatContext,
  useAssistantChat,
} from '@/hooks/use-assistant-chat';
import React from 'react';

/**
 *
 * @param param0
 * @returns
 */
export const AssistantChatContextProvider: React.FC<{ children }> = ({
  children,
}) => {
  const actions = useAssistantChat();

  /**
   *
   */
  return (
    <AssistantChatContext.Provider value={actions}>
      {children}
    </AssistantChatContext.Provider>
  );
};
