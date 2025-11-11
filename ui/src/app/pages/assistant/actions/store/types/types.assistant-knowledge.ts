import { AssistantKnowledge } from '@rapidaai/react';
import { ColumnarType, PaginatedType } from '@/types';

/**
 * assistant context
 */

export type AssistantKnowledgeProperty = {
  /**
   * list of activity log
   */
  knowledges: AssistantKnowledge[];
};

/**
 *
 */
export type AssistantKnowledgeType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeAssistantKnowledges: (ep: AssistantKnowledge[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getAssistantKnowledge: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantKnowledge[]) => void,
  ) => void;

  /**
   *
   * @param assistantId
   * @param knowledgeId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  deleteAssistantKnowledge: (
    assistantId: string,
    knowledgeId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantKnowledge) => void,
  ) => void;
  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & AssistantKnowledgeProperty &
  PaginatedType &
  ColumnarType;
