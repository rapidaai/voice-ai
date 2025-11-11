import { Assistant } from '@rapidaai/react';
import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';

/**
 * assistant context
 */

export type AssistantTypeProperty = {
  /**
   * current assistant
   */
  currentAssistant: Assistant | null;

  /**
   * list of assistant
   */
  assistants: Assistant[];

  /**
   * should show instruction
   */
  instructionVisible: boolean;

  /**
   * edit tag visible
   */
  editTagVisible: boolean;

  /**
   * edit descipt
   */

  updateDescriptionVisible: boolean;
};

export type AssistantTypeAction = {
  /**
   *
   * @param assistant
   * @returns
   */
  onChangeCurrentAssistant: (assistant: Assistant) => void;

  /**
   *
   * @returns
   */
  onClearCurrentAssistant: () => void;

  /**
   *
   * @param ep
   * @returns
   */
  onChangeAssistants: (ep: Assistant[]) => void;

  /**
   *
   * @param assistant
   * @returns
   */
  onReloadAssistant: (assistant: Assistant) => void;

  /**
   *
   * @param assistant
   * @returns
   */
  onAddAssistant: (assistant: Assistant) => void;

  /**
   *
   * @param ep
   * @returns
   */
  onShowInstruction: (ep: Assistant) => void;
  /**
   *
   * @param is
   * @returns
   */
  onHideInstruction: () => void;

  /**
   *
   * @param ep
   * @returns
   */
  onShowEditTagVisible: (ep: Assistant) => void;

  /**
   *
   * @returns
   */
  onHideEditTagVisible: () => void;

  /**
   *
   * @param ep
   * @returns
   */
  onShowUpdateDescription: (ep: Assistant) => void;

  /**
   *
   * @returns
   */
  onHideUpdateDescription: () => void;
};
/**
 *
 */
export type AssistantType = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  onGetAllAssistant: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Assistant[]) => void,
  ) => void;

  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onGetAssistant: (
    assistantId: string,
    assistantProviderModelId: string | null,
    projectId: string,
    token: string,
    userId: string,
    onSuccess: (assistant: Assistant) => void,
    onError: (err: string) => void,
  ) => void;
  /**
   *
   * @param assistantId
   * @param name
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @param description
   * @returns
   */
  onUpdateAssistantDescription: (
    assistantId: string,
    name: string,
    description: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Assistant) => void,
  ) => void;

  /**
   *
   * @param assistantId
   * @param tags
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onCreateAssistantTag: (
    assistantId: string,
    tags: string[],
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Assistant) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & AssistantTypeProperty &
  AssistantTypeAction &
  PaginatedType &
  ColumnarType;
