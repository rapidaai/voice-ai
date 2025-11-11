import { AssistantWebhookLog } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

/**
 * assistant context
 */

export type WebhookLogTypeProperty = {
  /**
   * list of Webhook log
   */
  webhookLogs: AssistantWebhookLog[];
};

/**
 *
 */
export type WebhookLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeActivities: (ep: AssistantWebhookLog[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getActivities: (
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
} & WebhookLogTypeProperty &
  PaginatedType &
  ColumnarType;
