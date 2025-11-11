import { EndpointLog } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

/**
 * assistant context
 */

export type EndpointLogTypeProperty = {
  /**
   * list of Endpoint log
   */
  endpointLogs: EndpointLog[];
};

/**
 *
 */
export type EndpointLogType = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeLogs: (ep: EndpointLog[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getLogs: (
    endpointId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: EndpointLog[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & EndpointLogTypeProperty &
  PaginatedType &
  ColumnarType;
