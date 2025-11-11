import { Assistant, GetAllAssistantProviderResponse } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

export type AssistantProviderTypeProperty = {
  /**
   *
   */
  assistant: Assistant | null;

  /**
   *
   */
  assistantProviders: GetAllAssistantProviderResponse.AssistantProvider[];
};
/**
 * assistant context
 */
type AssistantProviderTypeAction = {
  /**
   *
   * @param assistant
   * @returns
   */
  onChangeAssistant: (assistant: Assistant) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
};

export type AssistantProviderType = {
  /**
   *
   * @param assistantProviderModelId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onReleaseVersion: (
    assistantProvider: string,
    assistantProviderId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Assistant) => void,
  ) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getAssistantProviders: (
    assistantId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: GetAllAssistantProviderResponse.AssistantProvider[]) => void,
  ) => void;
} & PaginatedType &
  ColumnarType &
  AssistantProviderTypeProperty &
  AssistantProviderTypeAction;
