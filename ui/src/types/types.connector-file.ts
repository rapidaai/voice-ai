import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { PaginatedType } from './types.paginated';
/**
 * knowledge context
 */

export type ConnectorFileTypeProperty = {
  // as local pagination is required
  allFiles: Struct[];

  //
  filterFiles: Struct[];
  /**
   *
   */
  files: Struct[];
};

//
//
export type ConnectorFileTypeAction = {
  onChangeFiles: (files: Struct[]) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
};

//
//
//
export type ConnectorFileType = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getAllConnectorFiles: (
    toolId: string,
    token: string,
    userId: string,
    projectId: string,
    onError: (err: string) => void,
    onSuccess: (e: Struct[]) => void,
  ) => void;
} & PaginatedType &
  ConnectorFileTypeProperty &
  ConnectorFileTypeAction;
