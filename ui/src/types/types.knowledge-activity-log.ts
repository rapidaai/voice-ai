import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { KnowledgeLog } from '@rapidaai/react';

/**
 * assistant context
 */

export type KnowledgeActivityLogTypeProperty = {
  /**
   * list of activity log
   */
  activities: KnowledgeLog[];
};

/**
 *
 */
export type KnowledgeActivityLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeActivities: (ep: KnowledgeLog[]) => void;
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
    onSuccess: (e: KnowledgeLog[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & KnowledgeActivityLogTypeProperty &
  PaginatedType &
  ColumnarType;
