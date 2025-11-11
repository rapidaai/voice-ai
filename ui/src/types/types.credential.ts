import { VaultCredential } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

export type CredentialTypeProperties = {
  /**
   *
   */
  credentials: VaultCredential[];
};

export type CredentialTypeAction = {
  /**
   *
   * @param ep
   * @returns
   */
  onChangeCredentials: (ep: VaultCredential[]) => void;
};
/**
 * endpoint context
 */
export type CredentialType = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  onGetAllCredentials: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: VaultCredential[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & PaginatedType &
  ColumnarType &
  CredentialTypeProperties &
  CredentialTypeAction;
