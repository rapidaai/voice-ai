import { AssistantTool } from '@rapidaai/react';
import { ColumnarType, PaginatedType } from '@/types';

/**
 * assistant context
 */

export type AssistantToolProperty = {
  /**
   * list of activity log
   */
  tools: AssistantTool[];
};

/**
 *
 */
export type AssistantToolType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeAssistantTools: (ep: AssistantTool[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getAssistantTool: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantTool[]) => void,
  ) => void;

  /**
   *
   * @param assistantId
   * @param toolId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  deleteAssistantTool: (
    assistantId: string,
    toolId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantTool) => void,
  ) => void;
  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & AssistantToolProperty &
  PaginatedType &
  ColumnarType;
