import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { AssistantConversationMessage } from '@rapidaai/react';

export type ConversationLogProperty = {
  /**
   * all the conversation messages
   */
  assistantMessages: AssistantConversationMessage[];

  /**
   *
   */
  fields: ('metadata' | 'metric' | 'stage' | 'request' | 'response')[];
};
/**
 * assistant context
 */
type ConversationLogAction = {
  /**
   * clear everything
   * @returns
   */
  clear: () => void;

  /**
   *
   * @returns
   */
  setFields: (
    fl: ('metadata' | 'metric' | 'stage' | 'request' | 'response')[],
  ) => void;
};

export type ConversationLog = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getMessages: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConversationMessage[]) => void,
  ) => void;
} & PaginatedType &
  ColumnarType &
  ConversationLogProperty &
  ConversationLogAction;
