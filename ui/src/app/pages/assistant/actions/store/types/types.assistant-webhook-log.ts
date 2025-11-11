import { AssistantWebhookLog } from '@rapidaai/react';
import { ColumnarType } from '@/types/types.columnar';
import { PaginatedType } from '@/types/types.paginated';

/**
 * assistant context
 */

export type AssistantWebhookLogProperty = {
  /**
   * list of activity log
   */
  webhookLogs: AssistantWebhookLog[];
};

/**
 *
 */
export type AssistantWebhookLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeAssistantWebhookLogs: (ep: AssistantWebhookLog[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getAssistantWebhookLogs: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantWebhookLog[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & AssistantWebhookLogProperty &
  PaginatedType &
  ColumnarType;
