import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

export type EndpointTypeProperties = {
  /**
   *
   */
  currentEndpoint: Endpoint | null;

  /**
   *
   */
  currentEndpointProviderModel: EndpointProviderModel | null;

  /**
   *
   */
  endpoints: Endpoint[];

  /**
   * dialog flag
   */
  instructionVisible: boolean;
  editTagVisible: boolean;
  updateDetailVisible: boolean;
  configureRetryVisible: boolean;
  configureCachingVisible: boolean;
};

export type EndpointTypeAction = {
  /**
   *
   * @param endpoint
   * @returns
   */
  onChangeCurrentEndpoint: (endpoint: Endpoint) => void;

  /**
   *
   * @param e
   * @returns
   */
  onChangeCurrentEndpointProviderModel: (e: EndpointProviderModel) => void;

  /**
   *
   * @returns
   */
  onClearEndpoint: () => void;

  /**
   *
   * @param ep
   * @returns
   */
  onChangeEndpoints: (ep: Endpoint[]) => void;

  /**
   *
   * @param endpoint
   * @returns
   */
  onReloadEndpoint: (endpoint: Endpoint) => void;

  /**
   *
   * @param endpoint
   * @returns
   */
  onAddEndpoint: (endpoint: Endpoint) => void;

  /**
   *
   * @param is
   * @returns
   */
  onHideInstruction: () => void;

  /**
   *
   * @param ep
   * @param epm
   * @returns
   */
  onShowInstruction: () => void;

  /**
   * retry modal control
   */
  onShowConfigureRetry: (ep: Endpoint) => void;
  onHideConfigureRetry: () => void;

  /**
   * caching configure control
   */
  onShowConfigureCaching: (ep: Endpoint) => void;
  onHideConfigureCaching: () => void;

  /**
   * edit tag visible
   */
  onShowEditTagVisible: (ep: Endpoint) => void;
  onHideEditTagVisible: () => void;

  onShowUpdateDetailVisible: (ep: Endpoint) => void;
  onHideUpdateDetailVisible: () => void;
};
/**
 * endpoint context
 */
export type EndpointType = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  onGetAllEndpoint: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Endpoint[]) => void,
  ) => void;

  /**
   *
   * @param endpointId
   * @param tags
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onCreateEndpointTag: (
    endpointId: string,
    tags: string[],
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Endpoint) => void,
  ) => void;

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
   * @returns
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
    onSuccess: (e: Endpoint) => void,
  ) => void;

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
    onSuccess: (e: Endpoint) => void,
  ) => void;

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
   * @returns
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
  ) => void;

  /**
   *
   * @param endpointId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onGetEndpoint: (
    endpointId: string,
    endpointProviderModelId: string | null,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Endpoint) => void,
  ) => void;
  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & PaginatedType &
  ColumnarType &
  EndpointTypeProperties &
  EndpointTypeAction;
