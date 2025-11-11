import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

export type EndpointProviderModelTypeProperty = {
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
  endpointProviderModels: EndpointProviderModel[];

  /**
   * dialog visible
   */
  instructionVisible: boolean;
  configureRetryVisible: boolean;
  configureCachingVisible: boolean;
};
/**
 * endpoint context
 */
type EndpointProviderModelTypeAction = {
  /**
   *
   * @param endpoint
   * @returns
   */
  onChangeCurrentEndpoint: (endpoint: Endpoint) => void;

  /**
   *
   * @param e
   */
  onChangeCurrentEndpointProviderModel: (e: EndpointProviderModel) => void;

  /**
   * should show instruction
   */

  /**
   *
   * @param is
   * @returns
   */
  hideInstruction: () => void;

  /**
   *
   * @param ep
   * @param epm
   * @returns
   */
  showInstruction: (ep: Endpoint, epm: EndpointProviderModel | null) => void;

  /**
   * @returns
   */
  hideConfigureRetry: () => void;

  /**
   * retry modal control
   */
  showConfigureRetry: (ep: Endpoint) => void;

  /**
   * caching configure control
   */
  showConfigureCaching: (ep: Endpoint) => void;

  /**
   *
   * @returns
   */
  hideConfigureCaching: () => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
};

export type EndpointProviderModelType = {
  /**
   *
   * @param endpointProviderModelId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onReleaseVersion: (
    endpointProviderModelId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Endpoint) => void,
  ) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getEndpointProviderModels: (
    endpointId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: EndpointProviderModel[]) => void,
  ) => void;
} & PaginatedType &
  ColumnarType &
  EndpointProviderModelTypeProperty &
  EndpointProviderModelTypeAction;
