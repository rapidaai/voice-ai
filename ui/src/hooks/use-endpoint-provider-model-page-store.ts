import {
  GetAllEndpointProviderModel,
  GetEndpoint,
  UpdateEndpointVersion,
} from '@rapidaai/react';
import {
  Endpoint,
  EndpointProviderModel,
  GetAllEndpointProviderModelResponse,
  GetEndpointResponse,
  UpdateEndpointVersionResponse,
} from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';

import {
  EndpointProviderModelType,
  EndpointProviderModelTypeProperty,
} from '@/types';
import { initialPaginated } from '@/types/types.paginated';
import { create } from 'zustand';
import { connectionConfig } from '@/configs';

const initialState: EndpointProviderModelTypeProperty = {
  /**
   * current endpoint which will be targeted
   */
  currentEndpoint: null,

  /**
   *
   */
  currentEndpointProviderModel: null,

  /**
   * list of endpoint where these will be part of it
   */
  endpointProviderModels: [],

  /**
   *
   */
  instructionVisible: false,

  /**
   *
   */
  configureRetryVisible: false,

  /**
   *
   */
  configureCachingVisible: false,
};

/**
 *
 */
export const useEndpointProviderModelPageStore =
  create<EndpointProviderModelType>((set, get) => ({
    ...initialState,
    ...initialPaginated,

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

    onChangeCurrentEndpoint: (endpoint: Endpoint) => {
      set({ currentEndpoint: endpoint });
    },

    setEndpointProviderModels: (
      endpointProviderModels: EndpointProviderModel[],
    ) => {
      set({ endpointProviderModels: endpointProviderModels });
    },
    /**
     *
     * @param endpointProviderModel
     */
    onChangeCurrentEndpointProviderModel: (
      endpointProviderModel: EndpointProviderModel,
    ) => {
      set({
        currentEndpointProviderModel: endpointProviderModel,
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
     * @param endpointId
     * @param projectId
     * @param token
     * @param userId
     * @param onError
     * @param onSuccess
     */

    getEndpointProviderModels: (
      endpointId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: EndpointProviderModel[]) => void,
    ) => {
      const afterGetAllEndpointProviderModel = (
        err: ServiceError | null,
        gur: GetAllEndpointProviderModelResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          set({
            endpointProviderModels: gur.getDataList(),
          });
          let paginated = gur.getPaginated();
          if (paginated) {
            get().setTotalCount(paginated.getTotalitem());
          }
          onSuccess(gur.getDataList());
        } else {
          let errorMessage = gur?.getError();
          if (errorMessage) {
            onError(errorMessage.getHumanmessage());
            return;
          }
          onError('Unable to get all the version of endpoint.');
        }
      };

      GetAllEndpointProviderModel(
        connectionConfig,
        endpointId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllEndpointProviderModel,
        {
          authorization: token,
          'x-auth-id': userId,
          'x-project-id': projectId,
        },
      );
    },

    /**
     *
     * @param projectId
     * @param token
     * @param userId
     */
    getEndpoint: (
      endpointId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: Endpoint) => void,
    ) => {
      const afterGetEndpoint = (
        err: ServiceError | null,
        gur: GetEndpointResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          let ed = gur.getData();
          if (ed) {
            get().onChangeCurrentEndpoint(ed);
            onSuccess(ed);
            return;
          }
        } else {
          let errorMessage = gur?.getError();
          if (errorMessage) {
            onError(errorMessage.getHumanmessage());
            return;
          }
          onError('Unable to get your endpoint, please try again later.');
        }
      };

      GetEndpoint(
        connectionConfig,
        endpointId,
        null,
        {
          authorization: token,
          'x-auth-id': userId,
          'x-project-id': projectId,
        },
        afterGetEndpoint,
      );
    },

    onReleaseVersion: (
      endpointProviderModelId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: Endpoint) => void,
    ) => {
      let endpoint = get().currentEndpoint;
      if (!endpoint && endpoint === null) {
        return;
      }

      if (endpoint?.getEndpointprovidermodelid() === endpointProviderModelId) {
        onSuccess(endpoint);
        return;
      }

      const afterUpdateEndpointVersion = (
        err: ServiceError | null,
        aur: UpdateEndpointVersionResponse | null,
      ) => {
        if (aur?.getSuccess()) {
          const ed = aur.getData();
          if (ed) onSuccess(ed);
        } else {
          let _er = aur?.getError();
          if (_er) {
            onError(_er.getHumanmessage());
            return;
          }
          onError('Unable to process your request. please try again later.');
          return;
        }
      };

      UpdateEndpointVersion(
        connectionConfig,
        endpoint?.getId(),
        endpointProviderModelId,
        {
          authorization: token,
          'x-auth-id': userId,
          'x-project-id': projectId,
        },
        afterUpdateEndpointVersion,
      );
    },

    /**
     * columns
     */
    columns: [
      { name: 'System', key: 'getSystemPrompt', visible: true },
      { name: 'User', key: 'getUserPrompt', visible: true },
      { name: 'Created on', key: 'getCreateddate', visible: true },
    ],

    /**
     *
     * @param cl
     */
    setColumns(cl: { name: string; key: string; visible: boolean }[]) {
      set({
        columns: cl,
      });
    },

    /**
     *
     * @param k
     * @returns
     */
    visibleColumn: (k: string): boolean => {
      const column = get().columns.find(c => c.key === k);
      return column ? column.visible : false;
    },
    /**
     *
     * @param is
     * @returns
     */
    hideInstruction: () => {
      set({
        currentEndpoint: null,
        currentEndpointProviderModel: null,
        instructionVisible: false,
      });
    },

    /**
     *
     * @param ep
     * @param epm
     * @returns
     */
    showInstruction: (ep: Endpoint, epm: EndpointProviderModel | null) => {
      set({
        currentEndpoint: ep,
        currentEndpointProviderModel: epm,
        instructionVisible: true,
      });
    },

    /**
     * retry modal control
     */

    showConfigureRetry: (ep: Endpoint) => {
      set({
        currentEndpoint: ep,
        configureRetryVisible: true,
      });
    },

    hideConfigureRetry: () => {
      set({
        currentEndpoint: null,
        configureRetryVisible: false,
      });
    },

    /**
     * caching configure control
     */

    showConfigureCaching: (ep: Endpoint) => {
      set({
        currentEndpoint: ep,
        configureRetryVisible: true,
      });
    },

    hideConfigureCaching: () => {
      set({
        currentEndpoint: null,
        configureRetryVisible: false,
      });
    },
    /**
     * clear everything from the context
     * @returns
     */
    clear: () => set({}, true),
  }));
