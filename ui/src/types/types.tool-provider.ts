import { PaginatedType } from './types.paginated';
import { ToolProvider } from '@rapidaai/react';

/**
 * knowledge context
 */

export type ToolProviderTypeProperty = {
  /**
   *
   */
  toolProviders: ToolProvider[];
};

//
//
export type ToolProviderTypeAction = {
  onChangeToolProviders: (tools: ToolProvider[]) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
};

//
//
//
export type ToolProviderType = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getAllToolProvider: (
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: ToolProvider[]) => void,
  ) => void;
} & PaginatedType &
  ToolProviderTypeProperty &
  ToolProviderTypeAction;
