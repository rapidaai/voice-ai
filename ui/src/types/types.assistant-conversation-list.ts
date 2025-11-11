import { AssistantConversation } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

export type AssistantConversationProperty = {
  /**
   * all the conversation messages
   */
  assistantConversations: AssistantConversation[];

  /**
   * current conversation from the list
   */
  currentConversation: AssistantConversation | null;

  /**
   *
   */
  dialogVisible: boolean;
};
/**
 * assistant context
 */
type AssistantConversationAction = {
  /**
   *
   * @param msgs
   * @returns
   */
  onChangeAssistantConversations: (msgs: AssistantConversation[]) => void;

  /**
   *
   * @param conv
   * @returns
   */
  onChangeCurrentConversation: (conv: AssistantConversation | null) => void;

  /**
   *
   * @param conv
   * @returns
   */
  showDialog: (conv: AssistantConversation) => void;

  /**
   *
   * @returns
   */
  hideDialog: () => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
};

export type AssistantConversationList = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getAssistantConversations: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConversation[]) => void,
  ) => void;
} & PaginatedType &
  ColumnarType &
  AssistantConversationAction &
  AssistantConversationProperty;
