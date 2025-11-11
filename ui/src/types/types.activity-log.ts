import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { AuditLog } from '@rapidaai/react';

/**
 * assistant context
 */

export type ActivityLogTypeProperty = {
  /**
   * list of activity log
   */
  activities: AuditLog[];
};

/**
 *
 */
export type ActivityLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeActivities: (ep: AuditLog[]) => void;
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
    onSuccess: (e: AuditLog[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & ActivityLogTypeProperty &
  PaginatedType &
  ColumnarType;
