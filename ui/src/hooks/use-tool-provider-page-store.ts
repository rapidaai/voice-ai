import {
  ToolProviderType,
  ToolProviderTypeAction,
  ToolProviderTypeProperty,
} from '@/types/types.tool-provider';
import { initialPaginated } from '@/types/types.paginated';
import { create } from 'zustand';
import { GetAllToolProvider } from '@rapidaai/react';
import React from 'react';
import { ServiceError } from '@rapidaai/react';
import { GetAllToolProviderResponse, ToolProvider } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

/**
 *
 */

const initialToolProviderType: ToolProviderTypeProperty = {
  toolProviders: [],
};

const initialToolAction: ToolProviderTypeAction = {
  clear: function (): void {
    throw new Error('Function not implemented.');
  },
  onChangeToolProviders: function (tools: ToolProvider[]): void {
    throw new Error('Function not implemented.');
  },
};

const tool = {
  getAllToolProvider: function (
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: ToolProvider[]) => void,
  ): void {
    throw new Error('Function not implemented.');
  },
};

export const ToolProviderContext = React.createContext<ToolProviderType>({
  ...initialToolProviderType,
  ...initialPaginated,
  ...initialToolAction,
  ...tool,
});

export const useToolProviderPageStore = create<ToolProviderType>(
  (set, get) => ({
    ...initialPaginated,
    ...initialToolProviderType,

    onChangeToolProviders: (tools: ToolProvider[]) => {
      set({
        toolProviders: tools,
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
      get().criteria.push({ key: k, value: v, logic: logic });
    },

    /**
     *
     * @param projectId
     * @param token
     * @param userId
     */
    getAllToolProvider: (
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: ToolProvider[]) => void,
    ) => {
      const afterGetAllToolProvider = (
        err: ServiceError | null,
        gur: GetAllToolProviderResponse | null,
      ) => {
        console.dir(gur?.toObject());
        if (gur?.getSuccess()) {
          get().onChangeToolProviders(gur.getDataList());
          let paginated = gur.getPaginated();
          if (paginated) {
            get().setTotalCount(paginated.getTotalitem());
          }
          get().onChangeToolProviders(gur.getDataList());
          onSuccess(gur.getDataList());
        } else {
          let errorMessage = gur?.getError();
          if (errorMessage) {
            onError(errorMessage.getHumanmessage());
            return;
          }
          onError('Unable to get all tool provider, please try again later.');
        }
      };

      GetAllToolProvider(
        connectionConfig,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllToolProvider,
        {
          authorization: token,
          'x-auth-id': userId,
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
          ...initialToolProviderType,
          ...initialPaginated,
        },
        false,
      ),
  }),
);
