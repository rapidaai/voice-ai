import {
  ConnectorFileType,
  ConnectorFileTypeAction,
  ConnectorFileTypeProperty,
} from '@/types/types.connector-file';
import { initialPaginated } from '@/types/types.paginated';
import { create } from 'zustand';
import React from 'react';
import { ServiceError, GetConnectorFiles } from '@rapidaai/react';
import { GetConnectorFilesResponse } from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { connectionConfig } from '@/configs';

/**
 *
 */

const initialConnectorFileType: ConnectorFileTypeProperty = {
  // all the files
  allFiles: [],

  // after filter the files
  filterFiles: [],

  // files that are shown to the user
  files: [],
};

const initialConnectorFileAction: ConnectorFileTypeAction = {
  clear: function (): void {
    throw new Error('Function not implemented.');
  },
  onChangeFiles: function (files: Struct[]): void {
    throw new Error('Function not implemented.');
  },
};

const connectorFiles = {
  getAllConnectorFiles: function (
    toolId: string,
    token: string,
    userId: string,
    projectId: string,
    onError: (err: string) => void,
    onSuccess: (e: Struct[]) => void,
  ): void {
    throw new Error('Function not implemented.');
  },
};

export const ConnectorFileContext = React.createContext<ConnectorFileType>({
  ...initialConnectorFileType,
  ...initialPaginated,
  ...initialConnectorFileAction,
  ...connectorFiles,
});

export const useConnectorFilePageStore = create<ConnectorFileType>(
  (set, get) => ({
    ...initialPaginated,
    ...initialConnectorFileType,

    onChangeFiles: (files: Struct[]) => {
      set({
        files: files,
      });
    },
    /**
     *
     * @param number
     * @returns
     */
    setPageSize: (pageSize: number) => {
      // when someone change pagesize change the page to zero
      set({
        page: 1,
        pageSize: pageSize,
      });
    },

    /**
     *
     * @param number
     * @returns
     */
    setPage: (pg: number) => {
      set({
        page: pg,
      });
    },

    /**
     *
     * @param number
     * @returns
     */
    setTotalCount: (tc: number) => {
      set({
        totalCount: tc,
      });
    },

    /**
     *
     * @param k
     * @param v
     */
    addCriteria: (k: string, v: string, logic: string) => {
      let ct = get().criteria;

      set({
        criteria: [
          ...ct.filter(x => x.key !== k),
          { key: k, value: v, logic: logic },
        ],
      });
    },

    /**
     *
     * @param projectId
     * @param token
     * @param userId
     */
    getAllConnectorFiles: (
      toolId: string,
      token: string,
      userId: string,
      projectId: string,
      onError: (err: string) => void,
      onSuccess: (e: Struct[]) => void,
    ) => {
      //
      //
      let ctr = get().criteria;
      let data = get().allFiles;
      if (data.length > 0) {
        const page = get().page;
        const pageSize = get().pageSize;

        //

        ctr.forEach(y => {
          data = data.filter(x => {
            if (y.logic === '=') {
              return x
                .getFieldsMap()
                .get(y.key)
                ?.getStringValue()
                .includes(y.value);
            }
            if (y.logic === '!=') {
              return x.getFieldsMap().get(y.key)?.getStringValue() !== y.value;
            }
            return true;
          });
        });

        let _a = data.slice((page - 1) * pageSize, page * pageSize);
        set({
          files: _a,
          filterFiles: data,
          totalCount: data.length,
        });
        onSuccess(_a);
        return;
      }
      //
      //
      GetConnectorFiles(
        connectionConfig,
        toolId,
        [],
        {
          authorization: token,
          'x-auth-id': userId,
          'x-project-id': projectId,
        },
        (err: ServiceError | null, uvcr: GetConnectorFilesResponse | null) => {
          if (uvcr?.getSuccess()) {
            let data = uvcr?.getDataList();
            if (data) {
              let _a = data.slice(0, 20);
              set({
                files: _a,
                allFiles: data,
                filterFiles: data,
                totalCount: data.length,
              });
              onSuccess(_a);
              return;
            }
          }
          onError(
            'Unable to get the list of files and folders for the connection, please try again later.',
          );
          return;
        },
      );
    },

    /**
     *
     */
    clearCriteria: () => {
      set({
        criteria: [],
      });
    },

    /**
     * clear everything from the context
     * @returns
     */
    clear: () =>
      set(
        {
          ...initialConnectorFileType,
          // ...initialPaginated,
        },
        // false,
      ),
  }),
);
