import {
  CreateEndpointCacheConfiguration,
  CreateEndpointRetryConfiguration,
  CreateEndpointTag,
  GetAllEndpoint,
  GetEndpoint,
  UpdateEndpointDetail,
} from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import {
  CreateEndpointCacheConfigurationResponse,
  CreateEndpointRetryConfigurationResponse,
  Endpoint,
  EndpointProviderModel,
  GetAllEndpointResponse,
  GetEndpointResponse,
} from '@rapidaai/react';

import { EndpointType } from '@/types';
import { initialPaginated } from '@/types/types.paginated';
import { create } from 'zustand';
import { connectionConfig } from '@/configs';

/**
 *
 */
export const initialEndpointType = {
  /**
   * current endpoint which will be targeted
   */
  currentEndpoint: null,

  /**
   * list of endpoint where these will be part of it
   */
  endpoints: [],

  /**
   *
   */
  currentEndpointProviderModel: null,

  /**
   * edit of tag dialog
   */
  editTagVisible: false,

  /**
   * should show instruction
   */
  instructionVisible: false,

  /**
   * retry modal control
   */
  configureRetryVisible: false,

  /**
   * visibility of edit dialog
   */
  updateDetailVisible: false,

  /**
   * caching configure control
   */

  configureCachingVisible: false,
};
/**
 *
 */
export const useEndpointPageStore = create<EndpointType>((set, get) => ({
  ...initialPaginated,
  ...initialEndpointType,

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
   * @param endpoint
   */
  onChangeCurrentEndpoint: (endpoint: Endpoint) => {
    set({
      currentEndpoint: endpoint,
    });
  },

  /**
   *
   * @param ep
   */
  onChangeEndpoints: (ep: Endpoint[]) => {
    set({
      endpoints: ep,
    });
  },

  /**
   *
   * @param endpoint
   */
  onReloadEndpoint: (endpoint: Endpoint) => {
    get().onChangeEndpoints([
      endpoint,
      ...get().endpoints.filter(ep => endpoint.getId() !== ep.getId()),
    ]);
    get().onChangeCurrentEndpoint(endpoint);
  },

  /**
   *
   */
  onClearEndpoint: () => {
    set({
      currentEndpoint: null,
    });
  },

  /**
   *
   * @param endpoint
   */
  onAddEndpoint: (endpoint: Endpoint) => {
    get().onReloadEndpoint(endpoint);
  },

  /**
   *
   * @param k
   * @param v
   */
  addCriteria: (k: string, v: string, logic: string) => {
    let current = get().criteria.filter(x => x.key !== k && x.logic !== logic);
    if (v) current.push({ key: k, value: v, logic: logic });
    set({
      criteria: current,
    });
  },

  /**
   *
   */
  removeCriteria: (k: string) => {
    set(state => ({
      criteria: state.criteria.filter(criterion => criterion.key !== k),
    }));
  },

  /**
   *
   * @param v
   */
  setCriterias: (v: { k: string; v: string; logic: string }[]) => {
    set({
      criteria: v.map(c => {
        return { key: c.k, value: c.v, logic: c.logic };
      }),
    });
  },

  /**
   *
   * @param v
   */
  addCriterias: (v: { k: string; v: string; logic: string }[]) => {
    let current = get().criteria.filter(
      x => !v.find(y => y.k === x.key && x.logic === y.logic),
    );
    v.forEach(c => {
      current.push({ key: c.k, value: c.v, logic: c.logic });
    });
    set({
      criteria: current,
    });
  },

  /**
   *
   * @param projectId
   * @param token
   * @param userId
   */
  onGetAllEndpoint: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Endpoint[]) => void,
  ) => {
    const afterGetAllEndpoint = (
      err: ServiceError | null,
      gur: GetAllEndpointResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        get().onChangeEndpoints(gur.getDataList());
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
        onError(
          'Something went wrong while retrieving your endpoints. Please refresh the page or try again later.',
        );
      }
    };

    GetAllEndpoint(
      connectionConfig,
      get().page,
      get().pageSize,
      get().criteria,
      afterGetAllEndpoint,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
    );
  },

  /**
   * columns
   */
  columns: [
    { name: 'Endpoint Status', key: 'getStatus', visible: true },
    { name: 'Endpoint', key: 'getName', visible: true },
    { name: 'Current Version', key: 'getVersion', visible: false },
    { name: 'Tags', key: 'getTags', visible: true },
    { name: 'Run Count (7D)', key: 'getCount', visible: true },
    { name: 'Error Rate (7D)', key: 'getErrorRate', visible: true },
    { name: 'Current Model', key: 'getCurrentModel', visible: true },
    { name: 'Total Token (7D)', key: 'getTotalToken', visible: true },
    { name: 'P50 Latency (7D)', key: 'getP50', visible: true },
    { name: 'P99 Latency (7D)', key: 'getP99', visible: true },
    { name: 'Most Recent Run', key: 'getMRR', visible: true },
    { name: 'Created by', key: 'getCreatedBy', visible: true },
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
  onHideInstruction: () => {
    set({
      instructionVisible: false,
    });
  },

  /**
   *
   * @param ep
   * @param epm
   * @returns
   */
  onShowInstruction: () => {
    set({
      instructionVisible: true,
    });
  },

  /**
   *
   * @param ep
   */
  onShowConfigureRetry: (ep: Endpoint) => {
    set({
      currentEndpoint: ep,
      configureRetryVisible: true,
    });
  },

  /**
   *
   */
  onHideConfigureRetry: () => {
    set({
      configureRetryVisible: false,
    });
  },

  /**
   *
   * @param ep
   */
  onShowConfigureCaching: (ep: Endpoint) => {
    set({
      currentEndpoint: ep,
      configureCachingVisible: true,
    });
  },

  /**
   *
   */
  onHideUpdateDetailVisible: () => {
    set({
      updateDetailVisible: false,
    });
  },

  /**
   *
   * @param ep
   */
  onShowUpdateDetailVisible: (ep: Endpoint) => {
    set({
      currentEndpoint: ep,
      updateDetailVisible: true,
    });
  },
  /**
   *
   */
  onHideConfigureCaching: () => {
    set({
      configureCachingVisible: false,
    });
  },

  /**
   * edit tag
   */

  onShowEditTagVisible: (ep: Endpoint) => {
    set({
      currentEndpoint: ep,
      editTagVisible: true,
    });
  },

  onHideEditTagVisible: () => {
    set({
      editTagVisible: false,
    });
  },

  /**
   *
   * @param endpointId
   * @param projectId
   * @param tags
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onCreateEndpointTag: (
    endpointId: string,
    tags: string[],
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Endpoint) => void,
  ) => {
    const afterCreateEndpointTag = (
      err: ServiceError | null,
      gur: GetEndpointResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        let endpoint = gur.getData();
        if (endpoint) {
          get().onReloadEndpoint(endpoint);
          onSuccess(endpoint);
        }
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError('Unable to update endpoint tag, please try again later.');
      }
    };

    CreateEndpointTag(
      connectionConfig,
      endpointId,
      tags,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
      afterCreateEndpointTag,
    );
  },

  /**
   *
   * @param endpointId
   * @param retryType
   * @param maxAttempts
   * @param delaySeconds
   * @param exponentialBackoff
   * @param retryables
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onCreateEndpointRetryConfiguration: (
    endpointId: string,
    retryType: string,
    maxAttempts: string,
    delaySeconds: string,
    exponentialBackoff: boolean,
    retryables: string[],
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e) => void,
  ) => {
    const afterCreateEndpointRetryConfiguration = (
      err: ServiceError | null,
      gur: CreateEndpointRetryConfigurationResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        let rtc = gur.getData();
        if (rtc) {
          let rE = get().currentEndpoint;
          if (rE) {
            rE?.setEndpointretry(rtc);
            get().onReloadEndpoint(rE);
          }
        }
        onSuccess(rtc);
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError(
          'Unable to update endpoint retry configuration, please try again later.',
        );
      }
    };

    CreateEndpointRetryConfiguration(
      connectionConfig,
      endpointId,
      retryType,
      maxAttempts,
      delaySeconds,
      exponentialBackoff,
      retryables,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
      afterCreateEndpointRetryConfiguration,
    );
  },
  // /

  /**
   *
   * @param endpointId
   * @param cacheType
   * @param expiryInterval
   * @param matchThreshold
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onCreateEndpointCacheConfiguration: (
    endpointId: string,
    cacheType: string,
    expiryInterval: string,
    matchThreshold: number,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e) => void,
  ) => {
    const afterCreateEndpointCacheConfiguration = (
      err: ServiceError | null,
      gur: CreateEndpointCacheConfigurationResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        onSuccess(gur.getData());
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError(
          'Unable to update endpoint cache configuration, please try again later.',
        );
      }
    };

    CreateEndpointCacheConfiguration(
      connectionConfig,
      endpointId,
      cacheType,
      expiryInterval,
      matchThreshold,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
      afterCreateEndpointCacheConfiguration,
    );
  },

  /**
   *
   * @param endpointId
   * @param name
   * @param description
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onUpdateEndpointDetail: (
    endpointId: string,
    name: string,
    description: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Endpoint) => void,
  ) => {
    const afterUpdateEndpointDetail = (
      err: ServiceError | null,
      gur: GetEndpointResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        let endpoint = gur.getData();
        if (endpoint) {
          get().onReloadEndpoint(endpoint);
          onSuccess(endpoint);
        }
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError('Unable to update endpoint, please try again later.');
      }
    };

    UpdateEndpointDetail(
      connectionConfig,
      endpointId,
      name,
      description,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
      afterUpdateEndpointDetail,
    );
  },

  /**
   *
   * @param endpointId
   * @param endpointProviderModelId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onGetEndpoint: (
    endpointId: string,
    endpointProviderModelId: string | null,
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
        let endpoint = gur.getData();
        if (endpoint) {
          get().onReloadEndpoint(endpoint);
          onSuccess(endpoint);
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
      endpointProviderModelId,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
      afterGetEndpoint,
    );
  },
  /**
   * clear everything from the context
   * @returns
   */
  clear: () => set({ ...initialEndpointType }, true),
}));
