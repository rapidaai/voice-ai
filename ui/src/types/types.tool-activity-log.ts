import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { AssistantToolLog } from '@rapidaai/react';

/**
 * assistant context
 */

export type ToolActivityLogTypeProperty = {
  /**
   * list of activity log
   */
  activities: AssistantToolLog[];
};

/**
 *
 */
export type ToolActivityLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeActivities: (ep: AssistantToolLog[]) => void;
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
    onSuccess: (e: AssistantToolLog[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & ToolActivityLogTypeProperty &
  PaginatedType &
  ColumnarType;
