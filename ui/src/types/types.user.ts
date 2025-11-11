import { User } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

/**
 * endpoint context
 */
export type UserType = {
  /**
   *list of user
   */
  users: User[];

  /**
   *
   * @param user
   * @returns
   */
  setUsers: (user: User[]) => void;
  /**
   *
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  getAllUser: (
    token: string,
    userId: string,
    projectId: string,
    onError: (err: string) => void,
    onSuccess: (e: User[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & PaginatedType &
  ColumnarType;
