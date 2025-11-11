import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { AssistantConversationMessage } from '@rapidaai/react';
import { Assistant } from '@rapidaai/react';

/**
 *
 */
export type AssistantChatProperty = {
  /**
   * current assistant
   */
  currentAssistant: Assistant | null;

  /**
   * assistant conversation
   */
  currentAssistantConversationId: string | null;

  /**
   *
   */
  conversations: AssistantConversationMessage[];
};

/**
 * assistant context
 */
type AssistantChatApiCallAction = {
  /**
   *
   * @param assistantId
   * @param conversationId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onGetConversationMessages: (
    assistantId: string,
    conversationId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConversationMessage[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
};

export type AssistantChatType = {
  /**
   *
   * @param assistant
   * @returns
   */
  onChangeCurrentAssistant: (assistant: Assistant) => void;

  /**
   *
   * @param message
   * @returns
   */
  onChangeConversationMessages: (
    message: Array<AssistantConversationMessage>,
  ) => void;

  /**
   *
   * @param msg
   * @returns
   */
  onChangeConversactionMessage: (msg: AssistantConversationMessage) => void;

  /**
   *
   * @param assistantConversationId
   * @returns
   */
  onChangeAssistantConversationId: (assistantConversationId: string) => void;
} & PaginatedType &
  ColumnarType &
  AssistantChatProperty &
  AssistantChatApiCallAction;
